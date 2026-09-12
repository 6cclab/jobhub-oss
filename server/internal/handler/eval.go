package handler

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/eval"
	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

type EvalHandler struct {
	evals    *repository.EvalResultRepo
	renderer *render.Renderer
	engine   *eval.Engine
}

func NewEvalHandler(evals *repository.EvalResultRepo, renderer *render.Renderer, engine *eval.Engine) *EvalHandler {
	return &EvalHandler{evals: evals, renderer: renderer, engine: engine}
}

// RunEval handles POST /api/eval — runs the eval engine and persists the result.
func (h *EvalHandler) RunEval(c *fiber.Ctx) error {
	var req eval.EvalRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.Company == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company is required")
	}
	if req.RoleTitle == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_title is required")
	}
	if len(req.PostingTerms) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "posting_terms must not be empty")
	}
	if req.EvalMode == "" {
		req.EvalMode = "manual"
	}

	result := h.engine.Run(req)

	keywordJSON, err := json.Marshal(result.KeywordCoverage)
	if err != nil {
		return fmt.Errorf("marshal keyword details: %w", err)
	}
	gapFillJSON, err := json.Marshal(result.GapFill)
	if err != nil {
		return fmt.Errorf("marshal gap fill details: %w", err)
	}
	styleJSON, err := json.Marshal(result.Style)
	if err != nil {
		return fmt.Errorf("marshal style details: %w", err)
	}
	aiTellsJSON, err := json.Marshal(result.AITells)
	if err != nil {
		return fmt.Errorf("marshal ai tells details: %w", err)
	}

	var primaryIssue *string
	if result.PrimaryIssue != "" {
		primaryIssue = &result.PrimaryIssue
	}

	er := models.EvalResult{
		ResumeFile:      result.ResumeFile,
		Company:         result.Company,
		RoleTitle:       result.RoleTitle,
		PostingURL:      result.PostingURL,
		KeywordScore:    string(result.Scores.KeywordCoverage),
		GapFillScore:    string(result.Scores.GapFill),
		StyleScore:      string(result.Scores.Style),
		SkillsScore:     string(result.Scores.SkillsRelevance),
		StructuralScore: string(result.Scores.Structural),
		AITellsScore:    string(result.Scores.AITells),
		Verdict:         string(result.Verdict),
		PrimaryIssue:    primaryIssue,
		KeywordDetails:  string(keywordJSON),
		GapFillDetails:  string(gapFillJSON),
		StyleDetails:    string(styleJSON),
		AITellsDetails:  string(aiTellsJSON),
		EvalMode:        req.EvalMode,
	}

	id, err := h.evals.Create(er)
	if err != nil {
		return fmt.Errorf("persist eval result: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":     id,
		"url":    "/eval-results/" + id,
		"result": result,
	})
}

// ListEvalResults handles GET /api/eval-results.
func (h *EvalHandler) ListEvalResults(c *fiber.Ctx) error {
	results, err := h.evals.List()
	if err != nil {
		return fmt.Errorf("list eval results: %w", err)
	}

	items := make([]fiber.Map, 0, len(results))
	for _, er := range results {
		items = append(items, evalResultToJSON(er))
	}

	return c.JSON(fiber.Map{"results": items})
}

// GetEvalResult handles GET /api/eval-results/:id.
func (h *EvalHandler) GetEvalResult(c *fiber.Ctx) error {
	id := c.Params("id")

	er, err := h.evals.FindByID(id)
	if err != nil {
		return fmt.Errorf("find eval result: %w", err)
	}
	if er == nil {
		return fiber.NewError(fiber.StatusNotFound, "eval result not found")
	}

	return c.JSON(evalResultToJSON(*er))
}

type dimensionStat struct {
	Name string
	Pass int
	Warn int
	Fail int
}

type evalGroup struct {
	Latest  models.EvalResult
	History []models.EvalResult
}

type evalListViewModel struct {
	Groups         []evalGroup
	TotalResults   int
	VerdictCounts  map[string]int
	VerdictPct     map[string]int
	DimensionStats []dimensionStat
}

// List renders the eval-results HTML page with trend data.
func (h *EvalHandler) List(c *fiber.Ctx) error {
	grouped, err := h.evals.ListGrouped()
	if err != nil {
		return fmt.Errorf("list eval results: %w", err)
	}

	var vm evalListViewModel
	for _, g := range grouped {
		eg := evalGroup{Latest: g[0]}
		if len(g) > 1 {
			eg.History = g[1:]
		}
		vm.Groups = append(vm.Groups, eg)
		vm.TotalResults += len(g)
	}

	verdictCounts, err := h.evals.VerdictCounts()
	if err == nil {
		vm.VerdictCounts = verdictCounts
		total := 0
		for _, v := range verdictCounts {
			total += v
		}
		vm.VerdictPct = make(map[string]int)
		if total > 0 {
			for k, v := range verdictCounts {
				vm.VerdictPct[k] = (v * 100) / total
			}
		}
	}

	breakdown, err := h.evals.ScoreBreakdown()
	if err == nil {
		dimNames := []struct{ key, label string }{
			{"keyword", "Keywords"},
			{"gap_fill", "Gap-Fill"},
			{"style", "Style"},
			{"skills", "Skills"},
			{"structural", "Structural"},
		}
		for _, d := range dimNames {
			counts := breakdown[d.key]
			vm.DimensionStats = append(vm.DimensionStats, dimensionStat{
				Name: d.label,
				Pass: counts["pass"],
				Warn: counts["warn"],
				Fail: counts["fail"],
			})
		}
	}

	return h.renderer.Page(c, "evalresults/list.html", vm)
}

type termComparisonRow struct {
	Term     string
	Statuses []string
}

type evalCompareViewModel struct {
	Company        string
	RoleTitle      string
	Evals          []models.EvalResult
	TermComparison []termComparisonRow
}

// Compare renders the side-by-side comparison page for a company/role.
func (h *EvalHandler) Compare(c *fiber.Ctx) error {
	company := c.Query("company")
	role := c.Query("role")

	if company == "" || role == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company and role query params are required")
	}

	results, err := h.evals.FindByCompanyRole(company, role)
	if err != nil {
		return fmt.Errorf("find eval results for compare: %w", err)
	}

	vm := evalCompareViewModel{
		Company:   company,
		RoleTitle: role,
		Evals:     results,
	}

	if len(results) >= 2 {
		allTerms := make(map[string]bool)
		termOrder := []string{}
		type evalTermMap map[string]string

		evalTermMaps := make([]evalTermMap, len(results))
		for i, er := range results {
			var kwDetails eval.KeywordCoverage
			evalTermMaps[i] = make(evalTermMap)
			if err := json.Unmarshal([]byte(er.KeywordDetails), &kwDetails); err == nil {
				for _, t := range kwDetails.Terms {
					evalTermMaps[i][t.Term] = string(t.Status)
					if !allTerms[t.Term] {
						allTerms[t.Term] = true
						termOrder = append(termOrder, t.Term)
					}
				}
			}
		}

		for _, term := range termOrder {
			row := termComparisonRow{Term: term}
			for _, etm := range evalTermMaps {
				status, ok := etm[term]
				if !ok {
					status = ""
				}
				row.Statuses = append(row.Statuses, status)
			}
			vm.TermComparison = append(vm.TermComparison, row)
		}
	}

	return h.renderer.Page(c, "evalresults/compare.html", vm)
}

type keywordTermView struct {
	Term           string
	Category       string
	Priority       string
	Status         string
	Verdict        string
	GapFillProject string
}

type gapFillDetailView struct {
	Term     string
	Project  string
	Priority string
}

type evalDetailViewModel struct {
	Result          models.EvalResult
	KeywordPct      int
	KeywordCovered  int
	KeywordTotal    int
	GapFillFailures int
	KeywordTerms    []keywordTermView
	GapFillDetails  []gapFillDetailView
	StyleIssues     []string
	AITells         []aiTellView
	AITellScore     int
	AITellCV        string
}

type aiTellView struct {
	Label  string
	Text   string
	Detail string
}

// Detail renders a single eval result HTML page.
func (h *EvalHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	er, err := h.evals.FindByID(id)
	if err != nil {
		return fmt.Errorf("find eval result: %w", err)
	}
	if er == nil {
		return fiber.NewError(fiber.StatusNotFound, "eval result not found")
	}

	vm := evalDetailViewModel{Result: *er}

	var kwDetails eval.KeywordCoverage
	if err := json.Unmarshal([]byte(er.KeywordDetails), &kwDetails); err == nil {
		vm.KeywordPct = kwDetails.Percentage
		vm.KeywordCovered = kwDetails.Covered
		vm.KeywordTotal = kwDetails.Total
		for _, t := range kwDetails.Terms {
			tv := keywordTermView{
				Term:     t.Term,
				Category: string(t.Category),
				Priority: string(t.Priority),
				Status:   string(t.Status),
				Verdict:  string(t.Verdict),
			}
			if t.GapFillProject != nil {
				tv.GapFillProject = *t.GapFillProject
			}
			vm.KeywordTerms = append(vm.KeywordTerms, tv)
		}
	}

	var gfDetails eval.GapFillResult
	if err := json.Unmarshal([]byte(er.GapFillDetails), &gfDetails); err == nil {
		vm.GapFillFailures = gfDetails.Failures
		for _, d := range gfDetails.Details {
			vm.GapFillDetails = append(vm.GapFillDetails, gapFillDetailView{
				Term:     d.Term,
				Project:  d.Project,
				Priority: string(d.Priority),
			})
		}
	}

	var styleDetails eval.StyleResult
	if err := json.Unmarshal([]byte(er.StyleDetails), &styleDetails); err == nil {
		if !styleDetails.SummaryFirstPerson {
			vm.StyleIssues = append(vm.StyleIssues, "Summary does not start with \"I\" — uses third-person voice")
		}
		for _, p := range styleDetails.BannedPhrasesFound {
			vm.StyleIssues = append(vm.StyleIssues, fmt.Sprintf("Banned phrase found: \"%s\"", p))
		}
		for _, m := range styleDetails.UnverifiedMetrics {
			vm.StyleIssues = append(vm.StyleIssues, fmt.Sprintf("Unverified metric used: \"%s\"", m))
		}
		if !styleDetails.LengthOK {
			vm.StyleIssues = append(vm.StyleIssues, fmt.Sprintf("Length outside target range (%d chars)", styleDetails.EstimatedChars))
		}
		for _, pi := range styleDetails.ProseIssues {
			if pi.Text != "" {
				vm.StyleIssues = append(vm.StyleIssues, fmt.Sprintf("%s: %s — \"%s\"", proseLabel(pi.Type), pi.Detail, pi.Text))
			} else {
				vm.StyleIssues = append(vm.StyleIssues, fmt.Sprintf("%s: %s", proseLabel(pi.Type), pi.Detail))
			}
		}
	}

	var aiDetails eval.AITellsResult
	if err := json.Unmarshal([]byte(er.AITellsDetails), &aiDetails); err == nil {
		vm.AITellScore = aiDetails.WeightedScore
		if aiDetails.SentenceCount > 0 {
			vm.AITellCV = fmt.Sprintf("%.2f", aiDetails.LengthCV)
		}
		for _, t := range aiDetails.Tells {
			vm.AITells = append(vm.AITells, aiTellView{
				Label:  aiTellLabel(t.Type),
				Text:   t.Text,
				Detail: t.Detail,
			})
		}
	}

	return h.renderer.Page(c, "evalresults/detail.html", vm)
}

// aiTellLabel turns an AI tell type into a human-readable heading.
func aiTellLabel(t string) string {
	switch t {
	case eval.AITellPhrase:
		return "LLM phrase"
	case eval.AITellParticiple:
		return "Participle cascade"
	case eval.AITellNotJust:
		return "\"Not just X but Y\""
	case eval.AITellTriad:
		return "Rule of three"
	case eval.AITellAdverbPadding:
		return "Adverb padding"
	case eval.AITellVagueMetric:
		return "Vague quantifier"
	case eval.AITellUniformity:
		return "Uniform sentence length"
	default:
		return "AI tell"
	}
}

// proseLabel turns a prose issue type into a human-readable heading.
func proseLabel(t string) string {
	switch t {
	case eval.ProseFragment:
		return "Sentence fragment"
	case eval.ProseEmDash:
		return "Em-dashes"
	case eval.ProseRepeatedWord:
		return "Repeated word"
	case eval.ProseDoubleSpace:
		return "Double space"
	case eval.ProseLongSentence:
		return "Long sentence"
	default:
		return "Prose"
	}
}

func evalResultToJSON(er models.EvalResult) fiber.Map {
	return fiber.Map{
		"id":                  er.ID,
		"resume_file":         er.ResumeFile,
		"company":             er.Company,
		"role_title":          er.RoleTitle,
		"posting_url":         er.PostingURL,
		"keyword_score":       er.KeywordScore,
		"gap_fill_score":      er.GapFillScore,
		"style_score":         er.StyleScore,
		"skills_score":        er.SkillsScore,
		"structural_score":    er.StructuralScore,
		"verdict":             er.Verdict,
		"primary_issue":       er.PrimaryIssue,
		"keyword_details":     er.KeywordDetails,
		"gap_fill_details":    er.GapFillDetails,
		"style_details":       er.StyleDetails,
		"adversarial_details": er.AdversarialDetails,
		"eval_mode":           er.EvalMode,
		"attempt_number":      er.AttemptNumber,
		"created_at":          er.CreatedAt,
		"updated_at":          er.UpdatedAt,
	}
}
