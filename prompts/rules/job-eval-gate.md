---
description: Enforced when running /job for resume tailoring — how to read the eval, and why it is not an iteration target
---

# The Eval Is Advisory. The Preflight Is The Gate.

**Changed 2026-08-21. The previous version of this file made the eval a hard gate that had to
*pass*, with instructions to fix and re-run until it did. That was the wrong control, and it caused
the failure it was meant to prevent.**

Run the eval **once**, on every tailored resume. Read the output. Act on what is genuinely wrong.
Then move on to the judge panel and `resume_preflight.py`, which is the actual gate.

- **Do not re-run the eval to move the score.**
- **Do not edit the resume to change a number.** Edit it because a finding named something real.
- A `needs_rework` or `critical` verdict is information to weigh, not a wall. Say what it flagged,
  say what you did about it, and continue.
- Skipping the eval entirely is still not an option. Running it once and reporting it honestly is.

## Why this changed

The gate created a scored loop, and a scored loop gets optimized rather than satisfied. On
2026-08-21 that produced keyword stuffing **twice in one session** — "performance testing," "agile
delivery" and "version control" bolted onto a skills list, then two whole sentences added to another
resume — for no reason except to move the number. Both were caught, but only after they were
written, and both violate the rule immediately below that has been on the books far longer.

The eval is a keyword matcher. It cannot tell whether a resume is honest, well-aimed, or worth
sending. Treating its score as the objective replaces the artifact with the metric.

## A term the matcher misses is NOT a gap in Andre's experience

**Added 2026-08-20 after getting this badly wrong on The Farmer's Dog.**

`containsTerm` is a **literal substring match**. When a posting term does not appear verbatim in the
resume, the engine reports `missing` and labels it `genuine_gap`. That label describes **the string**,
not the candidate. Reporting those terms back to Andre as "genuine gaps" tells him he lacks things he
demonstrably has.

What that produced: "written communication," "feature flags," "code review" and "A/B experiments" were
all reported to him as genuine gaps. He has every one of them —

- **Written communication:** authored the LaunchDarkly Relay ADR and drove it through technical review
  with directors and SRE leadership; wrote the canary migration documentation and trained the on-call
  engineers; designed the General Assembly React Native curriculum from scratch. It is also a
  **staff-level trait he is explicitly evaluated on**, which is what made the mislabel insulting.
- **Feature flags:** LaunchDarkly experiment flags on the Grasshopper launch, the relay gateway ADR and
  POC, and the polling-mode workaround that restored service during the October 20 outage.
- **SDK work:** generated TypeScript SDK clients across the BFF deprecation, SDK-backed API action
  handlers in Account Tools, OpenAPI-first generated SDK in volttrack.

**Rules:**

1. **Before reporting any term as a gap, check `master-resume.md` and `personal-projects.md` for the
   capability.** `resume_preflight.py --missing "term,term"` does this lookup for you. If the evidence
   exists, the finding is "the resume does not surface this yet" — a drafting problem to fix — not a
   gap. Only call something a gap when the *record* lacks it.
2. **Never suggest putting an unmatched phrase on the resume just to satisfy the matcher.** Nobody
   writes "written communication" on a resume. Surface the *evidence* instead: the ADR, the docs, the
   curriculum.
3. **Do not let the eval author the resume.** Bullets are selected from Andre's record against the
   posting; the eval then comments on the result. When a bullet carrying real evidence gets cut for
   length and the term goes `missing`, the fix is to restore the bullet, not to invent a phrase.
4. **Same failure class as reporting a tool limitation as a fact.** On 2026-08-20 a salary range was
   reported as "not attached to the posting" when the extract had simply been truncated before it.
   Both times an absence in *my* pipeline was stated as an absence in *the world*. Say "I did not find
   it," never "it is not there," unless the whole artifact was actually checked.

## A term the matcher misses is NOT a gap in Andre's experience

**Added 2026-08-20 after getting this badly wrong on The Farmer's Dog.**

`containsTerm` is a **literal substring match**. When a posting term does not appear verbatim in the
resume, the engine reports `missing` and labels it `genuine_gap`. That label describes **the string**,
not the candidate. Reporting those terms back to Andre as "genuine gaps" tells him he lacks things he
demonstrably has.

What that produced: "written communication," "feature flags," "code review" and "A/B experiments" were
all reported to him as genuine gaps. He has every one of them —

- **Written communication:** authored the LaunchDarkly Relay ADR and drove it through technical review
  with directors and SRE leadership; wrote the canary migration documentation and trained the on-call
  engineers; designed the General Assembly React Native curriculum from scratch. It is also a
  **staff-level trait he is explicitly evaluated on**, which is what made the mislabel insulting.
- **Feature flags:** LaunchDarkly experiment flags on the Grasshopper launch, the relay gateway ADR and
  POC, and the polling-mode workaround that restored service during the October 20 outage.
- **SDK work:** generated TypeScript SDK clients across the BFF deprecation, SDK-backed API action
  handlers in Account Tools, OpenAPI-first generated SDK in volttrack.

**Rules:**

1. **Before reporting any term as a gap, check `master-resume.md` and `personal-projects.md` for the
   capability.** If the evidence exists, the finding is "the resume does not surface this yet" — a
   drafting problem to fix — not a gap. Only call something a gap when the *record* lacks it.
2. **Never suggest putting an unmatched phrase on the resume just to satisfy the matcher.** Nobody
   writes "written communication" on a resume. Surface the *evidence* instead: the ADR, the docs, the
   curriculum.
3. **Do not let the eval author the resume.** It is a gate. Bullets are selected from Andre's record
   against the posting; the eval then checks the result. When a bullet carrying real evidence gets cut
   for length and the term goes `missing`, the fix is to restore the bullet, not to accept the loss.
4. **Same failure class as reporting a tool limitation as a fact.** On 2026-08-20 a salary range was
   reported as "not attached to the posting" when the extract had simply been truncated before it.
   Both times an absence in *my* pipeline was stated as an absence in *the world*. Say "I did not find
   it," never "it is not there," unless the whole artifact was actually checked.
