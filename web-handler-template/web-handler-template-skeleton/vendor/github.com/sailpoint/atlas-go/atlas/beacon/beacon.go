// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.

// Package beacon facilities running a microservice locally that is integrated with infrastructure and other services in the cloud.
package beacon

import (
	"context"
	"os"
	"time"

	"github.com/sailpoint/atlas-go/atlas/log"
)

// TenantID is the name of an org in atlas (I know this is confusing...)
type TenantID string

// ServiceID is the name of a microservice (ie stack)
type ServiceID string

// RegistrationID is a unique ID for a new beacon registration.
type RegistrationID string

// ConnectionID is the name of a beacon connection.
type ConnectionID string

// Configuration is a type that holds the beacon configuration for a service.
type Configuration map[string]string

// RegistrationRequest is an input type for creation of a beacon registration.
type RegistrationRequest struct {
	ServiceID    ServiceID
	TenantID     TenantID
	ConnectionID ConnectionID
	Port         int
	Hostname     string
}

// Registration is a representation within Beacon that a local instance of a service is running
// (typically on a developer's laptop)
type Registration struct {
	ID         RegistrationID
	Created    time.Time
	TenantID   TenantID
	ServiceID  ServiceID
	Hostname   string
	Connection string
}

// Registrar is an interface for interacting with the beacon registry.
type Registrar interface {
	Register(request RegistrationRequest) (*Registration, error)
	Heartbeat(registrationID RegistrationID) (bool, error)
	Cancel(registrationID RegistrationID) error
	FindAllByService(serviceID ServiceID) ([]*Registration, error)
	FindByTenantAndService(tenantID TenantID, serviceID ServiceID) (*Registration, error)
}

// Configurator is an interface for getting customer service configuration for operation
// in beacon mode.
type Configurator interface {
	FindByTenantAndService(tenantID TenantID, serviceID ServiceID) (*Configuration, error)
}

// GetConfiguration returns the configuration that should be used for this beacon registration.
func (r *Registration) GetConfiguration(configurator Configurator) (*Configuration, error) {
	return configurator.FindByTenantAndService(r.TenantID, r.ServiceID)
}

// Cancel will delete this registration from the specified Registrar.
func (r *Registration) Cancel(registrar Registrar) error {
	return registrar.Cancel(r.ID)
}

// StartHeartbeat will periodically send a heartbeat to the Registrar, letting the beacon
// system know that this service is still alive. Failure to heartbeat will result in
// the registration expiring.
func (r *Registration) StartHeartbeat(ctx context.Context, registrar Registrar) {
	for {
		exists, err := registrar.Heartbeat(r.ID)
		if err != nil {
			log.Warnf(ctx, "beacon heartbeat error: %v", err)
		} else if !exists {
			return
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(1 * time.Minute):
		}
	}
}

// GetString gets a value from the Configuration. This is a match for the atlas config.Source interface.
func (c *Configuration) GetString(key string) string {
	if c == nil {
		return ""
	}

	return (*c)[key]
}

// OverrideEnvironment takes all of the values in the Configuration object and sets values in the local
// system environment.
func (c *Configuration) OverrideEnvironment() {
	if c == nil {
		return
	}

	for k, v := range *c {
		log.Infof(context.Background(), "beacon override: %s => %s", k, v)
		os.Setenv(k, v)
	}
}

// OverrideEnvironmentWithConfiguration uses the default beacon implementations to override
// the system environment configuration.
func OverrideEnvironmentWithConfiguration(tenantID TenantID, serviceID ServiceID) error {
	configurator := NewDynamoConfigurator()

	config, err := configurator.FindByTenantAndService(tenantID, serviceID)
	if err != nil {
		return err
	}

	config.OverrideEnvironment()
	return nil
}
