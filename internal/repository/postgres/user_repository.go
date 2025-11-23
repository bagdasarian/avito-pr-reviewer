package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type userRepository struct {
	executor DBExecutor
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{executor: db}
}

func NewUserRepositoryWithTx(tx *sql.Tx) *userRepository {
	return &userRepository{executor: tx}
}

func stringIDToInt(stringID string) (int, error) {
	idStr := strings.TrimPrefix(stringID, "u")
	return strconv.Atoi(idStr)
}

func intToStringID(id int) string {
	return fmt.Sprintf("u%d", id)
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (name, team_id, is_active, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var dbID int
	var updatedAt sql.NullTime
	err := r.executor.QueryRowContext(
		ctx,
		query,
		user.Username,
		user.TeamID,
		user.IsActive,
		now,
	).Scan(&dbID, &user.CreatedAt, &updatedAt)

	if updatedAt.Valid {
		user.UpdatedAt = &updatedAt.Time
	} else {
		user.UpdatedAt = nil
	}

	if err != nil {
		return err
	}

	user.ID = intToStringID(dbID)

	return nil
}

func (r *userRepository) CreateWithID(ctx context.Context, user *domain.User) error {
	dbID, err := stringIDToInt(user.ID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	query := `
		INSERT INTO users (id, name, team_id, is_active, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at
	`

	now := time.Now()
	var updatedAt sql.NullTime
	err = r.executor.QueryRowContext(
		ctx,
		query,
		dbID,
		user.Username,
		user.TeamID,
		user.IsActive,
		now,
	).Scan(&user.CreatedAt, &updatedAt)

	if updatedAt.Valid {
		user.UpdatedAt = &updatedAt.Time
	} else {
		user.UpdatedAt = nil
	}

	if err != nil {
		return err
	}

	return nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	dbID, err := stringIDToInt(user.ID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	query := `
		UPDATE users
		SET name = $2, team_id = $3, is_active = $4, updated_at = $5
		WHERE id = $1
		RETURNING created_at, updated_at
	`

	var updatedAt sql.NullTime
	err = r.executor.QueryRowContext(
		ctx,
		query,
		dbID,
		user.Username,
		user.TeamID,
		user.IsActive,
		time.Now(),
	).Scan(&user.CreatedAt, &updatedAt)

	if updatedAt.Valid {
		user.UpdatedAt = &updatedAt.Time
	} else {
		user.UpdatedAt = nil
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("user not found")
		}
		return err
	}

	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	dbID, err := stringIDToInt(id)
	if err != nil {
		return nil, errors.New("invalid user ID")
	}

	query := `
		SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.id = $1
	`

	user := &domain.User{}
	var updatedAt sql.NullTime
	err = r.executor.QueryRowContext(ctx, query, dbID).Scan(
		&dbID,
		&user.Username,
		&user.TeamID,
		&user.TeamName,
		&user.IsActive,
		&user.CreatedAt,
		&updatedAt,
	)

	if updatedAt.Valid {
		user.UpdatedAt = &updatedAt.Time
	} else {
		user.UpdatedAt = nil
	}

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("user not found")
		}
		return nil, err
	}

	user.ID = intToStringID(dbID)

	return user, nil
}

func (r *userRepository) GetActiveByTeamID(ctx context.Context, teamID int) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.team_id = $1 AND u.is_active = TRUE
		ORDER BY u.created_at
	`

	rows, err := r.executor.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		var dbID int
		var updatedAt sql.NullTime
		err := rows.Scan(
			&dbID,
			&user.Username,
			&user.TeamID,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			user.UpdatedAt = &updatedAt.Time
		} else {
			user.UpdatedAt = nil
		}
		user.ID = intToStringID(dbID)
		users = append(users, user)
	}

	return users, rows.Err()
}

func (r *userRepository) GetByTeamID(ctx context.Context, teamID int) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.team_id = $1
		ORDER BY u.created_at
	`

	rows, err := r.executor.QueryContext(ctx, query, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user := &domain.User{}
		var dbID int
		var updatedAt sql.NullTime
		err := rows.Scan(
			&dbID,
			&user.Username,
			&user.TeamID,
			&user.TeamName,
			&user.IsActive,
			&user.CreatedAt,
			&updatedAt,
		)
		if err != nil {
			return nil, err
		}
		if updatedAt.Valid {
			user.UpdatedAt = &updatedAt.Time
		} else {
			user.UpdatedAt = nil
		}
		user.ID = intToStringID(dbID)
		users = append(users, user)
	}

	return users, rows.Err()
}

func (r *userRepository) SetIsActive(ctx context.Context, userID string, isActive bool) error {
	dbID, err := stringIDToInt(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	query := `
		UPDATE users
		SET is_active = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.executor.ExecContext(ctx, query, dbID, isActive, time.Now())
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("user not found")
	}

	return nil
}
