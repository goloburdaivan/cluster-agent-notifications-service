package ports

import "notifications-service/internal/core/domain"

type TemplateRenderer interface {
	Render(eventID domain.EventType, channelType domain.ChannelType, payload any) (string, error)
}
