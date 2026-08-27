---
description: Final read of a tailored resume as a rendered PDF — layout, hiring-manager skim, and claims re-verified against the record. Run after preflight passes, before anything is sent.
---

Review the artifact that actually gets sent: the **rendered PDF**, looked at as a page rather than
read as markdown.

## Input

A tailored resume directory, e.g. `user/tailored/courier-health/senior-software-engineer`.
If none is given, ask which one. **Do not guess.**

## Why this exists, and why the overlap with preflight is deliberate

`resume_preflight.py` reads the PDF's *extracted text*. `/summary-review` reads the summary as
*markdown*. **Neither one looks at the page.** A heading orphaned at the bottom of page 1, a bullet
splitting across the break so the number lands alone, a section that visually swallows the page, a
last page that measures 81% full and still looks lopsided — every one of those passes both gates.

The claims pass here is **not** a duplicate of preflight's, and the difference is the point:
preflight checks rules encoded in `user/claim-rules.json` against text derived from the markdown.
This checks **what a reader will actually see on the page**, against `master-resume.md`, with no
rule list constraining what counts. A claim can be wrong in a way no rule anticipates — the 70%
incident reduction attributed to on-call instead of the canary migration sat on three resumes and
passed preflight clean, because no rule existed for it. That is the class of defect this catches.

**This is not a gate and not a loop.** Preflight is the gate. Run this once, read the findings, fix
what is genuinely wrong, and say what you did. **Do not re-run it to get a cleaner report** — that
is the failure `prompts/rules/job-eval-gate.md` documents.

## Process

### 1. Confirm you are reviewing the current artifact

```bash
ls -la {dir}/resume.md {dir}/resume.pdf
```

**If `resume.pdf` is not newer than `resume.md`, stop.** Rebuild first with
`python3 scripts/build_resume.py {dir}`, then start over. Reviewing a stale PDF produces findings
about a document that no longer exists, and reports them as if they were current.

Note the page count and fill that `build_resume.py` printed. You will check it against what you see.

### 2. Rasterize the pages

```bash
pdftoppm -r 130 -png {dir}/resume.pdf $SCRATCH/pdfreview/page
pdftotext -layout {dir}/resume.pdf $SCRATCH/pdfreview/resume.txt
```

Use your scratchpad directory, not the resume directory — these are working files and must not end
up beside the artifact.

### 3. Look at every page yourself

`Read` each PNG. **Every page, before delegating anything.** You are the one reporting to Andre; do
not report a finding you have not seen. This is also the step that catches the cases where the
rendered page and the markdown disagree.

### 4. Spawn three judges IN PARALLEL

Use `model: "sonnet"`. Send them in a single message so they run concurrently. Give each the PNG
paths and tell it to `Read` them — the images are the evidence, not a description of them.

**Judge A — layout and typography.** Give it: the page images, and the page count + fill figure.

```
You are reviewing the RENDERED PAGES of a resume, as images. Read every page image.
Judge only what the page looks like. Say nothing about wording, claims, or content quality.

Check:
- Orphans and widows: a section heading alone at the bottom of a page, a bullet's last
  one or two words alone at the top of the next.
- Bad breaks: a bullet split across pages so the metric lands away from the claim it
  belongs to; a job header separated from its first bullet.
- Balance: does one section visually dominate in a way its importance doesn't justify?
  Is the last page's fill honest, or does it look sparse/lopsided despite the number?
- Density and rhythm: walls of text with no breathing room; wildly uneven bullet lengths;
  more than about 5 bullets in a row without a subsection break.
- Consistency: heading sizes, spacing between sections, date alignment, bold usage.
  NOTE: bold inside experience bullets is FORBIDDEN. Flag any you see.
- Anything visibly broken: clipped text, a stray character, a link rendering as raw
  markdown, a hyphen breaking across a line.

For each finding: page number, where on the page, what is wrong, and what would fix it.
Return JSON: {"findings": [{"page": 1, "where": "", "issue": "", "fix": "", "severity": "high|medium|low"}]}
If a page is clean, say so explicitly rather than inventing something.
```

**Judge B — hiring-manager skim, on the page.** Give it: the page images and the job posting.

```
You are a hiring manager for the role below. Read the RENDERED PAGE IMAGES. You are
skimming, the way you actually would: six seconds on the top third, then decide whether
to keep reading.

Answer, based on what is VISIBLE and where it sits on the page:
- After six seconds on page 1, what do you think this person does? Is that what the
  posting is hiring for?
- Is the strongest evidence for this posting above the fold, or buried on page 2?
- Does anything read as filler, padding, or a generic phrase you have seen 200 times?
- Is there a claim you would immediately want to challenge in a screen?
- Would you take the call? One line, honest.

Judge placement and prominence, not prose quality. "This bullet is the best evidence in
the document and it is the last line of page 2" is exactly the kind of finding wanted.
Return JSON: {"six_second_read": "", "findings": [{"issue": "", "fix": "", "severity": ""}], "would_take_call": true|false, "why": ""}
```

**Judge C — claims re-verified against the record.** Give it: the extracted `resume.txt`, and tell
it to read `user/master-resume.md` and `user/personal-projects.md` itself.

```
Read user/master-resume.md and user/personal-projects.md. Then check EVERY factual claim
in the resume text below against them. You are not checking a rule list. You are checking
whether the record supports what the page says.

Default to FLAGGING unless the record clearly supports the claim as worded. Check:
- Does the accomplishment exist in the record at all?
- Is the METRIC attached to the RIGHT thing? A real number moved onto an adjacent
  accomplishment is the most common defect and the hardest to see. Verify the owner of
  every figure, not just the figure.
- Are required qualifiers present? The Constraints tables in master-resume.md name them
  ("projected" on the $6M ARR, scoping on the 99.999%, "targeted" on the 3-hour SLA,
  "instrumented across" not "adopted by" for the DX platform).
- Has a verb escalated? "partnered with" -> "led", "contributed to" -> "owned",
  "informed" -> "found". Check every verb against the source.
- Is anything on the "never claimable" list present?
- Is scope inflated? Sole credit for co-led work, a team's work claimed individually,
  a tenure stretched, a depth claimed above what the Constraints tables permit.

For each: quote the resume line, quote the source line, and state the discrepancy.
Return JSON: {"findings": [{"resume_line": "", "source_line": "", "discrepancy": "", "severity": "high|medium|low"}]}
```

### 5. Cross-reference, and overrule judges that are wrong

Two judges independently landing on the same thing makes it real. A single judge can be wrong, and
**you are expected to overrule one when it is** — but say so, and say why. Judge C in particular
will sometimes flag a legitimate paraphrase as a discrepancy; check the source line yourself before
accepting or rejecting.

**A term or phrasing the record does not contain is not automatically a defect.** Apply
`prompts/rules/job-eval-gate.md`: the question is whether the *record* supports the claim, not
whether the wording matches verbatim.

### 6. Report

**Findings only. Write no files, and produce no annotated images.**

- Lead with anything `high` severity, and say plainly if there is nothing high.
- Group by lens so it is clear what kind of problem each is.
- Name every finding you **rejected** and why. A review that only reports what it accepted is not
  showing its work.
- If the layout judge and your own read of the pages disagree, say that, and say which you trust.
- End with what you changed, if anything, and what you left alone deliberately.

**If you fix something, the PDF is now stale.** Re-render with `build_resume.py`, and re-run
`resume_preflight.py` — a summary edit also invalidates the `review.json` hash, which means
`/summary-review` has to run again. Say that plainly rather than presenting a fixed resume as ready.

## Comparing two versions

When asked to compare two resumes, rasterize both and read all pages, then report the differences
as findings. **Do not build a side-by-side sheet with two shrunken pages on one landscape page** —
it was tried on 2026-08-21 and is unreadable. If a visual comparison is genuinely wanted, render
each page full-size on its own sheet, alternating versions, so both are legible.
