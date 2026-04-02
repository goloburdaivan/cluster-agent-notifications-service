package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
	"notifications-service/internal/infra/db"
)

type channelRepository struct {
	queries   *db.Queries
	encrypter ports.Encrypter
}

func NewChannelRepository(
	conn db.DBTX,
	encrypter ports.Encrypter,
) ports.ChannelRepository {
	return &channelRepository{
		queries:   db.New(conn),
		encrypter: encrypter,
	}
}

func (c *channelRepository) List(ctx context.Context) ([]domain.Channel, error) {
	channels, err := c.queries.ListChannels(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Channel, 0, len(channels))
	for _, channel := range channels {
		decrypted, err := c.encrypter.Decrypt(channel.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials for channel %s: %w", channel.ID, err)
		}

		result = append(result, domain.Channel{
			Id:          db.UUIDToString(channel.ID),
			Type:        domain.ChannelType(channel.Type),
			Credentials: decrypted,
			Name:        channel.Name,
			Enabled:     channel.Enabled,
		})
	}

	return result, nil
}

func (c *channelRepository) GetActiveChannels(ctx context.Context) ([]domain.Channel, error) {
	channels, err := c.queries.GetActiveChannels(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]domain.Channel, 0, len(channels))
	for _, channel := range channels {
		decrypted, err := c.encrypter.Decrypt(channel.Credentials)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt credentials for channel %s: %w", channel.ID, err)
		}

		result = append(result, domain.Channel{
			Id:          db.UUIDToString(channel.ID),
			Type:        domain.ChannelType(channel.Type),
			Credentials: decrypted,
			Name:        channel.Name,
			Enabled:     true,
		})
	}

	return result, nil
}

func (c *channelRepository) Get(ctx context.Context, id string) (domain.Channel, error) {
	channelID, err := db.StringToUUID(id)
	if err != nil {
		return domain.Channel{}, domain.NewValidationError("invalid channel ID format")
	}

	channel, err := c.queries.GetChannel(ctx, channelID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Channel{}, domain.NewNotFoundError("channel not found")
		}
		return domain.Channel{}, err
	}

	decrypted, err := c.encrypter.Decrypt(channel.Credentials)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("failed to decrypt credentials: %w", err)
	}

	return domain.Channel{
		Id:          db.UUIDToString(channel.ID),
		Type:        domain.ChannelType(channel.Type),
		Credentials: decrypted,
		Name:        channel.Name,
		Enabled:     channel.Enabled,
	}, nil
}

func (c *channelRepository) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	encryptedCreds, err := c.encrypter.Encrypt(channel.Credentials)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	created, err := c.queries.CreateChannel(ctx, db.CreateChannelParams{
		Name:        channel.Name,
		Credentials: encryptedCreds,
		Type:        db.ChannelType(channel.Type),
		Enabled:     channel.Enabled,
	})
	if err != nil {
		return domain.Channel{}, err
	}

	return domain.Channel{
		Id:          db.UUIDToString(created.ID),
		Type:        domain.ChannelType(created.Type),
		Credentials: channel.Credentials,
		Name:        created.Name,
		Enabled:     created.Enabled,
	}, nil
}

func (c *channelRepository) Update(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	channelID, err := db.StringToUUID(channel.Id)
	if err != nil {
		return domain.Channel{}, domain.NewValidationError("invalid channel ID format")
	}

	encryptedCreds, err := c.encrypter.Encrypt(channel.Credentials)
	if err != nil {
		return domain.Channel{}, fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	updated, err := c.queries.UpdateChannel(ctx, db.UpdateChannelParams{
		ID:          channelID,
		Name:        channel.Name,
		Credentials: encryptedCreds,
		Type:        db.ChannelType(channel.Type),
		Enabled:     channel.Enabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Channel{}, domain.NewNotFoundError("channel to update not found")
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.Channel{}, domain.NewAlreadyExistsError("channel name already taken")
		}
		return domain.Channel{}, err
	}

	return domain.Channel{
		Id:          db.UUIDToString(updated.ID),
		Type:        domain.ChannelType(updated.Type),
		Credentials: channel.Credentials,
		Name:        updated.Name,
		Enabled:     updated.Enabled,
	}, nil
}

func (c *channelRepository) Delete(ctx context.Context, channel domain.Channel) error {
	channelID, err := db.StringToUUID(channel.Id)
	if err != nil {
		return domain.NewValidationError("invalid channel ID format")
	}

	rowsAffected, err := c.queries.DeleteChannel(ctx, channelID)
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return domain.NewNotFoundError("channel not found")
	}

	return nil
}
