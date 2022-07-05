// Copyright (c) 2021, SailPoint Technologies, Inc. All rights reserved.
package queue

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/google/uuid"
	"github.com/sailpoint/atlas-go/atlas/config"
)

type sqsQueueService struct {
	sqs *sqs.SQS
}

// NewSqsQueueService creates a new instance of sqsQueueService
func NewSqsQueueService(cfgs ...*aws.Config) Service {
	q := &sqsQueueService{sqs: sqs.New(config.GlobalAwsSession(), cfgs...)}

	return q
}

// CreateQueue creates a new SQS queue with given name and CreateQueueOptions.
// If a queue already exists, finds and returns the ID of existing queue.
//
// Colon(:) character(s) in the queue name is replaced with underscore(_) character(s).
// If provided VisibilityTimeout option is 0, then default to 5 minutes.
func (q *sqsQueueService) CreateQueue(ctx context.Context, name string, options CreateQueueOptions) (ID, error) {
	if options.VisibilityTimeout == 0 {
		options.VisibilityTimeout = 5 * time.Minute
	}

	queueName := strings.ReplaceAll(name, ":", "_")

	input := &sqs.CreateQueueInput{}
	input.SetQueueName(queueName)

	attributes := make(map[string]*string)
	attributes[sqs.QueueAttributeNameVisibilityTimeout] = aws.String(strconv.Itoa(int(options.VisibilityTimeout.Seconds())))

	if options.FIFO {
		attributes[sqs.QueueAttributeNameFifoQueue] = aws.String("true")
	}

	input.SetAttributes(attributes)

	output, err := q.sqs.CreateQueueWithContext(ctx, input)

	if err == nil {
		return ID(*output.QueueUrl), nil
	}

	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case sqs.ErrCodeQueueNameExists:
			urlInput := &sqs.GetQueueUrlInput{QueueName: aws.String(queueName)}
			urlOutput, err := q.sqs.GetQueueUrlWithContext(ctx, urlInput)
			if err != nil {
				return "", err
			}

			return ID(*urlOutput.QueueUrl), nil
		}
	}

	return "", err
}

// DeleteQueue deletes a SQS queue of given ID.
func (q *sqsQueueService) DeleteQueue(ctx context.Context, id ID) error {
	input := &sqs.DeleteQueueInput{}
	input.SetQueueUrl(string(id))

	if _, err := q.sqs.DeleteQueueWithContext(ctx, input); err != nil {
		return err
	}

	return nil
}

// Publish sends a message to a SQS queue of given ID with specified PublishOptions.
func (q *sqsQueueService) Publish(ctx context.Context, id ID, v interface{}, options PublishOptions) error {
	jsBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}

	js := string(jsBytes)

	input := &sqs.SendMessageInput{}
	input.SetQueueUrl(string(id))
	input.SetMessageBody(js)

	if strings.HasSuffix(string(id), ".fifo") {
		if options.MessageGroupID == "" {
			options.MessageGroupID = uuid.New().String()
		}
		input.SetMessageGroupId(options.MessageGroupID)

		if options.DeduplicationID == "" {
			options.DeduplicationID = uuid.New().String()
		}
		input.SetMessageDeduplicationId(options.DeduplicationID)
	} else {
		//DelayInSeconds can be set for individual messages only for non-Fifo queues
		if options.DelayInSeconds != nil {
			delaySeconds, err := getDelaySeconds(options)
			if err != nil {
				return err
			}
			input.SetDelaySeconds(delaySeconds)
		}
	}

	if options.MessageAttributes != nil {
		attributes := make(map[string]*sqs.MessageAttributeValue)
		for k, v := range options.MessageAttributes {
			value := &sqs.MessageAttributeValue{}
			value.SetDataType("String")
			value.SetStringValue(v)

			attributes[k] = value
		}
		input.SetMessageAttributes(attributes)
	}

	if _, err := q.sqs.SendMessageWithContext(ctx, input); err != nil {
		return err
	}

	return nil
}

// Poll receives message(s) from a SQS queue of given ID, with option to wait (long poll) via timeout parameter.
func (q *sqsQueueService) Poll(ctx context.Context, id ID, timeout time.Duration, options PollOptions) ([]Message, error) {
	if options.MaxMessages <= 0 {
		options.MaxMessages = 1
	}

	input := &sqs.ReceiveMessageInput{}
	input.SetQueueUrl(string(id))
	input.SetMaxNumberOfMessages(options.MaxMessages)
	input.SetWaitTimeSeconds(int64(timeout.Seconds()))
	if options.VisibilityTimeout != 0 {
		input.SetVisibilityTimeout(int64(options.VisibilityTimeout.Seconds()))
	}

	var attributeNames []*string
	for _, s := range options.AttributeNames {
		attributeNames = append(attributeNames, aws.String(s))
	}
	input.SetMessageAttributeNames(attributeNames)

	var systemAttributeNames []*string
	for _, s := range options.SystemAttributeNames {
		systemAttributeNames = append(systemAttributeNames, aws.String(s))
	}
	input.SetAttributeNames(systemAttributeNames)

	output, err := q.sqs.ReceiveMessageWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0)

	for _, sqsMessage := range output.Messages {
		msg := Message{}
		msg.ReceivedAt = time.Now().UTC()
		msg.PayloadJSON = *sqsMessage.Body
		msg.ReceiptHandle = ReceiptHandle(*sqsMessage.ReceiptHandle)

		if len(sqsMessage.MessageAttributes) > 0 {
			msg.Attributes = make(map[string]string)

			for k, v := range sqsMessage.MessageAttributes {
				if v.StringValue != nil {
					msg.Attributes[k] = *v.StringValue
				}
			}
		}

		if len(sqsMessage.Attributes) > 0 {
			msg.SystemAttributes = make(map[string]string)

			for k, v := range sqsMessage.Attributes {
				msg.SystemAttributes[k] = *v
			}
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// DeleteMessage deletes a message from a SQS queue of given ID, using ReceiptHandle.
func (q *sqsQueueService) DeleteMessage(ctx context.Context, id ID, receiptHandle ReceiptHandle) error {
	input := &sqs.DeleteMessageInput{}
	input.SetQueueUrl(string(id))
	input.SetReceiptHandle(string(receiptHandle))

	_, err := q.sqs.DeleteMessageWithContext(ctx, input)

	return err
}

// SetVisibilityTimeout changes visibility timeout of a message in a SQS queue of given ID, using ReceiptHandle.
func (q *sqsQueueService) SetVisibilityTimeout(ctx context.Context, id ID, receiptHandle ReceiptHandle, timeout time.Duration) error {
	input := &sqs.ChangeMessageVisibilityInput{}
	input.SetQueueUrl(string(id))
	input.SetReceiptHandle(string(receiptHandle))
	input.SetVisibilityTimeout(int64(timeout.Seconds()))

	_, err := q.sqs.ChangeMessageVisibilityWithContext(ctx, input)

	return err
}

// MessageCounts returns the queue.MessageCounts of a SQS queue of given ID.
func (q *sqsQueueService) MessageCounts(ctx context.Context, id ID) (*MessageCounts, error) {
	input := &sqs.GetQueueAttributesInput{}
	input.SetQueueUrl(string(id))
	input.SetAttributeNames([]*string{
		aws.String(sqs.QueueAttributeNameApproximateNumberOfMessages),
		aws.String(sqs.QueueAttributeNameApproximateNumberOfMessagesNotVisible),
	})

	attrs, err := q.sqs.GetQueueAttributesWithContext(ctx, input)
	if err != nil {
		return nil, err
	}

	counts := &MessageCounts{}
	counts.Pending, err = strconv.Atoi(*attrs.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessages])
	if err != nil {
		return nil, err
	}

	counts.InFlight, err = strconv.Atoi(*attrs.Attributes[sqs.QueueAttributeNameApproximateNumberOfMessagesNotVisible])
	if err != nil {
		return nil, err
	}

	return counts, nil
}

func getDelaySeconds(options PublishOptions) (int64, error) {
	return int64((*options.DelayInSeconds).Seconds()), nil
}
