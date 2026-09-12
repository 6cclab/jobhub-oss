package models

// ResearchBrief captures company research: stability, culture, comp, etc.
type ResearchBrief struct {
	ID               string
	CompanyID        string
	StabilityVerdict *string
	StabilityNotes   *string
	Stage            *string
	Headcount        *string
	Founded          *string
	RemotePolicy     *string
	CultureNotes     *string
	TechStack        *string
	SalaryRangeText  *string
	SalarySource     *string
	RawMarkdown      string
	ResearchedAt     string
	CreatedAt        string
	UpdatedAt        string

	// Joined fields
	CompanyName string
	CompanySlug string
}

// CreateResearchRequest is the payload for creating/upserting a research brief.
type CreateResearchRequest struct {
	Company          string   `json:"company"`
	StabilityVerdict *string  `json:"stability_verdict"`
	StabilityNotes   *string  `json:"stability_notes"`
	Stage            *string  `json:"stage"`
	Headcount        *string  `json:"headcount"`
	Founded          *string  `json:"founded"`
	RemotePolicy     *string  `json:"remote_policy"`
	CultureNotes     *string  `json:"culture_notes"`
	TechStack        []string `json:"tech_stack"`
	SalaryRangeText  *string  `json:"salary_range_text"`
	SalarySource     *string  `json:"salary_source"`
	ResearchedAt     string   `json:"researched_at"`
	RawMarkdown      string   `json:"raw_markdown"`
}
