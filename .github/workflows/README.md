# Workflows

`test.yaml` runs `go test ./...` on every pull request and push to `main`, on a
GitHub-hosted runner. It needs no configuration and works on a fork out of the box.

There is no deploy workflow here on purpose. How you host JobHub is your business, and
a pipeline wired to someone else's cluster is noise in a repo you cloned to run
locally. See [docs/getting-started.md](../../docs/getting-started.md) for running it on
your own machine.

If you add one: **never give a deploy job a `pull_request` trigger if it runs on a
self-hosted runner.** On a public repo that lets a fork execute arbitrary code on your
hardware.
