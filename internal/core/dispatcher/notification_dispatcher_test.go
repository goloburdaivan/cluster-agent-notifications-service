package dispatcher

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

func TestDispatch_NoHandlers(t *testing.T) {
	d := NewEventDispatcher()
	event := domain.NotificationEnvelope{EventType: "unknown.event"}
	err := d.Dispatch(context.Background(), event)
	assert.NoError(t, err)
}

func TestDispatch_SingleHandler(t *testing.T) {
	d := NewEventDispatcher()
	called := false
	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		called = true
		return nil
	})

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDispatch_MultipleHandlers(t *testing.T) {
	d := NewEventDispatcher()
	var calls []int

	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		calls = append(calls, 1)
		return nil
	})
	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		calls = append(calls, 2)
		return nil
	})

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2}, calls)
}

func TestDispatch_HandlerError(t *testing.T) {
	d := NewEventDispatcher()
	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		return errors.New("handler failed")
	})

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	err := d.Dispatch(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "handler failed")
}

func TestDispatch_StopsOnFirstError(t *testing.T) {
	d := NewEventDispatcher()
	secondCalled := false

	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		return errors.New("first handler failed")
	})
	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		secondCalled = true
		return nil
	})

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	err := d.Dispatch(context.Background(), event)
	assert.Error(t, err)
	assert.False(t, secondCalled)
}

func TestRegister_DifferentEventTypes(t *testing.T) {
	d := NewEventDispatcher()
	securityCalled := false
	observabilityCalled := false

	d.Register(domain.EventTypeSecurity, func(_ context.Context, _ domain.NotificationEnvelope) error {
		securityCalled = true
		return nil
	})
	d.Register(domain.EventTypeObservability, func(_ context.Context, _ domain.NotificationEnvelope) error {
		observabilityCalled = true
		return nil
	})

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	err := d.Dispatch(context.Background(), event)
	require.NoError(t, err)
	assert.True(t, securityCalled)
	assert.False(t, observabilityCalled)
}
