package service

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type statsService struct {
	statsRepo repository.StatsRepository
}

func NewStatsService(statsRepo repository.StatsRepository) StatsService {
	return &statsService{statsRepo: statsRepo}
}

func (s *statsService) GetReviewerStats(ctx context.Context) ([]*domain.ReviewerStat, error) {
	return s.statsRepo.GetReviewerStats(ctx)
}

func (s *statsService) GetPRStatsByStatus(ctx context.Context) ([]*domain.PRStatusStat, error) {
	return s.statsRepo.GetPRStatsByStatus(ctx)
}
