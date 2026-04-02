package ports

import (
	"context"
	"notifications-service/internal/core/domain"
)

type (
	ChannelService interface {
		Create(ctx context.Context, cmd domain.CreateChannelCmd) (domain.Channel, error)
		Update(ctx context.Context, cmd domain.UpdateChannelCmd) (domain.Channel, error)
		Delete(ctx context.Context, id string) error
		Get(ctx context.Context, id string) (domain.Channel, error)
		List(ctx context.Context) ([]domain.Channel, error)
	}
)
