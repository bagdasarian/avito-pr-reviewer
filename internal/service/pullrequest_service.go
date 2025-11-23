package service

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type PullRequestService interface {
	CreatePR(ctx context.Context, prID, title, authorID string) (*domain.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*domain.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldReviewerID string) (*domain.PullRequest, string, error)
}
