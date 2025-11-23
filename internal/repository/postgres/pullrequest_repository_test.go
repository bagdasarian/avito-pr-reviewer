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

// setupMockDB создает мок базы данных для тестов
// Автоматически закрывает соединение при завершении теста
func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err, "не удалось создать мок БД")
	t.Cleanup(func() { db.Close() })
	return db, mock
}

// setupPRRepo создает мок БД и репозиторий для PullRequest
func setupPRRepo(t *testing.T) (*pullRequestRepository, sqlmock.Sqlmock) {
	db, mock := setupMockDB(t)
	return NewPullRequestRepository(db), mock
}

// TestPullRequestRepository_Create - основной тест для метода Create()
// Этот метод выполняет сложную транзакцию с несколькими запросами:
// 1. Начало транзакции (Begin)
// 2. Получение ID статуса из справочника
// 3. Проверка существования автора
// 4. Создание PR
// 5. Для каждого ревьювера: проверка существования и добавление связи
// 6. Коммит транзакции
func TestPullRequestRepository_Create(t *testing.T) {
	t.Run("успешное создание PR с ревьюверами", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "u3"},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prID := 1001
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(prID, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(prID, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(prID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(prID, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(prID, 3, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), pr)

		require.NoError(t, err, "Create() не должна возвращать ошибку")

		assert.Equal(t, "pr-1001", pr.ID, "ID должен остаться в исходном формате")

		assert.NotNil(t, pr.CreatedAt, "CreatedAt должен быть установлен")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "не все ожидания SQL-запросов были выполнены")
	})

	t.Run("успешное создание PR без ревьюверов", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(1001).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Create(context.Background(), pr)

		require.NoError(t, err)
		assert.Equal(t, "pr-1001", pr.ID)
		assert.NotNil(t, pr.CreatedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: автор не найден", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u999",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 999, 1, sqlmock.AnyArg()).
			WillReturnError(errors.New("author not found"))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err, "должна быть возвращена ошибка")
		assert.Contains(t, err.Error(), "author", "текст ошибки должен содержать 'author'")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: ревьювер не найден", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "u999"},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(1001).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WillReturnError(errors.New("reviewer not found"))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Equal(t, "reviewer not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID автора", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "invalid-id",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid author ID")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "invalid-reviewer"},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(1001).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: статус не найден", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WillReturnError(sql.ErrNoRows)

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "должна быть возвращена ошибка sql.ErrNoRows")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось начать транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Error(t, err, "должна быть возвращена ошибка начала транзакции")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать PR", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Error(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось добавить ревьювера", func(t *testing.T) {

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2"},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(1001).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnError(errors.New("database error"))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Error(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			ID:                "pr-1001",
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs(1001, "Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		mock.ExpectExec("SELECT setval").
			WithArgs(1001).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Create(context.Background(), pr)

		require.Error(t, err)
		assert.Error(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_ReplaceReviewer - тест для метода ReplaceReviewer()
// Этот метод заменяет одного ревьювера на другого в транзакции
func TestPullRequestRepository_ReplaceReviewer(t *testing.T) {
	t.Run("успешная замена ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Проверка, что новый ревьювер не назначен (exists = false)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 2).
			WillReturnRows(existsRows)

		// UPDATE для замены
		mock.ExpectExec("UPDATE pull_request_reviewers").
			WithArgs(2, 1001, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешная замена ревьювера когда новый уже назначен", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Проверка, что новый ревьювер уже назначен (exists = true)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 2).
			WillReturnRows(existsRows)

		// DELETE старого ревьювера
		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WithArgs(1001, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: старый ревьювер не назначен на PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Проверка, что новый ревьювер не назначен (exists = false)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 2).
			WillReturnRows(existsRows)

		// UPDATE не находит запись (rowsAffected = 0)
		mock.ExpectExec("UPDATE pull_request_reviewers").
			WithArgs(2, 1001, 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.Error(t, err)
		assert.Equal(t, "reviewer is not assigned to this PR", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: старый ревьювер не назначен на PR (когда новый уже назначен)", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Проверка, что новый ревьювер уже назначен (exists = true)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 2).
			WillReturnRows(existsRows)

		// DELETE не находит запись (rowsAffected = 0)
		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WithArgs(1001, 1).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.Error(t, err)
		assert.Equal(t, "reviewer is not assigned to this PR", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.ReplaceReviewer(context.Background(), "invalid", "u1", "u2")

		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID старого ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "invalid", "u2")

		require.Error(t, err)
		assert.Equal(t, "invalid old reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID нового ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "invalid")

		require.Error(t, err)
		assert.Equal(t, "invalid new reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось начать транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")
		repo, mock := setupPRRepo(t)

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.Error(t, err)
		assert.Error(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию (удален - репозитории больше не создают транзакции)", func(t *testing.T) {
		t.Skip("Репозитории больше не создают транзакции, этот тест не актуален")
		repo, mock := setupPRRepo(t)

		mock.ExpectExec("UPDATE pull_request_reviewers").
			WithArgs(2, 1001, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.ReplaceReviewer(context.Background(), "pr-1001", "u1", "u2")

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_UpdateStatus - тест для метода UpdateStatus()
func TestPullRequestRepository_UpdateStatus(t *testing.T) {
	t.Run("успешное обновление статуса", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnRows(updateRows)

		err := repo.UpdateStatus(context.Background(), "pr-1001", domain.StatusMerged, nil)

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное обновление статуса с mergedAt", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mergedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnRows(updateRows)

		err := repo.UpdateStatus(context.Background(), "pr-1001", domain.StatusMerged, &mergedAt)

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(9999, 2, sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		err := repo.UpdateStatus(context.Background(), "pr-9999", domain.StatusMerged, nil)

		require.Error(t, err)
		assert.Equal(t, "pull request not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: статус не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WillReturnError(sql.ErrNoRows)

		err := repo.UpdateStatus(context.Background(), "pr-1001", domain.Status("INVALID"), nil)

		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("идемпотентность: повторное обновление на тот же статус", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 1, sqlmock.AnyArg()).
			WillReturnRows(updateRows)

		err := repo.UpdateStatus(context.Background(), "pr-1001", domain.StatusOpen, nil)
		require.NoError(t, err)

		statusRows2 := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows2)

		updateRows2 := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 1, sqlmock.AnyArg()).
			WillReturnRows(updateRows2)

		err = repo.UpdateStatus(context.Background(), "pr-1001", domain.StatusOpen, nil)

		require.NoError(t, err, "повторное обновление должно быть успешным")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.UpdateStatus(context.Background(), "invalid", domain.StatusOpen, nil)

		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_GetByID - тест для метода GetByID()
func TestPullRequestRepository_GetByID(t *testing.T) {
	t.Run("успешное получение PR с ревьюверами", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
		updatedAt := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

		prRows := sqlmock.NewRows([]string{"id", "title", "id", "name", "created_at", "updated_at"}).
			AddRow(1001, "Test PR", 1, "MERGED", createdAt, updatedAt)
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at").
			WithArgs(1001).
			WillReturnRows(prRows)

		reviewerRows := sqlmock.NewRows([]string{"id"}).
			AddRow(2).
			AddRow(3)
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		pr, err := repo.GetByID(context.Background(), "pr-1001")

		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-1001", pr.ID)
		assert.Equal(t, "Test PR", pr.Title)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, domain.StatusMerged, pr.Status)
		assert.Equal(t, []string{"u2", "u3"}, pr.AssignedReviewers)
		assert.NotNil(t, pr.CreatedAt)
		assert.NotNil(t, pr.MergedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение PR без ревьюверов", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		createdAt := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)

		prRows := sqlmock.NewRows([]string{"id", "title", "id", "name", "created_at", "updated_at"}).
			AddRow(1001, "Test PR", 1, "OPEN", createdAt, nil)
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at").
			WithArgs(1001).
			WillReturnRows(prRows)

		reviewerRows := sqlmock.NewRows([]string{"id"})
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		pr, err := repo.GetByID(context.Background(), "pr-1001")

		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-1001", pr.ID)
		if pr.AssignedReviewers != nil {
			assert.Len(t, pr.AssignedReviewers, 0)
		}
		assert.Nil(t, pr.MergedAt)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at").
			WillReturnError(sql.ErrNoRows)

		pr, err := repo.GetByID(context.Background(), "pr-9999")

		require.Error(t, err)
		assert.Nil(t, pr)
		assert.Equal(t, "pull request not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		pr, err := repo.GetByID(context.Background(), "invalid")

		require.Error(t, err)
		assert.Nil(t, pr)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_AddReviewer - тест для метода AddReviewer()
func TestPullRequestRepository_AddReviewer(t *testing.T) {
	t.Run("успешное добавление ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.AddReviewer(context.Background(), "pr-1001", "u2")

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.AddReviewer(context.Background(), "invalid", "u2")

		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_RemoveReviewer - тест для метода RemoveReviewer()
func TestPullRequestRepository_RemoveReviewer(t *testing.T) {
	t.Run("успешное удаление ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.RemoveReviewer(context.Background(), "pr-1001", "u2")

		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: ревьювер не назначен на PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.RemoveReviewer(context.Background(), "pr-1001", "u999")

		require.Error(t, err)
		assert.Equal(t, "reviewer not assigned to this PR", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.RemoveReviewer(context.Background(), "invalid", "u2")

		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		err := repo.RemoveReviewer(context.Background(), "pr-1001", "invalid")

		require.Error(t, err)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_GetReviewersByPRID - тест для метода GetReviewersByPRID()
func TestPullRequestRepository_GetReviewersByPRID(t *testing.T) {
	t.Run("успешное получение списка ревьюверов", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		reviewerRows := sqlmock.NewRows([]string{"id"}).
			AddRow(2).
			AddRow(3).
			AddRow(4)
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		reviewers, err := repo.GetReviewersByPRID(context.Background(), "pr-1001")

		require.NoError(t, err)
		assert.Equal(t, []string{"u2", "u3", "u4"}, reviewers)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пустого списка ревьюверов", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		reviewerRows := sqlmock.NewRows([]string{"id"})
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		reviewers, err := repo.GetReviewersByPRID(context.Background(), "pr-1001")

		require.NoError(t, err)
		assert.Nil(t, reviewers)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		reviewers, err := repo.GetReviewersByPRID(context.Background(), "invalid")

		require.Error(t, err)
		assert.Nil(t, reviewers)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_GetPRsByReviewerID - тест для метода GetPRsByReviewerID()
func TestPullRequestRepository_GetPRsByReviewerID(t *testing.T) {
	t.Run("успешное получение списка PR для ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		prRows := sqlmock.NewRows([]string{"id", "title", "id", "name"}).
			AddRow(1001, "PR 1", 1, "OPEN").
			AddRow(1002, "PR 2", 2, "MERGED").
			AddRow(1003, "PR 3", 3, "OPEN")
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name").
			WithArgs(2).
			WillReturnRows(prRows)

		prs, err := repo.GetPRsByReviewerID(context.Background(), "u2")

		require.NoError(t, err)
		require.Len(t, prs, 3)
		assert.Equal(t, "pr-1001", prs[0].ID)
		assert.Equal(t, "PR 1", prs[0].Title)
		assert.Equal(t, "u1", prs[0].AuthorID)
		assert.Equal(t, domain.StatusOpen, prs[0].Status)
		assert.Equal(t, "pr-1002", prs[1].ID)
		assert.Equal(t, domain.StatusMerged, prs[1].Status)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пустого списка PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		prRows := sqlmock.NewRows([]string{"id", "title", "id", "name"})
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name").
			WithArgs(2).
			WillReturnRows(prRows)

		prs, err := repo.GetPRsByReviewerID(context.Background(), "u2")

		require.NoError(t, err)
		assert.Nil(t, prs)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		prs, err := repo.GetPRsByReviewerID(context.Background(), "invalid")

		require.Error(t, err)
		assert.Nil(t, prs)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
