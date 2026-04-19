package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"html"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
	"notifications-service/internal/infra/clients/slack"
)

var _ ports.ChannelSender = (*SlackSender)(nil)

type slackCreds struct {
	BotToken  string `json:"bot_token"`
	ChannelID string `json:"channel_id"`
}

type SlackSender struct {
	client   slack.Client
	renderer ports.TemplateRenderer
}

func NewSlackSender(client slack.Client, renderer ports.TemplateRenderer) *SlackSender {
	return &SlackSender{
		client:   client,
		renderer: renderer,
	}
}

func (s *SlackSender) Type() domain.ChannelType {
	return domain.ChannelTypeSlack
}

func (s *SlackSender) Send(ctx context.Context, rawCreds []byte, eventType domain.EventType, payload any) error {
	var creds slackCreds
	if err := json.Unmarshal(rawCreds, &creds); err != nil {
		return fmt.Errorf("failed to unmarshal slack credentials: %w", err)
	}

	if creds.BotToken == "" || creds.ChannelID == "" {
		return fmt.Errorf("invalid slack credentials: bot_token or channel_id is missing")
	}

	text, err := s.renderer.Render(eventType, s.Type(), payload)
	if err != nil {
		return fmt.Errorf("failed to render template for event %s: %w", eventType, err)
	}

	// The shared renderer uses html/template which escapes HTML entities;
	// Slack expects plain mrkdwn, so we unescape them back.
	text = html.UnescapeString(text)

	if err := s.client.PostMessage(ctx, creds.BotToken, creds.ChannelID, text); err != nil {
		return fmt.Errorf("slack client failed to send message: %w", err)
	}

	return nil
}
