package handler

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	reviewerStats, err := h.statsService.GetReviewerStats(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	prStats, err := h.statsService.GetPRStatsByStatus(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := StatsResponse{
		ReviewerStats: make([]ReviewerStatResponse, len(reviewerStats)),
		PRStats:       make([]PRStatusStatResponse, len(prStats)),
	}

	for i, stat := range reviewerStats {
		response.ReviewerStats[i] = ReviewerStatResponse{
			UserID:          stat.UserID,
			Username:        stat.Username,
			AssignmentCount: stat.AssignmentCount,
		}
	}

	for i, stat := range prStats {
		response.PRStats[i] = PRStatusStatResponse{
			Status: stat.Status,
			Count:  stat.Count,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
