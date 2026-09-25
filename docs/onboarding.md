# Setting Up Your `user/` Files

Everything JobHub knows about you lives in `user/`. It is gitignored, so none of it is ever
committed or published.

`/onboard` creates these files from `templates/`. This page is the field-by-field guide — read it
if you'd rather fill them in by hand, or when you want to know what a file actually drives.

**The two that matter most are `master-resume.md` and `preferences.md`.** Everything else is
refinement. If you only have an hour, spend it on the master resume.

---

## The files, and what reads them

| File | Drives | Priority |
|---|---|---|
| `config.yaml` | your name and contact block on every PDF | do it first, 2 minutes |
| `master-resume.md` | **every bullet on every tailored resume** | the real work |
| `preferences.md` | which postings get filtered out and flagged | do it early |
| `personal-projects.md` | gap-fill when work history doesn't cover a requirement | high value |
| `resume-style.md` | voice rules the judge panel enforces | tune later |
| `eval-config.yaml` | banned phrases and metrics you can't defend | tune later |
| `cover-letter-style.md` | cover letter structure | only if you send them |
| `communication-style.md` | your voice in application answers | only if you use them |
| `boards.md` | seed list for board discovery | optional |

---

## `config.yaml`

Identity. Copied onto every rendered PDF.

```yaml
name: "Jane Doe"
email: "jane@example.com"
phone: "555-123-4567"
location: "City, ST"
linkedin: "linkedin.com/in/janedoe"
github: "github.com/janedoe"
sign_off: "Jane"
```

Write links as **bare domains**, not full URLs. The renderer builds the anchor; a full
`https://` here renders the protocol as visible text on the PDF.

`sign_off` is the name used to close a cover letter.

## `master-resume.md` — the law

This is the single most important file, and the one rule that governs it is worth stating plainly:

> **A tailored resume is a *selection* from this file, never an authorship.**

Every bullet and every summary sentence on a tailored resume must trace back to a line here.
`scripts/check_provenance.py` enforces it, and it runs inside the preflight gate. If a claim is
true but not written down here, **amend this file first** — never write the claim onto a resume
and justify it afterward.

**Sections**, in order: `Summary`, `Skills`, `Experience`, `Education`.

**Summary.** 3–5 sentences, first person. What you care about and what you do — not a corporate
bio. "I am a Staff Software Engineer who genuinely cares about how other engineers experience
their tools" beats any third-person title stack.

**Skills.** Grouped by category — Languages, Frontend, APIs & Data, Infrastructure. Only list what
a bullet somewhere can back up. The eval counts a skill claimed here but never demonstrated in a
bullet as `SkillsOnly`, which is a weaker signal than not listing it.

**Experience.** One heading per role, then 3–8 bullets. Write **more bullets than any one resume
would use** — this is a pool to select from, not a document to send. Lead with impact and embed
the metric naturally:

```markdown
- Designed and built a deployment pipeline serving 200+ engineers, reducing deploy
  time from 45 minutes to under 10 minutes.
```

Active verbs only. "Designed," "Built," "Led" — never "Responsible for" or "Involved in."

**Quantify without fabricating.** A number you cannot defend in an interview is worse than no
number. If you are unsure of a figure, put it in `eval-config.yaml` under `unverified_metrics` and
the eval will flag it every time it appears.

## `personal-projects.md` — the gap-fill pool

Work history is the primary source for bullets. Personal projects fill **specific gaps**, at full
weight, when they are the only evidence of something a posting asks for.

Per project:

```markdown
## project-name — Short Description

**What you built:** One paragraph on purpose and architecture.
**Tech stack:** The key technologies.
**Architecture decisions:**
- Decision and why
**State:** Active / Archived / In Progress
**Skill gaps this fills:**
- Skill (how this project demonstrates it)
```

**`Skill gaps this fills` is the load-bearing section.** It is a literal lookup table: when a
posting demands something no work bullet covers, the eval checks this list. If a project could
have filled the gap and wasn't used, that is a `GapFillFailure` — and a top-3 one fails the whole
eval. Write these as the skill names a posting would use, not as project features.

## `preferences.md` — what you want, what you'll refuse

Drives filtering and red-flag detection. Sections: Target Role, Compensation, Location, Company
Criteria, Tech Stack Preferences, Target Companies, Anti-Targets, Notes.

Two that do real work:

- **Deal-breakers**, under Company Criteria. These become hard filters. Be specific and be honest
  — a deal-breaker you don't mean costs you roles silently.
- **Anti-Targets.** Patterns rather than names: "roles labeled Senior but scoped as an IC
  team-of-one with no peer senior engineers" catches far more than a company list.

Put the salary floor in as a real number. A vague range cannot filter anything.

## `resume-style.md` — voice rules

Enforced by the judge panel in `/summary-review` and partly by the eval. Covers voice, banned
phrases, bullet style and layout constraints.

Layout constraints (one page, point size, margins, 5–7 bullets on the most recent role) are **not**
enforced by the engine — they are PDF facts. When a tailored resume overflows, the instruction is
to cut the weakest bullet, never to shrink the type.

Keep the banned phrase list here in sync with `eval-config.yaml`. Same list, two consumers.

## `eval-config.yaml` — what the engine checks

Read fresh on every eval run. **There is nothing to restart** — an edit takes effect on the next
`/job` or `/job-eval`. A missing or unreadable file falls back to built-in defaults and the eval
still runs.

Three lists:

```yaml
common_stack_skills:   # exempt from the "unmatched skill" penalty
  - typescript
  - postgresql
```
Table-stakes technologies you can list even when a posting doesn't mention them, without it
counting as padding.

```yaml
banned_phrases:        # any occurrence is an automatic style Fail
  - "track record of"
  - "leveraging"
  - "passionate about"
```

```yaml
unverified_metrics: [] # numbers you haven't confirmed
```
Your own soft figures. If one appears in a tailored resume the eval flags it — a forcing function
to either verify the number or stop using it. Empty by default; populate it as you find soft spots
in your own claims.

The file also accepts tuning keys the templates don't ship: `max_em_dashes`,
`max_sentence_words`, and the page-length bounds (`one_page_min_chars`, `one_page_max_chars`,
`two_page_min_chars`, `two_page_max_chars`).

## `cover-letter-style.md` and `communication-style.md`

Only needed if you send cover letters or answer free-text application questions.
`cover-letter-style.md` defines the structure; `communication-style.md` defines your voice, and is
what `/app-review` checks drafted answers against.

---

## Files that appear on their own

These have no template. The commands create them as you work — don't write them during setup.

| File | Written by |
|---|---|
| `applications/{company}-{role}.md` | `/job` and `/job-auto`, one record per application |
| `applications.md` | **generated** by `scripts/build_application_index.py` |
| `jobhub.db` | the funnel store, created on first use |
| `tailored/{company}/{role}/` | every resume artifact for one application |
| `research/{company}.md` | `/company-research` |
| `offer-timeline.md` | you, once more than one offer is live — all cross-company reasoning |
| `claim-rules.json` | you, whenever you correct a fact the pipeline got wrong |
| `screen-profile.md` | distilled from `preferences.md` for automated triage |

---

## When you're done

```bash
python3 scripts/resume_preflight.py --all
```

Nothing needs to be started first. Then run `/job` against a real posting — the fit evaluation
reads every file above before it does anything, so it will tell you immediately if something is
missing or malformed.

Next: [commands.md](commands.md) for what each command does with all this.
