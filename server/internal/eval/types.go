package eval

// Score represents a dimension grade.
type Score string

const (
	Pass Score = "pass"
	Warn Score = "warn"
	Fail Score = "fail"
)

// Verdict represents the overall eval outcome.
type Verdict string

const (
	Strong      Verdict = "strong"
	Acceptable  Verdict = "acceptable"
	NeedsRework Verdict = "needs_rework"
	Critical    Verdict = "critical"
)

// TermCategory classifies what kind of requirement a posting term represents.
type TermCategory string

const (
	ExplicitSkill      TermCategory = "explicit_skill"
	ImplicitCapability TermCategory = "implicit_capability"
	DomainKeyword      TermCategory = "domain_keyword"
	SenioritySignal    TermCategory = "seniority_signal"
)

// TermPriority indicates how important a term is to the posting.
type TermPriority string

const (
	Top3     TermPriority = "top3"
	Standard TermPriority = "standard"
)

// TermStatus indicates how a posting term was matched on the resume.
type TermStatus string

const (
	Covered    TermStatus = "covered"
	SkillsOnly TermStatus = "skills_only"
	Missing    TermStatus = "missing"
)

// GapVerdict indicates whether a missing term was fillable by a project.
type GapVerdict string

const (
	GapFillFailure GapVerdict = "gap_fill_failure"
	GenuineGap     GapVerdict = "genuine_gap"
	NotApplicable  GapVerdict = "not_applicable"
)

// PostingTerm is a single requirement extracted from a job posting.
type PostingTerm struct {
	Term     string       `json:"term"`
	Category TermCategory `json:"category"`
	Priority TermPriority `json:"priority"`
}

// EvalRequest is the input to the eval engine.
type EvalRequest struct {
	ResumeFile   string              `json:"resume_file"`
	Company      string              `json:"company"`
	RoleTitle    string              `json:"role_title"`
	PostingURL   *string             `json:"posting_url,omitempty"`
	EvalMode     string              `json:"eval_mode"`
	PostingTerms []PostingTerm       `json:"posting_terms"`
	ResumeText   string              `json:"resume_text"`
	SkillsList   []string            `json:"skills_list"`
	ProjectGaps  map[string][]string `json:"project_gaps"`
	StyleInput   StyleInput          `json:"style_input"`
}

// StyleInput holds the data needed for style checks.
type StyleInput struct {
	SummaryText    string `json:"summary_text"`
	TitleLine      string `json:"title_line"`
	ExpectedLevel  string `json:"expected_level"`
	HasEducation   bool   `json:"has_education"`
	HasPriorRoles  bool   `json:"has_prior_roles"`
	FullResumeText string `json:"full_resume_text"`
	// TargetPages is the page count the resume was deliberately built for.
	// resume-style.md sanctions two pages on broad postings; without this the
	// evaluator assumes one and scores every two-pager as too long. 0 means
	// unspecified and is treated as one page.
	TargetPages int `json:"target_pages"`
}

// TermResult is the eval outcome for a single posting term.
type TermResult struct {
	Term           string       `json:"term"`
	Category       TermCategory `json:"category"`
	Priority       TermPriority `json:"priority"`
	Status         TermStatus   `json:"status"`
	MatchedVia     *string      `json:"matched_via"`
	GapFillProject *string      `json:"gap_fill_project,omitempty"`
	Verdict        GapVerdict   `json:"verdict"`
}

// KeywordCoverage summarizes the keyword matching results.
type KeywordCoverage struct {
	Total      int          `json:"total"`
	Covered    int          `json:"covered"`
	SkillsOnly int          `json:"skills_only"`
	Missing    int          `json:"missing"`
	Percentage int          `json:"percentage"`
	Score      Score        `json:"score"`
	Terms      []TermResult `json:"terms"`
}

// GapFillResult summarizes gap-fill checking.
type GapFillResult struct {
	Failures     int             `json:"failures"`
	Top3Failures int             `json:"top3_failures"`
	Score        Score           `json:"score"`
	Details      []GapFillDetail `json:"details"`
}

// GapFillDetail is a single gap-fill failure.
type GapFillDetail struct {
	Term     string       `json:"term"`
	Project  string       `json:"project"`
	Priority TermPriority `json:"priority"`
}

// StyleResult summarizes style checking.
type StyleResult struct {
	SummaryFirstPerson bool          `json:"summary_first_person"`
	BannedPhrasesFound []string      `json:"banned_phrases_found"`
	UnverifiedMetrics  []string      `json:"unverified_metrics_found"`
	EstimatedChars     int           `json:"estimated_chars"`
	LengthOK           bool          `json:"length_ok"`
	TargetPages        int           `json:"target_pages"`
	LengthMin          int           `json:"length_min"`
	LengthMax          int           `json:"length_max"`
	RoleBullets        []RoleBullets `json:"role_bullets"`
	BulletIssues       []string      `json:"bullet_issues"`
	EmDashCount        int           `json:"em_dash_count"`
	ProseIssues        []ProseIssue  `json:"prose_issues"`
	Score              Score         `json:"score"`
}

// SkillsRelevance summarizes skills section quality.
type SkillsRelevance struct {
	SkillsOnlyTerms []string `json:"skills_only_terms"`
	UnmatchedSkills []string `json:"unmatched_skills"`
	Score           Score    `json:"score"`
}

// StructuralResult summarizes structural checks.
type StructuralResult struct {
	TitleMatches      bool  `json:"title_matches"`
	EducationPresent  bool  `json:"education_present"`
	PriorRolesPresent bool  `json:"prior_roles_present"`
	Score             Score `json:"score"`
}

// Scores holds the per-dimension grades.
type Scores struct {
	KeywordCoverage Score `json:"keyword_coverage"`
	GapFill         Score `json:"gap_fill"`
	Style           Score `json:"style"`
	SkillsRelevance Score `json:"skills_relevance"`
	Structural      Score `json:"structural"`
	AITells         Score `json:"ai_tells"`
}

// EvalResult is the complete eval output.
type EvalResult struct {
	ResumeFile      string           `json:"resume_file"`
	Company         string           `json:"company"`
	RoleTitle       string           `json:"role_title"`
	PostingURL      *string          `json:"posting_url,omitempty"`
	EvalMode        string           `json:"eval_mode"`
	KeywordCoverage KeywordCoverage  `json:"keyword_coverage"`
	GapFill         GapFillResult    `json:"gap_fill"`
	Style           StyleResult      `json:"style"`
	SkillsRelevance SkillsRelevance  `json:"skills_relevance"`
	Structural      StructuralResult `json:"structural"`
	AITells         AITellsResult    `json:"ai_tells"`
	Scores          Scores           `json:"scores"`
	Verdict         Verdict          `json:"verdict"`
	PrimaryIssue    string           `json:"primary_issue"`
}
