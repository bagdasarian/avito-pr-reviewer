package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/bagdasarian/avito-pr-reviewer/internal/domain"
)

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var domainErr *domain.DomainError
	if errors.As(err, &domainErr) {
		statusCode := getStatusCode(domainErr.Code)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Code:    domainErr.Code,
				Message: domainErr.Message,
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error: ErrorDetail{
			Code:    "INTERNAL_ERROR",
			Message: "internal server error",
		},
	})
}

func getStatusCode(errorCode string) int {
	switch errorCode {
	case "TEAM_EXISTS":
		return http.StatusBadRequest
	case "PR_EXISTS", "PR_MERGED", "NOT_ASSIGNED", "NO_CANDIDATE":
		return http.StatusConflict
	case "NOT_FOUND":
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
