package service

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type TeamService interface {
	CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error)
	GetTeam(ctx context.Context, name string) (*domain.Team, error)
}
