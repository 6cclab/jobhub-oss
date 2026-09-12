# An Application Record Holds One Company

**Standing instruction from Andre, 2026-09-10:** *"why are we leaving notes about other companies
within application briefs? Everything should be isolated but cross referenced."*

`user/applications/{company}-{role}.md` describes **that company and nothing else**. It may *point*
at another record. It may not *narrate* one.

```bash
python3 scripts/build_application_index.py           # warns on any record that names another company
python3 scripts/build_application_index.py --gate    # exit 1 on any NEW leak
```

## The line

| Allowed | Not allowed |
|---|---|
| `See user/offer-timeline.md for the pipeline picture.` | "This moves Postman from last to second in the pipeline." |
| `Withdrawn; accepted a role elsewhere.` | "Andre accepted Talkspace's offer same day and withdrew from Vanta, GitLab and Hims & Hers." |
| A path in backticks, or a markdown link | The company's name in a sentence |

**Cross-company reasoning goes in `user/offer-timeline.md`.** That file exists to hold it. Comparing
two offers, ranking the pipeline, or reasoning about which round conflicts with which — all of it,
one place.

## Why, beyond tidiness

On 2026-09-10 the check was run for the first time and **15 of 35 records** named another company.
The Postman record carried four. Three distinct costs, and only the first is obvious:

1. **It goes stale instantly and invisibly.** "Postman is second in the pipeline" was true for six
   days. Nothing updates it when the pipeline moves, because the sentence lives in a file nobody
   opens when Talkspace changes.
2. **It is invisible from the company it is about.** The Talkspace decision was recorded inside
   Postman. Reading the Talkspace record would never surface it.
3. **It makes every read expensive.** The Postman record is 500+ lines. Answering "what is
   Postman's status" meant loading four other companies' state to get one frontmatter field.

Andre's framing, same day: *"Data fetching should be efficient not crawling a million files to
maybe find something and then contaminating or fucking other stuff."*

## `mentions_ok:` is for products, not for pipeline talk

Some companies in this funnel are technologies first: **ClickHouse, Neon, Mercury**. Discussing the
product on its merits is legitimate, so a record may declare it:

```yaml
mentions_ok: ClickHouse
```

**This is not an escape hatch.** Declaring a company to silence a legitimate finding reproduces the
exact failure the check exists to catch. If the sentence is about a *process*, a *decision*, or a
*comparison*, it does not belong here no matter what key is set.

## A warning on history, a gate on new writes

**Changed 2026-09-11.** The original version was a warning on everything, with the note below about
why. That was right on day one and wrong by day two: 13 findings on every single run is how a
warning becomes background noise, which is the failure mode it was trying to avoid.

So the backlog is now written down. `user/isolation-baseline.txt` lists the 13 records that were
already leaking when the check shipped. `--gate` fails on anything **not** in that list.

```
13 pre-existing leak(s) in user/isolation-baseline.txt (backlog, not new):
  postman-senior-software-engineer-fern: CrowdStrike, GitLab, Imprint, Neon, Talkspace, Vanta
  ...

GATE FAILED: 1 record(s) outside the baseline.
```

Same idiom as `user/provenance-grandfathered.txt`, and the same two rules apply:

**The list is meant to shrink to zero. Never add to it.** Fix the record instead — move the
cross-company reasoning to `user/offer-timeline.md` and leave a path reference.

**An entry that no longer leaks must be deleted from the list.** The check tells you which, on every
run. A baseline that never shrinks stops being a backlog and becomes permission nobody re-examined.

Records are gitignored, so "what changed" cannot come from `git diff`. A baseline is the only way to
gate new writes without gating history.

### The original reasoning, still true for the backlog

It fires on the existing corpus as well as on new writes, and a gate that fails 15 files on the day
it ships is a gate that gets switched off.

## What this does not fix

The check reads prose for names. It cannot tell that a paragraph is stale, that a debrief is thin,
or that the right cross-reference was never written at all. **A record that names no other company
is not thereby well written** — it is only not leaking.
