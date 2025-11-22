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

type pullRequestRepository struct {
	db *sql.DB
}

func NewPullRequestRepository(db *sql.DB) *pullRequestRepository {
	return &pullRequestRepository{db: db}
}

// prStringIDToInt конвертирует строковый ID PR (например "pr-1001") в числовой
func prStringIDToInt(stringID string) (int, error) {
	// Убираем префикс "pr-" если есть
	idStr := strings.TrimPrefix(stringID, "pr-")
	return strconv.Atoi(idStr)
}

// prIntToStringID конвертирует числовой ID в строковый (например 1001 -> "pr-1001")
func prIntToStringID(id int) string {
	return fmt.Sprintf("pr-%d", id)
}

func (r *pullRequestRepository) Create(pr *domain.PullRequest) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var statusID int
	err = tx.QueryRow("SELECT id FROM statuses WHERE name = $1", string(pr.Status)).Scan(&statusID)
	if err != nil {
		return err
	}

	// Получаем числовой ID автора
	authorDBID, err := stringIDToInt(pr.AuthorID)
	if err != nil {
		return errors.New("invalid author ID")
	}

	// Проверяем существование автора
	var authorExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", authorDBID).Scan(&authorExists)
	if err != nil {
		return err
	}
	if !authorExists {
		return errors.New("author not found")
	}

	// Создаем PR (updated_at не устанавливаем при создании, остается NULL)
	query := `
		INSERT INTO pull_requests (title, author_id, status_id, created_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var prID int
	var updatedAt sql.NullTime
	err = tx.QueryRow(
		query,
		pr.Title,
		authorDBID,
		statusID,
		now,
	).Scan(&prID, &pr.CreatedAt, &updatedAt)
	if err != nil {
		return err
	}

	// Конвертируем числовой ID обратно в строковый
	pr.ID = prIntToStringID(prID)

	// Добавляем ревьюверов
	for _, reviewerID := range pr.AssignedReviewers {
		reviewerDBID, err := stringIDToInt(reviewerID)
		if err != nil {
			return errors.New("invalid reviewer ID")
		}

		// Проверяем существование ревьювера
		var reviewerExists bool
		err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", reviewerDBID).Scan(&reviewerExists)
		if err != nil {
			return err
		}
		if !reviewerExists {
			return errors.New("reviewer not found")
		}

		_, err = tx.Exec(
			"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, created_at) VALUES ($1, $2, $3)",
			prID,
			reviewerDBID,
			now,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *pullRequestRepository) GetByID(id string) (*domain.PullRequest, error) {
	prDBID, err := prStringIDToInt(id)
	if err != nil {
		return nil, errors.New("invalid pull request ID")
	}

	query := `
		SELECT pr.id, pr.title, u.id, s.name, pr.created_at, pr.updated_at
		FROM pull_requests pr
		JOIN users u ON pr.author_id = u.id
		JOIN statuses s ON pr.status_id = s.id
		WHERE pr.id = $1
	`

	pr := &domain.PullRequest{}
	var statusName string
	var createdAt time.Time
	var updatedAt sql.NullTime
	var authorDBID int
	err = r.db.QueryRow(query, prDBID).Scan(
		&prDBID,
		&pr.Title,
		&authorDBID,
		&statusName,
		&createdAt,
		&updatedAt,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("pull request not found")
		}
		return nil, err
	}

	pr.ID = prIntToStringID(prDBID)
	pr.AuthorID = intToStringID(authorDBID)
	pr.Status = domain.Status(statusName)
	pr.CreatedAt = createdAt

	// Получаем ревьюверов
	reviewers, err := r.GetReviewersByPRID(id)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	// Если статус MERGED и updated_at установлен, используем его как mergedAt
	if pr.Status == domain.StatusMerged && updatedAt.Valid {
		pr.MergedAt = &updatedAt.Time
	}

	return pr, nil
}

func (r *pullRequestRepository) UpdateStatus(id string, status domain.Status, mergedAt *time.Time) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prDBID, err := prStringIDToInt(id)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	// Получаем ID статуса
	var statusID int
	err = tx.QueryRow("SELECT id FROM statuses WHERE name = $1", string(status)).Scan(&statusID)
	if err != nil {
		return err
	}

	query := `
		UPDATE pull_requests
		SET status_id = $2, updated_at = $3
		WHERE id = $1
		RETURNING id
	`

	updateTime := time.Now()
	if mergedAt != nil {
		updateTime = *mergedAt
	}

	var prID int
	err = tx.QueryRow(query, prDBID, statusID, updateTime).Scan(&prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("pull request not found")
		}
		return err
	}

	return tx.Commit()
}

func (r *pullRequestRepository) AddReviewer(prID string, reviewerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	reviewerDBID, err := stringIDToInt(reviewerID)
	if err != nil {
		return errors.New("invalid reviewer ID")
	}

	// Проверяем существование PR
	var prExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM pull_requests WHERE id = $1)", prDBID).Scan(&prExists)
	if err != nil {
		return err
	}
	if !prExists {
		return errors.New("pull request not found")
	}

	// Проверяем существование ревьювера
	var reviewerExists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", reviewerDBID).Scan(&reviewerExists)
	if err != nil {
		return err
	}
	if !reviewerExists {
		return errors.New("reviewer not found")
	}

	_, err = tx.Exec(
		"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, created_at) VALUES ($1, $2, $3)",
		prDBID,
		reviewerDBID,
		time.Now(),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *pullRequestRepository) RemoveReviewer(prID string, reviewerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	reviewerDBID, err := stringIDToInt(reviewerID)
	if err != nil {
		return errors.New("invalid reviewer ID")
	}

	result, err := tx.Exec(
		"DELETE FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2",
		prDBID,
		reviewerDBID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("reviewer not assigned to this PR")
	}

	return tx.Commit()
}

func (r *pullRequestRepository) GetReviewersByPRID(prID string) ([]string, error) {
	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return nil, errors.New("invalid pull request ID")
	}

	query := `
		SELECT u.id
		FROM pull_request_reviewers prr
		JOIN users u ON prr.reviewer_id = u.id
		JOIN pull_requests pr ON prr.pull_request_id = pr.id
		WHERE pr.id = $1
		ORDER BY prr.created_at
	`

	rows, err := r.db.Query(query, prDBID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reviewers []string
	for rows.Next() {
		var reviewerDBID int
		if err := rows.Scan(&reviewerDBID); err != nil {
			return nil, err
		}
		reviewers = append(reviewers, intToStringID(reviewerDBID))
	}

	return reviewers, rows.Err()
}

func (r *pullRequestRepository) GetPRsByReviewerID(reviewerID string) ([]*domain.PullRequestShort, error) {
	reviewerDBID, err := stringIDToInt(reviewerID)
	if err != nil {
		return nil, errors.New("invalid reviewer ID")
	}

	query := `
		SELECT pr.id, pr.title, u.id, s.name
		FROM pull_request_reviewers prr
		JOIN pull_requests pr ON prr.pull_request_id = pr.id
		JOIN users u ON pr.author_id = u.id
		JOIN statuses s ON pr.status_id = s.id
		JOIN users reviewer ON prr.reviewer_id = reviewer.id
		WHERE reviewer.id = $1
		ORDER BY pr.created_at DESC
	`

	rows, err := r.db.Query(query, reviewerDBID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prs []*domain.PullRequestShort
	for rows.Next() {
		pr := &domain.PullRequestShort{}
		var statusName string
		var prDBID, authorDBID int
		err := rows.Scan(
			&prDBID,
			&pr.Title,
			&authorDBID,
			&statusName,
		)
		if err != nil {
			return nil, err
		}
		pr.ID = prIntToStringID(prDBID)
		pr.AuthorID = intToStringID(authorDBID)
		pr.Status = domain.Status(statusName)
		prs = append(prs, pr)
	}

	return prs, rows.Err()
}

func (r *pullRequestRepository) ReplaceReviewer(prID string, oldReviewerID string, newReviewerID string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	oldReviewerDBID, err := stringIDToInt(oldReviewerID)
	if err != nil {
		return errors.New("invalid old reviewer ID")
	}

	newReviewerDBID, err := stringIDToInt(newReviewerID)
	if err != nil {
		return errors.New("invalid new reviewer ID")
	}

	// Проверяем, что старый ревьювер назначен
	var exists bool
	err = tx.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM pull_request_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2)",
		prDBID,
		oldReviewerDBID,
	).Scan(&exists)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("reviewer is not assigned to this PR")
	}

	// Заменяем ревьювера
	_, err = tx.Exec(
		"UPDATE pull_request_reviewers SET reviewer_id = $1 WHERE pull_request_id = $2 AND reviewer_id = $3",
		newReviewerDBID,
		prDBID,
		oldReviewerDBID,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}
 