// Copyright (c) 2020. SailPoint Technologies, Inc. All rights reserved.
package event

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/sailpoint/atlas-go/atlas/config"
	"github.com/sailpoint/atlas-go/atlas/log"
)

// uploadedExternalEvent contains the result data from uploading a large event to an external store.
type uploadedExternalEvent struct {
	// Location contains the location of the uploaded event data.
	Location string
	// Size contains the content length of the uploaded event data.
	Size int
}

// externalUploader is an interface for uploading large (> message.max.bytes) Event to non-Kafka destination.
type externalUploader interface {
	ShouldUpload(ctx context.Context, event *Event) bool
	Upload(ctx context.Context, topic Topic, event *Event) (*uploadedExternalEvent, error)
}

// uploaderConfig is any config needed for S3ExternalUploader.
type uploaderConfig struct {
	bucket          string
	uploadThreshold int
}

// s3ExternalUploader is an AWS S3 implementation of ExternalUploader that uploads large Event to a S3 bucket.
type s3ExternalUploader struct {
	uploader *s3manager.Uploader
	config   uploaderConfig
}

// newS3ExternalUploader creates a new S3ExternalUploader.
func newS3ExternalUploader(uc uploaderConfig) *s3ExternalUploader {
	return &s3ExternalUploader{
		uploader: s3manager.NewUploader(config.GlobalAwsSession()),
		config:   uc,
	}
}

// ShouldUpload returns a bool indicating whether an Event is large enough that it needs to be uploaded to an S3 bucket.
func (s *s3ExternalUploader) ShouldUpload(ctx context.Context, event *Event) bool {
	if s.config.bucket == "" || event == nil {
		return false
	}

	eventJson, _ := json.Marshal(event)
	return len(eventJson) > s.config.uploadThreshold
}

// Upload returns object key of uploaded large Event, the content length or an error.
func (s *s3ExternalUploader) Upload(ctx context.Context, topic Topic, event *Event) (*uploadedExternalEvent, error) {
	eventJson, err := json.Marshal(event)
	if err != nil {
		return &uploadedExternalEvent{}, err
	}

	s3ObjectKey := getKey(topic, event)

	upParams := &s3manager.UploadInput{
		Bucket:      aws.String(s.config.bucket),
		Key:         aws.String(s3ObjectKey),
		Body:        bytes.NewReader(eventJson),
		ContentType: aws.String("application/json"),
		Metadata:    aws.StringMap(map[string]string{"eventId": event.ID}),
	}

	_, err = s.uploader.Upload(upParams)
	if err != nil {
		log.Errorf(ctx, "error upload event %s on s3 bucket %s: %v", event.Type, s.config.bucket, err)
		return &uploadedExternalEvent{}, err
	}

	return &uploadedExternalEvent{Location: s3ObjectKey, Size: len(eventJson)}, nil
}

// getKey defines an Event's S3 object key.
func getKey(topic Topic, event *Event) string {
	return string(topic.Name()) + "/event-" + strings.ToLower(event.Type) + "-" + strings.ToLower(event.ID) + ".json"
}

// externalDownloader is an interface for downloading large Event from external (non-kafka) source.
type externalDownloader interface {
	Download(ctx context.Context, location string) (*Event, error)
}

// downloaderConfig is any config needed for S3ExternalDownloader
type downloaderConfig struct {
	bucket string
}

// s3ExternalDownloader is an AWS S3 implementation of ExternalDownloader that downloads large Event from a S3 bucket.
type s3ExternalDownloader struct {
	downloader *s3manager.Downloader
	config     downloaderConfig
}

// newS3ExternalDownloader creates a new S3ExternalDownloader.
func newS3ExternalDownloader(dc downloaderConfig) *s3ExternalDownloader {
	return &s3ExternalDownloader{
		downloader: s3manager.NewDownloader(config.GlobalAwsSession()),
		config:     dc,
	}
}

// Download returns Event downloaded from a S3 bucket or error.
func (s *s3ExternalDownloader) Download(ctx context.Context, location string) (*Event, error) {
	downParams := &s3.GetObjectInput{
		Bucket: aws.String(s.config.bucket),
		Key:    aws.String(location),
	}

	writeAt := new(aws.WriteAtBuffer)
	_, err := s.downloader.Download(writeAt, downParams)
	if err != nil {
		return nil, err
	}

	event := new(Event)
	if err := json.Unmarshal(writeAt.Bytes(), event); err != nil {
		return nil, err
	}

	return event, nil
}
