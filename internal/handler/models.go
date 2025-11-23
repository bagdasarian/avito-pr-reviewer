package handler

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type TeamMemberRequest struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamRequest struct {
	TeamName string              `json:"team_name"`
	Members  []TeamMemberRequest `json:"members"`
}

type TeamMemberResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	IsActive bool   `json:"is_active"`
}

type TeamResponse struct {
	TeamName string               `json:"team_name"`
	Members  []TeamMemberResponse `json:"members"`
}

type CreateTeamResponse struct {
	Team TeamResponse `json:"team"`
}

type SetIsActiveRequest struct {
	UserID   string `json:"user_id"`
	IsActive bool   `json:"is_active"`
}

type UserResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

type SetIsActiveResponse struct {
	User UserResponse `json:"user"`
}

type CreatePRRequest struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
}

type PullRequestResponse struct {
	PullRequestID     string   `json:"pull_request_id"`
	PullRequestName   string   `json:"pull_request_name"`
	AuthorID          string   `json:"author_id"`
	Status            string   `json:"status"`
	AssignedReviewers []string `json:"assigned_reviewers"`
	CreatedAt         *string  `json:"createdAt,omitempty"`
	MergedAt          *string  `json:"mergedAt,omitempty"`
}

type CreatePRResponse struct {
	PR PullRequestResponse `json:"pr"`
}

type MergePRRequest struct {
	PullRequestID string `json:"pull_request_id"`
}

type MergePRResponse struct {
	PR PullRequestResponse `json:"pr"`
}

type ReassignReviewerRequest struct {
	PullRequestID string `json:"pull_request_id"`
	OldUserID     string `json:"old_user_id"`
}

type ReassignReviewerResponse struct {
	PR         PullRequestResponse `json:"pr"`
	ReplacedBy string              `json:"replaced_by"`
}

type PullRequestShortResponse struct {
	PullRequestID   string `json:"pull_request_id"`
	PullRequestName string `json:"pull_request_name"`
	AuthorID        string `json:"author_id"`
	Status          string `json:"status"`
}

type GetReviewPRsResponse struct {
	UserID       string                      `json:"user_id"`
	PullRequests []PullRequestShortResponse `json:"pull_requests"`
}

type ReviewerStatResponse struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	AssignmentCount int    `json:"assignment_count"`
}

type PRStatusStatResponse struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type StatsResponse struct {
	ReviewerStats []ReviewerStatResponse `json:"reviewer_stats"`
	PRStats       []PRStatusStatResponse `json:"pr_stats"`
}
