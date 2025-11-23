package service

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository/postgres"
)

type teamService struct {
	db       *sql.DB
	teamRepo repository.TeamRepository
	userRepo repository.UserRepository
}

func NewTeamService(db *sql.DB, teamRepo repository.TeamRepository, userRepo repository.UserRepository) TeamService {
	return &teamService{
		db:       db,
		teamRepo: teamRepo,
		userRepo: userRepo,
	}
}

func stringIDToInt(stringID string) (int, error) {
	idStr := strings.TrimPrefix(stringID, "u")
	return strconv.Atoi(idStr)
}

func (s *teamService) CreateTeam(ctx context.Context, team *domain.Team) (*domain.Team, error) {
	existingTeam, err := s.teamRepo.GetByName(ctx, team.Name)
	if err == nil && existingTeam != nil {
		return nil, domain.ErrTeamExists
	}

	team.CreatedAt = time.Now()
	team.UpdatedAt = nil

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	teamRepoWithTx := postgres.NewTeamRepositoryWithTx(tx)
	userRepoWithTx := postgres.NewUserRepositoryWithTx(tx)

	err = teamRepoWithTx.Create(ctx, team)
	if err != nil {
		return nil, err
	}

	for _, member := range team.Members {
		user := &domain.User{
			ID:       member.UserID,
			Username: member.Username,
			TeamID:   team.ID,
			IsActive: member.IsActive,
		}

		_, err := stringIDToInt(member.UserID)
		if err != nil {
			err = userRepoWithTx.Create(ctx, user)
			if err != nil {
				return nil, err
			}
			continue
		}

		user.TeamID = team.ID
		err = userRepoWithTx.Update(ctx, user)
		if err != nil {
			if err.Error() == "user not found" {
				user.ID = member.UserID
				err = userRepoWithTx.CreateWithID(ctx, user)
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	createdTeam, err := s.teamRepo.GetByName(ctx, team.Name)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, domain.NewNotFoundError("team with name " + team.Name)
		}
		return nil, err
	}

	if createdTeam.UpdatedAt != nil {
		return nil, domain.ErrTeamExists
	}

	users, err := s.userRepo.GetByTeamID(ctx, createdTeam.ID)
	if err != nil {
		return nil, err
	}

	createdTeam.Members = make([]domain.TeamMember, 0, len(users))
	for _, user := range users {
		createdTeam.Members = append(createdTeam.Members, domain.TeamMember{
			UserID:   user.ID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return createdTeam, nil
}

func (s *teamService) GetTeam(ctx context.Context, name string) (*domain.Team, error) {
	team, err := s.teamRepo.GetByName(ctx, name)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, domain.NewNotFoundError("team with name " + name)
		}
		return nil, err
	}

	users, err := s.userRepo.GetByTeamID(ctx, team.ID)
	if err != nil {
		return nil, err
	}

	team.Members = make([]domain.TeamMember, 0, len(users))
	for _, user := range users {
		team.Members = append(team.Members, domain.TeamMember{
			UserID:   user.ID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return team, nil
}
