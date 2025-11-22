package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/DATA-DOG/go-sqlmock"
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
		// Создаем мок БД
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name: "Team Alpha",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
				{UserID: "u2", Username: "user2", IsActive: true},
			},
		}

		// Ожидание 1: Начало транзакции
		mock.ExpectBegin()

		// Ожидание 2: Создание команды (INSERT с ON CONFLICT)
		// При создании новой команды updated_at должен быть NULL
		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, nil) // updated_at = nil для новой команды
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team Alpha", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		// Ожидание 3: Создание первого пользователя через UserRepository.Create()
		// UserRepository.Create() с ID "u1" попытается обновить, но если не найдет - создаст нового
		// Для простоты теста предположим, что пользователь уже существует и будет обновлен
		user1UpdateRows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "user1", 1, true, sqlmock.AnyArg()).
			WillReturnRows(user1UpdateRows)

		// Ожидание 4: Создание второго пользователя
		user2UpdateRows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(2, "user2", 1, true, sqlmock.AnyArg()).
			WillReturnRows(user2UpdateRows)

		// Ожидание 5: Коммит транзакции
		mock.ExpectCommit()

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "Team Alpha", team.Name)
		assert.NotNil(t, team.CreatedAt)
		assert.Nil(t, team.UpdatedAt, "updated_at должен быть nil для новой команды")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное обновление существующей команды", func(t *testing.T) {
		// ON CONFLICT срабатывает, команда обновляется
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		updatedAt := now.Add(1 * time.Hour)
		team := &domain.Team{
			Name: "Existing Team",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
			},
		}

		mock.ExpectBegin()

		// При обновлении существующей команды updated_at устанавливается
		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now.Add(-7*24*time.Hour), updatedAt) // created_at из прошлого, updated_at обновлен
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Existing Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		// Обновление пользователя
		userUpdateRows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "user1", 1, true, sqlmock.AnyArg()).
			WillReturnRows(userUpdateRows)

		mock.ExpectCommit()

		// Выполнение
		err := repo.Create(team)

		// Проверки
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
			Members: []domain.TeamMember{}, // Пустой список участников
		}

		mock.ExpectBegin()

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(2, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Empty Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		// Нет запросов для пользователей, так как список пустой
		mock.ExpectCommit()

		// Выполнение
		err := repo.Create(team)

		// Проверки
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

		mock.ExpectBegin()

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(3, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Large Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		// Три пользователя
		user1Rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "user1", 3, true, sqlmock.AnyArg()).
			WillReturnRows(user1Rows)

		user2Rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(2, "user2", 3, false, sqlmock.AnyArg()).
			WillReturnRows(user2Rows)

		user3Rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), now)
		mock.ExpectQuery("UPDATE users").
			WithArgs(3, "user3", 3, true, sqlmock.AnyArg()).
			WillReturnRows(user3Rows)

		mock.ExpectCommit()

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, 3, team.ID)
		assert.Len(t, team.Members, 3)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать команду", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		team := &domain.Team{
			Name:    "Team",
			Members: []domain.TeamMember{},
		}

		mock.ExpectBegin()

		expectedError := errors.New("database error")
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team", sqlmock.AnyArg()).
			WillReturnError(expectedError)

		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать пользователя", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name: "Team",
			Members: []domain.TeamMember{
				{UserID: "u1", Username: "user1", IsActive: true},
			},
		}

		mock.ExpectBegin()

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		// UPDATE не находит пользователя (sql.ErrNoRows), затем вызывается INSERT
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "user1", 1, true, sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		// Ошибка при INSERT нового пользователя
		expectedError := errors.New("user creation failed")
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("user1", 1, true, sqlmock.AnyArg()).
			WillReturnError(expectedError)

		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		now := time.Now()
		team := &domain.Team{
			Name:    "Team",
			Members: []domain.TeamMember{},
		}

		mock.ExpectBegin()

		teamRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1, now, nil)
		mock.ExpectQuery("INSERT INTO teams").
			WithArgs("Team", sqlmock.AnyArg()).
			WillReturnRows(teamRows)

		expectedError := errors.New("commit failed")
		mock.ExpectCommit().WillReturnError(expectedError)

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось начать транзакцию", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		team := &domain.Team{
			Name:    "Team",
			Members: []domain.TeamMember{},
		}

		expectedError := errors.New("connection failed")
		mock.ExpectBegin().WillReturnError(expectedError)

		// Выполнение
		err := repo.Create(team)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestTeamRepository_GetByName - тест для метода GetByName()
func TestTeamRepository_GetByName(t *testing.T) {
	t.Run("успешное получение команды с участниками", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		// Ожидание 1: Получение команды
		teamRows := sqlmock.NewRows([]string{"id", "name", "created_at", "updated_at"}).
			AddRow(1, "Team Alpha", createdAt, updatedAt)
		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("Team Alpha").
			WillReturnRows(teamRows)

		// Ожидание 2: Получение участников через UserRepository.GetByTeamID()
		userRows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"}).
			AddRow(1, "user1", 1, "Team Alpha", true, createdAt, nil).
			AddRow(2, "user2", 1, "Team Alpha", false, createdAt, nil)
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(userRows)

		// Выполнение
		team, err := repo.GetByName("Team Alpha")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, 1, team.ID)
		assert.Equal(t, "Team Alpha", team.Name)
		assert.Len(t, team.Members, 2)
		assert.Equal(t, "u1", team.Members[0].UserID)
		assert.Equal(t, "user1", team.Members[0].Username)
		assert.True(t, team.Members[0].IsActive)
		assert.Equal(t, "u2", team.Members[1].UserID)
		assert.False(t, team.Members[1].IsActive)
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

		// Нет участников
		userRows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"})
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(userRows)

		// Выполнение
		team, err := repo.GetByName("Empty Team")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Equal(t, 1, team.ID)
		assert.Len(t, team.Members, 0)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: команда не найдена", func(t *testing.T) {
		repo, mock := setupTeamRepo(t)

		mock.ExpectQuery("SELECT id, name, created_at, updated_at").
			WithArgs("Non-existent Team").
			WillReturnError(sql.ErrNoRows)

		// Выполнение
		team, err := repo.GetByName("Non-existent Team")

		// Проверки
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

		userRows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"})
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(userRows)

		// Выполнение
		team, err := repo.GetByName("New Team")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, team)
		assert.Nil(t, team.UpdatedAt, "updated_at должен быть nil, если не установлен")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

