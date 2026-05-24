package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

func TestTelegramSender_Type(t *testing.T) {
	s := NewSender(nil, nil)
	assert.Equal(t, domain.ChannelTypeTelegram, s.Type())
}

func TestTelegramSender_Send_Success(t *testing.T) {
	client := new(mockTelegramClient)
	renderer := new(mockRenderer)
	sender := NewSender(client, renderer)

	creds := []byte(`{"bot_token":"token123","chat_id":"chat456"}`)
	renderer.On("Render", domain.EventTypeSecurity, domain.ChannelTypeTelegram, mock.Anything).Return("rendered text", nil)
	client.On("SendMessage", mock.Anything, "token123", "chat456", "rendered text").Return(nil)

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, "payload")
	require.NoError(t, err)
	client.AssertExpectations(t)
	renderer.AssertExpectations(t)
}

func TestTelegramSender_Send_InvalidCreds(t *testing.T) {
	sender := NewSender(nil, nil)
	err := sender.Send(context.Background(), []byte("invalid"), domain.EventTypeSecurity, "payload")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal telegram credentials")
}

func TestTelegramSender_Send_MissingCredFields(t *testing.T) {
	sender := NewSender(nil, nil)
	err := sender.Send(context.Background(), []byte(`{"bot_token":"","chat_id":""}`), domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid telegram credentials")
}

func TestTelegramSender_Send_RenderError(t *testing.T) {
	renderer := new(mockRenderer)
	sender := NewSender(nil, renderer)

	creds := []byte(`{"bot_token":"token","chat_id":"chat"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("render error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render template")
}

func TestTelegramSender_Send_ClientError(t *testing.T) {
	client := new(mockTelegramClient)
	renderer := new(mockRenderer)
	sender := NewSender(client, renderer)

	creds := []byte(`{"bot_token":"token","chat_id":"chat"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("text", nil)
	client.On("SendMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("client error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "telegram client failed")
}
