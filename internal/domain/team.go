package domain

import "time"

type Team struct {
	ID        int
	Name      string
	Members   []TeamMember
	CreatedAt time.Time
	UpdatedAt *time.Time
}

type TeamMember struct {
	UserID   string
	Username string
	IsActive bool
}
