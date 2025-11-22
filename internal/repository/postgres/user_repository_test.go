package postgres

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupUserRepo создает мок БД и репозиторий для User
func setupUserRepo(t *testing.T) (*userRepository, sqlmock.Sqlmock) {
	db, mock := setupMockDB(t)
	return NewUserRepository(db), mock
}

// TestUserRepository_Create - тест для метода Create()
// Этот метод реализует upsert-логику: обновляет пользователя, если передан ID, иначе создает нового
func TestUserRepository_Create(t *testing.T) {
	t.Run("успешное создание нового пользователя", func(t *testing.T) {
		// Создаем мок БД
		repo, mock := setupUserRepo(t)

		now := time.Now()
		user := &domain.User{
			ID:       "", // Пустой ID означает создание нового пользователя
			Username: "john_doe",
			TeamID:   1,
			IsActive: true,
		}

		// Ожидание INSERT запроса
		rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(5, now, nil) // id=5, updated_at=NULL
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("john_doe", 1, true, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Выполнение
		err := repo.Create(user)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, "u5", user.ID, "ID должен быть сконвертирован в строковый формат")
		assert.NotNil(t, user.CreatedAt)
		assert.Nil(t, user.UpdatedAt, "updated_at должен быть nil при создании")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное обновление существующего пользователя", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		now := time.Now()
		updatedAt := now.Add(1 * time.Hour)
		user := &domain.User{
			ID:       "u1", // Указан ID - значит обновление
			Username: "john_updated",
			TeamID:   2,
			IsActive: false,
		}

		// Ожидание UPDATE запроса
		rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), updatedAt) // created_at из прошлого, updated_at обновлен
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "john_updated", 2, false, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Выполнение
		err := repo.Create(user)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, "u1", user.ID, "ID должен остаться прежним")
		assert.NotNil(t, user.UpdatedAt, "updated_at должен быть установлен при обновлении")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("создание нового пользователя, если UPDATE не нашел запись", func(t *testing.T) {
		// Если передан ID, но пользователь не найден, создается новый
		repo, mock := setupUserRepo(t)

		now := time.Now()
		user := &domain.User{
			ID:       "u999", // ID указан, но пользователь не существует
			Username: "new_user",
			TeamID:   1,
			IsActive: true,
		}

		// UPDATE не находит запись (sql.ErrNoRows)
		mock.ExpectQuery("UPDATE users").
			WithArgs(999, "new_user", 1, true, sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		// После неудачного UPDATE выполняется INSERT
		rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(10, now, nil)
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("new_user", 1, true, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Выполнение
		err := repo.Create(user)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, "u10", user.ID, "должен быть присвоен новый ID")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID при обновлении", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		user := &domain.User{
			ID:       "invalid-id", // Невалидный формат ID
			Username: "user",
			TeamID:   1,
			IsActive: true,
		}

		// stringIDToInt вернет ошибку, UPDATE не будет вызван
		// Вместо этого будет вызван INSERT
		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(11, now, nil)
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("user", 1, true, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Выполнение
		err := repo.Create(user)

		// Проверки - невалидный ID игнорируется, создается новый пользователь
		require.NoError(t, err)
		assert.Equal(t, "u11", user.ID)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать пользователя", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		user := &domain.User{
			ID:       "",
			Username: "user",
			TeamID:   1,
			IsActive: true,
		}

		expectedError := errors.New("database error")
		mock.ExpectQuery("INSERT INTO users").
			WithArgs("user", 1, true, sqlmock.AnyArg()).
			WillReturnError(expectedError)

		// Выполнение
		err := repo.Create(user)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestUserRepository_Update - тест для метода Update()
func TestUserRepository_Update(t *testing.T) {
	t.Run("успешное обновление пользователя", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		now := time.Now()
		updatedAt := now.Add(1 * time.Hour)
		user := &domain.User{
			ID:       "u1",
			Username: "updated_name",
			TeamID:   2,
			IsActive: false,
		}

		rows := sqlmock.NewRows([]string{"created_at", "updated_at"}).
			AddRow(now.Add(-24*time.Hour), updatedAt)
		mock.ExpectQuery("UPDATE users").
			WithArgs(1, "updated_name", 2, false, sqlmock.AnyArg()).
			WillReturnRows(rows)

		// Выполнение
		err := repo.Update(user)

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, user.UpdatedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: пользователь не найден", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		user := &domain.User{
			ID:       "u999",
			Username: "user",
			TeamID:   1,
			IsActive: true,
		}

		mock.ExpectQuery("UPDATE users").
			WithArgs(999, "user", 1, true, sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		// Выполнение
		err := repo.Update(user)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "user not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		user := &domain.User{
			ID:       "invalid",
			Username: "user",
			TeamID:   1,
			IsActive: true,
		}

		// Выполнение
		err := repo.Update(user)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid user ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestUserRepository_GetByID - тест для метода GetByID()
func TestUserRepository_GetByID(t *testing.T) {
	t.Run("успешное получение пользователя", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"}).
			AddRow(1, "john_doe", 1, "Team A", true, createdAt, updatedAt)
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		user, err := repo.GetByID("u1")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "u1", user.ID)
		assert.Equal(t, "john_doe", user.Username)
		assert.Equal(t, 1, user.TeamID)
		assert.Equal(t, "Team A", user.TeamName)
		assert.True(t, user.IsActive)
		assert.NotNil(t, user.UpdatedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пользователя без updated_at", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"}).
			AddRow(1, "john_doe", 1, "Team A", true, createdAt, nil)
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		user, err := repo.GetByID("u1")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Nil(t, user.UpdatedAt, "updated_at должен быть nil, если не установлен")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: пользователь не найден", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(999).
			WillReturnError(sql.ErrNoRows)

		// Выполнение
		user, err := repo.GetByID("u999")

		// Проверки
		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "user not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		// Выполнение
		user, err := repo.GetByID("invalid")

		// Проверки
		require.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "invalid user ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestUserRepository_GetActiveByTeamID - тест для метода GetActiveByTeamID()
func TestUserRepository_GetActiveByTeamID(t *testing.T) {
	t.Run("успешное получение активных пользователей команды", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"}).
			AddRow(1, "user1", 1, "Team A", true, createdAt, nil).
			AddRow(2, "user2", 1, "Team A", true, createdAt, nil)
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		users, err := repo.GetActiveByTeamID(1)

		// Проверки
		require.NoError(t, err)
		require.Len(t, users, 2)
		assert.Equal(t, "u1", users[0].ID)
		assert.Equal(t, "u2", users[1].ID)
		assert.True(t, users[0].IsActive)
		assert.True(t, users[1].IsActive)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пустого списка", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"})
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		users, err := repo.GetActiveByTeamID(1)

		// Проверки
		require.NoError(t, err)
		// При отсутствии результатов возвращается nil слайс
		assert.Nil(t, users)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestUserRepository_GetByTeamID - тест для метода GetByTeamID()
func TestUserRepository_GetByTeamID(t *testing.T) {
	t.Run("успешное получение всех пользователей команды", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"}).
			AddRow(1, "user1", 1, "Team A", true, createdAt, nil).
			AddRow(2, "user2", 1, "Team A", false, createdAt, nil).
			AddRow(3, "user3", 1, "Team A", true, createdAt, nil)
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		users, err := repo.GetByTeamID(1)

		// Проверки
		require.NoError(t, err)
		require.Len(t, users, 3)
		assert.Equal(t, "u1", users[0].ID)
		assert.True(t, users[0].IsActive)
		assert.False(t, users[1].IsActive) // Второй пользователь неактивен
		assert.True(t, users[2].IsActive)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пустого списка", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		rows := sqlmock.NewRows([]string{"id", "name", "team_id", "name", "is_active", "created_at", "updated_at"})
		mock.ExpectQuery("SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at").
			WithArgs(1).
			WillReturnRows(rows)

		// Выполнение
		users, err := repo.GetByTeamID(1)

		// Проверки
		require.NoError(t, err)
		// При отсутствии результатов возвращается nil слайс
		assert.Nil(t, users)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestUserRepository_SetIsActive - тест для метода SetIsActive()
func TestUserRepository_SetIsActive(t *testing.T) {
	t.Run("успешное изменение статуса активности", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		// Ожидание UPDATE запроса
		mock.ExpectExec("UPDATE users").
			WithArgs(1, false, sqlmock.AnyArg()).     // userID=1, isActive=false
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 строка обновлена

		// Выполнение
		err := repo.SetIsActive("u1", false)

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: пользователь не найден", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		// UPDATE не находит строку (0 строк затронуто)
		mock.ExpectExec("UPDATE users").
			WithArgs(999, true, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(0, 0))

		// Выполнение
		err := repo.SetIsActive("u999", true)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "user not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID", func(t *testing.T) {
		repo, mock := setupUserRepo(t)

		// Выполнение
		err := repo.SetIsActive("invalid", true)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid user ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
