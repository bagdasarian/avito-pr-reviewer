package server

import (
	"net/http"

	"github.com/bagdasarian/avito-pr-reviewer/internal/handler"
)

func SetupRoutes(mux *http.ServeMux, h *handler.Handler) {
	mux.HandleFunc("POST /team/add", h.CreateTeam)
	mux.HandleFunc("GET /team/get", h.GetTeam)
	mux.HandleFunc("POST /users/setIsActive", h.SetIsActive)
	mux.HandleFunc("GET /users/getReview", h.GetReviewPRs)
	mux.HandleFunc("POST /pullRequest/create", h.CreatePR)
	mux.HandleFunc("POST /pullRequest/merge", h.MergePR)
	mux.HandleFunc("POST /pullRequest/reassign", h.ReassignReviewer)
	mux.HandleFunc("GET /stats", h.GetStats)
}
