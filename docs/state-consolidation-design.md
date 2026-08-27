# Consolidating job-search state: one owner per entity

_Design, 2026-08-26. Implemented 2026-08-26._

**Status: implemented.** `user/applications/` is authoritative; `user/applications.md` is generated
by `scripts/build_application_index.py`. The `/api/applications` endpoints still exist and are no
longer written to — removing them is deliberately deferred.

## The problem

Job-search state is spread across four stores that overlap and are kept in agreement by hand.
Writing one fact means writing it several times, and nothing detects when a write is missed.

| Store | Holds | Read by |
|---|---|---|
| `user/*.md` (gitignored) | `applications.md`, `research/`, `preferences.md`, `master-resume.md`, week sheets | `scripts/resume_preflight.py:310`, `scripts/triage.py:44` — the gate |
| JobHub Postgres | `applications`, `application_events`, `fit_reports`, `fit_signals`, `research_briefs`, `eval_results`, `search_batches`, `search_results`, `companies`, `boards` | the dashboard |
| personal-kb | ~700 chunks — a lossy copy of `user/*.md` | `prompts/commands/job.md:24-27`, optional |
| `~/projects/interview-prep` | stories, drill log, per-company prep | its own commands |

Observed on 2026-08-26, recording a single event — one application's recruiter screen passing, with
the next two rounds and their format:

- It was written **four times**: `user/applications.md`, `user/screens-2026-08-week35.md`, the
  JobHub `notes` column, and an attempted KB ingest.
- The KB ingest **failed silently, twice**. `kb_sync.py` reported a successful upsert and unchanged
  chunk counts; four `kb_search` probes confirmed the indexed copy was two edits stale.
- A **concurrent session** added a different application's row to `applications.md` mid-conversation,
  so collisions on the single table file are real rather than theoretical.
- `application_events` exists in the schema with `from_status`/`to_status`/`note`/`created_at`, and
  has repository and model code in Go, but **has no HTTP endpoint** — `GET
  /api/applications/{id}/events` returns `Cannot GET`. The structured timeline is unreachable, so
  every status change is appended to a free-text `notes` blob instead.

The root cause is not any one store. It is that no entity has a single owner.

## Decisions

Made with Andre on 2026-08-26. Each is a decision, not a recommendation.

| Question | Decision |
|---|---|
| What does the server own? | **The funnel only** — scan results, triage, dedup, queue, fit reports. |
| Where do application records live? | **`user/applications/{company}-{role}.md`** — one file per application. |
| Does `applications.md` survive? | **Yes, as a generated index.** Never hand-edited. |
| Cross-application strategy prose? | **`user/pipeline.md`** — hand-written, never generated. |
| Migration style | **Verbatim or fail.** Nothing is summarised or tidied in transit. |
| `interview-prep` repo | **Out of scope entirely.** Not referenced by this design. |
| personal-kb | **Out of scope**, revisit separately. Not written to in the interim. |
| `user/` in git | **Stays ignored.** Migration protects itself with file-level backups. |

## The ownership rule

**The server owns everything produced before a role is picked. Files own everything produced
after.** Promotion is the copy point.

A role enters as one of hundreds of scan results, is triaged, and sits in a queue. That data is
machine-generated, machine-read, disposable, and far too voluminous for markdown. It stays in
Postgres: `search_batches`, `search_results`, `companies`, `boards`, the dedup ledger, and
`fit_reports` / `fit_signals`, which exist to rank what to promote.

On promotion, the record becomes a file. The fit verdict that justified promotion is copied into
the record **as a value, not a URL**, so the record reads correctly with no server running and the
Postgres row becomes history rather than a live dependency.

Two entities move off the server that are not obviously funnel:

- **`research_briefs` → files.** Research is already double-written as `/research/{uuid}` and
  `user/research/{company}.md`. It is long-lived and re-read constantly. Human-read means files.
  The table stops being written.
- **`eval_results` → the packet.** The eval is advisory and runs once per tailored resume. It
  belongs beside `review.json` in `user/tailored/{company}/{role}/`.

What remains on the server is exactly what the cron funnel writes unattended. `scripts/post_results.py`
(`/api/search-results`) and `scripts/fit_batch.py` (`/api/fit-reports`) already write only funnel
entities and **do not change**.

## The record format

`user/applications/{company}-{role}.md`. Frontmatter carries only what the index renders and a
script can validate. Everything else is prose, unschema'd, because that is where the reasoning
lives and schemas make reasoning worse.

```yaml
---
company: Acme
role: Software Engineer, Backend (Mid/Senior)
status: phone_screen        # server's existing 7-value vocabulary, unchanged
source: recruiter-inbound
submitted: false            # live recruiter process; nothing sent through a portal
applied: 2026-08-24
resume: acme/software-engineer-backend/resume.pdf
packet: user/tailored/acme/software-engineer-backend
server_id: 00000000-0000-0000-0000-000000000000   # history, not a live dependency
fit: strong
events:
  - date: 2026-08-24
    to: inbound
    note: recruiter outbound
  - date: 2026-08-26
    to: phone_screen
    note: screen passed; behavioral then technical
---
```

Block style rather than flow mappings: this repo has no PyYAML, and flow mappings would need real
quoting rules. A value is everything after the first ": " to end of line, so notes need no
escaping.

**Filename slug:** `{company}-{role}.md`, lowercase and hyphenated, reusing the segments already
used by the packet directory — `user/tailored/acme/software-engineer-backend` yields
`acme-software-engineer-backend.md`. Reusing the existing slugs means the record-to-packet link
is derivable rather than invented, and two roles at one company cannot collide.

**`applied` is the date the pursuit began, not the date a portal form was submitted.** For a cold
portal application the two coincide. For a recruiter inbound they do not: the example above has
`applied: 2026-08-24` with `submitted: false`, meaning the process started that day and nothing was
ever sent through a portal. `submitted` is the field that answers "did we send something."

Two deliberate choices:

- **`submitted` is separate from `status`.** A recruiter-inbound role can be a live phone screen that
  was never submitted through a portal, and the current table cannot express that without prose —
  this happens often enough to need a field rather than a sentence. It also encodes
  the standing rule in `prompts/rules/ats-portals.md` — a form that was filled but not submitted is
  not an application.
- **`events` mirrors the `application_events` schema exactly.** If the dashboard is ever revived,
  the mapping is mechanical rather than a rewrite.

**A third choice, made during implementation rather than in the original design session:** a table
row with no `status` column is not a record. `user/applications.md` turned out to hold a table of
rows describing packets that were built but never sent, with a "stage reached" column instead of a
status. Inventing a status for those rows to force them into the record format would have violated
verbatim-or-fail, and it would also have contradicted the standing rule in
`prompts/rules/ats-portals.md` that a form filled but not submitted is not an application. Their
prose moves to `user/pipeline.md` verbatim instead, same as other cross-application content, where
Andre can promote any of them to a record by hand later.

`status` values stay the server's: `applied`, `phone_screen`, `onsite`, `offer`, `rejected`,
`ghosted`, `withdrawn` (`server/internal/handler/api.go:345`).

## The generator

`scripts/build_application_index.py`

- Reads frontmatter from every `user/applications/*.md`.
- Writes `user/applications.md`, newest first, with a `DO NOT EDIT — generated by
  scripts/build_application_index.py` header.
- **Fails loudly on a malformed record rather than skipping it.** Silent skipping is the failure
  mode this design exists to remove.

Because the index is a projection, it cannot drift from the records. Concurrent sessions write to
different record files and regenerate the index idempotently, which is what the single table file
could not survive.

## `user/pipeline.md`

`applications.md` contains 118 lines that are not rows: live counts, a ranked preference ordering
across several employers with Andre's verbatim quotes, timing analysis about one offer likely
landing before a preferred employer had even screened, and channel-level conclusions about which
application sources have ever produced a response.

This is portfolio-level reasoning. It belongs to no single application and a generated
`applications.md` would destroy it. It moves verbatim to `user/pipeline.md`, hand-written and never
generated.

Much of it is stale — live counts and scheduled rounds from dates already passed. **Staleness is
Andre's to curate later, not the migration's to fix.**

## Migration

**Nothing is summarised.** Each row's Notes cell becomes its record's prose body verbatim;
frontmatter is derived mechanically from the columns. The prose contains corrections, verbatim
quotes, and struck-through reasoning trails. Lossy migration by an agent is the exact failure class
this repo keeps being bitten by.

**The design was wrong about one thing, found during implementation.** `user/applications.md` was
not one table but three, each with its own schema: a live table (8 columns), a closed table (6
columns, no `Age`/`Resume`), and a "started but never submitted" table (5 columns, **no `Status`
column at all**). Assuming a single schema would have parsed every row against the wrong header and
either crashed or silently mis-attributed columns. The migration parses each row against its own
table's header instead of one global header. See "The record format" above for what happens to the
table with no `Status` column.

`user/` is gitignored (`.gitignore:1`), so there is no version history to fall back on. Protection
is file-level, following the existing `user/_backup-2026-08-21/` pattern.

**Step 1 — Back up.** Copy `user/applications.md` to `user/_backup-2026-08-26/` before anything is
written.

**Step 2 — Reconcile.** The server and the table disagree on how many applications exist — run
against the live data, reconcile found 33 server records against 25 table rows. The migration
script emits a diff report and **stops**. Andre resolves mismatches before any record is written.

Run for real on 2026-08-26, reconcile found **three applications that had server records and
packets on disk but no row in the tracker.** Andre confirmed all three and backfilled their rows
from the server record and packet evidence; after that, both sides agreed at 28 rows / 33 server
records / 118 pipeline lines. This is the gate doing its job: a silent skip here would have dropped
three real applications from the authoritative store on day one.

**Step 3 — Write.** `scripts/migrate_applications.py` writes `user/applications/*.md` and
`user/pipeline.md`. Idempotent. The old `applications.md` is left untouched.

**Step 4 — Verify.** A byte-containment check asserts every byte of every Notes cell appears in
exactly one record, and every non-row line appears in `pipeline.md`. Checked against the backup, not
by eye.

Run for real, verify also caught the `submitted` heuristic getting **one record wrong** — it infers
`submitted` from a substring match on `source`, and one record's own body text stated plainly it had
in fact been submitted, contradicting the inferred value. Caught by reading that record's text, not
by the heuristic itself, and corrected by hand before the index was built. Two catches from one
migration run — a missing row and a wrong inferred field — is exactly the class of error this
design exists to make visible instead of silent.

**Known limits, stated plainly rather than engineered around:**

- The status-less population (rows with no `Status` column, migrated into `pipeline.md`) has no
  per-row target file, so `verify` checks it by whole-file blob containment rather than per-record
  absence. A dropped status-less row could in principle hide behind another line's text already
  present in `pipeline.md`. The record-bound population does not have this gap — a missing record
  file is caught by absence before any text comparison runs.
- `verify --backup` resolves a date but not a same-day numbered suffix. If `migrate` is re-run more
  than once on the same day, later runs get a numbered backup directory (`-2`, `-3`, ...), and
  `--backup` without an explicit suffix only checks the first one. An operator re-running migration
  twice in one day has to know which backup is authoritative; the tool does not resolve it for them.

**Step 5 — Replace.** Only after step 4 passes, `applications.md` is replaced by the generated
index.

Steps 3–5 are separate commits for the tracked files they touch. The `user/` changes are protected
by the step-1 backup, not by git.

## Command and rule changes

| File | Change |
|---|---|
| `prompts/commands/job.md:200` | Append a row → write/update the record file, then regenerate the index |
| `prompts/commands/job.md:264` | Flat-file fallback description — files are now the primary path, not a fallback |
| `prompts/rules/ats-portals.md:87-88` | Post-submission logging; also where `submitted: true` is set |
| `prompts/commands/job-eval.md:36` | Recovers a posting URL from `applications.md` → reads the record |
| `prompts/commands/onboard.md:35` | Templates a starter `applications.md` → templates an empty `user/applications/` and `pipeline.md` |
| `prompts/rules/use-deployed-server.md` | **Rewritten, not edited.** It frames files as a fallback for a downed server; under the new boundary the server is not in the record path at all |

## The public mirror

This repo is republished to a public OSS repo by `.github/workflows/mirror.yaml` on every push to
`main`. Two consequences for this design:

**New record files are safe by construction.** The mirror `rsync`s the tree with
`--exclude='user'`, and the scrub gate independently fails the job if a `user/` directory is present
in the transformed tree. `user/applications/`, `user/pipeline.md` and `user/_backup-*/` inherit that
protection with no new configuration. `/user/` in `.gitignore` covers the same ground for commits.

**`docs/` is published, so this file is public.** That is why the frontmatter example above uses
`Acme` and a zero UUID rather than a real application. The scrub gate matches patterns — hostnames,
contact details, keys, module paths — and **cannot** detect an employer name or a pipeline stage. A
design doc that quoted the live pipeline would pass the gate and still leak. Anything written into
`docs/` from here on has to be checked by judgment, not by the gate.

Both points apply to the implementation as well: worked examples in `scripts/*_cases.py` fixtures
use placeholder companies, never rows copied out of `user/`.

## Non-goals

- **Deleting the `/api/applications` endpoints.** They stay and keep working. Commands stop writing
  to them. Removal is a follow-up once the file path has run for a while — ripping out endpoints in
  the same change as a migration leaves a bad migration nowhere to fall back to.
- **Building the missing `application_events` endpoint.** It stays unreachable. The `events` list in
  frontmatter serves the same purpose without a server.
- ~~**Touching the personal-kb.** Out of scope by decision.~~ **Resolved 2026-08-26, immediately
  after this shipped — see "The personal-kb, retired" below.** It was out of scope while this design
  was executed, and the disagreement flagged here (a rule requiring re-ingest of everything under
  `user/`, including the new `user/applications/`) is what forced the question.
- **Anything in `~/projects/interview-prep`.**
- **Curating stale content** during migration.
- **Changing `user/`'s git status.**

## Testing

Following the existing `scripts/*_cases.py` convention — standalone, run directly, no pytest
(`scripts/triage_cases.py`).

`scripts/application_index_cases.py`:

- Malformed frontmatter is caught, not skipped.
- An unknown `status` value is rejected.
- A record whose `packet` path does not exist is flagged.
- A known set of records round-trips to expected index output.

The byte-containment verifier from migration step 4 runs once during the migration and then stays as
a regression guard.

## Open questions

- **Should `user/` be version-controlled?** Deferred, not resolved. It is presumably ignored because
  this repo is shareable and `user/` is real career data. Tracking it would give history to a record
  this design makes authoritative. A separate private repo or submodule is the likely shape. Needs
  its own design pass.
- ~~**The personal-kb's fate.** Deferred by decision.~~ **Resolved — retired. See below.**
- **Whether the JobHub dashboard is worth reviving.** Andre does not use it. This design makes it a
  funnel view rather than a record view, which is a smaller and more defensible surface — but nobody
  has decided whether it should exist at all.

## The personal-kb, retired

**Decided 2026-08-26, the same day this design shipped.** The knowledge base is gone from this repo:
the optional branch in `prompts/commands/job.md`, `prompts/rules/keep-kb-in-sync.md`, and
`scripts/kb_sync.py` are all deleted, and `PERSONAL_KB_URL` is unset. The KB server itself is
untouched and still running; job-search simply stops writing to it. Reversible.

**Why.** Its only consumer was `job.md`, and its only job was substituting for reading
`master-resume.md` and `personal-projects.md` during bullet selection — roughly 13K tokens, on a path
the command's own text described as *"a token optimization, off by default, [that] produces the same
resume."* Against that: it was a second, lossy copy of files that are themselves the source of truth,
and every one of its documented failure modes is silent. A stale index returns confident,
well-formed, wrong answers with nothing to signal it.

**What made it urgent rather than theoretical.** Two things, both observed:

- On 2026-08-26 `kb_sync.py` reported a successful upsert with unchanged chunk counts **twice**, and
  four `kb_search` probes confirmed the indexed copy was two edits stale. The script's success output
  and reality disagreed, silently.
- This migration shrank `user/applications.md` from ~43K to an 8K generated index, leaving the KB
  holding **37 chunks** of application prose from a file that no longer contains it. Re-syncing cannot
  fix that: the upsert is per-chunk, so a shrunk file overwrites the first chunks and leaves the rest
  as searchable ghosts. Removing them would require a full collection rebuild.

**The general rule this is an instance of:** the same one this whole design is built on. Two copies of
one thing, kept in agreement by hand, drift — and the copy that is only a performance optimization is
never worth the drift. If a KB is ever reintroduced, index only files with no lifecycle
(`master-resume.md`, `personal-projects.md`) and never anything with a status that changes.
