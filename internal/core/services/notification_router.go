package services

import (
	"context"
	"fmt"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

type notificationRouter struct {
	strategies map[domain.ChannelType]ports.ChannelSender
}

func NewNotificationRouter(senders ...ports.ChannelSender) ports.NotificationRouter {
	strategies := make(map[domain.ChannelType]ports.ChannelSender)
	for _, sender := range senders {
		strategies[sender.Type()] = sender
	}
	return &notificationRouter{strategies: strategies}
}

func (r *notificationRouter) RouteAndSend(
	ctx context.Context,
	channel domain.Channel,
	eventType domain.EventType,
	payload any,
) error {
	strategy, exists := r.strategies[channel.Type]
	if !exists {
		return fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	return strategy.Send(ctx, channel.Credentials, eventType, payload)
}
