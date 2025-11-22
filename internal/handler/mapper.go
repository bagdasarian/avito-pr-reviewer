package handler

import (
	"time"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

func domainTeamToHTTP(team *domain.Team) TeamResponse {
	members := make([]TeamMemberResponse, 0, len(team.Members))
	for _, member := range team.Members {
		members = append(members, TeamMemberResponse{
			UserID:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	return TeamResponse{
		TeamName: team.Name,
		Members:  members,
	}
}

func httpTeamToDomain(req TeamRequest) *domain.Team {
	members := make([]domain.TeamMember, 0, len(req.Members))
	for _, member := range req.Members {
		members = append(members, domain.TeamMember{
			UserID:   member.UserID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}

	return &domain.Team{
		Name:    req.TeamName,
		Members: members,
	}
}

func domainUserToHTTP(user *domain.User) UserResponse {
	return UserResponse{
		UserID:   user.ID,
		Username: user.Username,
		TeamName: user.TeamName,
		IsActive: user.IsActive,
	}
}

func domainPRToHTTP(pr *domain.PullRequest) PullRequestResponse {
	var createdAt, mergedAt *string
	if !pr.CreatedAt.IsZero() {
		createdAtStr := pr.CreatedAt.Format(time.RFC3339)
		createdAt = &createdAtStr
	}
	if pr.MergedAt != nil {
		mergedAtStr := pr.MergedAt.Format(time.RFC3339)
		mergedAt = &mergedAtStr
	}

	return PullRequestResponse{
		PullRequestID:     pr.ID,
		PullRequestName:   pr.Title,
		AuthorID:          pr.AuthorID,
		Status:            string(pr.Status),
		AssignedReviewers: pr.AssignedReviewers,
		CreatedAt:         createdAt,
		MergedAt:          mergedAt,
	}
}

func domainPRShortToHTTP(pr *domain.PullRequestShort) PullRequestShortResponse {
	return PullRequestShortResponse{
		PullRequestID:   pr.ID,
		PullRequestName: pr.Title,
		AuthorID:        pr.AuthorID,
		Status:          string(pr.Status),
	}
}

func domainPRShortsToHTTP(prs []*domain.PullRequestShort) []PullRequestShortResponse {
	result := make([]PullRequestShortResponse, 0, len(prs))
	for _, pr := range prs {
		result = append(result, domainPRShortToHTTP(pr))
	}
	return result
}

