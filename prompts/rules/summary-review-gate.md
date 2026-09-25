---
description: A tailored resume never ships without a current review.json — run /summary-review, then scripts/resume_preflight.py must exit 0
---

# Summary Review Gate

## Do this

Never present a tailored resume, attach it to an application, or describe it to the user as "ready"
until the preflight on its directory exits 0:

```bash
python3 scripts/resume_preflight.py user/tailored/{company}/{role}
```

Non-zero means not ready. Not "ready with caveats." Fix the finding and re-run.

- Produce `review.json` with **`/summary-review {dir}`**. Never hand-write it, and never write it
  from your own read of the summary.
- Run `--all` after any change to the renderer, the resume template, or `user/claim-rules.json`.
- When the user corrects a fact, add the rule to `user/claim-rules.json` the same day.
- Never report a passing preflight as proof the resume is good.
