package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"notifications-service/internal/infra/clients/telegram"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

var _ ports.ChannelSender = (*Sender)(nil)

type telegramCreds struct {
	BotToken string `json:"bot_token"`
	ChatID   string `json:"chat_id"`
}

type Sender struct {
	client   telegram.Client
	renderer ports.TemplateRenderer
}

func NewSender(client telegram.Client, renderer ports.TemplateRenderer) *Sender {
	return &Sender{
		client:   client,
		renderer: renderer,
	}
}

func (s *Sender) Type() domain.ChannelType {
	return domain.ChannelTypeTelegram
}

func (s *Sender) Send(ctx context.Context, rawCreds []byte, eventType domain.EventType, payload any) error {
	var creds telegramCreds
	if err := json.Unmarshal(rawCreds, &creds); err != nil {
		return fmt.Errorf("failed to unmarshal telegram credentials: %w", err)
	}

	if creds.BotToken == "" || creds.ChatID == "" {
		return fmt.Errorf("invalid telegram credentials: bot_token or chat_id is missing")
	}

	text, err := s.renderer.Render(eventType, s.Type(), payload)
	if err != nil {
		return fmt.Errorf("failed to render template for event %s: %w", eventType, err)
	}

	err = s.client.SendMessage(ctx, creds.BotToken, creds.ChatID, text)
	if err != nil {
		return fmt.Errorf("telegram client failed to send message: %w", err)
	}

	return nil
}
