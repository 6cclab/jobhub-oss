package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// ResearchRepo provides access to the research_briefs table.
type ResearchRepo struct {
	db *sql.DB
}

// NewResearchRepo constructs a ResearchRepo.
func NewResearchRepo(db *sql.DB) *ResearchRepo {
	return &ResearchRepo{db: db}
}

// Upsert inserts a new research brief, or updates the existing one for the
// given company_id if one already exists. Returns the brief's ID.
func (r *ResearchRepo) Upsert(brief models.ResearchBrief) (string, error) {
	var existingID string
	err := r.db.QueryRow(
		`SELECT id FROM research_briefs WHERE company_id = $1`,
		brief.CompanyID,
	).Scan(&existingID)

	switch {
	case err == sql.ErrNoRows:
		id := uuid.NewString()
		_, err = r.db.Exec(
			`INSERT INTO research_briefs (
				id, company_id, stability_verdict, stability_notes, stage, headcount,
				founded, remote_policy, culture_notes, tech_stack, salary_range_text,
				salary_source, raw_markdown, researched_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			id, brief.CompanyID, brief.StabilityVerdict, brief.StabilityNotes, brief.Stage,
			brief.Headcount, brief.Founded, brief.RemotePolicy, brief.CultureNotes,
			brief.TechStack, brief.SalaryRangeText, brief.SalarySource, brief.RawMarkdown,
			brief.ResearchedAt,
		)
		if err != nil {
			return "", fmt.Errorf("insert research brief: %w", err)
		}
		return id, nil

	case err != nil:
		return "", fmt.Errorf("query existing research brief: %w", err)

	default:
		_, err = r.db.Exec(
			`UPDATE research_briefs SET
				stability_verdict = $1, stability_notes = $2, stage = $3, headcount = $4,
				founded = $5, remote_policy = $6, culture_notes = $7, tech_stack = $8,
				salary_range_text = $9, salary_source = $10, raw_markdown = $11,
				researched_at = $12, updated_at = NOW()
			 WHERE id = $13`,
			brief.StabilityVerdict, brief.StabilityNotes, brief.Stage, brief.Headcount,
			brief.Founded, brief.RemotePolicy, brief.CultureNotes, brief.TechStack,
			brief.SalaryRangeText, brief.SalarySource, brief.RawMarkdown,
			brief.ResearchedAt, existingID,
		)
		if err != nil {
			return "", fmt.Errorf("update research brief: %w", err)
		}
		return existingID, nil
	}
}

// FindByCompanyID returns the research brief for a given company, or nil if none exists.
func (r *ResearchRepo) FindByCompanyID(companyID string) (*models.ResearchBrief, error) {
	row := r.db.QueryRow(
		`SELECT rb.id, rb.company_id, rb.stability_verdict, rb.stability_notes, rb.stage,
			rb.headcount, rb.founded, rb.remote_policy, rb.culture_notes, rb.tech_stack,
			rb.salary_range_text, rb.salary_source, rb.raw_markdown,
			to_char(rb.researched_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS researched_at,
			to_char(rb.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(rb.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name, c.slug
		 FROM research_briefs rb
		 JOIN companies c ON c.id = rb.company_id
		 WHERE rb.company_id = $1`,
		companyID,
	)

	brief, err := scanResearchBriefRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan research brief: %w", err)
	}
	return brief, nil
}

// FindByID returns the research brief with the given ID, joined with company info.
func (r *ResearchRepo) FindByID(id string) (*models.ResearchBrief, error) {
	row := r.db.QueryRow(
		`SELECT rb.id, rb.company_id, rb.stability_verdict, rb.stability_notes, rb.stage,
			rb.headcount, rb.founded, rb.remote_policy, rb.culture_notes, rb.tech_stack,
			rb.salary_range_text, rb.salary_source, rb.raw_markdown,
			to_char(rb.researched_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS researched_at,
			to_char(rb.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(rb.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name, c.slug
		 FROM research_briefs rb
		 JOIN companies c ON c.id = rb.company_id
		 WHERE rb.id = $1`,
		id,
	)

	brief, err := scanResearchBriefRow(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan research brief: %w", err)
	}
	return brief, nil
}

// List returns all research briefs, joined with company info, newest first.
func (r *ResearchRepo) List() ([]models.ResearchBrief, error) {
	rows, err := r.db.Query(
		`SELECT rb.id, rb.company_id, rb.stability_verdict, rb.stability_notes, rb.stage,
			rb.headcount, rb.founded, rb.remote_policy, rb.culture_notes, rb.tech_stack,
			rb.salary_range_text, rb.salary_source, rb.raw_markdown,
			to_char(rb.researched_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS researched_at,
			to_char(rb.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(rb.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name, c.slug
		 FROM research_briefs rb
		 JOIN companies c ON c.id = rb.company_id
		 ORDER BY rb.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query research briefs: %w", err)
	}
	defer rows.Close()

	var briefs []models.ResearchBrief
	for rows.Next() {
		var b models.ResearchBrief
		if err := scanResearchBrief(rows, &b); err != nil {
			return nil, fmt.Errorf("scan research brief: %w", err)
		}
		briefs = append(briefs, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate research briefs: %w", err)
	}
	return briefs, nil
}

// rowScanner is the common subset of *sql.Row and *sql.Rows used for scanning.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanResearchBriefRow(row rowScanner) (*models.ResearchBrief, error) {
	var b models.ResearchBrief
	if err := scanResearchBrief(row, &b); err != nil {
		return nil, err
	}
	return &b, nil
}

func scanResearchBrief(row rowScanner, b *models.ResearchBrief) error {
	return row.Scan(
		&b.ID, &b.CompanyID, &b.StabilityVerdict, &b.StabilityNotes, &b.Stage,
		&b.Headcount, &b.Founded, &b.RemotePolicy, &b.CultureNotes, &b.TechStack,
		&b.SalaryRangeText, &b.SalarySource, &b.RawMarkdown, &b.ResearchedAt,
		&b.CreatedAt, &b.UpdatedAt, &b.CompanyName, &b.CompanySlug,
	)
}
