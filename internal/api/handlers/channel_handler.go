package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"notifications-service/internal/core/domain"
	"notifications-service/internal/core/ports"
)

type ChannelHandler struct {
	service ports.ChannelService
}

func NewChannelHandler(service ports.ChannelService) *ChannelHandler {
	return &ChannelHandler{
		service: service,
	}
}

type createChannelReq struct {
	Name        string          `json:"name" binding:"required,min=2"`
	Type        string          `json:"type" binding:"required,oneof=email slack telegram"`
	Credentials json.RawMessage `json:"credentials" binding:"required"`
	Enabled     bool            `json:"enabled"`
}

type updateChannelReq struct {
	Name        *string         `json:"name" binding:"omitempty,min=2"`
	Type        *string         `json:"type" binding:"omitempty,oneof=email slack telegram"`
	Credentials json.RawMessage `json:"credentials"`
	Enabled     *bool           `json:"enabled"`
}

type channelResp struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        string          `json:"type"`
	Credentials json.RawMessage `json:"credentials,omitempty"`
	Enabled     bool            `json:"enabled"`
}

func mapToChannelResp(c domain.Channel) channelResp {
	return channelResp{
		ID:          c.Id,
		Name:        c.Name,
		Type:        string(c.Type),
		Credentials: c.Credentials,
		Enabled:     c.Enabled,
	}
}

func (h *ChannelHandler) Create(c *gin.Context) {
	var req createChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	cmd := domain.CreateChannelCmd{
		Name:        req.Name,
		Type:        domain.ChannelType(req.Type),
		Credentials: req.Credentials,
		Enabled:     req.Enabled,
	}

	created, err := h.service.Create(c.Request.Context(), cmd)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, mapToChannelResp(created))
}

func (h *ChannelHandler) Get(c *gin.Context) {
	id := c.Param("id")

	channel, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, mapToChannelResp(channel))
}

func (h *ChannelHandler) List(c *gin.Context) {
	channels, err := h.service.List(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	resp := make([]channelResp, 0, len(channels))
	for _, ch := range channels {
		resp = append(resp, mapToChannelResp(ch))
	}

	c.JSON(http.StatusOK, resp)
}

func (h *ChannelHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var req updateChannelReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	var chType *domain.ChannelType
	if req.Type != nil {
		t := domain.ChannelType(*req.Type)
		chType = &t
	}

	cmd := domain.UpdateChannelCmd{
		Id:          id,
		Name:        req.Name,
		Type:        chType,
		Credentials: req.Credentials,
		Enabled:     req.Enabled,
	}

	updated, err := h.service.Update(c.Request.Context(), cmd)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, mapToChannelResp(updated))
}

func (h *ChannelHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	err := h.service.Delete(c.Request.Context(), id)
	if err != nil {
		c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
