// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package atlas

import (
	"context"
)

// TenantID is a unique UUID for a tenant. (eg. "68df224b-535c-4b03-8d33-05b08fa2eebe")
type TenantID string

// Org is the human-readable name of a tenant - URL safe. (eg. "acme-corp")
type Org string

// Pod is the name of the shard that the tenant belongs to. (eg. "prd01-useast1")
type Pod string

// IdentityID is the unique UUID for an identity. (eg. "f923fe74-2c0b-4f72-bb1f-e23801edfae5")
type IdentityID string

// IdentityName is the unique name of an identity, the format differs from tenant to tenant. (eg. "john.doe")
type IdentityName string

type contextKey int

const (
	requestContextKey contextKey = iota
)

// RequestContext contains standard attributes that are generally associated
// with an HTTP/Event/Message handling request.
type RequestContext struct {
	TenantID     TenantID
	Pod          Pod
	Org          Org
	IdentityID   IdentityID
	IdentityName IdentityName
}

// GetRequestContext extracts a RequestContext from the specified context.
// Nil is returned if the context isn't associated with a RequestContext.
func GetRequestContext(ctx context.Context) *RequestContext {
	v := ctx.Value(requestContextKey)

	if v == nil {
		return nil
	}

	return v.(*RequestContext)

}

// WithRequestContext derives a new context from the specified context that
// contains a reference to a RequestContext.
func WithRequestContext(ctx context.Context, rc *RequestContext) context.Context {
	return context.WithValue(ctx, requestContextKey, rc)
}
