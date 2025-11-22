package domain

import "time"

type PullRequest struct {
	ID                string
	Title             string
	AuthorID          string
	Status            Status
	AssignedReviewers []string
	CreatedAt         time.Time
	MergedAt          *time.Time
}

type PullRequestShort struct {
	ID       string
	Title    string
	AuthorID string
	Status   Status
}

type Status string

const (
	StatusOpen   Status = "OPEN"
	StatusMerged Status = "MERGED"
)
