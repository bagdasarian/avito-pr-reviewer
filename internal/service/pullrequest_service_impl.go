package service

import (
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
	"github.com/bagdasarian/avito-pr-reviewer/internal/repository"
)

type pullRequestService struct {
	pullRequestRepo repository.PullRequestRepository
	userRepo        repository.UserRepository
	teamRepo        repository.TeamRepository
}

// NewPullRequestService создает новый экземпляр PullRequestService
func NewPullRequestService(
	pullRequestRepo repository.PullRequestRepository,
	userRepo repository.UserRepository,
	teamRepo repository.TeamRepository,
) PullRequestService {
	return &pullRequestService{
		pullRequestRepo: pullRequestRepo,
		userRepo:        userRepo,
		teamRepo:        teamRepo,
	}
}

// CreatePR создает PR и автоматически назначает до 2 активных ревьюверов из команды автора
func (s *pullRequestService) CreatePR(prID, title, authorID string) (*domain.PullRequest, error) {
	existingPR, err := s.pullRequestRepo.GetByID(prID)
	if err == nil && existingPR != nil {
		return nil, domain.ErrPRExists
	}
	if err != nil && err.Error() != "pull request not found" && err.Error() != "invalid pull request ID" {
		return nil, err
	}

	author, err := s.userRepo.GetByID(authorID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, domain.NewNotFoundError("user with id " + authorID)
		}
		return nil, err
	}

	team, err := s.teamRepo.GetByName(author.TeamName)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, domain.NewNotFoundError("team with name " + author.TeamName)
		}
		return nil, err
	}

	teamMembers, err := s.userRepo.GetByTeamID(team.ID)
	if err != nil {
		return nil, err
	}

	selectedReviewers := SelectReviewers(teamMembers, authorID, 2)

	pr := &domain.PullRequest{
		ID:                prID,
		Title:             title,
		AuthorID:          authorID,
		Status:            domain.StatusOpen,
		AssignedReviewers: selectedReviewers,
		CreatedAt:         time.Now(),
		MergedAt:          nil,
	}

	err = s.pullRequestRepo.Create(pr)
	if err != nil {
		return nil, err
	}

	return pr, nil
}

// MergePR помечает PR как MERGED (идемпотентная операция)
func (s *pullRequestService) MergePR(prID string) (*domain.PullRequest, error) {
	pr, err := s.pullRequestRepo.GetByID(prID)
	if err != nil {
		if err.Error() == "pull request not found" {
			return nil, domain.NewNotFoundError("pull request with id " + prID)
		}
		return nil, err
	}

	if pr.Status == domain.StatusMerged {
		return pr, nil
	}

	now := time.Now()
	err = s.pullRequestRepo.UpdateStatus(prID, domain.StatusMerged, &now)
	if err != nil {
		if err.Error() == "pull request not found" {
			return nil, domain.NewNotFoundError("pull request with id " + prID)
		}
		return nil, err
	}

	mergedPR, err := s.pullRequestRepo.GetByID(prID)
	if err != nil {
		if err.Error() == "pull request not found" {
			return nil, domain.NewNotFoundError("pull request with id " + prID)
		}
		return nil, err
	}

	return mergedPR, nil
}

// ReassignReviewer переназначает конкретного ревьювера на другого из его команды
func (s *pullRequestService) ReassignReviewer(prID, oldReviewerID string) (*domain.PullRequest, string, error) {
	pr, err := s.pullRequestRepo.GetByID(prID)
	if err != nil {
		if err.Error() == "pull request not found" {
			return nil, "", domain.NewNotFoundError("pull request with id " + prID)
		}
		return nil, "", err
	}

	if pr.Status == domain.StatusMerged {
		return nil, "", domain.ErrPRMerged
	}

	isAssigned := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldReviewerID {
			isAssigned = true
			break
		}
	}

	if !isAssigned {
		return nil, "", domain.ErrNotAssigned
	}

	oldReviewer, err := s.userRepo.GetByID(oldReviewerID)
	if err != nil {
		if err.Error() == "user not found" {
			return nil, "", domain.NewNotFoundError("user with id " + oldReviewerID)
		}
		return nil, "", err
	}

	team, err := s.teamRepo.GetByName(oldReviewer.TeamName)
	if err != nil {
		if err.Error() == "team not found" {
			return nil, "", domain.NewNotFoundError("team with name " + oldReviewer.TeamName)
		}
		return nil, "", err
	}

	teamMembers, err := s.userRepo.GetByTeamID(team.ID)
	if err != nil {
		return nil, "", err
	}

	selectedReviewers := SelectReviewers(teamMembers, oldReviewerID, 1)
	if len(selectedReviewers) == 0 {
		return nil, "", domain.ErrNoCandidate
	}

	newReviewerID := selectedReviewers[0]

	err = s.pullRequestRepo.ReplaceReviewer(prID, oldReviewerID, newReviewerID)
	if err != nil {
		if err.Error() == "reviewer is not assigned to this PR" {
			return nil, "", domain.ErrNotAssigned
		}
		return nil, "", err
	}

	updatedPR, err := s.pullRequestRepo.GetByID(prID)
	if err != nil {
		if err.Error() == "pull request not found" {
			return nil, "", domain.NewNotFoundError("pull request with id " + prID)
		}
		return nil, "", err
	}

	return updatedPR, newReviewerID, nil
}
