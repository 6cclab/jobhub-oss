---
description: A tailored resume never ships without a current review.json — run /summary-review, then scripts/resume_preflight.py must exit 0
---

# Summary Review Gate

**No tailored resume is presented, POSTed, attached to an application, or described to Andre as
"ready" until `scripts/resume_preflight.py` on its directory exits 0.**

```bash
python3 scripts/resume_preflight.py user/tailored/{company}/{role}
```

Non-zero means not ready. Not "ready with caveats" — not ready. Fix the finding and re-run.

## The review half

The preflight requires `review.json` beside `resume.md`, carrying a SHA-256 of the summary that was
reviewed. **Edit one word of the summary and the hash stops matching**, so the review goes stale
automatically the same way the PDF does.

Produce it with **`/summary-review {dir}`**. Do not hand-write it, and do not write it from your own
read of the summary — self-review is the exact thing that has been failing.

## Why this is a rule and not a note

This is the fourth home for the same instruction. It was in `user/resume-style.md` (marked
**MANDATORY**, 2026-08-09, with a worked example of it catching four overclaims a "better-feeling"
revision had introduced). It was in `prompts/commands/job.md`. It was in a memory. **It was skipped on
every resume built on 2026-08-20**, including two reported to Andre as ready.

Andre, that day: *"you keep leaving notes and then forget. They should be enforced."* And then:
*"Ok, so how do we enforce it?"*

The answer this rule encodes: **a note that must be remembered is not a control. Bind the requirement
to an artifact, bind the artifact to a hash of what it covers, and make the script refuse.**

## What the preflight checks, so you know what "exit 0" means

- **CLAIMS** — fabricated outcomes, missing qualifiers (`projected` on the $6M, scoping on the
  99.999%, `targeted` on the 3-hour SLA), not-claimable items, superseded figures, verb escalation,
  banned phrases. Rules are data in `user/claim-rules.json`.
- **PROSE** — vague back-references, dangling openers, first-person-plural ownership in the summary,
  repeated sentence openers.
- **REVIEW** — `review.json` present, hash-current, 3-4 distinct lenses, honesty lens included, all
  verdicts pass.
- **ARTIFACT** — PDF exists, is newer than the markdown, has no raw markdown in it, loses no hyphens
  in text extraction, and is not over two pages.
- **GAPS** — with `--missing`, every term you are about to call a gap is looked up in **both**
  `master-resume.md` and `personal-projects.md` and the matching lines are printed for you to read.

## Two standing obligations

**When Andre corrects a fact, add the rule to `user/claim-rules.json` the same day.** That is what
makes the next violation get caught by a script instead of by him.

**When the renderer, the template, or a claim rule changes, run `--all`.** A renderer change can break
every resume at once — that is exactly how a lost hyphen turned `on-call` into `oncall` on nine
resumes, one of them already submitted.

## The limit, stated plainly

The preflight checks claims, prose shape, and artifacts. It cannot tell you a summary is dull, that it
buries the point, or that a bullet is at the wrong altitude. It also cannot stop a lazy judge panel,
because the same agent writes the artifact. Those still need reading. Do not report a passing
preflight as proof the resume is good — only as proof it is not broken in the ways that have broken
before.
