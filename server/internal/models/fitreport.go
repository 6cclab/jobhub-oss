package models

// FitReport captures an assessment of how well a role/company fits the candidate.
type FitReport struct {
	ID              string
	CompanyID       string
	RoleTitle       string
	Location        *string
	Level           *string
	PostingURL      *string
	Verdict         string
	VerdictSummary  string
	WhyApply        *string
	ResearchBriefID *string
	CreatedAt       string
	UpdatedAt       string

	// Joined / loaded fields
	CompanyName string
	Signals     []FitSignal
	Research    *ResearchBrief
}

// FitSignal is a single match/gap/flag signal backing a FitReport verdict.
type FitSignal struct {
	ID          string
	FitReportID string
	Kind        string
	Requirement string
	Evidence    string
	Source      *string
	SortOrder   int
}

// CreateFitReportRequest is the payload for creating a fit report and its signals.
type CreateFitReportRequest struct {
	Company        string        `json:"company"`
	RoleTitle      string        `json:"role_title"`
	Location       *string       `json:"location"`
	Level          *string       `json:"level"`
	PostingURL     *string       `json:"posting_url"`
	Verdict        string        `json:"verdict"`
	VerdictSummary string        `json:"verdict_summary"`
	WhyApply       *string       `json:"why_apply"`
	MatchSignals   []SignalInput `json:"match_signals"`
	GapSignals     []SignalInput `json:"gap_signals"`
	FlagSignals    []SignalInput `json:"flag_signals"`
}

// UpdateFitReportRequest is the payload for PATCH /api/fit-reports/:id.
// Every field is optional; only the ones supplied are written. A verdict can
// change after the fact — company research routinely turns a "worth" into a
// "skip" — so the report is editable without recreating it and its signals.
type UpdateFitReportRequest struct {
	RoleTitle      *string `json:"role_title"`
	Location       *string `json:"location"`
	Level          *string `json:"level"`
	PostingURL     *string `json:"posting_url"`
	Verdict        *string `json:"verdict"`
	VerdictSummary *string `json:"verdict_summary"`
	WhyApply       *string `json:"why_apply"`
}

// SignalInput is a single input signal (match, gap, or flag) supplied on create.
type SignalInput struct {
	Requirement string  `json:"requirement"`
	Evidence    string  `json:"evidence"`
	Source      *string `json:"source"`
}
