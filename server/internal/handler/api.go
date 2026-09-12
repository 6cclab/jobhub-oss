package handler

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/repository"
)

// APIHandler serves the JSON API consumed by the /job skill.
type APIHandler struct {
	companies     *repository.CompanyRepo
	boards        *repository.BoardRepo
	research      *repository.ResearchRepo
	fitReports    *repository.FitReportRepo
	searchResults *repository.SearchResultRepo
	apps          *repository.ApplicationRepo
}

// NewAPIHandler constructs an APIHandler.
func NewAPIHandler(
	companies *repository.CompanyRepo,
	boards *repository.BoardRepo,
	research *repository.ResearchRepo,
	fitReports *repository.FitReportRepo,
	searchResults *repository.SearchResultRepo,
	apps *repository.ApplicationRepo,
) *APIHandler {
	return &APIHandler{
		companies:     companies,
		boards:        boards,
		research:      research,
		fitReports:    fitReports,
		searchResults: searchResults,
		apps:          apps,
	}
}

// --- POST /api/fit-reports ---------------------------------------------

// CreateFitReport handles POST /api/fit-reports.
func (h *APIHandler) CreateFitReport(c *fiber.Ctx) error {
	var req models.CreateFitReportRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.Company == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company is required")
	}
	if req.RoleTitle == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_title is required")
	}
	if req.Verdict == "" {
		return fiber.NewError(fiber.StatusBadRequest, "verdict is required")
	}
	if !validFitVerdict(req.Verdict) {
		return fiber.NewError(fiber.StatusBadRequest, fitVerdictError(req.Verdict))
	}
	if req.VerdictSummary == "" {
		return fiber.NewError(fiber.StatusBadRequest, "verdict_summary is required")
	}

	company, err := h.companies.FindOrCreate(req.Company, slugify(req.Company))
	if err != nil {
		return fmt.Errorf("api: find or create company %q: %w", req.Company, err)
	}

	var researchBriefID *string
	existingResearch, err := h.research.FindByCompanyID(company.ID)
	if err != nil {
		return fmt.Errorf("api: find research for company %q: %w", company.ID, err)
	}
	if existingResearch != nil {
		researchBriefID = &existingResearch.ID
	}

	report := models.FitReport{
		CompanyID:       company.ID,
		RoleTitle:       req.RoleTitle,
		Location:        req.Location,
		Level:           req.Level,
		PostingURL:      req.PostingURL,
		Verdict:         req.Verdict,
		VerdictSummary:  req.VerdictSummary,
		WhyApply:        req.WhyApply,
		ResearchBriefID: researchBriefID,
	}

	signals := buildFitSignals(req.MatchSignals, "match")
	signals = append(signals, buildFitSignals(req.GapSignals, "gap")...)
	signals = append(signals, buildFitSignals(req.FlagSignals, "flag")...)

	id, err := h.fitReports.Create(report, signals)
	if err != nil {
		return fmt.Errorf("api: create fit report: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":  id,
		"url": "/fit-reports/" + id,
	})
}

// --- PATCH /api/fit-reports/:id ----------------------------------------

// fitVerdicts mirrors the fit_reports.verdict CHECK constraint in
// migrations/0001_init.up.sql. Validating here turns a bad verdict into a 400
// instead of letting the constraint surface as a 500.
var fitVerdicts = []string{"strong", "worth", "stretch", "skip"}

func validFitVerdict(v string) bool {
	for _, allowed := range fitVerdicts {
		if v == allowed {
			return true
		}
	}
	return false
}

func fitVerdictError(v string) string {
	return fmt.Sprintf("verdict %q is not valid; must be one of: %s",
		v, strings.Join(fitVerdicts, ", "))
}

// UpdateFitReport handles PATCH /api/fit-reports/:id. Only the supplied fields
// are written; signals are left alone.
func (h *APIHandler) UpdateFitReport(c *fiber.Ctx) error {
	id := c.Params("id")

	var req models.UpdateFitReportRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	fields := map[string]interface{}{}
	if req.RoleTitle != nil {
		if *req.RoleTitle == "" {
			return fiber.NewError(fiber.StatusBadRequest, "role_title cannot be empty")
		}
		fields["role_title"] = *req.RoleTitle
	}
	if req.Location != nil {
		fields["location"] = *req.Location
	}
	if req.Level != nil {
		fields["level"] = *req.Level
	}
	if req.PostingURL != nil {
		fields["posting_url"] = *req.PostingURL
	}
	if req.Verdict != nil {
		if !validFitVerdict(*req.Verdict) {
			return fiber.NewError(fiber.StatusBadRequest, fitVerdictError(*req.Verdict))
		}
		fields["verdict"] = *req.Verdict
	}
	if req.VerdictSummary != nil {
		if *req.VerdictSummary == "" {
			return fiber.NewError(fiber.StatusBadRequest, "verdict_summary cannot be empty")
		}
		fields["verdict_summary"] = *req.VerdictSummary
	}
	if req.WhyApply != nil {
		fields["why_apply"] = *req.WhyApply
	}

	if len(fields) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "no updatable fields supplied")
	}

	if err := h.fitReports.Update(id, fields); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "fit report not found")
		}
		return fmt.Errorf("api: update fit report %q: %w", id, err)
	}

	updated, err := h.fitReports.FindByID(id)
	if err != nil {
		return fmt.Errorf("api: reload fit report %q: %w", id, err)
	}
	if updated == nil {
		return fiber.NewError(fiber.StatusNotFound, "fit report not found")
	}

	return c.JSON(fiber.Map{
		"id":              updated.ID,
		"company":         updated.CompanyName,
		"role_title":      updated.RoleTitle,
		"verdict":         updated.Verdict,
		"verdict_summary": updated.VerdictSummary,
		"updated_at":      updated.UpdatedAt,
		"url":             "/fit-reports/" + updated.ID,
	})
}

func buildFitSignals(inputs []models.SignalInput, kind string) []models.FitSignal {
	signals := make([]models.FitSignal, 0, len(inputs))
	for i, in := range inputs {
		signals = append(signals, models.FitSignal{
			Kind:        kind,
			Requirement: in.Requirement,
			Evidence:    in.Evidence,
			Source:      in.Source,
			SortOrder:   i,
		})
	}
	return signals
}

// --- POST /api/search-results -------------------------------------------

// CreateSearchResults handles POST /api/search-results.
func (h *APIHandler) CreateSearchResults(c *fiber.Ctx) error {
	var req models.CreateSearchResultsRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.RanAt == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ran_at is required")
	}
	if len(req.Results) == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "results must contain at least one entry")
	}

	batch := models.SearchBatch{
		RanAt:          req.RanAt,
		BoardCount:     req.BoardCount,
		RawCount:       req.RawCount,
		LocationFilter: req.LocationFilter,
	}

	results := make([]models.SearchResult, 0, len(req.Results))
	for i, in := range req.Results {
		if in.Company == "" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("results[%d].company is required", i))
		}
		if in.RoleTitle == "" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("results[%d].role_title is required", i))
		}
		if in.PostingURL == "" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("results[%d].posting_url is required", i))
		}

		company, err := h.companies.FindOrCreate(in.Company, slugify(in.Company))
		if err != nil {
			return fmt.Errorf("api: find or create company %q: %w", in.Company, err)
		}

		results = append(results, models.SearchResult{
			CompanyID:       company.ID,
			RoleTitle:       in.RoleTitle,
			Location:        in.Location,
			IsRemote:        in.IsRemote,
			SalaryDisclosed: in.SalaryDisclosed,
			BelowFloor:      in.BelowFloor,
			SalaryMin:       in.SalaryMin,
			SalaryMax:       in.SalaryMax,
			PostingURL:      in.PostingURL,
			FitTier:         in.FitTier,
			Tags:            strings.Join(in.Tags, ","),
			LevelTag:        in.LevelTag,
			DomainTag:       in.DomainTag,
		})
	}

	batchID, err := h.searchResults.CreateBatch(batch, results)
	if err != nil {
		return fmt.Errorf("api: create search batch: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"batch_id": batchID,
		"url":      "/search-results/" + batchID,
	})
}

// --- POST /api/applications ----------------------------------------------

// CreateApplication handles POST /api/applications.
func (h *APIHandler) CreateApplication(c *fiber.Ctx) error {
	var req models.CreateApplicationRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.Company == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company is required")
	}
	if req.RoleTitle == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_title is required")
	}

	company, err := h.companies.FindOrCreate(req.Company, slugify(req.Company))
	if err != nil {
		return fmt.Errorf("api: find or create company %q: %w", req.Company, err)
	}

	var status *string
	if req.Status != "" {
		status = &req.Status
	}

	app := models.Application{
		CompanyID:   company.ID,
		RoleTitle:   req.RoleTitle,
		Source:      req.Source,
		Status:      status,
		AppliedAt:   req.AppliedAt,
		ResumeFile:  req.ResumeFile,
		FitReportID: req.FitReportID,
		Notes:       req.Notes,
	}

	id, err := h.apps.Create(app)
	if err != nil {
		return fmt.Errorf("api: create application: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":  id,
		"url": "/applications/" + id,
	})
}

// --- GET /api/applications ------------------------------------------------

// applicationStatuses is the status vocabulary, and it must stay in lockstep
// with the applications.status CHECK constraint in
// migrations/0001_init.up.sql. Same contract as fitVerdicts above, same reason:
// if the migration changes and this does not, a valid filter starts 400ing.
var applicationStatuses = []string{
	"applied", "phone_screen", "onsite", "offer", "rejected", "ghosted", "withdrawn",
}

func validApplicationStatus(s string) bool {
	for _, v := range applicationStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// filterApplications narrows a list by company and status.
//
// A pure function over an already-loaded slice rather than a WHERE clause,
// because the whole table is dozens of rows and a second query shape is a
// second thing to keep correct. Split out from the handler so it is testable
// without a database — this package's tests are pure-logic by convention and
// there is no DB harness to hang a query test on.
//
// `company` matches case-insensitively on a substring, so "headway" finds
// "Headway" and "crowd" finds "CrowdStrike". Exact matching would push the
// caller into knowing the stored capitalisation, which is the kind of detail
// that turns a lookup into two round trips. `status` is exact, because it is a
// closed vocabulary and a substring match there would make "applied" also
// select nothing useful while quietly looking like it worked.
func filterApplications(apps []models.Application, company, status string) []models.Application {
	out := make([]models.Application, 0, len(apps))
	needle := strings.ToLower(company)
	for _, app := range apps {
		if company != "" && !strings.Contains(strings.ToLower(app.CompanyName), needle) {
			continue
		}
		if status != "" {
			current := "applied" // the column default, for rows that predate a status
			if app.Status != nil {
				current = *app.Status
			}
			if current != status {
				continue
			}
		}
		out = append(out, app)
	}
	return out
}

// ListApplications handles GET /api/applications.
//
// This exists because PATCH /api/applications/:id needs a UUID and nothing in
// the JSON API produced one. The HTML list at /applications has it, but that
// route sits behind SSO, which a bearer token does not satisfy — so an agent
// holding a perfectly good API token could create and update applications while
// being structurally unable to discover the id of one it did not just create.
// Recording a status change then meant asking a human to copy a UUID out of a
// browser, which is not an integration.
//
// Optional `?company=` and `?status=` filters, so the common case — "what is
// the id of the Headway application" — is one request rather than a fetch and a
// client-side scan.
func (h *APIHandler) ListApplications(c *fiber.Ctx) error {
	company := c.Query("company")
	status := c.Query("status")

	if status != "" && !validApplicationStatus(status) {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf(
			"invalid status %q: must be one of %s", status, strings.Join(applicationStatuses, ", ")))
	}

	apps, err := h.apps.List()
	if err != nil {
		return fmt.Errorf("api: list applications: %w", err)
	}

	matched := filterApplications(apps, company, status)
	items := make([]fiber.Map, 0, len(matched))
	for _, app := range matched {
		items = append(items, applicationToJSON(app))
	}

	return c.JSON(fiber.Map{"applications": items, "count": len(items)})
}

// --- GET /api/applications/:id --------------------------------------------

// GetApplication handles GET /api/applications/:id.
//
// Symmetric with GET /api/eval-results/:id, and the reason to have it beyond
// symmetry is verification: after a PATCH, reading the row back is how a caller
// confirms the change actually landed rather than assuming the 200 meant what
// it looked like.
func (h *APIHandler) GetApplication(c *fiber.Ctx) error {
	id := c.Params("id")

	app, err := h.apps.FindByID(id)
	if err != nil {
		return fmt.Errorf("api: find application %q: %w", id, err)
	}
	if app == nil {
		return fiber.NewError(fiber.StatusNotFound, "application not found")
	}

	return c.JSON(applicationToJSON(*app))
}

// --- PATCH /api/applications/:id -----------------------------------------

// updatableApplicationFields is the set a PATCH body may carry, named in the
// 400 so a caller learns the vocabulary from the error rather than by guessing.
const updatableApplicationFields = "status, notes, source, role_title, applied_at, resume_file, fit_report_id"

// decodeUpdateApplication parses a PATCH body strictly.
//
// It rejects unknown keys and bodies that carry no updatable field. Before
// 2026-08-24 the handler used Fiber's lenient BodyParser, so PATCHing an
// unsupported column -- `source` was the one that bit -- parsed cleanly, wrote
// nothing, and returned 200. A silent no-op is worse than a refusal: it looks
// like a successful write to every caller and corrupts anything computed from
// the field afterwards.
func decodeUpdateApplication(body []byte) (models.UpdateApplicationRequest, error) {
	var req models.UpdateApplicationRequest

	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return req, fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if (req.Status == nil || *req.Status == "") && len(req.Fields()) == 0 {
		return req, fiber.NewError(fiber.StatusBadRequest,
			"no updatable fields in body; expected one or more of: "+updatableApplicationFields)
	}

	return req, nil
}

// UpdateApplication handles PATCH /api/applications/:id.
func (h *APIHandler) UpdateApplication(c *fiber.Ctx) error {
	id := c.Params("id")

	req, err := decodeUpdateApplication(c.Body())
	if err != nil {
		return err
	}

	fields := req.Fields()
	statusChange := req.Status != nil && *req.Status != ""

	existing, err := h.apps.FindByID(id)
	if err != nil {
		return fmt.Errorf("api: find application %q: %w", id, err)
	}
	if existing == nil {
		return fiber.NewError(fiber.StatusNotFound, "application not found")
	}

	if statusChange {
		fromStatus := ""
		if existing.Status != nil {
			fromStatus = *existing.Status
		}
		if fromStatus != *req.Status {
			// Notes accompanying a real status change annotate the event, so they
			// are consumed here and must not also be written to the column.
			if err := h.apps.UpdateStatus(id, fromStatus, *req.Status, req.Notes); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fiber.NewError(fiber.StatusNotFound, "application not found")
				}
				return fmt.Errorf("api: update application status %q: %w", id, err)
			}
			delete(fields, "notes")
		}
	}

	if len(fields) > 0 {
		if err := h.apps.Update(id, fields); err != nil {
			return fmt.Errorf("api: update application %q: %w", id, err)
		}
	}

	updated, err := h.apps.FindByID(id)
	if err != nil {
		return fmt.Errorf("api: reload application %q: %w", id, err)
	}
	if updated == nil {
		return fiber.NewError(fiber.StatusNotFound, "application not found")
	}

	return c.JSON(applicationToJSON(*updated))
}

func applicationToJSON(app models.Application) fiber.Map {
	return fiber.Map{
		"id":            app.ID,
		"company_id":    app.CompanyID,
		"company":       app.CompanyName,
		"role_title":    app.RoleTitle,
		"source":        app.Source,
		"status":        app.Status,
		"applied_at":    app.AppliedAt,
		"resume_file":   app.ResumeFile,
		"fit_report_id": app.FitReportID,
		"notes":         app.Notes,
		"created_at":    app.CreatedAt,
		"updated_at":    app.UpdatedAt,
	}
}

// --- POST /api/applications/:id/resume ------------------------------------

// UploadResume handles POST /api/applications/:id/resume, a multipart form
// file upload. The uploaded file's bytes are stored via StoreResume.
func (h *APIHandler) UploadResume(c *fiber.Ctx) error {
	id := c.Params("id")

	existing, err := h.apps.FindByID(id)
	if err != nil {
		return fmt.Errorf("api: find application %q: %w", id, err)
	}
	if existing == nil {
		return fiber.NewError(fiber.StatusNotFound, "application not found")
	}

	fileHeader, err := c.FormFile("resume")
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "missing 'resume' file in multipart form: "+err.Error())
	}

	f, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("api: open uploaded resume file: %w", err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("api: read uploaded resume file: %w", err)
	}

	if err := h.apps.StoreResume(id, data); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "application not found")
		}
		return fmt.Errorf("api: store resume for %q: %w", id, err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"download_url": "/applications/" + id + "/resume",
	})
}

// --- POST /api/research ----------------------------------------------------

// CreateResearch handles POST /api/research.
func (h *APIHandler) CreateResearch(c *fiber.Ctx) error {
	var req models.CreateResearchRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.Company == "" {
		return fiber.NewError(fiber.StatusBadRequest, "company is required")
	}
	if req.RawMarkdown == "" {
		return fiber.NewError(fiber.StatusBadRequest, "raw_markdown is required")
	}
	if req.ResearchedAt == "" {
		return fiber.NewError(fiber.StatusBadRequest, "researched_at is required")
	}

	company, err := h.companies.FindOrCreate(req.Company, slugify(req.Company))
	if err != nil {
		return fmt.Errorf("api: find or create company %q: %w", req.Company, err)
	}

	var techStack *string
	if len(req.TechStack) > 0 {
		joined := strings.Join(req.TechStack, ",")
		techStack = &joined
	}

	brief := models.ResearchBrief{
		CompanyID:        company.ID,
		StabilityVerdict: req.StabilityVerdict,
		StabilityNotes:   req.StabilityNotes,
		Stage:            req.Stage,
		Headcount:        req.Headcount,
		Founded:          req.Founded,
		RemotePolicy:     req.RemotePolicy,
		CultureNotes:     req.CultureNotes,
		TechStack:        techStack,
		SalaryRangeText:  req.SalaryRangeText,
		SalarySource:     req.SalarySource,
		RawMarkdown:      req.RawMarkdown,
		ResearchedAt:     req.ResearchedAt,
	}

	id, err := h.research.Upsert(brief)
	if err != nil {
		return fmt.Errorf("api: upsert research brief: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":  id,
		"url": "/research/" + id,
	})
}

// --- GET /api/boards --------------------------------------------------------

// ListBoards handles GET /api/boards, returning tracked and discovery
// boards grouped by status.
func (h *APIHandler) ListBoards(c *fiber.Ctx) error {
	tracked, err := h.boards.ListByStatus("tracked")
	if err != nil {
		return fmt.Errorf("api: list tracked boards: %w", err)
	}
	discovery, err := h.boards.ListByStatus("discovery")
	if err != nil {
		return fmt.Errorf("api: list discovery boards: %w", err)
	}

	return c.JSON(fiber.Map{
		"tracked":   boardsToJSON(tracked),
		"discovery": boardsToJSON(discovery),
	})
}

func boardsToJSON(boards []models.Board) []fiber.Map {
	out := make([]fiber.Map, 0, len(boards))
	for _, b := range boards {
		out = append(out, fiber.Map{
			"id":             b.ID,
			"slug":           b.Slug,
			"name":           b.Name,
			"ats":            b.ATS,
			"tags":           splitCommaList(b.Tags),
			"status":         b.Status,
			"last_probed_at": b.LastProbedAt,
			"created_at":     b.CreatedAt,
			"updated_at":     b.UpdatedAt,
		})
	}
	return out
}

// --- POST /api/boards -------------------------------------------------------

// createBoardRequest is the payload for creating a board via the API.
type createBoardRequest struct {
	Slug   string   `json:"slug"`
	Name   string   `json:"name"`
	ATS    string   `json:"ats"`
	Tags   []string `json:"tags"`
	Status string   `json:"status"`
}

// CreateBoard handles POST /api/boards.
func (h *APIHandler) CreateBoard(c *fiber.Ctx) error {
	var req createBoardRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid JSON body: "+err.Error())
	}

	if req.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name is required")
	}
	if req.ATS == "" {
		return fiber.NewError(fiber.StatusBadRequest, "ats is required")
	}

	slug := req.Slug
	if slug == "" {
		slug = slugify(req.Name)
	}

	status := req.Status
	if status == "" {
		status = "discovery"
	}

	board := models.Board{
		ID:     uuid.NewString(),
		Slug:   slug,
		Name:   req.Name,
		ATS:    req.ATS,
		Tags:   strings.Join(req.Tags, ","),
		Status: status,
	}

	if err := h.boards.Create(board); err != nil {
		return fmt.Errorf("api: create board: %w", err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": board.ID})
}

// --- helpers ----------------------------------------------------------------

var (
	slugNonAlnum   = regexp.MustCompile(`[^a-z0-9]+`)
	slugTrimHyphen = regexp.MustCompile(`^-+|-+$`)
)

// slugify converts a company/board name into a lowercase, hyphen-separated
// slug: lowercase, replace runs of non-alphanumeric characters with a single
// hyphen, and strip leading/trailing hyphens.
func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = slugNonAlnum.ReplaceAllString(s, "-")
	s = slugTrimHyphen.ReplaceAllString(s, "")
	return s
}
