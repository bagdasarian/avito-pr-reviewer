package repository

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	CreateWithID(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetActiveByTeamID(ctx context.Context, teamID int) ([]*domain.User, error)
	GetByTeamID(ctx context.Context, teamID int) ([]*domain.User, error)
	SetIsActive(ctx context.Context, userID string, isActive bool) error
}
