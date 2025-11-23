package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type teamRepository struct {
	executor DBExecutor
}

func NewTeamRepository(db *sql.DB) *teamRepository {
	return &teamRepository{executor: db}
}

func NewTeamRepositoryWithTx(tx *sql.Tx) *teamRepository {
	return &teamRepository{executor: tx}
}

func (r *teamRepository) Create(ctx context.Context, team *domain.Team) error {
	query := `
		INSERT INTO teams (name, created_at)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE
		SET updated_at = CURRENT_TIMESTAMP
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var teamID int
	var updatedAt sql.NullTime
	err := r.executor.QueryRowContext(ctx, query, team.Name, now).Scan(&teamID, &team.CreatedAt, &updatedAt)

	if updatedAt.Valid {
		team.UpdatedAt = &updatedAt.Time
	} else {
		team.UpdatedAt = nil
	}
	if err != nil {
		return err
	}
	team.ID = teamID

	return nil
}

func (r *teamRepository) GetByName(ctx context.Context, name string) (*domain.Team, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM teams
		WHERE name = $1
	`

	team := &domain.Team{}
	var updatedAt sql.NullTime
	err := r.executor.QueryRowContext(ctx, query, name).Scan(
		&team.ID,
		&team.Name,
		&team.CreatedAt,
		&updatedAt,
	)

	if updatedAt.Valid {
		team.UpdatedAt = &updatedAt.Time
	} else {
		team.UpdatedAt = nil
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("team not found")
		}
		return nil, err
	}

	return team, nil
}
