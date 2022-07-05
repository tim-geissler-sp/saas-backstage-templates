// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package message

import (
	"context"
	"time"
)

// Priority defines the  priority of a given message
type Priority string

const (
	PriorityHigh   = "HIGH"
	PriorityMedium = "MEDIUM"
	PriorityLow    = "LOW"
)

// PublishOptions
type PublishOptions struct {
	Delay    time.Duration
	Priority Priority
}

// Publisher is an interface that enables message publication.
type Publisher interface {
	PublishAtomic(ctx context.Context, scope Scope, message *Message, options PublishOptions) error
	PublishAtomicFromContext(ctx context.Context, sd ScopeDescriptor, message *Message, options PublishOptions) error
}
