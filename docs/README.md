# JobHub Documentation

## Start here

- **[getting-started.md](getting-started.md)** — install it, set up your data, send your first
  tailored application. Start here if you've just cloned the repo.
- **[troubleshooting.md](troubleshooting.md)** — symptoms and fixes when something breaks.

## Reference

- **[onboarding.md](onboarding.md)** — field-by-field guide to every file under `user/`. What
  goes in your master resume, how the personal-projects gap-fill works, what the eval config
  actually checks.
- **[commands.md](commands.md)** — what each command reads, writes, and how to customize it.
- **[architecture.md](architecture.md)** — system design, eval engine internals, data flow.
  Read this before changing how scoring works.
- **[state-consolidation-design.md](state-consolidation-design.md)** — accepted design, not yet
  implemented. Which store owns which entity, why application records become one file each, and
  how the migration protects itself.
- **[../AGENTS.md](../AGENTS.md)** — agent-facing project instructions: directory layout,
  delegation conventions, environment variables. This is what your coding agent reads.

## Which file answers my question?

| Question | File |
|---|---|
| How do I install and run this? | [getting-started.md](getting-started.md) |
| What do I put in my master resume? | [onboarding.md](onboarding.md) |
| Why is my PDF three pages / not rendering? | [troubleshooting.md](troubleshooting.md) |
| Why did my resume fail the eval gate? | [troubleshooting.md](troubleshooting.md), then [architecture.md](architecture.md) |
| How do I change the banned phrase list? | [onboarding.md](onboarding.md) (`eval-config.yaml`) |
| How do I use this with Codex / Gemini / Cursor? | [getting-started.md](getting-started.md) |
| Can I run this without paying for anything? | [getting-started.md](getting-started.md) — yes |
| How does the eval actually score a resume? | [architecture.md](architecture.md) |
| How do I add a new command? | [commands.md](commands.md) + [../AGENTS.md](../AGENTS.md) |
