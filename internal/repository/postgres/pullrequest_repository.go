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

type pullRequestRepository struct {
	executor DBExecutor
}

func NewPullRequestRepository(db *sql.DB) *pullRequestRepository {
	return &pullRequestRepository{executor: db}
}

func prStringIDToInt(stringID string) (int, error) {
	idStr := strings.TrimPrefix(stringID, "pr-")
	return strconv.Atoi(idStr)
}

func prIntToStringID(id int) string {
	return fmt.Sprintf("pr-%d", id)
}

func (r *pullRequestRepository) Create(ctx context.Context, pr *domain.PullRequest) error {
	var statusID int
	err := r.executor.QueryRowContext(ctx, "SELECT id FROM statuses WHERE name = $1", string(pr.Status)).Scan(&statusID)
	if err != nil {
		return err
	}

	authorDBID, err := stringIDToInt(pr.AuthorID)
	if err != nil {
		return errors.New("invalid author ID")
	}

	prDBID, err := prStringIDToInt(pr.ID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	query := `
		INSERT INTO pull_requests (id, title, author_id, status_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	now := time.Now()
	var prID int
	var updatedAt sql.NullTime
	err = r.executor.QueryRowContext(
		ctx,
		query,
		prDBID,
		pr.Title,
		authorDBID,
		statusID,
		now,
	).Scan(&prID, &pr.CreatedAt, &updatedAt)
	if err != nil {
		return err
	}

	_, err = r.executor.ExecContext(ctx, `
		SELECT setval('pull_requests_id_seq', GREATEST((SELECT MAX(id) FROM pull_requests), $1))
	`, prID)
	if err != nil {
		return err
	}

	for _, reviewerID := range pr.AssignedReviewers {
		reviewerDBID, err := stringIDToInt(reviewerID)
		if err != nil {
			return errors.New("invalid reviewer ID")
		}

		_, err = r.executor.ExecContext(
			ctx,
			"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, created_at) VALUES ($1, $2, $3)",
			prDBID,
			reviewerDBID,
			now,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *pullRequestRepository) GetByID(ctx context.Context, id string) (*domain.PullRequest, error) {
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
	err = r.executor.QueryRowContext(ctx, query, prDBID).Scan(
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

	reviewers, err := r.GetReviewersByPRID(ctx, id)
	if err != nil {
		return nil, err
	}
	pr.AssignedReviewers = reviewers

	if pr.Status == domain.StatusMerged && updatedAt.Valid {
		pr.MergedAt = &updatedAt.Time
	}

	return pr, nil
}

func (r *pullRequestRepository) UpdateStatus(ctx context.Context, id string, status domain.Status, mergedAt *time.Time) error {
	prDBID, err := prStringIDToInt(id)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	var statusID int
	err = r.executor.QueryRowContext(ctx, "SELECT id FROM statuses WHERE name = $1", string(status)).Scan(&statusID)
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
	err = r.executor.QueryRowContext(ctx, query, prDBID, statusID, updateTime).Scan(&prID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("pull request not found")
		}
		return err
	}

	return nil
}

func (r *pullRequestRepository) AddReviewer(ctx context.Context, prID string, reviewerID string) error {
	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	reviewerDBID, err := stringIDToInt(reviewerID)
	if err != nil {
		return errors.New("invalid reviewer ID")
	}

	_, err = r.executor.ExecContext(
		ctx,
		"INSERT INTO pull_request_reviewers (pull_request_id, reviewer_id, created_at) VALUES ($1, $2, $3)",
		prDBID,
		reviewerDBID,
		time.Now(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *pullRequestRepository) RemoveReviewer(ctx context.Context, prID string, reviewerID string) error {
	prDBID, err := prStringIDToInt(prID)
	if err != nil {
		return errors.New("invalid pull request ID")
	}

	reviewerDBID, err := stringIDToInt(reviewerID)
	if err != nil {
		return errors.New("invalid reviewer ID")
	}

	result, err := r.executor.ExecContext(
		ctx,
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

	return nil
}

func (r *pullRequestRepository) GetReviewersByPRID(ctx context.Context, prID string) ([]string, error) {
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

	rows, err := r.executor.QueryContext(ctx, query, prDBID)
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

func (r *pullRequestRepository) GetPRsByReviewerID(ctx context.Context, reviewerID string) ([]*domain.PullRequestShort, error) {
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

	rows, err := r.executor.QueryContext(ctx, query, reviewerDBID)
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

func (r *pullRequestRepository) ReplaceReviewer(ctx context.Context, prID string, oldReviewerID string, newReviewerID string) error {
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

	_, err = r.executor.ExecContext(
		ctx,
		"UPDATE pull_request_reviewers SET reviewer_id = $1 WHERE pull_request_id = $2 AND reviewer_id = $3",
		newReviewerDBID,
		prDBID,
		oldReviewerDBID,
	)
	if err != nil {
		return err
	}

	return nil
}
