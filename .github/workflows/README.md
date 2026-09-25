# Workflows

`test.yaml` runs on every pull request and push to `main`, on GitHub-hosted runners.
It needs no configuration and works on a fork out of the box. Two jobs:

| Job | What it runs |
|---|---|
| `python` | every `scripts/*_cases.py` file directly, as plain asserts |
| `eval` | `go test ./...` inside `eval/` |

`scripts/triage_cases.py` is skipped in CI because it drives a local Ollama model,
which no runner has. Run it by hand after editing `user/screen-profile.md`.

There is no deploy workflow, because there is nothing to deploy: JobHub runs entirely
on your own machine, with no server and no database. See
[docs/getting-started.md](../../docs/getting-started.md).

If you add one: **never give a deploy job a `pull_request` trigger if it runs on a
self-hosted runner.** On a public repo that lets a fork execute arbitrary code on your
hardware.
