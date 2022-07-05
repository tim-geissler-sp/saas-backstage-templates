// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package event

import (
	"context"
	"fmt"
	"strings"

	"github.com/sailpoint/atlas-go/atlas"
)

// TopicScope is an enumeration for how a topic is distrubted across tenants.
type TopicScope int

const (
	// The default scope where all tenants in a pod share a topic.
	TopicScopePod TopicScope = iota

	// A scope where each tenant gets it's own topic.
	TopicScopeOrg

	// A scope where all tenants share a single topic.
	TopicScopeGlobal
)

// TopicName is a type alias for the name of a topic. (eg. "identity")
type TopicName string

// TopicID is the unique name of a physical topic in Kafka (eg. "echo__identity")
type TopicID string

// TopicDescriptor is an interface for topic definitions.
type TopicDescriptor interface {
	Scope() TopicScope
	Name() TopicName
}

// SimpleTopicDescriptor is a type that implements the TopicDescriptor interface
type SimpleTopicDescriptor struct {
	scope TopicScope
	name  TopicName
}

// A topic is an instanceof a TopicDescriptor that has a physical ID.
type Topic interface {
	TopicDescriptor
	ID() TopicID
}

// globalTopic is a topic type where all tenant's share the topic.
type globalTopic struct {
	name TopicName
}

// podTopic is a topic type where tenant's share a topic with others on the same pod.
type podTopic struct {
	pod  atlas.Pod
	name TopicName
}

// orgTopic is a topic type where tenant's each get their own topic.
type orgTopic struct {
	pod  atlas.Pod
	org  atlas.Org
	name TopicName
}

// Matches gets whether or not two TopicDescriptors match (same name and scope)
func Matches(a TopicDescriptor, b TopicDescriptor) bool {
	if a.Scope() != b.Scope() {
		return false
	}

	if !strings.EqualFold(string(a.Name()), string(b.Name())) {
		return false
	}

	return true
}

// NewSimpleTopicDescriptor constructs a new TopicDescriptor with the specified scope and name.
func NewSimpleTopicDescriptor(scope TopicScope, name TopicName) *SimpleTopicDescriptor {
	d := &SimpleTopicDescriptor{}
	d.scope = scope
	d.name = name

	return d
}

// Scope gets the descriptor's scope.
func (d *SimpleTopicDescriptor) Scope() TopicScope {
	return d.scope
}

// Name gets the descriotor's name.
func (d *SimpleTopicDescriptor) Name() TopicName {
	return d.name
}

// ParseTopic parses a topic ID and constructs a resulting topic.
func ParseTopic(id string) (Topic, error) {
	components := strings.Split(id, "__")

	switch len(components) {
	case 1:
		return NewGlobalTopic(TopicName(components[0])), nil
	case 2:
		return NewPodTopic(TopicName(components[0]), atlas.Pod(components[1])), nil
	case 3:
		return NewOrgTopic(TopicName(components[0]), atlas.Pod(components[1]), atlas.Org(components[2])), nil
	default:
		return nil, fmt.Errorf("invalid topic id: %s", id)
	}
}

// buildTopicID builds a new TopicID out of the specified components.
func buildTopicID(components ...string) TopicID {
	return TopicID(strings.Join(components, "__"))
}

// NewOrgTopic constructs a new per-org topic.
func NewOrgTopic(name TopicName, pod atlas.Pod, org atlas.Org) Topic {
	t := &orgTopic{}
	t.name = name
	t.pod = pod
	t.org = org

	return t
}

// NewGlobalTopic constructs a new global topic.
func NewGlobalTopic(name TopicName) Topic {
	t := &globalTopic{}
	t.name = name

	return t
}

// NewPodTopic constructs a new per-pod topic.
func NewPodTopic(name TopicName, pod atlas.Pod) Topic {
	t := &podTopic{}
	t.name = name
	t.pod = pod

	return t
}

// NewTopic constructs a new topic base from IdnTopic struct
func NewTopic(ctx context.Context, topic TopicDescriptor) (Topic, error) {
	if rc := atlas.GetRequestContext(ctx); rc != nil {
		switch topic.Scope() {
		case TopicScopePod:
			return NewPodTopic(topic.Name(), rc.Pod), nil
		case TopicScopeOrg:
			return NewOrgTopic(topic.Name(), rc.Pod, rc.Org), nil
		case TopicScopeGlobal:
			return NewGlobalTopic(topic.Name()), nil
		}
	}

	return nil, fmt.Errorf("no request context")
}

// ID gets the topic's unique id.
func (t *orgTopic) ID() TopicID {
	return buildTopicID(string(t.name), string(t.pod), string(t.org))
}

// Name gets the topic's name.
func (t *orgTopic) Name() TopicName {
	return t.name
}

// Scope gets the topic's scope.
func (t *orgTopic) Scope() TopicScope {
	return TopicScopeOrg
}

// ID gets the topic's unique id.
func (t *globalTopic) ID() TopicID {
	return TopicID(t.name)
}

// Name gets the topic's name.
func (t *globalTopic) Name() TopicName {
	return t.name
}

// Scope gets the topic's scope.
func (t *globalTopic) Scope() TopicScope {
	return TopicScopeGlobal
}

// ID gets the topic's unique id.
func (t *podTopic) ID() TopicID {
	return buildTopicID(string(t.name), string(t.pod))
}

// Name gets the topic's name.
func (t *podTopic) Name() TopicName {
	return t.name
}

// Scope gets the topic's scope.
func (t *podTopic) Scope() TopicScope {
	return TopicScopePod
}
