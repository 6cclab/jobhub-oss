# Getting Started

A start-to-finish walkthrough: install it, set up your data, and send your first tailored
application. Budget about 45 minutes, most of it spent writing your master resume — which is
work you'd have to do anyway.

If something breaks, see [troubleshooting.md](troubleshooting.md).

---

## What this actually does

You paste a job posting URL. JobHub reads it, compares it against your real work history, and
tells you honestly whether it's worth applying to — including the gaps, not just the matches.
If you decide to go ahead, it writes a resume tailored to that posting using only bullets you
actually wrote, runs the result through a quality gate that tries to catch overclaiming, and
renders a PDF. Every posting you evaluate and every application you send is recorded on disk, so
`user/applications.md` shows your whole pipeline in one place.

The things it deliberately will not do: invent accomplishments, inflate numbers, or tell you a
job is a good fit because you want it to be. The eval gate exists specifically to catch a
resume that has drifted from what you can defend in an interview.

---

## Step 1 — Prerequisites

There is no server and no database to provision. You need Python, and a coding agent.

On macOS with Homebrew:

```bash
pip3 install weasyprint
brew install go        # optional — only for the resume eval engine
```

On Debian/Ubuntu:

```bash
sudo apt install python3-pip
pip3 install weasyprint
sudo apt install golang   # optional — only for the resume eval engine
```

| Tool | Why | Required? |
|---|---|---|
| Python 3 | every script under `scripts/`, including the preflight gate | yes |
| WeasyPrint | renders resume and cover letter PDFs | yes, for PDFs |
| A coding agent | runs the commands | yes — see Step 2 |
| Go 1.26+ | the deterministic eval engine under `eval/` | optional |
| Ollama | local triage for the automated overnight scan | optional |

Verify:

```bash
python3 -m weasyprint --version
go version    # only if you want the eval engine
```

The funnel store is a SQLite file at `user/jobhub.db`, created on first use by Python's stdlib
`sqlite3`. Nothing to install, nothing to start.

## Step 2 — Wire up your agent

The commands are plain Markdown prompts, so they work in any agent. Pick one:

```bash
scripts/install-harness.sh codex      # or: cursor | gemini | opencode | all
```

**Claude Code needs nothing** — it works on clone.

**If you're between jobs and not paying for a subscription**, [Gemini
CLI](https://github.com/google-gemini/gemini-cli) has a free tier that needs no card, and
[opencode](https://github.com/sst/opencode) runs against a local model through Ollama for
nothing at all. Both run this whole framework.

**If your agent isn't on the list**, you're still fine. Open `prompts/commands/job.md`, copy
the whole thing into your chat, and paste the job posting URL after it. That's all a slash
command does.

Some commands ask for a second opinion — a tone review, or an adversarial check on whether your
resume really supports a claim. If your agent can spawn subagents, it'll use them. If not, it
runs the check inline. Nothing gets skipped either way.

## Step 3 — Set up your data

In your agent:

```
/onboard
```

It asks for your name and contact details, then copies `templates/` into a `user/` directory
and walks you through filling it in. **`user/` is gitignored** — nothing personal is ever
committed.

The one file worth real effort is **`user/master-resume.md`**. It is not a resume. It's every
accomplishment you can defend, written out in full, which the tailoring step draws from and
reorders per posting. Include things that feel too minor. A bullet you thought was filler is
often the only evidence you have for something a posting asks for.

Two things that make tailored resumes noticeably better:

- **Numbers you can actually source.** "Cut build times 50-62%" beats "improved CI performance."
  If you're unsure of a figure, put it in `eval-config.yaml` under `unverified_metrics` and the
  eval will fail any resume that uses it — a forcing function to verify it or drop it.
- **`user/personal-projects.md`.** Each project gets a "Skill gaps this fills" section. When a
  posting asks for something your job history doesn't cover, this is what fills it. Side
  projects are treated as real evidence when they're the only evidence.

See [onboarding.md](onboarding.md) for a field-by-field guide to every file.

## Step 4 — Your first posting

```
/job https://job-boards.greenhouse.io/somecompany/jobs/1234567
```

Here's what happens, and roughly what to expect:

**1. Fit evaluation.** It reads the posting, maps requirements against your history, and gives
you one of four verdicts: *strong fit*, *worth a shot*, *stretch*, or *skip*. It will name gaps
plainly. A verdict of "stretch" means don't apply without a specific reason, and it means it.

The fit report is saved to `user/tailored/{company}/{role}/fit-report.html`. **It stops here and
waits.** It won't tailor a resume until you say to.

**2. Optional company research.** `/company-research {company}` runs a few web-search passes on
funding, layoffs, engineering culture, leadership tenure, and comp, and writes a brief. Worth
running before you invest in an application — it has killed applications that looked good on
paper.

**3. Resume tailoring.** Say yes and it selects bullets, writes a summary in your voice, and
saves to `user/tailored/{company}/{role}/resume.md`.

**4. The eval.** This is the part that makes the framework worth using. It scores the resume on
keyword coverage against the posting, checks that skills you claim are backed by an actual bullet
and not just listed, flags banned corporate phrases and AI-sounding prose, and runs an adversarial
pass that tries to *refute* each claim you're making. Anything it can't defend gets downgraded.

**The eval is advisory, not the gate.** It runs once. You read it and fix what is genuinely wrong.
It is never re-run just to move the number, and a term the matcher missed is a drafting problem,
not a gap in your experience.

**5. The judge panel.** `/summary-review` puts the summary in front of several independent judges
and writes `review.json`.

**6. The gate.** `python3 scripts/resume_preflight.py user/tailored/{company}/{role}` must exit 0.
This is the real gate — not "ready with caveats." It also verifies every bullet cites a line of
your master resume.

**7. PDF.** `python3 scripts/build_resume.py user/tailored/{company}/{role}` renders `resume.pdf`
in the same folder. Then `/pdf-review` reads the rendered page one last time.

**8. Tracking.** It writes `user/applications/{company}-{role}.md` and regenerates the index with
`python3 scripts/build_application_index.py`.

## Step 5 — The other commands

| Command | Use it when |
|---|---|
| `/job` | the main one — postings, tailoring, cover letters, job search, tracking, strategy |
| `/company-research {company}` | before investing in an application |
| `/job-eval` | re-score a resume you already wrote, without re-tailoring |
| `/app-review` | you drafted an answer to an application question and want a tone check |
| `/summary-review` | required before any resume ships — writes the `review.json` the gate checks |
| `/pdf-review` | the final read of the rendered PDF, after the preflight passes |
| `/ats-check` | which of a posting's terms your PDF actually carries |
| `/interview-prep` | a screen got scheduled and you need the prep sheet |
| `/job-auto` | build packets in batch from the overnight scan queue |

See [commands.md](commands.md) for what each one reads and writes.

`/job` also handles things that aren't tailoring: "what should I be asking for in salary,"
"find me roles from my tracked boards," "log this application," "help me answer this
application question."

---

## Where things live

```
user/                          your data — gitignored, never committed
├── config.yaml                name, email, phone, links
├── master-resume.md           the bullet pool
├── personal-projects.md       side projects that fill skill gaps
├── preferences.md             what you're looking for, what you'll refuse
├── eval-config.yaml           banned phrases, unverified metrics
├── applications/              one record per application — the authoritative state
├── applications.md            generated index of the above. Never hand-edit it
├── jobhub.db                  the funnel store: boards, scans, fit reports
├── research/                  company briefs
└── tailored/{company}/{role}/ generated resume.md, .html, .pdf, cover letters
```

**Two stores, on purpose.** Anything hand-written and worth diffing is a file. Anything
machine-generated and voluminous is a row in `user/jobhub.db`. Nothing lives in both, because a
second copy of a fact drifts silently. The store is disposable — delete it and rebuild from
`user/boards-export/` plus the next scan.

## If you're sharing this with someone

`user/` is gitignored, so a fork carries the framework and none of your history.

**Everyone's data is their own.** The store lives inside `user/`, so there is nothing shared to
collide over.

## Next

- [troubleshooting.md](troubleshooting.md) — when something breaks
- [onboarding.md](onboarding.md) — field-by-field guide to your `user/` files
- [commands.md](commands.md) — what each command reads, writes, and how to customize it
- [architecture.md](architecture.md) — how the eval engine works, if you want to change it
