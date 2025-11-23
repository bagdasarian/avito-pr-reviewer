package handler

import "github.com/bagdasarian/avito-pr-reviewer/internal/service"

type Handler struct {
	teamService        service.TeamService
	userService        service.UserService
	pullRequestService service.PullRequestService
}

func NewHandler(
	teamService service.TeamService,
	userService service.UserService,
	pullRequestService service.PullRequestService,
) *Handler {
	return &Handler{
		teamService:        teamService,
		userService:        userService,
		pullRequestService: pullRequestService,
	}
}

