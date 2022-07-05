// Copyright (c) 2021. SailPoint Technologies, Inc. All rights reserved.

// Package metric contains data and functions relevant to metrics in an atlas-go based application.
package metric

import "github.com/sailpoint/atlas-go/atlas/feature"

const(
	// normalizedMetricFlag is the name of the feature flag controlling the enablement of the normalized metrics
	normalizedMetricFlag = "PLAT_ENABLE_NORMALIZED_METRICS"

	// deprecatedMetricFlag is the name of the feature flag controlling the disabling of the deprecated metrics
	deprecatedMetricFlag = "PLAT_DISABLE_DEPRECATED_METRICS"
)

// MetricsConfig provides an interface to determine if specific metrics are enabled or disabled.
type MetricsConfig interface {
	IsNormalizedMetricEnabled() (bool, error)
	IsDeprecatedMetricEnabled() (bool, error)
}

// FeatureFlagMetricsConfig is an implementation of MetricsConfig that is backed by feature flags.
type FeatureFlagMetricsConfig struct {
	featureUser feature.User
	store feature.Store
}

// NewMetricsConfig creates a new instance of the FeatureFlagMetricsConfig.
func NewMetricsConfig(store feature.Store) *FeatureFlagMetricsConfig {
	stackUser := feature.User{
		Org: "no-context",
		Pod: "no-context",
	}

	return &FeatureFlagMetricsConfig{
		featureUser: stackUser,
		store: store,
	}
}

// IsNormalizedMetricEnabled returns whether the normalized metrics are enabled or an error.
func (mc *FeatureFlagMetricsConfig) IsNormalizedMetricEnabled() (bool, error) {
	return mc.store.IsEnabledForUser(mc.featureUser, normalizedMetricFlag, false)
}

// IsDeprecatedMetricEnabled returns whether the deprecated metrics are enabled or an error.
func (mc *FeatureFlagMetricsConfig) IsDeprecatedMetricEnabled() (bool, error) {
	enabled, err := mc.store.IsEnabledForUser(mc.featureUser, deprecatedMetricFlag, false)

	return !enabled, err
}
