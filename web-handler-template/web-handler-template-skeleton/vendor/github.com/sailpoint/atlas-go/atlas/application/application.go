// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.

// Package application is a convenience library that makes it simple to write a new atlas application
// with standard configuration and behavior.
package application

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis/v8"

	"github.com/sailpoint/atlas-go/atlas/auth"
	"github.com/sailpoint/atlas-go/atlas/auth/access"
	"github.com/sailpoint/atlas-go/atlas/beacon"
	"github.com/sailpoint/atlas-go/atlas/client"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/db"
	"github.com/sailpoint/atlas-go/atlas/event"
	"github.com/sailpoint/atlas-go/atlas/feature"
	"github.com/sailpoint/atlas-go/atlas/log"
	"github.com/sailpoint/atlas-go/atlas/message"
	"github.com/sailpoint/atlas-go/atlas/metric"
	"github.com/sailpoint/atlas-go/atlas/web"
)

// Application is the central atlas type that holds references to all of the internal
// functionality provided by atlas.
type Application struct {
	Stack                  string
	Config                 config.Source
	TokenValidator         auth.TokenValidator
	AccessSummarizer       access.Summarizer
	EventPublisher         event.Publisher
	BaseURLProvider        client.BaseURLProvider
	ServiceLocator         client.ServiceLocator
	InternalClientProvider client.InternalClientProvider
	InternalRestClient     client.InternalRestClient
	RedisClient            redis.Cmdable
	BeaconRegistrar        beacon.Registrar
	BeaconRegistration     *beacon.Registration
	FeatureStore           feature.Store
	MessagePublisher       message.Publisher
	MetricsConfig          metric.MetricsConfig
}

type ConfigurationOption func(app *Application) error

func WithConfig(cfg config.Source) ConfigurationOption {
	return func(app *Application) error {
		app.Config = cfg
		return nil
	}
}

func WithDefaultConfig() ConfigurationOption {
	return func(app *Application) error {
		app.Config = config.NewSource()
		return nil
	}
}
func WithDefaultBeaconRegistrar() ConfigurationOption {
	return func(app *Application) error {
		app.BeaconRegistrar = beacon.NewDynamoRegistrar()
		return nil
	}
}
func WithDefaultBeaconRegistration() ConfigurationOption {
	return func(app *Application) error {
		if app.BeaconRegistrar == nil {
			WithDefaultBeaconRegistrar()(app)
		}
		beaconRegistration, err := initBeacon(app.Stack, app.BeaconRegistrar)
		if err != nil {
			return fmt.Errorf("beacon init: %w", err)
		}
		app.BeaconRegistration = beaconRegistration
		return nil
	}
}

func WithDefaultTokenValidator() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		signingKey := config.GetHex(app.Config, "ATLAS_JWT_KEY", nil)
		if len(signingKey) == 0 {
			return fmt.Errorf("signing key is missing or invalid")
		}
		composedTokenValidator := auth.NewComposedTokenValidator(signingKey, jwt.SigningMethodHS256)

		app.TokenValidator = composedTokenValidator
		if dConfig, ok := app.Config.(*config.DefaultSource); ok {
			LoadJWTPublicKeys(*dConfig, composedTokenValidator, "ATLAS_JWT_PUBLIC_KEYS_SECRET_NAME")
		}
		return nil
	}
}

func WithDefaultRedisClient() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		redisHost := config.GetString(app.Config, "ATLAS_REDIS_HOST", "localhost")
		redisPort := config.GetInt(app.Config, "ATLAS_REDIS_PORT", 6379)
		redisClient := redis.NewClient(&redis.Options{
			Addr: fmt.Sprintf("%s:%d", redisHost, redisPort),
		})
		app.RedisClient = redisClient
		return nil
	}
}

func WithDefaultEventPublisher() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		eventPublisher, err := event.NewPublisher(event.NewPublisherConfig(app.Config), app.MetricsConfig)
		if err != nil {
			return err
		}
		app.EventPublisher = eventPublisher
		return nil
	}
}

func WithDefaultBaseURLProvider() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		baseURLProvider := client.NewBaseURLProvider(app.Config)
		app.BaseURLProvider = baseURLProvider
		return nil
	}
}

func WithDefaultServiceLocator() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		if app.BeaconRegistrar == nil {
			WithDefaultBeaconRegistrar()(app)
		}
		serviceLocator := client.NewServiceLocator(app.Config, app.BeaconRegistrar)
		app.ServiceLocator = serviceLocator
		return nil
	}
}

func WithDefaultInternalClientProvider() ConfigurationOption {
	return func(app *Application) error {
		if app.Config == nil {
			WithDefaultConfig()(app)
		}
		if app.BaseURLProvider == nil {
			WithDefaultBaseURLProvider()(app)
		}
		if app.ServiceLocator == nil {
			WithDefaultServiceLocator()(app)
		}
		signingKey := config.GetHex(app.Config, "ATLAS_JWT_KEY", nil)
		internalCredentialsProvider := client.NewInternalCredentialsProvider(app.Stack, signingKey)
		internalClientProvider := client.NewInternalClientProvider(app.Stack, app.BaseURLProvider, app.ServiceLocator, internalCredentialsProvider)
		app.InternalClientProvider = internalClientProvider
		return nil
	}
}

func WithDefaultInternalRestClient() ConfigurationOption {
	return func(app *Application) error {
		if app.ServiceLocator == nil {
			err := WithDefaultServiceLocator()(app)
			if err != nil {
				return err
			}
		}
		if app.InternalClientProvider == nil {
			err := WithDefaultInternalClientProvider()(app)
			if err != nil {
				return err
			}
		}
		app.InternalRestClient = client.NewInternalRestClient(app.ServiceLocator, app.InternalClientProvider)
		return nil
	}
}

func WithDefaultAccessSummarizer() ConfigurationOption {
	return func(app *Application) error {
		if app.RedisClient == nil {
			WithDefaultRedisClient()(app)
		}
		if app.BaseURLProvider == nil {
			WithDefaultBaseURLProvider()(app)
		}
		if app.InternalClientProvider == nil {
			WithDefaultInternalClientProvider()(app)
		}
		accessSummarizer := access.NewSummarizer(app.RedisClient, app.BaseURLProvider, app.InternalClientProvider)
		app.AccessSummarizer = accessSummarizer
		return nil
	}
}

func WithDefaultMessagePublisher() ConfigurationOption {
	return func(app *Application) error {
		messagePublisher := message.NewRedisPublisher(app.RedisClient)
		app.MessagePublisher = messagePublisher
		return nil
	}
}

func WithDefaultFeatureStore() ConfigurationOption {
	return func(app *Application) error {
		var featureStore feature.Store
		if key := config.GetString(app.Config, "ATLAS_FEATURE_FLAG_KEY", ""); key != "" {
			var err error
			featureStore, err = feature.NewLaunchDarklyStore(app.Stack, key)
			if err != nil {
				return err
			}
		} else {
			featureStore = feature.NewMemoryStore()
		}
		app.FeatureStore = featureStore
		return nil
	}
}

func WithDefaultMetricsConfig() ConfigurationOption {
	return func(app *Application) error {
		if app.FeatureStore == nil {
			WithDefaultFeatureStore()(app)
		}
		metricsConfig := metric.NewMetricsConfig(app.FeatureStore)
		app.MetricsConfig = metricsConfig
		return nil
	}
}

// New constructs a new atlas application with the specified stack and given options if provided.
// If no options are provided then the defaults will be implemented.
// Constructor for Application
// example usage:
// application.New(stack)
// application.New(stack, WithConfig(myconfig))
// application.New(stack, WithBeaconRegistrar(myBeaconRegistrar))
// application.New(stack, WithConfig(myconfig), WithBeaconRegistrar(myBeaconRegistrar))
// In the above example the calling code of application.New would implement the WithXXX functions to be passed
// into the application.New constructor.  The WithXXX functions will return a ConfigurationOption function that implements
// the initialization of a interface associated to the Application struct.
// Options should be executed in the order of dependencies.  If an Interface initialization is needed by options further down the chain then the
// needed dependency should be set before the options that depends on that Interface initialization.  If a Interface dependency is found to be nil during startup
// the default implementation will attempt to be created.
func New(stack string, options ...ConfigurationOption) (*Application, error) {

	app := &Application{}
	app.Stack = stack

	for _, option := range options {
		if err := option(app); nil != err {
			return nil, err
		}
	}

	if app.Config == nil {
		if err := WithDefaultConfig()(app); nil != err {
			return nil, err
		}
	}

	if app.BeaconRegistrar == nil {
		if err := WithDefaultBeaconRegistrar()(app); nil != err {
			return nil, err
		}
	}

	if app.BeaconRegistration == nil {
		if err := WithDefaultBeaconRegistration()(app); nil != err {
			return nil, err
		}
	}

	//default logging to production level if not configured
	if config.GetBool(app.Config, "ATLAS_PRODUCTION", true) {
		log.ConfigureJSON(stack)
	}

	if app.TokenValidator == nil {
		if err := WithDefaultTokenValidator()(app); nil != err {
			return nil, err
		}
	}

	if app.RedisClient == nil {
		if err := WithDefaultRedisClient()(app); nil != err {
			return nil, err
		}
	}

	if app.FeatureStore == nil {
		if err := WithDefaultFeatureStore()(app); nil != err {
			return nil, err
		}
	}

	if app.MetricsConfig == nil {
		if err := WithDefaultMetricsConfig()(app); nil != err {
			return nil, err
		}
	}

	if app.EventPublisher == nil {
		if err := WithDefaultEventPublisher()(app); nil != err {
			return nil, err
		}
	}

	if app.BaseURLProvider == nil {
		if err := WithDefaultBaseURLProvider()(app); nil != err {
			return nil, err
		}
	}

	if app.MessagePublisher == nil {
		if err := WithDefaultMessagePublisher()(app); nil != err {
			return nil, err
		}
	}

	if app.ServiceLocator == nil {
		if err := WithDefaultServiceLocator()(app); nil != err {
			return nil, err
		}
	}

	if app.InternalClientProvider == nil {
		if err := WithDefaultInternalClientProvider()(app); nil != err {
			return nil, err
		}
	}

	if app.AccessSummarizer == nil {
		if err := WithDefaultAccessSummarizer()(app); nil != err {
			return nil, err
		}
	}

	return app, nil
}

// NewWithConfig constructs a new atlas application with the specified stack and config.
// Deprecated
// Deprecated Date: 6/17/2021
// Use New(stack string, WithConfig(myConfig))
func NewWithConfig(stack string, cfg config.Source) (*Application, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config source cannot be nil")
	}

	return New(stack, WithConfig(cfg))
}

// Close shuts down the application.
func (app *Application) Close() {
	app.FeatureStore.Close()
}

// StartEventConsumer starts an event consumer in the background using the specified Router.
func (app *Application) StartEventConsumer(ctx context.Context, router *event.Router) error {
	groupID := config.GetString(app.Config, "ATLAS_EVENT_CONSUMER_GROUP_ID", app.Stack)

	consumerConfig := event.NewConsumerConfig(app.Config)
	consumerConfig.GroupID = groupID
	consumerConfig.Topics = router.Topics()

	return event.StartConsumer(ctx, consumerConfig, router, app.MetricsConfig)
}

// StartMetricsServer starts a metrics server using the default configuration.
func (app *Application) StartMetricsServer(ctx context.Context) error {
	return web.StartMetricsServer(ctx, web.NewMetricsConfig(app.Config))
}

// StartWebServer starts a web server using the specified Handler.
func (app *Application) StartWebServer(ctx context.Context, handler http.Handler) error {
	return web.RunServer(ctx, web.NewRunConfig(app.Config), handler)
}

// StartBeaconHeartbeat starts a background process that heartbeats
// the current registration with the beacon registry.
func (app *Application) StartBeaconHeartbeat(ctx context.Context) error {
	if app.BeaconRegistration == nil {
		return nil
	}
	defer app.BeaconRegistration.Cancel(app.BeaconRegistrar)

	app.BeaconRegistration.StartHeartbeat(ctx, app.BeaconRegistrar)
	return nil
}

// ConnectDB connects to and runs migrations on the database specified in configuration.
func (app *Application) ConnectDB() (*sql.DB, error) {
	database, err := db.Connect(db.NewConfig(app.Config))
	if err != nil {
		return nil, fmt.Errorf("db connect: %w", err)
	}

	if err = db.Migrate(database); err != nil {
		return nil, fmt.Errorf("db migrate: %w", err)
	}

	return database, nil
}

// WaitForInterrupt invokes a done function when an OS interrupt is received
func (app *Application) WaitForInterrupt(ctx context.Context, done func()) error {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-c:
		log.Global().Sugar().Infof("process received %q signal calling done()", sig)
		done()
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// initBeacon overrides beacon configuration and registered this local service instance
// with the registry. If no BEACON_TENANT is enabled, this function is a no-op.
func initBeacon(stack string, registrar beacon.Registrar) (*beacon.Registration, error) {
	beaconTenant := os.Getenv("BEACON_TENANT")
	if beaconTenant == "" {
		return nil, nil
	}

	beaconTenantValues := strings.Split(beaconTenant, ":")
	if len(beaconTenantValues) != 2 {
		return nil, fmt.Errorf("invalid beacon tenant value '%s'", beaconTenant)
	}

	beaconTenantID := beacon.TenantID(beaconTenantValues[0])
	beaconConnectionID := beacon.ConnectionID(beaconTenantValues[1])
	beaconServiceID := beacon.ServiceID(stack)

	beacon.OverrideEnvironmentWithConfiguration(beaconTenantID, beaconServiceID)

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	port, err := getRestPort()
	if err != nil {
		return nil, err
	}

	return registrar.Register(beacon.RegistrationRequest{
		TenantID:     beaconTenantID,
		ServiceID:    beaconServiceID,
		ConnectionID: beaconConnectionID,
		Hostname:     hostname,
		Port:         port,
	})
}

// getRestPort returns the port that this application has configured for REST invocation.
func getRestPort() (int, error) {
	if portValue := os.Getenv("ATLAS_REST_PORT"); portValue != "" {
		port, err := strconv.Atoi(portValue)
		if err != nil {
			return 0, fmt.Errorf("invalid rest port '%s': %w", portValue, err)
		}

		return port, nil
	}

	return 7100, nil
}

// LoadJWTPublicKeys is responsible for loading RSA Public Keys from Secret Manager.  It will handle the case when there are multiple keys as well.
func LoadJWTPublicKeys(cfg config.DefaultSource, tokenValidator *auth.ComposedTokenValidator, envString string) {
	jwtPublicKeyStringFromSecrets := config.GetMultipleSecretValues(cfg, envString, make([]string, 0))
	if len(jwtPublicKeyStringFromSecrets) == 0 {
		// Commented out since this isn't part of the task defn for most services yet
		// Uncomment this after devops adds the env var to all services
		//return nil, fmt.Errorf("secret manager signing key is missing or invalid")
		log.Global().Sugar().Warnf("secret manager signing key is missing or invalid")
	} else {
		for _, jsonStr := range jwtPublicKeyStringFromSecrets {
			if signingKeyFromSecretManger, err := config.GetPublicKeyString(jsonStr); err == nil {
				tokenValidator.AddValidator(signingKeyFromSecretManger, jwt.SigningMethodRS256)
			}
		}

	}
}
