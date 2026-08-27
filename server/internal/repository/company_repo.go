package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// CompanyRepo provides access to the companies table.
type CompanyRepo struct {
	db *sql.DB
}

// NewCompanyRepo constructs a CompanyRepo.
func NewCompanyRepo(db *sql.DB) *CompanyRepo {
	return &CompanyRepo{db: db}
}

// FindOrCreate looks up a company by slug, creating it if it doesn't exist.
//
// The lookup is keyed on slug rather than name because slug is what the table
// enforces uniqueness on. Keying it on name meant two spellings that normalise
// to the same slug — "LaunchDarkly" and "Launchdarkly" — both missed the
// lookup and then collided on insert, failing the whole request with a
// duplicate-key error. Callers still pass the display name; it is only used
// for the row they may create.
func (r *CompanyRepo) FindOrCreate(name, slug string) (models.Company, error) {
	existing, err := r.FindBySlug(slug)
	if err != nil {
		return models.Company{}, fmt.Errorf("find company by slug: %w", err)
	}
	if existing != nil {
		return *existing, nil
	}

	// ON CONFLICT closes the race between the lookup above and this insert;
	// two concurrent requests for a new company would otherwise have one fail.
	id := uuid.NewString()
	_, err = r.db.Exec(
		`INSERT INTO companies (id, name, slug) VALUES ($1, $2, $3)
		 ON CONFLICT (slug) DO NOTHING`,
		id, name, slug,
	)
	if err != nil {
		return models.Company{}, fmt.Errorf("insert company: %w", err)
	}

	created, err := r.FindBySlug(slug)
	if err != nil {
		return models.Company{}, fmt.Errorf("reload created company: %w", err)
	}
	if created == nil {
		return models.Company{}, fmt.Errorf("company %q (slug %q) not found after insert", name, slug)
	}
	return *created, nil
}

// FindBySlug returns the company with the given slug, or nil if none exists.
func (r *CompanyRepo) FindBySlug(slug string) (*models.Company, error) {
	return r.findBy(`slug = $1`, slug)
}

// FindByName returns the company with the given name, or nil if none exists.
func (r *CompanyRepo) FindByName(name string) (*models.Company, error) {
	return r.findBy(`name = $1`, name)
}

func (r *CompanyRepo) findBy(where string, arg any) (*models.Company, error) {
	row := r.db.QueryRow(
		`SELECT id, name, slug,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM companies WHERE `+where,
		arg,
	)

	var c models.Company
	err := row.Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan company: %w", err)
	}
	return &c, nil
}

// List returns all companies ordered by name.
func (r *CompanyRepo) List() ([]models.Company, error) {
	rows, err := r.db.Query(
		`SELECT id, name, slug,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM companies ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("query companies: %w", err)
	}
	defer rows.Close()

	var companies []models.Company
	for rows.Next() {
		var c models.Company
		if err := rows.Scan(&c.ID, &c.Name, &c.Slug, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan company: %w", err)
		}
		companies = append(companies, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate companies: %w", err)
	}
	return companies, nil
}
