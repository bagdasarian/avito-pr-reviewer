package service

import (
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type teamService struct {
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
}

// NewTeamService создает новый экземпляр TeamService
func NewTeamService(teamRepo repository.TeamRepository, userRepo repository.UserRepository) TeamService {
	return &teamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

// CreateTeam создает команду с участниками
func (s *teamService) CreateTeam(team *domain.Team) (*domain.Team, error) {
	existingTeam, err := s.teamRepo.GetByName(team.Name)
	if err == nil && existingTeam != nil {
		return nil, domain.ErrTeamExists
	}

	team.CreatedAt = time.Now()
	team.UpdatedAt = nil

	err = s.teamRepo.Create(team)
	if err != nil {
		return nil, err
	}

	createdTeam, err := s.teamRepo.GetByName(team.Name)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, domain.NewNotFoundError("team with name " + team.Name)
		}
		return nil, err
	}

	if createdTeam.UpdatedAt != nil {
		return nil, domain.ErrTeamExists
	}

	return createdTeam, nil
}

// GetTeam получает команду с участниками по имени
func (s *teamService) GetTeam(name string) (*domain.Team, error) {
	team, err := s.teamRepo.GetByName(name)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, domain.NewNotFoundError("team with name " + name)
		}
		return nil, err
	}

	return team, nil
}
