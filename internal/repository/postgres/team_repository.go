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

	// Создаем команду (updated_at не устанавливаем при создании, остается NULL)
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

	// Создаем/обновляем пользователей
	userRepo := NewUserRepository(r.db)
	for _, member := range team.Members {
		user := &domain.User{
			ID:       member.UserID,
			Username: member.Username,
			TeamID:   teamID,
			IsActive: member.IsActive,
		}
		if err := userRepo.Create(user); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *teamRepository) GetByName(name string) (*domain.Team, error) {
	// Получаем команду
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

	// Получаем участников команды
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
