# Where JobHub API Work Goes

**The server owns the funnel. Files own the record.** Search results, triage verdicts, the dedup
ledger and fit reports go to `$JOBHUB_URL`. Applications, research and evals are files under
`user/` and are never POSTed.

- **`$JOBHUB_URL` is the single source of truth for where funnel work is written.** Use it, plus
  `$JOBHUB_API_TOKEN` if set. Never hardcode a host.
- If `$JOBHUB_URL` is unset it defaults to `http://localhost:8080`.
- **If the server is down, the funnel stops and the record does not.** Say so plainly rather than
  letting the user believe a scan ran. Nothing about an application is lost — it was never on the
  server.

## If the user has deployed JobHub

Some users run a deployed instance and set `$JOBHUB_URL` to it. When that is the case:

- **Do not fall back to a local server when the deployed one errors.** Writing real records
  to a throwaway local database means they vanish and the dashboard shows a stale picture.
  Fix the deployment or report the failure.
- **The exception is developing the server itself** — a new eval dimension, a migration, a
  handler. That work is throwaway and should not write records to the real database. Run it
  locally on a separate `$JOBHUB_URL`.
- When a server change ships, **verify the deployed instance actually picked it up** before
  running real work against it. A merged PR is not a deployed PR. Probe the API for the new
  field or behavior rather than assuming.
