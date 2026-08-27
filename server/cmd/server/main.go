// Command server runs the job search dashboard HTTP server: an HTMX-driven
// Fiber app backed by SQLite, plus a JSON API consumed by the /job skill.
package main

import (
	"log"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"

	"github.com/6cclab/jobhub/internal/config"
	"github.com/6cclab/jobhub/internal/db"
	"github.com/6cclab/jobhub/internal/eval"
	"github.com/6cclab/jobhub/internal/handler"
	"github.com/6cclab/jobhub/internal/render"
	"github.com/6cclab/jobhub/internal/repository"
	"github.com/6cclab/jobhub/migrations"
	staticassets "github.com/6cclab/jobhub/web/static"
	templatefiles "github.com/6cclab/jobhub/web/templates"
)

func main() {
	cfg := config.Load()

	sqlDB, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer sqlDB.Close()

	if err := db.Migrate(sqlDB, migrations.FS); err != nil {
		log.Fatalf("migrate db: %v", err)
	}

	// Repositories
	companyRepo := repository.NewCompanyRepo(sqlDB)
	boardRepo := repository.NewBoardRepo(sqlDB)
	researchRepo := repository.NewResearchRepo(sqlDB)
	fitReportRepo := repository.NewFitReportRepo(sqlDB)
	searchResultRepo := repository.NewSearchResultRepo(sqlDB)
	applicationRepo := repository.NewApplicationRepo(sqlDB)
	evalResultRepo := repository.NewEvalResultRepo(sqlDB)

	// Renderer
	renderer := render.NewRenderer(templatefiles.FS)

	// Handlers
	dashboardHandler := handler.NewDashboardHandler(fitReportRepo, applicationRepo, boardRepo, renderer)
	fitReportHandler := handler.NewFitReportHandler(fitReportRepo, renderer)
	searchResultHandler := handler.NewSearchResultHandler(searchResultRepo, renderer)
	applicationHandler := handler.NewApplicationHandler(applicationRepo, renderer)
	researchHandler := handler.NewResearchHandler(researchRepo, renderer)
	boardHandler := handler.NewBoardHandler(boardRepo, renderer)
	evalCfg, err := eval.LoadConfig("../user/eval-config.yaml")
	if err != nil {
		log.Printf("eval config not found, using defaults: %v", err)
		evalCfg = eval.DefaultConfig()
	}
	evalEngine := eval.New(evalCfg)
	evalHandler := handler.NewEvalHandler(evalResultRepo, renderer, evalEngine)
	apiHandler := handler.NewAPIHandler(
		companyRepo, boardRepo, researchRepo, fitReportRepo, searchResultRepo, applicationRepo,
	)

	app := fiber.New(fiber.Config{
		AppName: "jobhub",
	})

	app.Use(logger.New())

	// Static assets (CSS/JS) served from the embedded web/static directory.
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.FS(staticassets.FS),
		Browse: false,
	}))

	// Health check
	app.Get("/healthz", handler.Health)

	// Page routes
	app.Get("/", dashboardHandler.Show)
	app.Get("/fit-reports", fitReportHandler.List)
	app.Get("/fit-reports/:id", fitReportHandler.Detail)
	app.Get("/search-results", searchResultHandler.List)
	app.Get("/search-results/:id", searchResultHandler.Detail)
	app.Get("/search-results/:id/rows", searchResultHandler.Rows)
	app.Get("/applications", applicationHandler.List)
	app.Get("/applications/:id/resume", applicationHandler.Resume)
	app.Get("/research", researchHandler.List)
	app.Get("/research/:id", researchHandler.Detail)
	app.Get("/boards", boardHandler.List)
	app.Get("/eval-results", evalHandler.List)
	app.Get("/eval-results/compare", evalHandler.Compare)
	app.Get("/eval-results/:id", evalHandler.Detail)

	// JSON API routes, consumed by the /job skill.
	api := app.Group("/api", handler.BearerAuth(cfg.APIToken))
	api.Post("/fit-reports", apiHandler.CreateFitReport)
	api.Patch("/fit-reports/:id", apiHandler.UpdateFitReport)
	api.Post("/search-results", apiHandler.CreateSearchResults)
	api.Get("/applications", apiHandler.ListApplications)
	api.Get("/applications/:id", apiHandler.GetApplication)
	api.Post("/applications", apiHandler.CreateApplication)
	api.Patch("/applications/:id", apiHandler.UpdateApplication)
	api.Post("/applications/:id/resume", apiHandler.UploadResume)
	// Read path for the stored PDF. The same handler is mounted at
	// /applications/:id/resume for the dashboard, but that route sits behind the
	// auth proxy and is unreachable with a bearer token -- which left the upload's
	// 201 as the only evidence it had worked. A write with no read path cannot be
	// verified, so this mounts the read under /api too.
	api.Get("/applications/:id/resume", applicationHandler.Resume)
	api.Post("/research", apiHandler.CreateResearch)
	api.Get("/boards", apiHandler.ListBoards)
	api.Post("/boards", apiHandler.CreateBoard)
	api.Post("/eval", evalHandler.RunEval)
	api.Get("/eval-results", evalHandler.ListEvalResults)
	api.Get("/eval-results/:id", evalHandler.GetEvalResult)

	log.Printf("jobhub listening on :%s", cfg.Port)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
