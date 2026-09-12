# Manual Onboarding

`/onboard` handles this interactively, but if you'd rather set up `user/` by hand (or fix something onboarding got wrong), this is the field-by-field guide. Copy each file from `templates/` to `user/` and fill it in per the notes below. `user/` is gitignored — none of this leaves your machine unless you push it somewhere yourself.

## config.yaml

```yaml
name: "Jane Doe"
email: "jane@example.com"
phone: "555-123-4567"
location: "City, ST"
linkedin: "linkedin.com/in/janedoe"
github: "github.com/janedoe"
sign_off: "Jane"
```

This is the only file every command reads unconditionally — it drives the identity fields on generated PDFs (`{{NAME}}`, `{{EMAIL}}`, `{{PHONE}}`, `{{LOCATION}}`, `{{LINKS}}`) and the cover letter sign-off. `linkedin`/`github` should be bare domains (no `https://`) — the PDF templates prepend the scheme when rendering the link text.

## master-resume.md

This is the bullet pool `/job` pulls from — it's never sent to an employer as-is, only reordered and filtered per posting.

**Structure:**

```markdown
# Your Name

**Your Title** | Location | email@example.com | (555) 123-4567
[linkedin.com/in/you](https://linkedin.com/in/you) | [github.com/you](https://github.com/you)

> MASTER RESUME — reference document. Pull and reorder bullets per application; do not send as-is.

## Summary

<3-5 sentences, first person, conversational — not corporate bio voice>

## Skills

- **Languages:** ...
- **Frontend:** ...
- **APIs & Data:** ...
- **Infrastructure:** ...

## Experience

### Company Name — Location

**Title** — Start Date - End Date

- Bullet 1
- Bullet 2
...

## Education

Institution — Degree/Program, Location
```

**Bullet format:** one accomplishment per bullet, lead with the action and outcome (not the technology), embed metrics in the sentence rather than tacking them on. 3-8 bullets per role — `/job` selects the most relevant 5-7 from your most recent role and 1-2 from prior roles when tailoring.

Two optional sections worth adding once you have real content:
- **"Tailored Summary Variants"** — a few pre-written summary angles `/job` can pull ideas from (it rewrites in your voice rather than copying verbatim)
- **"Note on Unverified Metrics"** — any numbers you're not 100% sure of; `/job` checks this before using a dollar figure or percentage and will ask you to confirm rather than guess
- **"Manager Feedback"** — reference-only, never rendered onto a resume, but used to inform how strengths get framed

## personal-projects.md — the gap-fill mechanism

This file exists specifically so that side projects can substitute for missing work experience when a posting demands a skill your job history doesn't cover. Work experience always wins when both exist for the same skill — this is gap-fill only, not a second resume section by default.

**Structure per project:**

```markdown
## project-name — Short Description

**What you built:** One paragraph — purpose and architecture.

**Tech stack:** List the key technologies.

**Architecture decisions:**
- Key decision 1 and why
- Key decision 2 and why

**State:** Active / Archived / In Progress

**Skill gaps this fills:**
- Skill 1 (explain how this project demonstrates it)
- Skill 2 (explain how this project demonstrates it)
```

**The "Skill gaps this fills" pattern is the load-bearing part.** Every entry under it should name a specific, postable skill — not a vague theme. When `/job` (or `/job-eval`'s gap-fill check) finds a posting requirement with no matching bullet in `master-resume.md`, it scans every project's "Skill gaps this fills" list for a match. If found, it drafts a bullet from that project under a "Selected Projects" section on the tailored resume, framed as "Designed and shipped" (not "Built from scratch" — the framing should acknowledge agentic/AI-assisted development as a strength, not downplay it). If a posting's top-3 requirement has no work bullet AND no project covers it, that's a hard eval failure (`Critical` verdict) — so keeping this section current directly affects whether your resumes pass the eval gate.

## preferences.md

Every field here either filters what `/job` shows you or gets checked as a red flag during fit evaluation:

- **Target Role** (level, title variations, domain priorities) — used to judge whether a posting is a level/domain match
- **Compensation** (salary floor, total comp target, equity preferences) — postings below the salary floor get flagged during job search and as a red flag during fit evaluation
- **Location** (current, remote preference, relocation willingness, time zone) — filters postings and flags location mismatches
- **Company Criteria** (size preference, industry preferences, deal-breakers, must-haves) — deal-breakers are hard filters; must-haves inform the fit verdict narrative
- **Tech Stack Preferences** (preferred / willing to learn / avoid) — informs job search filtering and how gaps get framed (a gap in an "avoid" technology is treated differently than a gap in something you're targeting)
- **Target Companies** — populated from your tracked boards list; top-priority targets
- **Anti-Targets** — patterns to actively screen out (e.g. recent major layoffs, "Staff" roles that are actually solo IC slots, platform teams without exec sponsorship)
- **Notes** — freeform differentiators; anything that makes you stand out beyond job titles (teaching background, a technical niche, etc.)

`/job`'s strategy-discussion mode updates this file automatically when a conversation surfaces new preference info — you don't have to keep it in sync by hand once it's seeded.

## resume-style.md — banned phrases and voice rules

This file (plus its mirror in `eval-config.yaml`, see below) is what makes the eval gate's style check enforceable rather than aspirational.

**Voice rules:**
- Summary must open with "I am" or "I" — third-person resume-speak ("Staff Software Engineer with 7+ years...") fails the eval's `SummaryFirstPerson` check outright
- Talk about what you care about, not just what you accomplished — personality over corporate bio
- Embed metrics in narrative sentences, don't list them
- Mirror the posting's priorities in your own voice — never copy posting language verbatim
- One em-dash per resume max

**Banned phrases** — listed here in prose form (`## Banned Phrases`) and mechanically enforced via the matching list in `user/eval-config.yaml`'s `banned_phrases`. If you add a phrase to one, add it to the other — the eval engine only reads `eval-config.yaml`; `resume-style.md` is what the LLM reads while drafting. Keeping them in sync means the model avoids the phrase before the eval even has to catch it.

**Bullet style:** lead with action + outcome, one accomplishment per bullet, active verbs only ("Designed," "Built," "Led" — not "Responsible for," "Involved in"), quantify without fabricating (cross-check "Note on Unverified Metrics" in master-resume.md).

**Formatting constraints:** one page, 10pt Georgia, 0.6/0.7in margins, 5-7 bullets from the most recent role. These aren't enforced by the eval engine (they're PDF layout constraints) — if a tailored resume overflows, `/job` is instructed to cut weak bullets rather than shrink type.

## eval-config.yaml — what the engine actually checks

This file is loaded by the Go server at startup (`eval.LoadConfig`, falls back to defaults if missing) and drives the deterministic parts of resume scoring. Three lists:

```yaml
common_stack_skills:
  - typescript
  - react
  - postgresql
  # ...
```
Skills here are exempt from the "unmatched skill" penalty in the skills-relevance check — i.e., table-stakes technologies you can list even when a specific posting doesn't mention them, without it counting as padding. Add anything you consider baseline for your stack.

```yaml
banned_phrases:
  - "track record of"
  - "leveraging"
  # ...
```
Any of these appearing anywhere in the resume text is an automatic style `Fail`. This should match `resume-style.md`'s banned phrase list — see above.

```yaml
unverified_metrics: []
```
Your own list of specific numbers/claims you've used before but haven't fully confirmed (e.g. "$2M in savings" if you're not sure that figure is exact). If any of these strings appear in a tailored resume, the eval fails and flags them — a forcing function to either verify the number or stop using it. Empty by default; populate it as you discover soft spots in your own numbers.

Restart the server (or redeploy) after editing this file — it's read once at startup, not per-request.

## After Editing

Once `user/config.yaml`, `master-resume.md`, and `preferences.md` have real content, start the server and run `/job` against a real posting — the fit evaluation step will immediately tell you if anything is missing or malformed, since it reads all of these files before doing anything else.
