package service

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type PullRequestService interface {
	// CreatePR создает PR и автоматически назначает до 2 активных ревьюверов из команды автора
	CreatePR(prID, title, authorID string) (*domain.PullRequest, error)

	// MergePR помечает PR как MERGED (идемпотентная операция)
	MergePR(prID string) (*domain.PullRequest, error)

	// ReassignReviewer переназначает конкретного ревьювера на другого из его команды
	ReassignReviewer(prID, oldReviewerID string) (*domain.PullRequest, string, error)
}
