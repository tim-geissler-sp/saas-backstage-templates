// Copyright (c) 2020. SailPoint Technologies, Inc. All rights reserved.
package event

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/sailpoint/atlas-go/atlas/metric"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sailpoint/atlas-go/atlas/log"
)

// Publisher is an interface that enables external event publication.
type Publisher interface {
	BulkPublish(ctx context.Context, events []EventAndTopic) ([]*FailedEventAndTopic, error)
	Publish(ctx context.Context, td TopicDescriptor, event *Event) error
	PublishToTopic(ctx context.Context, topic Topic, event *Event) error
}

// DefaultPublisher is a publisher implementation that pushes events
// Kafka.
type DefaultPublisher struct {
	p             *kafka.Producer
	uploader      *s3ExternalUploader
	metricsConfig metric.MetricsConfig
}

// NewPublisher constructs a new DefaultPublisher using the specified config.
func NewPublisher(config PublisherConfig, metricsConfig metric.MetricsConfig) (*DefaultPublisher, error) {
	p, err := kafka.NewProducer(&kafka.ConfigMap{
		BootstrapServersConfig: config.BootstrapServers,
		CompressionTypeConfig:  config.CompressionType,
		MessageMaxBytesConfig:  config.MessageMaxBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("create publisher: %w", err)
	}

	uploaderConfig := uploaderConfig{
		bucket:          config.ExternalBucket,
		uploadThreshold: config.MessageMaxBytes - 100000, // arbitrary 100 KB padding for record metadata,
	}
	if uploaderConfig.uploadThreshold < 0 {
		uploaderConfig.uploadThreshold = 0
	}

	uploader := newS3ExternalUploader(uploaderConfig)

	publisher := &DefaultPublisher{
		p:             p,
		uploader:      uploader,
		metricsConfig: metricsConfig,
	}

	return publisher, nil
}

func toKafkaMessage(et EventAndTopic) (*kafka.Message, error) {
	topicID := string(et.Topic.ID())

	eventJSON, err := json.Marshal(et.Event)
	if err != nil {
		return nil, fmt.Errorf("parse event on topic %s: %w", topicID, err)
	}

	msg := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topicID, Partition: kafka.PartitionAny},
		Value:          eventJSON,
		Headers:        getHeaders(et.Event),
	}

	// Set the partition key if specified in the event...
	if key := et.Event.Headers[HeaderKeyPartitionKey]; key != "" {
		msg.Key = []byte(key)
	}

	return msg, nil
}

// BulkPublish publishes a batch of events to Kafka. If any event fails, it will be skipped with a warning log message
func (p *DefaultPublisher) BulkPublish(ctx context.Context, events []EventAndTopic) ([]*FailedEventAndTopic, error) {

	failedEvents := make([]*FailedEventAndTopic, 0, len(events))
	deliveries := make(chan kafka.Event)
	enqueuedEventCount := 0

	for _, et := range events {

		// If large event, upload actual event to S3 and publish compact event to Kafka
		if p.uploader.ShouldUpload(ctx, et.Event) {
			uploadStart := time.Now()

			uploadedEvent, err := p.uploader.Upload(ctx, et.Topic, et.Event)
			if err != nil {
				failedEvents = append(failedEvents, NewFailedFailedEventAndTopic(et, err))
				log.Warnf(ctx, "%v", err)
				continue
			}

			uploadDuration := time.Since(uploadStart)

			log.Infof(ctx, "kafka event stored to s3 with topic %s, eventType %s, groupId %s, and payload size %d. Upload time is %d.",
				string(et.Topic.Name()),
				et.Event.Type,
				et.Event.Headers[HeaderKeyGroupID],
				float64(uploadedEvent.Size),
				uploadDuration)

			s3ObjectKeyJsonBytes, err := json.Marshal(uploadedEvent.Location)
			if err != nil {
				failedEvents = append(failedEvents, NewFailedFailedEventAndTopic(et, err))
				log.Warnf(ctx, "failed to parse large event location %s to JSON: %v", uploadedEvent.Location, err)
				continue
			}

			et.Event.Headers[HeaderKeyIsCompactEvent] = strconv.FormatBool(true)
			et.Event = &Event{
				Headers:     et.Event.Headers,
				ID:          et.Event.ID,
				Timestamp:   et.Event.Timestamp,
				Type:        et.Event.Type,
				ContentJSON: string(s3ObjectKeyJsonBytes),
			}
		}

		msg, err := toKafkaMessage(et)
		if err != nil {
			failedEvents = append(failedEvents, NewFailedFailedEventAndTopic(et, err))
			log.Warnf(ctx, "failed to convert event to kafka message: %e", err)
			continue
		}

		if err := p.p.Produce(msg, deliveries); err != nil {
			failedEvents = append(failedEvents, NewFailedFailedEventAndTopic(et, err))
			log.Warnf(ctx, "failed to enqueue event on topic %s: %v", et.Topic.ID(), err)
			continue
		}

		enqueuedEventCount++

		if enabled, _ := p.metricsConfig.IsNormalizedMetricEnabled(); enabled {
			eventPublishedCountNormalized.WithLabelValues(string(et.Topic.Name()), et.Event.Type).Inc()
		}

		if enabled, _ := p.metricsConfig.IsDeprecatedMetricEnabled(); enabled {
			eventPublishedCount.WithLabelValues(string(et.Topic.Name()), et.Event.Type).Inc()
		}
	}

	for i := 0; i < enqueuedEventCount; i++ {
		select {
		case <-ctx.Done():
			return failedEvents, ctx.Err()
		case e := <-deliveries:
			m := e.(*kafka.Message)

			if m.TopicPartition.Error != nil {
				topicID := ""
				if m.TopicPartition.Topic != nil {
					topicID = *m.TopicPartition.Topic
				}

				log.Warnf(ctx, "failed to publish event to topic %s: %v", topicID, m.TopicPartition.Error)
				var failedEvent Event
				err := json.Unmarshal(m.Value, &failedEvent)
				if err != nil {
					log.Warnf(ctx, "could not unmarshal enqueued kafka msg from topic %s: %v", topicID, err)
					continue
				}
				fEvT := EventAndTopic{}
				fEvT.Event = &failedEvent
				fEvT.Topic, _ = ParseTopic(*m.TopicPartition.Topic)

				thisFailedEventAndTopic := NewFailedFailedEventAndTopic(fEvT, m.TopicPartition.Error)
				failedEvents = append(failedEvents, thisFailedEventAndTopic)

				if enabled, _ := p.metricsConfig.IsNormalizedMetricEnabled(); enabled {
					eventPublishedFailed.WithLabelValues(string(fEvT.Topic.Name()), fEvT.Event.Type).Inc()
				}
			}
		}
	}

	if len(failedEvents) > 0 {
		return failedEvents, errors.New("one or more event failed to send")
	}
	return nil, nil
}

// Publish sends a single event to an IDN Kafka topic
func (p *DefaultPublisher) Publish(ctx context.Context, td TopicDescriptor, et *Event) error {
	topic, err := NewTopic(ctx, td)
	if err != nil {
		return err
	}

	return p.PublishToTopic(ctx, topic, et)
}

// PublishToTopic sends a single event to Kafka.
func (p *DefaultPublisher) PublishToTopic(ctx context.Context, topic Topic, event *Event) error {
	et := EventAndTopic{
		Event: event,
		Topic: topic,
	}

	events := make([]EventAndTopic, 1, 1)
	events[0] = et

	_, err := p.BulkPublish(ctx, events)
	return err
}

// getHeaders returns the Event's groupId and isCompactEvent headers as native, Kafka headers
func getHeaders(event *Event) []kafka.Header {
	headers := make([]kafka.Header, 0, 2)

	if val, keyExists := event.Headers[HeaderKeyGroupID]; keyExists {
		headers = append(headers, kafka.Header{
			Key:   HeaderKeyGroupID,
			Value: []byte(val),
		})
	}

	if val, keyExists := event.Headers[HeaderKeyIsCompactEvent]; keyExists {
		headers = append(headers, kafka.Header{
			Key:   HeaderKeyIsCompactEvent,
			Value: []byte(val),
		})
	}

	return headers
}
