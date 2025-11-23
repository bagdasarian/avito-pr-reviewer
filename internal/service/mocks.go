package service

import (
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/mock"
)

type MockTeamRepository struct {
	mock.Mock
}

func (m *MockTeamRepository) Create(team *domain.Team) error {
	args := m.Called(team)
	return args.Error(0)
}

func (m *MockTeamRepository) GetByName(name string) (*domain.Team, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Team), args.Error(1)
}

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Update(user *domain.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id string) (*domain.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetActiveByTeamID(teamID int) ([]*domain.User, error) {
	args := m.Called(teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) GetByTeamID(teamID int) ([]*domain.User, error) {
	args := m.Called(teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (m *MockUserRepository) SetIsActive(userID string, isActive bool) error {
	args := m.Called(userID, isActive)
	return args.Error(0)
}

type MockPullRequestRepository struct {
	mock.Mock
}

func (m *MockPullRequestRepository) Create(pr *domain.PullRequest) error {
	args := m.Called(pr)
	return args.Error(0)
}

func (m *MockPullRequestRepository) GetByID(id string) (*domain.PullRequest, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PullRequest), args.Error(1)
}

func (m *MockPullRequestRepository) UpdateStatus(id string, status domain.Status, mergedAt *time.Time) error {
	args := m.Called(id, status, mergedAt)
	return args.Error(0)
}

func (m *MockPullRequestRepository) AddReviewer(prID string, reviewerID string) error {
	args := m.Called(prID, reviewerID)
	return args.Error(0)
}

func (m *MockPullRequestRepository) RemoveReviewer(prID string, reviewerID string) error {
	args := m.Called(prID, reviewerID)
	return args.Error(0)
}

func (m *MockPullRequestRepository) GetReviewersByPRID(prID string) ([]string, error) {
	args := m.Called(prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockPullRequestRepository) GetPRsByReviewerID(reviewerID string) ([]*domain.PullRequestShort, error) {
	args := m.Called(reviewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.PullRequestShort), args.Error(1)
}

func (m *MockPullRequestRepository) ReplaceReviewer(prID string, oldReviewerID string, newReviewerID string) error {
	args := m.Called(prID, oldReviewerID, newReviewerID)
	return args.Error(0)
}
