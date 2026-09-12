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
renders a PDF. Every posting you evaluate and every application you send is recorded in a
dashboard so you can see your pipeline in one place.

The things it deliberately will not do: invent accomplishments, inflate numbers, or tell you a
job is a good fit because you want it to be. The eval gate exists specifically to catch a
resume that has drifted from what you can defend in an interview.

---

## Step 1 — Prerequisites

You need four things. On macOS with Homebrew:

```bash
brew install go postgresql@16
brew services start postgresql@16
pip3 install weasyprint
```

On Debian/Ubuntu:

```bash
sudo apt install golang postgresql python3-pip
sudo systemctl start postgresql
pip3 install weasyprint
```

| Tool | Why | Required? |
|---|---|---|
| Go 1.22+ | runs the server | yes |
| PostgreSQL | stores your applications, evals, research | yes |
| Python 3 + WeasyPrint | renders resume and cover letter PDFs | yes, for PDFs |
| A coding agent | runs the commands | yes — see Step 3 |

Verify:

```bash
go version && psql --version && python3 -m weasyprint --version
```

## Step 2 — Create the database

The server's default connection string expects a `jobhub` role and a `jobhub` database:

```bash
createuser -s jobhub 2>/dev/null || true
psql -d postgres -c "ALTER USER jobhub WITH PASSWORD 'jobhub';"
createdb -O jobhub jobhub
```

**You don't need to run migrations.** The server creates its own tables on first start.

If you'd rather use an existing Postgres, set `DATABASE_URL` instead:

```bash
export DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=disable"
```

Now start the server:

```bash
cd server && go run ./cmd/server
```

Open <http://localhost:8080>. An empty dashboard means everything works. Leave it running in
its own terminal.

## Step 3 — Wire up your agent

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

## Step 4 — Set up your data

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

## Step 5 — Your first posting

```
/job https://job-boards.greenhouse.io/somecompany/jobs/1234567
```

Here's what happens, and roughly what to expect:

**1. Fit evaluation.** It reads the posting, maps requirements against your history, and gives
you one of four verdicts: *strong fit*, *worth a shot*, *stretch*, or *skip*. It will name gaps
plainly. A verdict of "stretch" means don't apply without a specific reason, and it means it.

The report is saved to the dashboard. **It stops here and waits.** It won't tailor a resume
until you say to.

**2. Optional company research.** `/company-research {company}` runs a few web-search passes on
funding, layoffs, engineering culture, leadership tenure, and comp, and writes a brief. Worth
running before you invest in an application — it has killed applications that looked good on
paper.

**3. Resume tailoring.** Say yes and it selects bullets, writes a summary in your voice, and
saves to `user/tailored/{company}/{role}/resume.md`.

**4. The eval gate.** This is the part that makes the framework worth using. It scores the
resume on keyword coverage against the posting, checks that skills you claim are backed by an
actual bullet and not just listed, flags banned corporate phrases, and runs an adversarial pass
that tries to *refute* each claim you're making. Anything it can't defend gets downgraded.

**It's a hard gate.** A resume that fails goes back for rework before any PDF is generated.
Expect it to catch things — that's the point.

**5. PDF.** Rendered to `resume.pdf` in the same folder.

**6. Tracking.** It offers to log the application and upload the PDF to the dashboard.

## Step 6 — The other commands

| Command | Use it when |
|---|---|
| `/job` | the main one — postings, tailoring, cover letters, job search, tracking, strategy |
| `/company-research {company}` | before investing in an application |
| `/job-eval` | re-score a resume you already wrote, without re-tailoring |
| `/app-review` | you drafted an answer to an application question and want a tone check |

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
├── applications.md            flat-file mirror of your pipeline
├── research/                  company briefs
└── tailored/{company}/{role}/ generated resume.md, .html, .pdf, cover letters
```

The dashboard at `$JOBHUB_URL` is the nicer view of the same data. Flat files are written
either way, so nothing is lost if the server is down.

## If you're sharing this with someone

`user/` is gitignored, so a fork carries the framework and none of your history.

**Everyone needs their own server and database.** Nothing in JobHub separates users — if two
people point `JOBHUB_URL` at the same instance, their applications land in the same dashboard.

## Next

- [troubleshooting.md](troubleshooting.md) — when something breaks
- [onboarding.md](onboarding.md) — field-by-field guide to your `user/` files
- [commands.md](commands.md) — what each command reads, writes, and how to customize it
- [architecture.md](architecture.md) — how the eval engine works, if you want to change it
