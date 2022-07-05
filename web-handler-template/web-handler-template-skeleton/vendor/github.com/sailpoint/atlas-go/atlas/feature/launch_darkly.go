// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package feature

import (
	"context"
	"fmt"
	"time"

	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/log"
	"gopkg.in/launchdarkly/go-sdk-common.v1/ldvalue"
	ld "gopkg.in/launchdarkly/go-server-sdk.v4"
)

type launchDarklyStore struct {
	stack  string
	client *ld.LDClient
}

// NewLaunchDarklyStore constructs a new feature store with the specified configuration.
func NewLaunchDarklyStore(stack string, key string) (Store, error) {
	client, err := ld.MakeClient(key, 5*time.Second)
	if err != nil {
		return nil, err
	}

	s := &launchDarklyStore{}
	s.stack = stack
	s.client = client

	return s, nil
}

// IsEnabled gets whether or not the specified feature flag is enabled for the current context.
func (s *launchDarklyStore) IsEnabled(ctx context.Context, flag Flag, defaultValue bool) (bool, error) {
	return s.IsEnabledForUser(s.extractUser(ctx), flag, defaultValue)
}

// IsEnabled gets whether or not the specified feature flag is enabled for the specified user.
func (s *launchDarklyStore) IsEnabledForUser(user User, flag Flag, defaultValue bool) (bool, error) {
	return s.client.BoolVariation(string(flag), s.toLaunchDarklyUser(user), defaultValue)
}

// IsExistsAndEnabled gets whether or not the flag is enabled for the current context if it exists. If it
// does not exist, then the defaultIfFlagDoesNotExist is served. If it exists and there are any errors
// then defaultValue is served.
func (s *launchDarklyStore) IsExistsAndEnabled(ctx context.Context, flag Flag, defaultValue bool, defaultIfFlagDoesNotExist bool) (bool, error) {
	enabled, evaluationDetail, err := s.client.BoolVariationDetail(string(flag), s.toLaunchDarklyUser(s.extractUser(ctx)), defaultValue)
	if err != nil {
		if ld.EvalErrorFlagNotFound == evaluationDetail.Reason.GetErrorKind() {
			return defaultIfFlagDoesNotExist, nil
		}
		return defaultValue, err
	}
	return enabled, nil
}

// toLaunchDarklyUser converts a feature user to a user as expected from the launch darkly client.
func (s *launchDarklyStore) toLaunchDarklyUser(user User) ld.User {
	id := s.stack

	if user.Org != "" {
		id = string(user.Org)
	}

	if user.Org != "" && user.Name != "" {
		id = fmt.Sprintf("%s:%s", user.Org, user.Name)
	}

	builder := ld.NewUserBuilder(id)
	builder.Custom("stack", ldvalue.String(s.stack))

	if user.Pod != "" {
		builder.Custom("pod", ldvalue.String(user.Pod))
	}

	if user.Org != "" {
		builder.Custom("org", ldvalue.String(user.Org))
	}

	if user.Name != "" {
		builder.Custom("alias", ldvalue.String(user.Name))
	}

	for k, v := range user.Custom {
		builder.Custom(k, ldvalue.CopyArbitraryValue(v))
	}

	return builder.Build()
}

// extractUser gets a feature user from the current context.
func (s *launchDarklyStore) extractUser(ctx context.Context) User {
	user := User{}

	if rc := atlas.GetRequestContext(ctx); rc != nil {
		user.Name = string(rc.IdentityName)
		user.Pod = string(rc.Pod)
		user.Org = string(rc.Org)
	}

	return user
}

// Close shuts down the LaunchDarkly client
func (s *launchDarklyStore) Close() {
	if err := s.client.Close(); err != nil {
		log.Global().Sugar().Warnf("error closing launch darkly client: %v", err)
	}
}
