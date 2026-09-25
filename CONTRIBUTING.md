# Contributing

This is a personal tool, published because the framework is useful on its own. Fixes and
improvements are welcome; treat the scope as "make the framework better," not "make it fit my
job search."

## Before you start

**Your data never goes in this repo.** `user/` is gitignored and holds everything personal — your
resume, applications, preferences, the funnel store. Run `/onboard` to create it from
`templates/`. If you are about to commit something from `user/`, stop.

## Where to edit

| Want to change | Edit |
|---|---|
| a command (`/job`, `/job-eval`, …) | `prompts/commands/*.md` |
| a standing rule | `prompts/rules/*.md` |
| a starter file new users get | `templates/*` |
| the funnel, gates or renderers | `scripts/*.py` |
| resume scoring | `eval/internal/eval/*.go` |

**Never edit through `.claude/commands` or `.claude/rules`.** They are symlinks to `prompts/`.
Editing through them works locally and produces a confusing diff.

If a doc mentions `docs/internal/`, that directory is not published — it holds design records
written about a real job search. Nothing in the framework depends on it, and the rules in
`prompts/rules/` are self-contained.

## Running the tests

Same two suites CI runs, and neither needs configuration:

```bash
# Python: the funnel, the gates, the index
for f in scripts/*_cases.py; do
  [ "$f" = "scripts/triage_cases.py" ] && continue
  python3 "$f" || break
done

# Go: the eval engine
cd eval && go test ./...
```

The cases are plain `assert`s with no test-runner dependency, so run any single file directly:

```bash
python3 scripts/jobhub_db_cases.py
```

**`scripts/triage_cases.py` is excluded from CI on purpose** — it drives a local Ollama model, so
no hosted runner can run it. If you change `user/screen-profile.md` or the triage prompt, run it
by hand. It is the only thing standing between a prompt edit and a silently narrower funnel.

## Conventions worth knowing

- **The master resume is the source of every resume claim.** A tailored resume is a *selection*
  from `user/master-resume.md`, never an authorship. `scripts/check_provenance.py` enforces the
  citation and runs inside the preflight.
- **The eval is advisory; the preflight is the gate.** `scripts/resume_preflight.py {dir}` must
  exit 0. Never change a resume to move an eval score, and never re-run the eval to get a
  different number.
- **Commands are harness-neutral Markdown.** Anything that only works in one agent belongs behind
  a capability check, not hardcoded. Where a command wants a verification pass, it should use
  subagents if the harness has them and run inline if it doesn't — never skip the check, and never
  tell the user their tool is unsupported.
- **Rules are directives.** `prompts/rules/*.md` say what to do and what not to do. Rationale and
  history belong elsewhere.

## Pull requests

- Keep the diff to one concern.
- Run both suites first.
- If you change the renderer, the resume template or scoring, say so — those have knock-on effects
  on every previously generated resume, and `scripts/resume_preflight.py --all` is the way to see
  them.
- If you add a dependency, say why. The project deliberately runs on Python's stdlib plus
  WeasyPrint, and Go's stdlib plus `yaml.v3` for config and `testify` for tests.

## Reporting a problem

Include what you ran, what happened, and what you expected. If it involves a resume or an
application, **redact it** — describe the shape of the problem rather than pasting your data.
