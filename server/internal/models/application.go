package models

// Application represents a job application submitted by the candidate.
type Application struct {
	ID          string
	CompanyID   string
	RoleTitle   string
	Source      *string
	Status      *string // defaults to "applied"
	AppliedAt   *string
	ResumeFile  *string
	ResumeData  []byte
	FitReportID *string
	Notes       *string
	CreatedAt   string
	UpdatedAt   string

	// Joined / loaded fields
	CompanyName string
	HasResume   bool
	Events      []ApplicationEvent
}

// ApplicationEvent is a status-change event in an application's history.
type ApplicationEvent struct {
	ID            string
	ApplicationID string
	FromStatus    *string
	ToStatus      string
	CreatedAt     string
	Note          *string
}

// CreateApplicationRequest is the payload for creating a new application.
type CreateApplicationRequest struct {
	Company     string  `json:"company"`
	RoleTitle   string  `json:"role_title"`
	Source      *string `json:"source"`
	Status      string  `json:"status"` // defaults to "applied" if empty
	AppliedAt   *string `json:"applied_at"`
	ResumeFile  *string `json:"resume_file"`
	FitReportID *string `json:"fit_report_id"`
	Notes       *string `json:"notes"`
}

// UpdateApplicationRequest is the payload for partially updating an application.
//
// Every field here must also be listed in ApplicationRepo.Update's allowed map,
// or the write is rejected at the repository layer. Fields absent from the JSON
// body are left unchanged; unknown fields are a 400 rather than a silent drop.
type UpdateApplicationRequest struct {
	Status      *string `json:"status"`
	Notes       *string `json:"notes"`
	Source      *string `json:"source"`
	RoleTitle   *string `json:"role_title"`
	AppliedAt   *string `json:"applied_at"`
	ResumeFile  *string `json:"resume_file"`
	FitReportID *string `json:"fit_report_id"`
}

// Fields returns the non-status column updates as a repository field map.
// Status is excluded: it routes through UpdateStatus so the change is recorded
// as an event rather than an in-place column write.
func (r UpdateApplicationRequest) Fields() map[string]any {
	f := map[string]any{}
	for col, val := range map[string]*string{
		"notes":         r.Notes,
		"source":        r.Source,
		"role_title":    r.RoleTitle,
		"applied_at":    r.AppliedAt,
		"resume_file":   r.ResumeFile,
		"fit_report_id": r.FitReportID,
	} {
		if val != nil {
			f[col] = *val
		}
	}
	return f
}
