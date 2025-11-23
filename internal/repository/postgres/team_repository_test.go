package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTeamRepo создает мок БД и репозиторий для Team
func setupTeamRepo(t *testing.T) (*teamRepository, sqlmock.Sqlmock) {
	db, mock := setupMockDB(t)
	return NewTeamRepository(db), mock
}

// TestTeamRepository_Create - тест для метода Create()
// Этот метод создает команду и всех её участников в транзакции
// Использует ON CONFLICT для upsert команды
func TestTeamRepository_Create(t *testing.T) {
	t.Run("успешное создание новой команды с участниками", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name: "Team Alpha",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
				{UserID: "u2", Username: "user2", IsActive: true},
			},
		}

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team Alpha", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		err := repo.Create(context.Background(), team)

		require.NoError(t, err)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "Team Alpha", team.Name)
		assert.NotNil(t, team.CreatedAt)
		assert.Nil(t, team.UpdatedAt, "updated_at должен быть nil для новой команды")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное обновление существующей команды", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		updatedAt := now.Add(1 * time.Hour)
		team := &domain.Team{
			Name: "Existing Team",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
			},
		}

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now.Add(-7*24*time.Hour), updatedAt)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Existing Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		err := repo.Create(context.Background(), team)

		require.NoError(t, err)
		assert.Equal(t, 1, team.ID)
		assert.NotNil(t, team.UpdatedAt, "updated_at должен быть установлен при обновлении")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное создание команды без участников", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name:    "Empty Team",
			Members: []domain.TeamMember{},
		}

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(2, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Empty Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		err := repo.Create(context.Background(), team)

		require.NoError(t, err)
		assert.Equal(t, 2, team.ID)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("создание команды с несколькими участниками", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name: "Large Team",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
				{UserID: "u2", Username: "user2", IsActive: false},
				{UserID: "u3", Username: "user3", IsActive: true},
			},
		}

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(3, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Large Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		err := repo.Create(context.Background(), team)

		require.NoError(t, err)
		assert.Equal(t, 3, team.ID)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать команду", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		team := &domain.Team{
			Name:    "Team",
			Members: []domain.TeamMember{},
		}

		expectedError := errors.New("database error")
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team", sqlmock.AnyArg()).
			WillReturnError(expectedError)

		err := repo.Create(context.Background(), team)

		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")
	})

	t.Run("ошибка: не удалось начать транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")
	})

	t.Run("ошибка: не удалось создать команду (ошибка БД)", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		team := &domain.Team{
			Name: "Team",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
			},
		}

		expectedError := errors.New("connection failed")
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team", sqlmock.AnyArg()).
			WillReturnError(expectedError)

		err := repo.Create(context.Background(), team)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
	})
}

// TestTeamRepository_GetByName - тест для метода GetByName()
func TestTeamRepository_GetByName(t *testing.T) {
	t.Run("успешное получение команды с участниками", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(1, "Team Alpha", createdAt, updatedAt)
		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("Team Alpha").
			WillReturnRows(teamRows)

		team, err := repo.GetByName(context.Background(), "Team Alpha")

		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "Team Alpha", team.Name)
		assert.NotNil(t, team.UpdatedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение команды без участников", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(1, "Empty Team", createdAt, nil)
		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("Empty Team").
			WillReturnRows(teamRows)

		team, err := repo.GetByName(context.Background(), "Empty Team")

		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "Empty Team", team.Name)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: команда не найдена", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("Non-existent Team").
			WillReturnError(sql.ErrNoRows)

		team, err := repo.GetByName(context.Background(), "Non-existent Team")

		require.Error(t, err)
		assert.Nil(t, team)
		assert.Equal(t, "team not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение команды без updated_at", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(1, "New Team", createdAt, nil)
		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("New Team").
			WillReturnRows(teamRows)

		team, err := repo.GetByName(context.Background(), "New Team")

		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "New Team", team.Name)
		assert.Nil(t, team.UpdatedAt, "updated_at должен быть nil, если не установлен")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
