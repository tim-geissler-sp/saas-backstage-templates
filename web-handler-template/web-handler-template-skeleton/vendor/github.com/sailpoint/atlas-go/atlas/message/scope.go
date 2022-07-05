// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package message

import (
	"context"
	"fmt"

	"github.com/sailpoint/atlas-go/atlas"
)

// QueueType is an enumeration for how a message queue is distributed across tenants.
type QueueType int

const (
	// QueueTypeOrg is the default type where each tenant gets its' own queue.
	QueueTypeOrg QueueType = iota

	// QueueTypePod is a type where all tenants on a pod share a queue.
	QueueTypePod
)

// ScopeID is the unique name of a physical queue in Redis (eg. "echo/jeff-test/qpoc")
type ScopeID string

// ScopeName is a type alias for the name of a message scope. (eg. "qpoc")
type ScopeName string

// ScopeDescriptor is an interface for scope definitions.
type ScopeDescriptor interface {
	QueueType() QueueType
	Name() ScopeName
}

// A Scope is an instanceof a ScopeDescriptor that has a physical ID.
type Scope interface {
	ScopeDescriptor
	ID() ScopeID
}

// orgScope is a scope type where tenants each get their own queue.
type orgScope struct {
	pod  atlas.Pod
	org  atlas.Org
	name ScopeName
}

// podScope is a scope type where tenants in the same pod share a queue.
type podScope struct {
	pod  atlas.Pod
	name ScopeName
}

// NewOrgScope constructs a new org scope
func NewOrgScope(name ScopeName, pod atlas.Pod, org atlas.Org) Scope {
	s := &orgScope{}
	s.name = name
	s.pod = pod
	s.org = org

	return s
}

func (s *orgScope) ID() ScopeID {
	return ScopeID(string(s.pod) + "/" + string(s.org) + "/" + string(s.name))
}

func (s *orgScope) Name() ScopeName {
	return s.name
}

func (s *orgScope) QueueType() QueueType {
	return QueueTypeOrg
}

// NewPodScope constructs a new pod scope
func NewPodScope(name ScopeName, pod atlas.Pod) Scope {
	s := &podScope{}
	s.name = name
	s.pod = pod

	return s
}

func (s *podScope) ID() ScopeID {
	return ScopeID(string(s.pod) + "/" + string(s.name))
}

func (s *podScope) Name() ScopeName {
	return s.name
}

func (s *podScope) QueueType() QueueType {
	return QueueTypePod
}

// NewScopeFromContext converts a ScopeDescriptor into a physical scope, based on data
// parsed from the current context.
func NewScopeFromContext(ctx context.Context, sd ScopeDescriptor) (Scope, error) {
	rc := atlas.GetRequestContext(ctx)
	if rc == nil {
		return nil, fmt.Errorf("no request context")
	}

	switch sd.QueueType() {
	case QueueTypePod:
		return NewPodScope(sd.Name(), rc.Pod), nil
	case QueueTypeOrg:
		return NewOrgScope(sd.Name(), rc.Pod, rc.Org), nil
	default:
		return nil, fmt.Errorf("invalid queue type on scope descriptor: %v", sd.QueueType())
	}
}
