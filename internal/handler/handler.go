package handler

import "github.com/bagdasarian/avito-pr-reviewer/internal/service"

type Handler struct {
	teamService        service.TeamService
	userService        service.UserService
	pullRequestService service.PullRequestService
	statsService       service.StatsService
}

func NewHandler(
	teamService service.TeamService,
	userService service.UserService,
	pullRequestService service.PullRequestService,
	statsService service.StatsService,
) *Handler {
	return &Handler{
		teamService:        teamService,
		userService:        userService,
		pullRequestService: pullRequestService,
		statsService:       statsService,
	}
}
