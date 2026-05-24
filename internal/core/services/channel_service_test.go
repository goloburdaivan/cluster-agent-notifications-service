package services

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"notifications-service/internal/core/domain"
)

type mockChannelRepository struct {
	mock.Mock
}

func (m *mockChannelRepository) List(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func (m *mockChannelRepository) Get(ctx context.Context, id string) (domain.Channel, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepository) Create(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepository) Update(ctx context.Context, channel domain.Channel) (domain.Channel, error) {
	args := m.Called(ctx, channel)
	return args.Get(0).(domain.Channel), args.Error(1)
}

func (m *mockChannelRepository) Delete(ctx context.Context, channel domain.Channel) error {
	args := m.Called(ctx, channel)
	return args.Error(0)
}

func (m *mockChannelRepository) GetActiveChannels(ctx context.Context) ([]domain.Channel, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Channel), args.Error(1)
}

func TestChannelService_List_Success(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	expected := []domain.Channel{{Id: "1", Name: "test"}}
	repo.On("List", mock.Anything).Return(expected, nil)

	result, err := svc.List(context.Background())
	require.NoError(t, err)
	assert.Equal(t, expected, result)
	repo.AssertExpectations(t)
}

func TestChannelService_List_Error(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	repo.On("List", mock.Anything).Return(nil, errors.New("db error"))

	result, err := svc.List(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestChannelService_Get_Success(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	expected := domain.Channel{Id: "1", Name: "test"}
	repo.On("Get", mock.Anything, "1").Return(expected, nil)

	result, err := svc.Get(context.Background(), "1")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestChannelService_Get_Error(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	repo.On("Get", mock.Anything, "1").Return(domain.Channel{}, errors.New("not found"))

	_, err := svc.Get(context.Background(), "1")
	assert.Error(t, err)
}

func TestChannelService_Create_Success(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	cmd := domain.CreateChannelCmd{
		Name:        "test",
		Type:        domain.ChannelTypeSlack,
		Credentials: []byte(`{"token":"abc"}`),
		Enabled:     true,
	}

	expected := domain.Channel{
		Id:          "1",
		Name:        "test",
		Type:        domain.ChannelTypeSlack,
		Credentials: []byte(`{"token":"abc"}`),
		Enabled:     true,
	}

	repo.On("Create", mock.Anything, mock.MatchedBy(func(ch domain.Channel) bool {
		return ch.Name == "test" && ch.Type == domain.ChannelTypeSlack
	})).Return(expected, nil)

	result, err := svc.Create(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestChannelService_Create_Error(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	cmd := domain.CreateChannelCmd{Name: "test", Type: domain.ChannelTypeSlack, Credentials: []byte(`{}`)}
	repo.On("Create", mock.Anything, mock.Anything).Return(domain.Channel{}, errors.New("create error"))

	_, err := svc.Create(context.Background(), cmd)
	assert.Error(t, err)
}

func TestChannelService_Update_Success(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	existing := domain.Channel{Id: "1", Name: "old", Type: domain.ChannelTypeSlack, Enabled: true}
	newName := "new"
	cmd := domain.UpdateChannelCmd{Id: "1", Name: &newName}

	repo.On("Get", mock.Anything, "1").Return(existing, nil)

	updated := existing
	updated.Name = "new"
	repo.On("Update", mock.Anything, mock.MatchedBy(func(ch domain.Channel) bool {
		return ch.Name == "new"
	})).Return(updated, nil)

	result, err := svc.Update(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, "new", result.Name)
}

func TestChannelService_Update_GetError(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	cmd := domain.UpdateChannelCmd{Id: "1"}
	repo.On("Get", mock.Anything, "1").Return(domain.Channel{}, errors.New("not found"))

	_, err := svc.Update(context.Background(), cmd)
	assert.Error(t, err)
}

func TestChannelService_Update_UpdateError(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	existing := domain.Channel{Id: "1", Name: "old"}
	newName := "new"
	cmd := domain.UpdateChannelCmd{Id: "1", Name: &newName}

	repo.On("Get", mock.Anything, "1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.Anything).Return(domain.Channel{}, errors.New("update error"))

	_, err := svc.Update(context.Background(), cmd)
	assert.Error(t, err)
}

func TestChannelService_Update_AllFields(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	existing := domain.Channel{Id: "1", Name: "old", Type: domain.ChannelTypeSlack, Enabled: false}
	newName := "new"
	newType := domain.ChannelTypeTelegram
	enabled := true
	newCreds := []byte(`{"new":"creds"}`)
	cmd := domain.UpdateChannelCmd{
		Id:          "1",
		Name:        &newName,
		Type:        &newType,
		Credentials: newCreds,
		Enabled:     &enabled,
	}

	repo.On("Get", mock.Anything, "1").Return(existing, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(ch domain.Channel) bool {
		return ch.Name == "new" &&
			ch.Type == domain.ChannelTypeTelegram &&
			ch.Enabled &&
			string(ch.Credentials) == `{"new":"creds"}`
	})).Return(domain.Channel{
		Id: "1", Name: "new", Type: domain.ChannelTypeTelegram, Enabled: true, Credentials: newCreds,
	}, nil)

	result, err := svc.Update(context.Background(), cmd)
	require.NoError(t, err)
	assert.Equal(t, "new", result.Name)
	assert.Equal(t, domain.ChannelTypeTelegram, result.Type)
	assert.True(t, result.Enabled)
}

func TestChannelService_Delete_Success(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	repo.On("Delete", mock.Anything, domain.Channel{Id: "1"}).Return(nil)

	err := svc.Delete(context.Background(), "1")
	assert.NoError(t, err)
}

func TestChannelService_Delete_Error(t *testing.T) {
	repo := new(mockChannelRepository)
	svc := NewChannelService(repo)

	repo.On("Delete", mock.Anything, domain.Channel{Id: "1"}).Return(errors.New("delete error"))

	err := svc.Delete(context.Background(), "1")
	assert.Error(t, err)
}
