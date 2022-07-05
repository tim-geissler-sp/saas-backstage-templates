// Copyright (c) 2022, SailPoint Technologies, Inc. All rights reserved.

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/sailpoint/atlas-go/atlas"
	"github.com/sailpoint/atlas-go/atlas/log"
	"github.com/sailpoint/atlas-go/atlas/trace"
)

type contextKey string

// contextKeyTarget is the context key to store the name of the target service of the REST API call,
// primarily to be used to label metrics
var contextKeyTarget = contextKey("target")

// InternalRestClient is an interface for HTTP client that performs internal, service-to-service REST API calls.
type InternalRestClient interface {
	Get(ctx context.Context, service, path string, respBody interface{}) error
	Post(ctx context.Context, service, path string, reqBody interface{}, respBody interface{}) error
	Put(ctx context.Context, service, path string, reqBody interface{}, respBody interface{}) error
	Delete(ctx context.Context, service, path string, respBody interface{}) error
}

// DefaultInternalRestClient is a default implementation of InternalRestClient.
// It is capable of performing internal REST API calls using its ServiceLocator
// to locate the target service and http.Client instance from InternalClientProvider.
// It assumes JSON format for all request and response body.
//
// Example:
//
//     // Create an internal REST client using atlas service locator and internal client provider
//     client := client.NewInternalRestClient(myServiceLocator, myClientProvider)
//
//     // Perform a GET request
//     var respDto ResponseDto
//     err := client.Get(ctx, "sp-scheduler", "/health/system", &respDto)
//     if err != nil {
//         if clientErr, ok := err.(client.Error); ok {
//             switch clientErr.StatusCode {
//             case 401, 403:
//                 // handle 401 or 403
//                 break
//             case 500:
//                 // handle 500 internal server error
//                 break
//             }
//         }
//     }
//
//     // Perform a POST request
//     reqDto := RequestDto{}
//     var respDto ResponseDto
//     err := client.Post(ctx, "sp-scheduler", "/scheduled-actions", &reqDto, &respDto)
//     if err != nil {
//         if clientErr, ok := err.(client.Error); ok {
//             switch {
//             case clientErr.StatusCode >= 400 && clientErr.StatusCode < 500:
//                 // handle all client errors
//                 break
//             case clientErr.StatusCode >= 500:
//                 // handle all server errors
//                 break
//             }
//         }
//     }
type DefaultInternalRestClient struct {
	serviceLocator ServiceLocator
	clientProvider InternalClientProvider
}

// NewInternalRestClient constructs a DefaultInternalRestClient.
func NewInternalRestClient(serviceLocator ServiceLocator, clientProvider InternalClientProvider) *DefaultInternalRestClient {
	return &DefaultInternalRestClient{
		serviceLocator: serviceLocator,
		clientProvider: clientProvider,
	}
}

// ErrorMessage is the standard API error response message type.
type ErrorMessage struct {
	Locale       string `json:"locale"`
	LocaleOrigin string `json:"localeOrigin"`
	Text         string `json:"text"`
}

// Error is the standard API error response type.
type Error struct {
	StatusCode int
	DetailCode string         `json:"detailCode"`
	TrackingID string         `json:"trackingId"`
	Messages   []ErrorMessage `json:"messages"`
}

// Implements the built-in error interface to return Error as string
func (e Error) Error() string {
	errMsg := fmt.Sprintf("%d %s: ", e.StatusCode, e.DetailCode)
	for _, msg := range e.Messages {
		if msg.Locale == "en-US" {
			errMsg = errMsg + msg.Text
			break
		}
	}

	if e.TrackingID != "" {
		errMsg = fmt.Sprintf("%s (%s)", errMsg, e.TrackingID)
	}

	return errMsg
}

// DetailCode is an alias for http.StatusText.
func DetailCode(code int) string {
	return http.StatusText(code)
}

// NewError constructs a new standard error with the specified default text.
func NewError(ctx context.Context, statusCode int, messageText string) Error {
	message := ErrorMessage{
		Locale:       "en-US",
		LocaleOrigin: "DEFAULT",
		Text:         messageText,
	}

	e := Error{
		StatusCode: statusCode,
		DetailCode: DetailCode(statusCode),
		Messages:   []ErrorMessage{message},
	}

	if tc := trace.GetTracingContext(ctx); tc != nil {
		e.TrackingID = string(tc.RequestID)
	}

	return e
}

// handleResponse decodes http.Response body (assumes JSON) to respBody; error is returned if body cannot be decoded.
// Error is returned when response status code >= 400.
func handleResponse(ctx context.Context, resp *http.Response, respBody interface{}) error {
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Warnf(ctx, "failed to close response body: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode >= http.StatusBadRequest {
		var clientErr Error
		err := json.NewDecoder(resp.Body).Decode(&clientErr)
		if err != nil {
			return NewError(ctx, resp.StatusCode, "request failed")
		}

		clientErr.StatusCode = resp.StatusCode

		return clientErr
	}

	return json.NewDecoder(resp.Body).Decode(respBody)
}

func WithTarget(ctx context.Context, target string) context.Context {
	return context.WithValue(ctx, contextKeyTarget, target)
}

// Get performs a GET request.
// The context is expected to contain atlas.RequestContext.
func (c *DefaultInternalRestClient) Get(ctx context.Context, service, path string, respBody interface{}) error {
	rc := atlas.GetRequestContext(ctx)
	if rc == nil {
		return fmt.Errorf("request context is nil")
	}

	ctx = WithTarget(ctx, service)
	url := c.serviceLocator.GetURL(rc.Org, service) + filepath.Join("/", path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.clientProvider.GetInternalClient(rc.TenantID, rc.Org).Do(req)
	if err != nil {
		return err
	}

	return handleResponse(ctx, resp, respBody)
}

// Post performs a POST request.
// The context is expected to contain atlas.RequestContext.
func (c *DefaultInternalRestClient) Post(ctx context.Context, service, path string, reqBody interface{}, respBody interface{}) error {
	rc := atlas.GetRequestContext(ctx)
	if rc == nil {
		return fmt.Errorf("request context is nil")
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	ctx = WithTarget(ctx, service)
	url := c.serviceLocator.GetURL(rc.Org, service) + filepath.Join("/", path)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	resp, err := c.clientProvider.GetInternalClient(rc.TenantID, rc.Org).Do(req)
	if err != nil {
		return err
	}

	return handleResponse(ctx, resp, respBody)
}

// Put performs a PUT request.
// The context is expected to contain atlas.RequestContext.
func (c *DefaultInternalRestClient) Put(ctx context.Context, service, path string, reqBody interface{}, respBody interface{}) error {
	rc := atlas.GetRequestContext(ctx)
	if rc == nil {
		return fmt.Errorf("request context is nil")
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	ctx = WithTarget(ctx, service)
	url := c.serviceLocator.GetURL(rc.Org, service) + filepath.Join("/", path)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	resp, err := c.clientProvider.GetInternalClient(rc.TenantID, rc.Org).Do(req)
	if err != nil {
		return err
	}

	return handleResponse(ctx, resp, respBody)
}

// Delete performs a DELETE request.
// The context is expected to contain atlas.RequestContext.
func (c *DefaultInternalRestClient) Delete(ctx context.Context, service, path string, respBody interface{}) error {
	rc := atlas.GetRequestContext(ctx)
	if rc == nil {
		return fmt.Errorf("request context is nil")
	}

	ctx = WithTarget(ctx, service)
	url := c.serviceLocator.GetURL(rc.Org, service) + filepath.Join("/", path)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, http.NoBody)
	if err != nil {
		return err
	}

	resp, err := c.clientProvider.GetInternalClient(rc.TenantID, rc.Org).Do(req)
	if err != nil {
		return err
	}

	return handleResponse(ctx, resp, respBody)
}
