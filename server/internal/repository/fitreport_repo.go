package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// FitReportRepo provides access to the fit_reports and fit_signals tables.
type FitReportRepo struct {
	db *sql.DB
}

// NewFitReportRepo constructs a FitReportRepo.
func NewFitReportRepo(db *sql.DB) *FitReportRepo {
	return &FitReportRepo{db: db}
}

// Create inserts a fit report and its signals in a single transaction.
// Returns the new fit report's ID.
func (r *FitReportRepo) Create(report models.FitReport, signals []models.FitSignal) (string, error) {
	if report.ID == "" {
		report.ID = uuid.NewString()
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO fit_reports (
			id, company_id, role_title, location, level, posting_url,
			verdict, verdict_summary, why_apply, research_brief_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		report.ID, report.CompanyID, report.RoleTitle, report.Location, report.Level,
		report.PostingURL, report.Verdict, report.VerdictSummary, report.WhyApply,
		report.ResearchBriefID,
	)
	if err != nil {
		return "", fmt.Errorf("insert fit report: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO fit_signals (id, fit_report_id, kind, requirement, evidence, source, sort_order)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
	)
	if err != nil {
		return "", fmt.Errorf("prepare fit signal insert: %w", err)
	}
	defer stmt.Close()

	for _, s := range signals {
		if s.ID == "" {
			s.ID = uuid.NewString()
		}
		_, err = stmt.Exec(s.ID, report.ID, s.Kind, s.Requirement, s.Evidence, s.Source, s.SortOrder)
		if err != nil {
			return "", fmt.Errorf("insert fit signal: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return report.ID, nil
}

// Update applies a partial update to a fit report. Signals are left untouched;
// this is for correcting the report's own fields, most often the verdict after
// research lands. Returns sql.ErrNoRows if no row matched.
func (r *FitReportRepo) Update(id string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	allowed := map[string]bool{
		"role_title": true, "location": true, "level": true, "posting_url": true,
		"verdict": true, "verdict_summary": true, "why_apply": true,
		"research_brief_id": true,
	}

	setClauses := make([]string, 0, len(fields)+1)
	args := make([]any, 0, len(fields)+1)

	for col, val := range fields {
		if !allowed[col] {
			return fmt.Errorf("field %q is not updatable", col)
		}
		args = append(args, val)
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, id)

	query := "UPDATE fit_reports SET "
	for i, c := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += c
	}
	query += fmt.Sprintf(" WHERE id = $%d", len(args))

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update fit report: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// FindByID returns a fit report joined with company info, with its signals
// loaded, and its linked research brief loaded if present.
func (r *FitReportRepo) FindByID(id string) (*models.FitReport, error) {
	row := r.db.QueryRow(
		`SELECT fr.id, fr.company_id, fr.role_title, fr.location, fr.level, fr.posting_url,
			fr.verdict, fr.verdict_summary, fr.why_apply, fr.research_brief_id,
			to_char(fr.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(fr.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name
		 FROM fit_reports fr
		 JOIN companies c ON c.id = fr.company_id
		 WHERE fr.id = $1`,
		id,
	)

	var fr models.FitReport
	err := row.Scan(
		&fr.ID, &fr.CompanyID, &fr.RoleTitle, &fr.Location, &fr.Level, &fr.PostingURL,
		&fr.Verdict, &fr.VerdictSummary, &fr.WhyApply, &fr.ResearchBriefID,
		&fr.CreatedAt, &fr.UpdatedAt, &fr.CompanyName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan fit report: %w", err)
	}

	signals, err := r.loadSignals(fr.ID)
	if err != nil {
		return nil, fmt.Errorf("load fit signals: %w", err)
	}
	fr.Signals = signals

	if fr.ResearchBriefID != nil {
		researchRepo := NewResearchRepo(r.db)
		research, err := researchRepo.FindByID(*fr.ResearchBriefID)
		if err != nil {
			return nil, fmt.Errorf("load research brief: %w", err)
		}
		fr.Research = research
	}

	return &fr, nil
}

// List returns all fit reports joined with company info, ordered by created_at DESC.
func (r *FitReportRepo) List() ([]models.FitReport, error) {
	return r.listWithLimit(0)
}

// Recent returns the most recent fit reports, joined with company info, limited to `limit`.
func (r *FitReportRepo) Recent(limit int) ([]models.FitReport, error) {
	return r.listWithLimit(limit)
}

func (r *FitReportRepo) listWithLimit(limit int) ([]models.FitReport, error) {
	query := `SELECT fr.id, fr.company_id, fr.role_title, fr.location, fr.level, fr.posting_url,
			fr.verdict, fr.verdict_summary, fr.why_apply, fr.research_brief_id,
			to_char(fr.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(fr.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name
		 FROM fit_reports fr
		 JOIN companies c ON c.id = fr.company_id
		 ORDER BY fr.created_at DESC`

	var (
		rows *sql.Rows
		err  error
	)
	if limit > 0 {
		query += ` LIMIT $1`
		rows, err = r.db.Query(query, limit)
	} else {
		rows, err = r.db.Query(query)
	}
	if err != nil {
		return nil, fmt.Errorf("query fit reports: %w", err)
	}
	defer rows.Close()

	var reports []models.FitReport
	for rows.Next() {
		var fr models.FitReport
		if err := rows.Scan(
			&fr.ID, &fr.CompanyID, &fr.RoleTitle, &fr.Location, &fr.Level, &fr.PostingURL,
			&fr.Verdict, &fr.VerdictSummary, &fr.WhyApply, &fr.ResearchBriefID,
			&fr.CreatedAt, &fr.UpdatedAt, &fr.CompanyName,
		); err != nil {
			return nil, fmt.Errorf("scan fit report: %w", err)
		}
		reports = append(reports, fr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fit reports: %w", err)
	}
	return reports, nil
}

func (r *FitReportRepo) loadSignals(fitReportID string) ([]models.FitSignal, error) {
	rows, err := r.db.Query(
		`SELECT id, fit_report_id, kind, requirement, evidence, source, sort_order
		 FROM fit_signals WHERE fit_report_id = $1 ORDER BY kind, sort_order`,
		fitReportID,
	)
	if err != nil {
		return nil, fmt.Errorf("query fit signals: %w", err)
	}
	defer rows.Close()

	var signals []models.FitSignal
	for rows.Next() {
		var s models.FitSignal
		if err := rows.Scan(&s.ID, &s.FitReportID, &s.Kind, &s.Requirement, &s.Evidence, &s.Source, &s.SortOrder); err != nil {
			return nil, fmt.Errorf("scan fit signal: %w", err)
		}
		signals = append(signals, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate fit signals: %w", err)
	}
	return signals, nil
}
