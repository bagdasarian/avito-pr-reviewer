package db

import (
	"database/sql"
	"fmt"

	"github.com/bagdasarian/avito-pr-reviewer/internal/config"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgres(cfg *config.Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.DBName,
		cfg.Database.SSLMode,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

func MustLoad(cfg *config.Config) *sql.DB {
	db, err := NewPostgres(cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to connect to database: %v", err))
	}
	return db
}
