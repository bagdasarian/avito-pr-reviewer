package postgres

import (
	"context"
	"database/sql"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

type statsRepository struct {
	executor DBExecutor
}

func NewStatsRepository(db *sql.DB) *statsRepository {
	return &statsRepository{executor: db}
}

func (r *statsRepository) GetReviewerStats(ctx context.Context) ([]*domain.ReviewerStat, error) {
	query := `
		SELECT u.id, u.name, COUNT(prr.id) as assignment_count
		FROM users u
		LEFT JOIN pull_request_reviewers prr ON u.id = prr.reviewer_id
		GROUP BY u.id, u.name
		ORDER BY assignment_count DESC
	`

	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*domain.ReviewerStat
	for rows.Next() {
		stat := &domain.ReviewerStat{}
		var userDBID int
		err := rows.Scan(&userDBID, &stat.Username, &stat.AssignmentCount)
		if err != nil {
			return nil, err
		}
		stat.UserID = intToStringID(userDBID)
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}

func (r *statsRepository) GetPRStatsByStatus(ctx context.Context) ([]*domain.PRStatusStat, error) {
	query := `
		SELECT s.name as status, COUNT(pr.id) as count
		FROM statuses s
		LEFT JOIN pull_requests pr ON s.id = pr.status_id
		GROUP BY s.name
		ORDER BY s.name
	`

	rows, err := r.executor.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*domain.PRStatusStat
	for rows.Next() {
		stat := &domain.PRStatusStat{}
		err := rows.Scan(&stat.Status, &stat.Count)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}

	return stats, rows.Err()
}
