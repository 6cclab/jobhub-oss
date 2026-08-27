# Command Reference

All commands live in `prompts/commands/` and are invoked as `/command-name [args]` in whichever agent you wired up (see `AGENTS.md`). They share the JobHub server config: `JOBHUB_URL` (default `http://localhost:8080`) and `JOBHUB_API_TOKEN` (optional, only needed if the server was started with `API_TOKEN` set).

## /job

The main assistant. One command, several modes — it detects intent from what you give it and acts accordingly.

**Modes:**

- **Fit evaluation** (always runs first when you share a posting) — assesses match signals, gaps, and red flags against your background, assigns a verdict (`strong` / `worth` / `stretch` / `skip`), and posts a fit report to the dashboard. Won't offer to tailor a resume for `stretch`/`skip` roles unless you explicitly ask.
- **Resume tailoring** (after you confirm you want to apply) — selects and reorders bullets from your master resume, pulls in personal-project bullets to fill gaps the posting demands but your work history doesn't cover, writes a tailored resume, runs it through the **eval gate** (hard gate, see below), and generates a PDF via WeasyPrint.
- **Cover letter** — writes a cover letter following your `cover-letter-style.md` structure (hook → technical → metrics → leadership → honest gap → close) and generates a PDF.
- **Application questions** — drafts answers to free-text application questions in your voice, then runs every answer through a mandatory Haiku tone-review subagent before showing it to you (violations are fixed silently, never shown raw).
- **Job search** — pulls postings from your tracked Greenhouse boards, filters/classifies them against your preferences, and posts a batch of search results to the dashboard.
- **Application tracking** — logs applications and status changes to the server.
- **Strategy discussion** — open-ended conversation about salary, targeting, and preferences; updates `user/preferences.md` if the conversation surfaces new information.
- **Adding/discovering boards** — probes Greenhouse slugs and tracks new companies.

**Reads:** `user/config.yaml`, `user/preferences.md`, `user/master-resume.md`, `user/personal-projects.md`, `user/resume-style.md`, `user/cover-letter-style.md`, `user/communication-style.md`, `user/boards.md` (discovery seed list only — tracked boards come from the server).

**Writes:** `user/tailored/` (resumes, cover letters, PDFs), `user/research/` (dual-write fallback), `user/applications.md` (dual-write fallback), `user/boards.md` (dual-write fallback). POSTs to `/api/fit-reports`, `/api/eval`, `/api/search-results`, `/api/applications`, `/api/boards`.

**Customize by editing:**
- `user/resume-style.md` — voice rules, banned phrases, bullet formatting
- `user/preferences.md` — target roles, salary floor, deal-breakers (drives which postings get filtered out and which get flagged as red flags)
- `user/communication-style.md` — your writing voice, used for application answers and cover letters
- `user/master-resume.md` and `user/personal-projects.md` — your evidence pool; the "Skill gaps this fills" sections in the latter directly control which personal-project bullets get pulled in during gap-fill

## /job-eval

Standalone resume quality evaluator. Runs the same deterministic eval engine + adversarial verification that `/job`'s eval gate uses, but as a manual, on-demand check — useful for auditing resumes that already went out, or catching systemic issues across your whole `user/tailored/` directory.

**Usage:**
- `/job-eval` or `/job-eval all` — evaluate every `.md` resume in `user/tailored/`, produce a scorecard, flag systemic failures
- `/job-eval vanta` (or any company name) — evaluate just that company's resume(s)
- `/job-eval latest` — evaluate the most recently modified resume

**Process:** recovers the original job posting (from a cached fit report, `_fit.html` file, or `applications.md`), extracts posting terms, builds the project-gaps map from `personal-projects.md`, POSTs to `/api/eval`, runs adversarial verification (single-resume mode only — skipped in batch), and prints a keyword diff / gap-fill check / style compliance / skills relevance / structural breakdown with a verdict.

**Reads:** `user/master-resume.md`, `user/personal-projects.md`, `user/resume-style.md`, `user/preferences.md`, resumes under `user/tailored/`.

**Writes:** nothing directly — it only reads and reports. All scoring happens via `POST /api/eval`, and results are viewable at `$JOBHUB_URL/eval-results/{id}`.

**Customize by editing:** `user/eval-config.yaml` (banned phrases, common stack skills that don't count as unmatched padding, your own unverified metrics to flag).

## /company-research

Lightweight company research, scoped to job-search decisions (not general deep research — see the separate `deep-research` skill for that). Cheap by design: runs on Sonnet, not Opus.

**Usage:** `/company-research {company name}` — optionally mention a specific role you're evaluating.

**Process:** fans out 3 research passes (parallel subagents where supported, otherwise sequential):
1. **Stability & financials** — funding stage, layoffs, headcount trend, leadership changes → stability verdict (Strong/Stable/Caution/Avoid)
2. **Engineering culture & tech stack** — engineering blog, Glassdoor/Reddit/Blind sentiment, remote policy, interview process
3. **Compensation** — Levels.fyi, Glassdoor, Blind, public salary bands for the target level

Synthesizes all three into a Company Brief (overview, stability, culture, compensation, red flags, bottom line).

**Writes:** `user/research/{company}.md` locally, then POSTs to `POST /api/research` (upserts by company name — re-running updates the existing brief). If a fit report for the same company is generated in the same session, the server automatically links the research brief into that fit report's dashboard view.

**Customize by editing:** nothing config-driven — to change the research angle, edit the three research prompts directly in `prompts/commands/company-research.md`.

## /app-review

Tone review for application text — old-org shots, project pitching, self-deprecation, banned phrases, em-dash overuse. `/job` already runs this automatically (via a Haiku subagent) before showing you any drafted application answer; `/app-review` is the manual way to check text yourself (e.g., something you wrote by hand, or want a second pass on).

**Usage:** `/app-review {paste text}` — or with no argument, reviews the most recently drafted application answer in the conversation.

**Checks:**
1. **Old-org shots** — phrases implying your current/previous employer is worse than the target company ("not a cost center," "somewhere that actually X," etc.)
2. **Project pitching** — does the answer read like a showcase of what you built instead of an answer to the actual question?
3. **Self-deprecation** — personal anecdotes that frame you as having a shortcoming, rather than the tools/systems being bad
4. **Banned phrases** — "passionate about," "leveraging," "excited to," "robust solutions," "synergy," "track record of," "proven ability to," "demonstrated experience in," "results-driven," "self-motivated," "detail-oriented"
5. **Excessive em-dashes** — more than one in the text

**Writes:** nothing — prints pass/fail with quoted violations and a corrected version if it fails.

**Customize by editing:** the violation checklist directly in `prompts/commands/app-review.md` (there's no external config file for this one — it's a fixed Haiku prompt).

## /onboard

Interactive setup. Creates your `user/` directory from `templates/` so `/job`, `/job-eval`, and `/company-research` have data to work with.

**Process:**
1. Checks whether `user/config.yaml` already exists (asks before overwriting)
2. Asks for identity info (name, email, phone, location, LinkedIn, GitHub, sign-off name) → writes `user/config.yaml`
3. Copies every template file from `templates/` to `user/` (master resume, preferences, resume style, cover letter style, communication style, personal projects, boards, applications, eval config), creates `user/tailored/`, `user/research/`, `user/resumes/`
4. Optionally walks through filling in your master resume interactively (role, company history, top accomplishments, education, tech stack)
5. Optionally walks through setting preferences (target level, salary floor, location, industries, tech stack, deal-breakers)
6. Prints next steps: start the server, try `/job` with a posting

**Reads:** nothing pre-existing except checking for `user/config.yaml`.

**Writes:** `user/config.yaml`, plus a full copy of `templates/*` into `user/*`.

If you'd rather skip the interactive flow and fill in files by hand, see [docs/onboarding.md](onboarding.md) for a field-by-field guide.

## Scripts

- `scripts/build_application_index.py` — regenerates `user/applications.md` from the records.
  `--check` exits 1 if stale. Run it after any record edit.
- `scripts/migrate_applications.py` — one-shot table→records migration. Kept for reference; it
  should never need to run again.
