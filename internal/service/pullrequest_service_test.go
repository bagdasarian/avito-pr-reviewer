package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPullRequestService_CreatePR(t *testing.T) {
	t.Run("успешное создание PR с ревьюверами", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		title := "Add feature"
		authorID := "u1"

		author := &domain.User{
			ID:       authorID,
			Username: "Alice",
			TeamID:   1,
			TeamName: "backend",
			IsActive: true,
			CreatedAt: time.Now(),
		}

		team := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
			},
			CreatedAt: time.Now(),
		}

		teamMembers := []*domain.User{
			{ID: "u1", Username: "Alice", TeamID: 1, TeamName: "backend", IsActive: true},
			{ID: "u2", Username: "Bob", TeamID: 1, TeamName: "backend", IsActive: true},
			{ID: "u3", Username: "Charlie", TeamID: 1, TeamName: "backend", IsActive: true},
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()
		mockUserRepo.On("GetByID", mock.Anything, authorID).Return(author, nil).Once()
		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(team, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return(teamMembers, nil).Once()
		mockPRRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.PullRequest")).Return(nil).Once()

		result, err := service.CreatePR(context.Background(), prID, title, authorID)

		require.NoError(t, err)
		assert.Equal(t, prID, result.ID)
		assert.Equal(t, title, result.Title)
		assert.Equal(t, authorID, result.AuthorID)
		assert.Equal(t, domain.StatusOpen, result.Status)
		assert.LessOrEqual(t, len(result.AssignedReviewers), 2)
		assert.NotContains(t, result.AssignedReviewers, authorID)
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: PR уже существует", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		existingPR := &domain.PullRequest{
			ID:       prID,
			Title:    "Existing PR",
			AuthorID: "u1",
			Status:   domain.StatusOpen,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(existingPR, nil).Once()

		result, err := service.CreatePR(context.Background(), prID, "New PR", "u1")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrPRExists))
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: автор не найден", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		authorID := "u999"

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()
		mockUserRepo.On("GetByID", mock.Anything, authorID).Return(nil, errors.New("user not found")).Once()

		result, err := service.CreatePR(context.Background(), prID, "New PR", authorID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда не найдена", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		authorID := "u1"

		author := &domain.User{
			ID:       authorID,
			Username: "Alice",
			TeamID:   1,
			TeamName: "nonexistent",
			IsActive: true,
			CreatedAt: time.Now(),
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()
		mockUserRepo.On("GetByID", mock.Anything, authorID).Return(author, nil).Once()
		mockTeamRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, errors.New("team not found")).Once()

		result, err := service.CreatePR(context.Background(), prID, "New PR", authorID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("создание PR без ревьюверов (нет доступных)", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		title := "Add feature"
		authorID := "u1"

		author := &domain.User{
			ID:       authorID,
			Username: "Alice",
			TeamID:   1,
			TeamName: "backend",
			IsActive: true,
			CreatedAt: time.Now(),
		}

		team := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
			CreatedAt: time.Now(),
		}

		teamMembers := []*domain.User{
			{ID: "u1", Username: "Alice", TeamID: 1, TeamName: "backend", IsActive: true},
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()
		mockUserRepo.On("GetByID", mock.Anything, authorID).Return(author, nil).Once()
		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(team, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return(teamMembers, nil).Once()
		mockPRRepo.On("Create", mock.Anything, mock.AnythingOfType("*domain.PullRequest")).Return(nil).Once()

		result, err := service.CreatePR(context.Background(), prID, title, authorID)

		require.NoError(t, err)
		assert.Equal(t, prID, result.ID)
		assert.Empty(t, result.AssignedReviewers)
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTeamRepo.AssertExpectations(t)
	})
}

func TestPullRequestService_MergePR(t *testing.T) {
	t.Run("успешный merge PR", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		openPR := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		mergedTime := time.Now()
		mergedPR := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusMerged,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         openPR.CreatedAt,
			MergedAt:          &mergedTime,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(openPR, nil).Once()
		mockPRRepo.On("UpdateStatus", mock.Anything, prID, domain.StatusMerged, mock.AnythingOfType("*time.Time")).Return(nil).Once()
		mockPRRepo.On("GetByID", mock.Anything, prID).Return(mergedPR, nil).Once()

		result, err := service.MergePR(context.Background(), prID)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusMerged, result.Status)
		assert.NotNil(t, result.MergedAt)
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("идемпотентность: PR уже в статусе MERGED", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		mergedTime := time.Now()
		mergedPR := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusMerged,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         time.Now(),
			MergedAt:          &mergedTime,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(mergedPR, nil).Once()

		result, err := service.MergePR(context.Background(), prID)

		require.NoError(t, err)
		assert.Equal(t, domain.StatusMerged, result.Status)
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-999"

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()

		result, err := service.MergePR(context.Background(), prID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockPRRepo.AssertExpectations(t)
	})
}

func TestPullRequestService_ReassignReviewer(t *testing.T) {
	t.Run("успешное переназначение ревьювера", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		oldReviewerID := "u2"
		newReviewerID := "u3"

		pr := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{oldReviewerID, "u4"},
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		oldReviewer := &domain.User{
			ID:       oldReviewerID,
			Username: "Bob",
			TeamID:   1,
			TeamName: "backend",
			IsActive: true,
			CreatedAt: time.Now(),
		}

		team := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u2", Username: "Bob", IsActive: true},
				{UserID: "u3", Username: "Charlie", IsActive: true},
				{UserID: "u5", Username: "Dave", IsActive: true},
			},
			CreatedAt: time.Now(),
		}

		teamMembers := []*domain.User{
			{ID: "u2", Username: "Bob", TeamID: 1, TeamName: "backend", IsActive: true},
			{ID: "u3", Username: "Charlie", TeamID: 1, TeamName: "backend", IsActive: true},
			{ID: "u5", Username: "Dave", TeamID: 1, TeamName: "backend", IsActive: true},
		}

		updatedPR := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{newReviewerID, "u4"},
			CreatedAt:         pr.CreatedAt,
			MergedAt:          nil,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(pr, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, oldReviewerID).Return(oldReviewer, nil).Once()
		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(team, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return(teamMembers, nil).Once()
		mockPRRepo.On("ReplaceReviewer", mock.Anything, prID, oldReviewerID, mock.AnythingOfType("string")).Return(nil).Once()
		mockPRRepo.On("GetByID", mock.Anything, prID).Return(updatedPR, nil).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, oldReviewerID)

		require.NoError(t, err)
		assert.Equal(t, prID, result.ID)
		assert.NotEmpty(t, newReviewer)
		assert.NotEqual(t, oldReviewerID, newReviewer)
		assert.NotContains(t, result.AssignedReviewers, oldReviewerID)
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-999"

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(nil, errors.New("pull request not found")).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, "u2")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, newReviewer)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: PR уже в статусе MERGED", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		mergedTime := time.Now()
		mergedPR := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusMerged,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         time.Now(),
			MergedAt:          &mergedTime,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(mergedPR, nil).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, "u2")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, newReviewer)
		assert.True(t, errors.Is(err, domain.ErrPRMerged))
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: ревьювер не назначен на PR", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		pr := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2"},
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(pr, nil).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, "u999")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, newReviewer)
		assert.True(t, errors.Is(err, domain.ErrNotAssigned))
		mockPRRepo.AssertExpectations(t)
	})

	t.Run("ошибка: нет доступных кандидатов для замены", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		oldReviewerID := "u2"

		pr := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{oldReviewerID},
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		oldReviewer := &domain.User{
			ID:       oldReviewerID,
			Username: "Bob",
			TeamID:   1,
			TeamName: "backend",
			IsActive: true,
			CreatedAt: time.Now(),
		}

		team := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
			CreatedAt: time.Now(),
		}

		teamMembers := []*domain.User{
			{ID: "u2", Username: "Bob", TeamID: 1, TeamName: "backend", IsActive: true},
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(pr, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, oldReviewerID).Return(oldReviewer, nil).Once()
		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(team, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return(teamMembers, nil).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, oldReviewerID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, newReviewer)
		assert.True(t, errors.Is(err, domain.ErrNoCandidate))
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: старый ревьювер не найден", func(t *testing.T) {
		mockPRRepo := new(MockPullRequestRepository)
		mockUserRepo := new(MockUserRepository)
		mockTeamRepo := new(MockTeamRepository)

		service := NewPullRequestService(mockPRRepo, mockUserRepo, mockTeamRepo)

		prID := "pr-1"
		oldReviewerID := "u999"

		pr := &domain.PullRequest{
			ID:                prID,
			Title:             "Add feature",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{oldReviewerID},
			CreatedAt:         time.Now(),
			MergedAt:          nil,
		}

		mockPRRepo.On("GetByID", mock.Anything, prID).Return(pr, nil).Once()
		mockUserRepo.On("GetByID", mock.Anything, oldReviewerID).Return(nil, errors.New("user not found")).Once()

		result, newReviewer, err := service.ReassignReviewer(context.Background(), prID, oldReviewerID)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Empty(t, newReviewer)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockPRRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})
}
