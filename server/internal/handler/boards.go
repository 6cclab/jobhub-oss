package handler

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// BoardHandler serves the boards list page, grouped by ATS.
type BoardHandler struct {
	boards *repository.BoardRepo
	render *render.Renderer
}

// NewBoardHandler constructs a BoardHandler.
func NewBoardHandler(boards *repository.BoardRepo, renderer *render.Renderer) *BoardHandler {
	return &BoardHandler{boards: boards, render: renderer}
}

// boardRowView is the view-model for a row in boards/list.html.
type boardRowView struct {
	Slug        string
	Name        string
	Tags        []string
	Status      string
	StatusClass string
	LastProbed  string
}

type boardListViewModel struct {
	Greenhouse []boardRowView
	Lever      []boardRowView
	Ashby      []boardRowView
}

// List handles GET /boards.
func (h *BoardHandler) List(c *fiber.Ctx) error {
	boards, err := h.boards.List()
	if err != nil {
		return fmt.Errorf("boards: list: %w", err)
	}

	vm := boardListViewModel{}
	for _, b := range boards {
		row := boardToRowView(b)
		switch strings.ToLower(b.ATS) {
		case "greenhouse":
			vm.Greenhouse = append(vm.Greenhouse, row)
		case "lever":
			vm.Lever = append(vm.Lever, row)
		case "ashby":
			vm.Ashby = append(vm.Ashby, row)
		}
	}

	return h.render.Page(c, "boards/list.html", vm)
}

func boardToRowView(b models.Board) boardRowView {
	row := boardRowView{
		Slug:        b.Slug,
		Name:        b.Name,
		Tags:        splitCommaList(b.Tags),
		Status:      b.Status,
		StatusClass: boardStatusClass(b.Status),
	}
	if b.LastProbedAt != nil {
		row.LastProbed = *b.LastProbedAt
	} else {
		row.LastProbed = "never"
	}
	return row
}

// boardStatusClass maps a board status (one of: tracked, discovery, dead —
// see boards.status CHECK constraint) to a CSS class suffix reusing the
// stability badge palette.
func boardStatusClass(status string) string {
	switch strings.ToLower(status) {
	case "tracked":
		return "stable"
	case "discovery":
		return "caution"
	case "dead":
		return "avoid"
	default:
		return strings.ToLower(status)
	}
}
