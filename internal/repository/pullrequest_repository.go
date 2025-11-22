package repository

import (
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type PullRequestRepository interface {
	Create(pr *domain.PullRequest) error
	GetByID(id string) (*domain.PullRequest, error)
	UpdateStatus(id string, status domain.Status, mergedAt *time.Time) error
	AddReviewer(prID string, reviewerID string) error
	RemoveReviewer(prID string, reviewerID string) error
	GetReviewersByPRID(prID string) ([]string, error)
	GetPRsByReviewerID(reviewerID string) ([]*domain.PullRequestShort, error)
	ReplaceReviewer(prID string, oldReviewerID string, newReviewerID string) error
}
