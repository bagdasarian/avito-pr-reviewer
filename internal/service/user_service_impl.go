package service

import (
	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type userService struct {
	userRepo        repository.UserRepository
	pullRequestRepo repository.PullRequestRepository
}

// NewUserService создает новый экземпляр UserService
func NewUserService(userRepo repository.UserRepository, pullRequestRepo repository.PullRequestRepository) UserService {
	return &userService{
		userRepo:        userRepo,
		pullRequestRepo: pullRequestRepo,
	}
}

// SetIsActive устанавливает флаг активности пользователя
func (s *userService) SetIsActive(userID string, isActive bool) (*domain.User, error) {
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	err = s.userRepo.SetIsActive(userID, isActive)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	updatedUser, err := s.userRepo.GetByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	return updatedUser, nil
}

// GetReviewPRs получает список PR'ов, где пользователь назначен ревьювером
func (s *userService) GetReviewPRs(userID string) ([]*domain.PullRequestShort, error) {
	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	prs, err := s.pullRequestRepo.GetPRsByReviewerID(userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}
