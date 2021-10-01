package mocks

import (
	"context"
	"time"

	"github.com/dartkron/leetcodeBot/v2/internal/common"
	"github.com/dartkron/leetcodeBot/v2/pkg/leetcodeclient"
	"github.com/stretchr/testify/mock"
)

// MockLeetcodeClient useful for testing purposes
type MockLeetcodeClient struct {
	mock.Mock
}

// GetDailyTaskItemID use dateID strings to match responses
func (m *MockLeetcodeClient) GetDailyTaskItemID(ctx context.Context, date time.Time) (string, error) {
	args := m.Called(common.GetDateID(date))
	return args.String(0), args.Error(1)
}

// GetQuestionDetailsByItemID mock function meets the interface
func (m *MockLeetcodeClient) GetQuestionDetailsByItemID(ctx context.Context, itemID string) (leetcodeclient.LeetCodeTask, error) {
	args := m.Called(itemID)
	return args.Get(0).(leetcodeclient.LeetCodeTask), args.Error(1)
}

// GetDailyTask use dateID strings to match responses
func (m *MockLeetcodeClient) GetDailyTask(ctx context.Context, date time.Time) (leetcodeclient.LeetCodeTask, error) {
	args := m.Called(common.GetDateID(date))
	return args.Get(0).(leetcodeclient.LeetCodeTask), args.Error(1)
}
