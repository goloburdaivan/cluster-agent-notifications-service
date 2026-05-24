package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

type mockSender struct {
	mock.Mock
}

func (m *mockSender) Type() domain.ChannelType {
	args := m.Called()
	return args.Get(0).(domain.ChannelType)
}

func (m *mockSender) Send(ctx context.Context, credentials []byte, eventType domain.EventType, payload any) error {
	args := m.Called(ctx, credentials, eventType, payload)
	return args.Error(0)
}

func TestNotificationRouter_RouteAndSend_Success(t *testing.T) {
	sender := new(mockSender)
	sender.On("Type").Return(domain.ChannelTypeSlack)
	sender.On("Send", mock.Anything, []byte(`{"token":"abc"}`), domain.EventTypeSecurity, "payload").Return(nil)

	router := NewNotificationRouter(sender)

	ch := domain.Channel{Type: domain.ChannelTypeSlack, Credentials: []byte(`{"token":"abc"}`)}
	err := router.RouteAndSend(context.Background(), ch, domain.EventTypeSecurity, "payload")
	require.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestNotificationRouter_RouteAndSend_UnsupportedType(t *testing.T) {
	router := NewNotificationRouter()

	ch := domain.Channel{Type: "unknown"}
	err := router.RouteAndSend(context.Background(), ch, domain.EventTypeSecurity, "payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported channel type")
}

func TestNotificationRouter_RouteAndSend_SenderError(t *testing.T) {
	sender := new(mockSender)
	sender.On("Type").Return(domain.ChannelTypeTelegram)
	sender.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("send failed"))

	router := NewNotificationRouter(sender)

	ch := domain.Channel{Type: domain.ChannelTypeTelegram, Credentials: []byte(`{}`)}
	err := router.RouteAndSend(context.Background(), ch, domain.EventTypeSecurity, "payload")
	assert.Error(t, err)
}

func TestNotificationRouter_MultipleSenders(t *testing.T) {
	slackSender := new(mockSender)
	slackSender.On("Type").Return(domain.ChannelTypeSlack)

	tgSender := new(mockSender)
	tgSender.On("Type").Return(domain.ChannelTypeTelegram)
	tgSender.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	router := NewNotificationRouter(slackSender, tgSender)

	ch := domain.Channel{Type: domain.ChannelTypeTelegram, Credentials: []byte(`{}`)}
	err := router.RouteAndSend(context.Background(), ch, domain.EventTypeSecurity, "payload")
	require.NoError(t, err)
	tgSender.AssertCalled(t, "Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	slackSender.AssertNotCalled(t, "Send")
}
