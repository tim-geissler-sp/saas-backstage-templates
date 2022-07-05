// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package feature

import "context"

// Flag is an alias for a string that represents a feature flag name (eg. ENABLE_SPECIAL_FEATURE)
type Flag string

// User is the context in which a feature flag evaluation takes place.
type User struct {
	Name   string
	Pod    string
	Org    string
	Custom map[string]interface{}
}

// Store is an interface for interacting with a feature-flag store.
type Store interface {

	// IsEnabled gets whether or not the flag is enabled for the current context. The atlas.RequestContext
	// is extracted from the context variable, if present.
	IsEnabled(ctx context.Context, flag Flag, defaultValue bool) (bool, error)

	// IsExistsAndEnabled gets whether or not the flag is enabled for the current context if it exists. If it
	// does not exist, then the defaultIfFlagDoesNotExist is served. If it exists and there are any errors
	// then defaultValue is served, i.e. behavior will be same as the IsEnabled method.
	IsExistsAndEnabled(ctx context.Context, flag Flag, defaultValue bool, defaultIfFlagDoesNotExist bool) (bool, error)

	// IsEnabledForUser gets whether or not the flag is enabled for the specified user.
	IsEnabledForUser(user User, flag Flag, defaultValue bool) (bool, error)

	// Close shuts down any internal state for the store.
	Close()
}
