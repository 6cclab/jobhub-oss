package handler

import (
	"fmt"

	"github.com/gofiber/fiber/v2"

	"github.com/6cclab/jobhub/internal/models"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
)

// SearchResultHandler serves search batch list/detail pages and the HTMX
// filtered-rows partial.
type SearchResultHandler struct {
	searchResults *repository.SearchResultRepo
	render        *render.Renderer
}

// NewSearchResultHandler constructs a SearchResultHandler.
func NewSearchResultHandler(searchResults *repository.SearchResultRepo, renderer *render.Renderer) *SearchResultHandler {
	return &SearchResultHandler{searchResults: searchResults, render: renderer}
}

// searchBatchRow is the view-model for a row in searchresults/list.html.
type searchBatchRow struct {
	ID            string
	Date          string
	BoardsScanned int
	TopPicks      int
	CompanyCount  int
}

type searchResultListViewModel struct {
	Batches []searchBatchRow
}

// List handles GET /search-results.
func (h *SearchResultHandler) List(c *fiber.Ctx) error {
	batches, err := h.searchResults.ListBatches()
	if err != nil {
		return fmt.Errorf("searchresults: list batches: %w", err)
	}

	vm := searchResultListViewModel{}
	for _, b := range batches {
		vm.Batches = append(vm.Batches, searchBatchRow{
			ID:            b.ID,
			Date:          b.RanAt,
			BoardsScanned: b.BoardCount,
			TopPicks:      b.ResultCount,
			CompanyCount:  b.ResultCount,
		})
	}

	return h.render.Page(c, "searchresults/list.html", vm)
}

// searchResultRowView is the view-model for a row in
// partials/_search_result_row.html.
type searchResultRowView struct {
	Company         string
	Role            string
	URL             string
	Location        string
	IsRemote        bool
	Salary          string
	SalaryDisclosed bool
	BelowFloor      bool
	Tags            []string
}

// searchBatchDetailView is the view-model for the header of
// searchresults/detail.html.
type searchBatchDetailView struct {
	ID             string
	Date           string
	BoardsScanned  int
	TopPicks       int
	CompanyCount   int
	LocationFilter string
}

type searchResultDetailViewModel struct {
	Batch     searchBatchDetailView
	StrongFit []searchResultRowView
	GoodFit   []searchResultRowView
}

// Detail handles GET /search-results/:id.
func (h *SearchResultHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	batch, results, err := h.searchResults.FindBatchByID(id)
	if err != nil {
		return fmt.Errorf("searchresults: find batch %q: %w", id, err)
	}
	if batch == nil {
		return fiber.NewError(fiber.StatusNotFound, "search batch not found")
	}

	companies := make(map[string]struct{})
	var strongFit, goodFit []searchResultRowView
	for _, res := range results {
		companies[res.CompanyID] = struct{}{}
		row := searchResultToRowView(res)
		if res.FitTier == "strong" {
			strongFit = append(strongFit, row)
		} else {
			goodFit = append(goodFit, row)
		}
	}

	vm := searchResultDetailViewModel{
		Batch: searchBatchDetailView{
			ID:             batch.ID,
			Date:           batch.RanAt,
			BoardsScanned:  batch.BoardCount,
			TopPicks:       len(strongFit),
			CompanyCount:   len(companies),
			LocationFilter: stringOrEmpty(batch.LocationFilter),
		},
		StrongFit: strongFit,
		GoodFit:   goodFit,
	}

	return h.render.Page(c, "searchresults/detail.html", vm)
}

// Rows handles GET /search-results/:id/rows, an HTMX partial returning
// filtered <tr> rows. Query params: tier, loc, level, domain. A value of
// "all" (or empty) is treated as "no filter" for that field.
func (h *SearchResultHandler) Rows(c *fiber.Ctx) error {
	id := c.Params("id")

	tier := queryFilter(c, "tier")
	loc := queryFilter(c, "loc")
	level := queryFilter(c, "level")
	domain := queryFilter(c, "domain")

	// "remote" is a special-cased location filter value meaning "only
	// remote roles", since location is otherwise a substring match against
	// free-text location strings.
	var results []models.SearchResult
	var err error
	if loc != nil && *loc == "remote" {
		results, err = h.searchResults.FilterResults(id, tier, nil, level, domain)
		if err == nil {
			results = filterRemoteOnly(results)
		}
	} else {
		results, err = h.searchResults.FilterResults(id, tier, loc, level, domain)
	}
	if err != nil {
		return fmt.Errorf("searchresults: filter results for batch %q: %w", id, err)
	}

	rows := make([]searchResultRowView, 0, len(results))
	for _, res := range results {
		rows = append(rows, searchResultToRowView(res))
	}

	// partials/_search_result_row.html renders a single <tr> per invocation
	// (it's also used via {{range}} from searchresults/detail.html), so for
	// this HTMX fragment endpoint we render each row and concatenate them
	// into one response.
	return h.render.PartialEach(c, "partials/_search_result_row.html", rows)
}

func queryFilter(c *fiber.Ctx, name string) *string {
	v := c.Query(name)
	if v == "" || v == "all" {
		return nil
	}
	return &v
}

func filterRemoteOnly(results []models.SearchResult) []models.SearchResult {
	out := make([]models.SearchResult, 0, len(results))
	for _, r := range results {
		if r.IsRemote {
			out = append(out, r)
		}
	}
	return out
}

func searchResultToRowView(res models.SearchResult) searchResultRowView {
	row := searchResultRowView{
		Company:         res.CompanyName,
		Role:            res.RoleTitle,
		URL:             res.PostingURL,
		IsRemote:        res.IsRemote,
		SalaryDisclosed: res.SalaryDisclosed,
		BelowFloor:      res.BelowFloor,
		Tags:            splitCommaList(res.Tags),
	}
	if res.Location != nil {
		row.Location = *res.Location
	}
	row.Salary = formatSalaryRange(res.SalaryMin, res.SalaryMax, res.SalaryDisclosed)
	return row
}

func formatSalaryRange(min, max *int, disclosed bool) string {
	if !disclosed || (min == nil && max == nil) {
		return "Undisclosed"
	}
	switch {
	case min != nil && max != nil:
		return fmt.Sprintf("$%dk–$%dk", *min/1000, *max/1000)
	case min != nil:
		return fmt.Sprintf("$%dk+", *min/1000)
	case max != nil:
		return fmt.Sprintf("up to $%dk", *max/1000)
	default:
		return "Undisclosed"
	}
}
