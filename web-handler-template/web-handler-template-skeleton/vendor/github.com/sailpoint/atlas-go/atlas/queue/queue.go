// Copyright (c) 2021, SailPoint Technologies, Inc. All rights reserved.
package queue

import (
	"context"
	"encoding/json"
	"time"
)

// ID is a unique ID for a queue (in SQS, for example, this will be the queue URL)
type ID string

// ReceiptHandle is a token indicating the receipt of a message. It can be used to
// delete or extend the visbility of a message.
type ReceiptHandle string

// CreateQueueOptions are the set of optional parameters that influence how
// a queue is created.
type CreateQueueOptions struct {
	VisibilityTimeout time.Duration
	FIFO              bool
}

// PublishOptions are the set of optional parameters that influence how
// a message is sent to a queue.
type PublishOptions struct {
	DeduplicationID   string
	MessageGroupID    string
	DelayInSeconds    *time.Duration
	MessageAttributes map[string]string
}

// PollOptions are the set of optional parameters that influence how
// a polling operation is performed on a queue.
type PollOptions struct {
	// MaxMessage is the maxmum number of messages to return in this poll operation (default: 1)
	MaxMessages int64

	// AttributeNames are the names of the attributes to request in this poll operation (default: None)
	// "All" can be specified to retrieve all attributes attached to the message.
	AttributeNames []string

	// SystemAttributeNames are the names of the *system* attributes to request in this poll operation (default: None)
	// "All" can be specified to retrieve all attributes related to the message.
	SystemAttributeNames []string

	// VisibilityTimeout is the amount of time before a message becomes visible again after being received.
	VisibilityTimeout time.Duration
}

// Message is the result of polling a queue.
type Message struct {
	ReceivedAt       time.Time
	PayloadJSON      string
	ReceiptHandle    ReceiptHandle
	Attributes       map[string]string
	SystemAttributes map[string]string
}

// UnmarshalPayload unmarshalls the PayloadJSON of a message to the specified object.
func (m *Message) UnmarshalPayload(v interface{}) error {
	return json.Unmarshal([]byte(m.PayloadJSON), v)
}

// MessageCounts is the number of pending and in-flight messages
type MessageCounts struct {
	Pending  int
	InFlight int
}

// Service is an abstract interface for a queueing system.
type Service interface {

	// CreateQueue creates a new queue with the specified name and options.
	CreateQueue(ctx context.Context, name string, options CreateQueueOptions) (ID, error)

	// DeleteQueue deletes the queue with the specified ID.
	DeleteQueue(ctx context.Context, id ID) error

	// Publish will send a message to the queue. The payload is JSON-encoded on the queue.
	Publish(ctx context.Context, id ID, v interface{}, options PublishOptions) error

	// DeleteMessage will remove a message from the queue.
	DeleteMessage(ctx context.Context, id ID, receiptHandle ReceiptHandle) error

	// SetVisibilityTimeout will set the duration of time before the message is made available to other consumers.
	// Note: timeout is the duration *since* the message was received - **NOT** the duration added to the current time.
	SetVisibilityTimeout(ctx context.Context, id ID, receiptHandle ReceiptHandle, timeout time.Duration) error

	// Poll reads messages from the queue. This is a long-polling operation, where timeout is the duration
	// of the long-poll request.
	Poll(ctx context.Context, id ID, timeout time.Duration, options PollOptions) ([]Message, error)

	// MessageCounts returns the count of pending and in-flight messages in the queue
	MessageCounts(ctx context.Context, id ID) (*MessageCounts, error)
}
