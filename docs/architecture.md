# Architecture

## System Overview

JobHub splits work between two layers:

- **Agent commands** (`prompts/commands/*.md`) do the reasoning — reading your personal data, evaluating job postings, tailoring resumes, checking tone. They have no persistence of their own.
- **The Go server** (`server/`) is the system of record. Commands POST structured JSON to its API, it writes to Postgres, and it renders an HTMX dashboard so you can browse everything in a browser.

```
agent command ──curl POST──> /api/*  ──> repository layer ──> Postgres
                                       │
                                       └──> returns {"id": ..., "url": "/fit-reports/{id}"}

Browser ──GET /fit-reports/{id}──> handler ──> repository ──> HTMX template ──> HTML
```

Commands never talk to Postgres directly — they only speak JSON over HTTP to the server's `/api/*` routes, and read HTML pages back through the browser-facing routes for humans.

## Server Stack

- **Go 1.26** + **Fiber v2** — HTTP routing and middleware
- **HTMX** — server-rendered dashboard pages; partial updates via HTML fragments (`/search-results/:id/rows`), no JS framework or build step
- **PostgreSQL** via `jackc/pgx/v5` — the persistence layer (`server/internal/db/db.go`). `DATABASE_URL` defaults to `postgres://jobhub:jobhub@localhost:5432/jobhub?sslmode=disable`
- **golang-migrate** — schema migrations in `server/migrations/*.sql`, embedded via `embed.FS` and applied automatically on server startup (see `db.Migrate` in `main.go`)
- **Embedded templates and static assets** — `server/web/templates` (HTML, via Go's `html/template` through `internal/render`) and `server/web/static` (CSS/JS) are compiled into the binary with `embed.FS`, so the server ships as a single binary with no external asset directory to deploy
- **Bearer auth** — `internal/handler/auth.go` gates the `/api/*` group with `BearerAuth(cfg.APIToken)`. If `API_TOKEN` is unset, auth is a no-op (local dev mode)

## Eval Engine

The eval engine (`server/internal/eval/`) is a deterministic scoring pipeline — no LLM calls inside the engine itself. It takes a structured `EvalRequest` (posting terms, resume text, skills list, project gap map, style input) and runs it through five independent checks:

1. **Keyword matching** (`matchTerms` / `scoreKeywords`) — for each posting term, checks whether it (or a known synonym, see `synonyms.go`) appears in the resume body (`Covered`), only in the skills list (`SkillsOnly`), or not at all (`Missing`). Coverage percentage drives the score: ≥80% pass, 60–79% warn, <60% fail.
2. **Gap-fill checking** (`checkGapFill`) — for every term not covered by a work bullet, checks whether a personal project's "Skill gaps this fills" section could cover it (via the `project_gaps` map the command builds). If a project could fill the gap but wasn't used, that's a `GapFillFailure`. Any top-priority (`top3`) gap-fill failure forces the eval to `Fail`.
3. **Style validation** (`checkStyle`) — summary must open in first person ("I "/"I'"), must not contain any `banned_phrases` from `eval-config.yaml`, must not contain any of your `unverified_metrics`, and resume length must land in a reasonable character range (1500–4500 chars).
4. **Skills relevance** (`checkSkillsRelevance`) — flags skills listed on the resume that don't map to any posting term and aren't in `common_stack_skills` (table-stakes tech nobody expects tailored). Also counts `SkillsOnly` terms (claimed in the skills list but not demonstrated in a bullet).
5. **Structural checks** (`checkStructural`) — title line must match the expected seniority level; flags missing education or prior-roles sections.

`deriveVerdict` rolls the five dimension scores into one verdict: `Strong`, `Acceptable`, `NeedsRework`, or `Critical`. Any top-3 gap-fill failure is an automatic `Critical`. Any hard `Fail` on a dimension is `NeedsRework`. Three or more `Warn`s is also `NeedsRework`. Otherwise `Acceptable` (with warnings) or `Strong` (clean).

**Adversarial verification layer:** the deterministic engine only checks for term presence, which is gameable (a term can appear in text without real evidence). After the engine returns `Covered` terms, `/job` and `/job-eval` spawn a Sonnet subagent instructed to default to REFUTED unless the resume text is clearly demonstrative evidence, not just a mention. Any refuted term is downgraded from `Covered` to `SkillsOnly` via `Engine.ApplyAdversarialDowngrades`, and all scores are recalculated with the same thresholds. This is a second pass on top of the engine, not part of it — the engine itself has no model calls.

**Configuration:** the engine loads `user/eval-config.yaml` at server startup (`eval.LoadConfig`, called from `main.go` with a relative path of `../user/eval-config.yaml`). If the file is missing, it falls back to `eval.DefaultConfig()`. Three lists are configurable: `common_stack_skills`, `banned_phrases`, `unverified_metrics`. See [docs/onboarding.md](onboarding.md) for details on customizing this file.

## Command → Server Interaction

Every command that produces a durable artifact (fit report, tailored resume eval, research brief, application, search results batch, tracked board) POSTs it to the server via `curl` from inside the command's Bash tool calls — there is no SDK or client library, just raw HTTP.

Pattern used throughout:

```bash
curl -s -X POST $JOBHUB_URL/api/fit-reports \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{ ... }'
```

`JOBHUB_URL` defaults to `http://localhost:8080`; `JOBHUB_API_TOKEN` is optional and only required if the server was started with `API_TOKEN` set. The server responds with an id and a dashboard URL (e.g. `{"id": "...", "url": "/fit-reports/..."}`), which the command prints for the user to open in a browser.

**Dual-write fallback:** after every successful server POST, `/job` also writes the equivalent flat file under `user/` (fit reports → `user/tailored/{company}_{role}_fit.html`, applications → append to `user/applications.md`, boards → append to `user/boards.md`, research → `user/research/{company}.md`). If the POST fails (connection refused, 500, etc.), the command still writes the flat file and warns the user that the server is down — no data is lost, it just isn't queryable in the dashboard until the server comes back and is re-synced manually.

## Directory Layout

```
job-search/
├── prompts/commands/    Command definitions (Markdown + frontmatter, harness-neutral)
├── prompts/rules/       Standing rules applied across commands
├── templates/           Starter files copied into user/ during onboarding; also the source
│                        of HTML/CSS shells (resume.html, cover-letter.html) used for PDF generation
├── user/                Gitignored personal data — identity, resume content, preferences,
│                        generated artifacts. Nothing here is committed.
├── docs/                This documentation
└── server/
    ├── cmd/server/      main.go — wires config, DB, migrations, repositories, handlers, routes
    ├── cmd/backfill/    One-off backfill utility for migrating flat-file data into Postgres
    ├── cmd/migrate-data/One-off data migration utility
    ├── internal/config/ Env var loading (PORT, DATABASE_URL, API_TOKEN)
    ├── internal/db/     Postgres connection + migration bootstrap
    ├── internal/eval/   The scoring engine described above
    ├── internal/models/ Request/response and domain structs
    ├── internal/handler/One handler file per resource (fitreports, applications, boards,
    │                    research, searchresults, eval, dashboard) plus api.go for the
    │                    JSON API and auth.go for bearer token middleware
    ├── internal/repository/ Postgres data access, one repo per resource
    ├── internal/render/ Template rendering wrapper around html/template
    ├── migrations/       SQL schema migrations (golang-migrate format), embedded and
    │                    auto-applied on startup
    └── web/
        ├── templates/    Embedded HTML templates for the dashboard (HTMX partials + pages)
        └── static/       Embedded CSS/JS
```

## Data Flow: Job Posting to PDF

1. **Job posting in** — user pastes a URL or text to `/job`. The command fetches it (`WebFetch`) if needed.
2. **Fit evaluation** — `/job` diffs the posting's requirements against `user/master-resume.md` and `user/personal-projects.md`, optionally pulling a cached company research brief from `GET /research`. Produces match/gap/flag signals and a verdict (`strong`/`worth`/`stretch`/`skip`). POSTed to `POST /api/fit-reports`, dual-written to `user/tailored/{company}_{role}_fit.html`.
3. **Resume tailoring** — if the user confirms, `/job` selects and reorders bullets from the master resume, fills skill gaps from personal projects where the posting demands something work history doesn't cover, and writes `user/tailored/{company}_{role}.md`.
4. **Eval gate (hard gate)** — the tailored resume is scored via `POST /api/eval` (deterministic engine) plus the adversarial verification passt verification pass described above. If the verdict is `NeedsRework` or `Critical`, `/job` fixes the flagged issues (adds gap-fill bullets, rewrites banned phrases) and re-runs the eval, up to 2 retries, before ever generating a PDF.
5. **PDF generation** — only after the eval gate passes. The command reads `templates/resume.html` + `templates/resume.css`, inlines the CSS, populates identity fields from `user/config.yaml` and content from the tailored resume markdown, writes the assembled HTML to `user/tailored/{company}_{role}.html`, and shells out to `weasyprint` to produce the final `.pdf`.
6. **Tracking** — the user can log the application via `POST /api/applications`, optionally attaching the generated PDF via `POST /api/applications/:id/resume`.

The same eval gate (steps 4) is available standalone via `/job-eval` for auditing resumes that already went out, or for batch-scoring everything in `user/tailored/`.
