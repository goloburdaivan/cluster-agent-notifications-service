package ports

import (
	"context"
	"notifications-service/internal/core/domain"
)

type NotificationRouter interface {
	RouteAndSend(ctx context.Context, channel domain.Channel, eventType domain.EventType, payload any) error
}
