package repository

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type StatsRepository interface {
	GetReviewerStats(ctx context.Context) ([]*domain.ReviewerStat, error)
	GetPRStatsByStatus(ctx context.Context) ([]*domain.PRStatusStat, error)
}
