package ldclient

import (
	"log"
	"net/http"
	"os"
	"time"

	"gopkg.in/launchdarkly/go-server-sdk.v4/ldhttp"
	"gopkg.in/launchdarkly/go-server-sdk.v4/ldlog"
)

// Config exposes advanced configuration options for the LaunchDarkly client.
type Config struct {
	// The base URI of the main LaunchDarkly service. This should not normally be changed except for testing.
	BaseUri string
	// The base URI of the LaunchDarkly streaming service. This should not normally be changed except for testing.
	StreamUri string
	// The base URI of the LaunchDarkly service that accepts analytics events. This should not normally be
	// changed except for testing.
	EventsUri string
	// The full URI for posting analytics events. This is different from EventsUri in that the client will not
	// add the default URI path to it. It should not normally be changed except for testing, and if set, it
	// causes EventsUri to be ignored.
	EventsEndpointUri string
	// The capacity of the events buffer. The client buffers up to this many events in memory before flushing.
	// If the capacity is exceeded before the buffer is flushed, events will be discarded.
	Capacity int
	// The time between flushes of the event buffer. Decreasing the flush interval means that the event buffer
	// is less likely to reach capacity.
	FlushInterval time.Duration
	// Enables event sampling if non-zero. When set to the default of zero, all events are sent to Launchdarkly.
	// If greater than zero, there is a 1 in SamplingInterval chance that events will be sent (for example, a
	// value of 20 means on average 5% of events will be sent).
	//
	// Deprecated: This feature will be removed in a future version of the SDK.
	SamplingInterval int32
	// The polling interval (when streaming is disabled). Values less than the default of MinimumPollInterval
	// will be set to the default.
	PollInterval time.Duration
	// An object that can be used to produce log output. Setting this property is equivalent to passing
	// the same object to config.Loggers.SetBaseLogger().
	//
	// Deprecated: This property may be removed in the future. Use Loggers.SetBaseLogger() instead.
	Logger Logger
	// Configures the SDK's logging behavior. You may call its SetBaseLogger() method to specify the
	// output destination (the default is standard error), and SetMinLevel() to specify the minimum level
	// of messages to be logged (the default is ldlog.Info).
	Loggers ldlog.Loggers
	// The connection timeout to use when making polling requests to LaunchDarkly.
	Timeout time.Duration
	// Sets the implementation of FeatureStore for holding feature flags and related data received from
	// LaunchDarkly.
	//
	// Except for testing purposes, you should not set this property directly but instead use
	// FeatureStoreFactory, which ensures that the FeatureStore component will use the same logging
	// configuration as the rest of the SDK.
	FeatureStore FeatureStore
	// Sets the implementation of FeatureStore for holding feature flags and related data received from
	// LaunchDarkly. See NewInMemoryFeatureStoreFactory (the default) and the redis, ldconsul, and lddynamodb packages.
	FeatureStoreFactory FeatureStoreFactory
	// Sets whether streaming mode should be enabled. By default, streaming is enabled. It should only be
	// disabled on the advice of LaunchDarkly support.
	Stream bool
	// Sets the initial reconnect delay for the streaming connection.
	//
	// The streaming service uses a backoff algorithm (with jitter) every time the connection needs
	// to be reestablished. The delay for the first reconnection will start near this value, and then
	// increase exponentially for any subsequent connection failures (up to a maximum of 30 seconds).
	//
	// This value is ignored if streaming is disabled. If it is zero, the default of 1 second is used.
	StreamInitialReconnectDelay time.Duration
	// Sets whether this client should use the LaunchDarkly relay in daemon mode. In this mode, the client does
	// not subscribe to the streaming or polling API, but reads data only from the feature store. See:
	// https://docs.launchdarkly.com/docs/the-relay-proxy
	UseLdd bool
	// Sets whether to send analytics events back to LaunchDarkly. By default, the client will send events. This
	// differs from Offline in that it only affects sending events, not streaming or polling for events from the
	// server.
	SendEvents bool
	// Sets whether this client is offline. An offline client will not make any network connections to LaunchDarkly,
	// and will return default values for all feature flags.
	Offline bool
	// Sets whether or not all user attributes (other than the key) should be hidden from LaunchDarkly. If this
	// is true, all user attribute values will be private, not just the attributes specified in PrivateAttributeNames.
	AllAttributesPrivate bool
	// Set to true if you need to see the full user details in every analytics event.
	InlineUsersInEvents bool
	// Marks a set of user attribute names private. Any users sent to LaunchDarkly with this configuration
	// active will have attributes with these names removed.
	PrivateAttributeNames []string
	// Sets whether the client should log a warning message whenever a flag cannot be evaluated due to an error
	// (e.g. there is no flag with that key, or the user properties are invalid). By default, these messages are
	// not logged, although you can detect such errors programmatically using the VariationDetail methods.
	LogEvaluationErrors bool
	// Sets whether log messages for errors related to a specific user can include the user key. By default, they
	// will not, since the user key might be considered privileged information.
	LogUserKeyInErrors bool
	// Deprecated: Please use UpdateProcessorFactory.
	UpdateProcessor UpdateProcessor
	// Factory to create an object that is responsible for receiving feature flag updates from LaunchDarkly.
	// If nil, a default implementation will be used depending on the rest of the configuration
	// (streaming, polling, etc.); a custom implementation can be substituted for testing.
	UpdateProcessorFactory UpdateProcessorFactory
	// An object that is responsible for recording or sending analytics events. If nil, a
	// default implementation will be used; a custom implementation can be substituted for testing.
	EventProcessor EventProcessor
	// The number of user keys that the event processor can remember at any one time, so that
	// duplicate user details will not be sent in analytics events.
	UserKeysCapacity int
	// The interval at which the event processor will reset its set of known user keys.
	UserKeysFlushInterval time.Duration
	// The User-Agent header to send with HTTP requests. This defaults to a value that identifies the version
	// of the Go SDK for LaunchDarkly usage metrics.
	UserAgent string
	// Set to true to opt out of sending diagnostic events.
	//
	// Unless DiagnosticOptOut is set to true, the client will send some diagnostics data to the LaunchDarkly
	// servers in order to assist in the development of future SDK improvements. These diagnostics consist of an
	// initial payload containing some details of the SDK in use, the SDK's configuration, and the platform the
	// SDK is being run on, as well as payloads sent periodically with information on irregular occurrences such
	// as dropped events.
	DiagnosticOptOut bool
	// The interval at which periodic diagnostic events will be sent, if DiagnosticOptOut is false.
	//
	// The default is every 15 minutes and the minimum is every minute.
	DiagnosticRecordingInterval time.Duration
	// For use by wrapper libraries to set an identifying name for the wrapper being used.
	//
	// This will be sent in request headers during requests to the LaunchDarkly servers to allow recording
	// metrics on the usage of these wrapper libraries.
	WrapperName string
	// For use by wrapper libraries to set the version to be included alongside a WrapperName.
	//
	// If WrapperName is unset, this field will be ignored.
	WrapperVersion string
	// If not nil, this function will be called to create an HTTP client instead of using the default
	// client. You may use this to specify custom HTTP properties such as a proxy URL or CA certificates.
	// The SDK may modify the client properties after that point (for instance, to add caching),
	// but will not replace the underlying Transport, and will not modify any timeout properties you set.
	// See NewHTTPClientFactory().
	//
	// Usage:
	//
	//     config := ld.DefaultConfig
	//     config.HTTPClientFactory = ld.NewHTTPClientFactory(ldhttp.ProxyURL(myProxyURL))
	HTTPClientFactory HTTPClientFactory
	// Used internally to share a diagnosticsManager instance between components.
	diagnosticsManager *diagnosticsManager
}

// HTTPClientFactory is a function that creates a custom HTTP client.
type HTTPClientFactory func(Config) http.Client

// UpdateProcessorFactory is a function that creates an UpdateProcessor.
type UpdateProcessorFactory func(sdkKey string, config Config) (UpdateProcessor, error)

// MinimumPollInterval describes the minimum value for Config.PollInterval. If you specify a smaller interval,
// the minimum will be used instead.
const MinimumPollInterval = 30 * time.Second

func (c Config) newHTTPClient() *http.Client {
	factory := c.HTTPClientFactory
	if factory == nil {
		factory = NewHTTPClientFactory()
	}
	client := factory(c)
	return &client
}

// NewHTTPClientFactory creates an HTTPClientFactory based on the standard SDK configuration as well
// as any custom ldhttp.TransportOption properties you specify.
//
// Usage:
//
//     config := ld.DefaultConfig
//     config.HTTPClientFactory = ld.NewHTTPClientFactory(ldhttp.CACertFileOption("my-cert.pem"))
func NewHTTPClientFactory(options ...ldhttp.TransportOption) HTTPClientFactory {
	return func(c Config) http.Client {
		client := http.Client{
			Timeout: c.Timeout,
		}
		allOpts := []ldhttp.TransportOption{ldhttp.ConnectTimeoutOption(c.Timeout)}
		allOpts = append(allOpts, options...)
		if transport, _, err := ldhttp.NewHTTPTransport(allOpts...); err == nil {
			client.Transport = transport
		}
		return client
	}
}

// The ldlog package already has its own logic for using a default logger if none was set.
// However, in the past we've always guaranteed that DefaultConfig.Logger is non-nil, so
// we need to continue doing so for now. If the client initialization logic sees that
// config.Logger is set to this exact instance, it'll ignore it.
var defaultLogger = log.New(os.Stderr, "[LaunchDarkly] ", log.LstdFlags)

// DefaultConfig provides the default configuration options for the LaunchDarkly client.
// The easiest way to create a custom configuration is to start with the
// default config, and set the custom options from there. For example:
//
//     var config = DefaultConfig
//     config.Capacity = 2000
var DefaultConfig = Config{
	BaseUri:                     "https://app.launchdarkly.com",
	StreamUri:                   "https://stream.launchdarkly.com",
	EventsUri:                   "https://events.launchdarkly.com",
	Capacity:                    10000,
	FlushInterval:               5 * time.Second,
	PollInterval:                MinimumPollInterval,
	Timeout:                     3000 * time.Millisecond,
	Stream:                      true,
	StreamInitialReconnectDelay: defaultStreamRetryDelay,
	FeatureStore:                nil,
	UseLdd:                      false,
	SendEvents:                  true,
	Offline:                     false,
	UserKeysCapacity:            1000,
	UserKeysFlushInterval:       5 * time.Minute,
	UserAgent:                   "",
	Logger:                      defaultLogger,
	DiagnosticRecordingInterval: 15 * time.Minute,
}
