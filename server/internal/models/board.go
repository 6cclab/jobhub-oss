package models

// Board represents a job board / ATS source being tracked or probed.
type Board struct {
	ID           string
	Slug         string
	Name         string
	ATS          string
	Tags         string
	Status       string
	LastProbedAt *string
	CreatedAt    string
	UpdatedAt    string
}
