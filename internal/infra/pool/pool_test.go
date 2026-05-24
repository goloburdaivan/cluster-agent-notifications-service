package pool

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"notifications-service/internal/core/dispatcher"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/infra/queue"
)

type mockDispatcher struct {
	mock.Mock
}

func (m *mockDispatcher) Register(eventType domain.EventType, handler dispatcher.EventHandler) {
	m.Called(eventType, handler)
}

func (m *mockDispatcher) Dispatch(ctx context.Context, event domain.NotificationEnvelope) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type mockMessage struct {
	body    []byte
	acked   bool
	nacked  bool
	ackErr  error
	nackErr error
}

func (m *mockMessage) Body() []byte                      { return m.body }
func (m *mockMessage) Ack() error                        { m.acked = true; return m.ackErr }
func (m *mockMessage) Nack(multiple, requeue bool) error { m.nacked = true; return m.nackErr }

type mockQueue struct {
	ch  chan queue.Message
	err error
}

func (m *mockQueue) Consume(_ context.Context) (<-chan queue.Message, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.ch, nil
}

func (m *mockQueue) Close() error { return nil }

func TestHandleMessage_Success(t *testing.T) {
	disp := new(mockDispatcher)

	envelope := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	body, _ := json.Marshal(envelope)
	msg := &mockMessage{body: body}

	disp.On("Dispatch", mock.Anything, mock.MatchedBy(func(e domain.NotificationEnvelope) bool {
		return e.EventType == domain.EventTypeSecurity
	})).Return(nil)

	p := &WorkerPool{dispatcher: disp}
	p.handleMessage(context.Background(), msg)

	assert.True(t, msg.acked)
	assert.False(t, msg.nacked)
	disp.AssertExpectations(t)
}

func TestHandleMessage_UnmarshalError(t *testing.T) {
	p := &WorkerPool{dispatcher: new(mockDispatcher)}
	msg := &mockMessage{body: []byte("invalid json")}

	p.handleMessage(context.Background(), msg)

	assert.True(t, msg.nacked)
	assert.False(t, msg.acked)
}

func TestHandleMessage_DispatchError(t *testing.T) {
	disp := new(mockDispatcher)

	envelope := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	body, _ := json.Marshal(envelope)
	msg := &mockMessage{body: body}

	disp.On("Dispatch", mock.Anything, mock.Anything).Return(errors.New("dispatch error"))

	p := &WorkerPool{dispatcher: disp}
	p.handleMessage(context.Background(), msg)

	assert.True(t, msg.nacked)
	assert.False(t, msg.acked)
}

func TestHandleMessage_AckError(t *testing.T) {
	disp := new(mockDispatcher)

	envelope := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	body, _ := json.Marshal(envelope)
	msg := &mockMessage{body: body, ackErr: errors.New("ack failed")}

	disp.On("Dispatch", mock.Anything, mock.Anything).Return(nil)

	p := &WorkerPool{dispatcher: disp}
	p.handleMessage(context.Background(), msg)

	assert.True(t, msg.acked)
}

func TestNackMessage_Success(t *testing.T) {
	msg := &mockMessage{}
	p := &WorkerPool{}
	p.nackMessage(msg)
	assert.True(t, msg.nacked)
}

func TestNackMessage_Error(t *testing.T) {
	msg := &mockMessage{nackErr: errors.New("nack failed")}
	p := &WorkerPool{}
	p.nackMessage(msg)
	assert.True(t, msg.nacked)
}

func TestWorkerPool_Start_ProcessesMessage(t *testing.T) {
	disp := new(mockDispatcher)
	msgCh := make(chan queue.Message, 1)
	mq := &mockQueue{ch: msgCh}

	envelope := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity}
	body, _ := json.Marshal(envelope)
	msg := &mockMessage{body: body}

	disp.On("Dispatch", mock.Anything, mock.Anything).Return(nil)

	wp := NewWorkerPool(mq, disp, 1)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		wp.Start(ctx)
		close(done)
	}()

	msgCh <- msg
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}

	assert.True(t, msg.acked)
}

func TestWorkerPool_Start_ConsumeError(t *testing.T) {
	mq := &mockQueue{err: errors.New("consume error")}
	wp := NewWorkerPool(mq, nil, 1)

	done := make(chan struct{})
	go func() {
		wp.Start(context.Background())
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after consume error")
	}
}

func TestWorkerPool_Start_ChannelClosed(t *testing.T) {
	disp := new(mockDispatcher)
	msgCh := make(chan queue.Message)
	mq := &mockQueue{ch: msgCh}

	wp := NewWorkerPool(mq, disp, 1)

	done := make(chan struct{})
	go func() {
		wp.Start(context.Background())
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	close(msgCh)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after channel close")
	}
}
