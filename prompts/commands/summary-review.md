---
description: Run the mandatory judge panel over a tailored resume summary and write review.json — required before any resume ships
---

Run the summary review loop that `user/resume-style.md` has marked MANDATORY since 2026-08-09, and
write the `review.json` artifact that `scripts/resume_preflight.py` requires.

## Input

A tailored resume directory, e.g. `user/tailored/lyric/senior-software-engineer`.
If none is given, ask which one. Do not guess.

## Why this exists as a command

The loop was documented in prose and skipped anyway — on every resume built on 2026-08-20, including
two that were reported to Andre as ready. His response: *"you keep leaving notes and then forget.
They should be enforced."* Prose that must be remembered is not a control. This is one invocation and
it writes a checkable artifact.

## Process

**1. Read the inputs.** The summary from `{dir}/resume.md`, the voice rules in
`user/resume-style.md` (especially "Reference resume — The Farmer's Dog"), and the verified facts
behind every claim in the summary from `user/master-resume.md` and `user/personal-projects.md`.

**2. Spawn 3-4 judges IN PARALLEL, each with a different lens.** Use `model: "sonnet"`. Send them in
a single message so they run concurrently. Four copies of the same reviewer is not a panel — the
lenses are the point:

- **Evidence and honesty** — REQUIRED, and `resume-style.md` says it "catches the most." Paste the
  verified source facts into the prompt. Every claim must trace to one. Watch specifically for verb
  escalation ("partnered with SRE" becoming "I led"), dropped qualifiers ("projected", "targeted"),
  and scope inflation ("the systems I shipped" becoming "platform-wide").
- **Sentence structure** — word count and variance, rhythm, repeated openers, clause density, whether
  the close lands. Reject any sentence over 45 words.
- **Voice authenticity** — against `user/resume-style.md`. Banned phrases, corporate filler,
  third-person drift, and whether the personality reads as genuine or performed.
- **Hiring-manager skim** — eight seconds. What lands, what is skippable, does the first sentence
  carry the qualification or waste itself on a mood.

Each judge returns: a score out of 10, specific issues quoting the offending text, and a full rewrite.

**3. Cross-reference.** A finding two or more judges reach independently is real. A single judge's
stylistic preference usually is not.

**4. Overrule judges when they are wrong, and say so in the artifact.** They have been wrong before:
one proposed adding Jenkins, which Andre has never used; another called "I'd rather leave a pattern
behind than a fix" manufactured, not knowing it is Andre's own recorded phrasing. **Never accept a
suggestion that adds experience.** Record the overrule and the reason.

**5. Revise, then run round two** with at least the structure and honesty lenses. This is not
optional. On the Affirm CI resume, round one scored 7/7/6/8 and the *revision* introduced four new
overclaims that round two caught at 4/10 on honesty.

**6. Write `{dir}/review.json`:**

```json
{
  "reviewed_at": "YYYY-MM-DD",
  "rounds": 2,
  "summary_sha256": "<sha256 of the FINAL summary text>",
  "lenses": [
    {"lens": "evidence-and-honesty", "verdict": "pass", "score": 9, "findings": ["..."]},
    {"lens": "sentence-structure",   "verdict": "pass", "score": 8, "findings": ["..."]},
    {"lens": "voice-authenticity",   "verdict": "pass", "score": 8, "findings": ["..."]},
    {"lens": "hiring-manager-skim",  "verdict": "pass", "score": 8, "findings": ["..."]}
  ],
  "overruled": [{"judge": "...", "suggestion": "...", "why_rejected": "..."}],
  "changes_made": ["..."]
}
```

The SHA must be of the **final** summary, computed the same way the preflight does:

```bash
python3 -c "
import hashlib,re,pathlib,sys
t=pathlib.Path(sys.argv[1]+'/resume.md').read_text()
s=re.sub(r'\s+',' ',t.split('## Summary')[1].split('##')[0]).strip()
print(hashlib.sha256(s.encode()).hexdigest())" {dir}
```

**7. Rebuild and verify.** If the summary changed, `python3 scripts/build_resume.py {dir}`, then
`python3 scripts/resume_preflight.py {dir}` must exit 0.

## Honest limits

The artifact is written by the same agent that wrote the summary, so nothing here prevents a lazy
panel. The SHA makes staleness visible and the lens requirements make thinness visible; Andre reading
`review.json` occasionally is what actually backstops it. Do not describe this as a guarantee.
