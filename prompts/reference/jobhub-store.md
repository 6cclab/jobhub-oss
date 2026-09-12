# The Funnel Store — Module Reference

`user/jobhub.db`, driven by `scripts/jobhub_db.py`. **Look calls up here; do not memorise them.**
The decision-making lives in `prompts/commands/job.md`; this file is only the shape.

Replaces `jobhub-api.md`, deleted 2026-09-11 with the server. There is no host, no token and no
`curl`. Ownership of what goes where: `prompts/rules/where-funnel-work-goes.md`.

```python
import sys; sys.path.insert(0, 'scripts')
import jobhub_db as db

conn = db.connect()          # user/jobhub.db; creates and migrates it if absent
```

`$JOBHUB_DB` redirects the path for a throwaway run. Unset means the real store.

## Calls

| Call | Returns |
|---|---|
| `connect(path=None)` | a connection; WAL on, foreign keys on, schema applied |
| `upsert_company(conn, name)` | `company_id`, keyed on slug. First display name wins |
| `upsert_board(conn, slug, name, ats, tags=(), status='tracked')` | `board_id`; updates in place |
| `list_boards(conn)` | `{"tracked": [...], "discovery": [...]}` |
| `list_boards(conn, status='tracked')` | a flat list of that one bucket |
| `post_search_results(conn, batch, results)` | `search_batch_id`; batch and rows in one transaction |
| `list_search_results(conn, batch_id)` | rows, with `company` joined in and `tags` as a list |
| `latest_batch(conn)` | the most recent batch, or `None` |
| `create_fit_report(conn, payload)` | `fit_report_id`; writes the signals too |
| `get_fit_report(conn, id)` | the report with a `signals` list |
| `patch_fit_report(conn, id, **fields)` | `None`; narrative fields only, unknown keys raise |
| `delete_fit_signal(conn, signal_id)` | `None` |
| `list_fit_reports(conn, verdict=None, limit=100)` | reports, newest first |
| `load_boards_export(conn, path)` | count loaded, from a `server/export/boards-*.json` dump |

## Payload shapes

**`post_search_results`** — `batch` takes `ran_at`, `board_count`, `raw_count`,
`location_filter`, `notes`. Each result:

```python
{'company': 'Acme', 'role_title': 'Senior Software Engineer',
 'location': 'Remote (US)', 'is_remote': True,
 'salary_min': 180000, 'salary_max': 220000, 'salary_disclosed': True,
 'posting_url': 'https://...', 'fit_tier': 'strong',      # strong | good
 'tags': ['devex', 'senior'], 'level_tag': 'senior', 'domain_tag': 'devex'}
```

**`create_fit_report`**:

```python
{'company': 'Acme', 'role_title': '...', 'location': '...', 'level': 'senior',
 'posting_url': '...', 'verdict': 'strong',                # strong|worth|stretch|skip
 'verdict_summary': 'one sentence, concrete', 'why_apply': '<p>...</p>',
 'match_signals': [{'requirement': '...', 'evidence': '...'}],
 'gap_signals':   [{'requirement': '...', 'evidence': '...'}],
 'flag_signals':  []}
```

## Things worth knowing

**Enum values are enforced by the store**, not by a handler. A bad `verdict`, `fit_tier`, `ats` or
`status` raises `sqlite3.IntegrityError`. That is the intended behaviour: fail at the write, not
three steps later.

**`patch_fit_report` refuses unknown keys.** A typo raises rather than silently updating nothing.
`verdict` is deliberately not patchable — changing one after the fact rewrites history the digest
already reported. Create a new report.

**Signals can be deleted.** The HTTP API could not do this, and warned that "a wrong one cannot be
removed." `delete_fit_signal` fixes that.

**`below_floor` does not exist.** The comp floor was withdrawn; never gate on comp.

**Applications, research briefs and evals are NOT in here.** They are files. Do not add tables for
them — see `where-funnel-work-goes.md` for what that has already cost.
