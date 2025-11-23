package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUserService_SetIsActive(t *testing.T) {
	t.Run("успешная установка активности", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u1"
		user := &domain.User{
			ID:        userID,
			Username:  "Alice",
			TeamID:    1,
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now(),
		}

		updatedUser := &domain.User{
			ID:        userID,
			Username:  "Alice",
			TeamID:    1,
			TeamName:  "backend",
			IsActive:  false,
			CreatedAt: user.CreatedAt,
		}

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockUserRepo.On("SetIsActive", mock.Anything, userID, false).Return(nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(updatedUser, nil).Once()

		result, err := service.SetIsActive(ctx, userID, false)

		require.NoError(t, err)
		assert.Equal(t, false, result.IsActive)
		assert.Equal(t, userID, result.ID)
		mockUserRepo.AssertExpectations(t)
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: пользователь не найден", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u999"

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found")).Once()

		result, err := service.SetIsActive(ctx, userID, false)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("ошибка при обновлении", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u1"
		user := &domain.User{
			ID:        userID,
			Username:  "Alice",
			TeamID:    1,
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now(),
		}

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockUserRepo.On("SetIsActive", mock.Anything, userID, false).Return(errors.New("database error")).Once()

		result, err := service.SetIsActive(ctx, userID, false)

		require.Error(t, err)
		assert.Nil(t, result)
		mockUserRepo.AssertExpectations(t)
	})
}

func TestUserService_GetReviewPRs(t *testing.T) {
	t.Run("успешное получение PR для ревью", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u1"
		user := &domain.User{
			ID:        userID,
			Username:  "Alice",
			TeamID:    1,
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now(),
		}

		prs := []*domain.PullRequestShort{
			{
				ID:       "pr-1",
				Title:    "Add feature",
				AuthorID: "u2",
				Status:   domain.StatusOpen,
			},
			{
				ID:       "pr-2",
				Title:    "Fix bug",
				AuthorID: "u3",
				Status:   domain.StatusOpen,
			},
		}

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockPRRepo.On("GetPRsByReviewerID", mock.Anything, userID).Return(prs, nil).Once()

		result, err := service.GetReviewPRs(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "pr-1", result[0].ID)
		mockUserRepo.AssertExpectations(t)
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("успешное получение пустого списка PR", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u1"
		user := &domain.User{
			ID:        userID,
			Username:  "Alice",
			TeamID:    1,
			TeamName:  "backend",
			IsActive:  true,
			CreatedAt: time.Now(),
		}

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(user, nil).Once()
		mockPRRepo.On("GetPRsByReviewerID", mock.Anything, userID).Return([]*domain.PullRequestShort{}, nil).Once()

		result, err := service.GetReviewPRs(ctx, userID)

		require.NoError(t, err)
		assert.Len(t, result, 0)
		mockUserRepo.AssertExpectations(t)
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: пользователь не найден", func(t *testing.T) {
		mockUserRepo := new(mocks.MockUserRepository)
		mockPRRepo := new(mocks.MockPullRequestRepository)

		service := NewUserService(mockUserRepo, mockPRRepo)

		userID := "u999"

		ctx := context.Background()
		mockUserRepo.On("GetByID", mock.Anything, userID).Return(nil, errors.New("user not found")).Once()

		result, err := service.GetReviewPRs(ctx, userID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockUserRepo.AssertExpectations(t)
	})
}
