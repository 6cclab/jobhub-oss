# JobHub

An AI-powered job search framework. Bring your own coding agent.

JobHub tracks applications, tailors resumes, evaluates resume quality, and researches companies — driven by a set of harness-neutral Markdown prompts. Applications, resumes and research are plain Markdown files you can read and diff. The job funnel (tracked boards, scan results, fit reports) is a local SQLite file. Nothing runs as a service and there is nothing to deploy.

**No vendor lock-in.** The commands are plain Markdown prompts. They run in Claude Code, Codex, Cursor, Gemini CLI, opencode — or pasted into any chat window. See [Choosing an agent](#choosing-an-agent) for options that cost nothing.

## Quick Start

```bash
git clone <this-repo>
cd job-search

# 1. Wire the commands into your agent (Claude Code works out of the box)
scripts/install-harness.sh codex      # or: cursor | gemini | opencode | all

# 2. Set up your personal data — in your agent:
/onboard

# 3. Try it out — in your agent:
/job https://boards.greenhouse.io/example/jobs/1234567
```

`/onboard` walks you through identity info and copies `templates/` into a gitignored `user/` directory. If you'd rather set it up by hand, see [docs/onboarding.md](docs/onboarding.md).

Nothing needs to be started first: there is no server and no database to provision.

Prerequisites: **Python 3** with **WeasyPrint** (`pip install weasyprint`) for PDF output. **Go 1.26+** only if you want the resume eval engine. The automated funnel additionally wants **Ollama** for local triage.

## Choosing an agent

The commands are Markdown. Nothing here is tied to a vendor, a model, or a subscription.

| Harness | Setup | Cost |
|---|---|---|
| [Gemini CLI](https://github.com/google-gemini/gemini-cli) | `scripts/install-harness.sh gemini` | free tier, no card |
| [opencode](https://github.com/sst/opencode) | `scripts/install-harness.sh opencode` | free with a local model via Ollama |
| [Codex CLI](https://github.com/openai/codex) | `scripts/install-harness.sh codex` | included with a ChatGPT plan |
| Claude Code | works on clone | included with a Claude plan |
| [Cursor](https://cursor.com) | `scripts/install-harness.sh cursor` | free tier available |
| anything else | none | paste `prompts/commands/job.md` into the chat |

**If you're between jobs and not paying for anything: Gemini CLI's free tier or opencode against a local model will run this whole framework.** Resume tailoring and fit evaluation want a capable model; tone review and term verification are cheap and work fine on a small one.

Commands that ask for a verification pass will use subagents if your harness has them and run the check inline if it doesn't. Nothing is skipped either way.

## Architecture

Agent commands are the interface. They read your personal data files, do the reasoning (fit evaluation, resume tailoring, tone review), and write the results to disk.

```
                    ┌─> user/*.md ............ applications, research, resumes  (authoritative)
agent command ──────┤
                    └─> user/jobhub.db ....... boards, scans, fit reports       (derived)
```

**The split is deliberate.** Anything hand-written and worth diffing is a file. Anything machine-generated and voluminous is a table. Nothing lives in both, because a second copy of a fact drifts and the drift is silent.

The store is disposable: delete it and rebuild from `user/boards-export/boards-*.json` plus the next scan. Nothing in it is the only copy of anything.

The only Go in the tree is the eval engine under `eval/`, which runs as a local CLI (`eval/cmd/eval`). It speaks JSON on stdin and stdout, not HTTP. There is no server: the funnel is a local SQLite file, so there is nothing to deploy and nothing to start.

See [docs/architecture.md](docs/architecture.md) for the full breakdown, including the eval engine's scoring pipeline.

## Project Structure

```
job-search/
├── AGENTS.md               # Entry point for any coding agent (Claude Code reads .claude/CLAUDE.md -> here)
├── prompts/
│   ├── commands/           # The commands: job, job-auto, job-eval, company-research, app-review,
│   │                       #   summary-review, pdf-review, onboard
│   └── rules/              # Standing rules applied across commands
├── .claude/                # Claude Code wiring — both symlink into prompts/
├── scripts/                # The automated funnel + helpers
│   ├── scan.py             # 1. fetch every tracked board + enterprise ATS, prefilter, dedup
│   ├── triage.py           # 2. local Ollama keep/drop (free)
│   ├── vetoes.py           #    deal-breakers, enforced in Python not by a model
│   ├── appeal.py           # 3. batched Haiku second opinion on every judgement drop
│   ├── post_results.py     # 4. record the batch in user/jobhub.db + write the digest
│   ├── jobhub_db.py        #    the funnel store (SQLite): boards, scans, fit reports
│   ├── run_daily_scan.sh   #    chains 1-4; scheduled via launchd/
│   ├── triage_cases.py     #    regression test — run after editing screen-profile.md
│   └── ...                 # install-harness.sh, build_resume.py, build_application_index.py
├── templates/              # Starter templates for all user data files
├── user/                   # Your personal data (gitignored) — created via /onboard
│   ├── config.yaml         # Identity: name, email, phone, links
│   ├── applications/       # One record per application — the authoritative state
│   ├── applications.md     # Generated index of the above. Never hand-edit it
│   ├── offer-timeline.md   # Cross-company reasoning: pipeline comparisons, competing offers
│   ├── master-resume.md    # The facts: bullet pool + claim constraints, never sent as-is
│   ├── master-resume-notes.md  # Reasoning trail — history, not loaded during normal work
│   ├── preferences.md      # Active constraints: target roles, comp, location, deal-breakers
│   ├── preferences-notes.md    # Superseded preferences — history, not loaded during normal work
│   ├── screen-profile.md   # Distilled screen for automated triage (derived from preferences)
│   ├── search-results/     # Scan digests, run logs, and the dedup ledger
│   ├── personal-projects.md# Side projects that fill skill gaps
│   ├── eval-config.yaml    # Eval engine customization
│   ├── boards-export/      # Board list exports, loaded into user/jobhub.db
│   ├── tailored/           # Generated resumes, cover letters, PDFs
│   ├── research/           # Company research briefs
│   └── employment.yaml     # Employment facts. The one file the finances project reads
├── eval/                   # The deterministic eval engine, a standalone Go module
│   ├── cmd/eval/           # CLI: request JSON on stdin, result JSON on stdout
│   └── internal/eval/      # Keyword coverage, gap fill, prose checks, AI tells
└── docs/                   # Architecture, command reference, onboarding guide
```

## Commands

| Command | Purpose |
|---|---|
| `/job` | Main assistant — fit evaluation, resume tailoring (eval + judge panel + preflight gate), cover letters, Greenhouse job search, application tracking, strategy discussion |
| `/job-auto` | Batch packet builder — turns the overnight scan queue into ready-to-submit application packets |
| `/job-eval` | Standalone resume quality evaluation against a job posting — single resume, by company, or batch |
| `/summary-review` | Mandatory judge panel over a resume summary — writes the `review.json` the preflight requires |
| `/pdf-review` | Final read of the rendered PDF — layout, hiring-manager skim, claims re-verified against the record |
| `/company-research` | Lightweight research on stability, culture, and comp before you apply |
| `/app-review` | Tone check for application answers — catches old-org shots, project pitching, banned phrases |
| `/interview-prep` | Prep sheet for a scheduled round — questions to ask, what to lead with, gaps to name unprompted |
| `/ats-check` | Posting vocabulary check against a tailored resume — present, evidenced-but-unsurfaced, or a real gap |
| `/onboard` | Interactive setup — creates your `user/` files from templates |

See [docs/commands.md](docs/commands.md) for what each command reads, writes, and how to customize it.

## Environment

| Variable | Default | Purpose |
|---|---|---|
| `OLLAMA_HOST` | `http://localhost:11434` | Ollama endpoint for `scripts/triage.py`. Automated funnel only |
| `JOBHUB_DB` | `user/jobhub.db` | Redirects the funnel store, so a call site can be exercised without writing rows into the real funnel |

## Documentation

**New here? Read [docs/getting-started.md](docs/getting-started.md)** — install, setup, and your first tailored application, end to end.

- [docs/getting-started.md](docs/getting-started.md) — full walkthrough from clone to sent application
- [docs/troubleshooting.md](docs/troubleshooting.md) — symptoms and fixes when something breaks
- [docs/onboarding.md](docs/onboarding.md) — field-by-field guide to your `user/` files
- [docs/commands.md](docs/commands.md) — full command reference
- [docs/architecture.md](docs/architecture.md) — system design, eval engine internals, data flow
- [AGENTS.md](AGENTS.md) — agent-facing instructions, delegation conventions, environment variables

[docs/README.md](docs/README.md) indexes all of it with a "which file answers my question" table.

## A note on sharing

`user/` is gitignored, so none of your personal data is in this repo. If you fork this for someone
else, they get the framework and none of your history.

**Everyone's data is their own.** `user/` is gitignored and the funnel store lives inside it, so
there is nothing shared to collide over.

## Built With

- **SQLite** — the funnel store, via Python's stdlib `sqlite3`. No server, no driver, no migration tool
- **Go** — the deterministic eval engine, run as a local CLI
- **Any coding agent** — the commands are harness-neutral Markdown; see [Choosing an agent](#choosing-an-agent)
- **WeasyPrint** — HTML-to-PDF rendering for resumes and cover letters
- **Greenhouse API** — job board integration

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) — where to edit, how to run the two test suites, and the
conventions worth knowing before you change scoring or the renderer.

The short version: edit `prompts/`, never the `.claude/` symlinks, and never commit anything from
`user/`.

## License

[MIT](LICENSE)
