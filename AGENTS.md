# JobHub

An AI-powered job search framework: a set of harness-neutral prompts, a local SQLite funnel store,
and a Go/HTMX dashboard that is **no longer deployed** (see "Where state lives" below).

**This file is the entry point for any coding agent.** `.claude/CLAUDE.md` points here;
Codex, Cursor, Gemini CLI, opencode, Zed and others read `AGENTS.md` directly.

## Project Structure

- `prompts/commands/` — the commands (`job`, `job-auto`, `job-eval`, `company-research`,
  `app-review`, `onboard`, `finance-plan`, `finance-guru`). Plain Markdown, no harness-specific
  syntax. **This is the source of truth.**
- `prompts/rules/` — standing rules that apply across commands.
- `prompts/skills/` — Claude Code skills. Shared domain knowledge that auto-loads when a relevant
  question comes up, so the commands reference it instead of duplicating it.
- `prompts/agents/` — Claude Code subagent definitions. A subagent's **tool list is a control**:
  `financial-guru` has no `Write` and no `WebSearch`, which is what makes "never edits a numbers
  file" and "no web access" enforceable rather than advisory.
- `.claude/commands`, `.claude/rules`, `.claude/skills/*`, `.claude/agents/*` — symlinks into
  `prompts/`. Do not edit through them; edit `prompts/` directly. Skills and agents are linked
  leaf by leaf so third-party ones can sit beside them; **re-run `scripts/install-harness.sh
  claude` after adding a new skill or agent**, unlike commands which are covered by one directory
  link.
- `user/` — your personal data (gitignored). Created via the `onboard` command or by copying
  `templates/`.
  - `user/applications/` — one file per application. **Authoritative.**
  - `user/applications.md` — generated index. **Never hand-edit.** Read it first; see below.
  - `user/jobhub.db` — the funnel store (boards, scans, fit reports). **Derived, rebuildable.**
  - `user/pipeline.md` — cross-application strategy. Hand-written.
  - `user/finances/` — the financial record. **Authoritative and the only copy** — nothing in it
    is derived or rebuildable. See below.
- `templates/` — starter templates for all user data files.
- `server/` — Go/Fiber/HTMX dashboard. **Retired 2026-09-11, left in the tree as reference.**
  Not deployed, not required, and nothing in `scripts/` calls it.
- `scripts/` — the automated funnel (below), plus `jobhub_db.py` (the funnel store),
  `build_resume.py` (PDF rendering helper) and `build_application_index.py` (regenerates
  `user/applications.md` from the records).

## Where state lives, and how to read it

Two stores, with a hard line between them. Getting this backwards has cost real time twice.

| | Lives in | Authority | Read it with |
|---|---|---|---|
| Applications, research, evals, resumes | files under `user/` | **authoritative** | `user/applications.md`, then the record |
| Boards, scans, fit reports | `user/jobhub.db` | **derived** | `scripts/jobhub_db.py` |
| Runway, budget, goals, scenarios | files under `user/finances/` | **authoritative** | `user/finances/README.md`, then the snapshot |

**Answer status, stage and pipeline questions from `user/applications.md`.** It is 50 lines and
carries company, role, status, date, source and record path for every application. Opening a
record file to read a frontmatter field is a bug: the records run to 600 lines and the index was
regenerated from them, so it cannot be stale. Open the record when you need the *prose* — a round
debrief, interviewer notes, the history.

**Never copy application state into `user/jobhub.db`.** Applications moved to files on 2026-08-26
precisely so there would be one home for them. On 2026-09-11 the scan's rejection filter was found
still reading the server's superseded `applications` table and silently missing three rejected
companies; that is the failure this line prevents.

**The funnel store is disposable.** If it is ever wrong, delete it and rebuild: the boards reload
from `server/export/boards-*.json` and everything else regenerates on the next scan. Nothing in it
is the only copy of anything.

## The Financial Record

`user/finances/` is the one directory here that is **irreplaceable**. `user/jobhub.db` can be
deleted and rebuilt; a runway model assembled by hand from bank statements cannot. Back it up
outside the repo, the same as `server/export/`.

Two commands, and the line between them is the design:

| | `/finance-plan` | `/finance-guru` |
|---|---|---|
| Job | Keeps the numbers current | Answers a judgment question |
| Writes | Dated snapshots, `README.md`, `goals.md` | Prose only, or nothing |
| Advises | **Never** | That is the whole job |
| Web | Yes, to source a figure | **Never** |

**An opinion must never become a figure.** A recommendation written into a numbers file is
indistinguishable from a measurement on the next read, and the guru treats those files as its
evidence base, so advice written there comes back as corroboration of itself. That is why the
planner is forbidden to advise and why the `financial-guru` subagent has no `Write` tool.

Three conventions the commands enforce, all detailed in
`prompts/skills/financial-planner/SKILL.md`:

- **Every figure is an actual with its source named, or an estimate with its assumption stated**,
  visible in the text rather than in a footnote.
- **Derived figures show their inputs on the row.** The math here is prompt-only — there is no
  `finances.py` and no test — so a wrong subtraction is silent unless it is auditable by eye.
- **External facts enter only through `user/finances/reference-facts.md`**, each with a source and
  an entered date. Contribution limits and tax brackets are never recalled from model weights,
  which carry a stale tax year with complete confidence and nothing marking it.

**Read `user/finances/README.md` first.** It is ~40 lines and carries the current position.
`runway-2026-08-06.md` is 1,500 lines and is only for questions about the runway model itself.

## The Automated Funnel

Manual searching does not scale, so discovery and screening run unattended and only the writing
stays with a capable model.

```
scan.py  ->  triage.py  ->  appeal.py  ->  post_results.py  ->  /job-auto
(fetch)      (local LLM)    (Haiku)        (store + digest)    (packets)
```

| Stage | Script | Cost | What it does |
|---|---|---|---|
| Scan | `scan.py` | free | Reads tracked boards from `user/jobhub.db`, fetches them + `EMPLOYERS`, deterministic prefilter, collapses one role posted to many locations, drops anything seen before |
| Triage | `triage.py` | free | Local Ollama keep/drop + tier. Hard deal-breakers applied after in `vetoes.py` as an override the model cannot argue with |
| Appeal | `appeal.py` | ~$0.004/drop | Batched Haiku second opinion on every **judgement** drop. Vetoed drops are skipped — they were never uncertain |
| Post | `post_results.py` | free | Records the batch in `user/jobhub.db`, writes the digest regardless, records the dedup ledger |
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

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama endpoint for `triage.py`. Required by the automated funnel only |
| `JOBHUB_DB` | `user/jobhub.db` | Redirects the funnel store. Use it to develop against a throwaway file instead of writing test rows into the real funnel |

`JOBHUB_URL`, `JOBHUB_API_TOKEN`, `DATABASE_URL`, `API_TOKEN` and `PORT` are **gone**. Nothing in
`scripts/` speaks HTTP to a JobHub server any more.

## The retired server

`server/` still builds and its Go tests still run in CI. It is not deployed, nothing calls it, and
it is kept only as reference for the schema that `scripts/jobhub_db.py` was translated from.
Deleting it is a separate decision that has not been taken.

It was retired on 2026-09-11 after a probe established that `GET /api/boards` was the only readable
endpoint (`search-results` and `fit-reports` returned 405, `companies` 404), the launchd scan agent
was not loaded, and no write had reached it since 2026-08-28. The one irreplaceable table, the
hand-curated board list, is exported to `server/export/boards-<date>.json` and loaded by
`jobhub_db.load_boards_export()`. Full reasoning in `docs/sqlite-funnel-design.md`.

**`server/export/` is gitignored** — a board list is 100+ companies you are targeting, and the public mirror copies everything outside `user/`. It is the funnel's only irreplaceable data, so back it up somewhere other than this repo.

**If you are tempted to stand it back up to answer a question, read `user/applications.md`
instead.** The half-alive server is what made an agent rewrite a correct rule on 2026-08-28.

**There is no knowledge-base integration.** Commands read `user/master-resume.md` and
`user/personal-projects.md` directly. A `PERSONAL_KB_URL` semantic-search option existed until
2026-08-26 and was retired: it was a second, lossy copy of files that are themselves the source of
truth, its only benefit was saving tokens on bullet selection, and a stale index returns confident
wrong answers with nothing to signal it. See `docs/state-consolidation-design.md`.

**Every user's data is their own.** `user/` is gitignored and `user/jobhub.db` lives inside it, so
there is nothing shared to collide over. This used to be a warning about two people pointing at one
`JOBHUB_URL`; with the server retired, the problem no longer exists.

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
2. Run `job` with a job posting URL. Nothing needs to be started first.

For the automated funnel you also need Ollama running and a board list in the store:

```bash
python3 -c "import sys;sys.path.insert(0,'scripts');import jobhub_db as d;\
print(d.load_boards_export(d.connect(),'server/export/boards-<date>.json'),'boards')"
```

Fresh checkouts have no export. Build a board list with `/job` discovery mode, or restore your own.
