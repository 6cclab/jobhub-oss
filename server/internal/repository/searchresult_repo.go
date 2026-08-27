package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// SearchResultRepo provides access to the search_batches and search_results tables.
type SearchResultRepo struct {
	db *sql.DB
}

// NewSearchResultRepo constructs a SearchResultRepo.
func NewSearchResultRepo(db *sql.DB) *SearchResultRepo {
	return &SearchResultRepo{db: db}
}

// CreateBatch inserts a search batch and all its results in a single transaction.
// Returns the new batch's ID.
func (r *SearchResultRepo) CreateBatch(batch models.SearchBatch, results []models.SearchResult) (string, error) {
	if batch.ID == "" {
		batch.ID = uuid.NewString()
	}

	tx, err := r.db.Begin()
	if err != nil {
		return "", fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO search_batches (id, ran_at, board_count, raw_count, location_filter, notes)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		batch.ID, batch.RanAt, batch.BoardCount, batch.RawCount, batch.LocationFilter, batch.Notes,
	)
	if err != nil {
		return "", fmt.Errorf("insert search batch: %w", err)
	}

	stmt, err := tx.Prepare(
		`INSERT INTO search_results (
			id, search_batch_id, company_id, role_title, location, is_remote,
			salary_min, salary_max, salary_disclosed, below_floor, posting_url,
			fit_tier, tags, level_tag, domain_tag
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
	)
	if err != nil {
		return "", fmt.Errorf("prepare search result insert: %w", err)
	}
	defer stmt.Close()

	for _, res := range results {
		if res.ID == "" {
			res.ID = uuid.NewString()
		}
		_, err = stmt.Exec(
			res.ID, batch.ID, res.CompanyID, res.RoleTitle, res.Location, res.IsRemote,
			res.SalaryMin, res.SalaryMax, res.SalaryDisclosed, res.BelowFloor,
			res.PostingURL, res.FitTier, res.Tags, res.LevelTag, res.DomainTag,
		)
		if err != nil {
			return "", fmt.Errorf("insert search result: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit tx: %w", err)
	}
	return batch.ID, nil
}

// ListBatches returns all search batches with their result counts, newest first.
func (r *SearchResultRepo) ListBatches() ([]models.SearchBatch, error) {
	rows, err := r.db.Query(
		`SELECT b.id,
			to_char(b.ran_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS ran_at,
			b.board_count, b.raw_count, b.location_filter, b.notes,
			to_char(b.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			COUNT(sr.id) AS result_count
		 FROM search_batches b
		 LEFT JOIN search_results sr ON sr.search_batch_id = b.id
		 GROUP BY b.id
		 ORDER BY b.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query search batches: %w", err)
	}
	defer rows.Close()

	var batches []models.SearchBatch
	for rows.Next() {
		var b models.SearchBatch
		if err := rows.Scan(
			&b.ID, &b.RanAt, &b.BoardCount, &b.RawCount, &b.LocationFilter, &b.Notes,
			&b.CreatedAt, &b.ResultCount,
		); err != nil {
			return nil, fmt.Errorf("scan search batch: %w", err)
		}
		batches = append(batches, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search batches: %w", err)
	}
	return batches, nil
}

// FindBatchByID returns a search batch and all its results (joined with company info).
func (r *SearchResultRepo) FindBatchByID(id string) (*models.SearchBatch, []models.SearchResult, error) {
	row := r.db.QueryRow(
		`SELECT b.id,
			to_char(b.ran_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS ran_at,
			b.board_count, b.raw_count, b.location_filter, b.notes,
			to_char(b.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			COUNT(sr.id) AS result_count
		 FROM search_batches b
		 LEFT JOIN search_results sr ON sr.search_batch_id = b.id
		 WHERE b.id = $1
		 GROUP BY b.id`,
		id,
	)

	var batch models.SearchBatch
	err := row.Scan(
		&batch.ID, &batch.RanAt, &batch.BoardCount, &batch.RawCount, &batch.LocationFilter,
		&batch.Notes, &batch.CreatedAt, &batch.ResultCount,
	)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("scan search batch: %w", err)
	}

	results, err := r.queryResults(
		`SELECT sr.id, sr.search_batch_id, sr.company_id, sr.role_title, sr.location,
			sr.is_remote, sr.salary_disclosed, sr.below_floor, sr.salary_min, sr.salary_max,
			sr.posting_url, sr.fit_tier, sr.tags, sr.level_tag, sr.domain_tag,
			to_char(sr.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			c.name
		 FROM search_results sr
		 JOIN companies c ON c.id = sr.company_id
		 WHERE sr.search_batch_id = $1
		 ORDER BY sr.created_at`,
		id,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query search results: %w", err)
	}

	return &batch, results, nil
}

// FilterResults returns results for a batch, optionally filtered by fit tier,
// location (substring match), level tag, and domain tag. Any nil filter is skipped.
func (r *SearchResultRepo) FilterResults(batchID string, tier, loc, level, domain *string) ([]models.SearchResult, error) {
	query := strings.Builder{}
	query.WriteString(
		`SELECT sr.id, sr.search_batch_id, sr.company_id, sr.role_title, sr.location,
			sr.is_remote, sr.salary_disclosed, sr.below_floor, sr.salary_min, sr.salary_max,
			sr.posting_url, sr.fit_tier, sr.tags, sr.level_tag, sr.domain_tag,
			to_char(sr.created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			c.name
		 FROM search_results sr
		 JOIN companies c ON c.id = sr.company_id
		 WHERE sr.search_batch_id = $1`,
	)

	args := []any{batchID}

	if tier != nil {
		args = append(args, *tier)
		query.WriteString(fmt.Sprintf(" AND sr.fit_tier = $%d", len(args)))
	}
	if loc != nil {
		args = append(args, "%"+*loc+"%")
		query.WriteString(fmt.Sprintf(" AND sr.location LIKE $%d", len(args)))
	}
	if level != nil {
		args = append(args, *level)
		query.WriteString(fmt.Sprintf(" AND sr.level_tag = $%d", len(args)))
	}
	if domain != nil {
		args = append(args, *domain)
		query.WriteString(fmt.Sprintf(" AND sr.domain_tag = $%d", len(args)))
	}

	query.WriteString(" ORDER BY sr.created_at")

	return r.queryResults(query.String(), args...)
}

func (r *SearchResultRepo) queryResults(query string, args ...any) ([]models.SearchResult, error) {
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query search results: %w", err)
	}
	defer rows.Close()

	var results []models.SearchResult
	for rows.Next() {
		var sr models.SearchResult
		if err := rows.Scan(
			&sr.ID, &sr.SearchBatchID, &sr.CompanyID, &sr.RoleTitle, &sr.Location,
			&sr.IsRemote, &sr.SalaryDisclosed, &sr.BelowFloor, &sr.SalaryMin, &sr.SalaryMax,
			&sr.PostingURL, &sr.FitTier, &sr.Tags, &sr.LevelTag, &sr.DomainTag, &sr.CreatedAt,
			&sr.CompanyName,
		); err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, sr)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate search results: %w", err)
	}
	return results, nil
}
