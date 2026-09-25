# JobHub Documentation

## Start here

- **[getting-started.md](getting-started.md)** — clone to first tailored application. Start here.
- **[troubleshooting.md](troubleshooting.md)** — symptoms and fixes when something breaks.

## Reference

- **[onboarding.md](onboarding.md)** — field-by-field guide to every file under `user/`.
- **[commands.md](commands.md)** — what each command reads, writes, and how to customize it.
- **[architecture.md](architecture.md)** — how the pieces fit, and how the eval engine scores.
- **[../AGENTS.md](../AGENTS.md)** — what your coding agent reads. Directives only.

## Which file answers my question?

| Question | File |
|---|---|
| How do I install and run this? | [getting-started.md](getting-started.md) |
| Do I need a server or a database? | No. [getting-started.md](getting-started.md) |
| What do I put in my master resume? | [onboarding.md](onboarding.md) |
| What does each command actually do? | [commands.md](commands.md) |
| Why is my PDF three pages / not rendering? | [troubleshooting.md](troubleshooting.md) |
| Why did my resume fail the gate? | [troubleshooting.md](troubleshooting.md), then [architecture.md](architecture.md) |
| How does the eval score a resume? | [architecture.md](architecture.md) |
| How do I change the banned phrase list? | [onboarding.md](onboarding.md) (`eval-config.yaml`) |
| How do I use this with Codex / Gemini / Cursor? | [getting-started.md](getting-started.md) |
| Can I run this without paying for anything? | [getting-started.md](getting-started.md) — yes |
| How do I add a new command? | [commands.md](commands.md) + [../AGENTS.md](../AGENTS.md) |
| Where did a board or scan result go? | [troubleshooting.md](troubleshooting.md) |

## The one thing to understand first

Two stores, and they never overlap:

| Files under `user/` | `user/jobhub.db` |
|---|---|
| applications, research, resumes, cover letters | boards, scan batches, fit reports |
| hand-written, worth diffing | machine-generated, voluminous |
| **authoritative** | **derived, and disposable** |

`user/applications.md` is generated from `user/applications/`. Never hand-edit it — edit the
record and run `python3 scripts/build_application_index.py`.

## A note on `docs/internal/`

Design records and the rationale behind each rule live in `docs/internal/`, which is **not
published**. It is written about a real job search and names real companies. If you cloned this
repo you will not have it, and nothing in the framework depends on it — the rules in
`prompts/rules/` are self-contained directives.
