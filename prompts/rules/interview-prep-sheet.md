---
description: Any application at phone_screen, onsite or offer gets a prep sheet written to a file — never into chat
---

# When A Screen Is Scheduled, The Questions Go In A File

## Do this

The moment an application moves to `phone_screen`, `onsite`, or `offer`, write the prep to
`user/tailored/{company}/{role}/recruiter-call.md`. Not into chat. A file. Use `/interview-prep`.

Sections, in this order:

- **Settled before you dial** — comp, office policy, anything already answered on the application.
  Include the career-years figure on the *submitted* resume, so a newer number is not introduced
  mid-call by accident.
- **Ask these** — numbered, each with the reason it matters to *them*, and the actual sentence to
  say.
- **Lead with this** — the two or three pieces of the record mapping most directly onto the
  product.
- **Be straight about this, unprompted** — the genuine gaps, phrased the way they would say them.
- **Worth knowing before you dial** — company facts, org health, interviewer background.

Date it. Name the interviewer if known. Follow an existing `recruiter-call.md` in a packet you
have already built; it is the reference.

**Re-read the sheet before each round.** A sheet written for a recruiter screen is the wrong sheet
for a system design round.

## When prep lives elsewhere

Deeper company prep lives in `~/projects/interview-prep/local/companies/{company}.md`. Declare it
in the application record so the check does not cry wolf:

```yaml
prep: ~/projects/interview-prep/local/companies/{company}.md
```

The key accepts a repo-relative or absolute path and is verified to exist.

## The check

`scripts/build_application_index.py` warns on any application at screen stage with no prep sheet.
It is a warning, not a hard failure. Do not silence it by writing a thin sheet; write the sheet or
declare `prep:`.
