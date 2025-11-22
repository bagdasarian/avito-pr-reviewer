package domain

import "time"

type User struct {
	ID        string
	Username  string
	TeamID    int
	TeamName  string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt *time.Time
}
