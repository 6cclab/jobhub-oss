---
description: Build application packets in batch from the automated scan queue — fit, tailored resume, eval gate, PDFs, screening answers, and a portal field sheet
---

You are building **application packets** from the automated scan queue: everything the user needs
to fill out a job portal by hand, assembled ahead of time.

**This command does not submit applications.** No browser driving, no clicking Submit. The
deliverable is a folder per role. The user reads it, fills the portal, and submits. This is a
deliberate decision, not a limitation to work around — **do not offer to submit**, and do not drive
a browser to an application form while running `/job-auto`.

**The scope of that rule is this command, not the harness.** If the user asks directly — "can you
apply for me" — that is a different mode and it is allowed. It was used on 2026-08-23 to submit
nine applications. Read `prompts/rules/ats-portals.md` before driving any portal; it records what
works, what silently fails, and the one thing that is a hard stop. The standing constraint that
never relaxes: **never bypass a CAPTCHA or an emailed human-verification code**, whatever the user
says.

## Where the queue comes from

`scripts/run_daily_scan.sh` runs scan → triage → appeal → post each morning. Its output is:

- the newest digest matching `user/search-results/{date}-scan*.md`
- a search-results batch on the dashboard at `$JOBHUB_URL/search-results/{id}`

**Glob for it and take the newest by modification time — do not assume `{date}-scan.md`.** A second
run on the same day writes `{date}-scan-{HHMM}.md` rather than overwriting, because on 2026-08-21 an
afternoon run replaced the morning's 266-role queue with a 4-role one and it could not be recovered.

Read the most recent digest. If none exists or it is stale, say so and offer to run
`./scripts/run_daily_scan.sh` rather than silently working from old data.

**Older digests on the same day are still live queues, not history.** Their roles are marked seen and
will not be re-scanned, so if the newest digest is small, check whether an earlier one holds roles the
user has not worked through yet.

## Step 1 — present the queue

Show the kept roles as a compact table: tier, company, role, location, salary, and the one-line
reason. Mark the two flags the digest carries:

- **⤴ appealed** — the local model dropped this and Haiku overturned it. Worth a second look; the
  local model's reasoning was found wrong.
- **⚠ prior rejection** — this company already rejected an application. Not filtered out; a
  different team and a different req may still be worth it. **State it once, neutrally, and let
  the user decide. Do not argue either way.**

Then ask which to build. **Default batch is 5-8.** If the user names more than 8, build the first
8 and say plainly that the rest are queued rather than quietly truncating.

## Step 2 — build each packet

For each selected role, run the **existing `/job` pipeline, unchanged and in full**.

**Read `prompts/commands/job.md` and follow its numbered steps.** Do not work from the summary
below and do not reconstruct the order from memory — that file changes, and this command is
deliberately not a second copy of it. As of 2026-08-21 the chain is: fit evaluation with the gap
confirmation gate → tailoring → eval (**once, advisory**) → `/summary-review` → render →
`resume_preflight.py` → `/pdf-review` → application tracking. If that has changed, `job.md` is
right and this paragraph is stale.

The rules it governs itself by apply here identically — `job-eval-gate.md` and
`summary-review-gate.md` in particular. Two worth restating because batch pressure is exactly when
they get dropped:

- **The eval is advisory and runs once.** Never re-run it to move the score, never edit a resume to
  change a number. A scored loop gets optimized rather than satisfied, and it has already produced
  keyword stuffing. A term the matcher misses is a **drafting problem**, never a gap in the user's
  record.
- **The judge panel and the preflight are not optional, per role.** They were skipped on every
  resume built on 2026-08-20, including two reported as ready. Eight roles is not a reason to run
  the gate seven times. `resume_preflight.py` must exit zero before you call anything ready.

Then, per role, add the packet-specific work:

- **`posting.md`** — archive the listing. See below. Do this **first**, before tailoring, so the
  role is captured even if the rest of the packet fails.
- **Screening questions** — every question on the posting, answered in the user's voice, through
  the mandatory tone-review gate in `job.md`.
- **`packet.md`** — see below.

If a resume cannot pass the preflight after two honest rework rounds, stop on that role, report
why, and move to the next. An eight-packet batch with one declared failure beats eight packets
where one is padded.

Run verification passes (adversarial term checking, tone review) as subagents on a fast, cheap
model where the harness supports it — see **Delegation** in `AGENTS.md`.

## Step 2b — archive the listing to `posting.md`

**Every packet keeps its own copy of the posting.** Write the full listing to
`user/tailored/{company}/{role}/posting.md`.

The queue already carries it: `scan.py` stores the ATS `description` on each candidate and
`triage.py` passes the row through with `dict(p)`, so the body is in the digest and does not need
re-fetching. If it is missing for a source that does not expose one, fetch the posting URL and say
so in the file rather than leaving it empty.

```markdown
# {Company} — {Role}

**URL:** {posting URL}
**ATS:** {greenhouse|ashby|lever|workday|oracle}
**Captured:** {ISO date}   **Posted/updated:** {date from the ATS, or "not exposed"}
**Location:** {as written on the posting}
**Compensation as posted:** {verbatim, or "not disclosed"}

---

{the full listing body, as text}
```

**Why this is not optional.** A URL is not a record. Postings get taken down, edited, or reposted
under a new ID between the scan and the moment the user sits down to apply — and the listing is
what they need again at interview prep, weeks later, to answer "what did they actually ask for?"
A tailored resume whose posting has vanished cannot be explained or defended.

This has already cost something concrete: the Courier Health application recorded on 2026-08-21 has
no posting URL stored anywhere in the repo, because nothing in the pipeline was responsible for
keeping it.

**Never paraphrase or summarize the body here.** Verbatim, or an explicit note saying what could not
be captured and why.

## Step 3 — write `packet.md`

Alongside the existing artifacts in `user/tailored/{company}/{role}/`, write `packet.md`. This is
the file the user actually works from while filling the portal. It must be complete enough that
they never need to come back and ask you something.

```markdown
# {Company} — {Role}

**Apply at:** {posting URL}
**ATS:** {greenhouse|ashby|lever|workday|oracle}
**Posted:** {date}   **Location:** {location}   **Comp:** {range or "not disclosed — ask early"}
**Full listing:** `posting.md` (archived {date} — the URL may be gone by the time you read this)

## What they asked for

{5-8 bullets, verbatim phrasing from the posting's requirements and responsibilities.
Enough that the user can answer "what does this role actually want?" without reopening
the URL. Quote them; do not paraphrase into your own words.}

## Field sheet

| Field | Value |
|---|---|
| Name | {from user/config.yaml} |
| Email | {config} |
| Phone | {config} |
| Location | **New Jersey** |
| LinkedIn | {config} |
| GitHub | {config} |
| Work authorization | Authorized to work in the US |
| Sponsorship required | No |
| Notice period | Immediate |

## Attachments
- `resume.pdf`
- `cover-letter.pdf` {omit if not written}
- `posting.md` — reference copy, not submitted

## Screening questions
{each question, then the approved answer, ready to paste}

## Fit
{verdict and one paragraph}

**Org health:** {the six dimensions, one line each}

## Ask on the first call
1. …
2. …
3. …

## Before you submit
- [ ] Warm path checked — LegalZoom alumni or former GA instructors at this company?
- [ ] {any role-specific item, e.g. confirm the comp band is undisclosed}
```

**Location is always "New Jersey".** Never a town, never a NYC address, on any field or document.
This was decided 2026-08-18 and is not a per-role judgement call.

**Comp is information, never a gate.** The floor is withdrawn. Report the number; never skip a
role over it and never frame it as a disqualifier.

## Step 4 — close out

Report per role: packet path, fit verdict, eval verdict with its dashboard URL, and anything left
open. Then give the batch total and name anything that failed and why.

Do not claim a packet is ready unless `resume_preflight.py` exited zero, the PDFs exist on disk, and
`posting.md` holds the listing. Check all three before saying so -- run the check, do not infer it
from the fact that you wrote the file.

$ARGUMENTS
