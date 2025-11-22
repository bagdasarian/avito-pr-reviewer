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
		// Создаем репозиторий с мок-БД
		repo, mock := setupPRRepo(t)

		// Шаг 3: Подготавливаем тестовые данные
		// Создаем PR с автором и двумя ревьюверами
		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1", // строковый ID автора (будет конвертирован в 1)
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "u3"}, // два ревьювера
		}

		// Шаг 4: Настраиваем ожидания для SQL-запросов
		// Важно: порядок ожиданий должен точно соответствовать порядку вызовов в коде!

		// Ожидание 1: Начало транзакции
		// Begin() вызывается первым в методе Create()
		// sqlmock.NewBegin() создает мок-транзакцию
		mock.ExpectBegin()

		// Ожидание 2: Получение ID статуса "OPEN" из справочника statuses
		// В коде: tx.QueryRow("SELECT id FROM statuses WHERE name = $1", string(pr.Status))
		// Мы ожидаем запрос с параметром "OPEN" и возвращаем ID = 1
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN"). // Проверяем, что передается именно "OPEN"
			WillReturnRows(statusRows)

		// Ожидание 3: Проверка существования автора
		// В коде: tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", authorDBID)
		// authorDBID = 1 (конвертировано из "u1")
		// EXISTS возвращает boolean, поэтому используем AddRow(true)
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1). // Проверяем, что запрашивается пользователь с ID = 1
			WillReturnRows(authorExistsRows)

		// Ожидание 4: Создание PR
		// В коде: INSERT INTO pull_requests ... RETURNING id, created_at, updated_at
		// QueryRow возвращает три значения: id, created_at, updated_at
		// updated_at должен быть NULL при создании
		prID := 1001
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(prID, now, nil) // updated_at = nil (NULL в БД)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()). // title, author_id, status_id, created_at
			WillReturnRows(prRows)

		// Ожидание 5: Проверка существования первого ревьювера (u2 -> ID = 2)
		reviewer1ExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(2). // Проверяем ревьювера с ID = 2
			WillReturnRows(reviewer1ExistsRows)

		// Ожидание 6: Добавление первого ревьювера
		// В коде: tx.Exec("INSERT INTO pull_request_reviewers ...")
		// Exec не возвращает строки, только количество затронутых строк
		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(prID, 2, sqlmock.AnyArg()).      // pull_request_id, reviewer_id, created_at
			WillReturnResult(sqlmock.NewResult(1, 1)) // 1 строка вставлена

		// Ожидание 7: Проверка существования второго ревьювера (u3 -> ID = 3)
		reviewer2ExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(3). // Проверяем ревьювера с ID = 3
			WillReturnRows(reviewer2ExistsRows)

		// Ожидание 8: Добавление второго ревьювера
		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(prID, 3, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Ожидание 9: Коммит транзакции
		// Если Commit() не будет вызван, тест упадет с ошибкой
		mock.ExpectCommit()

		// Шаг 5: Выполняем тестируемый метод
		err := repo.Create(pr)

		// Шаг 6: Проверяем результаты
		// require.NoError проверяет, что ошибки нет, и останавливает тест при ошибке
		require.NoError(t, err, "Create() не должна возвращать ошибку")

		// Проверяем, что PR получил правильный ID (конвертированный обратно в строку)
		assert.Equal(t, "pr-1001", pr.ID, "ID должен быть конвертирован в строковый формат")

		// Проверяем, что CreatedAt установлен
		assert.NotNil(t, pr.CreatedAt, "CreatedAt должен быть установлен")

		// Проверяем, что все ожидания выполнены
		// Это важно: если какой-то запрос не был выполнен или был выполнен лишний, тест упадет
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err, "не все ожидания SQL-запросов были выполнены")
	})

	t.Run("успешное создание PR без ревьюверов", func(t *testing.T) {
		// Тест для случая, когда PR создается без назначенных ревьюверов
		// Это упрощенный сценарий - не нужно проверять и добавлять ревьюверов

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "PR without reviewers",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{}, // Пустой список ревьюверов
		}

		// Настройка ожиданий (без запросов для ревьюверов)
		mock.ExpectBegin()

		// Получение ID статуса
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Проверка автора
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// Создание PR
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("PR without reviewers", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		// Коммит (без запросов для ревьюверов, так как список пустой)
		mock.ExpectCommit()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, "pr-1001", pr.ID)
		assert.NotNil(t, pr.CreatedAt)

		// Проверка, что все ожидания выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: автор не найден", func(t *testing.T) {
		// Тест проверяет обработку случая, когда автор PR не существует в БД
		// В этом случае метод должен вернуть ошибку "author not found"

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u999", // Несуществующий автор
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Получение статуса (успешно)
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Проверка автора - возвращаем false (автор не существует)
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(999). // ID = 999 (конвертировано из "u999")
			WillReturnRows(authorExistsRows)

		// Ожидаем откат транзакции (Rollback вызывается через defer)
		// Когда метод возвращает ошибку, defer tx.Rollback() выполняется автоматически
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err, "должна быть возвращена ошибка")
		assert.Equal(t, "author not found", err.Error(), "текст ошибки должен быть 'author not found'")

		// Проверка, что все ожидания выполнены
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: ревьювер не найден", func(t *testing.T) {
		// Тест проверяет обработку случая, когда один из ревьюверов не существует
		// Метод должен вернуть ошибку "reviewer not found" и откатить транзакцию

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "u999"}, // Второй ревьювер не существует
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Автор существует
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// PR создается успешно
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		// Первый ревьювер существует
		reviewer1ExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(2).
			WillReturnRows(reviewer1ExistsRows)

		// Первый ревьювер добавляется
		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Второй ревьювер НЕ существует (возвращаем false)
		reviewer2ExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(999). // Несуществующий ревьювер
			WillReturnRows(reviewer2ExistsRows)

		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "reviewer not found", err.Error())

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID автора", func(t *testing.T) {
		// Тест проверяет обработку невалидного формата ID автора
		// stringIDToInt("invalid") вернет ошибку, и метод должен вернуть "invalid author ID"

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "invalid-id", // Невалидный формат ID
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус получаем успешно
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// stringIDToInt("invalid-id") вернет ошибку
		// Поэтому запрос к БД для проверки автора не будет выполнен
		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid author ID", err.Error())

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		// Тест проверяет обработку невалидного формата ID ревьювера

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2", "invalid-reviewer"}, // Второй ревьювер с невалидным ID
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Автор
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// PR
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		// Первый ревьювер успешно проверяется и добавляется
		reviewer1ExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(2).
			WillReturnRows(reviewer1ExistsRows)

		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Второй ревьювер имеет невалидный ID - stringIDToInt вернет ошибку
		// Запрос к БД не будет выполнен
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: статус не найден", func(t *testing.T) {
		// Тест проверяет обработку случая, когда статус не существует в справочнике
		// QueryRow вернет sql.ErrNoRows, который будет обработан как ошибка

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Запрос статуса возвращает ошибку (статус не найден)
		// sqlmock.ErrRows возвращает sql.ErrNoRows
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnError(sql.ErrNoRows)

		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows, "должна быть возвращена ошибка sql.ErrNoRows")

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось начать транзакцию", func(t *testing.T) {
		// Тест проверяет обработку ошибки при начале транзакции
		// Это может произойти, например, при проблемах с подключением к БД

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий: Begin() возвращает ошибку
		expectedError := errors.New("connection failed")
		mock.ExpectBegin().WillReturnError(expectedError)

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err, "должна быть возвращена ошибка начала транзакции")

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось создать PR", func(t *testing.T) {
		// Тест проверяет обработку ошибки при вставке PR в БД
		// Это может произойти, например, при нарушении ограничений БД

		repo, mock := setupPRRepo(t)

		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус успешно получен
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Автор существует
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// Ошибка при вставке PR (например, нарушение уникальности)
		expectedError := errors.New("duplicate key value")
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnError(expectedError)

		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось добавить ревьювера", func(t *testing.T) {
		// Тест проверяет обработку ошибки при вставке связи PR-ревьювер
		// Это может произойти, например, при нарушении уникальности (ревьювер уже назначен)

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{"u2"},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Автор
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// PR создан успешно
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		// Ревьювер существует
		reviewerExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(2).
			WillReturnRows(reviewerExistsRows)

		// Ошибка при добавлении ревьювера (например, дубликат)
		expectedError := errors.New("duplicate key value violates unique constraint")
		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnError(expectedError)

		// Ожидаем откат транзакции
		mock.ExpectRollback()

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию", func(t *testing.T) {
		// Тест проверяет обработку ошибки при коммите транзакции
		// Это может произойти, например, при проблемах с БД во время коммита

		repo, mock := setupPRRepo(t)

		now := time.Now()
		pr := &domain.PullRequest{
			Title:             "Test PR",
			AuthorID:          "u1",
			Status:            domain.StatusOpen,
			AssignedReviewers: []string{},
		}

		// Настройка ожиданий
		mock.ExpectBegin()

		// Статус
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		// Автор
		authorExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1).
			WillReturnRows(authorExistsRows)

		// PR
		prRows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(1001, now, nil)
		mock.ExpectQuery("INSERT INTO pull_requests").
			WithArgs("Test PR", 1, 1, sqlmock.AnyArg()).
			WillReturnRows(prRows)

		// Ошибка при коммите
		expectedError := errors.New("commit failed")
		mock.ExpectCommit().WillReturnError(expectedError)

		// Выполнение
		err := repo.Create(pr)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		// Проверка ожиданий
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_ReplaceReviewer - тест для метода ReplaceReviewer()
// Этот метод заменяет одного ревьювера на другого в транзакции
func TestPullRequestRepository_ReplaceReviewer(t *testing.T) {
	t.Run("успешная замена ревьювера", func(t *testing.T) {
		// Создаем мок БД
		repo, mock := setupPRRepo(t)

		// Настройка ожиданий
		mock.ExpectBegin()

		// Проверка существования старого ревьювера (должен быть назначен)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 1). // prID=1001, oldReviewerID=1
			WillReturnRows(existsRows)

		// Замена ревьювера (UPDATE)
		mock.ExpectExec("UPDATE pull_request_reviewers").
			WithArgs(2, 1001, 1). // newReviewerID=2, prID=1001, oldReviewerID=1
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		// Выполнение
		err := repo.ReplaceReviewer("pr-1001", "u1", "u2")

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: старый ревьювер не назначен на PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		// Старый ревьювер не назначен (возвращаем false)
		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 1).
			WillReturnRows(existsRows)

		mock.ExpectRollback()

		// Выполнение
		err := repo.ReplaceReviewer("pr-1001", "u1", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "reviewer is not assigned to this PR", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение с невалидным ID PR
		err := repo.ReplaceReviewer("invalid", "u1", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID старого ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение с невалидным ID старого ревьювера
		err := repo.ReplaceReviewer("pr-1001", "invalid", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid old reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID нового ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение с невалидным ID нового ревьювера
		err := repo.ReplaceReviewer("pr-1001", "u1", "invalid")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid new reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось начать транзакцию", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		expectedError := errors.New("connection failed")
		mock.ExpectBegin().WillReturnError(expectedError)

		// Выполнение
		err := repo.ReplaceReviewer("pr-1001", "u1", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: не удалось закоммитить транзакцию", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		existsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001, 1).
			WillReturnRows(existsRows)

		mock.ExpectExec("UPDATE pull_request_reviewers").
			WithArgs(2, 1001, 1).
			WillReturnResult(sqlmock.NewResult(1, 1))

		expectedError := errors.New("commit failed")
		mock.ExpectCommit().WillReturnError(expectedError)

		// Выполнение
		err := repo.ReplaceReviewer("pr-1001", "u1", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, expectedError, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_UpdateStatus - тест для метода UpdateStatus()
func TestPullRequestRepository_UpdateStatus(t *testing.T) {
	t.Run("успешное обновление статуса", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		// Получение ID статуса
		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		// Обновление статуса PR
		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 2, sqlmock.AnyArg()). // prID, statusID, updated_at
			WillReturnRows(updateRows)

		mock.ExpectCommit()

		// Выполнение
		err := repo.UpdateStatus("pr-1001", domain.StatusMerged, nil)

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное обновление статуса с mergedAt", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mergedAt := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
		mock.ExpectBegin()

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 2, mergedAt). // Используется переданный mergedAt
			WillReturnRows(updateRows)

		mock.ExpectCommit()

		// Выполнение с указанным mergedAt
		err := repo.UpdateStatus("pr-1001", domain.StatusMerged, &mergedAt)

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(2)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("MERGED").
			WillReturnRows(statusRows)

		// UPDATE не находит строку
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(9999, 2, sqlmock.AnyArg()).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()

		// Выполнение
		err := repo.UpdateStatus("pr-9999", domain.StatusMerged, nil)

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "pull request not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: статус не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		// Статус не найден
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("INVALID").
			WillReturnError(sql.ErrNoRows)

		mock.ExpectRollback()

		// Выполнение
		err := repo.UpdateStatus("pr-1001", domain.Status("INVALID"), nil)

		// Проверки
		require.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("идемпотентность: повторное обновление на тот же статус", func(t *testing.T) {
		// Идемпотентность означает, что повторный вызов с теми же параметрами
		// должен работать корректно и не вызывать ошибок
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		statusRows := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows)

		updateRows := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 1, sqlmock.AnyArg()).
			WillReturnRows(updateRows)

		mock.ExpectCommit()

		// Выполнение - первый раз
		err := repo.UpdateStatus("pr-1001", domain.StatusOpen, nil)
		require.NoError(t, err)

		// Повторное выполнение с теми же параметрами
		// Нужно создать новые объекты rows для второго вызова
		mock.ExpectBegin()
		statusRows2 := sqlmock.NewRows([]string{"id"}).AddRow(1)
		mock.ExpectQuery("SELECT id FROM statuses WHERE name = \\$1").
			WithArgs("OPEN").
			WillReturnRows(statusRows2)
		updateRows2 := sqlmock.NewRows([]string{"id"}).AddRow(1001)
		mock.ExpectQuery("UPDATE pull_requests").
			WithArgs(1001, 1, sqlmock.AnyArg()).
			WillReturnRows(updateRows2)
		mock.ExpectCommit()

		err = repo.UpdateStatus("pr-1001", domain.StatusOpen, nil)

		// Проверки
		require.NoError(t, err, "повторное обновление должно быть успешным")

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение
		err := repo.UpdateStatus("invalid", domain.StatusOpen, nil)

		// Проверки
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

		// Ожидание 1: Получение PR
		prRows := sqlmock.NewRows([]string{"id", "title", "id", "name", "created_at", "updated_at"}).
			AddRow(1001, "Test PR", 1, "MERGED", createdAt, updatedAt)
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at").
			WithArgs(1001).
			WillReturnRows(prRows)

		// Ожидание 2: Получение ревьюверов (вызывается GetReviewersByPRID внутри)
		reviewerRows := sqlmock.NewRows([]string{"id"}).
			AddRow(2).
			AddRow(3)
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		// Выполнение
		pr, err := repo.GetByID("pr-1001")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-1001", pr.ID)
		assert.Equal(t, "Test PR", pr.Title)
		assert.Equal(t, "u1", pr.AuthorID)
		assert.Equal(t, domain.StatusMerged, pr.Status)
		assert.Equal(t, []string{"u2", "u3"}, pr.AssignedReviewers)
		assert.NotNil(t, pr.CreatedAt)
		assert.NotNil(t, pr.MergedAt) // Для MERGED статуса mergedAt должен быть установлен

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

		// Нет ревьюверов
		reviewerRows := sqlmock.NewRows([]string{"id"})
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(reviewerRows)

		// Выполнение
		pr, err := repo.GetByID("pr-1001")

		// Проверки
		require.NoError(t, err)
		assert.NotNil(t, pr)
		assert.Equal(t, "pr-1001", pr.ID)
		// При отсутствии ревьюверов может вернуться nil или пустой слайс
		if pr.AssignedReviewers != nil {
			assert.Len(t, pr.AssignedReviewers, 0)
		}
		assert.Nil(t, pr.MergedAt) // Для OPEN статуса mergedAt должен быть nil

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at").
			WithArgs(9999).
			WillReturnError(sql.ErrNoRows)

		// Выполнение
		pr, err := repo.GetByID("pr-9999")

		// Проверки
		require.Error(t, err)
		assert.Nil(t, pr)
		assert.Equal(t, "pull request not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Выполнение
		pr, err := repo.GetByID("invalid")

		// Проверки
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

		mock.ExpectBegin()

		// Проверка существования PR
		prExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001).
			WillReturnRows(prExistsRows)

		// Проверка существования ревьювера
		reviewerExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(2).
			WillReturnRows(reviewerExistsRows)

		// Добавление ревьювера
		mock.ExpectExec("INSERT INTO pull_request_reviewers").
			WithArgs(1001, 2, sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		// Выполнение
		err := repo.AddReviewer("pr-1001", "u2")

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: PR не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		prExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(9999).
			WillReturnRows(prExistsRows)

		mock.ExpectRollback()

		// Выполнение
		err := repo.AddReviewer("pr-9999", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "pull request not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: ревьювер не найден", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		prExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(true)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(1001).
			WillReturnRows(prExistsRows)

		reviewerExistsRows := sqlmock.NewRows([]string{"exists"}).AddRow(false)
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(999).
			WillReturnRows(reviewerExistsRows)

		mock.ExpectRollback()

		// Выполнение
		err := repo.AddReviewer("pr-1001", "u999")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "reviewer not found", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение
		err := repo.AddReviewer("invalid", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение
		err := repo.AddReviewer("pr-1001", "invalid")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}

// TestPullRequestRepository_RemoveReviewer - тест для метода RemoveReviewer()
func TestPullRequestRepository_RemoveReviewer(t *testing.T) {
	t.Run("успешное удаление ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		// Удаление ревьювера
		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WithArgs(1001, 2).
			WillReturnResult(sqlmock.NewResult(0, 1)) // 1 строка удалена

		mock.ExpectCommit()

		// Выполнение
		err := repo.RemoveReviewer("pr-1001", "u2")

		// Проверки
		require.NoError(t, err)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: ревьювер не назначен на PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()

		// DELETE не находит строку (0 строк затронуто)
		mock.ExpectExec("DELETE FROM pull_request_reviewers").
			WithArgs(1001, 999).
			WillReturnResult(sqlmock.NewResult(0, 0))

		mock.ExpectRollback()

		// Выполнение
		err := repo.RemoveReviewer("pr-1001", "u999")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "reviewer not assigned to this PR", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение
		err := repo.RemoveReviewer("invalid", "u2")

		// Проверки
		require.Error(t, err)
		assert.Equal(t, "invalid pull request ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		mock.ExpectBegin()
		mock.ExpectRollback()

		// Выполнение
		err := repo.RemoveReviewer("pr-1001", "invalid")

		// Проверки
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

		// Ожидание запроса с несколькими ревьюверами
		rows := sqlmock.NewRows([]string{"id"}).
			AddRow(2).
			AddRow(3).
			AddRow(4)
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(rows)

		// Выполнение
		reviewers, err := repo.GetReviewersByPRID("pr-1001")

		// Проверки
		require.NoError(t, err)
		assert.Equal(t, []string{"u2", "u3", "u4"}, reviewers)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("успешное получение пустого списка ревьюверов", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Нет ревьюверов
		rows := sqlmock.NewRows([]string{"id"})
		mock.ExpectQuery("SELECT u.id").
			WithArgs(1001).
			WillReturnRows(rows)

		// Выполнение
		reviewers, err := repo.GetReviewersByPRID("pr-1001")

		// Проверки
		require.NoError(t, err)
		// При отсутствии результатов возвращается nil
		assert.Nil(t, reviewers)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID PR", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Выполнение
		reviewers, err := repo.GetReviewersByPRID("invalid")

		// Проверки
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

		// Ожидание запроса с несколькими PR
		rows := sqlmock.NewRows([]string{"id", "title", "id", "name"}).
			AddRow(1001, "PR 1", 1, "OPEN").
			AddRow(1002, "PR 2", 2, "MERGED").
			AddRow(1003, "PR 3", 3, "OPEN")
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name").
			WithArgs(2).
			WillReturnRows(rows)

		// Выполнение
		prs, err := repo.GetPRsByReviewerID("u2")

		// Проверки
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

		// Нет PR
		rows := sqlmock.NewRows([]string{"id", "title", "id", "name"})
		mock.ExpectQuery("SELECT pr.id, pr.title, u.id, s.name").
			WithArgs(2).
			WillReturnRows(rows)

		// Выполнение
		prs, err := repo.GetPRsByReviewerID("u2")

		// Проверки
		require.NoError(t, err)
		// При отсутствии результатов возвращается nil
		assert.Nil(t, prs)

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("ошибка: невалидный ID ревьювера", func(t *testing.T) {
		repo, mock := setupPRRepo(t)

		// Выполнение
		prs, err := repo.GetPRsByReviewerID("invalid")

		// Проверки
		require.Error(t, err)
		assert.Nil(t, prs)
		assert.Equal(t, "invalid reviewer ID", err.Error())

		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})
}
