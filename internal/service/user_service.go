package service

import "github.com/bagdasarian/avito-pr-reviewer/internal/domain"

type UserService interface {
	// SetIsActive устанавливает флаг активности пользователя
	SetIsActive(userID string, isActive bool) (*domain.User, error)

	// GetReviewPRs получает список PR'ов, где пользователь назначен ревьювером
	GetReviewPRs(userID string) ([]*domain.PullRequestShort, error)
}
