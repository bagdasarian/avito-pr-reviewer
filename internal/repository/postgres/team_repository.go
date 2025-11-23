package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type teamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *teamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) Create(team *domain.Team) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

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
	err = tx.QueryRow(query, team.Name, now).Scan(&teamID, &team.CreatedAt, &updatedAt)

	if updatedAt.Valid {
		team.UpdatedAt = &updatedAt.Time
	} else {
		team.UpdatedAt = nil
	}
	if err != nil {
		return err
	}
	team.ID = teamID

	for _, member := range team.Members {
		dbID, err := stringIDToInt(member.UserID)
		if err != nil {
			query := `
				INSERT INTO users (name, team_id, is_active, created_at)
				VALUES ($1, $2, $3, $4)
				RETURNING id, created_at, updated_at
			`
			var userID int
			var userCreatedAt time.Time
			var updatedAt sql.NullTime
			err = tx.QueryRow(query, member.Username, teamID, member.IsActive, now).Scan(&userID, &userCreatedAt, &updatedAt)
			if err != nil {
				return err
			}
			continue
		}

		updateQuery := `
			UPDATE users
			SET name = $2, team_id = $3, is_active = $4, updated_at = $5
			WHERE id = $1
		`
		result, err := tx.Exec(updateQuery, dbID, member.Username, teamID, member.IsActive, now)
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected == 0 {
			insertQuery := `
				INSERT INTO users (id, name, team_id, is_active, created_at)
				VALUES ($1, $2, $3, $4, $5)
			`
			_, err = tx.Exec(insertQuery, dbID, member.Username, teamID, member.IsActive, now)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *teamRepository) GetByName(name string) (*domain.Team, error) {
	query := `
		SELECT id, name, created_at, updated_at
		FROM teams
		WHERE name = $1
	`

	team := &domain.Team{}
	var updatedAt sql.NullTime
	err := r.db.QueryRow(query, name).Scan(
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

	userRepo := NewUserRepository(r.db)
	users, err := userRepo.GetByTeamID(team.ID)
	if err != nil {
		return nil, err
	}

	team.Members = make([]domain.TeamMember, 0, len(users))
	for _, user := range users {
		team.Members = append(team.Members, domain.TeamMember{
			UserID:   user.ID,
			Username: user.Username,
			IsActive: user.IsActive,
		})
	}

	return team, nil
}
