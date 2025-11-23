package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

func (h *Handler) SetIsActive(w http.ResponseWriter, r *http.Request) {
	var req SetIsActiveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, err)
		return
	}

	user, err := h.userService.SetIsActive(r.Context(), req.UserID, req.IsActive)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SetIsActiveResponse{
		User: domainUserToHTTP(user),
	})
}

func (h *Handler) GetReviewPRs(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		h.handleError(w, &domain.DomainError{
			Code:    "BAD_REQUEST",
			Message: "user_id parameter is required",
		})
		return
	}

	prs, err := h.userService.GetReviewPRs(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(GetReviewPRsResponse{
		UserID:       userID,
		PullRequests: domainPRShortsToHTTP(prs),
	})
}
