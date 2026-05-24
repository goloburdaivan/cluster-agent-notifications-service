package providers

import (
	"context"

	"github.com/stretchr/testify/mock"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/infra/clients/email"
)

type mockRenderer struct {
	mock.Mock
}

func (m *mockRenderer) Render(eventType domain.EventType, channelType domain.ChannelType, payload any) (string, error) {
	args := m.Called(eventType, channelType, payload)
	return args.String(0), args.Error(1)
}

type mockTelegramClient struct {
	mock.Mock
}

func (m *mockTelegramClient) SendMessage(ctx context.Context, botToken string, chatID string, text string) error {
	args := m.Called(ctx, botToken, chatID, text)
	return args.Error(0)
}

type mockSlackClient struct {
	mock.Mock
}

func (m *mockSlackClient) PostMessage(ctx context.Context, botToken string, channelID string, text string) error {
	args := m.Called(ctx, botToken, channelID, text)
	return args.Error(0)
}

type mockEmailClient struct {
	mock.Mock
}

func (m *mockEmailClient) Send(ctx context.Context, cfg email.SMTPConfig, to string, subject string, body string) error {
	args := m.Called(ctx, cfg, to, subject, body)
	return args.Error(0)
}
