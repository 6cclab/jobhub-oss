# /interview-prep

Write the prep sheet for a scheduled interview round.

**Usage:** `/interview-prep {company}` or `/interview-prep {company} {round}`.
With no argument, prep the application at screen stage that has no sheet —
`python3 scripts/build_application_index.py` names it.

**Read `prompts/rules/interview-prep-sheet.md` before starting.** It is the standing instruction
this command exists to satisfy.

## Where it goes

`user/tailored/{company}/{role}/recruiter-call.md`. A file, never a chat reply. Questions written
into a reply are gone by the time they are dialling.

If deeper prep already lives in `{interview_prep_path}/local/companies/{company}.md`, read it,
build on it, and declare it in the application record so the check stops warning:

```yaml
prep: ~/projects/interview-prep/local/companies/{company}.md
```

## Read first

| Source | For |
|---|---|
| `user/applications/{company}-{role}.md` | status, history, what has already been answered, prior round debriefs |
| `user/tailored/{company}/{role}/resume.md` | **the resume actually submitted** — including its career-years figure |
| `user/tailored/{company}/{role}/posting.md` | what the role says it wants |
| `user/research/{company}.md` | if a brief exists; run `/company-research` if not |
| `{interview_prep_path}/local/drill-log.md` | current drill state, before giving any prep advice |
| `user/offer-timeline.md` | scheduling conflicts and where this sits in the pipeline |

Re-read an existing sheet before each round rather than assuming it is still right. A sheet
written for a recruiter screen is the wrong sheet for a system design round.

## Sections, in this order

1. **Settled before you dial — do not spend the call on these.** Comp, office policy, anything
   already answered on the application. Include the career-years figure on the submitted resume,
   so a newer number is not introduced mid-call by accident.
2. **Ask these.** Numbered. Each one carries the reason it matters *to them* and the actual
   sentence to say. Not a topic list.
3. **Lead with this.** The two or three pieces of the record that map most directly onto the
   product. Cite the master-resume lines.
4. **Be straight about this, unprompted.** The genuine gaps, phrased the way they would say them.
   Go is agent-written. Vector search, ML depth and healthcare domain are gaps where they are
   gaps — check `user/master-resume.md` Constraints before calling anything one.
5. **Worth knowing before you dial.** Company facts, org health, interviewer background, funding,
   recent shipping. Sourced, or labelled as unverified.

Date it. Name the interviewer if known.

## Rules

- **Answer from the record.** Every claim on this sheet is one they have to stand behind live. If
  `user/master-resume.md`, `user/personal-projects.md` or `user/config.yaml` does not support it,
  ask rather than infer.
- **Never name another company.** `prompts/rules/application-records-are-isolated.md` applies to
  everything under the packet directory. Scheduling conflicts and pipeline comparisons go in
  `user/offer-timeline.md`; link to it.
- **Aggregator question lists are filler.** jointaro and interviewquery are templated and are
  never corroboration for what a company actually asks.
- **One em-dash per document at most.** Use periods.

## After writing

Set `status:` and append an `events:` entry on the application record if the round moved it, then:

```bash
python3 scripts/build_application_index.py
```

The check warns on any application at screen stage with no sheet. It should stop warning for this
one.
