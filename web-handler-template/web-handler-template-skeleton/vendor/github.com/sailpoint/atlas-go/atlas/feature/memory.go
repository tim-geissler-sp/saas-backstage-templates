// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package feature

import "context"

type memoryStore struct {
}

// NewMemoryStore constructs a new feature store that always returns default values.
func NewMemoryStore() Store {
	return &memoryStore{}
}

// IsEnabled gets whether or not the flag is enabled for the current context. The atlas.RequestContext
// is extracted from the context variable, if present.
func (s *memoryStore) IsEnabled(ctx context.Context, flag Flag, defaultValue bool) (bool, error) {
	return defaultValue, nil
}

// IsEnabledForUser gets whether or not the flag is enabled for the specified user.
func (s *memoryStore) IsEnabledForUser(user User, flag Flag, defaultValue bool) (bool, error) {
	return defaultValue, nil
}

func (s *memoryStore) IsExistsAndEnabled(ctx context.Context, flag Flag, defaultValue bool, defaultIfFlagDoesNotExist bool) (bool, error) {
	return defaultValue, nil
}


// Close shuts down any internal state for the store.
func (s *memoryStore) Close() {
}
