package providers

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/infra/clients/email"
)

func TestEmailSender_Type(t *testing.T) {
	s := NewEmailSender(nil, nil)
	assert.Equal(t, domain.ChannelTypeEmail, s.Type())
}

func TestEmailSender_Send_Success(t *testing.T) {
	client := new(mockEmailClient)
	renderer := new(mockRenderer)
	sender := NewEmailSender(client, renderer)

	creds := []byte(`{"host":"smtp.example.com","port":"587","username":"user","password":"pass","from":"from@example.com","to":"to@example.com"}`)
	renderer.On("Render", domain.EventTypeSecurity, domain.ChannelTypeEmail, mock.Anything).Return("<h1>Alert</h1>", nil)
	client.On("Send", mock.Anything, email.SMTPConfig{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "user",
		Password: "pass",
		From:     "from@example.com",
	}, "to@example.com", "Security Alert: Trivy Vulnerability Report", "<h1>Alert</h1>").Return(nil)

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, "payload")
	require.NoError(t, err)
	client.AssertExpectations(t)
}

func TestEmailSender_Send_InvalidCreds(t *testing.T) {
	sender := NewEmailSender(nil, nil)
	err := sender.Send(context.Background(), []byte("invalid"), domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal email credentials")
}

func TestEmailSender_Send_MissingCredFields(t *testing.T) {
	sender := NewEmailSender(nil, nil)
	err := sender.Send(context.Background(), []byte(`{"host":"","port":"","from":"","to":""}`), domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email credentials")
}

func TestEmailSender_Send_InvalidPort(t *testing.T) {
	renderer := new(mockRenderer)
	sender := NewEmailSender(nil, renderer)

	creds := []byte(`{"host":"smtp.example.com","port":"not-a-number","from":"from@example.com","to":"to@example.com"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("text", nil)

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse SMTP port")
}

func TestEmailSender_Send_RenderError(t *testing.T) {
	renderer := new(mockRenderer)
	sender := NewEmailSender(nil, renderer)

	creds := []byte(`{"host":"smtp.example.com","port":"587","from":"from@example.com","to":"to@example.com"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("render error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "render template")
}

func TestEmailSender_Send_ClientError(t *testing.T) {
	client := new(mockEmailClient)
	renderer := new(mockRenderer)
	sender := NewEmailSender(client, renderer)

	creds := []byte(`{"host":"smtp.example.com","port":"587","from":"from@example.com","to":"to@example.com"}`)
	renderer.On("Render", mock.Anything, mock.Anything, mock.Anything).Return("text", nil)
	client.On("Send", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("send error"))

	err := sender.Send(context.Background(), creds, domain.EventTypeSecurity, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "email client failed")
}

func TestSubjectForEvent_KnownTypes(t *testing.T) {
	assert.Equal(t, "Security Alert: Trivy Vulnerability Report", subjectForEvent(domain.EventTypeSecurity))
	assert.Equal(t, "Observability Alert", subjectForEvent(domain.EventTypeObservability))
}

func TestSubjectForEvent_UnknownType(t *testing.T) {
	result := subjectForEvent("custom.event")
	assert.Contains(t, result, "custom.event")
}
