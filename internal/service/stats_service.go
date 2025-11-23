package service

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type StatsService interface {
	GetReviewerStats() ([]*domain.ReviewerStat, error)
	GetPRStatsByStatus() ([]*domain.PRStatusStat, error)
}
