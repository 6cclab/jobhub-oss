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

Run 3 research passes, each with a focused search task. Run them as parallel subagents on a fast, cheap model if your harness supports it; otherwise run them sequentially in this conversation. See **Delegation** in `AGENTS.md`.

### Agent 1: Company Stability & Financials
Use web search to find:
- Funding stage, last round, valuation (if private)
- Revenue/profitability signals (if public: market cap, revenue growth)
- Recent layoffs or hiring freezes (search "{company} layoffs 2025 2026")
- Headcount trend (growing, flat, shrinking)
- Any recent acquisitions, leadership changes, or pivots

Report: one paragraph summary + a stability verdict (Strong / Stable / Caution / Avoid) with reasoning.

### Agent 2: Engineering Culture & Tech Stack
Use web search to find:
- Engineering blog posts (search "{company} engineering blog")
- Tech stack details (search "{company} tech stack engineering")
- Developer experience reputation (search "{company} engineer glassdoor reddit")
- Remote work policy
- Interview process notes (search "{company} software engineer interview process")

Report: one paragraph summary of what it's like to work there as an engineer, plus confirmed tech stack details.

### Agent 3: Compensation Data
Use web search to find:
- Levels.fyi data (search "levels.fyi {company} staff software engineer" or the specific role/level)
- Glassdoor salary range (search "glassdoor {company} staff software engineer salary")
- Blind discussions (search "teamblind {company} staff engineer compensation")
- Any public salary bands from job postings

Report: salary range (base + total comp if available) for the target level, with source attribution. Flag if data is stale (>1 year old).

## Synthesis

After all 3 agents return, synthesize into a **Company Brief** with these sections:

1. **Overview** — one sentence: what the company does, size, stage
2. **Stability** — verdict + key signals (funding, headcount, recent news)
3. **Engineering Culture** — what engineers say, tech stack, remote policy
4. **Compensation** — salary range for the target level with sources
5. **Red Flags** — anything concerning (layoffs, glassdoor complaints, leadership churn). "None identified" if clean.
6. **Bottom Line** — one sentence recommendation: worth applying, proceed with caution, or skip

Write the brief to `user/research/{company}.md` for local reference, then **POST to the JobHub server** for persistence:

```bash
curl -s -X POST $JOBHUB_URL/api/research \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{
    "company": "{company name}",
    "stability_verdict": "{strong|stable|caution|avoid}",
    "stability_notes": "{key signals}",
    "stage": "{Series X / Public / etc}",
    "headcount": "{approximate}",
    "founded": "{year}",
    "remote_policy": "{policy}",
    "culture_notes": "{engineer experience summary}",
    "tech_stack": ["lang1", "lang2", "framework"],
    "salary_range_text": "{e.g. $200k-$260k base}",
    "salary_source": "{Levels.fyi / Glassdoor / etc}",
    "researched_at": "{ISO-8601 date}",
    "raw_markdown": "{full brief markdown}"
  }'
```

The server upserts by company name — re-running research updates the existing brief.

If a fit report is being generated in the same session, the server automatically links the research brief to the fit report's sidebar.

$ARGUMENTS
