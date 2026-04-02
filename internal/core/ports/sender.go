package ports

import (
	"context"
	"notifications-service/internal/core/domain"
)

type ChannelSender interface {
	Type() domain.ChannelType
	Send(ctx context.Context, credentials []byte, eventType domain.EventType, payload any) error
}
