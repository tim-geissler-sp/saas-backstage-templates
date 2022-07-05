// Copyright (c) 2021, SailPoint Technologies, Inc. All rights reserved.
package dynamoutil

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// SimpleKeyAttribute constructs a key for dynamodb queries with a single key/value pair.
func SimpleKeyAttribute(name string, value string) map[string]*dynamodb.AttributeValue {
	return map[string]*dynamodb.AttributeValue{
		name: StringAttribute(value),
	}
}

// Encoder is an interface for encoding of bytes.
type Encoder interface {
	Encode(value []byte) ([]byte, error)
}

// Decoder is an interface for decoding bytes.
type Decoder interface {
	Decode(encoded []byte) ([]byte, error)
}

// EncodedJSONAttribute marshals data into a string attribute using JSON and stores the value encoded.
func EncodedJSONAttribute(encoder Encoder, v interface{}) (*dynamodb.AttributeValue, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	encoded, err := encoder.Encode(b)
	if err != nil {
		return nil, err
	}

	return StringAttribute(base64.StdEncoding.EncodeToString(encoded)), nil
}

// JSONAttribute marshals data into a string attribute using JSON
func JSONAttribute(v interface{}) (*dynamodb.AttributeValue, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	return StringAttribute(string(b)), nil
}

// StringAttribute constructs a dynamo string attribute from an input string
func StringAttribute(s string) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{S: aws.String(s)}
}

// TimeAttribute constructs a dynamo string attribute from a timestamp, serialized as RFC3339
func TimeAttribute(t time.Time) *dynamodb.AttributeValue {
	return StringAttribute(t.Format(time.RFC3339Nano))
}

// NumberAttribute constructs a dynamo number attribute from an int64.
func NumberAttribute(n int64) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		N: aws.String(strconv.FormatInt(n, 10)),
	}
}

// BoolAttribute constructs a dynamo bool attribute from a bool.
func BoolAttribute(b bool) *dynamodb.AttributeValue {
	return &dynamodb.AttributeValue{
		BOOL: aws.Bool(b),
	}
}

// EpochTimeAttribute constructs a dynamo number attribute from a timestamp, using the epoc time.
// This format is suitable for use as a dynamo TTL attribute.
func EpochTimeAttribute(t time.Time) *dynamodb.AttributeValue {
	return NumberAttribute(t.Unix())
}

// GetEpochTime extracts the epoch time from a dynamo number attribute.
func GetEpochTime(value *dynamodb.AttributeValue) (time.Time, error) {
	seconds, err := GetNumber(value)
	if err != nil || seconds == 0 {
		return time.Time{}, err
	}

	return time.Unix(seconds, 0), nil
}

// GetNumber extracts a numeric value from a dynamo number attribute.
func GetNumber(value *dynamodb.AttributeValue) (int64, error) {
	if value == nil || value.N == nil {
		return 0, nil
	}

	return strconv.ParseInt(*value.N, 10, 64)
}

// GetBool extracts a boolean value from a dynamo bool attribute.
func GetBool(value *dynamodb.AttributeValue) bool {
	if value == nil || value.BOOL == nil {
		return false
	}

	return *value.BOOL
}

// GetJSON extracts a JSON value from a dynamo string attribute.
func GetJSON(value *dynamodb.AttributeValue, v interface{}) error {
	s := GetString(value)
	return json.Unmarshal([]byte(s), v)
}

// GetEncodedJSON extracts an encoded JSON value from a dynamo string attribute.
func GetEncodedJSON(value *dynamodb.AttributeValue, decoder Decoder, v interface{}) error {
	s := GetString(value)

	encoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	b, err := decoder.Decode(encoded)
	if err != nil {
		return err
	}

	return json.Unmarshal(b, v)
}

// GetTime extracts a timestamp from a dynamo string attribute (serialized as RFC3339)
func GetTime(value *dynamodb.AttributeValue) (time.Time, error) {
	if value == nil || value.S == nil {
		return time.Time{}, nil
	}

	return time.Parse(time.RFC3339Nano, *value.S)
}

// GetString extracts a string from a dynamo string attribute.
func GetString(value *dynamodb.AttributeValue) string {
	if value == nil || value.S == nil {
		return ""
	}

	return *value.S
}
