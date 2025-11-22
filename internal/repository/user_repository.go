package repository

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type UserRepository interface {
	Create(user *domain.User) error
	Update(user *domain.User) error
	GetByID(id string) (*domain.User, error)
	GetActiveByTeamID(teamID int) ([]*domain.User, error)
	GetByTeamID(teamID int) ([]*domain.User, error)
	SetIsActive(userID string, isActive bool) error
}
