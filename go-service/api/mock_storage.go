package api

import (
	"context"

	"github.com/jamestexas/context-keep/go-service/redis"
	"github.com/stretchr/testify/mock"
)

// MockStorage is a mock implementation of the redis.Storage interface.
type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) GetSummary(ctx context.Context, convoID string) (string, error) {
	args := m.Called(ctx, convoID)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) StoreSummary(ctx context.Context, convoID string, summary string) error {
	args := m.Called(ctx, convoID, summary)
	return args.Error(0)
}

func (m *MockStorage) DeleteConversation(ctx context.Context, convoID string) error {
	args := m.Called(ctx, convoID)
	return args.Error(0)
}

func (m *MockStorage) DeleteSummary(ctx context.Context, convoID string) error {
	args := m.Called(ctx, convoID)
	return args.Error(0)
}

func (m *MockStorage) StoreEvent(ctx context.Context, convoID, eventID, parentID, eventText string, tags []string) error {
	args := m.Called(ctx, convoID, eventID, parentID, eventText, tags)
	return args.Error(0)
}

func (m *MockStorage) GetRecentEvents(ctx context.Context, convoID string, count int) ([]string, error) {
	args := m.Called(ctx, convoID, count)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockStorage) GetEvent(ctx context.Context, convoID, eventID string) (redis.EventNode, error) {
	args := m.Called(ctx, convoID, eventID)
	return args.Get(0).(redis.EventNode), args.Error(1)
}
