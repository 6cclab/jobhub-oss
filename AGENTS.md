# JobHub

An AI-powered job search framework: a set of harness-neutral prompts plus a Go/HTMX dashboard.

**This file is the entry point for any coding agent.** `.claude/CLAUDE.md` points here;
Codex, Cursor, Gemini CLI, opencode, Zed and others read `AGENTS.md` directly.

## Project Structure

- `prompts/commands/` — the commands (`job`, `job-auto`, `job-eval`, `company-research`,
  `app-review`, `onboard`). Plain Markdown, no harness-specific syntax. **This is the source of
  truth.**
- `prompts/rules/` — standing rules that apply across commands.
- `.claude/commands` and `.claude/rules` — symlinks into `prompts/`. Do not edit through them;
  edit `prompts/` directly.
- `user/` — your personal data (gitignored). Created via the `onboard` command or by copying
  `templates/`.
  - `user/applications/` — one file per application. **Authoritative.**
  - `user/applications.md` — generated index. **Never hand-edit.**
  - `user/pipeline.md` — cross-application strategy. Hand-written.
- `templates/` — starter templates for all user data files.
- `server/` — Go/Fiber/HTMX dashboard with the eval engine, application tracking, and research.
- `scripts/` — the automated funnel (below), plus `build_resume.py` (PDF rendering helper) and
  `build_application_index.py` (regenerates `user/applications.md` from the records).

## The Automated Funnel

Manual searching does not scale, so discovery and screening run unattended and only the writing
stays with a capable model.

```
scan.py  ->  triage.py  ->  appeal.py  ->  post_results.py  ->  /job-auto
(fetch)      (local LLM)    (Haiku)        (dashboard)         (packets)
```

| Stage | Script | Cost | What it does |
|---|---|---|---|
| Scan | `scan.py` | free | Fetches all tracked boards + `EMPLOYERS`, deterministic prefilter, collapses one role posted to many locations, drops anything seen before |
| Triage | `triage.py` | free | Local Ollama keep/drop + tier. Hard deal-breakers applied after in `vetoes.py` as an override the model cannot argue with |
| Appeal | `appeal.py` | ~$0.004/drop | Batched Haiku second opinion on every **judgement** drop. Vetoed drops are skipped — they were never uncertain |
| Post | `post_results.py` | free | POSTs the batch, writes the digest, dual-writes the flat file |
| Packets | `/job-auto` | Opus | Runs the existing `/job` pipeline per role, eval gate included |

`run_daily_scan.sh` chains stages 1-4 and is scheduled via `scripts/launchd/`.

**The asymmetry that shapes the design:** a wrongly *kept* role costs one line of human review; a
wrongly *dropped* role is invisible. So the screen profile biases toward keeping, and every
judgement drop gets appealed. Errors are pushed toward over-inclusion on purpose.

**What the local model may and may not decide.** It classifies and extracts. It never enforces a
deal-breaker — those live in `vetoes.py` — and it never writes anything a human or an employer
reads. Resume bullets, summaries, cover letters and screening answers stay with the capable model.

**`user/screen-profile.md` is a prompt, and prompts have no compiler.** After editing it, run
`python3 scripts/triage_cases.py`. It exists because a plausible-looking edit once narrowed the
funnel silently — dropping platform roles for containing the word "AI".

## Running Commands in Your Harness

Commands are Markdown prompt files. How they are invoked depends on your tool:

| Harness | Setup | Invocation |
|---|---|---|
| Claude Code | already wired via `.claude/commands` symlink | `/job`, `/job-eval`, … |
| Codex CLI | `scripts/install-harness.sh codex` | `/job`, `/job-eval`, … |
| Cursor | `scripts/install-harness.sh cursor` | `/job`, `/job-eval`, … |
| Gemini CLI | `scripts/install-harness.sh gemini` | `/job`, `/job-eval`, … |
| opencode | `scripts/install-harness.sh opencode` | `/job`, `/job-eval`, … |
| anything else | none | paste the contents of `prompts/commands/job.md` into the chat |

The fallback row is not a consolation prize. These are prompts; pasting one works.

## Delegation

Several commands ask for a verification or review pass to be run separately from the main
thread — adversarial term checking in `job`/`job-eval`, tone review in `app-review`, the
three research passes in `company-research`.

**Run these as subagents on a fast, cheap model if your harness supports subagents. If it does
not, run them inline in the same conversation.** The check is what matters, not the mechanism.
Do not skip a verification step because subagents are unavailable, and do not tell the user
their tool is unsupported — inline is a valid execution path for every command here.

Where a command needs a specific model tier, it says "fast and cheap" or "most capable"
rather than naming a model. Map that onto whatever your provider offers.

## Model Requirements

Nothing here requires a specific vendor. The commands assume an agent that can read and write
files, run shell commands, and fetch web pages. Everything else is prompt text.

The heaviest reasoning is resume tailoring and fit evaluation; the cheapest work is tone review
and term verification. A capable model for the former and a small one for the latter is the
efficient split, but a single mid-tier model runs the whole framework fine.

## Running the Server

```bash
cd server && go run ./cmd/server
```

Environment variables:

| Variable | Default | Purpose |
|---|---|---|
| `PORT` | `8080` | server port |
| `DATABASE_URL` | `postgres://jobhub:jobhub@localhost:5432/jobhub?sslmode=disable` | Postgres connection string |
| `API_TOKEN` | unset | bearer token for API auth (optional for local dev) |
| `JOBHUB_URL` | `http://localhost:8080` | where commands POST results |
| `JOBHUB_API_TOKEN` | unset | must match `API_TOKEN` when auth is on |
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama endpoint for `triage.py`. Required by the automated funnel only |

**There is no knowledge-base integration.** Commands read `user/master-resume.md` and
`user/personal-projects.md` directly. A `PERSONAL_KB_URL` semantic-search option existed until
2026-08-26 and was retired: it was a second, lossy copy of files that are themselves the source of
truth, its only benefit was saving tokens on bullet selection, and a stale index returns confident
wrong answers with nothing to signal it. See `docs/state-consolidation-design.md`.

**Every user needs their own server and database.** Nothing in JobHub namespaces users — if
two people point at the same `JOBHUB_URL`, their applications land in the same dashboard.

## Key Files

- `user/config.yaml` — name, email, phone, links (drives PDF generation)
- `user/master-resume.md` — **the facts.** Full bullet pool plus the Constraints tables that govern
  how each claim may be worded. Never sent as-is.
- `user/preferences.md` — **active constraints only.** Target roles, comp, location, company
  criteria, org health screen. Every rule in it is currently in force.
- `user/screen-profile.md` — distilled screen for automated triage. **Derived from
  `preferences.md`, never authoritative over it.** Regenerate when preferences change
- `user/eval-config.yaml` — eval engine customization (banned phrases, stack skills, unverified metrics)

**Two history files exist and are deliberately not loaded during normal work:**

- `user/master-resume-notes.md` — the reasoning trail behind the resume facts: dated corrections,
  manager feedback, unverified-metric notes, tailored summary variants.
- `user/preferences-notes.md` — superseded preferences and the arguments that produced the current
  ones.

**Do not read either to evaluate a role or tailor a resume.** They answer *why* a rule exists or
whether a topic has already been settled — nothing more. Split out on 2026-08-21 because the
monolithic versions (76KB and 46KB) were majority narrative, which buried the facts and made the
files impossible to follow faithfully.

## First-Time Setup

1. Run the `onboard` command to create your `user/` files, or copy `templates/` → `user/` by
   hand and fill in the placeholders (see `docs/onboarding.md`).
2. Start Postgres and the server: `cd server && go run ./cmd/server`
3. Run `job` with a job posting URL.
