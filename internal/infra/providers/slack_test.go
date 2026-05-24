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

func TestSlackSender_Type(t *testing.T) {
	s := NewSlackSender(nil, nil)
	assert.Equal(t, domain.ChannelTypeSlack, s.Type())
}

func TestSlackSender_Send_Success(t *testing.T) {
	client := new(mockSlackClient)
	renderer := new(mockRenderer)
	sender := NewSlackSender(client, renderer)

	creds := []byte(`{"bot_token":"xoxb-token","channel_id":"C123"}`)
	renderer.On("Render", domain.EventTypeSecurity, domain.ChannelTypeSlack, mock.Anything).Return("rendered &amp; text", nil)
	client.On("PostMessage", mock.Anything, "xoxb-token", "C123", "rendered & text").Return(nil)

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, "payload")
	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestSlackSender_Send_InvalidCreds(t *testing.T) {
	sender := NewSlackSender(nil, nil)
	err := sender.Send(context.Background(), []byte("invalid"), domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal slack credentials")
}

func TestSlackSender_Send_MissingCredFields(t *testing.T) {
	sender := NewSlackSender(nil, nil)
	err := sender.Send(context.Background(), []byte(`{"bot_token":"","channel_id":""}`), domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid slack credentials")
}

func TestSlackSender_Send_RenderError(t *testing.T) {
	renderer := new(mockRenderer)
	sender := NewSlackSender(nil, renderer)

	creds := []byte(`{"bot_token":"token","channel_id":"C123"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("render error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render template")
}

func TestSlackSender_Send_ClientError(t *testing.T) {
	client := new(mockSlackClient)
	renderer := new(mockRenderer)
	sender := NewSlackSender(client, renderer)

	creds := []byte(`{"bot_token":"token","channel_id":"C123"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("text", nil)
	client.On("PostMessage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("client error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slack client failed")
}
