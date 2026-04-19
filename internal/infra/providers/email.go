package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
	"notifications-service/internal/infra/clients/email"
)

var _ ports.ChannelSender = (*EmailSender)(nil)

type emailCreds struct {
	SMTPHost string `json:"host"`
	SMTPPort string `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	From     string `json:"from"`
	To       string `json:"to"`
}

var eventSubjects = map[domain.EventType]string{
	domain.EventTypeSecurity:      "Security Alert: Trivy Vulnerability Report",
	domain.EventTypeObservability: "Observability Alert",
}

type EmailSender struct {
	client   email.Client
	renderer ports.TemplateRenderer
}

func NewEmailSender(client email.Client, renderer ports.TemplateRenderer) *EmailSender {
	return &EmailSender{
		client:   client,
		renderer: renderer,
	}
}

func (s *EmailSender) Type() domain.ChannelType {
	return domain.ChannelTypeEmail
}

func (s *EmailSender) Send(ctx context.Context, rawCreds []byte, eventType domain.EventType, payload any) error {
	var creds emailCreds
	if err := json.Unmarshal(rawCreds, &creds); err != nil {
		return fmt.Errorf("failed to unmarshal email credentials: %w", err)
	}

	if creds.SMTPHost == "" || creds.SMTPPort == "" || creds.From == "" || len(creds.To) == 0 {
		return fmt.Errorf("invalid email credentials: smtp_host, smtp_port, from, and to are required")
	}

	body, err := s.renderer.Render(eventType, s.Type(), payload)
	if err != nil {
		return fmt.Errorf("failed to render template for event %s: %w", eventType, err)
	}

	subject := subjectForEvent(eventType)
	port, err := strconv.Atoi(creds.SMTPPort)
	if err != nil {
		return fmt.Errorf("failed to parse SMTP port: %w", err)
	}

	cfg := email.SMTPConfig{
		Host:     creds.SMTPHost,
		Port:     port,
		Username: creds.Username,
		Password: creds.Password,
		From:     creds.From,
	}

	if err := s.client.Send(ctx, cfg, creds.To, subject, body); err != nil {
		return fmt.Errorf("email client failed to send message: %w", err)
	}

	return nil
}

func subjectForEvent(eventType domain.EventType) string {
	if subject, ok := eventSubjects[eventType]; ok {
		return subject
	}
	return fmt.Sprintf("Notification: %s", eventType)
}
