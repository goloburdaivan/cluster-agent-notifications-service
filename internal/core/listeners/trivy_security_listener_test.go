package listeners

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

type mockChannelRepo struct {
	mock.Mock
}

func (m *mockChannelRepo) List(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func (m *mockChannelRepo) Get(ctx context.Context, id string) (domain.Channel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepo) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepo) Update(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepo) Delete(ctx context.Context, channel domain.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *mockChannelRepo) GetActiveChannels(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Channel), args.Error(1)
}

type mockRouter struct {
	mock.Mock
}

func (m *mockRouter) RouteAndSend(ctx context.Context, channel domain.Channel, eventType domain.EventType, payload any) error {
	args := m.Called(ctx, channel, eventType, payload)
	return args.Error(0)
}

func TestSecurityHandler_Handle_Success(t *testing.T) {
	repo := new(mockChannelRepo)
	router := new(mockRouter)
	handler := NewSecurityHandler(repo, router)

	vuln := domain.TrivyVulnerability{
		ReportName: "test-report",
		Namespace:  "default",
		Vulnerabilities: []domain.VulnerabilityDetail{
			{VulnerabilityID: "CVE-2024-001", Severity: "CRITICAL"},
		},
	}
	payload, _ := json.Marshal(vuln)

	channels := []domain.Channel{
		{Id: "1", Type: domain.ChannelTypeSlack, Enabled: true},
	}

	repo.On("GetActiveChannels", mock.Anything).Return(channels, nil)
	router.On("RouteAndSend", mock.Anything, channels[0], domain.EventTypeSecurity, mock.AnythingOfType("domain.TrivyVulnerability")).Return(nil)

	event := domain.NotificationEnvelope{
		EventType: domain.EventTypeSecurity,
		Payload:   payload,
	}

	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)
	repo.AssertExpectations(t)
	router.AssertExpectations(t)
}

func TestSecurityHandler_Handle_MultipleChannels(t *testing.T) {
	repo := new(mockChannelRepo)
	router := new(mockRouter)
	handler := NewSecurityHandler(repo, router)

	vuln := domain.TrivyVulnerability{ReportName: "report"}
	payload, _ := json.Marshal(vuln)

	channels := []domain.Channel{
		{Id: "1", Type: domain.ChannelTypeSlack},
		{Id: "2", Type: domain.ChannelTypeTelegram},
	}

	repo.On("GetActiveChannels", mock.Anything).Return(channels, nil)
	router.On("RouteAndSend", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity, Payload: payload}
	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)
	router.AssertNumberOfCalls(t, "RouteAndSend", 2)
}

func TestSecurityHandler_Handle_InvalidPayload(t *testing.T) {
	handler := NewSecurityHandler(nil, nil)

	event := domain.NotificationEnvelope{
		EventType: domain.EventTypeSecurity,
		Payload:   []byte("invalid json"),
	}

	err := handler.Handle(context.Background(), event)
	assert.Error(t, err)
}

func TestSecurityHandler_Handle_GetActiveChannelsError(t *testing.T) {
	repo := new(mockChannelRepo)
	handler := NewSecurityHandler(repo, nil)

	vuln := domain.TrivyVulnerability{}
	payload, _ := json.Marshal(vuln)

	repo.On("GetActiveChannels", mock.Anything).Return(nil, errors.New("db error"))

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity, Payload: payload}
	err := handler.Handle(context.Background(), event)
	assert.Error(t, err)
}

func TestSecurityHandler_Handle_RouterError(t *testing.T) {
	repo := new(mockChannelRepo)
	router := new(mockRouter)
	handler := NewSecurityHandler(repo, router)

	vuln := domain.TrivyVulnerability{}
	payload, _ := json.Marshal(vuln)

	channels := []domain.Channel{{Id: "1", Type: domain.ChannelTypeSlack}}
	repo.On("GetActiveChannels", mock.Anything).Return(channels, nil)
	router.On("RouteAndSend", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("route error"))

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity, Payload: payload}
	err := handler.Handle(context.Background(), event)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification router error")
}

func TestSecurityHandler_Handle_TruncatesVulnerabilities(t *testing.T) {
	repo := new(mockChannelRepo)
	router := new(mockRouter)
	handler := NewSecurityHandler(repo, router)

	vulns := make([]domain.VulnerabilityDetail, 20)
	for i := range vulns {
		vulns[i] = domain.VulnerabilityDetail{VulnerabilityID: fmt.Sprintf("CVE-%d", i)}
	}

	vuln := domain.TrivyVulnerability{Vulnerabilities: vulns}
	payload, _ := json.Marshal(vuln)

	channels := []domain.Channel{{Id: "1", Type: domain.ChannelTypeSlack}}
	repo.On("GetActiveChannels", mock.Anything).Return(channels, nil)
	router.On("RouteAndSend", mock.Anything, mock.Anything, mock.Anything, mock.MatchedBy(func(v domain.TrivyVulnerability) bool {
		return len(v.Vulnerabilities) == maxVulns
	})).Return(nil)

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity, Payload: payload}
	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)
	router.AssertExpectations(t)
}

func TestSecurityHandler_Handle_NoChannels(t *testing.T) {
	repo := new(mockChannelRepo)
	router := new(mockRouter)
	handler := NewSecurityHandler(repo, router)

	vuln := domain.TrivyVulnerability{}
	payload, _ := json.Marshal(vuln)

	repo.On("GetActiveChannels", mock.Anything).Return([]domain.Channel{}, nil)

	event := domain.NotificationEnvelope{EventType: domain.EventTypeSecurity, Payload: payload}
	err := handler.Handle(context.Background(), event)
	require.NoError(t, err)
	router.AssertNotCalled(t, "RouteAndSend")
}
