# When A Screen Is Scheduled, The Questions Go In A File

**Standing instruction from Andre, 2026-08-31.**

The moment an application moves to `phone_screen`, `onsite`, or `offer`, write the prep to
`user/tailored/{company}/{role}/recruiter-call.md`. Not into chat. A file.

## Why

Questions written into a reply are gone by the time he is dialling. He then has to ask for them
again, and the same research gets done twice — which is exactly what happened on 2026-08-31, when
the Postman questions were requested twice in one session and the second request was answered from
a file that already existed and had simply not been surfaced.

The file is also the only thing that survives to the *next* round. A recruiter screen becomes a
technical round becomes an onsite, and the liblab/Fern question, the equity markdown question and the
"vector search is a genuine gap" line are all still true three rounds later.

## The check

`scripts/build_application_index.py` warns on any application at screen stage with no prep sheet.
It runs on every status change, so the gap surfaces on its own.

```
1 application(s) at screen stage with no prep sheet -- write one with /interview-prep:
  [phone_screen] hims-hers-health-sr-swe-developer-platform-full-stack
      no prep sheet in user/tailored/hims-and-hers/sr-software-engineer-developer-platform
```

**It is a warning, not a hard failure.** A screen booked an hour ago has not had time to be prepped,
and a gate that fires on a legitimate in-between state is a gate that gets ignored.

## Prep that lives elsewhere

Deeper company prep lives in `~/projects/interview-prep/local/companies/{company}.md`. When it does,
declare it in the application record so the check does not cry wolf:

```yaml
prep: ~/projects/interview-prep/local/companies/talkspace.md
```

The `prep:` key accepts a repo-relative or absolute path and is verified to exist. This was added
because Talkspace had 31KB of prep in the interview-prep repo and still showed as missing — a false
positive, and false positives are how a warning becomes background noise.

## What goes in it

Follow `recruiter-call.md` in the Postman packet; it is the reference. Written 2026-08-31 and it
holds up:

- **Settled before you dial** — comp, office policy, anything already answered on the application.
  Naming these stops the call being spent on them. Include the career-years figure on the *submitted*
  resume, so a newer number is not introduced mid-call by accident.
- **Ask these** — numbered, each with the reason it matters to *him*, and the actual sentence to say.
- **Lead with this** — the two or three pieces of the record that map most directly onto the product.
- **Be straight about this, unprompted** — the genuine gaps, phrased the way he would say them.
- **Worth knowing before you dial** — company facts, org health, interviewer background.

Date it, and name the interviewer if known.

## The limit

This rule makes prep exist and be findable. It cannot make it good, and it cannot make it current —
a sheet written for a recruiter screen is the wrong sheet for a system design round. Re-read it
before each round rather than assuming the file is still right.
