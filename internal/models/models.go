package models

import "time"

type Team struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID        int       `json:"id"`
	TeamID    *int      `json:"team_id"`
	Name      string    `json:"name"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

type PR struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	AuthorID  int       `json:"author_id"`
	Status    PRStatus  `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type PRWithReviewers struct {
	PR
	Reviewers []User `json:"reviewers"`
}
