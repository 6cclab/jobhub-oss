# Where Funnel Work Goes

**The store owns the funnel. Files own the record.** Search results, the dedup ledger and fit
reports go to `user/jobhub.db`. Applications, research and evals are files under `user/` and are
never written to the store.

**There is no server.** It was retired on 2026-09-11. `$JOBHUB_URL` and `$JOBHUB_API_TOKEN` are
gone; nothing in `scripts/` speaks HTTP to JobHub. If you find a command, doc or habit that still
POSTs somewhere, that file missed the migration — fix the file.

```python
import sys; sys.path.insert(0, 'scripts')
import jobhub_db as db
conn = db.connect()                       # user/jobhub.db, created if absent
boards = db.list_boards(conn, status='tracked')
```

## The line, which has been crossed in both directions

| Goes in `user/jobhub.db` | Goes in files under `user/` |
|---|---|
| tracked boards | `user/applications/{company}-{role}.md` |
| search batches and results | research briefs |
| fit reports and their signals | eval output, resumes, cover letters |

**This line was briefly "corrected" in the wrong direction on 2026-08-28.** An agent found that
`company-research.md` still instructed a `POST /api/research`, concluded this rule was the stale
file, rewrote it to permit posting, and posted a brief that cannot now be deleted. It had the
ownership backwards. `docs/state-consolidation-design.md` decides it explicitly: **"`research_briefs`
→ files. The table stops being written."**

**And it was crossed the other way for two weeks without anyone noticing.** `scan.py` kept reading
`/api/applications?status=rejected` after applications moved to files. Measured on 2026-09-11: the
server returned 8 rejected companies where the records held 11. Three were missing, so their
postings were eligible to resurface in every scan — the one outcome Andre had explicitly asked to
eliminate. The filter swallowed every failure into an empty set, so nothing ever said so.

**The lesson is not "check the rule." It is that a second store for the same fact will drift, and
the drift will be silent.** That is why there is now one store per kind of fact and no network
between you and either of them.

## If the store is missing or wrong

**Delete it and rebuild.** It is derived, and that is the whole point:

```bash
rm user/jobhub.db*
python3 -c "import sys;sys.path.insert(0,'scripts');import jobhub_db as d;\
print(d.load_boards_export(d.connect(),'server/export/boards-<date>.json'),'boards')"
```

The boards come back from the export. Search results and fit reports regenerate on the next scan.
**Nothing about an application is lost, because an application was never in there.**

`scan.py` refuses to run against a store with no tracked boards rather than quietly scanning the
dozen hardcoded enterprise employers and reporting a thin day.

## What this rule no longer has to say

Most of the previous version managed deployment risk: do not fall back to a local server, a merged
PR is not a deployed PR, probe the API before running real work against it. **A local file has none
of those failure modes.** They are deleted rather than kept "just in case" — a rule that describes a
world that no longer exists is how the 2026-08-28 mistake happened.
