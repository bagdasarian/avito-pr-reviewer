package repository

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type StatsRepository interface {
	GetReviewerStats() ([]*domain.ReviewerStat, error)
	GetPRStatsByStatus() ([]*domain.PRStatusStat, error)
}
