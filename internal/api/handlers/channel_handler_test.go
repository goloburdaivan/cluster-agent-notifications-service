package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/api/middleware"
	"notifications-service/internal/core/domain"
)

type mockChannelService struct {
	mock.Mock
}

func (m *mockChannelService) Create(ctx context.Context, cmd domain.CreateChannelCmd) (domain.Channel, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelService) Update(ctx context.Context, cmd domain.UpdateChannelCmd) (domain.Channel, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelService) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *mockChannelService) Get(ctx context.Context, id string) (domain.Channel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelService) List(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func setupTestRouter(svc *mockChannelService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewChannelHandler(svc)
	r := gin.New()
	r.Use(middleware.ErrorHandler())
	g := r.Group("/channels")
	{
		g.GET("", h.List)
		g.GET("/:id", h.Get)
		g.POST("", h.Create)
		g.PATCH("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
	}
	return r
}

func TestHandler_Create_Success(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	created := domain.Channel{
		Id:          "uuid-1",
		Name:        "test-channel",
		Type:        domain.ChannelTypeSlack,
		Credentials: []byte(`{"token":"abc"}`),
		Enabled:     true,
	}

	svc.On("Create", mock.Anything, mock.MatchedBy(func(cmd domain.CreateChannelCmd) bool {
		return cmd.Name == "test-channel" && cmd.Type == domain.ChannelTypeSlack
	})).Return(created, nil)

	body := `{"name":"test-channel","type":"slack","credentials":{"token":"abc"},"enabled":true}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/channels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp channelResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "uuid-1", resp.ID)
	assert.Equal(t, "test-channel", resp.Name)
}

func TestHandler_Create_BindError(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/channels", bytes.NewBufferString(`{}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Create_ServiceError(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("Create", mock.Anything, mock.Anything).Return(domain.Channel{}, domain.NewAlreadyExistsError("duplicate"))

	body := `{"name":"test","type":"slack","credentials":{"token":"abc"}}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/channels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_Get_Success(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	channel := domain.Channel{Id: "uuid-1", Name: "test", Type: domain.ChannelTypeSlack, Enabled: true}
	svc.On("Get", mock.Anything, "uuid-1").Return(channel, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/channels/uuid-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp channelResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "uuid-1", resp.ID)
}

func TestHandler_Get_NotFound(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("Get", mock.Anything, "uuid-999").Return(domain.Channel{}, domain.NewNotFoundError("not found"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/channels/uuid-999", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_List_Success(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	channels := []domain.Channel{
		{Id: "1", Name: "ch1", Type: domain.ChannelTypeSlack},
		{Id: "2", Name: "ch2", Type: domain.ChannelTypeTelegram},
	}
	svc.On("List", mock.Anything).Return(channels, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/channels", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []channelResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Len(t, resp, 2)
}

func TestHandler_List_Empty(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("List", mock.Anything).Return([]domain.Channel{}, nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/channels", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []channelResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Empty(t, resp)
}

func TestHandler_List_Error(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("List", mock.Anything).Return(nil, errors.New("db error"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/channels", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestHandler_Update_Success(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	updated := domain.Channel{Id: "uuid-1", Name: "updated", Type: domain.ChannelTypeSlack, Enabled: true}
	svc.On("Update", mock.Anything, mock.MatchedBy(func(cmd domain.UpdateChannelCmd) bool {
		return cmd.Id == "uuid-1" && cmd.Name != nil && *cmd.Name == "updated"
	})).Return(updated, nil)

	body := `{"name":"updated"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/channels/uuid-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_WithType(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	updated := domain.Channel{Id: "uuid-1", Name: "ch", Type: domain.ChannelTypeTelegram}
	svc.On("Update", mock.Anything, mock.MatchedBy(func(cmd domain.UpdateChannelCmd) bool {
		return cmd.Id == "uuid-1" && cmd.Type != nil && *cmd.Type == domain.ChannelTypeTelegram
	})).Return(updated, nil)

	body := `{"type":"telegram"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/channels/uuid-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_Update_BindError(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	body := `{"type":"invalid_type"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/channels/uuid-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_Update_ServiceError(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("Update", mock.Anything, mock.Anything).Return(domain.Channel{}, domain.NewNotFoundError("not found"))

	body := `{"name":"updated"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/channels/uuid-1", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_Delete_Success(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("Delete", mock.Anything, "uuid-1").Return(nil)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/channels/uuid-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_Delete_NotFound(t *testing.T) {
	svc := new(mockChannelService)
	router := setupTestRouter(svc)

	svc.On("Delete", mock.Anything, "uuid-1").Return(domain.NewNotFoundError("not found"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/channels/uuid-1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
