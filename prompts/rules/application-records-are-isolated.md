---
description: An application record describes one company only — cross-company reasoning goes in user/offer-timeline.md, gated by build_application_index.py
---

# An Application Record Holds One Company

## Do this

`user/applications/{company}-{role}.md` describes **that company and nothing else**. It may
*point* at another record. It may not *narrate* one.

| Allowed | Not allowed |
|---|---|
| `See user/offer-timeline.md for the pipeline picture.` | "This moves Company A from last to second in the pipeline." |
| `Withdrawn; accepted a role elsewhere.` | "The user accepted Company A's offer same day and withdrew from Company B, C and D." |
| A path in backticks, or a markdown link | The company's name in a sentence |

- Put cross-company reasoning in `user/offer-timeline.md`. Comparing two offers, ranking the
  pipeline, reasoning about conflicting rounds — all of it, one place.
- Never add a record to `user/isolation-baseline.txt`. Fix the record instead.
- Delete a record from that list as soon as it stops leaking. The check names them on every run.

```bash
python3 scripts/build_application_index.py           # warns on any record naming another company
python3 scripts/build_application_index.py --gate    # exit 1 on any NEW leak
```

## `mentions_ok:` is for products, not pipeline talk

Some companies here are technologies first: **ClickHouse, Neon, Mercury**. Discussing the product
on its merits is legitimate, so a record may declare it:

```yaml
mentions_ok: ClickHouse
```

**This is not an escape hatch.** Never declare a company to silence a legitimate finding. If the
sentence is about a *process*, a *decision*, or a *comparison*, it does not belong in the record
no matter what key is set.
