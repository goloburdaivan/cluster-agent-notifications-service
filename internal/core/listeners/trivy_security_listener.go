package listeners

import (
	"context"
	"encoding/json"
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
	var payload domain.TrivyReportPayload
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return err
	}

	totalCount := len(payload.Vulnerabilities)

	if totalCount > maxVulns {
		payload.Vulnerabilities = payload.Vulnerabilities[:maxVulns]
	}

	channels, err := sh.channelsRepository.GetActiveChannels(ctx)
	if err != nil {
		return err
	}

	templateData := struct {
		domain.TrivyReportPayload
		TotalCount  int
		IsTruncated bool
	}{
		TrivyReportPayload: payload,
		TotalCount:         totalCount,
		IsTruncated:        totalCount > maxVulns,
	}

	for _, channel := range channels {
		_ = sh.notificationRouter.RouteAndSend(ctx, channel, event.EventType, templateData)
	}

	return nil
}
