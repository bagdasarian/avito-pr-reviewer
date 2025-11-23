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

func TestStatsIntegration(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	teamRepo := postgres.NewTeamRepository(db)
	userRepo := postgres.NewUserRepository(db)
	prRepo := postgres.NewPullRequestRepository(db)
	statsRepo := postgres.NewStatsRepository(db)

	teamService := service.NewTeamService(db, teamRepo, userRepo)
	prService := service.NewPullRequestService(prRepo, userRepo, teamRepo)
	statsService := service.NewStatsService(statsRepo)

	// Создаём команду с несколькими пользователями
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

	// Создаём несколько PR
	pr1, err := prService.CreatePR(ctx, "pr-1", "PR 1", "u1")
	require.NoError(t, err)
	require.NotEmpty(t, pr1.AssignedReviewers)

	pr2, err := prService.CreatePR(ctx, "pr-2", "PR 2", "u1")
	require.NoError(t, err)
	require.NotEmpty(t, pr2.AssignedReviewers)

	// Получаем статистику по ревьюверам
	reviewerStats, err := statsService.GetReviewerStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, reviewerStats)

	// Получаем статистику по статусам PR
	prStatusStats, err := statsService.GetPRStatsByStatus(ctx)
	require.NoError(t, err)
	require.NotNil(t, prStatusStats)

	// Проверяем, что статистика по ревьюверам содержит данные
	assert.Greater(t, len(reviewerStats), 0, "должна быть статистика по ревьюверам")

	// Проверяем, что ревьюверы из созданных PR есть в статистике
	allReviewers := make(map[string]bool)
	for _, reviewerID := range pr1.AssignedReviewers {
		allReviewers[reviewerID] = true
	}
	for _, reviewerID := range pr2.AssignedReviewers {
		allReviewers[reviewerID] = true
	}

	for reviewerID := range allReviewers {
		found := false
		for _, stat := range reviewerStats {
			if stat.UserID == reviewerID {
				found = true
				assert.Greater(t, stat.AssignmentCount, 0, "количество назначений должно быть больше 0")
				break
			}
		}
		assert.True(t, found, "ревьювер %s должен быть в статистике", reviewerID)
	}

	// Проверяем статистику по статусам
	assert.Greater(t, len(prStatusStats), 0, "должна быть статистика по статусам PR")
	
	// Проверяем, что есть PR со статусом OPEN
	openCount := 0
	for _, stat := range prStatusStats {
		if stat.Status == "OPEN" {
			openCount = stat.Count
			break
		}
	}
	assert.Greater(t, openCount, 0, "должен быть хотя бы один PR со статусом OPEN")
}

