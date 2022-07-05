// Copyright (c) 2020. Sailpoint Technologies, Inc. All rights reserved.
package event

import "context"

// HandlerFunc is a function alias so that functions can implement the Handler interface.
type HandlerFunc func(context.Context, Topic, *Event) error

// FilterFunc is a function alias for a filter.
type FilterFunc func(Topic, *Event) bool

// Handler is an interface for event handlers
type Handler interface {
	HandleEvent(ctx context.Context, topic Topic, event *Event) error
}

// Filter is an interface for filtering events before they arrive at a handler.
type Filter interface {
	Filter(topic Topic, e *Event) bool
}

// filteredHandler is a type that applies a filter prior to executing a downstream handler.
type filteredHandler struct {
	filter  Filter
	handler Handler
}

// compositeHandler is a type that forwards an event to a sequence of handlers.
type compositeHandler struct {
	handlers []Handler
}

// Filter delegates to the bound function, making FilterFunc implement the Filter interface.
func (f FilterFunc) Filter(topic Topic, e *Event) bool {
	return f(topic, e)
}

// HandleEvent delegates to a HandlerFunc so that HandlerFunc implements the Handler interface
func (f HandlerFunc) HandleEvent(ctx context.Context, topic Topic, event *Event) error {
	return f(ctx, topic, event)
}

// Add appends a handler to the end of the compositeHandler's chain.
func (ch *compositeHandler) Add(handler Handler) {
	ch.handlers = append(ch.handlers, handler)
}

// HandleEvent sends the event through the series of handlers in the chain.
func (ch *compositeHandler) HandleEvent(ctx context.Context, topic Topic, e *Event) error {
	for _, h := range ch.handlers {
		if err := h.HandleEvent(ctx, topic, e); err != nil {
			return err
		}
	}

	return nil
}

// HandleEvent sends an event to the delegate handler if the filter passes, otherwise returns nil.
func (h *filteredHandler) HandleEvent(ctx context.Context, topic Topic, e *Event) error {
	if h.filter.Filter(topic, e) {
		return h.handler.HandleEvent(ctx, topic, e)
	}

	return nil
}
