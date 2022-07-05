// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package trace

import (
	"context"

	"github.com/google/uuid"
)

// RequestID is a unique UUID for a request. (eg. "68df224b-535c-4b03-8d33-05b08fa2eebe"). Request IDs propagate
// across service boundaries via HTTP, messaging, and events.
type RequestID string

// SpanID is a unique UUID for a span (subsequence within a request).
type SpanID string

// TracingContext holds the information used to trace requests across service boundaries.
type TracingContext struct {
	RequestID RequestID
	SpanID    SpanID
}

type contextKey int

const (
	tracingContextKey contextKey = iota
)

// GetTracingContext extracts a TracingContext from the specified context. Returns nil if
// no TracingContext is associated.
func GetTracingContext(ctx context.Context) *TracingContext {
	v := ctx.Value(tracingContextKey)

	if v == nil {
		return nil
	}

	return v.(*TracingContext)
}

// WithTracingContext contructs a new context derived from ctx that contains
// the specified TracingContext.
func WithTracingContext(ctx context.Context, tc *TracingContext) context.Context {
	return context.WithValue(ctx, tracingContextKey, tc)
}

// NewTracingContext constructs a new TracingContext, using the passed-in RequestID.
// If requestID is empty, then a new RequestID is generated.
func NewTracingContext(requestID RequestID) *TracingContext {
	if requestID == "" {
		requestID = newRequestID()
	}

	return &TracingContext{
		RequestID: requestID,
		SpanID:    newSpanID(),
	}
}

// newRequestID generates a new random RequestID.
func newRequestID() RequestID {
	return RequestID(uuid.New().String())
}

// NewSpanID generates a new random SpanID.
func newSpanID() SpanID {
	return SpanID(uuid.New().String())
}
