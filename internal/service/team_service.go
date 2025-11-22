package service

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type TeamService interface {
	// CreateTeam создает команду с участниками
	CreateTeam(team *domain.Team) (*domain.Team, error)

	// GetTeam получает команду с участниками по имени
	GetTeam(name string) (*domain.Team, error)
}

