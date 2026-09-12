# Automate the JobHub funnel: scan → triage → packet

## Context

Manual job searching has stopped working. The `/job` command already does excellent work per
role, but it only runs when Andre drives it step by step, and each run costs a full Opus context.
The result is a funnel that produces two or three deeply-tailored applications a week when
`preferences.md` (2026-08-20) now states plainly that **breadth is the strategy**: "five warm
channels produced two interviews and no offers, so optimising any single application is not the
lever; volume and channel breadth are."

Nothing in the repo carries a role from *found* to *ready to submit*. `scripts/ats_scan.py` finds
Workday/Oracle roles but only prints them. The 77 tracked Greenhouse/Ashby/Lever boards are only
scanned when asked. There is no dedup against what's already been seen, no queue, and no
per-application packet.

**Intended outcome:** a scheduled daily scan across every API-reachable channel, triaged by local
Ollama models at zero token cost, feeding a queue from which Andre picks 5-8 roles. For each pick,
the existing `/job` pipeline produces a self-contained packet folder with everything needed to fill
out the portal by hand.

### Decisions made (2026-08-20)

| Decision | Choice |
|---|---|
| Submission | **Packets only. No browser driving, no auto-submit.** Andre fills and submits every portal himself. |
| Cadence | Scheduled end-to-end scan + triage; packet building on the roles he promotes. |
| Channels | Greenhouse/Ashby/Lever (API) + Workday/Oracle (API). **LinkedIn/Indeed dropped for now.** |
| Warm-path check | Not selected. One checklist line in the packet, no automation. |
| Volume | 5-8 packets per batch. |
| Cron host | This laptop, `launchd`. |
| Grunt work | Local Ollama. |

### Ollama — verified, not assumed

`$OLLAMA_HOST` = `http://192.168.3.168:11434`, reachable, 16 models.

**Model: `qwen3.8:latest`** (27.3B, Q4_K_M, 17.7GB). Chosen after benchmarking it against
`qwen3:14b` on real triage cases:

| | `qwen3:14b` | **`qwen3.8:latest`** |
|---|---|---|
| Structured output | valid JSON | valid JSON |
| Cold load | 108s | 11s |
| Generation | 4.2s | 4.1s |
| Reason quality | self-contradictory | clean, one sentence |

It is faster *and* better, so there is no tradeoff to weigh. Measured throughput on an 8-case
batch at `ThreadPoolExecutor(4)`: **44.8s wall clock, ~5.6s effective per posting.** A few hundred
postings triage in well under an hour, unattended.

#### Two findings from the benchmark that change the design

**1. The AI screen must be charter-based, and the model does not infer that on its own.** With a
naive exclusion list it scored 7/8 — and the miss was the Hims & Hers *Senior SWE, Developer
Platform* req, which Andre actually applied to on 2026-08-17. It dropped the role because "AI"
appeared in the text. `preferences.md` is explicit that the line is *"the charter, not the tools in
the stack"*: a platform/DX team that builds AI tooling is **in scope**; a team whose reason for
existing is AI is not. The screen profile must carry that distinction verbatim, plus a **when
uncertain, KEEP** bias — a wrongly-kept role costs one line of review, a wrongly-dropped one is
lost silently. Re-tested with both: the Hims & Hers case now passes, and the remaining miss
(kept an AI Platform role) errs in the safe direction, exactly as the bias intends.

**2. `keep` and `tier` contradicted each other** — several rows returned `keep=true` with
`tier="drop"`. Fix the schema rather than the prompt: `tier` becomes `strong|good` only, and
`keep` is the sole gate. The contradiction then cannot be represented.

Both are cheap to build in and expensive to discover later, which is why they are here.

### The appeal layer — Haiku second-opinions every drop

The keep-bias reduces false drops but does not remove them, and a false drop is invisible: the
role never reaches the queue and nobody knows it existed. So **every drop that came from the
model's judgement gets a second opinion from Haiku.**

Verified end to end. No `ANTHROPIC_API_KEY` is set, but the `claude` CLI (2.1.220) is authenticated
via `~/.claude/.credentials.json`, so `claude -p --model haiku --output-format json` runs headless
off the existing subscription — no separate key, no billing setup.

On a batch of 8 real drops it scored **8/8**, and overturned two the local model got wrong:

- the Hims & Hers Developer Platform req (dropped on "AI-assisted workflows")
- a Ramp NYC 3-day onsite req (dropped on "onsite" when NYC commute is explicitly in scope)

That second one is a false-drop class the AI rule alone would never have caught, which is the
argument for the layer.

**Cost — and the design constraint that falls out of it.** A single-posting call costs **$0.026**.
An eight-posting call costs **$0.027**. Cost is dominated by the CLI's fixed system-prompt
overhead, not by the postings, so **appeals must be batched — 25 per call.** Unbatched, ~150 drops
a day is $80-150/month and the whole token-conservation goal is defeated; batched, it is roughly
**$0.16/day**.

**Only judgement drops are appealed.** Drops from the hard Python vetoes (defense, Seeq, 10+ years,
management-only) are certain and are skipped — appealing them would spend money to re-confirm a
rule. This also keeps the batch count down.

Overturned roles enter the queue flagged `appealed:true` with Haiku's reason, so Andre can see
what was rescued and judge whether the screen profile needs tightening.

**The Claude/Ollama split:**

- **Ollama (`qwen3.8`, free):** first-pass keep/drop, fit tier, salary extraction,
  requirement-term extraction. Classification and extraction only.
- **Haiku (batched, ~$0.16/day):** appeal every judgement drop. Catches the local model's false
  negatives before they vanish.
- **Claude (Opus, in session):** resume tailoring, summaries, cover letters, screening answers,
  adversarial term verification, fit-report prose. Anything Andre or an employer reads.

**Hard exclusions stay in Python, never in the model.** Seeq, defense/military contractors,
politically aligned orgs, 10+ years required, management-only. A quantized model must not be the
thing enforcing a deal-breaker. The AI screen is deliberately **not** in this list — it needs
judgement about charter, so it stays with the model under the rule above.

---

## Work

### 1. `user/screen-profile.md` — distilled screening profile (new)

`preferences.md` is 370 lines of reasoning trail. It cannot be sent to a local model once per
posting. Write a ~40-line distillation: target titles, level band, in-scope geography, stack,
role-shape signals, and the exclusion list. **Derived from `preferences.md`, never authoritative
over it** — a header must say so.

~~`PERSONAL_KB_URL` is set, so per `prompts/rules/keep-kb-in-sync.md`, run
`python3 scripts/kb_sync.py screen-profile.md` in the same turn it is created, then verify with a
`kb_search`.~~ **Superseded 2026-08-26: the knowledge base was retired and both the rule and the
script are gone. `screen-profile.md` is read off disk by `scripts/triage.py`; there is nothing to
re-ingest.**

### 2. `scripts/ats_sources.py` — shared source registry (new, extracted)

Move `EMPLOYERS`, `REGION`, `ENG`, `LEVEL`, `EXCLUDE`, `workday()` and `oracle()` out of
`scripts/ats_scan.py` into an importable module. `ats_scan.py` keeps working by importing from it —
it stays useful as a standalone probe. Add Greenhouse/Ashby/Lever fetchers alongside, using the
endpoint shapes already documented in `user/boards.md`:

- Greenhouse `https://boards-api.greenhouse.io/v1/boards/{slug}/jobs?content=true`
- Lever `https://api.lever.co/v0/postings/{slug}`
- Ashby `https://api.ashbyhq.com/posting-api/job-board/{slug}`

### 3. `scripts/scan.py` — unified multi-channel scanner (new)

1. `GET $JOBHUB_URL/api/boards` for the 77 tracked boards (verified reachable, HTTP 200).
2. Fetch every board concurrently plus the `EMPLOYERS` Workday/Oracle set.
3. **Deterministic pre-filter** using the existing `ENG`/`LEVEL`/`EXCLUDE`/`REGION` regexes — cuts
   thousands of raw reqs to hundreds at zero cost before any model sees them.
4. **Dedup** against `user/search-results/.seen.jsonl` (posting URL → first-seen date). There is no
   `GET /api/search-results` endpoint, so dedup is local-file based — this deliberately avoids a Go
   server change.
5. Emit `candidates.json`.

### 4. `scripts/triage.py` — Ollama triage (new)

- Loads `user/screen-profile.md` as the system prompt — including the charter-based AI rule and
  the keep-when-uncertain bias.
- Per posting: `POST $OLLAMA_HOST/api/chat` with `stream:false`, `think:false`,
  `keep_alive:"30m"`, and a `format` JSON schema for `{keep, tier, salary_min, salary_max,
  salary_disclosed, level_tag, domain_tag, tags[], reason}`. **`tier` is `strong|good` only** —
  see finding 2.
- `ThreadPoolExecutor(4)`, `--model` flag defaulting to `qwen3.8:latest`.
- Applies the hard Python exclusions **after** the model, as a veto the model cannot override.
- POSTs kept results to `$JOBHUB_URL/api/search-results` using the payload shape in
  `prompts/commands/job.md` (Job Search step 8), and writes the flat-file fallback per the
  dual-write rule.
- Prints the dashboard batch URL.

### 4b. `scripts/appeal.py` — Haiku appeal pass (new)

- Takes `triage.py`'s drop list; **skips anything killed by a hard Python veto** (certain, not worth
  appealing).
- Batches the remainder **25 per call** into `claude -p --model haiku --output-format json`, using
  the numbered-list prompt shape validated in the benchmark. Parses the JSON array, tolerating the
  ```json fence the CLI wraps around it.
- Overturned roles rejoin the queue with `appealed:true` and Haiku's reason.
- Logs appeal counts and cost to the digest so the spend stays visible rather than silent.
- Degrades gracefully: if the `claude` CLI fails or is unauthenticated, log it loudly to the digest
  and ship the queue without appeals rather than failing the scan.

### 5. `scripts/run_daily_scan.sh` + `scripts/launchd/` (new)

Shell wrapper: scan → triage → appeal → POST → write a digest to
`user/search-results/{date}-digest.md` → `terminal-notifier` ping. Commit a
`com.andrepato.jobhub-scan.plist` template under `scripts/launchd/` with install instructions in
the README; the live copy goes in `~/Library/LaunchAgents/` (not committed — it carries the API
token path). Fails loudly into the digest if Ollama or JobHub is unreachable, per
`rules/surface-failures.md`.

### 6. `prompts/commands/job-auto.md` — batch packet builder (new)

The one piece Claude runs. Takes the queue, Andre names 5-8, and for each role it **calls the
existing `/job` flow unchanged** — Fit Evaluation → Resume Tailoring → **the hard eval gate** →
PDF → cover letter → screening answers with the tone-review gate → application tracking POST.

Nothing about `prompts/commands/job.md` changes. `prompts/rules/job-eval-gate.md` applies in full:
the eval is a hard gate before PDF generation, and per that rule a term the matcher misses is a
drafting problem, not a gap in Andre's record.

Per-role verification passes (adversarial term checking, tone review) delegate to subagents on a
cheap model per **Delegation** in `AGENTS.md`.

### 7. `packet.md` per role (new output, in the existing folder layout)

Written to `user/tailored/{company}/{role}/packet.md` alongside the existing `resume.pdf` /
`cover-letter.pdf`. This is the artifact Andre works from:

- Portal URL and ATS type
- **Field sheet** — copy-paste values from `user/config.yaml`: name, email, phone,
  **location "New Jersey"** (decided 2026-08-18, never a NYC address), LinkedIn, GitHub, work
  authorization, sponsorship = no, notice = immediate
- Every screening question on that posting, pre-answered in his voice, tone-gate passed
- Fit verdict + the six Org Health Screen dimensions
- Three questions to ask on the first call
- Comp: the number as fact, never as a gate (the floor is withdrawn)
- One line: *"Warm path checked? LegalZoom alumni / former GA instructors."*

---

## Files

| Path | Change |
|---|---|
| `user/screen-profile.md` | new — distilled screening profile |
| `scripts/ats_sources.py` | new — extracted from `ats_scan.py`, plus GH/Ashby/Lever fetchers |
| `scripts/ats_scan.py` | modified — import from `ats_sources`, behaviour unchanged |
| `scripts/scan.py` | new — multi-channel scan, pre-filter, dedup |
| `scripts/triage.py` | new — Ollama triage + POST to JobHub |
| `scripts/appeal.py` | new — batched Haiku second-opinion on drops |
| `scripts/triage_cases.py` | new — 8-case triage regression test |
| `scripts/run_daily_scan.sh` | new — cron wrapper + digest |
| `scripts/launchd/*.plist` | new — template |
| `prompts/commands/job-auto.md` | new — batch packet builder |
| `AGENTS.md`, `README.md` | modified — document the new pipeline and `OLLAMA_HOST` |

Reused as-is, not rewritten: `prompts/commands/job.md`, `prompts/commands/job-eval.md`,
`prompts/commands/app-review.md`, `scripts/build_resume.py`, ~~`scripts/kb_sync.py`~~ (retired
2026-08-26),
`templates/resume.*`, `templates/cover-letter.*`, and the whole `server/` eval engine.

**No Go server changes.**

---

## Verification

1. **Ollama** — `curl $OLLAMA_HOST/api/tags`; re-run the structured triage call and confirm valid
   JSON. Already passing on `qwen3.8:latest`.
   **Keep the 8-case benchmark as a regression test** (`scripts/triage_cases.py`) — it caught the
   charter bug. Every change to `screen-profile.md` re-runs it; the Hims & Hers Developer Platform
   case must stay `keep=true`.
2. **Scan** — `python3 scripts/scan.py --dry-run`; confirm it reaches all 77 boards plus the 12
   Workday/Oracle employers, and report per-source errors rather than swallowing them.
3. **Dedup** — run `scan.py` twice; the second run must return zero new candidates.
4. **Triage** — run over one real batch; hand-check ~10 keep/drop calls against `preferences.md`.
   Confirm a Seeq posting is dropped by the **Python veto**, not by the model.
5. **Appeal** — run `appeal.py` over a real drop list. Confirm batches are 25, that hard-veto drops
   are excluded, that overturns land in the queue flagged `appealed`, and that the digest reports
   the cost. Sanity-check the per-run spend against the ~$0.16/day estimate; if it is materially
   higher, the batching is not working and should be fixed before the cron is installed.
6. **POST** — confirm the batch appears at `$JOBHUB_URL/search-results/{id}` and the flat file was
   written.
7. **Packets** — build one end-to-end. Confirm: eval gate ran and passed (dashboard URL printed),
   `resume.pdf` and `cover-letter.pdf` exist, `packet.md` has every portal field, location reads
   "New Jersey", and the tone gate passed on every screening answer.
8. **Cron** — `launchctl kickstart` the job manually and confirm the digest and notification land.
9. ~~**KB** — `python3 scripts/kb_sync.py screen-profile.md`, then `kb_search` for changed
   wording.~~ **Superseded 2026-08-26 — the knowledge base was retired; this step no longer exists.**

---

## Risks, stated plainly

- **Volume is not response rate.** The record is 0-for-11+ on cold portal submissions. This plan
  raises throughput; it does not fix the cold-channel conversion problem, and it should not be
  reported as if it does. The warm-path check that `preferences.md` says to default to was not
  selected for automation — it stays a manual checklist line.
- **The local model will make bad calls — this is measured, not hypothetical.** It scored 7/8 on
  the benchmark in both configurations. Mitigated by the deterministic pre-filter running first,
  the hard exclusions vetoing after, the keep-when-uncertain bias pushing errors toward
  over-inclusion, and Andre reviewing the queue before any packet is built. Nothing reaches an
  employer on the local model's judgement.
- **The laptop must be awake** for the scan to run. Accepted as the cost of not deploying.
- **ATS endpoints drift.** `ats_scan.py` already handles this by reporting per-employer errors;
  `scan.py` will do the same rather than silently returning fewer results.

---

## Findings from the first real run — 2026-08-21

The full pipeline ran end to end for the first time. Three things the plan got wrong, recorded
here rather than quietly corrected.

### The Staff filter was not in the pipeline at all

`screen-profile.md` said *"Staff is in scope when the fit is strong."* `preferences.md` says the
opposite and has since 2026-08-21: a hard filter, do not surface, do not argue for an exception.
`vetoes.py` only cut Senior Staff and Principal.

**Result: 92 of 266 kept roles were Staff-titled** — a third of the queue was roles Andre had
already decided not to apply to. The filter now lives in `vetoes.py`, in Python, because a
quantized model asked to weigh "fit" against a ban will keep finding strong fits; strong fit is
exactly when the ban is tempting. So it is not asked. Multi-band reqs (`Senior / Staff`,
`Senior or Staff`) survive, because levelling during the process is not applying to Staff.

The filter is evidence-backed, not merely a preference: Seeq rejected him from a Staff req saying
they expected someone more technical for the level.

### Appeal cost is ~9x the estimate, and the estimate's inputs were both wrong

Planned $0.16/day. Actual first run: **$1.40**, 472 appeals in 19 batches of 25.

Batching works correctly — the log shows 19 full batches, no fallback. The estimate was wrong
twice over: it assumed ~150 drops/day (the run had 472, because a cold dedup ledger made every
posting new) and $0.027/batch (the benchmark measured an *8*-posting call; 25 postings cost
~$0.074).

**Steady-state cost is still unmeasured.** With 845 entries now in `.seen.jsonl`, subsequent runs
only appeal genuinely new drops. Measure it on the second run rather than assuming either figure.

### The vetoes belonged before the model, not after

`triage.py` applied them after scoring, so the model was paid to read 310 of 739 candidates that a
regex could refuse — 284 of them Staff. They now run in `scan.py` before the model, cutting a
triage pass from ~17 to ~10 minutes. `triage.py` still applies them afterwards as belt-and-braces.

Per-reason counts print in the scan output and land in the payload: a filter nobody can see is how
a funnel narrows without anyone noticing.

### A new veto masked three existing tests

Adding the Staff veto broke `triage_cases.py` in a way worth naming, because it will happen again.
Three fixtures used Staff titles, written when Staff was in scope. `geo_philly_onsite_ok` failed
outright. **`charter_true_ai_org` and `veto_ten_years` kept passing — via the Staff veto, without
ever exercising the AI charter rule or the years check.** A test that passes for the wrong reason
is worse than one that fails.

All three are Senior now, and the Staff filter has its own cases. **When you add a veto that fires
early, check what it stops other cases from reaching.**

`scripts/vetoes_cases.py` is new: 28 title cases, no model needed, runs in a second. Two regex
attempts at the Staff filter passed review and failed on real titles — `\bsr\.?\b(?!\s+staff)`
backtracked the period and let `Sr. Staff` through; `\s*(?!staff)` matched zero characters and let
`Senior Staff` through. Neither was caught by reading. Run it after any change to `vetoes.py`.

### The dedup ledger meant "fetched", not "seen" — 2026-08-21

The afternoon test run overwrote `2026-08-21-scan.md`, replacing a 63KB / 266-role digest with a
1.6KB / 4-role one. The roles were unrecoverable by re-running, because `scan.py` had already
recorded all 845 postings in `.seen.jsonl` **at fetch time**. Scanned, recorded, digest destroyed,
gone.

Andre: *"no silent losses. If I've seen it then dont go get it again."*

Two changes:

- **`post_results.py` refuses to overwrite a digest.** A same-day rerun writes
  `{date}-scan-{HHMM}.md`. `job-auto.md` now globs `{date}-scan*.md` and takes the newest, and is
  told that earlier same-day digests are live queues rather than history.
- **The ledger is written by `post_results.py` after the digest exists, not by `scan.py` at fetch
  time.** It now means "this reached him". A crash, an Ollama outage, or a failed digest leaves the
  postings unrecorded and they return on the next run. `scan.py --record` still exists for
  standalone use but is off by default.

**Kept and dropped are both recorded** — a drop is a decision, and re-triaging it tomorrow would
spend model time reaching the same answer. **Vetoed and rejected-company postings are not**, which
is deliberate: re-checking them costs a regex, and it means a veto rule change resurfaces everything
the old rule excluded. The Staff filter change earlier that day could not resurface anything, and
that was the wrong behaviour.

Verified: a standalone `scan.py` run left the ledger at 853 lines unchanged; `post_results.py`
wrote 3 postings (2 kept, 1 dropped) to a patched ledger only after the digest landed, and still did
so when the server POST failed, because the local digest is the fallback the user reads.

### 30-day TTL on the ledger — 2026-08-21

Andre: *"we should never re-fetch listings within 30 days."*

The ledger was permanent: once a posting was recorded, nothing ever brought it back. It now expires
after `SEEN_TTL_DAYS = 30`, checked at read time so the file stays an append-only record of what was
surfaced and when. A req still open after a month can come round again — it may have been reposted,
rescoped, or simply missed.

**Vetoed and rejected-company postings are now recorded too**, at scan time, tagged with the reason.
This reverses the call made a few hours earlier and it is the direct consequence of the 30-day rule:
leaving them out meant re-processing 310 deterministic exclusions on every single run. They are not
a silent loss — the exclusion is a stated rule and the reason is written next to the row.

The cost is real and is worth naming: **a veto rule change no longer resurfaces affected roles for
up to 30 days.** That is what `--forget` is for:

```bash
python3 scripts/scan.py --forget 'Staff+ title'   # after changing the Staff rule
python3 scripts/scan.py --forget all              # every reason-tagged entry
```

`--forget` only touches rows carrying a `reason`. Roles that reached a digest have no reason field
and are never removed, because forgetting those would resurface work already reviewed.

Verified against a synthetic ledger: 1d and 29d entries still seen, 31d and 200d expired, an entry
with an unparseable date treated as permanently seen rather than resurfacing every run,
`--forget 'Staff+ title'` removing only the Staff row while leaving the defense veto and both digest
rows intact. Then against the live boards: a real scan recorded 10 exclusions with reasons
(8 Staff, 1 mobile, 2 Reddit — 1 overlap), the next scan no longer re-processed them, and the two
candidates that reached the pipeline without a digest stayed out of the ledger, as they must.
