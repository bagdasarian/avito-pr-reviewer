package service

import (
	"errors"
	"testing"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTeamService_CreateTeam(t *testing.T) {
	t.Run("успешное создание команды", func(t *testing.T) {
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(mockTeamRepo, mockUserRepo)

		team := &domain.Team{
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
		}

		createdTeam := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: nil,
		}

		mockTeamRepo.On("GetByName", "backend").Return(nil, errors.New("team not found")).Once()
		mockTeamRepo.On("Create", mock.AnythingOfType("*domain.Team")).Return(nil).Once()
		mockTeamRepo.On("GetByName", "backend").Return(createdTeam, nil).Once()

		result, err := service.CreateTeam(team)

		require.NoError(t, err)
		assert.Equal(t, createdTeam.Name, result.Name)
		assert.Equal(t, len(createdTeam.Members), len(result.Members))
		assert.Nil(t, result.UpdatedAt)
		mockTeamRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда уже существует", func(t *testing.T) {
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(mockTeamRepo, mockUserRepo)

		team := &domain.Team{
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		existingTeam := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: nil,
		}

		mockTeamRepo.On("GetByName", "backend").Return(existingTeam, nil).Once()

		result, err := service.CreateTeam(team)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrTeamExists))
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда была обновлена (конфликт)", func(t *testing.T) {
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(mockTeamRepo, mockUserRepo)

		team := &domain.Team{
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
		}

		updatedTime := time.Now()
		updatedTeam := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: &updatedTime,
		}

		mockTeamRepo.On("GetByName", "backend").Return(nil, errors.New("team not found")).Once()
		mockTeamRepo.On("Create", mock.AnythingOfType("*domain.Team")).Return(nil).Once()
		mockTeamRepo.On("GetByName", "backend").Return(updatedTeam, nil).Once()

		result, err := service.CreateTeam(team)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrTeamExists))
		mockTeamRepo.AssertExpectations(t)
	})
}

func TestTeamService_GetTeam(t *testing.T) {
	t.Run("успешное получение команды", func(t *testing.T) {
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(mockTeamRepo, mockUserRepo)

		team := &domain.Team{
			ID:   1,
			Name: "backend",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "Alice", IsActive: true},
				{UserID: "u2", Username: "Bob", IsActive: true},
			},
			CreatedAt: time.Now(),
			UpdatedAt: nil,
		}

		mockTeamRepo.On("GetByName", "backend").Return(team, nil).Once()

		result, err := service.GetTeam("backend")

		require.NoError(t, err)
		assert.Equal(t, team.Name, result.Name)
		assert.Equal(t, len(team.Members), len(result.Members))
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда не найдена", func(t *testing.T) {
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(mockTeamRepo, mockUserRepo)

		mockTeamRepo.On("GetByName", "nonexistent").Return(nil, errors.New("team not found")).Once()

		result, err := service.GetTeam("nonexistent")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockTeamRepo.AssertExpectations(t)
	})
}
