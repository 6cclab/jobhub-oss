package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// FitReportHandler serves fit report list/detail pages.
type FitReportHandler struct {
	fitReports *repository.FitReportRepo
	render     *render.Renderer
}

// NewFitReportHandler constructs a FitReportHandler.
func NewFitReportHandler(fitReports *repository.FitReportRepo, renderer *render.Renderer) *FitReportHandler {
	return &FitReportHandler{fitReports: fitReports, render: renderer}
}

// fitReportListRow is the view-model for a row in fitreports/list.html.
type fitReportListRow struct {
	ID      string
	Company string
	Role    string
	Verdict string
	Date    string
}

type fitReportListViewModel struct {
	FitReports []fitReportListRow
}

// List handles GET /fit-reports.
func (h *FitReportHandler) List(c *fiber.Ctx) error {
	reports, err := h.fitReports.List()
	if err != nil {
		return fmt.Errorf("fitreports: list: %w", err)
	}

	vm := fitReportListViewModel{}
	for _, fr := range reports {
		vm.FitReports = append(vm.FitReports, fitReportListRow{
			ID:      fr.ID,
			Company: fr.CompanyName,
			Role:    fr.RoleTitle,
			Verdict: fr.Verdict,
			Date:    fr.CreatedAt,
		})
	}

	return h.render.Page(c, "fitreports/list.html", vm)
}

// fitSignalView is the view-model for a single signal card.
type fitSignalView struct {
	Requirement string
	Evidence    string
	Source      string
}

// fitReportDetailView is the view-model for the main column of
// fitreports/detail.html.
type fitReportDetailView struct {
	ID           string
	Company      string
	Role         string
	Location     string
	Level        string
	Verdict      string
	Summary      string
	PostingURL   string
	Date         string
	MatchSignals []fitSignalView
	Gaps         []fitSignalView
	RedFlags     []fitSignalView
	WhyApply     string
}

// researchSidebarView is the view-model for the research sidebar shared by
// fitreports/detail.html and research/detail.html.
type researchSidebarView struct {
	Stage              string
	Headcount          string
	Founded            string
	Remote             string
	StabilityRating    string
	StabilityNotes     string
	SalaryRange        string
	SalarySource       string
	TotalComp          string
	VsFloor            string
	VsFloorClass       string
	EngineeringCulture string
	TechStack          []string
}

type fitReportDetailViewModel struct {
	FitReport fitReportDetailView
	Research  *researchSidebarView
}

// Detail handles GET /fit-reports/:id.
func (h *FitReportHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	fr, err := h.fitReports.FindByID(id)
	if err != nil {
		return fmt.Errorf("fitreports: find by id %q: %w", id, err)
	}
	if fr == nil {
		return fiber.NewError(fiber.StatusNotFound, "fit report not found")
	}

	detail := fitReportDetailView{
		ID:      fr.ID,
		Company: fr.CompanyName,
		Role:    fr.RoleTitle,
		Verdict: fr.Verdict,
		Summary: fr.VerdictSummary,
		Date:    fr.CreatedAt,
	}
	if fr.Location != nil {
		detail.Location = *fr.Location
	}
	if fr.Level != nil {
		detail.Level = *fr.Level
	}
	if fr.PostingURL != nil {
		detail.PostingURL = *fr.PostingURL
	}
	if fr.WhyApply != nil && *fr.WhyApply != "" {
		detail.WhyApply = *fr.WhyApply
	}

	for _, s := range fr.Signals {
		sv := signalToView(s)
		switch s.Kind {
		case "match":
			detail.MatchSignals = append(detail.MatchSignals, sv)
		case "gap":
			detail.Gaps = append(detail.Gaps, sv)
		case "flag":
			detail.RedFlags = append(detail.RedFlags, sv)
		}
	}

	vm := fitReportDetailViewModel{FitReport: detail}
	if fr.Research != nil {
		vm.Research = buildResearchSidebarView(fr.Research)
	}

	return h.render.Page(c, "fitreports/detail.html", vm)
}

func signalToView(s models.FitSignal) fitSignalView {
	sv := fitSignalView{Requirement: s.Requirement, Evidence: s.Evidence}
	if s.Source != nil {
		sv.Source = *s.Source
	}
	return sv
}
