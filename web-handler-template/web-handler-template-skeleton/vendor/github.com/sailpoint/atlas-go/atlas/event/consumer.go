// Copyright (c) 2020-2022. SailPoint Technologies, Inc. All rights reserved.
package event

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sailpoint/atlas-go/atlas/metric"

	"github.com/cenkalti/backoff/v4"
	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/log"
)

// LargeEventStore is an interface for downloading data for large events.
type LargeEventStore interface {
	Download(ctx context.Context, location string) (*Event, error)
}

// kafkaEventConsumer is the main entity responsible for efficiently polling Kafka and
// concurrently submitting events to the application for processing.
type kafkaEventConsumer struct {
	consumer        *kafka.Consumer
	config          ConsumerConfig
	metricsConfig   metric.MetricsConfig
	largeEventStore LargeEventStore
	batchSize       int
	errorBackoff    backoff.BackOff
}

// topicPartition is a simple tuple representing a kafka topic and it's partition. Used
// here instead of the native one from the confluent library so that it serve
// as a map key.
type topicPartition struct {
	topic     string
	partition int32
}

// messageBatch is a set of records returned from a polling operation.
type messageBatch struct {
	messages     map[topicPartition][]*kafka.Message
	messageCount int
}

// newMessageBatch constructs a new, empty batch.
func newMessageBatch() messageBatch {
	b := messageBatch{}
	b.messages = make(map[topicPartition][]*kafka.Message)

	return b
}

// addMessages adds a message to the batch.
func (b *messageBatch) addMessage(msg *kafka.Message) error {
	tp, err := newTopicPartition(msg.TopicPartition)
	if err != nil {
		return err
	}

	b.messages[tp] = append(b.messages[tp], msg)
	b.messageCount++

	return nil
}

// newTopicPartition convets a kafka.TopicPartition to our internal topicPartition.
func newTopicPartition(tp kafka.TopicPartition) (topicPartition, error) {
	if tp.Topic == nil {
		return topicPartition{}, fmt.Errorf("topic is required")
	}

	return topicPartition{
		topic:     *tp.Topic,
		partition: tp.Partition,
	}, nil
}

// StartConsumer runs a consumer process that process until a context is closed.
func StartConsumer(ctx context.Context, config ConsumerConfig, handler Handler, metricsConfig metric.MetricsConfig) error {
	c, err := newKafkaEventConsumer(config, metricsConfig)
	if err != nil {
		return err
	}

	defer c.close(ctx)

	return c.run(ctx, handler)
}

// newKafkaEventConsumer constructs a new consumer instance based on the specified configuration.
func newKafkaEventConsumer(config ConsumerConfig, metricsConfig metric.MetricsConfig) (*kafkaEventConsumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		BootstrapServersConfig:             config.BootstrapServers,
		GroupIdConfig:                      config.GroupID,
		MessageMaxBytesConfig:              config.MessageMaxBytes,
		MaxPartitionFetchBytesConfig:       config.MaxPartitionFetchBytes,
		MaxPollIntervalMsConfig:            config.MaxPollIntervalMs,
		SessionTimeoutMsConfig:             config.SessionTimeoutMs,
		HeartbeatIntervalMsConfig:          config.HeartbeatIntervalMs,
		AutoOffsetResetConfig:              config.AutoOffsetReset,
		PartitionAssignmentStrategyConfig:  config.PartitionAssignmentStrategy,
		GoApplicationRebalanceEnableConfig: true,
		EnableAutoCommitConfig:             true, // must always be true for atlas-go consumer
		EnableAutoOffsetStoreConfig:        false,
		StatisticsIntervalMsConfig:         300000,
	})

	if err != nil {
		return nil, fmt.Errorf("kafka consumer start: %w", err)
	}

	largeEventStore := newS3ExternalDownloader(downloaderConfig{bucket: config.ExternalBucket})

	batchSize := config.MaxPollRecords
	if batchSize <= 0 {
		batchSize = 64
	}

	topics := make([]string, 0)
	for _, t := range config.Topics {
		topics = append(topics, buildTopicRegexes(t, config.Pods)...)
	}

	errorBackoff := backoff.NewExponentialBackOff()
	errorBackoff.MaxInterval = 30 * time.Second

	c := &kafkaEventConsumer{}
	c.consumer = consumer
	c.config = config
	c.metricsConfig = metricsConfig
	c.largeEventStore = largeEventStore
	c.batchSize = batchSize
	c.errorBackoff = errorBackoff

	return c, nil
}

// run starts the consumer polling loop and invokes the specified handler for each incoming event.
// This operation will last until ctx is cancelled.
func (c *kafkaEventConsumer) run(ctx context.Context, handler Handler) error {
	topics := make([]string, 0)
	for _, t := range c.config.Topics {
		topics = append(topics, buildTopicRegexes(t, c.config.Pods)...)
	}

	log.Infof(ctx, "subscribing to topics: %v", topics)
	if err := c.consumer.SubscribeTopics(topics, nil); err != nil {
		return fmt.Errorf("kafka topic subscribe: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		batch, err := c.pollBatch(ctx)
		if err != nil {
			sleepDuration := c.errorBackoff.NextBackOff()
			log.Errorf(ctx, "error polling kafka, sleeping for %s: %e", sleepDuration, err)
			atlas.SleepWithContext(ctx, sleepDuration)
			continue
		}
		c.errorBackoff.Reset()

		if batch.messageCount == 0 {
			continue
		}

		for tp, messages := range batch.messages {
			topicPartition := tp.toKafkaTopicPartition()
			c.pause(ctx, topicPartition)

			go c.processPartition(ctx, topicPartition, messages, handler)
		}
	}
}

// pause will pause the specified partition, an error here will panic and kill the application as it
// should never happen and will result in out of order processing
func (c *kafkaEventConsumer) pause(ctx context.Context, tp kafka.TopicPartition) {
	if err := c.consumer.Pause([]kafka.TopicPartition{tp}); err != nil {
		log.Fatalf(ctx, "error pausing partitions: %v", err)
	}
}

// resume will resume the specified partition, an error here will panic and kill the application as it
// should never happen and will result in failure to handle an assigned partition
func (c *kafkaEventConsumer) resume(ctx context.Context, tp kafka.TopicPartition) {
	if err := c.consumer.Resume([]kafka.TopicPartition{tp}); err != nil {
		log.Fatalf(ctx, "error resuming partitions: %v", err)
	}
}

// storeMessage records the message's offset to be committed at the next interval.
func (c *kafkaEventConsumer) storeMessage(ctx context.Context, msg *kafka.Message) {
	if _, err := c.consumer.StoreMessage(msg); err != nil {
		log.Errorf(ctx, "error storing offsets for message: %v", err)
	}
}

// close cleanly shuts down the event consumer, flushing any remaining offsets.
func (c *kafkaEventConsumer) close(ctx context.Context) {
	if err := c.consumer.Close(); err != nil {
		log.Warnf(ctx, "error closing kafka consumer: %v", err)
	}
}

// toKafkaTopicPartition converts our internal topicPartition representation to
// one used by the client library.
func (tp *topicPartition) toKafkaTopicPartition() kafka.TopicPartition {
	return kafka.TopicPartition{
		Topic:     &tp.topic,
		Partition: tp.partition,
	}
}

// writePartitionedMessages splits messages into runs that can be executed in parallel. Messages
// within a run must be executed sequentially. Each run is written to the specified
// output channel
func writePartitionedMessages(messages []*kafka.Message, out chan<- []*kafka.Message) {
	byKey := make(map[string][]*kafka.Message)

	for _, msg := range messages {
		key := string(msg.Key)

		if key == "" {
			out <- []*kafka.Message{msg}
		} else {
			byKey[key] = append(byKey[key], msg)
		}
	}

	for _, messagesForKey := range byKey {
		out <- messagesForKey
	}
}

// processPartition handles a set of messages for the same partition. It implements key-level parallelism, where messages with the same
// partition key are handled in order, but messages with different partition keys are handled concurrently and in arbitrary order.
// After all messages for the partition have been processed, offsets are stored in the consumer to be committed.
func (c *kafkaEventConsumer) processPartition(ctx context.Context, topicPartition kafka.TopicPartition, messages []*kafka.Message, handler Handler) {
	defer c.resume(ctx, topicPartition)

	partitions := make(chan []*kafka.Message)

	workerCount := c.config.MaxPartitionConcurrency
	if workerCount < 1 {
		workerCount = len(messages)
	}

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		go func() {
			defer wg.Done()

			for partition := range partitions {
				c.handleMessages(ctx, partition, handler)
			}
		}()
	}

	writePartitionedMessages(messages, partitions)
	close(partitions)

	wg.Wait()

	// Store message offsets to be committed by the client
	for _, msg := range messages {
		c.storeMessage(ctx, msg)
	}
}

// handleMessages invokes the message handler on a slice of kafka messages synchronously and in order
func (c *kafkaEventConsumer) handleMessages(ctx context.Context, messages []*kafka.Message, handler Handler) {
	for _, msg := range messages {
		c.handleMessage(ctx, msg, handler)
	}
}

// getHeader gets a string header from a Kafka message.
func getHeader(msg *kafka.Message, key string) string {
	for _, value := range msg.Headers {
		if strings.EqualFold(value.Key, key) {
			return string(value.Value)
		}
	}

	return ""
}

// getHeaderBool gets a boolean header from a Kafka message.
func getHeaderBool(msg *kafka.Message, key string) bool {
	value, _ := strconv.ParseBool(getHeader(msg, key))
	return value
}

// toEvent converts a Kafka message to an atlas Event, downloading from S3 if necessary.
func (c *kafkaEventConsumer) toEvent(ctx context.Context, topic Topic, msg *kafka.Message) (*Event, error) {
	var rawEvent Event
	if err := json.Unmarshal(msg.Value, &rawEvent); err != nil {
		return nil, fmt.Errorf("parse event: %w", err)
	}

	if !getHeaderBool(msg, HeaderKeyIsCompactEvent) {
		return &rawEvent, nil
	}

	// If compact event, download and handle actual event from S3
	var s3ObjectKey string
	if err := json.Unmarshal([]byte(rawEvent.ContentJSON), &s3ObjectKey); err != nil {
		return nil, fmt.Errorf("parse large event s3 location %s: %w", rawEvent.ContentJSON, err)
	}

	event, err := c.largeEventStore.Download(ctx, s3ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("download event from s3 key '%s': %w", s3ObjectKey, err)
	}

	return event, nil
}

// handleMessages parses, downloads (if necessary) the event, and hands it off to the
// specified event handler. This method does *NOT* return an error.
func (c *kafkaEventConsumer) handleMessage(ctx context.Context, msg *kafka.Message, handler Handler) {
	if groupID := getHeader(msg, HeaderKeyGroupID); groupID != "" {
		if !strings.EqualFold(groupID, c.config.GroupID) {
			// This event is targeted at another service
			return
		}
	}

	topic, err := ParseTopic(*msg.TopicPartition.Topic)
	if err != nil {
		log.Errorf(ctx, "invalid topic: %v", err)
		return
	}

	event, err := c.toEvent(ctx, topic, msg)
	if err != nil {
		log.Errorf(ctx, "error converting message to event: %v", err)
		return
	}

	if groupID := event.Headers[HeaderKeyGroupID]; groupID != "" {
		if !strings.EqualFold(groupID, c.config.GroupID) {
			// This should have been pruned out via Kafka header...
			// Remove when this is no longer happening
			log.Warnf(ctx, "skipping event targeted at group: %s", groupID)
			return
		}
	}

	handlerStart := time.Now()
	err = handler.HandleEvent(ctx, topic, event)
	handlerDuration := time.Since(handlerStart)

	labels := prometheus.Labels{
		"pod":       event.Headers[HeaderKeyPod],
		"org":       event.Headers[HeaderKeyOrg],
		"name":      string(topic.Name()),
		"eventType": event.Type,
		"groupId":   event.Headers[HeaderKeyGroupID],
	}

	reducedLabels := prometheus.Labels{
		"pod":       event.Headers[HeaderKeyPod],
		"name":      string(topic.Name()),
		"eventType": event.Type,
		"groupId":   event.Headers[HeaderKeyGroupID],
	}

	if err != nil {
		log.Errorf(ctx, "event handler failed: %v", err)

		if c.isNormalizedMetricEnabled() {
			eventHandlerFailed.With(labels).Inc()
		}
	} else if c.isNormalizedMetricEnabled() {
		eventHandlerSuccess.With(labels).Inc()
	}

	if c.isNormalizedMetricEnabled() {
		eventLatency := handlerStart.Sub(time.Time(event.Timestamp))
		eventHandlerDurationSummary.With(labels).Observe(float64(handlerDuration.Seconds()))

		eventProcessingLatency.With(reducedLabels).Observe(float64(eventLatency.Seconds()))
		eventHandlerDuration.With(reducedLabels).Observe(float64(handlerDuration.Seconds()))
	}
}

// isNormalizedMetricEnabled gets whether or not the feature flag for normalized metrics
// is enabled for this service.
func (c *kafkaEventConsumer) isNormalizedMetricEnabled() bool {
	enabled, _ := c.metricsConfig.IsNormalizedMetricEnabled()
	return enabled
}

// pollBatch pulls in the next batch of events to handle for the consumer.
func (c *kafkaEventConsumer) pollBatch(ctx context.Context) (messageBatch, error) {
	batch := newMessageBatch()

	for batch.messageCount < c.batchSize {

		// Check to see if the context is done..
		select {
		case <-ctx.Done():
			return batch, ctx.Err()
		default:
		}

		// First timeout will long-poll, subsequent will return as quickly as possible
		timeout := 5000
		if batch.messageCount > 0 {
			timeout = 0
		}

		// Ask for the next event from Kafka
		ev := c.consumer.Poll(timeout)
		if ev == nil {
			break
		}

		switch e := ev.(type) {

		// Partitions assigned to this consumer group...
		case kafka.AssignedPartitions:
			log.Infof(ctx, "assigned partitions: %v", e)
			if err := c.consumer.Assign(e.Partitions); err != nil {
				log.Errorf(ctx, "error assigning partitions: %v", err)
			}

		// Partitions revoked from this consumer group...
		case kafka.RevokedPartitions:
			eventConsumerPartitionsRevokedFreq.Inc()
			log.Infof(ctx, "revoked partitions: %v", e)
			if err := c.consumer.Unassign(); err != nil {
				log.Errorf(ctx, "error revoking partitions: %v", err)
			}

		// A message was polled...
		case *kafka.Message:
			if e.TopicPartition.Error != nil {
				log.Errorf(ctx, "topic partition error: %v", e.TopicPartition.Error)
				break
			}

			// Failure here indicates a missing or invalid topic in the kafka message
			if err := batch.addMessage(e); err != nil {
				log.Warnf(ctx, "error adding message to batch: %v", err)
			}

		// An error was returned from Kafka
		case kafka.Error:
			log.Errorf(ctx, "kafka error: %v", e.Error())

			if batch.messageCount > 0 {
				// TODO: what kind of errors might this be -  should we log and continue processing?
				return batch, nil
			}

			return batch, fmt.Errorf("kafka error: %s", e.Error())

		case *kafka.Stats:
			go c.reportConsumerLag(ctx, e)
		}
	}

	return batch, nil
}

// reportConsumerLag takes Kafka Stats event and reports to Prometheus consumer lag metrics
// for topic partitions assigned to this consumer
//
// Kafka Stats event only returns statistics for partitions assigned to this consumer
func (c *kafkaEventConsumer) reportConsumerLag(ctx context.Context, e *kafka.Stats) {
	assigned, err := c.consumer.Assignment()
	if err != nil {
		log.Warnf(ctx, "failed to get consumer partition assignments to report consumer lag: %v", err)
		return
	}

	if len(assigned) < 1 {
		return
	}

	if c.isNormalizedMetricEnabled() {
		s := &stats{}
		if err := json.Unmarshal([]byte(e.String()), s); err != nil {
			log.Warnf(ctx, "failed to unmarshal kafka stats: %v", err)
			return
		}

		for _, tp := range assigned {
			topic := *tp.Topic
			partition := strconv.Itoa(int(tp.Partition))
			eventConsumerLag.With(prometheus.Labels{"topic": topic, "partition": partition}).
				Set(float64(s.getConsumerLag(topic, partition)))
		}
	}
}

// buildTopicRegexes takes a TopicDescriptor and set of pods and returns a list of regex strings
// suitable for passing to Kafka's consumer configuration.
func buildTopicRegexes(topic TopicDescriptor, pods []atlas.Pod) []string {
	var result []string

	switch topic.Scope() {
	case TopicScopeGlobal:
		result = append(result, string(topic.Name()))
	case TopicScopePod:
		for _, p := range pods {
			result = append(result, fmt.Sprintf("%s__%s", topic.Name(), string(p)))
		}
	case TopicScopeOrg:
		for _, p := range pods {
			result = append(result, fmt.Sprintf("^%s__%s__.+", topic.Name(), string(p)))
		}
	}

	return result
}
