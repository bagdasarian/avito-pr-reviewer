package repository

import (
	"context"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type PullRequestRepository interface {
	Create(ctx context.Context, pr *domain.PullRequest) error
	GetByID(ctx context.Context, id string) (*domain.PullRequest, error)
	UpdateStatus(ctx context.Context, id string, status domain.Status, mergedAt *time.Time) error
	AddReviewer(ctx context.Context, prID string, reviewerID string) error
	RemoveReviewer(ctx context.Context, prID string, reviewerID string) error
	GetReviewersByPRID(ctx context.Context, prID string) ([]string, error)
	GetPRsByReviewerID(ctx context.Context, reviewerID string) ([]*domain.PullRequestShort, error)
	ReplaceReviewer(ctx context.Context, prID string, oldReviewerID string, newReviewerID string) error
}
