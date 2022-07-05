// Copyright (c) 2022. SailPoint Technologies, Inc. All rights reserved.

package event

import (
	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/config"
)

// For complete list of configurations for librdkafka,
// see https://github.com/edenhill/librdkafka/blob/master/CONFIGURATION.md

// Config that applies to both producer and consumer
const (
	BootstrapServersConfig     = "bootstrap.servers"
	MessageMaxBytesConfig      = "message.max.bytes" // in librdkafka, this is max.request.size
	StatisticsIntervalMsConfig = "statistics.interval.ms"

	DefaultMessageMaxBytes = 1000000 // 1 MB
)

// Producer-only config
const (
	CompressionTypeConfig = "compression.type"

	// These values are defaults for atlas-go, not necessarily defaults for kafka producer
	DefaultCompressionType = "gzip"
)

// PublisherConfig holds the required publisher configuration.
type PublisherConfig struct {
	BootstrapServers string
	CompressionType  string
	MessageMaxBytes  int
	ExternalBucket   string
}

// NewPublisherConfig reads PublisherConfig from a configuration source.
func NewPublisherConfig(cfg config.Source) PublisherConfig {
	c := PublisherConfig{
		BootstrapServers: config.GetString(cfg, "ATLAS_KAFKA_SERVERS", "localhost:9092"),
		CompressionType:  config.GetString(cfg, "IRIS_KAFKA_COMPRESSION_TYPE", DefaultCompressionType),
		MessageMaxBytes:  config.GetInt(cfg, "IRIS_KAFKA_MAX_MSG_SIZE_BYTE", DefaultMessageMaxBytes),
		ExternalBucket:   config.GetString(cfg, "ATLAS_KAFKA_S3_BUCKET", ""),
	}

	return c
}

// Consumer-only config
const (
	GroupIdConfig                     = "group.id"
	SessionTimeoutMsConfig            = "session.timeout.ms"
	HeartbeatIntervalMsConfig         = "heartbeat.interval.ms"
	AutoOffsetResetConfig             = "auto.offset.reset"
	MaxPartitionFetchBytesConfig      = "max.partition.fetch.bytes"
	MaxPollIntervalMsConfig           = "max.poll.interval.ms"
	PartitionAssignmentStrategyConfig = "partition.assignment.strategy"
	EnableAutoCommitConfig            = "enable.auto.commit"
	EnableAutoOffsetStoreConfig       = "enable.auto.offset.store"

	// See doc in https://github.com/confluentinc/confluent-kafka-go/blob/bb5bb31194f8046f3c9f09fbf0a3aae460a02000/kafka/consumer.go#L360
	GoApplicationRebalanceEnableConfig = "go.application.rebalance.enable"

	// These values are defaults for atlas-go, not necessarily defaults for kafka consumer
	DefaultMaxPartitionFetchBytes = 1048576
	DefaultMaxPollIntervalMs      = 300000
	DefaultSessionTimeoutMs       = 45000
	DefaultHeartbeatIntervalMs    = 3000
	DefaultAutoOffsetReset        = "earliest"
)

// ConsumerConfig is the required configuration for starting a new consumer.
// max.poll.records config does not exist in librdkafka
type ConsumerConfig struct {
	BootstrapServers            string
	GroupID                     string
	Topics                      []TopicDescriptor
	Pods                        []atlas.Pod
	MessageMaxBytes             int
	MaxPartitionFetchBytes      int
	MaxPollIntervalMs           int
	SessionTimeoutMs            int
	HeartbeatIntervalMs         int
	AutoOffsetReset             string
	ExternalBucket              string
	PartitionAssignmentStrategy string
	MaxPollRecords              int
	MaxPartitionConcurrency     int
}

// NewConsumerConfig reads ConsumerConfig from a configuration source.
func NewConsumerConfig(cfg config.Source) ConsumerConfig {
	c := ConsumerConfig{}
	c.BootstrapServers = config.GetString(cfg, "ATLAS_KAFKA_SERVERS", "localhost:9092")

	pods := config.GetStringSlice(cfg, "ATLAS_PODS", []string{"dev"})
	for _, pod := range pods {
		c.Pods = append(c.Pods, atlas.Pod(pod))
	}

	c.MessageMaxBytes = config.GetInt(cfg, "IRIS_KAFKA_MAX_MSG_SIZE_BYTE", DefaultMessageMaxBytes)
	c.MaxPartitionFetchBytes = config.GetInt(cfg, "IRIS_KAFKA_MAX_MSG_SIZE_BYTE", DefaultMaxPartitionFetchBytes)
	c.MaxPollIntervalMs = config.GetInt(cfg, "IRIS_KAFKA_MAX_POLL_INTERVAL", DefaultMaxPollIntervalMs)
	c.SessionTimeoutMs = config.GetInt(cfg, "IRIS_KAFKA_SESSION_TIMEOUT", DefaultSessionTimeoutMs)
	c.HeartbeatIntervalMs = config.GetInt(cfg, "IRIS_KAFKA_HEARTBEAT_INTERVAL", DefaultHeartbeatIntervalMs)
	c.AutoOffsetReset = config.GetString(cfg, "IRIS_KAFKA_AUTO_OFFSET_RESET", DefaultAutoOffsetReset)
	c.ExternalBucket = config.GetString(cfg, "ATLAS_KAFKA_S3_BUCKET", "")
	c.MaxPollRecords = config.GetInt(cfg, "IRIS_KAFKA_MAX_POLL_RECORDS", 64)
	c.MaxPartitionConcurrency = config.GetInt(cfg, "ATLAS_IRIS_CONFIG_MAX_PARTITION_CONCURRENCY", 0)

	useRoundRobin := config.GetBool(cfg, "IRIS_KAFKA_ROUND_ROBIN", false)
	if !useRoundRobin {
		c.PartitionAssignmentStrategy = "range"
	} else {
		c.PartitionAssignmentStrategy = "roundrobin,range"
	}

	return c
}
