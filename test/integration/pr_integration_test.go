//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository/postgres"
	"github.com/bagdasarian/avito-pr-reviewer/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePRWithAutoReviewers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	// Создаём репозитории и сервисы
	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// 1. Создаём команду с несколькими пользователями
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}
	createdTeam, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)
	require.NotNil(t, createdTeam)

	// 2. Создаём PR от пользователя u1
	pr, err := prService.CreatePR(ctx, "pr-1", "Test PR", "u1")
	require.NoError(t, err)
	require.NotNil(t, pr)

	// 3. Проверяем результаты
	assert.Equal(t, "pr-1", pr.ID)
	assert.Equal(t, "Test PR", pr.Title)
	assert.Equal(t, "u1", pr.AuthorID)
	assert.Equal(t, domain.StatusOpen, pr.Status)
	assert.LessOrEqual(t, len(pr.AssignedReviewers), 2, "должно быть назначено не более 2 ревьюверов")
	assert.GreaterOrEqual(t, len(pr.AssignedReviewers), 1, "должен быть назначен хотя бы 1 ревьювер")
	assert.NotContains(t, pr.AssignedReviewers, "u1", "автор не должен быть в списке ревьюверов")

	// Проверяем, что все ревьюверы активные и из той же команды
	for _, reviewerID := range pr.AssignedReviewers {
		reviewer, err := userRepo.GetByID(ctx, reviewerID)
		require.NoError(t, err)
		assert.True(t, reviewer.IsActive, "ревьювер должен быть активным")
		assert.Equal(t, createdTeam.ID, reviewer.TeamID, "ревьювер должен быть из команды автора")
	}
}

func TestCreatePRWithoutAvailableReviewers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// Создаём команду только с автором (нет других активных пользователей)
	team := &domain.Team{
		Name: "solo",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Solo", IsActive: true},
		},
	}
	_, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR
	pr, err := prService.CreatePR(ctx, "pr-2", "Solo PR", "u1")
	require.NoError(t, err)
	require.NotNil(t, pr)

	// Проверяем, что ревьюверы не назначены
	assert.Empty(t, pr.AssignedReviewers, "не должно быть ревьюверов, если нет доступных кандидатов")
}

func TestCreatePRWithInactiveUsers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// Создаём команду с активным автором и неактивными пользователями
	team := &domain.Team{
		Name: "mixed",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Active", IsActive: true},
			{UserID: "u2", Username: "Inactive1", IsActive: false},
			{UserID: "u3", Username: "Inactive2", IsActive: false},
		},
	}
	_, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR
	pr, err := prService.CreatePR(ctx, "pr-3", "Mixed PR", "u1")
	require.NoError(t, err)
	require.NotNil(t, pr)

	// Проверяем, что неактивные пользователи не назначены
	assert.Empty(t, pr.AssignedReviewers, "неактивные пользователи не должны быть назначены")
}

func TestReassignReviewer(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// Создаём команду с несколькими пользователями
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
			{UserID: "u4", Username: "Dave", IsActive: true},
		},
	}
	_, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)

	// Создаём PR
	pr, err := prService.CreatePR(ctx, "pr-4", "Test PR", "u1")
	require.NoError(t, err)
	require.NotNil(t, pr)
	require.NotEmpty(t, pr.AssignedReviewers, "должен быть назначен хотя бы один ревьювер")

	oldReviewerID := pr.AssignedReviewers[0]

	// Переназначаем ревьювера
	updatedPR, newReviewerID, err := prService.ReassignReviewer(ctx, "pr-4", oldReviewerID)
	require.NoError(t, err)
	require.NotNil(t, updatedPR)
	require.NotEmpty(t, newReviewerID)

	// Проверяем результаты
	assert.NotEqual(t, oldReviewerID, newReviewerID, "новый ревьювер должен отличаться от старого")
	assert.NotContains(t, updatedPR.AssignedReviewers, oldReviewerID, "старый ревьювер должен быть удалён")
	assert.Contains(t, updatedPR.AssignedReviewers, newReviewerID, "новый ревьювер должен быть в списке")

	// Проверяем, что новый ревьювер из той же команды
	newReviewer, err := userRepo.GetByID(ctx, newReviewerID)
	require.NoError(t, err)
	oldReviewer, err := userRepo.GetByID(ctx, oldReviewerID)
	require.NoError(t, err)
	assert.Equal(t, oldReviewer.TeamID, newReviewer.TeamID, "новый ревьювер должен быть из команды старого ревьювера")
	assert.True(t, newReviewer.IsActive, "новый ревьювер должен быть активным")
}

func TestMergePRAndBlockChanges(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// Создаём команду и PR
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
			{UserID: "u3", Username: "Charlie", IsActive: true},
		},
	}
	_, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)

	pr, err := prService.CreatePR(ctx, "pr-5", "Test PR", "u1")
	require.NoError(t, err)
	require.NotEmpty(t, pr.AssignedReviewers)

	oldReviewerID := pr.AssignedReviewers[0]

	// Merge PR
	mergedPR, err := prService.MergePR(ctx, "pr-5")
	require.NoError(t, err)
	require.NotNil(t, mergedPR)
	assert.Equal(t, domain.StatusMerged, mergedPR.Status)

	// Пытаемся переназначить ревьювера после merge
	_, _, err = prService.ReassignReviewer(ctx, "pr-5", oldReviewerID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "merged", "должна быть ошибка при попытке изменить ревьюверов после merge")
}

func TestMergePRIdempotency(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)

	// Создаём команду и PR
	team := &domain.Team{
		Name: "backend",
		Members: []domain.TeamMember{
			{UserID: "u1", Username: "Alice", IsActive: true},
			{UserID: "u2", Username: "Bob", IsActive: true},
		},
	}
	_, err := teamService.CreateTeam(ctx, team)
	require.NoError(t, err)

	_, err = prService.CreatePR(ctx, "pr-6", "Test PR", "u1")
	require.NoError(t, err)

	// Первый merge
	mergedPR1, err := prService.MergePR(ctx, "pr-6")
	require.NoError(t, err)
	assert.Equal(t, domain.StatusMerged, mergedPR1.Status)

	// Второй merge (идемпотентность)
	mergedPR2, err := prService.MergePR(ctx, "pr-6")
	require.NoError(t, err, "повторный merge не должен вызывать ошибку")
	assert.Equal(t, domain.StatusMerged, mergedPR2.Status)
	assert.NotNil(t, mergedPR2.MergedAt)

	// Проверяем, что PR действительно в статусе MERGED
	_, err = prService.MergePR(ctx, "pr-6")
	require.NoError(t, err, "третий merge также должен быть успешным (идемпотентность)")
}

