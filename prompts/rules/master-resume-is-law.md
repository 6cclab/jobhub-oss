# The Master Resume Is The Law

**A tailored resume is a SELECTION from `user/master-resume.md`, not an authorship.**

Every summary sentence and every bullet cites the line(s) of the law it was selected from, in
`provenance.json`. Every figure and every content word it asserts must appear in those cited lines —
or be **declared** in `reworded` with the reason it is still claimable.

```bash
python3 scripts/check_provenance.py user/tailored/{company}/{role}
```

It runs inside `resume_preflight.py`, so there is nothing extra to remember. Non-zero means not ready.

## The ordering that is the whole point

**If a claim is true but the law does not say it, amend the law first, then write the resume.**

Not the other way round. Not "write it and justify it later." The law is what a claim is checked
against, so a claim that outruns the law is unverifiable by construction.

## Why this exists

**2026-08-28.** An LLM review judge asserted that `analysis-gated canaries with automated rollback`
was a cross-bullet conflation — that automated rollback belonged only to Identity Evergreen. It was
wrong. `master-resume.md:134` says the canaries are analysis-gated *"so a bad release halts itself,"*
which **is** automated rollback, and `:154` records that Andre led the Argo Rollouts implementation
that performs it. `claim-rules.json:115` already said *"he led the Argo Rollouts design."*

Three places in the repo held the answer. The finding was accepted without reading any of them. A
true claim was stripped from two resumes, and a `forbidden_near` rule was written into
`claim-rules.json` that would have stripped it from **every future resume automatically**.

**Every gate stayed green.** Andre caught it: *"Why are you making things up? That was the point of
canary with analysis."*

The structural failure: **every check guarded what was ADDED. Nothing guarded what was REMOVED, and
nothing checked whether what remained still traced to the record.** A `removed_claims` list was
considered and rejected — it depends on the agent honestly declaring its own deletions, which is
self-reported and therefore not a control. Provenance does not have that hole: an omitted citation
fails, so you cannot omit your way past it.

## What it catches, verified by regression test

| Change | Result |
|---|---|
| Delete the reasoning behind a reworded term | `UNDECLARED_TERM` |
| Amend a line of the law | `STALE_SOURCE` on every resume citing it |
| Edit a bullet after the fact | `UNCITED_CLAIM` |
| Add a figure not in the cited lines | `UNSOURCED_FIGURE` |
| Declare a term with a blank reason | `EMPTY_DECLARATION` |

## Prefer a citation over a declaration

If a word is unsourced, first look for a law line that **does** support it and cite that. Declaring is
the fallback. Four terms flagged on the first retrofit (`frameworks`, `unblocking`, `rebuild`, `code`)
all turned out to be supported by lines that simply had not been cited yet.

## Two things it does not do

**It cannot tell you a summary is dull, buried, or at the wrong altitude.** Same limit
`summary-review-gate.md` states. Provenance proves a claim traces to the record; it says nothing about
whether the resume is any good.

**It does not make judges trustworthy.** Judges stay advisory — they caught two real regressions the
same day they produced this false one. The change is that a judge can now only *propose*. The law
disposes. This is the same move `job-eval-gate.md` already made for the eval: *"The Eval Is Advisory.
The Preflight Is The Gate."* The judge panel never got that treatment until now.

## The archive

`user/provenance-grandfathered.txt` lists 33 resumes that predate the gate. They report
`GRANDFATHERED` and exit 0. **That list is meant to shrink to zero. Never add to it** — retrofit the
resume instead:

```bash
python3 scripts/check_provenance.py <dir> --propose   # DRAFT ONLY
```

`--propose` fuzzy-matches and **its citations are usually wrong** — on the first run it produced 31
findings from bad matches. It is a scaffold to fill in, never an answer. Correct every citation by
hand, then delete the line from the grandfather list.
