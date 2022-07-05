// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package event

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/log"
	"github.com/sailpoint/atlas-go/atlas/trace"
	"go.uber.org/zap"
)

// Middleware is an interface for types that act as middleware. Middleware is code that executes for every event in the pipeline.
type Middleware interface {
	Middleware(next Handler) Handler
}

// MiddlewareFunc is a function alias for middleware. Middleware is code that executes for every event in the pipeline.
type MiddlewareFunc func(Handler) Handler

// Middleware delegates to the bound funcion, making MiddlewareFunc implement the Filter interface.
func (f MiddlewareFunc) Middleware(next Handler) Handler {
	return f(next)
}

// SetupTracingContext returns a MiddlewareFunc that sets up an atlas TracingContext
// instance by parsing data from event headers.
func SetupTracingContext() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, topic Topic, e *Event) error {
			tc := trace.NewTracingContext(trace.RequestID(e.Headers[HeaderKeyRequestID]))

			ctx = trace.WithTracingContext(ctx, tc)
			ctx = log.WithFields(ctx,
				zap.String("request_id", string(tc.RequestID)),
				zap.String("span_id", string(tc.SpanID)),
			)

			return next.HandleEvent(ctx, topic, e)
		})
	}

}

// SetupRequestContext returns a MiddlewareFunc that sets up an atlas RequestContext
// instance by parsing data from event headers.
func SetupRequestContext() MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, topic Topic, e *Event) error {
			rc := &atlas.RequestContext{
				TenantID: atlas.TenantID(e.Headers[HeaderKeyTenantID]),
				Pod:      atlas.Pod(e.Headers[HeaderKeyPod]),
				Org:      atlas.Org(e.Headers[HeaderKeyOrg]),
			}

			ctx = atlas.WithRequestContext(ctx, rc)
			ctx = log.WithFields(ctx,
				zap.String("pod", string(rc.Pod)),
				zap.String("org", string(rc.Org)),
				zap.String("event_topic", string(topic.Name())),
				zap.String("event_type", e.Type),
			)

			return next.HandleEvent(ctx, topic, e)
		})
	}
}

// EventMetrics returns a MiddlewareFunc that captures the default set of event handling metrics.
func EventMetrics() MiddlewareFunc {
	eventDurations := promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name: "event_handled_duration",
	}, []string{"topic", "type", "success"})

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, topic Topic, event *Event) error {
			start := time.Now()
			err := next.HandleEvent(ctx, topic, event)
			dt := time.Since(start)

			success := "true"
			if err != nil {
				success = "false"
			}

			eventDurations.WithLabelValues(string(topic.Name()), event.Type, success).Observe(float64(dt.Seconds()))

			return err
		})
	}
}

// LocalRetry returns a MiddlewareFunc that retries a failed downstream handler
// according to the specified policy.
func LocalRetry(backoffPolicy backoff.BackOff) MiddlewareFunc {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, topic Topic, e *Event) error {
			return backoff.Retry(func() error {
				return next.HandleEvent(ctx, topic, e)
			}, backoffPolicy)
		})
	}
}
