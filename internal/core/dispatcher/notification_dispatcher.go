package dispatcher

import (
	"context"
	"fmt"
	"log/slog"
	"notifications-service/internal/core/domain"
)

type EventHandler func(ctx context.Context, event domain.NotificationEnvelope) error

type EventDispatcher interface {
	Register(eventType domain.EventType, handler EventHandler)
	Dispatch(ctx context.Context, event domain.NotificationEnvelope) error
}

type eventDispatcher struct {
	listeners map[domain.EventType][]EventHandler
}

func (e *eventDispatcher) Register(eventType domain.EventType, handler EventHandler) {
	e.listeners[eventType] = append(e.listeners[eventType], handler)
}

func (e *eventDispatcher) Dispatch(ctx context.Context, event domain.NotificationEnvelope) error {
	handlers, exists := e.listeners[event.EventType]
	if !exists {
		slog.Warn("No handlers registered for event type", "event_type", event.EventType)
		return nil
	}

	for _, handler := range handlers {
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handler failed for event %s: %w", event.EventType, err)
		}
	}
	return nil
}

func NewEventDispatcher() EventDispatcher {
	return &eventDispatcher{
		listeners: make(map[domain.EventType][]EventHandler),
	}
}
