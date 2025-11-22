package repository

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type TeamRepository interface {
	Create(team *domain.Team) error
	GetByName(name string) (*domain.Team, error)
}
