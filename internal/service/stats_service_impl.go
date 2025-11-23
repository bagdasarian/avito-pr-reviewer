package service

import (
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type statsService struct {
	statsRepo repository.StatsRepository
}

func NewStatsService(statsRepo repository.StatsRepository) StatsService {
	return &statsService{statsRepo: statsRepo}
}

func (s *statsService) GetReviewerStats() ([]*domain.ReviewerStat, error) {
	return s.statsRepo.GetReviewerStats()
}

func (s *statsService) GetPRStatsByStatus() ([]*domain.PRStatusStat, error) {
	return s.statsRepo.GetPRStatsByStatus()
}
