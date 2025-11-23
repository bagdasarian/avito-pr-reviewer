//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) *sql.DB {
	ctx := context.Background()

	// Создаём контейнер Postgres через testcontainers
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:17.7"),
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	require.NoError(t, err)

	// Получаем DSN (connection string)
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Подключаемся к БД (используем pgx драйвер через stdlib)
	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)

	// Ждём готовности БД
	require.NoError(t, db.Ping())

	// Накатываем миграции
	applyMigrations(t, db)

	// Автоматическая очистка после теста
	t.Cleanup(func() {
		db.Close()
		require.NoError(t, postgresContainer.Terminate(ctx))
	})

	return db
}

func applyMigrations(t *testing.T, db *sql.DB) {
	// Пробуем разные пути к миграции
	var migrationSQL []byte
	var err error
	
	paths := []string{
		filepath.Join("..", "..", "migrations", "000001_init.up.sql"),
		filepath.Join("migrations", "000001_init.up.sql"),
		filepath.Join("..", "migrations", "000001_init.up.sql"),
	}
	
	for _, path := range paths {
		migrationSQL, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "не удалось прочитать файл миграции. Проверьте, что файл migrations/000001_init.up.sql существует")

	// Выполняем миграцию
	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err, "не удалось применить миграцию")
}

