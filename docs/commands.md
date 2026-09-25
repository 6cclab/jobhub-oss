# Command Reference

All commands live in `prompts/commands/` and are invoked as `/command-name [args]` in whichever
agent you wired up (see [`AGENTS.md`](../AGENTS.md)). They need no configuration and no running
service. Everything they read and write is a file under `user/`, or a row in `user/jobhub.db` via
`scripts/jobhub_db.py`.

Edit commands in `prompts/commands/`, never through the `.claude/commands` symlink.

| Command | One line |
|---|---|
| [`/job`](#job) | The main assistant — fit, tailoring, cover letters, tracking |
| [`/job-auto`](#job-auto) | Batch packet builder from the overnight scan queue |
| [`/job-eval`](#job-eval) | Standalone resume scoring against a posting |
| [`/summary-review`](#summary-review) | Mandatory judge panel; writes `review.json` |
| [`/pdf-review`](#pdf-review) | Final read of the rendered PDF |
| [`/ats-check`](#ats-check) | Posting vocabulary against the rendered PDF |
| [`/company-research`](#company-research) | Stability, culture and comp before you apply |
| [`/app-review`](#app-review) | Tone check for application text |
| [`/interview-prep`](#interview-prep) | Prep sheet for a scheduled round |
| [`/onboard`](#onboard) | Interactive first-run setup |

## The pipeline

Most of these commands are stations on one line. The order matters, and two of the stations are
gates:

```
/job ──> tailored resume ──> /job-eval ──> /summary-review ──> resume_preflight.py ──> /pdf-review
                             (advisory)     (writes review.json)   (THE GATE)          (final read)
```

- **The eval is advisory.** Run it once, read it, act on what is genuinely wrong. Never re-run it
  to move the score. See [`prompts/rules/job-eval-gate.md`](../prompts/rules/job-eval-gate.md).
- **The preflight is the gate.** `python3 scripts/resume_preflight.py {dir}` must exit 0 before a
  resume is described as ready. See
  [`prompts/rules/summary-review-gate.md`](../prompts/rules/summary-review-gate.md).

## /job

The main assistant. One command, several modes — it detects intent from what you give it.

**Modes:**

- **Fit evaluation** (always first when you share a posting) — match signals, gaps and red flags
  against your background, with a verdict (`strong` / `worth` / `stretch` / `skip`). Won't offer to
  tailor for `stretch`/`skip` unless you ask.
- **Resume tailoring** — **selects** and reorders bullets from your master resume, fills gaps from
  personal projects, records citations in `provenance.json`, then runs the eval, the judge panel
  and the preflight before any PDF exists.
- **Cover letter** — follows your `cover-letter-style.md` structure and renders a PDF.
- **Application questions** — drafts answers in your voice, then runs every answer through a
  mandatory tone-review pass before showing it to you.
- **Job search** — pulls postings from tracked Greenhouse boards, filters against your preferences,
  records a batch of search results.
- **Application tracking** — writes `user/applications/{company}-{role}.md` and regenerates the
  index.
- **Strategy discussion** — salary, targeting, preferences; updates `user/preferences.md` when the
  conversation surfaces something new.
- **Adding/discovering boards** — probes Greenhouse slugs and tracks new companies.

**Reads:** `user/config.yaml`, `user/preferences.md`, `user/master-resume.md`,
`user/personal-projects.md`, `user/resume-style.md`, `user/cover-letter-style.md`,
`user/communication-style.md`, `user/boards.md` (discovery seed list only — tracked boards come from
`user/jobhub.db`).

**Writes:** `user/tailored/{company}/{role}/` (resume.md, fit-report.html, provenance.json,
review.json, PDFs), `user/research/{company}.md`, `user/applications/{company}-{role}.md`. Funnel
entities (fit reports, search results, boards) go to `user/jobhub.db` via `jobhub_db.py`.

**Never hand-edit `user/applications.md`** — it is generated. Edit the record, then run
`python3 scripts/build_application_index.py`.

**Customize by editing:**
- `user/resume-style.md` — voice rules, banned phrases, bullet formatting
- `user/preferences.md` — target roles, salary floor, deal-breakers
- `user/communication-style.md` — your writing voice, for application answers and cover letters
- `user/master-resume.md` and `user/personal-projects.md` — your evidence pool. The "Skill gaps this
  fills" sections in the latter control which project bullets get pulled in during gap-fill

## /job-auto

Turns the overnight scan queue into ready-to-submit application packets, in batch. **It does not
submit anything** and will not offer to.

**Usage:** `/job-auto` — presents the queue, asks which roles to build. Default batch is 5–8; if you
name more than 8 it builds the first 8 and says the rest are queued.

**Process:** finds the newest `user/search-results/{date}-scan*.md` by mtime, presents kept roles as
a table (flagging appealed roles and prior rejections), then for each role you pick runs the full
`/job` pipeline unchanged — re-reading `prompts/commands/job.md` each time rather than working from
memory. Per role it also writes the posting to `posting.md` **before** tailoring, answers screening
questions through the tone gate, and writes `packet.md`. If a resume can't pass preflight after two
honest rework rounds, it stops on that role and moves on.

**Reads:** `user/search-results/{date}-scan*.md`, `user/jobhub.db`, `prompts/commands/job.md`,
`user/config.yaml`.

**Writes:** `user/tailored/{company}/{role}/posting.md` and `packet.md`, plus everything the `/job`
pipeline writes.

**Gate:** never calls a packet ready unless `resume_preflight.py` exited 0, the PDFs exist on disk,
and `posting.md` holds the listing. It checks this rather than inferring it.

**Customize by editing:** `prompts/commands/job.md` governs the pipeline this wraps; the `packet.md`
template is embedded in `job-auto.md` itself.

## /job-eval

Standalone resume scoring. Same deterministic engine and adversarial verification that `/job` uses,
run on demand — for auditing resumes that already went out, or catching systemic issues across
`user/tailored/`.

**Usage:**
- `/job-eval` or `/job-eval all` — every resume in `user/tailored/`, as a scorecard
- `/job-eval acme` — just that company's resume(s)
- `/job-eval latest` — the most recently modified resume

**Process:** recovers the original posting (from `posting.md`, a cached fit report, or the
application record), extracts posting terms, builds the project-gaps map from
`personal-projects.md`, runs the engine, then runs adversarial verification (single-resume mode
only — skipped in batch). Prints the keyword diff, gap-fill check, style compliance, skills
relevance, structural and AI-tells breakdown with a verdict.

```bash
cd eval && go run ./cmd/eval --config ../user/eval-config.yaml < request.json
```

**Reads:** `user/master-resume.md`, `user/personal-projects.md`, `user/resume-style.md`,
`user/preferences.md`, resumes under `user/tailored/`.

**Writes:** nothing. It reads and reports.

**Remember it is advisory.** A `needs_rework` verdict is information to weigh, not a wall, and the
engine's `genuine_gap` label describes a string it did not match — never the candidate. Check
`master-resume.md` and `personal-projects.md` before repeating it as a gap:

```bash
python3 scripts/resume_preflight.py {dir} --missing "term one,term two"
```

**Customize by editing:** `user/eval-config.yaml`. The config is read fresh on every run, so an edit
takes effect immediately.

## /summary-review

The mandatory judge panel over a tailored resume's summary. Produces the `review.json` that
`resume_preflight.py` requires. **Never hand-write that file.**

**Usage:** `/summary-review {dir}`, e.g. `/summary-review user/tailored/acme/senior-software-engineer`.
If no directory is given it asks rather than guessing.

**Process:** reads the resume and your style files, spawns 3–4 judges in parallel on a cheap model
(evidence-and-honesty is required; plus sentence-structure, voice-authenticity, hiring-manager-skim),
cross-references their findings — a finding two or more judges reach independently is treated as
real — overrules wrong judges with a recorded reason, revises, then runs a mandatory second round
with at least the structure and honesty lenses. Never accepts a suggestion that adds experience.

**Reads:** `{dir}/resume.md`, `user/resume-style.md`, `user/master-resume.md`,
`user/personal-projects.md`.

**Writes:** `{dir}/review.json` — `reviewed_at`, `rounds`, `summary_sha256`, `lenses[]`,
`overruled[]`, `changes_made[]`.

**Then:** if the summary changed, `python3 scripts/build_resume.py {dir}` and
`python3 scripts/resume_preflight.py {dir}` must exit 0. The `summary_sha256` must match what the
preflight computes, which is how a stale `review.json` gets caught.

**Customize by editing:** `user/resume-style.md`, or the judge lenses in the command file.

## /pdf-review

Final read of the **rendered PDF**, not the markdown. Run after preflight passes, before anything is
sent.

**Usage:** `/pdf-review {dir}`. Asks if not given.

**Process:** confirms `resume.pdf` is newer than `resume.md` (and stops to rebuild if not),
rasterizes with `pdftoppm -r 130 -png` and extracts text with `pdftotext -layout` into the
scratchpad, reads every page image itself, then spawns three judges in parallel: layout and
typography, hiring-manager skim on the page, and claims re-verified against the record.

**Reads:** `{dir}/resume.md`, `{dir}/resume.pdf`, the posting, `user/master-resume.md`,
`user/personal-projects.md`.

**Writes:** nothing. It reports findings and produces no annotated images.

**This is not a gate and not a loop.** The preflight is the gate.

## /ats-check

Checks whether the rendered PDF carries the vocabulary a recruiter would search for. **It produces
no match score, by design.**

**Usage:** `/ats-check {dir} {posting url or text}`. Asks if either is missing.

**Process:** extracts the posting's search vocabulary in the posting's own wording, extracts the PDF
text with `pdftotext -layout`, then buckets every term case-insensitively into three groups:

| Bucket | Meaning |
|---|---|
| PRESENT | on the resume |
| UNSURFACED | evidenced in your record but not on this resume — a drafting problem |
| GAP | not in the record either |

The PRESENT/UNSURFACED split comes from `python3 scripts/resume_preflight.py {dir} --missing "..."`,
which searches `master-resume.md` and `personal-projects.md` for you. Fix an UNSURFACED term by
restoring or editing a bullet that carries real evidence — **never by adding a bare phrase to a
skills list.** Also checks acronym/expansion pairing and parse integrity (standard section headers,
contact details in the body rather than a header or footer).

**Writes:** nothing directly. If you edit the resume as a result, re-render with `build_resume.py` —
and if you changed the summary, `review.json` is invalidated and `/summary-review` must run again.

## /company-research

Lightweight company research scoped to job-search decisions. Cheap by design — runs on a fast model.

**Usage:** `/company-research {company name}`, optionally naming a role.

**Process:** fans out three passes in parallel where supported:
1. **Stability and financials** — funding, layoffs, headcount trend, leadership churn → Strong /
   Stable / Caution / Avoid
2. **Engineering culture and stack** — engineering blog, sentiment, remote policy, interview process
3. **Compensation** — public bands for the target level

Synthesized into one brief: overview, stability, culture, compensation, red flags, bottom line.

**Writes:** `user/research/{company}.md`. Re-running updates the existing brief.

**Customize by editing:** the three research prompts in `prompts/commands/company-research.md`.

## /app-review

Tone review for application text. `/job` already runs this automatically before showing you any
drafted answer; `/app-review` is the manual pass for something you wrote yourself.

**Usage:** `/app-review {paste text}`, or with no argument reviews the most recent drafted answer.

**Checks:**
1. **Old-org shots** — phrases implying your current or previous employer is worse than the target
2. **Project pitching** — a showcase of what you built instead of an answer to the question asked
3. **Self-deprecation** — anecdotes framing you as the shortcoming rather than the tools
4. **Banned phrases** — "passionate about," "leveraging," "excited to," "robust solutions,"
   "synergy," "track record of," "proven ability to," "demonstrated experience in,"
   "results-driven," "self-motivated," "detail-oriented"
5. **Excessive em-dashes** — more than one

**Writes:** nothing. Prints pass/fail with quoted violations and a corrected version.

**Customize by editing:** the checklist in `prompts/commands/app-review.md`.

## /interview-prep

Writes the prep sheet for a scheduled round. The prep goes in a file, not into chat.

**Usage:** `/interview-prep {company}` or `/interview-prep {company} {round}`. With no argument it
preps the application at screen stage that has no sheet.

**Process:** reads the application record, the submitted resume, the posting, the research brief
(running `/company-research` if there is none), your drill log and the offer timeline, then writes
five sections in fixed order: **Settled before you dial**, **Ask these**, **Lead with this**, **Be
straight about this, unprompted**, **Worth knowing before you dial**. Dated, with the interviewer
named if known.

Re-read the sheet before each round. A sheet written for a recruiter screen is the wrong sheet for a
system design round.

**Writes:** `user/tailored/{company}/{role}/recruiter-call.md`, and updates the application
record's `status:`, `events:` and optionally `prep:`.

**If deeper prep lives elsewhere** (`~/projects/interview-prep/local/companies/{company}.md`),
declare it in the record with a `prep:` key so the index check doesn't cry wolf.

**Then:** `python3 scripts/build_application_index.py`.

## /onboard

Interactive first-run setup. Creates `user/` from `templates/`.

**Process:** checks whether `user/config.yaml` exists (asks before overwriting), collects identity
info, copies every template into `user/`, creates `user/tailored/`, `user/research/` and
`user/resumes/`, then optionally walks you through the master resume and preferences interactively.

**Writes:** `user/config.yaml` plus a copy of `templates/*`.

Prefer to fill files in by hand? See [onboarding.md](onboarding.md).

## Scripts

The commands above shell out to these. You can also run them directly.

| Script | Purpose |
|---|---|
| `scripts/resume_preflight.py {dir}` | **The gate.** Exit 0 or the resume does not ship. Runs `check_provenance.py` internally. `--all` re-checks every resume; `--missing "a,b"` looks terms up in your record |
| `scripts/build_resume.py {dir}` | Renders resume and cover letter PDFs. **The only sanctioned renderer** — never hand-assemble HTML or call WeasyPrint directly |
| `scripts/check_provenance.py {dir}` | Verifies every bullet cites a line of `master-resume.md`. `--propose` drafts citations, and its guesses are usually wrong — treat it as a scaffold |
| `scripts/build_application_index.py` | Regenerates `user/applications.md` from the records. `--check` exits 1 if stale; `--gate` exits 1 on a new cross-company leak |
| `scripts/jobhub_db.py` | The funnel store: boards, search batches, fit reports |
| `scripts/run_daily_scan.sh` | The overnight pipeline: scan → triage → appeal → digest |
| `scripts/install-harness.sh {codex\|cursor\|gemini\|opencode\|all}` | Wires the commands into another agent |
| `scripts/triage_cases.py` | Regression suite for the triage screen. Run it after editing `user/screen-profile.md` |
