package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTeamService_CreateTeam(t *testing.T) {
	t.Run("успешное создание команды", func(t *testing.T) {
		db, mockDB := setupMockDBForService(t)
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(db, mockTeamRepo, mockUserRepo)
		ctx := context.Background()

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

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(nil, errors.New("team not found")).Once()
		
		mockDB.ExpectBegin()
		mockDB.ExpectQuery(`INSERT INTO teams`).WithArgs("backend", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), nil))
		mockDB.ExpectQuery(`UPDATE users`).WithArgs(sqlmock.AnyArg(), "Alice", 1, true, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), nil))
		mockDB.ExpectQuery(`UPDATE users`).WithArgs(sqlmock.AnyArg(), "Bob", 1, true, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), nil))
		mockDB.ExpectCommit()

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(createdTeam, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return([]*domain.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		}, nil).Once()

		result, err := service.CreateTeam(ctx, team)

		require.NoError(t, err)
		assert.Equal(t, createdTeam.Name, result.Name)
		assert.Equal(t, len(createdTeam.Members), len(result.Members))
		assert.Nil(t, result.UpdatedAt)
		mockTeamRepo.AssertExpectations(t)
		mockUserRepo.AssertExpectations(t)
		require.NoError(t, mockDB.ExpectationsWereMet())
	})

	t.Run("ошибка: команда уже существует", func(t *testing.T) {
		db, _ := setupMockDBForService(t)
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(db, mockTeamRepo, mockUserRepo)
		ctx := context.Background()

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

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(existingTeam, nil).Once()

		result, err := service.CreateTeam(ctx, team)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrTeamExists))
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда была обновлена (конфликт)", func(t *testing.T) {
		db, mockDB := setupMockDBForService(t)
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(db, mockTeamRepo, mockUserRepo)
		ctx := context.Background()

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

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(nil, errors.New("team not found")).Once()
		
		mockDB.ExpectBegin()
		mockDB.ExpectQuery(`INSERT INTO teams`).WithArgs("backend", sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).AddRow(1, time.Now(), updatedTime))
		mockDB.ExpectQuery(`UPDATE users`).WithArgs(sqlmock.AnyArg(), "Alice", 1, true, sqlmock.AnyArg()).
			WillReturnRows(sqlmock.NewRows([]string{"created_at", "updated_at"}).AddRow(time.Now(), nil))
		mockDB.ExpectCommit()

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(updatedTeam, nil).Once()

		result, err := service.CreateTeam(ctx, team)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrTeamExists))
		mockTeamRepo.AssertExpectations(t)
		require.NoError(t, mockDB.ExpectationsWereMet())
	})
}

func setupMockDBForService(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func TestTeamService_GetTeam(t *testing.T) {
	t.Run("успешное получение команды", func(t *testing.T) {
		db, _ := setupMockDBForService(t)
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(db, mockTeamRepo, mockUserRepo)
		ctx := context.Background()

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

		mockTeamRepo.On("GetByName", mock.Anything, "backend").Return(team, nil).Once()
		mockUserRepo.On("GetByTeamID", mock.Anything, 1).Return([]*domain.User{
			{ID: "u1", Username: "Alice", IsActive: true},
			{ID: "u2", Username: "Bob", IsActive: true},
		}, nil).Once()

		result, err := service.GetTeam(ctx, "backend")

		require.NoError(t, err)
		assert.Equal(t, team.Name, result.Name)
		assert.Equal(t, len(team.Members), len(result.Members))
		mockTeamRepo.AssertExpectations(t)
	})

	t.Run("ошибка: команда не найдена", func(t *testing.T) {
		db, _ := setupMockDBForService(t)
		mockTeamRepo := new(MockTeamRepository)
		mockUserRepo := new(MockUserRepository)

		service := NewTeamService(db, mockTeamRepo, mockUserRepo)
		ctx := context.Background()

		mockTeamRepo.On("GetByName", mock.Anything, "nonexistent").Return(nil, errors.New("team not found")).Once()

		result, err := service.GetTeam(ctx, "nonexistent")

		require.Error(t, err)
		assert.Nil(t, result)
		assert.True(t, errors.Is(err, domain.ErrNotFound))
		mockTeamRepo.AssertExpectations(t)
	})
}
