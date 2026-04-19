package listeners

import (
	"context"
	"encoding/json"
	"fmt"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

const (
	maxVulns = 15
)

type SecurityHandler struct {
	channelsRepository ports.ChannelRepository
	notificationRouter ports.NotificationRouter
}

func NewSecurityHandler(
	channelsRepository ports.ChannelRepository,
	notificationRouter ports.NotificationRouter,
) *SecurityHandler {
	return &SecurityHandler{
		channelsRepository: channelsRepository,
		notificationRouter: notificationRouter,
	}
}

func (sh *SecurityHandler) Handle(ctx context.Context, event domain.NotificationEnvelope) error {
	var payload domain.TrivyVulnerability
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	if len(payload.Vulnerabilities) > maxVulns {
		payload.Vulnerabilities = payload.Vulnerabilities[:maxVulns]
	}

	channels, err := sh.channelsRepository.GetActiveChannels(ctx)
	if err != nil {
		return err
	}

	for _, channel := range channels {
		err = sh.notificationRouter.RouteAndSend(ctx, channel, event.EventType, payload)
		if err != nil {
			return fmt.Errorf("notification router error: %w", err)
		}
	}

	return nil
}
