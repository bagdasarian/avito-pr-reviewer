package service

import (
	"context"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockTeamRepository struct {
	mock.Mock
}

func (m *MockTeamRepository) Create(ctx context.Context, team *domain.Team) error {
	args := m.Called(ctx, team)
	return args.Error(0)
}

func (m *MockTeamRepository) GetByName(ctx context.Context, name string) (*domain.Team, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Team), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) CreateWithID(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(ctx context.Context, user *domain.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetActiveByTeamID(ctx context.Context, teamID int) ([]*domain.User, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByTeamID(ctx context.Context, teamID int) ([]*domain.User, error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	args := m.Called(ctx, userID, isActive)
	return args.Error(0)
}

type MockPullRequestRepository struct {
	mock.Mock
}

func (m *MockPullRequestRepository) Create(ctx context.Context, pr *domain.PullRequest) error {
	args := m.Called(ctx, pr)
	return args.Error(0)
}

func (m *MockPullRequestRepository) GetByID(ctx context.Context, id string) (*domain.PullRequest, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PullRequest), args.Error(1)
}

func (m *MockPullRequestRepository) UpdateStatus(ctx context.Context, id string, status domain.Status, mergedAt *time.Time) error {
	args := m.Called(ctx, id, status, mergedAt)
	return args.Error(0)
}

func (m *MockPullRequestRepository) AddReviewer(ctx context.Context, prID string, reviewerID string) error {
	args := m.Called(ctx, prID, reviewerID)
	return args.Error(0)
}

func (m *MockPullRequestRepository) RemoveReviewer(ctx context.Context, prID string, reviewerID string) error {
	args := m.Called(ctx, prID, reviewerID)
	return args.Error(0)
}

func (m *MockPullRequestRepository) GetReviewersByPRID(ctx context.Context, prID string) ([]string, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPullRequestRepository) GetPRsByReviewerID(ctx context.Context, reviewerID string) ([]*domain.PullRequestShort, error) {
	args := m.Called(ctx, reviewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PullRequestShort), args.Error(1)
}

func (m *MockPullRequestRepository) ReplaceReviewer(ctx context.Context, prID string, oldReviewerID string, newReviewerID string) error {
	args := m.Called(ctx, prID, oldReviewerID, newReviewerID)
	return args.Error(0)
}
