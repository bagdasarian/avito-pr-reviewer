package service

import (
	"context"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type userService struct {
	userRepo        repository.UserRepository
	pullRequestRepo repository.PullRequestRepository
}

func NewUserService(userRepo repository.UserRepository, pullRequestRepo repository.PullRequestRepository) UserService {
	return &userService{
		userRepo:        userRepo,
		pullRequestRepo: pullRequestRepo,
	}
}

func (s *userService) SetIsActive(ctx context.Context, userID string, isActive bool) (*domain.User, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	err = s.userRepo.SetIsActive(ctx, userID, isActive)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	updatedUser, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	return updatedUser, nil
}

func (s *userService) GetReviewPRs(ctx context.Context, userID string) ([]*domain.PullRequestShort, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + userID)
		}
		return nil, err
	}

	prs, err := s.pullRequestRepo.GetPRsByReviewerID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}
