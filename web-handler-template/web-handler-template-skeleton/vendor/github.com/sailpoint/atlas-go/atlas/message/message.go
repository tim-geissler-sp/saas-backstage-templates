// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package message

import "encoding/json"

// Headers is a type definition for a string map. Headers are able
// to be associated with each message and are stored separately from
// the content.
type Headers map[string]string

const (
	HeaderKeyRequestID     = "requestId"
	HeaderKeyTenantID      = "tenantId"
	HeaderKeyPod           = "pod"
	HeaderKeyOrg           = "org"
	HeaderKeyAttemptNumber = "attemptNumber"
	HeaderKeyPayloadType   = "payloadType"
	HeaderKeyMessageType   = "messageType"
)

// Message is a struct that represents a serialized atlas message.
type Message struct {
	Headers     Headers `json:"headers"`
	ContentJSON string  `json:"contentJson"`
}

// NewMessageJSON constructs a new message, where the message content has already been serialized to
// valid JSON.
func NewMessageJSON(contentJSON string, headers Headers) *Message {
	m := &Message{}
	m.ContentJSON = contentJSON

	m.Headers = make(Headers)
	for k, v := range headers {
		m.Headers[k] = v
	}

	return m
}

// NewMessage constructs a new message from a generic content type. The content is serialized
// to JSON and embedded in the message.
func NewMessage(content interface{}, headers Headers) (*Message, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	return NewMessageJSON(string(contentJSON), headers), nil
}

// GetContent parses the message content into the specified interface. An error is returned
// if parsing fails.
func (m *Message) GetContent(v interface{}) error {
	return json.Unmarshal([]byte(m.ContentJSON), v)
}
