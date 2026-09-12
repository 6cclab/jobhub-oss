package handler

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// ApplicationHandler serves the applications list page and resume downloads.
type ApplicationHandler struct {
	apps   *repository.ApplicationRepo
	render *render.Renderer
}

// NewApplicationHandler constructs an ApplicationHandler.
func NewApplicationHandler(apps *repository.ApplicationRepo, renderer *render.Renderer) *ApplicationHandler {
	return &ApplicationHandler{apps: apps, render: renderer}
}

// applicationRowView is the view-model for partials/_application_row.html.
type applicationRowView struct {
	ID          string
	Company     string
	Role        string
	Source      string
	Status      string
	AppliedDate string
	ResumePath  string
	Notes       string
}

type applicationListViewModel struct {
	Applications []applicationRowView
}

// List handles GET /applications.
func (h *ApplicationHandler) List(c *fiber.Ctx) error {
	apps, err := h.apps.List()
	if err != nil {
		return fmt.Errorf("applications: list: %w", err)
	}

	vm := applicationListViewModel{}
	for _, app := range apps {
		status := "applied"
		if app.Status != nil {
			status = *app.Status
		}
		appliedDate := app.CreatedAt
		if app.AppliedAt != nil {
			appliedDate = *app.AppliedAt
		}
		row := applicationRowView{
			ID:          app.ID,
			Company:     app.CompanyName,
			Role:        app.RoleTitle,
			Source:      stringOrEmpty(app.Source),
			Status:      status,
			AppliedDate: appliedDate,
			Notes:       stringOrEmpty(app.Notes),
		}
		if app.HasResume {
			row.ResumePath = fmt.Sprintf("/applications/%s/resume", app.ID)
		}
		vm.Applications = append(vm.Applications, row)
	}

	return h.render.Page(c, "applications/list.html", vm)
}

// Resume handles GET /applications/:id/resume, streaming the stored PDF
// resume as an attachment download.
func (h *ApplicationHandler) Resume(c *fiber.Ctx) error {
	id := c.Params("id")

	data, filename, err := h.apps.GetResume(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fiber.NewError(fiber.StatusNotFound, "application not found")
		}
		return fmt.Errorf("applications: get resume for %q: %w", id, err)
	}
	if len(data) == 0 {
		return fiber.NewError(fiber.StatusNotFound, "no resume stored for this application")
	}

	if filename == "" {
		filename = "resume.pdf"
	}

	c.Set(fiber.HeaderContentType, "application/pdf")
	c.Set(fiber.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))
	return c.Send(data)
}
