package services

import (
	"context"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

type channelService struct {
	repo ports.ChannelRepository
}

func NewChannelService(repo ports.ChannelRepository) ports.ChannelService {
	return &channelService{
		repo: repo,
	}
}

func (c *channelService) List(ctx context.Context) ([]domain.Channel, error) {
	channels, err := c.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	return channels, nil
}

func (c *channelService) Get(ctx context.Context, id string) (domain.Channel, error) {
	channel, err := c.repo.Get(ctx, id)
	if err != nil {
		return domain.Channel{}, err
	}

	return channel, nil
}

func (c *channelService) Create(ctx context.Context, cmd domain.CreateChannelCmd) (domain.Channel, error) {
	channel := domain.Channel{
		Name:        cmd.Name,
		Type:        cmd.Type,
		Credentials: cmd.Credentials,
		Enabled:     cmd.Enabled,
	}

	createdChannel, err := c.repo.Create(ctx, channel)
	if err != nil {
		return domain.Channel{}, err
	}

	return createdChannel, nil
}

func (c *channelService) Update(ctx context.Context, cmd domain.UpdateChannelCmd) (domain.Channel, error) {
	existingChannel, err := c.repo.Get(ctx, cmd.Id)
	if err != nil {
		return domain.Channel{}, err
	}

	if cmd.Name != nil {
		existingChannel.Name = *cmd.Name
	}
	if cmd.Type != nil {
		existingChannel.Type = *cmd.Type
	}
	if cmd.Credentials != nil {
		existingChannel.Credentials = cmd.Credentials
	}
	if cmd.Enabled != nil {
		existingChannel.Enabled = *cmd.Enabled
	}

	updatedChannel, err := c.repo.Update(ctx, existingChannel)
	if err != nil {
		return domain.Channel{}, err
	}

	return updatedChannel, nil
}

func (c *channelService) Delete(ctx context.Context, id string) error {
	channelToDelete := domain.Channel{Id: id}
	return c.repo.Delete(ctx, channelToDelete)
}
