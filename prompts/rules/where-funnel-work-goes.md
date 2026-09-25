---
description: The store owns the funnel, files own the record — never write application state to user/jobhub.db, and there is no server
---

# Where Funnel Work Goes

## Do this

The store owns the funnel. Files own the record.

| Goes in `user/jobhub.db` | Goes in files under `user/` |
|---|---|
| tracked boards | `user/applications/{company}-{role}.md` |
| search batches and results | research briefs |
| fit reports and their signals | eval output, resumes, cover letters |

```python
import sys; sys.path.insert(0, 'scripts')
import jobhub_db as db
conn = db.connect()                       # user/jobhub.db, created if absent
boards = db.list_boards(conn, status='tracked')
```

- Never write an application, a research brief or an eval to the store.
- Never read application state from anywhere but the files.
- There is no server. `$JOBHUB_URL` and `$JOBHUB_API_TOKEN` are gone. Fix any file that still
  POSTs somewhere.
- When the store is wrong, delete and rebuild it rather than repairing it:

```bash
rm user/jobhub.db*
python3 -c "import sys;sys.path.insert(0,'scripts');import jobhub_db as d;\
print(d.load_boards_export(d.connect(),'user/boards-export/boards-<date>.json'),'boards')"
```
