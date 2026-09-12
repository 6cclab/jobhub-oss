package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

// BoardRepo provides access to the boards table.
type BoardRepo struct {
	db *sql.DB
}

// NewBoardRepo constructs a BoardRepo.
func NewBoardRepo(db *sql.DB) *BoardRepo {
	return &BoardRepo{db: db}
}

// Create inserts a new board. If board.ID is empty, a new UUID is generated.
func (r *BoardRepo) Create(board models.Board) error {
	if board.ID == "" {
		board.ID = uuid.NewString()
	}

	_, err := r.db.Exec(
		`INSERT INTO boards (id, slug, name, ats, tags, status, last_probed_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		board.ID, board.Slug, board.Name, board.ATS, board.Tags, board.Status, board.LastProbedAt,
	)
	if err != nil {
		return fmt.Errorf("insert board: %w", err)
	}
	return nil
}

// List returns all boards ordered by name.
func (r *BoardRepo) List() ([]models.Board, error) {
	rows, err := r.db.Query(
		`SELECT id, slug, name, ats, tags, status,
			to_char(last_probed_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS last_probed_at,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM boards ORDER BY name`,
	)
	if err != nil {
		return nil, fmt.Errorf("query boards: %w", err)
	}
	defer rows.Close()

	return scanBoards(rows)
}

// ListByStatus returns all boards with the given status, ordered by name.
func (r *BoardRepo) ListByStatus(status string) ([]models.Board, error) {
	rows, err := r.db.Query(
		`SELECT id, slug, name, ats, tags, status,
			to_char(last_probed_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS last_probed_at,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM boards WHERE status = $1 ORDER BY name`,
		status,
	)
	if err != nil {
		return nil, fmt.Errorf("query boards by status: %w", err)
	}
	defer rows.Close()

	return scanBoards(rows)
}

// UpdateLastProbed sets last_probed_at (and updated_at) to the current time.
func (r *BoardRepo) UpdateLastProbed(id string) error {
	res, err := r.db.Exec(
		`UPDATE boards SET last_probed_at = NOW(), updated_at = NOW() WHERE id = $1`,
		id,
	)
	if err != nil {
		return fmt.Errorf("update board last_probed_at: %w", err)
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

func scanBoards(rows *sql.Rows) ([]models.Board, error) {
	var boards []models.Board
	for rows.Next() {
		var b models.Board
		if err := rows.Scan(
			&b.ID, &b.Slug, &b.Name, &b.ATS, &b.Tags, &b.Status,
			&b.LastProbedAt, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan board: %w", err)
		}
		boards = append(boards, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate boards: %w", err)
	}
	return boards, nil
}
