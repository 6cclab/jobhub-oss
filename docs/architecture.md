# Architecture

## System Overview

JobHub has no service layer. Agent commands do the reasoning; the filesystem and one local SQLite
file hold the results.

- **Agent commands** (`prompts/commands/*.md`) read your personal data, evaluate postings, tailor
  resumes and check tone. They are Markdown prompts, not code.
- **Python scripts** (`scripts/`) do the deterministic work: rendering PDFs, gating resumes,
  generating the application index, running the overnight scan.
- **One Go CLI** (`eval/cmd/eval`) scores a tailored resume. It reads JSON on stdin and writes
  JSON on stdout.

```
                    ┌─> user/*.md ............ applications, research, resumes  (authoritative)
agent command ──────┤
                    └─> user/jobhub.db ....... boards, scans, fit reports       (derived)
```

Nothing listens on a port. Nothing is deployed. A command that needs funnel data imports
`scripts/jobhub_db.py`; a command that needs application state reads the files.

## The two stores, and why

| Lives in files under `user/` | Lives in `user/jobhub.db` |
|---|---|
| applications, one record per company-role | tracked boards |
| research briefs | search batches and their results |
| tailored resumes, cover letters, evals | fit reports and their signals |

Anything hand-written and worth diffing is a file. Anything machine-generated and voluminous is a
table. Nothing lives in both, because a second copy of a fact drifts silently.

`user/applications.md` is a **generated projection**, not a record. `scripts/build_application_index.py`
rebuilds it from `user/applications/{company}-{role}.md`. It carries a DO-NOT-EDIT marker and hand
edits are lost on the next build.

The store is disposable. Delete it and rebuild from `user/boards-export/boards-*.json` plus the
next scan — nothing in it is the only copy of anything. See
[`prompts/rules/where-funnel-work-goes.md`](../prompts/rules/where-funnel-work-goes.md).

## Eval Engine

`eval/` is a standalone Go module. The engine (`eval/internal/eval/`) is a deterministic scoring
pipeline with no LLM calls inside it. It takes an `EvalRequest` (posting terms, resume text, skills
list, project gap map, style input) and runs six independent checks:

1. **Keyword coverage** (`scoreKeywords`) — for each posting term, is it in the resume body
   (`Covered`), only in the skills list (`SkillsOnly`), or absent (`Missing`)? Synonyms come from
   `synonyms.go`. ≥80% passes, 60–79% warns, <60% fails.
2. **Gap fill** (`checkGapFill`) — for a term no work bullet covers, could a personal project's
   "Skill gaps this fills" section have covered it? If so and it wasn't used, that's a
   `GapFillFailure`. Any top-3 gap-fill failure forces `Critical`.
3. **Style** (`checkStyle`) — summary opens in first person; no `banned_phrases`; no
   `unverified_metrics`; sentences under `MaxSentenceWords` (45); at most `MaxEmDashes` (1);
   per-role bullet counts; length within `Config.LengthBounds(pages)`.
4. **Skills relevance** (`checkSkillsRelevance`) — flags listed skills that map to no posting term
   and aren't `common_stack_skills`, and counts `SkillsOnly` terms.
5. **Structural** (`checkStructural`) — title line matches the expected seniority level; flags a
   missing education or prior-roles section.
6. **AI tells** (`checkAITells`) — detects generated-sounding prose: LLM stock phrases, participle
   cascades, "not just X but Y", rule-of-three triads, adverb padding, vague quantifiers, uniform
   sentence length. Contributes a weighted score.

**Length is page-dependent.** A one-page resume must land in 1500–4500 characters; a two-page
resume in 4500–9000. `Config.LengthBounds(targetPages)` picks the pair.

`deriveVerdict` rolls the six dimension scores into `Strong`, `Acceptable`, `NeedsRework` or
`Critical`. A top-3 gap-fill failure is an automatic `Critical`. Any hard `Fail` is `NeedsRework`.
Three or more `Warn`s is also `NeedsRework`. Otherwise `Acceptable` (with warnings) or `Strong`.

**Adversarial verification** runs on top of the engine, not inside it. Term presence is gameable —
a word can appear without real evidence behind it. After the engine returns `Covered` terms, `/job`
and `/job-eval` run a verification pass that defaults to REFUTED unless the resume text is clearly
demonstrative evidence. Refuted terms are downgraded from `Covered` to `SkillsOnly` via
`Engine.ApplyAdversarialDowngrades` and scores recalculate against the same thresholds. Run it as a
subagent on a fast, cheap model if your harness has subagents; run it inline if it doesn't. Never
skip it.

**Invocation.** The engine is a CLI, invoked fresh each time:

```bash
cd eval && go run ./cmd/eval --config ../user/eval-config.yaml < request.json
```

`--config` defaults to `../user/eval-config.yaml`. A missing or unreadable config is not fatal —
`LoadConfig` returns usable defaults, the CLI says so on stderr, and the eval runs. Because config
is read per invocation, **an edit to `eval-config.yaml` takes effect on the very next run.** There
is nothing to restart.

Exit codes: `0` means the eval ran, whatever its verdict; `1` means bad input JSON. A
`needs_rework` verdict is deliberately **not** a non-zero exit — per
[`prompts/rules/job-eval-gate.md`](../prompts/rules/job-eval-gate.md) the eval is advisory, and an
exit code would rebuild the scored loop that rule exists to remove.

See [onboarding.md](onboarding.md) for customizing `eval-config.yaml`.

## Directory Layout

```
job-search/
├── AGENTS.md            Agent-facing instructions (Claude Code reads .claude/CLAUDE.md -> here)
├── prompts/
│   ├── commands/        The commands: Markdown, harness-neutral
│   ├── rules/           Standing rules applied across commands
│   └── reference/       Reference notes commands cite (jobhub-store.md)
├── scripts/             Python and shell: the funnel, the gates, the renderers
├── templates/           Starter files copied into user/ at onboarding, plus the HTML/CSS
│                        shells (resume.html, resume.css, cover-letter.*) used for PDFs
├── eval/                Standalone Go module — the eval engine
│   ├── cmd/eval/        CLI: request JSON on stdin, result JSON on stdout
│   └── internal/eval/   engine, config, synonyms, bullets, prose, aitells
├── user/                Gitignored personal data. Nothing here is committed
└── docs/                This documentation
```

## Data Flow: Job Posting to PDF

1. **Posting in** — user gives `/job` a URL or pasted text.
2. **Fit evaluation** — `/job` diffs the posting against `user/master-resume.md` and
   `user/personal-projects.md`, optionally pulling a cached brief from `user/research/`. Produces
   match/gap/flag signals and a verdict (`strong`/`worth`/`stretch`/`skip`). Recorded via
   `jobhub_db.create_fit_report`, with a readable copy at
   `user/tailored/{company}/{role}/fit-report.html`.
3. **Resume tailoring** — bullets are **selected** from the master resume, never authored. Gaps the
   work history doesn't cover are filled from personal projects. Written to
   `user/tailored/{company}/{role}/resume.md`, with citations in `provenance.json`. See
   [`prompts/rules/master-resume-is-law.md`](../prompts/rules/master-resume-is-law.md).
4. **Eval (advisory)** — the resume is scored once by the Go CLI plus the adversarial pass. Findings
   are read and acted on where genuinely wrong. The eval is **not** re-run to move the number.
5. **Judge panel** — `/summary-review` produces `review.json`. It is never hand-written.
6. **Preflight (the hard gate)** — `python3 scripts/resume_preflight.py user/tailored/{company}/{role}`
   must exit 0. It runs `check_provenance.py` internally. Non-zero means not ready, not "ready with
   caveats." See [`prompts/rules/summary-review-gate.md`](../prompts/rules/summary-review-gate.md).
7. **PDF** — `python3 scripts/build_resume.py user/tailored/{company}/{role}`. **Never hand-assemble
   the HTML or call WeasyPrint directly**; that is how a lost hyphen turned `on-call` into `oncall`
   on nine resumes, one already submitted.
8. **Final read** — `/pdf-review` reads the rendered PDF for layout and re-verifies claims.
9. **Tracking** — a submitted application is written as a file at
   `user/applications/{company}-{role}.md`, then `python3 scripts/build_application_index.py`
   regenerates the index.

Steps 4–6 are available standalone via `/job-eval` and `/summary-review` for auditing resumes that
already went out.
