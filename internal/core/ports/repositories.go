package ports

import (
	"context"
	"notifications-service/internal/core/domain"
)

type (
	ChannelRepository interface {
		List(ctx context.Context) ([]domain.Channel, error)
		Get(ctx context.Context, id string) (domain.Channel, error)
		Create(ctx context.Context, channel domain.Channel) (domain.Channel, error)
		Update(ctx context.Context, channel domain.Channel) (domain.Channel, error)
		Delete(ctx context.Context, channel domain.Channel) error
		GetActiveChannels(ctx context.Context) ([]domain.Channel, error)
	}
)
