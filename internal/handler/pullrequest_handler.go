package handler

import (
	"encoding/json"
	"net/http"
)

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req CreatePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, err)
		return
	}

	pr, err := h.pullRequestService.CreatePR(req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(CreatePRResponse{
		PR: domainPRToHTTP(pr),
	})
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req MergePRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, err)
		return
	}

	pr, err := h.pullRequestService.MergePR(req.PullRequestID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(MergePRResponse{
		PR: domainPRToHTTP(pr),
	})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req ReassignReviewerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.handleError(w, err)
		return
	}

	pr, newReviewerID, err := h.pullRequestService.ReassignReviewer(req.PullRequestID, req.OldUserID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ReassignReviewerResponse{
		PR:         domainPRToHTTP(pr),
		ReplacedBy: newReviewerID,
	})
}
