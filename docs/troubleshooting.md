# Troubleshooting

Symptoms and fixes, grouped by where things usually go wrong.

---

## Setup and the server

### `connection refused` when starting the server

Postgres isn't running.

```bash
brew services start postgresql@16     # macOS
sudo systemctl start postgresql       # Linux
pg_isready                            # should print "accepting connections"
```

### `role "jobhub" does not exist` / `database "jobhub" does not exist`

The default connection string expects both. Create them:

```bash
createuser -s jobhub
psql -d postgres -c "ALTER USER jobhub WITH PASSWORD 'jobhub';"
createdb -O jobhub jobhub
```

Or point at a database you already have:

```bash
export DATABASE_URL="postgres://user:pass@host:5432/dbname?sslmode=disable"
```

### `password authentication failed for user "jobhub"`

The role exists but the password doesn't match the default connection string:

```bash
psql -d postgres -c "ALTER USER jobhub WITH PASSWORD 'jobhub';"
```

### The dashboard loads but every page is empty

That's correct on a fresh install. It fills in as you evaluate postings and log applications.
If you've already run `/job` and it's still empty, the command was probably writing somewhere
else — see *Records aren't showing up* below.

### `migrate db: ...` on startup

Migrations run automatically at startup and are usually fine. A failure here normally means the
database exists but the role can't create tables. Make sure the role owns the database:

```bash
psql -d postgres -c "ALTER DATABASE jobhub OWNER TO jobhub;"
```

---

## Commands and your agent

### The slash command doesn't exist

Wire your harness up:

```bash
scripts/install-harness.sh codex      # or: cursor | gemini | opencode | all
```

Claude Code needs nothing — `.claude/commands` symlinks into `prompts/`.

**Codex specifically:** prompts install to `~/.codex/prompts`, which is global, not per-repo.
They're copies, not links, so re-run the installer after you edit anything in
`prompts/commands/`.

**Gemini CLI specifically:** commands are generated as TOML into `.gemini/commands/`. Same
deal — regenerate after edits.

### I edited a command and nothing changed

You probably edited through a symlink or a stale copy. **`prompts/commands/` is the source of
truth.** Edit there, then re-run `install-harness.sh` for any harness that uses copies (Codex,
Gemini).

### My agent isn't supported

It doesn't need to be. Open `prompts/commands/job.md`, paste the contents into your chat, then
paste the job posting URL. Slash commands are a convenience, not a requirement.

### The agent says it can't spawn subagents

It doesn't need to. Every verification step runs inline when subagents aren't available. If
your agent tells you a step is being skipped because of this, that's wrong — point it at the
**Delegation** section of `AGENTS.md`.

### The agent wants me to set up a knowledge base

It shouldn't — there is no knowledge-base integration. A `PERSONAL_KB_URL` option existed until
2026-08-26 and was retired. Reading `user/master-resume.md` and `user/personal-projects.md`
directly is the only path, and it always produced the same output.

---

## PDFs

### `No module named weasyprint`

```bash
pip3 install weasyprint
python3 -m weasyprint --version    # verify
```

### WeasyPrint installs but fails with a `cairo` / `pango` / `gobject` error

WeasyPrint needs native libraries that pip doesn't install:

```bash
brew install cairo pango gdk-pixbuf libffi       # macOS
sudo apt install libpango-1.0-0 libpangoft2-1.0-0 libharfbuzz0b   # Debian/Ubuntu
```

### The PDF is three pages and looks padded

It's too long. Cut bullets rather than shrinking type — the style rules say never reduce the
font size or page margins to force a fit, because it reads as padding to anyone who has seen a
few hundred resumes.

Check the real page count rather than trusting file metadata, which caches:

```bash
python3 -c "from weasyprint import HTML; print(len(HTML(filename='user/tailored/COMPANY/ROLE/resume.html').render().pages))"
```

Aim for one page, or two that are genuinely full. A second page holding three bullets and an
education line looks worse than either option done properly.

### The PDF renders but links aren't clickable

Check `user/config.yaml`: `linkedin` and `github` should be bare domains
(`linkedin.com/in/you`), with no `https://`. The templates add the scheme themselves, and a
doubled scheme produces a dead link.

---

## Evals

### The eval fails on keyword coverage and I can't fix it honestly

That's a real result, not a bug. Coverage is measured against terms you extracted from the
posting, and some will be things you genuinely haven't done. **Don't invent evidence to pass.**

A verdict of `acceptable` clears the gate. If you're stuck below it, check whether the misses
are genuine gaps or artifacts of the matcher — it does close-to-literal string matching, so
"architectural design decision" won't match a term you listed as plural. Rewording a bullet to
match the posting's own language is legitimate when you did the work. Adding a skill you don't
have is not.

### The eval says a term is covered but I can't actually back it up

Good — that's what the adversarial pass is for, and you should trust it over the keyword score.
The matcher also substring-matches, which produces false positives: a resume mentioning
**Java**Script will register "Java" as covered. Anything the adversarial pass refutes should be
treated as missing.

### The eval flags a banned phrase I want to keep

Edit `user/eval-config.yaml`. The banned list is yours to change. It ships with corporate
filler ("track record of," "leveraging," "passionate about") because those phrases are
invisible to the writer and obvious to the reader.

### It flagged my closing line as a sentence fragment

The prose checker has false positives on short aphoristic sentences. Use judgment — a `warn` on
style doesn't block the gate.

---

## Records and data

### Records aren't showing up on the dashboard

Check where the commands are actually writing:

```bash
echo "$JOBHUB_URL"      # empty means http://localhost:8080
curl -s "${JOBHUB_URL:-http://localhost:8080}/api/boards"
```

If `JOBHUB_URL` points at a deployed instance, your records are there, not local.

### `401 Unauthorized` from the API

`API_TOKEN` is set on the server but `JOBHUB_API_TOKEN` isn't set in your shell, or they don't
match. They have to be the same value.

### Fetching a dashboard *page* returns an HTML login redirect

If you put JobHub behind an SSO proxy, the bearer token usually only satisfies `/api/*` routes.
HTML pages get redirected to the identity provider. This affects saving local copies of
rendered pages; it does not affect the API, which is what the commands use.

### I edited a file under `user/` and the change didn't take

It takes immediately — every command reads `user/` off disk. This entry used to describe a
stale knowledge-base index, which was retired on 2026-08-26 precisely because that failure was
invisible: searches kept returning confident, well-formed, wrong answers.

The one exception is `user/applications.md`, which is **generated**. Edit the record under
`user/applications/` and run `python3 scripts/build_application_index.py`.

### I lost my tailored resumes

They're in `user/tailored/{company}/{role}/`, which is gitignored — so they exist on disk but
were never committed and aren't in any backup you get from git. If that matters to you, back
`user/` up somewhere private. Note that `*.pdf` is gitignored globally, so PDFs would be
skipped even inside a repo you control.

---

## Still stuck

Two things worth checking before anything else:

1. **Is the server actually running?** `curl -s localhost:8080/api/boards` should return JSON.
2. **Is your agent reading the current prompt?** Re-run `install-harness.sh` for your harness.
   Copy-based harnesses go stale silently.
