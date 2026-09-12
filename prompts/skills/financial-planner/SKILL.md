---
name: financial-planner
description: Use when the question touches Andre's own money — runway, burn rate, budget, cash position, savings, debt payoff, 401(k) or retirement contributions, taxes and withholding, emergency fund, a monthly close, a scenario like relocating or changing jobs, or whether he can afford something. Establishes where the financial record lives, the actual / stated / estimate discipline every figure must carry, and the routing rule between the mechanical planner and the advisory guru. Read this before reading or writing anything under user/finances/.
user-invocable: false
---

# The Financial Record

Andre's finances live in `user/finances/`. The directory is **gitignored** and is the only copy of
this material. Nothing in it is derived or rebuildable, unlike `user/jobhub.db`. Treat every file
as authoritative and never overwrite one without leaving the supersession trail described below.

## Two entry points, and the line between them

| | `/finance-plan` | `/finance-guru` |
|---|---|---|
| Job | Keeps the numbers current | Answers a judgment question |
| Writes | Dated snapshots, `README.md`, `goals.md` | Prose only, or nothing |
| Advises | **Never** | That is the whole job |
| Web access | Yes, if a figure needs sourcing | **Never** |

**Route on whether the answer changes a number or expresses an opinion.**

- "My first paystub netted $9,812" → a number changed → planner.
- "Should I pay the car down or rebuild savings?" → an opinion → guru.
- "What's my runway?" → reading an existing number → answer from `user/finances/README.md`
  directly, no command needed.
- "What's my runway if I quit in March?" → a new number under assumptions → planner, scenario mode.

**The planner does not advise and the guru does not edit numbers.** This is not tidiness. An
opinion that gets written into a numbers file becomes a figure that a later read treats as fact,
and there is nothing in a markdown table that distinguishes the two. Keeping the roles apart is
what prevents that.

## The layout

| File | What it is |
|---|---|
| `README.md` | The index. Current cash, net, burn, runway, goals at a glance, pointers to the live snapshots. Rewritten by the planner on every run |
| `reference-facts.md` | External constants, hand-entered, each with a source and the date it was entered. The only legitimate door for a fact that did not come from Andre's own statements |
| `goals.md` | Targets and progress against them |
| `monthly/YYYY-MM.md` | One close per month: budgeted vs actual, drift, cash position |
| `runway-YYYY-MM-DD.md` | Dated runway snapshots |
| `budget-<label>-YYYY-MM-DD.md` | Dated budget snapshots |
| `scenario-<label>-YYYY-MM-DD.md` | Projections under stated assumptions |
| `projection-YYYY-MM-DD.md` | The forward view: year-by-year cash flow, debt payoff date, scoped retirement projection, stress cases |
| `advice-YYYY-MM-DD-<topic>.md` | Guru output, prose only, no tables of derived figures |
| `solar-srec-registration.md` | A live workstream, not a snapshot. Undated on purpose |

**`user/finances/budget-180k-2026-09-08.md` is the reference for what good looks like.** Its income
table shows every input on the row that produces the output, and it marks its three estimated tax
lines as estimates in the prose rather than in a footnote. Follow it.

## Every figure is an actual, a stated, or an estimate

Three categories, and which one a figure is must be **visible in the text** — not implied by
context, not deferred to a footnote at the bottom.

**Actual** — traces to a document Andre has. Name the document and its date: "Chase 8107,
statement of 2026-08-06", "first Talkspace paystub, 2026-10-10". A figure whose source cannot be
named is not an actual.

**Stated** — Andre told you, and no document confirms it yet. "Home Depot is zero." He is the
account holder, so this is much stronger than a guess, and it is the fastest way to close an
unknown. It is still not a statement: a pending charge, a payment in flight, or an honest
misremembering can move it. **Label it `stated` with the date he said it.**

**Estimate** — everything else, including anything computed from a tax table or an assumption
about future behaviour. State the assumption on the same line. The existing budget's
`Federal withholding *(est., one child)*` is the pattern.

**Promotion runs one way: estimate → stated → actual, and each step names what did it.** "Verified
against the October paystub" is a promotion to actual. "Andre confirmed 2026-09-12" is a promotion
to stated. "Updated" is neither.

**Never silently upgrade a `stated` to an `actual`.** The distinction exists so that a later read
knows which figures could still move without anyone having made a mistake. Added 2026-09-12, after
two card balances came in by word of mouth and there was nowhere honest to put them.

## Show the arithmetic

Prompt-only math has no test behind it. A wrong subtraction in a runway model is silent and can
survive for months, which is exactly what a table of bare results invites.

So: **a derived figure carries its inputs on the same row or in the rows immediately above it.**

Acceptable, because every input is visible and the total can be checked by eye:

```
| Gross                                  | $15,000.00 |
| Health (PPO 5000 family, pre-tax)      |   −$629.80 |
| 401(k) at 4%                           |   −$600.00 |
| FICA @ 7.65% on gross less health      | −$1,099.32 |
| **Net, estimated**                     | **≈ $10,050** |
```

Not acceptable: `Net ≈ $10,050` with the derivation in prose somewhere else, or a runway figure
with no burn and no starting balance beside it.

State percentages with the base they apply to. "7.65% on gross less health" is checkable;
"7.65% FICA" is not.

## Supersession is explicit, and goes in both files

`runway-2026-08-06.md` carries five stacked revision headers — `Revised 2026-08-07 (a)`, `(b)`,
`(c)` — because each correction was appended to the live file instead of superseding it. The result
is a document where the headline and the body disagree and the reader has to reconstruct the order.

When a snapshot replaces another:

1. The **new** file opens by naming the file it replaces and what changed.
2. The **old** file gets a single line at the top: `> Superseded by <filename>, YYYY-MM-DD.`
   Nothing else about it is edited. It stays as the record of what was believed then.

Never edit a dated snapshot's figures in place. The date in the filename is a claim about when
those numbers were true.

## `reference-facts.md` is the only door for outside facts

Contribution limits, tax brackets, state rules, prevailing savings rates: none of these come from
Andre's statements, and none of them may be recalled from training data into a financial model.
Model weights carry a stale tax year with total confidence and nothing in the output marks it.

Each entry carries the fact, its source, and the date it was entered:

```markdown
## 401(k) elective deferral limit
**$<figure>** (<tax year>) — source: <IRS notice number>, entered <YYYY-MM-DD>
```

**The placeholders above are deliberate.** An earlier version of this file used a plausible-looking
figure and notice number as the example, which puts a fabricated fact, formatted exactly like a
real record entry, in a file every agent reads. `financial-guru` declined it correctly on the first
test, but it had to *reason* that an illustration was not a record. Do not make it reason about
that. Examples in this file use angle-bracket placeholders; real facts live only in
`user/finances/reference-facts.md`.

- Cite the entered date whenever you use one.
- If a year-scoped fact was entered for a prior tax year, flag it as stale rather than using it.
- If the fact is absent, **say so and name the exact entry to add.** Do not estimate it.

The planner may search the web to source a fact and then propose the entry. The guru may not.

## Voice

- At most one em-dash in anything drafted for Andre. Use periods.
- Answer the question that was asked. Do not pitch adjacent work.
- No "you should have." The record is what it is.
- Dollar figures to the cent where the source has cents, rounded with a `≈` where they are derived
  from estimates. Do not imply precision the inputs do not have.
