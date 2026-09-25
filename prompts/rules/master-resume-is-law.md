---
description: Every tailored resume claim is selected from user/master-resume.md and cited in provenance.json — enforced by scripts/check_provenance.py
---

# The Master Resume Is The Law

## Do this

A tailored resume is a **selection** from `user/master-resume.md`, never an authorship.

- Cite the law line(s) behind every summary sentence and every bullet, in `provenance.json`.
- Make sure every figure and content word appears in the cited lines, or **declare** it in
  `reworded` with the reason it is still claimable.
- **Amend the law first when a claim is true but unrecorded.** Not the other way round, and never
  "write it and justify it later."
- **Prefer a citation over a declaration.** When a word is unsourced, look for a law line that
  supports it and cite that. Declaring is the fallback.
- Never add a resume to `user/provenance-grandfathered.txt`. Retrofit the resume instead.
- Never accept a judge's finding without reading the law lines it contradicts.

```bash
python3 scripts/check_provenance.py user/tailored/{company}/{role}
```

It runs inside `resume_preflight.py`. Non-zero means not ready.

## Retrofitting an archived resume

`user/provenance-grandfathered.txt` lists 33 resumes predating the gate. They report
`GRANDFATHERED` and exit 0. **That list is meant to shrink to zero.**

```bash
python3 scripts/check_provenance.py <dir> --propose   # DRAFT ONLY
```

`--propose` fuzzy-matches and **its citations are usually wrong** — the first run produced 31
findings from bad matches. Treat it as a scaffold, never an answer. Correct every citation by
hand, then delete the line from the grandfather list.
