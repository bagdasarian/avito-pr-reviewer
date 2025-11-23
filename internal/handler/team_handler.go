package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

func (h *Handler) CreateTeam(w http.ResponseWriter, r *http.Request) {
	var req TeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, err)
		return
	}

	team := httpTeamToDomain(req)
	createdTeam, err := h.teamService.CreateTeam(team)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreateTeamResponse{
		Team: domainTeamToHTTP(createdTeam),
	})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		h.handleError(w, &domain.DomainError{
			Code:    "BAD_REQUEST",
			Message: "team_name parameter is required",
		})
		return
	}

	team, err := h.teamService.GetTeam(teamName)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(domainTeamToHTTP(team))
}

