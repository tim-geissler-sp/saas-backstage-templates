// Copyright (c) 2020. SailPoint Technologies, Inc. All rights reserved.
package event

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/sailpoint/atlas-go/atlas"
)

// Event is a struct that represents iris event data.
type Event struct {
	Headers     Headers    `json:"headers"`
	ID          string     `json:"id"`
	Timestamp   atlas.Time `json:"timestamp"`
	Type        string     `json:"type"`
	ContentJSON string     `json:"contentJson"`
}

// Headers is a type definition for a string map. Headers are able
// to be associated with each event and are stored separately from
// the content.
type Headers map[string]string

const (
	HeaderKeyRequestID      = "requestId"
	HeaderKeyTenantID       = "tenantId"
	HeaderKeyPod            = "pod"
	HeaderKeyOrg            = "org"
	HeaderKeyPartitionKey   = "partitionKey"
	HeaderKeyGroupID        = "groupId"
	HeaderKeyIsCompactEvent = "isCompactedEvent"
)

// EventAndTopic is a convenience struct for publication that ties together and Event and Topic.
type EventAndTopic struct {
	Event *Event
	Topic Topic
}

type FailedEventAndTopic struct {
	EventAndTopic *EventAndTopic
	Err           error
}

func NewFailedFailedEventAndTopic(evt EventAndTopic, err error) *FailedEventAndTopic {
	fEvt := FailedEventAndTopic{}
	fEvt.EventAndTopic = &evt
	fEvt.Err = err
	return &fEvt
}

// NewEventJSON constructs a new event, where the event content has already been serialized to
// valid JSON.
func NewEventJSON(eventType string, contentJSON string, headers Headers) *Event {
	e := &Event{}
	e.ID = uuid.New().String()
	e.Timestamp = atlas.Now()
	e.Type = eventType
	e.ContentJSON = contentJSON

	e.Headers = make(Headers)
	for k, v := range headers {
		e.Headers[k] = v
	}

	return e
}

// NewEvent constructs a new event from a generic content type. The content is serialized
// to JSON and embedded in the event.
func NewEvent(eventType string, content interface{}, headers Headers) (*Event, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	return NewEventJSON(eventType, string(contentJSON), headers), nil
}

// GetContent parses the event content into the specified interface. An error is returned
// if parsing fails.
func (e *Event) GetContent(v interface{}) error {
	return json.Unmarshal([]byte(e.ContentJSON), v)
}
