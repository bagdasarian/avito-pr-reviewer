package domain

type ReviewerStat struct {
	UserID          string
	Username        string
	AssignmentCount int
}

type PRStatusStat struct {
	Status string
	Count  int
}
