package models

// SearchBatch represents a single run of the job search / scrape process.
type SearchBatch struct {
	ID             string
	RanAt          string
	BoardCount     int
	RawCount       int
	LocationFilter *string
	Notes          *string
	CreatedAt      string
	ResultCount    int // computed, not a DB column
}

// SearchResult is a single role surfaced by a search batch.
type SearchResult struct {
	ID              string
	SearchBatchID   string
	CompanyID       string
	RoleTitle       string
	Location        *string
	IsRemote        bool
	SalaryDisclosed bool
	BelowFloor      bool
	SalaryMin       *int
	SalaryMax       *int
	PostingURL      string
	FitTier         string
	Tags            string
	LevelTag        *string
	DomainTag       *string
	CreatedAt       string

	// Joined
	CompanyName string
}

// CreateSearchResultsRequest is the payload for recording a new search batch and its results.
type CreateSearchResultsRequest struct {
	RanAt          string              `json:"ran_at"`
	BoardCount     int                 `json:"board_count"`
	RawCount       int                 `json:"raw_count"`
	LocationFilter *string             `json:"location_filter"`
	Results        []SearchResultInput `json:"results"`
}

// SearchResultInput is a single result entry supplied when creating a search batch.
type SearchResultInput struct {
	Company         string   `json:"company"`
	RoleTitle       string   `json:"role_title"`
	Location        *string  `json:"location"`
	IsRemote        bool     `json:"is_remote"`
	SalaryDisclosed bool     `json:"salary_disclosed"`
	BelowFloor      bool     `json:"below_floor"`
	SalaryMin       *int     `json:"salary_min"`
	SalaryMax       *int     `json:"salary_max"`
	PostingURL      string   `json:"posting_url"`
	FitTier         string   `json:"fit_tier"`
	Tags            []string `json:"tags"`
	LevelTag        *string  `json:"level_tag"`
	DomainTag       *string  `json:"domain_tag"`
}
