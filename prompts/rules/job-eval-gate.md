---
description: Enforced when running /job for resume tailoring — how to read the eval, and why it is not an iteration target
---

# The Eval Is Advisory. The Preflight Is The Gate.

## Do this

Run the eval **once**, on every tailored resume. Read the output. Act on what is genuinely wrong.
Then move to the judge panel and `resume_preflight.py`, which is the actual gate.

- Never re-run the eval to move the score.
- Never edit a resume to change a number. Edit it because a finding named something real.
- Treat `needs_rework` or `critical` as information to weigh, not a wall. Say what it flagged, say
  what you did about it, continue.
- Never skip the eval. Running it once and reporting it honestly is the requirement.

## A term the matcher misses is not a gap in the user's experience

The engine's `genuine_gap` label describes a **string** it did not match, never the candidate.
Never repeat it to the user as a gap without doing the following.

1. **Check `master-resume.md` and `personal-projects.md` first.**
   `resume_preflight.py --missing "term,term"` does the lookup for you. If the evidence exists,
   the finding is "the resume does not surface this yet" — a drafting problem. Only call something
   a gap when the *record* lacks it.
2. **Never suggest putting an unmatched phrase on the resume to satisfy the matcher.** Nobody
   writes "written communication" on a resume. Surface the evidence instead: the ADR, the docs,
   the curriculum.
3. **Never let the eval author the resume.** Bullets are selected from the user's record against the
   posting; the eval comments on the result. When a bullet carrying real evidence is cut for
   length and its term goes `missing`, restore the bullet. Do not invent a phrase.
4. **Never state a limitation of your own pipeline as a fact about the world.** Say "I did not
   find it," never "it is not there," unless you checked the whole artifact.
