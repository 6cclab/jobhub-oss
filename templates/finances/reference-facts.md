# Reference facts

**External constants that did not come from your own statements.** Contribution limits, tax
brackets, state rules, prevailing rates.

This file exists because `/finance-guru` has no web access and may not recall these from memory.
A model's weights carry a stale tax year with complete confidence and nothing in the output marks
it. So outside facts enter the record here, deliberately, with a source and a date, or they do not
enter at all.

## The rules

- **Every entry carries a source and the date it was entered.** A figure with neither is not a
  reference fact, it is a guess someone typed.
- **Cite the entered date when you use one.** "the $24,500 limit, entered 2026-09-12" is checkable.
- **Year-scoped facts go stale.** An entry for a prior tax year gets flagged, not used.
- `/finance-plan` may search the web and propose an entry. `/finance-guru` may not. Only add what
  you have actually looked up.

## Format

```markdown
## <the fact, named so it can be found>
**<value>** (<scope, e.g. 2026 tax year>) — source: <where it came from>, entered YYYY-MM-DD
<optional: a line on anything non-obvious about how it applies>
```

---

## 401(k) elective deferral limit
**$0** (YYYY tax year) — source: , entered YYYY-MM-DD

## IRA contribution limit
**$0** (YYYY tax year) — source: , entered YYYY-MM-DD

## Standard deduction, married filing jointly
**$0** (YYYY tax year) — source: , entered YYYY-MM-DD

## State treatment of 401(k) contributions
 — source: , entered YYYY-MM-DD
Some states tax 401(k) contributions where the federal treatment defers them. Worth an entry if
yours does, because it changes the real cost of a contribution.

## Savings account APY
**0.00%** — source: , entered YYYY-MM-DD
