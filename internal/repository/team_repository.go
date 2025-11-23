package repository

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type TeamRepository interface {
	Create(ctx context.Context, team *domain.Team) error
	GetByName(ctx context.Context, name string) (*domain.Team, error)
}
