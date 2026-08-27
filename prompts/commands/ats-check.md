---
description: Check a tailored resume against a posting's vocabulary — which terms a recruiter would search for are present, which are evidenced in the record but not surfaced, and which are real gaps. No score.
---

Check whether the **rendered PDF** carries the words a recruiter would actually search for, and fix the
drafting where it does not.

## Input

A tailored resume directory and the posting (URL or pasted text), e.g.
`/ats-check user/tailored/clear/senior-fullstack-software-engineer-healthcare <url>`.
If either is missing, ask. **Do not guess the posting.**

## The model this is built on, and the one it rejects

Andre's funnel is **Greenhouse and Ashby** — 54 and 22 tracked boards, 16 of 20 applications. Those
are tracking systems. **A human recruiter filters, usually with a keyword search.** They do not
auto-reject below a match-score threshold; that model belongs to enterprise Taleo/Brassring-style
systems he is not applying through. *(Vendor behaviour changes — if this becomes decision-relevant,
verify it rather than trusting this paragraph.)*

Two consequences, and they are the entire design:

1. **The terms matter.** A recruiter typing "Kubernetes" into Greenhouse will not find a resume that
   says "container orchestration." This is a real problem worth fixing.
2. **Against a search, one clean evidenced mention is worth exactly as much as four.** Density does
   nothing. Every off-the-shelf ATS skill tells you to place 5-8 keywords in the summary and repeat
   critical ones 2-4 times; that optimises for a scorer that is not in the path, and it is how
   keyword stuffing happens.

**So: NO MATCH SCORE. Do not compute one, do not report one, do not let one be inferred.** A score
becomes a target, and `prompts/rules/job-eval-gate.md` documents what happened the last time this
pipeline had one — keyword stuffing twice in a single session. Report the three buckets below and
stop.

## Process

### 1. Extract the posting's search vocabulary

Pull the terms a recruiter would plausibly search on: languages, frameworks, datastores, cloud and
infra, methodologies, domain nouns, and the role's defining capability phrases. Keep the posting's
own wording — that is the string being matched.

**Skip the unsearchable.** "Comfort with ambiguity", "curiosity about technology", "strong
collaboration skills" are not search terms and never belong in this list. If a recruiter would not
type it into a box, it is out of scope here.

### 2. Extract the text a recruiter's search actually hits

```bash
pdftotext -layout {dir}/resume.pdf - > $SCRATCH/ats.txt
```

**Check the PDF is newer than resume.md first.** Checking a stale artifact reports on a document
that no longer exists.

**Use the PDF, not the markdown.** They can disagree — the renderer silently dropped every job title
and employment date from two resumes on 2026-08-21, and both passed the preflight. What the search
hits is the PDF.

### 3. Bucket every term. Three buckets, and the middle one is the point

Match **case-insensitively with word boundaries.** A naive substring check reports "Java" present
because the resume says "JavaScript", and reported "NPI" present because `master-resume.md` contains
the word "u**npi**ck" — that one happened on 2026-08-21 and was caught only by reading the match.

| Bucket | Meaning | Action |
|---|---|---|
| **PRESENT** | The term is in the PDF text | Nothing. Do not add a second mention. |
| **UNSURFACED** | Absent from the PDF, but the *record* has the capability | **This is the actionable bucket.** Fix the drafting. |
| **GAP** | Absent from the PDF *and* absent from the record | **Leave it alone. Never invent.** |

For the second and third buckets, let the script do the lookup rather than eyeballing it:

```bash
python3 scripts/resume_preflight.py {dir} --missing "term one,term two,term three"
```

It searches **both** `master-resume.md` and `personal-projects.md` and prints the matching lines.
`NOT_A_GAP` means the evidence exists — that is an UNSURFACED term. `MUST_ASK` means it is outside
his primary stack and absent from the record; put it to him before calling it a gap, per the gap
confirmation gate in `job.md`.

### 4. Fix the UNSURFACED terms — as evidence, never as vocabulary

For each one, quote the source line from the record, then propose the **specific bullet edit** that
surfaces it.

**The fix is always a bullet carrying evidence. It is never a bare phrase.**

- ❌ Adding "performance testing" to the skills list because the posting says performance testing.
- ❌ Adding a sentence to the summary to make a term appear.
- ✅ The record says he shipped Playwright end-to-end tests gating canary promotion; the resume calls
  it "automated tests." Reword the bullet to the posting's noun, because it is the same thing.
- ✅ A bullet carrying the evidence was cut for length and its term went missing. **Restore the
  bullet.**

If a term is UNSURFACED and there is no honest way to reach it without inventing, **say so and leave
it.** An unsurfaced term you cannot reach honestly is a gap in the resume's shape, not a licence.

### 5. Acronym pairing

Recruiters search both forms and pick one arbitrarily. Where the posting uses an acronym, the resume
should carry **both** the acronym and the expansion at least once between them — but only where the
evidence is real, and only once. He already does this well in places (`MCP (Model Context Protocol)`)
and not in others.

Check the acronyms this posting actually uses. Do not spell out things nobody expands (API, AWS, SQL).

### 6. Parse-integrity checks not already covered elsewhere

`resume_preflight.py` already covers hyphens lost in extraction, raw markdown in the PDF, missing
titles and dates, and page count. Do not repeat those. Check only:

- **Section headers are the standard ones** a parser recognises: Summary, Skills, Experience,
  Education. Not "Where I've Been."
- **Contact details are in the body, not a PDF header/footer.**
- **Filename.** Every file in this repo is `resume.pdf`. What a recruiter downloads should be
  `Andre_Pato_Resume.pdf`. Flag it if the delivered file is not named that.

## Report

Three tables — PRESENT, UNSURFACED, GAP — then the acronym and parse findings. **No score, no
percentage, no "X of Y matched."**

Lead with the UNSURFACED table, because it is the only one that produces work. State plainly if it is
empty; that is a good result and should read like one.

For every GAP term, say explicitly that you are leaving it. Naming what you deliberately did not do
is the part that keeps this from drifting back into stuffing.

**Run once.** If you change the resume, re-render with `build_resume.py`, and note that a summary
edit invalidates `review.json` and requires `/summary-review` again.

## The limit

This checks vocabulary, not fit. It cannot tell you the resume is aimed at the right thing, and it
will happily report a clean vocabulary match on a resume for a role Andre should not apply to.

**And it is not the main lever.** As of 2026-08-21: 16 cold portal applications, 0 screens. Both
screens he has ever had came from recruiter inbound. If the cold column stays at zero after this,
the answer is the channel, not the document — do not keep optimising the resume in response.
