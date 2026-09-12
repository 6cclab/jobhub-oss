package models

// EvalResult represents a persisted eval result row.
type EvalResult struct {
	ID                 string
	ResumeFile         string
	Company            string
	RoleTitle          string
	PostingURL         *string
	KeywordScore       string
	GapFillScore       string
	StyleScore         string
	SkillsScore        string
	StructuralScore    string
	AITellsScore       string
	Verdict            string
	PrimaryIssue       *string
	KeywordDetails     string
	GapFillDetails     string
	StyleDetails       string
	AITellsDetails     string
	AdversarialDetails *string
	EvalMode           string
	AttemptNumber      int
	CreatedAt          string
	UpdatedAt          string
}
