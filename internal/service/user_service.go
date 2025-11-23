package service

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error)
	GetReviewPRs(ctx context.Context, userID string) ([]*domain.PullRequestShort, error)
}
