package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *userRepository {
	return &userRepository{db: db}
}

// stringIDToInt конвертирует строковый ID (например "u1") в числовой
func stringIDToInt(stringID string) (int, error) {
	idStr := strings.TrimPrefix(stringID, "u")
	return strconv.Atoi(idStr)
}

// intToStringID конвертирует числовой ID в строковый (например 1 -> "u1")
func intToStringID(id int) string {
	return fmt.Sprintf("u%d", id)
}

func (r *userRepository) Create(user *domain.User) error {
	if user.ID != "" {
		dbID, err := stringIDToInt(user.ID)
		if err == nil {
			query := `
				UPDATE users
				SET name = $2, team_id = $3, is_active = $4, updated_at = $5
				WHERE id = $1
				RETURNING created_at, updated_at
			`
			var updatedAt sql.NullTime
			err = r.db.QueryRow(
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

			if err == nil {
				return nil
			}
		}
	}

	query := `
		INSERT INTO users (name, team_id, is_active, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var dbID int
	var updatedAt sql.NullTime
	err := r.db.QueryRow(
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

func (r *userRepository) Update(user *domain.User) error {
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
	err = r.db.QueryRow(
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

func (r *userRepository) GetByID(id string) (*domain.User, error) {
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
	err = r.db.QueryRow(query, dbID).Scan(
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

func (r *userRepository) GetActiveByTeamID(teamID int) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.team_id = $1 AND u.is_active = TRUE
		ORDER BY u.created_at
	`

	rows, err := r.db.Query(query, teamID)
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

func (r *userRepository) GetByTeamID(teamID int) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.team_id, t.name, u.is_active, u.created_at, u.updated_at
		FROM users u
		JOIN teams t ON u.team_id = t.id
		WHERE u.team_id = $1
		ORDER BY u.created_at
	`

	rows, err := r.db.Query(query, teamID)
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

func (r *userRepository) SetIsActive(userID string, isActive bool) error {
	dbID, err := stringIDToInt(userID)
	if err != nil {
		return errors.New("invalid user ID")
	}

	query := `
		UPDATE users
		SET is_active = $2, updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.Exec(query, dbID, isActive, time.Now())
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
