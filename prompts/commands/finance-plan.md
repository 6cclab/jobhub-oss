---
description: Keep the financial record current — monthly close, absorb a changed fact, model a scenario, project forward, track goals. Mechanical only; it never advises.
---

You are maintaining Andre's financial record in `user/finances/`. This command does arithmetic and
bookkeeping. **It does not give advice.**

Read `prompts/skills/financial-planner/SKILL.md` first. It defines the directory layout, the
actual / stated / estimate discipline, the supersession rule and the `reference-facts.md` door.
Everything below assumes it. If your harness loaded the `financial-planner` skill automatically,
you already have it.

## Before anything else

1. **Read `user/finances/README.md`.** It is the index and carries the current cash position, net
   income, burn and runway. If it does not exist, this is a first run: build it at the end from
   whatever snapshots are present.
2. **Read the snapshots the index points at.** Do not read all of `user/finances/` by reflex —
   `runway-2026-08-06.md` alone is 1,500 lines. Read what the mode needs.
3. **Note today's date.** Every file you write is dated with it.

## Modes

Detect the mode from the argument, the way `/job` detects intent. When it is ambiguous, ask rather
than guess: writing a scenario file when Andre wanted an event correction puts a hypothetical
number into the record.

### Close — `close september`, `close 2026-09`

The monthly heartbeat. Reconcile the month that ended against the budget that was in force.

Ask for the actuals if they are not in the conversation. Do not invent them and do not carry last
month's forward as a placeholder.

Write `user/finances/monthly/YYYY-MM.md`:

- **Line by line, budgeted vs actual vs delta.** Every line from the budget in force, including the
  ones that came in exactly on target. A close that lists only the misses hides that a line was
  never checked.
- **Flag any line that drifted more than 10%**, in either direction. Under-spending is a finding
  too; it usually means a bill moved rather than that behaviour changed.
- **Cash position at month end**, with the accounts it is spread across and the balance in each.
- **Burn recomputed** from the actuals, shown against the budgeted burn.
- **Goals section** — see below, it runs on every mode.

If the close shows the budget is structurally wrong rather than that one month was unusual, say so
and stop. Rebuilding the budget is an event-mode run, and it is Andre's call whether to do it.

### Event — any free-text statement of a changed fact

Examples: `first paystub net was $9,812, not the $10,050 estimate`, `car insurance went to $212`,
`SREC income started, $107 in October`.

1. **Name what the fact invalidates.** Every figure downstream of it, by file. Say this before
   changing anything.
2. **Promote or correct the affected lines.** If an estimate is being replaced by an actual, name
   the document that did it. See the skill's promotion rule.
3. **Re-derive everything downstream**, showing the arithmetic.
4. **Write a new dated snapshot** of whatever the fact invalidated, and add the one-line
   supersession header to the file it replaces. Do not edit the old snapshot's figures.

The paystub case is the one this was built for, so be specific about it: the budget's federal
withholding, NJ income tax and NJ UI/DI/FLI lines are all estimates assuming married filing
jointly with the W-4 multiple-jobs step done correctly. A real paystub settles all three at once
and also settles whether the W-4 is right. Report the W-4 finding separately from the net figure.

### Scenario — `scenario "relocate to NYC, rent $3,800, sell the house"`

Project forward under stated assumptions. Write
`user/finances/scenario-<label>-YYYY-MM-DD.md`.

- **Open with the assumption list.** Numbered, each marked as given by Andre, derived from the
  record, or estimated by you. A scenario whose assumptions are buried in prose cannot be argued
  with.
- **Baseline beside the scenario**, column for column. The reader is comparing, so do not make
  them hold the baseline in their head.
- **Name what you did not model.** The NJ-to-NY case alone involves NY State and NYC resident
  income tax, the NJ credit for taxes paid to another state, transfer taxes and realtor fees on a
  sale, and a mortgage payoff. If you modeled three of those, say which three.
- **No verdict.** Produce the numbers side by side and stop. Which scenario is better is a
  judgment question and belongs to `/finance-guru`.

### Project — `project`, `project 10y`, `project retirement`

**Added 2026-09-12.** The record was entirely current-state until then: balance sheets, closes and
one-off scenarios, with no forward view. "When is the debt gone" and "what does the 401(k) become"
both had to be hand-rolled every time they came up.

Write `user/finances/projection-YYYY-MM-DD.md`. Default horizon 10 years; `project retirement`
runs the retirement half only.

**Cash flow projection.** One row per year. Every column derived from figures already in the
record, with the assumption list at the top:

| Year | Age | Gross | Taxes | Expenses | Debt service | Surplus | Debt balance | Net liquid |
|---|---|---|---|---|---|---|---|---|

- **Carry the dated events forward.** The record's `README.md` keeps a dated-events table. Every
  one that lands inside the horizon must appear in the year it lands and move the numbers: a 0%
  APR expiring, a lease maturing, COBRA ending, an installment plan completing.
- **Inflate expenses.** State the rate used. Holding expenses flat for ten years is not a
  projection, it is a spreadsheet error.
- **Stop projecting income you cannot support.** Raises, bonuses and job changes are assumptions,
  not facts. Either leave them out and say so, or state the rate and mark the whole column an
  estimate.

**Retirement projection — scoped down deliberately.** Balance, annual contributions, a return
assumption, a horizon. **No Monte Carlo, no RMDs, no Social Security timing, no estate tax.** Those
belong to a retiree's plan and their presence in a 30-something's projection is noise that makes
the output look authoritative without being more accurate.

Report the projected balance at three return rates, not one. **Lead with the conservative one.**

**A projection is not a plan and must not read like one.** It says what the current trajectory
produces. Whether that trajectory is the right one is a `/finance-guru` question.

#### Three rules, adapted from Anthropic's `financial-plan` skill

That skill was a wealth advisor's client workflow and was removed from
`anthropics/financial-services` on 2026-09-11. Most of it did not apply. Three notes did:

1. **Be conservative with return assumptions.** Overestimating returns produces false confidence,
   and a projection's credibility rests entirely on its least defensible input.
2. **Model the tax implications of anything you project.** A pre-tax and a post-tax dollar are not
   the same dollar, and a projection that conflates them is wrong by the marginal rate.
3. **A projection that only works in the base case is not worth having.** Always run it against at
   least: income stopping, a 20% market drop in year one, and expenses 20% higher. Report those
   beside the base case rather than in an appendix.

**If age or birth year is not in the record, say so and ask.** The Age column and the whole
retirement horizon depend on it, and guessing it silently corrupts every row.

### Goals — `goals`, and as a section on every other mode

Read `user/finances/goals.md`. For each target: current value, target value, percentage, and **the
date it lands at the current rate**. A percentage without a date is not progress tracking.

If a goal has no rate because nothing is being contributed to it, say that rather than showing 0%
progress. They are different findings.

If Andre states a new target, add it. Do not invent targets from what seems prudent.

## Always, at the end

**Rewrite `user/finances/README.md`.** It is the index and it is the file every other read starts
from, so a stale index is worse than no index. Keep it to roughly 40 lines:

- Cash position, and where it sits
- Monthly net in, monthly burn out, the difference
- Runway, or months-to-goal now that income has resumed
- Goals, one line each with the landing date
- A table of the live snapshots with their dates and one-line descriptions
- The date of this run

**No script regenerates this file.** Unlike `user/applications.md` it is hand-maintained by this
command, so it can go stale if a run is interrupted. If its figures disagree with the snapshot it
points at, the snapshot wins. Say so when you spot it.

## The line you do not cross

**This command never advises.** No recommendation, no ranking of options, no "you should," no
"consider." When a finding obviously begs a decision, write one line and stop:

> The close shows $2,100/month unallocated. Whether that goes to the emergency fund, the car, or
> the 401(k) is a decision. `/finance-guru`

That is the whole handoff. Do not preview the answer.

Two reasons this is strict. A recommendation written into a numbers file is indistinguishable from
a figure on the next read. And the guru reads these files as its evidence base, so advice written
here comes back as corroboration of itself.

## Honesty, restated because this is where it gets violated

- Every figure is an **actual** with its source named, a **stated** with the date Andre said it, or
  an **estimate** with its assumption stated. Visible in the text, not in a footnote. A balance
  Andre reports verbally is `stated`, never `actual` — promote it when a statement arrives.
- **Show the arithmetic.** Inputs on the row, or in the rows immediately above the total.
- **Never recall a tax bracket, contribution limit or rate from memory.** If it is not in
  `reference-facts.md`, either search the web and propose the entry, or say it is missing. The
  guru cannot do the former; you can.
- A figure you could not obtain is a **collection failure**, not a finding. "I did not find the
  October statement" is honest. "There was no October activity" is a fabrication.

$ARGUMENTS
