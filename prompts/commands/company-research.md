---
description: Lightweight company research for job search — stability, culture, salary data, recent news. Uses cheap, fast model passes.
---

Research a company to inform a job application decision. This is a lightweight alternative to a full deep-research pass, scoped to job search needs and kept cheap on purpose.

## Input

The user will provide a company name, optionally with a specific role they're evaluating.

## Step 0 — Resolve who the company actually is. Before any search. Not optional.

**A company name is not an identity.** Search engines resolve a bare name to whichever company is
most famous, which is frequently not the one hiring. Every agent you launch inherits that error at
once, and the output looks completely normal.

This has happened. On 2026-08-25 three agents were launched against **`neon.tech`** (serverless
Postgres, acquired by Databricks) when the application was to **`neonpay.com`** (payments for game
publishers). It was caught only because someone read the posting body afterwards. Two days later the
wrong company's acquisition was still sitting in the application record and nearly went into an
interview. The same collision poisons every downstream source: `levels.fyi/companies/neon` is a
Brazilian bank, and a "Neon lays off 210 employees" headline is that bank too.

So, first:

1. **Get the canonical domain from the posting**, not from a search. The job posting URL, the
   careers-page host, or the `resume`/`packet` path in `user/applications/{company}-{role}.md`. If
   you cannot establish the domain, **stop and ask** — do not proceed on a guess.
2. **Write one sentence on what they actually do**, taken from the posting body. "Payments for game
   publishers" is what makes a wrong-company result obvious three searches later.
3. **Search for the collisions on purpose.** `"{company}" company` and see what else comes back. Name
   the ones you find.
4. **Put that identity block at the top of every agent prompt**, with the collisions listed as
   explicit exclusions, and instruct each agent that **every search must be disambiguated** —
   `{domain}`, `site:{domain}`, `"{company}" {what they do}` — and that any source it cannot
   attribute to the right entity must be **discarded and reported as discarded**, not used.
5. **The output file must open with that identity block** — canonical domain, one-line description,
   and the collisions ruled out. `user/research/neon.md` and `user/research/talkspace.md` are the
   shape to follow. A brief without it is not finished.

A bare `"{company} layoffs 2026"` search is not research when the name collides. It is three agents
confidently describing someone else.

## Process

Run 4 research passes, each with a focused search task. Run them as parallel subagents on a fast, cheap model if your harness supports it; otherwise run them sequentially in this conversation. See **Delegation** in `AGENTS.md`.

### Agent 1: Company Stability & Financials
Use web search to find:
- Funding stage, last round, valuation (if private)
- Revenue/profitability signals (if public: market cap, revenue growth)
- Recent layoffs or hiring freezes (search "{company} layoffs 2025 2026")
- Headcount trend (growing, flat, shrinking)
- Any recent acquisitions, leadership changes, or pivots

Report: one paragraph summary + a stability verdict (Strong / Stable / Caution / Avoid) with reasoning.

### Agent 2: Engineering Culture & Tech Stack

**Scope every review claim to engineering. A pooled company rating is not evidence about engineering
and must never be reported as if it were.** Established 2026-08-28 on Hims & Hers, where a 249-review
average mixing pharmacy, support and fulfillment staff had been carrying an org-health verdict.

Use web search to find:
- Engineering blog posts (search "{company} engineering blog")
- Tech stack details (search "{company} tech stack engineering")
- **Function-scoped reviews** — Glassdoor has a per-title reviews page
  (`/Reviews/{company}-Software-Engineer-Reviews-...`). Search for it explicitly. Also search for the
  *specific* claims rather than a general vibe: `"{company}" engineer review layoff`,
  `... favoritism promotion`, `... codebase conventions tech debt`, `... hours culture`
- **Blind, and note why it matters** — its population skews heavily to tech, so a Blind rating is a
  better engineering proxy than a pooled Glassdoor average. Report `n` every time; n<20 is weak
- Remote work policy
- Interview process notes (search "{company} software engineer interview process") — **and whether
  live coding is algorithmic or practical.** That distinction decides Andre's prep and sometimes
  whether to pursue at all

Report: what it is like to work there **as an engineer specifically**, plus confirmed tech stack.
**Record positives as well as negatives** — a brief that only collects the bad case is not evidence,
it is a prosecution.

### Agent 3: Compensation Data
Use web search to find:
- Levels.fyi data (search "levels.fyi {company} staff software engineer" or the specific role/level)
- Glassdoor salary range (search "glassdoor {company} staff software engineer salary")
- Blind discussions (search "teamblind {company} staff engineer compensation")
- Any public salary bands from job postings

Report: salary range (base + total comp if available) for the target level, with source attribution. Flag if data is stale (>1 year old).

### Agent 4: Leadership — named people, tenures, and formation

Added 2026-08-28. Org Health Screen dimension 1 is the highest-weighted dimension and cannot be
answered from a title on LinkedIn.

- **Name the CEO, CTO, VP Eng.** Exact appointment dates, current status, **and their title in full**
  — "VP of Engineering, *Growth*" is a different job from "VP Engineering," and it decides who the
  reporting chain runs through
- **Count churn in the C-suite over 3 years**, with dates. A short tenure is a finding: on Hims, a
  20-year Amazon veteran was hired as COO and gone in **under six months**
- **Where did leadership come from, and what did they do there?** Prior-company formation predicts the
  regime. Search the previous employer plus "layoffs," "restructuring," "culture"
- **Cluster events by month and look for convergence.** The single most valuable finding on Hims came
  from noticing that a CTO hire, a COO hire, a COO exit and an engineering layoff all landed in **May
  2025** — which turned anonymous "they want an Amazon culture" review text into something
  corroborated by first-party appointment records
- **Date the negative signal.** If it all post-dates a regime change, say so: that is current
  conditions, not history. If it all predates one, that is the opposite finding and equally important

## Evidence standards — apply to every brief

Established across the 2026-08-28 Hims & Hers pass. These are not style notes; each one exists
because its absence produced a wrong conclusion.

1. **Label sourced claims and inferences differently, in the text.** "The timing is exact, but no
   source names him — this is inference" is a usable finding. The same sentence without that clause
   is a fabrication risk.
2. **A collection failure is never a finding about the company.** Glassdoor returned HTTP 403 on every
   direct fetch across two separate days. The honest line is "there is no SE-specific rating **in what
   I could collect**," never "there is no SE-specific rating." Same failure class as reporting a
   truncated extract as a missing salary range.
3. **Score against `preferences.md` flag by flag, not as a narrative.** Walk the five "draining" flags
   and all six Org Health Screen dimensions, each with its own verdict and the quote behind it. A
   paragraph of vibes hides which specific thing failed — and hides when a dimension fails in an
   unexpected direction.
4. **Re-scoping evidence can move a verdict in the company's favour, and that must be reported too.**
   Scoped to engineering, the Hims favoritism finding went from "corroborated fail" to partial,
   because the vivid quote turned out to be about the operations org. Report the downgrade as loudly
   as an upgrade.
5. **Check comp against the target, not against the posting.** `$170-190K` reads fine until it is set
   beside a stated `$250K+` TC target and the company's own higher band on a sibling req.
6. **When new evidence lands, update the fit report.** `jobhub_db.patch_fit_report(conn, id, ...)`
   takes `verdict_summary`, `why_apply`, `level`, `location`, `posting_url`. A wrong signal can now
   be removed with `delete_fit_signal` — the old HTTP API had no DELETE and could not. `verdict` is
   deliberately not patchable: record a new report instead of rewriting a reported one.

## Synthesis

After all 3 agents return, synthesize into a **Company Brief** with these sections:

1. **Overview** — one sentence: what the company does, size, stage
2. **Stability** — verdict + key signals (funding, headcount, recent news)
3. **Engineering Culture** — what engineers say, tech stack, remote policy
4. **Compensation** — salary range for the target level with sources
5. **Red Flags** — anything concerning (layoffs, glassdoor complaints, leadership churn). "None identified" if clean.
6. **Bottom Line** — one sentence recommendation: worth applying, proceed with caution, or skip

Write the brief to **`user/research/{company}.md`. That file is the record — there is no second
write.**

**There is no second write, and no endpoint.** `research_briefs` moved to files on 2026-08-26
(`docs/state-consolidation-design.md`: *"the table stops being written"*), and the server itself was
retired on 2026-09-11. A `POST /api/research` block lived here for two days after the migration and
caused an agent to conclude the *rule* forbidding the POST was the stale file and rewrite it. The
block is gone rather than kept as history: a payload shape nobody can run is an invitation to try.

Re-running research overwrites `user/research/{company}.md`. That is the update path.

$ARGUMENTS
