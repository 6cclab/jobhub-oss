# Troubleshooting

Symptoms and fixes, grouped by where things usually go wrong.

---

## Setup

### `weasyprint: command not found` or a native-library error

See the PDFs section below. WeasyPrint is the only hard dependency.

### `go: command not found` when running an eval

Go is optional and only needed for the eval engine under `eval/`. Install Go 1.26+, or skip the
eval — it is advisory, and `resume_preflight.py` (which is Python and *is* the gate) runs without
it.

### The eval says it is using built-in defaults

That is the CLI telling you it could not read `user/eval-config.yaml`:

```
eval: using built-in defaults (../user/eval-config.yaml: ...)
```

It is not fatal and the eval still runs. On a fresh checkout it just means you have not run
`/onboard` yet. If you have, check you are invoking the CLI from inside `eval/`, since `--config`
defaults to the relative path `../user/eval-config.yaml`.

### There is nothing to start

There is no server, no port and no database to provision. If a doc, script or command tells you to
start one, that file is stale — the funnel lives in `user/jobhub.db`, a plain SQLite file created
on first use.

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

### A board or scan result isn't showing up

Funnel data lives in `user/jobhub.db`. Look in the store, not on a dashboard:

```bash
python3 -c "import sys;sys.path.insert(0,'scripts');import jobhub_db as db;\
print(db.list_boards(db.connect(), status='tracked'))"
```

If `JOBHUB_DB` is set in your shell, the store you are reading is not the default one:

```bash
echo "$JOBHUB_DB"      # empty means user/jobhub.db
```

That variable exists so a call site can be exercised without writing rows into the real funnel.
If it is set and you did not mean to set it, your rows went to the other file.

### The store looks wrong

Rebuild it rather than repairing it. Nothing in it is the only copy of anything:

```bash
rm user/jobhub.db*
python3 -c "import sys;sys.path.insert(0,'scripts');import jobhub_db as d;\
print(d.load_boards_export(d.connect(),'user/boards-export/boards-<date>.json'),'boards')"
```

Back `user/boards-export/` up outside the tree — that is what makes this safe.

### An application isn't in `user/applications.md`

That file is generated, so it lags the records. Confirm the record exists under
`user/applications/{company}-{role}.md`, then rebuild:

```bash
python3 scripts/build_application_index.py
python3 scripts/build_application_index.py --check    # exits 1 if the index is stale
```

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

1. **Is your agent reading the current prompt?** Re-run `install-harness.sh` for your harness.
   Copy-based harnesses go stale silently — Codex and Gemini copy the prompts rather than
   symlinking them, so an edit to `prompts/commands/` does not reach them until you re-run it.
2. **Is the index current?** `python3 scripts/build_application_index.py --check` exits 1 when
   `user/applications.md` no longer matches the records behind it.
