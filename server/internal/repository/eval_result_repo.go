package repository

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
)

type EvalResultRepo struct {
	db *sql.DB
}

func NewEvalResultRepo(db *sql.DB) *EvalResultRepo {
	return &EvalResultRepo{db: db}
}

func (r *EvalResultRepo) Create(er models.EvalResult) (string, error) {
	if er.ID == "" {
		er.ID = uuid.NewString()
	}

	var maxAttempt int
	err := r.db.QueryRow(
		`SELECT COALESCE(MAX(attempt_number), 0) FROM eval_results WHERE company = $1 AND role_title = $2`,
		er.Company, er.RoleTitle,
	).Scan(&maxAttempt)
	if err != nil {
		return "", fmt.Errorf("query max attempt: %w", err)
	}
	er.AttemptNumber = maxAttempt + 1

	_, err = r.db.Exec(
		`INSERT INTO eval_results (
			id, resume_file, company, role_title, posting_url,
			keyword_score, gap_fill_score, style_score, skills_score, structural_score, ai_tells_score,
			verdict, primary_issue,
			keyword_details, gap_fill_details, style_details, ai_tells_details, adversarial_details,
			eval_mode, attempt_number
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)`,
		er.ID, er.ResumeFile, er.Company, er.RoleTitle, er.PostingURL,
		er.KeywordScore, er.GapFillScore, er.StyleScore, er.SkillsScore, er.StructuralScore, er.AITellsScore,
		er.Verdict, er.PrimaryIssue,
		er.KeywordDetails, er.GapFillDetails, er.StyleDetails, er.AITellsDetails, er.AdversarialDetails,
		er.EvalMode, er.AttemptNumber,
	)
	if err != nil {
		return "", fmt.Errorf("insert eval result: %w", err)
	}
	return er.ID, nil
}

func (r *EvalResultRepo) FindByID(id string) (*models.EvalResult, error) {
	row := r.db.QueryRow(
		`SELECT id, resume_file, company, role_title, posting_url,
			keyword_score, gap_fill_score, style_score, skills_score, structural_score, ai_tells_score,
			verdict, primary_issue,
			keyword_details, gap_fill_details, style_details, ai_tells_details, adversarial_details,
			eval_mode, attempt_number,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM eval_results WHERE id = $1`, id,
	)

	var er models.EvalResult
	err := row.Scan(
		&er.ID, &er.ResumeFile, &er.Company, &er.RoleTitle, &er.PostingURL,
		&er.KeywordScore, &er.GapFillScore, &er.StyleScore, &er.SkillsScore, &er.StructuralScore, &er.AITellsScore,
		&er.Verdict, &er.PrimaryIssue,
		&er.KeywordDetails, &er.GapFillDetails, &er.StyleDetails, &er.AITellsDetails, &er.AdversarialDetails,
		&er.EvalMode, &er.AttemptNumber, &er.CreatedAt, &er.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan eval result: %w", err)
	}
	return &er, nil
}

// FindByCompanyRole returns all eval results for a given company+role, ordered
// by created_at DESC. Used for side-by-side comparison.
func (r *EvalResultRepo) FindByCompanyRole(company, roleTitle string) ([]models.EvalResult, error) {
	rows, err := r.db.Query(
		`SELECT id, resume_file, company, role_title, posting_url,
			keyword_score, gap_fill_score, style_score, skills_score, structural_score, ai_tells_score,
			verdict, primary_issue,
			keyword_details, gap_fill_details, style_details, ai_tells_details, adversarial_details,
			eval_mode, attempt_number,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM eval_results
		 WHERE company = $1 AND role_title = $2
		 ORDER BY created_at DESC`, company, roleTitle,
	)
	if err != nil {
		return nil, fmt.Errorf("query eval results by company/role: %w", err)
	}
	defer rows.Close()

	var results []models.EvalResult
	for rows.Next() {
		var er models.EvalResult
		if err := rows.Scan(
			&er.ID, &er.ResumeFile, &er.Company, &er.RoleTitle, &er.PostingURL,
			&er.KeywordScore, &er.GapFillScore, &er.StyleScore, &er.SkillsScore, &er.StructuralScore, &er.AITellsScore,
			&er.Verdict, &er.PrimaryIssue,
			&er.KeywordDetails, &er.GapFillDetails, &er.StyleDetails, &er.AITellsDetails, &er.AdversarialDetails,
			&er.EvalMode, &er.AttemptNumber, &er.CreatedAt, &er.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval result: %w", err)
		}
		results = append(results, er)
	}
	return results, rows.Err()
}

// VerdictCounts returns the count of each verdict across all eval results.
func (r *EvalResultRepo) VerdictCounts() (map[string]int, error) {
	rows, err := r.db.Query(
		`SELECT verdict, COUNT(*) FROM eval_results GROUP BY verdict`,
	)
	if err != nil {
		return nil, fmt.Errorf("query verdict counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var verdict string
		var count int
		if err := rows.Scan(&verdict, &count); err != nil {
			return nil, fmt.Errorf("scan verdict count: %w", err)
		}
		counts[verdict] = count
	}
	return counts, rows.Err()
}

// ScoreBreakdown returns the count of pass/warn/fail for each dimension.
func (r *EvalResultRepo) ScoreBreakdown() (map[string]map[string]int, error) {
	dimensions := []struct {
		name   string
		column string
	}{
		{"keyword", "keyword_score"},
		{"gap_fill", "gap_fill_score"},
		{"style", "style_score"},
		{"skills", "skills_score"},
		{"structural", "structural_score"},
	}

	result := make(map[string]map[string]int)
	for _, dim := range dimensions {
		rows, err := r.db.Query(
			fmt.Sprintf(`SELECT %s, COUNT(*) FROM eval_results GROUP BY %s`, dim.column, dim.column),
		)
		if err != nil {
			return nil, fmt.Errorf("query %s breakdown: %w", dim.name, err)
		}

		counts := make(map[string]int)
		for rows.Next() {
			var score string
			var count int
			if err := rows.Scan(&score, &count); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan %s count: %w", dim.name, err)
			}
			counts[score] = count
		}
		rows.Close()
		result[dim.name] = counts
	}
	return result, nil
}

// ListGrouped returns eval results grouped by company+role_title.
// Each group is ordered by attempt_number DESC (latest first).
func (r *EvalResultRepo) ListGrouped() ([][]models.EvalResult, error) {
	rows, err := r.db.Query(
		`SELECT id, resume_file, company, role_title, posting_url,
			keyword_score, gap_fill_score, style_score, skills_score, structural_score, ai_tells_score,
			verdict, primary_issue,
			keyword_details, gap_fill_details, style_details, ai_tells_details, adversarial_details,
			eval_mode, attempt_number,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM eval_results ORDER BY company, role_title, attempt_number DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query eval results grouped: %w", err)
	}
	defer rows.Close()

	groupMap := make(map[string]int)
	var groups [][]models.EvalResult

	for rows.Next() {
		var er models.EvalResult
		if err := rows.Scan(
			&er.ID, &er.ResumeFile, &er.Company, &er.RoleTitle, &er.PostingURL,
			&er.KeywordScore, &er.GapFillScore, &er.StyleScore, &er.SkillsScore, &er.StructuralScore, &er.AITellsScore,
			&er.Verdict, &er.PrimaryIssue,
			&er.KeywordDetails, &er.GapFillDetails, &er.StyleDetails, &er.AITellsDetails, &er.AdversarialDetails,
			&er.EvalMode, &er.AttemptNumber, &er.CreatedAt, &er.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval result: %w", err)
		}
		key := er.Company + "\x00" + er.RoleTitle
		if idx, ok := groupMap[key]; ok {
			groups[idx] = append(groups[idx], er)
		} else {
			groupMap[key] = len(groups)
			groups = append(groups, []models.EvalResult{er})
		}
	}
	return groups, rows.Err()
}

func (r *EvalResultRepo) List() ([]models.EvalResult, error) {
	rows, err := r.db.Query(
		`SELECT id, resume_file, company, role_title, posting_url,
			keyword_score, gap_fill_score, style_score, skills_score, structural_score, ai_tells_score,
			verdict, primary_issue,
			keyword_details, gap_fill_details, style_details, ai_tells_details, adversarial_details,
			eval_mode, attempt_number,
			to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS created_at,
			to_char(updated_at, 'YYYY-MM-DD"T"HH24:MI:SS.MS"Z"') AS updated_at
		 FROM eval_results ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("query eval results: %w", err)
	}
	defer rows.Close()

	var results []models.EvalResult
	for rows.Next() {
		var er models.EvalResult
		if err := rows.Scan(
			&er.ID, &er.ResumeFile, &er.Company, &er.RoleTitle, &er.PostingURL,
			&er.KeywordScore, &er.GapFillScore, &er.StyleScore, &er.SkillsScore, &er.StructuralScore, &er.AITellsScore,
			&er.Verdict, &er.PrimaryIssue,
			&er.KeywordDetails, &er.GapFillDetails, &er.StyleDetails, &er.AITellsDetails, &er.AdversarialDetails,
			&er.EvalMode, &er.AttemptNumber, &er.CreatedAt, &er.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan eval result: %w", err)
		}
		results = append(results, er)
	}
	return results, rows.Err()
}
