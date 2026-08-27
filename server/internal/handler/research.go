package handler

import (
	"fmt"
	"html/template"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// ResearchHandler serves company research list/detail pages.
type ResearchHandler struct {
	research *repository.ResearchRepo
	render   *render.Renderer
}

// NewResearchHandler constructs a ResearchHandler.
func NewResearchHandler(research *repository.ResearchRepo, renderer *render.Renderer) *ResearchHandler {
	return &ResearchHandler{research: research, render: renderer}
}

// researchListRow is the view-model for a row in research/list.html.
type researchListRow struct {
	ID              string
	Company         string
	StabilityRating string
	SalaryRange     string
	Remote          string
	ResearchedDate  string
}

type researchListViewModel struct {
	Research []researchListRow
}

// List handles GET /research.
func (h *ResearchHandler) List(c *fiber.Ctx) error {
	briefs, err := h.research.List()
	if err != nil {
		return fmt.Errorf("research: list: %w", err)
	}

	vm := researchListViewModel{}
	for _, rb := range briefs {
		vm.Research = append(vm.Research, researchListRow{
			ID:              rb.ID,
			Company:         rb.CompanyName,
			StabilityRating: stringOrEmpty(rb.StabilityVerdict),
			SalaryRange:     stringOrEmpty(rb.SalaryRangeText),
			Remote:          stringOrEmpty(rb.RemotePolicy),
			ResearchedDate:  rb.ResearchedAt,
		})
	}

	return h.render.Page(c, "research/list.html", vm)
}

// researchDetailView is the view-model for the main column of
// research/detail.html.
type researchDetailView struct {
	ID                 string
	Company            string
	ResearchedDate     string
	RenderedMarkdown   template.HTML
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

type researchDetailViewModel struct {
	Research researchDetailView
}

// Detail handles GET /research/:id.
func (h *ResearchHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	rb, err := h.research.FindByID(id)
	if err != nil {
		return fmt.Errorf("research: find by id %q: %w", id, err)
	}
	if rb == nil {
		return fiber.NewError(fiber.StatusNotFound, "research brief not found")
	}

	vm := researchDetailViewModel{
		Research: researchDetailView{
			ID:                 rb.ID,
			Company:            rb.CompanyName,
			ResearchedDate:     rb.ResearchedAt,
			RenderedMarkdown:   template.HTML(h.render.RenderMarkdown(rb.RawMarkdown)), //nolint:gosec // trusted operator-authored markdown
			Stage:              stringOrEmpty(rb.Stage),
			Headcount:          stringOrEmpty(rb.Headcount),
			Founded:            stringOrEmpty(rb.Founded),
			Remote:             stringOrEmpty(rb.RemotePolicy),
			StabilityRating:    stringOrEmpty(rb.StabilityVerdict),
			StabilityNotes:     stringOrEmpty(rb.StabilityNotes),
			SalaryRange:        stringOrEmpty(rb.SalaryRangeText),
			SalarySource:       stringOrEmpty(rb.SalarySource),
			EngineeringCulture: stringOrEmpty(rb.CultureNotes),
			TechStack:          splitTechStack(rb.TechStack),
		},
	}

	return h.render.Page(c, "research/detail.html", vm)
}

// buildResearchSidebarView converts a models.ResearchBrief into the sidebar
// view-model shared by fit report and research detail pages.
func buildResearchSidebarView(rb *models.ResearchBrief) *researchSidebarView {
	return &researchSidebarView{
		Stage:              stringOrEmpty(rb.Stage),
		Headcount:          stringOrEmpty(rb.Headcount),
		Founded:            stringOrEmpty(rb.Founded),
		Remote:             stringOrEmpty(rb.RemotePolicy),
		StabilityRating:    stringOrEmpty(rb.StabilityVerdict),
		StabilityNotes:     stringOrEmpty(rb.StabilityNotes),
		SalaryRange:        stringOrEmpty(rb.SalaryRangeText),
		SalarySource:       stringOrEmpty(rb.SalarySource),
		EngineeringCulture: stringOrEmpty(rb.CultureNotes),
		TechStack:          splitTechStack(rb.TechStack),
	}
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func splitTechStack(techStack *string) []string {
	if techStack == nil || *techStack == "" {
		return nil
	}
	return splitCommaList(*techStack)
}
