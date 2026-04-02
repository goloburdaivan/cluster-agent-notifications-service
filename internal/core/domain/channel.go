package domain

import "encoding/json"

type ChannelType string

const (
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeSlack    ChannelType = "slack"
	ChannelTypeTelegram ChannelType = "telegram"
)

type Channel struct {
	Id          string          `json:"id"`
	Type        ChannelType     `json:"type"`
	Credentials json.RawMessage `json:"credentials"`
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
}

type CreateChannelCmd struct {
	Name        string
	Type        ChannelType
	Credentials json.RawMessage
	Enabled     bool
}

type UpdateChannelCmd struct {
	Id          string
	Name        *string
	Type        *ChannelType
	Credentials json.RawMessage
	Enabled     *bool
}
