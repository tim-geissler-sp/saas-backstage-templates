// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.

/*
Package config supports microservice configuration via environment variables, with
support for retrieving and decrypting values from Amazon's SSM parameter store.

It has support for parsing of various native go types, like strings, slices, integers, and duration.

Given an example environment:
BATCH_SIZE=100
PODS=a,b,c
SECRET_KEY_SSM=/param/store/value
JWT_KEY_PARAM_NAME=/param/store/value2

	s := NewSource()
	config.GetInt(s, "BATCH_SIZE") == 100
	config.GetString(s, "BATCH_SIZE") == "100"
	config.GetStringSlice(s, "PODS") == []string{"a", "b", "c"}
	config.GetString(s, "SECRET_KEY") == AWS Param store value for the key "/param/store/value"
	config.GetString(s, "JWT_KEY") == AWS Param store value for the key "/param/store/value2"
*/
package config

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-sdk-go/service/ssm"
	_ "github.com/joho/godotenv/autoload"
	"github.com/sailpoint/atlas-go/atlas/log"
)

const (
	usEast1    = "us-east-1"
	usGovWest1 = "us-gov-west-1"
)

var (
	// globalAwsSession is a singleton session to be shared across all AWS service clients in atlas-go application
	globalAwsSession = session.Must(session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			MaxRetries: aws.Int(5),
		},
		SharedConfigState: session.SharedConfigEnable,
	}))

	// currentRegion is the current AWS region this application is running on; defaults to usEast1 locally
	currentRegion = usEast1

	// mainRegion is the AWS partition main region, i.e. default region for global resources; defaults to usEast1 locally
	mainRegion = usEast1
)

// TODO: We need an init method here to do the config loading hierarchy the same way atlas does
// Otherwise we will have a chicken and egg problem where NewSource() -> SSM cliwnt
// -> AWS session -> GetString() -> NewSource().
/*

func init() {

	defaultConfig = map[string]string {
		"region": "us-east-1",
	}
	config = make(map[string]string)

	for k, v := range defaultConfig {
		config[k] = v
	}

	for _, e := range os.Environ() {
        variable := strings.Split(e, "=")
        config[variable[0]] = variable[1]
	}

	globalAwsSession = ...

	// Consider putting SSM post processing here
}
*/

func init() {
	initAwsRegions()
}

func initAwsRegions() {
	region, err := ec2metadata.New(globalAwsSession).Region()
	if err != nil {
		if v := os.Getenv("ATLAS_AWS_REGION"); v != "" {
			currentRegion = v
		}
	} else {
		currentRegion = region
	}

	if strings.HasPrefix(currentRegion, "us-gov") {
		mainRegion = usGovWest1
	}
}

// Source is an interface for reading configuration data.
type Source interface {
	GetString(key string) string
}

// SecretsManager is an interface for retrieving secrets only.  Writing secrets isn't supported
type SecretsManager interface {
	GetSecretValue(key string) (string, error)
}

// AwsSecretsManager is intended as the implementation of SecretManager Inteface; though it might implement other
// interfaces in the future.
type AwsSecretsManager struct {
	mySecretManager *secretsmanager.SecretsManager
}

func (s AwsSecretsManager) GetSecretValue(key string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := s.mySecretManager.GetSecretValue(input)
	if err != nil {
		return "", fmt.Errorf("secret manager read '%s': %w", key, err)
	}

	var stringResult string = *result.SecretString
	return stringResult, nil

}

// DefaultSource is the default Source implementation that reads from the system environment.
type DefaultSource struct {
	ssmClient *ssm.SSM
	mu        sync.RWMutex
	ssmCache  map[string]string
	svc       SecretsManager
}

// NewSource returns a new default configuration Source.
func NewSource() *DefaultSource {
	s := &DefaultSource{}
	s.ssmCache = make(map[string]string)

	s.ssmClient = ssm.New(globalAwsSession)
	s.svc = AwsSecretsManager{secretsmanager.New(globalAwsSession)}
	return s
}

// GlobalAwsSession returns a global AWS session for AWS service clients
func GlobalAwsSession() *session.Session {
	return globalAwsSession
}

// CurrentRegion returns the current AWS region this application is running on
func CurrentRegion() string {
	return currentRegion
}

// MainRegion returns our AWS partition main region, i.e. default region for global resources
// This is "us-east-1" in commercial account and "us-gov-west-1" in GovCloud account
func MainRegion() string {
	return mainRegion
}

// GetString retrieves a configuration value for the specified key, if no value is present, "" is returned.
func (s *DefaultSource) GetString(key string) string {
	if value := os.Getenv(key + "_PARAM_NAME"); value != "" {
		v, err := s.ssmGet(value)
		if err != nil {
			log.Global().Sugar().Fatalf("config get: %s: %v", key, err)
		}

		return v
	}

	if value := os.Getenv(key + "_SSM"); value != "" {
		v, err := s.ssmGet(value)
		if err != nil {
			log.Global().Sugar().Fatalf("config get: %s: %v", key, err)
		}

		return v
	}

	if value := os.Getenv(key + "_SECRET_NAME"); value != "" {
		v, err := s.svc.GetSecretValue(value)
		if err != nil {
			log.Global().Sugar().Fatalf("config get: %s: %v", key, err)
		}

		return v
	}

	return os.Getenv(key)
}

// ssmGet gets a parameter from AWS SSM, if the value has been ready previously, the
// cached value is returned.
func (s *DefaultSource) ssmGet(key string) (string, error) {
	if v := s.ssmGetFromCache(key); v != "" {
		return v, nil
	}

	return s.ssmLoad(key)
}

// ssmGetFromCache reads from the source's cache. An empty string is returned if
// no cached value exists.
func (s *DefaultSource) ssmGetFromCache(key string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.ssmCache[key]
}

// ssmLoad reads a key from AWS SSM and loads the result into the cache.
// The resulting value is returned.
func (s *DefaultSource) ssmLoad(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out, err := s.ssmClient.GetParameter(&ssm.GetParameterInput{
		Name:           aws.String(key),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return "", fmt.Errorf("ssm read '%s': %w", key, err)
	}

	if out.Parameter.Value != nil {
		s.ssmCache[key] = *out.Parameter.Value
		return *out.Parameter.Value, nil
	}

	return "", nil
}

// GetString retrieves a configuration value for the specified key, if no value is present, defaultValue is returned.
func GetString(s Source, key string, defaultValue string) string {
	if value := s.GetString(key); value != "" {
		return value
	}

	return defaultValue
}

// GetStringSlice retrieves a []string value for the specified key, if no value is present, defaultValue is returned.
// For environment variables, the value is expected to be comma-separated.
// (eg. given env PODS="a,b,c" -> GetStringSlice("PODS", nil) will return []string{"a", "b", "c"})
func GetStringSlice(s Source, key string, defaultValue []string) []string {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	values := strings.Split(value, ",")
	for i := range values {
		values[i] = strings.TrimSpace(values[i])
	}

	return values
}

// GetBool retrieves a boolean value for the specified key, if no value is present, or the value cannot be converted to
// a boolean, defaultValue is returned.
func GetBool(s Source, key string, defaultValue bool) bool {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	if v, err := strconv.ParseBool(value); err == nil {
		return v
	}

	return defaultValue
}

// GetInt retrieves an int value for the specified key, if no value is present, or the value cannot be converted to
// an integer, defaultValue is returned.
func GetInt(s Source, key string, defaultValue int) int {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	if v, err := strconv.Atoi(value); err == nil {
		return v
	}

	return defaultValue
}

// GetInt64 retrieves an int64 value for the specified key, if no value is present, or the value cannot be converted to
// an integer, defaultValue is returned.
func GetInt64(s Source, key string, defaultValue int64) int64 {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	if v, err := strconv.ParseInt(value, 10, 64); err == nil {
		return v
	}

	return defaultValue
}

// GetDuration retrieves a duration value for the specified key, if no value is present, or the value cannot be converted
// to a duration, defaultValue is returned.
func GetDuration(s Source, key string, defaultValue time.Duration) time.Duration {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	if v, err := time.ParseDuration(value); err == nil {
		return v
	}

	return defaultValue
}

// GetHex retrieves and decodes a hexidecimal value for the specified key, if no value is present, or the value
// is not valid hex, defaultValue is returned.
func GetHex(s Source, key string, defaultValue []byte) []byte {
	value := s.GetString(key)
	if value == "" {
		return defaultValue
	}

	if bytes, err := hex.DecodeString(value); err == nil {
		return bytes
	}

	return defaultValue
}

// GetPublicKeyString does parsing of json string from Secrets Manager to get byte array of key
func GetPublicKeyString(jsonStr string) ([]byte, error) {
	secretStringMap := map[string]interface{}{}
	if err := json.Unmarshal([]byte(jsonStr), &secretStringMap); err != nil {
		return nil, err
	}
	if key, ok := secretStringMap["publicKey"].(string); ok {
		jwtPublicKeyStringFromSecrets, err := base64.StdEncoding.DecodeString(key)
		if err != nil {
			return nil, err
		} else {
			return jwtPublicKeyStringFromSecrets, nil
		}
	} else {
		return nil, errors.New("Cannot get string from secret manager's unmarshaled map")
	}
}

func GetMultipleSecretValues(s DefaultSource, key string, defaultValue []string) []string {
	secretValues := make([]string, 0, 1)
	if envValue := os.Getenv(key); envValue != "" {
		paths := strings.Split(envValue, ",")
		for i := range paths {
			paths[i] = strings.TrimSpace(paths[i])
			value, err := s.svc.GetSecretValue(paths[i])
			if err == nil {
				secretValues = append(secretValues, value)
			} else {
				log.Global().Sugar().Errorf("config couldn't get: %s: %v", key, err)
			}
		}
	}
	return secretValues
}
