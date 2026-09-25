# JobHub — Agent Instructions

Directives only. What the system is and why it is shaped this way lives in `README.md` and
`docs/`. Do not add either here.

## Read first, in this order

1. `user/applications.md` — the generated index.
2. The record under `user/applications/` — only for prose the index does not carry.
3. The rule in `prompts/rules/` governing what you are about to do.

Answer every status, stage and pipeline question from `user/applications.md`.

Never hand-edit `user/applications.md`. Edit the record, then run
`python3 scripts/build_application_index.py`.

Never read `user/master-resume-notes.md` or `user/preferences-notes.md` to evaluate a role or
tailor a resume.

## Where to write

| Writing this | Goes here |
|---|---|
| Application, research brief, eval, resume, cover letter | a file under `user/` |
| Boards, scan batches, fit reports | `user/jobhub.db`, via `scripts/jobhub_db.py` |
| Employment facts | `user/employment.yaml` |
| Cross-company reasoning, pipeline comparisons | `user/offer-timeline.md` |
| Interview prep for a scheduled round | `user/tailored/{company}/{role}/recruiter-call.md` |

Never put application state in `user/jobhub.db`. Never restate a figure from
`user/employment.yaml`. Never name another company inside an application record.

## Read the rule before you act

| Before you | Read | Then run |
|---|---|---|
| ship a tailored resume | `prompts/rules/summary-review-gate.md` | `resume_preflight.py {dir}` — exit 0 or it does not ship |
| write a bullet or summary | `prompts/rules/master-resume-is-law.md` | `check_provenance.py {dir}` |
| report an eval finding | `prompts/rules/job-eval-gate.md` | the eval, once |
| call a term a gap | `prompts/rules/job-eval-gate.md` | `resume_preflight.py --missing "term,term"` |
| write an application record | `prompts/rules/application-records-are-isolated.md` | `build_application_index.py --gate` |
| touch state | `prompts/rules/where-funnel-work-goes.md` | — |
| drive an ATS form | `prompts/rules/ats-portals.md` | — |
| prep a scheduled round | `prompts/rules/interview-prep-sheet.md` | — |

Select every resume claim from `user/master-resume.md`. Amend it first when a true claim is
missing from it; never write the claim ahead of the record.

Never re-run the eval to move a score. Never edit a resume to satisfy the matcher.

Never solve a CAPTCHA, enter an emailed verification code, create an account, enter a password or
government ID, or answer a form field the record does not support. Ask instead.

## Run after editing

| After editing | Run |
|---|---|
| `user/screen-profile.md` | `python3 scripts/triage_cases.py` (needs Ollama; CI skips it) |
| the renderer, the resume template, or `user/claim-rules.json` | `python3 scripts/resume_preflight.py --all` |
| any application record | `python3 scripts/build_application_index.py` |

When the user corrects a fact, add the rule to `user/claim-rules.json` the same day.

## Commands

Edit `prompts/commands/`. Never edit through the `.claude/commands` symlink.

`/job` `/job-auto` `/job-eval` `/company-research` `/app-review` `/summary-review` `/pdf-review`
`/ats-check` `/interview-prep` `/onboard`

Install into another harness with `scripts/install-harness.sh {codex|cursor|gemini|opencode}`.
Without slash commands, paste the file contents into the chat.

## Delegation

Run verification and review passes — adversarial term checking, tone review, research passes — as
subagents on a fast, cheap model. Run them inline when the harness has no subagents.

Never skip a verification step for lack of subagents. Never tell the user their tool is
unsupported.

Map the tier a command names onto your provider. Do not substitute a model for a named tier.

## Commands to run

```bash
python3 scripts/build_application_index.py        # after any record change
python3 scripts/resume_preflight.py {dir}         # before any resume ships
cd eval && go run ./cmd/eval < request.json       # the eval engine
scripts/run_daily_scan.sh                         # scan -> triage -> appeal -> digest
```

Nothing here speaks HTTP to a JobHub server. Fix any file that still does.

Rebuild the funnel store rather than repairing it: delete `user/jobhub.db*`, then reload boards
from `user/boards-export/boards-<date>.json` via `jobhub_db.load_boards_export()`.

Back `user/boards-export/` up outside the tree.

## The finances bridge

`/finance-plan`, `/finance-guru` and the `budget-track` MCP server live in `~/projects/finances`
(`finances_path` in `user/config.yaml`). Do not look for them here.

Read across the bridge. Never restate a figure from it, and never cite a file in it by line
number. Keep the `updated:` line in `user/employment.yaml` current.
