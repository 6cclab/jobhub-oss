package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// ApplicationRepo provides access to the applications and application_events tables.
type ApplicationRepo struct {
	db *sql.DB
}

// NewApplicationRepo constructs an ApplicationRepo.
func NewApplicationRepo(db *sql.DB) *ApplicationRepo {
	return &ApplicationRepo{db: db}
}

// Create inserts a new application. Returns the new application's ID.
func (r *ApplicationRepo) Create(app models.Application) (string, error) {
	if app.ID == "" {
		app.ID = uuid.NewString()
	}

	status := "applied"
	if app.Status != nil && *app.Status != "" {
		status = *app.Status
	}

	_, err := r.db.Exec(
		`INSERT INTO applications (
			id, company_id, role_title, source, status, applied_at, resume_file,
			resume_data, fit_report_id, notes
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		app.ID, app.CompanyID, app.RoleTitle, app.Source, status, app.AppliedAt,
		app.ResumeFile, app.ResumeData, app.FitReportID, app.Notes,
	)
	if err != nil {
		return "", fmt.Errorf("insert application: %w", err)
	}
	return app.ID, nil
}

// FindByID returns an application joined with company info, with its events loaded.
func (r *ApplicationRepo) FindByID(id string) (*models.Application, error) {
	row := r.db.QueryRow(
		`SELECT a.id, a.company_id, a.role_title, a.source, a.status,
			a.applied_at,
			a.resume_file, a.fit_report_id, a.notes,
			to_char(a.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(a.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name
		 FROM applications a
		 JOIN companies c ON c.id = a.company_id
		 WHERE a.id = $1`,
		id,
	)

	var app models.Application
	err := row.Scan(
		&app.ID, &app.CompanyID, &app.RoleTitle, &app.Source, &app.Status, &app.AppliedAt,
		&app.ResumeFile, &app.FitReportID, &app.Notes, &app.CreatedAt, &app.UpdatedAt,
		&app.CompanyName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan application: %w", err)
	}

	events, err := r.loadEvents(app.ID)
	if err != nil {
		return nil, fmt.Errorf("load application events: %w", err)
	}
	app.Events = events

	return &app, nil
}

// List returns all applications joined with company info, ordered by created_at DESC.
func (r *ApplicationRepo) List() ([]models.Application, error) {
	rows, err := r.db.Query(
		`SELECT a.id, a.company_id, a.role_title, a.source, a.status,
			a.applied_at,
			a.resume_file, a.fit_report_id, a.notes,
			to_char(a.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(a.updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at,
			c.name,
			(a.resume_data IS NOT NULL AND length(a.resume_data) > 0) AS has_resume
		 FROM applications a
		 JOIN companies c ON c.id = a.company_id
		 ORDER BY a.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query applications: %w", err)
	}
	defer rows.Close()

	var apps []models.Application
	for rows.Next() {
		var app models.Application
		var hasResume bool
		if err := rows.Scan(
			&app.ID, &app.CompanyID, &app.RoleTitle, &app.Source, &app.Status, &app.AppliedAt,
			&app.ResumeFile, &app.FitReportID, &app.Notes, &app.CreatedAt, &app.UpdatedAt,
			&app.CompanyName, &hasResume,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		app.HasResume = hasResume
		apps = append(apps, app)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applications: %w", err)
	}
	return apps, nil
}

// UpdateStatus updates an application's status and records an application_event,
// in a single transaction.
func (r *ApplicationRepo) UpdateStatus(id, fromStatus, toStatus string, note *string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.Exec(
		`UPDATE applications SET status = $1, updated_at = NOW() WHERE id = $2`,
		toStatus, id,
	)
	if err != nil {
		return fmt.Errorf("update application status: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return sql.ErrNoRows
	}

	var fromStatusPtr *string
	if fromStatus != "" {
		fromStatusPtr = &fromStatus
	}

	_, err = tx.Exec(
		`INSERT INTO application_events (id, application_id, from_status, to_status, note)
		 VALUES ($1, $2, $3, $4, $5)`,
		uuid.NewString(), id, fromStatusPtr, toStatus, note,
	)
	if err != nil {
		return fmt.Errorf("insert application event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// Update applies a partial update to an application using the given column/value map.
// Keys must be valid column names on the applications table; values are bound directly.
// updated_at is always refreshed.
func (r *ApplicationRepo) Update(id string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}

	allowed := map[string]bool{
		"role_title": true, "source": true, "status": true, "applied_at": true,
		"resume_file": true, "fit_report_id": true, "notes": true,
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

	query := "UPDATE applications SET "
	for i, c := range setClauses {
		if i > 0 {
			query += ", "
		}
		query += c
	}
	query += fmt.Sprintf(" WHERE id = $%d", len(args))

	res, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
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

// StoreResume attaches resume binary data to an application.
func (r *ApplicationRepo) StoreResume(id string, data []byte) error {
	res, err := r.db.Exec(
		`UPDATE applications SET resume_data = $1, updated_at = NOW() WHERE id = $2`,
		data, id,
	)
	if err != nil {
		return fmt.Errorf("store resume: %w", err)
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

// GetResume returns the resume binary data and filename for an application.
func (r *ApplicationRepo) GetResume(id string) ([]byte, string, error) {
	var data []byte
	var filename *string

	err := r.db.QueryRow(
		`SELECT resume_data, resume_file FROM applications WHERE id = $1`,
		id,
	).Scan(&data, &filename)
	if err == sql.ErrNoRows {
		return nil, "", sql.ErrNoRows
	}
	if err != nil {
		return nil, "", fmt.Errorf("get resume: %w", err)
	}

	name := ""
	if filename != nil {
		name = *filename
	}
	return data, name, nil
}

// CountByStatus returns a map of status -> count of applications, for dashboard stats.
func (r *ApplicationRepo) CountByStatus() (map[string]int, error) {
	rows, err := r.db.Query(
		`SELECT status, COUNT(*) FROM applications GROUP BY status`,
	)
	if err != nil {
		return nil, fmt.Errorf("query application counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan application count: %w", err)
		}
		counts[status] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application counts: %w", err)
	}
	return counts, nil
}

func (r *ApplicationRepo) loadEvents(applicationID string) ([]models.ApplicationEvent, error) {
	rows, err := r.db.Query(
		`SELECT id, application_id, from_status, to_status,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at, note
		 FROM application_events WHERE application_id = $1 ORDER BY created_at`,
		applicationID,
	)
	if err != nil {
		return nil, fmt.Errorf("query application events: %w", err)
	}
	defer rows.Close()

	var events []models.ApplicationEvent
	for rows.Next() {
		var e models.ApplicationEvent
		if err := rows.Scan(&e.ID, &e.ApplicationID, &e.FromStatus, &e.ToStatus, &e.CreatedAt, &e.Note); err != nil {
			return nil, fmt.Errorf("scan application event: %w", err)
		}
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application events: %w", err)
	}
	return events, nil
}
