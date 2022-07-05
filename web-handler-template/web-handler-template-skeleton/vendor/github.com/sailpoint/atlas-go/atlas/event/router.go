// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package event

import (
	"context"
	"strings"
)

// Router is a type that supports HTTP-like routing for event handlers. It supports middleware and filtered handlers
// so that they can be registered by topic/event type and arbitrary filters.
type Router struct {
	middleware []Middleware
	handlers   []*filteredHandler
	topics     []TopicDescriptor
}

// NewRouter constructs a new, empty router.
func NewRouter() *Router {
	return &Router{}
}

// NewRouterWithDefaultMiddleware construct a new router with the default atlas middleware added.
// The default atlas middleware includes:
// - Middleware that sets up the request context and logger
// - Middleware that captures event handling metrics
func NewRouterWithDefaultMiddleware() *Router {
	r := NewRouter()
	r.Use(SetupTracingContext())
	r.Use(SetupRequestContext())
	r.Use(EventMetrics())

	return r
}

// Topics gets the set of topics that have bindings in the router
func (r *Router) Topics() []TopicDescriptor {
	return r.topics
}

// addTopic adds a topic to the list of topic descriptors in the router.
func (r *Router) addTopic(topic TopicDescriptor) {
	for _, t := range r.topics {
		if Matches(topic, t) {
			return
		}
	}

	r.topics = append(r.topics, topic)
}

// Use appends the specified middleware functions to the middleware chain.
func (r *Router) Use(mw ...MiddlewareFunc) {
	for _, f := range mw {
		r.middleware = append(r.middleware, f)
	}
}

// On adds a handler to the handler chain that executes only of the specified filter passes.
func (r *Router) On(filter Filter, handler Handler) {
	r.handlers = append(r.handlers, &filteredHandler{filter, handler})
}

// OnTopic registers a handler to run when an event is received on the specified topic.
func (r *Router) OnTopic(topicDescriptor TopicDescriptor, handler Handler) {
	filter := FilterFunc(func(topic Topic, e *Event) bool {
		return Matches(topic, topicDescriptor)
	})

	r.addTopic(topicDescriptor)
	r.On(filter, handler)
}

// OnTopicAndEventType registers a handler to run when an event of the specified type is received on the specified topic.
func (r *Router) OnTopicAndEventType(topicDescriptor TopicDescriptor, eventType string, handler Handler) {
	filter := FilterFunc(func(topic Topic, e *Event) bool {
		if !strings.EqualFold(e.Type, eventType) && eventType != "*" {
			return false
		}

		return Matches(topic, topicDescriptor)
	})

	r.addTopic(topicDescriptor)
	r.On(filter, handler)
}

// anyFiltersMatch gets whether or not any filters registered in the router match
// the given topic and event.
func (r *Router) anyFiltersMatch(topic Topic, event *Event) bool {
	for _, h := range r.handlers {
		if h.filter.Filter(topic, event) {
			return true
		}
	}

	return false
}

// runAllHandlers returns a handler that runs all of the handlers registered in the router.
func (r *Router) runAllHandlers() Handler {
	return HandlerFunc(func(ctx context.Context, topic Topic, event *Event) error {
		for _, h := range r.handlers {
			if err := h.HandleEvent(ctx, topic, event); err != nil {
				return err
			}
		}

		return nil
	})
}

// HandleEvent makes Router implement the Handler interface, invoking the middleware chain and registered handlers
// appropriately.
func (r *Router) HandleEvent(ctx context.Context, topic Topic, event *Event) error {
	var handler Handler = r.runAllHandlers()

	// Don't run any middleware if no filters match...
	if !r.anyFiltersMatch(topic, event) {
		return nil
	}

	// Iterate in reverse order so middleware is applied in the correct order...
	for i := len(r.middleware) - 1; i >= 0; i-- {
		handler = r.middleware[i].Middleware(handler)
	}

	return handler.HandleEvent(ctx, topic, event)
}
