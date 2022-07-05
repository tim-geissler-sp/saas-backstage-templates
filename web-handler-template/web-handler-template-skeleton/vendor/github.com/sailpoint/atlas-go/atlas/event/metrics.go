// Copyright (c) 2022. SailPoint Technologies, Inc. All rights reserved.
package event

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// eventConsumerLag is a metric that measures how many events in a topic
// the consumers have not yet processed.
var eventConsumerLag = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "kafka_event_consumer_lag",
	Help: "The difference between the latest offset and the current offset for a consumer of a topic",
}, []string{"topic", "partition"})

// eventProcessingLatency is a metric that times the latency between event publish and
// the consumer receiving the event.
var eventProcessingLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "kafka_event_processing_latency_seconds",
	Help:    "The amount of time between when an event was submitted and when it was processed",
	Buckets: []float64{0.1, 0.5, 1.0, 5.0, 15.0, 30.0, 60.0, 120.0, 180.0, 300.0, 600.0},
}, []string{"pod", "name", "eventType", "groupId"})

// eventHandlerDuration is a metric that times the duration of a message handler invocation.
var eventHandlerDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "kafka_event_consumer_duration_seconds",
	Help:    "The amount of time a consumer takes to handle an event",
	Buckets: []float64{0.1, 0.5, 1.0, 5.0, 15.0, 30.0, 60.0, 120.0, 180.0, 300.0, 600.0},
}, []string{"pod", "name", "eventType", "groupId"})

var eventHandlerDurationSummary = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Name: "kafka_event_consumer_duration_summary",
	Help: "Event consumer processing duration summary",
}, []string{"org", "pod", "name", "eventType", "groupId"})

// eventHandlerSuccess is a metric that counts the number of events successfully handled.
var eventHandlerSuccess = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kafka_event_processed_success",
	Help: "The number of events successfully handled",
}, []string{"pod", "org", "name", "eventType", "groupId"})

// eventHandlerFailed is a metric that counts the number of event handler failures.
var eventHandlerFailed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kafka_event_processed_failure",
	Help: "The number of events that were not successfully handled",
}, []string{"pod", "org", "name", "eventType", "groupId"})

// eventPublishedCount is a counter metric that keeps track of how many
// events have been published.
var eventPublishedCount = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "event_published_count",
}, []string{"topic", "type"})

// eventPublishedCountNormalized is a counter metric that keeps track of how many
// events have been published.
var eventPublishedCountNormalized = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kafka_event_published",
	Help: "The number of events published",
}, []string{"topic", "eventType"})

// eventPublishedFailed is a counter metric that keeps track of how many
// events have failed to be published.
var eventPublishedFailed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kafka_event_published_failed",
	Help: "The number of events failed to publish",
}, []string{"topic", "eventType"})

// eventConsumerPartitionsRevokedFreq is a counter metric that keeps track of the number
// of times a consumer's assigned partitions have been revoked.
var eventConsumerPartitionsRevokedFreq = promauto.NewCounter(prometheus.CounterOpts{
	Name: "kafka_partitions_revoked_frequency",
	Help: "The number of times assigned partitions have been revoked from a consumer due to group rebalance",
})
