package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// DashboardHandler serves the top-level dashboard page.
type DashboardHandler struct {
	fitReports *repository.FitReportRepo
	apps       *repository.ApplicationRepo
	boards     *repository.BoardRepo
	render     *render.Renderer
}

// NewDashboardHandler constructs a DashboardHandler.
func NewDashboardHandler(
	fitReports *repository.FitReportRepo,
	apps *repository.ApplicationRepo,
	boards *repository.BoardRepo,
	renderer *render.Renderer,
) *DashboardHandler {
	return &DashboardHandler{
		fitReports: fitReports,
		apps:       apps,
		boards:     boards,
		render:     renderer,
	}
}

// dashboardFitReportRow is the view-model for a row in the dashboard's
// "Recent Fit Reports" table.
type dashboardFitReportRow struct {
	ID      string
	Company string
	Role    string
	Verdict string
	Date    string
}

// dashboardApplicationRow is the view-model for a row in the dashboard's
// "Recent Applications" table.
type dashboardApplicationRow struct {
	ID          string
	Company     string
	Role        string
	Status      string
	AppliedDate string
}

// dashboardViewModel is the data passed to templates/dashboard.html.
type dashboardViewModel struct {
	FitReportCount     int
	ApplicationCount   int
	TrackedBoardCount  int
	RecentFitReports   []dashboardFitReportRow
	RecentApplications []dashboardApplicationRow
}

// Show handles GET /.
func (h *DashboardHandler) Show(c *fiber.Ctx) error {
	recentReports, err := h.fitReports.Recent(5)
	if err != nil {
		return fmt.Errorf("dashboard: load recent fit reports: %w", err)
	}

	allReports, err := h.fitReports.List()
	if err != nil {
		return fmt.Errorf("dashboard: count fit reports: %w", err)
	}

	statusCounts, err := h.apps.CountByStatus()
	if err != nil {
		return fmt.Errorf("dashboard: count applications by status: %w", err)
	}
	totalApps := 0
	for _, n := range statusCounts {
		totalApps += n
	}

	allApps, err := h.apps.List()
	if err != nil {
		return fmt.Errorf("dashboard: load applications: %w", err)
	}

	trackedBoards, err := h.boards.ListByStatus("tracked")
	if err != nil {
		return fmt.Errorf("dashboard: count tracked boards: %w", err)
	}

	vm := dashboardViewModel{
		FitReportCount:    len(allReports),
		ApplicationCount:  totalApps,
		TrackedBoardCount: len(trackedBoards),
	}

	for _, fr := range recentReports {
		vm.RecentFitReports = append(vm.RecentFitReports, dashboardFitReportRow{
			ID:      fr.ID,
			Company: fr.CompanyName,
			Role:    fr.RoleTitle,
			Verdict: fr.Verdict,
			Date:    fr.CreatedAt,
		})
	}

	recentAppLimit := 5
	if len(allApps) < recentAppLimit {
		recentAppLimit = len(allApps)
	}
	for _, app := range allApps[:recentAppLimit] {
		status := "applied"
		if app.Status != nil {
			status = *app.Status
		}
		appliedDate := app.CreatedAt
		if app.AppliedAt != nil {
			appliedDate = *app.AppliedAt
		}
		vm.RecentApplications = append(vm.RecentApplications, dashboardApplicationRow{
			ID:          app.ID,
			Company:     app.CompanyName,
			Role:        app.RoleTitle,
			Status:      status,
			AppliedDate: appliedDate,
		})
	}

	return h.render.Page(c, "dashboard.html", vm)
}
