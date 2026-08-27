# JobHub API — Request Reference

Payload shapes for `$JOBHUB_URL`. **Look them up here when you need one; do not memorise them.**
The decision-making lives in `prompts/commands/job.md` — this file is only the wire format.

Every request carries `-H "Authorization: Bearer $JOBHUB_API_TOKEN"`. If the token is unset, omit
the header (local dev). `$JOBHUB_URL` defaults to `http://localhost:8080`. Never hardcode a host —
see `prompts/rules/use-deployed-server.md`.

Every POST that succeeds prints a dashboard URL. **Print it for the user.**

---

## Fit reports

```bash
curl -s -X POST $JOBHUB_URL/api/fit-reports \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{
    "company": "", "role_title": "", "location": "", "level": "", "posting_url": "",
    "verdict": "strong|worth|stretch|skip",
    "verdict_summary": "one sentence",
    "why_apply": "<p>para 1</p><p>para 2</p>",
    "match_signals": [{"requirement": "They want: X", "evidence": "how he maps", "source": "resume bullet"}],
    "gap_signals":   [{"requirement": "They want: X", "evidence": "honest assessment"}],
    "flag_signals":  [{"requirement": "conflict", "evidence": "why concerning"}]
  }'
```

Returns `{"id": "...", "url": "/fit-reports/..."}`.

**Signal arrays are create-only.** `PATCH /api/fit-reports/{id}` accepts only the narrative fields —
`role_title`, `location`, `level`, `posting_url`, `verdict`, `verdict_summary`, `why_apply`. There is
**no DELETE route**. Get the signals right the first time; a wrong one cannot be removed.

---

## Research briefs

```bash
curl -s -X POST $JOBHUB_URL/api/research \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{
    "company": "", "stability_verdict": "strong|stable|caution|avoid", "stability_notes": "",
    "stage": "", "headcount": "", "founded": "", "remote_policy": "", "culture_notes": "",
    "tech_stack": [], "salary_range_text": "", "salary_source": "",
    "researched_at": "YYYY-MM-DD", "raw_markdown": ""
  }'
```

Upserts by company name — re-running research updates the existing brief, and the server links it
into the fit report sidebar automatically.

**`GET /api/research` returns 405 — it is POST-only.** To check whether a brief exists, look for
`user/research/{company}.md` on disk. Do not try to grep the endpoint.

---

## Applications

**The write examples below are superseded.** Applications and resumes are no longer POSTed or
PATCHed — the write path is the record file, `user/applications/{company}-{role}.md`, regenerated
into the index with `python3 scripts/build_application_index.py`. See
`docs/state-consolidation-design.md` for the record format. The endpoints still exist and still
work — keeping them is an explicit non-goal of that design — so the two read examples below remain
current; use them to inspect what the server holds.

```bash
# SUPERSEDED — do not use. create
curl -s -X POST $JOBHUB_URL/api/applications \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{"company": "", "role_title": "", "source": "", "status": "applied",
       "applied_at": "YYYY-MM-DD", "resume_file": "", "notes": ""}'

# SUPERSEDED — do not use. attach the PDF — this used to run immediately after create
curl -s -X POST $JOBHUB_URL/api/applications/{id}/resume \
  -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -F "resume=@user/tailored/{company}/{role}/resume.pdf"

# find an id — still current, e.g. to inspect what the server holds
curl -s "$JOBHUB_URL/api/applications?company=acme" -H "Authorization: Bearer $JOBHUB_API_TOKEN"

# read one back — still current
curl -s "$JOBHUB_URL/api/applications/{id}" -H "Authorization: Bearer $JOBHUB_API_TOKEN"

# SUPERSEDED — do not use. status change
curl -s -X PATCH $JOBHUB_URL/api/applications/{id} \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{"status": "", "notes": "optional"}'
```

- `company` matches case-insensitively on a substring. `status` matches exactly and accepts only
  `applied`, `phone_screen`, `onsite`, `offer`, `rejected`, `ghosted`, `withdrawn`. Both optional.
- **On a status change, `notes` is recorded on that transition's `application_events` row — the
  audit trail — and does not overwrite the application's `notes` column.** A subsequent `GET`
  showing `"notes": null` is correct, not a failed write. (History — the PATCH that produced this is
  itself superseded above.)
- **Do not try to read an id off the dashboard.** `/applications` is an HTML route behind SSO; the
  bearer token answers a 302 to the login page. The API is the only path an agent has.

---

## Eval

```bash
curl -s -X POST $JOBHUB_URL/api/eval \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{
    "resume_file": "{company}_{role}.md", "company": "", "role_title": "", "posting_url": "",
    "eval_mode": "gate",
    "posting_terms": [{"term": "", "category": "explicit_skill|implicit_capability|domain_keyword|seniority_signal", "priority": "top3|standard"}],
    "resume_text": "", "skills_list": [], "project_gaps": {"term": ["project"]},
    "style_input": {"summary_text": "", "title_line": "", "expected_level": "staff|senior",
                    "has_education": true, "has_prior_roles": true, "full_resume_text": ""}
  }'
```

Dashboard: `$JOBHUB_URL/eval-results/{id}`.

**Read `prompts/rules/job-eval-gate.md` before acting on the output.** The engine matches literal
substrings; a `missing` term is a statement about the string, not about Andre.

---

## Search results

```bash
curl -s -X POST $JOBHUB_URL/api/search-results \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{
    "ran_at": "ISO-8601", "board_count": 0, "raw_count": 0, "location_filter": "",
    "results": [{
      "company": "", "role_title": "", "location": "", "is_remote": true,
      "salary_min": null, "salary_max": null, "salary_disclosed": false, "below_floor": false,
      "posting_url": "", "fit_tier": "strong|good", "tags": [],
      "level_tag": "senior", "domain_tag": "platform|identity|devplatform|observability|fullstack|infra"
    }]
  }'
```

`below_floor` is vestigial — **there is no comp floor** (`preferences.md`). Send `false`.

Returns `{"batch_id": "...", "url": "/search-results/..."}`.

---

## Boards

```bash
curl -s $JOBHUB_URL/api/boards        # -> {"tracked": [...], "discovery": [...]}

curl -s -X POST $JOBHUB_URL/api/boards \
  -H "Content-Type: application/json" -H "Authorization: Bearer $JOBHUB_API_TOKEN" \
  -d '{"slug": "", "name": "", "ats": "greenhouse", "tags": "comma,separated", "status": "tracked"}'
```

Probe a slug before adding it: `https://boards-api.greenhouse.io/v1/boards/{slug}` — valid JSON
means the slug is good.

---

## Greenhouse public board API (not JobHub)

```
https://boards-api.greenhouse.io/v1/boards/{slug}/jobs?content=true    # all jobs
https://boards-api.greenhouse.io/v1/boards/{slug}/jobs/{id}?content=true
https://boards-api.greenhouse.io/v1/boards/{slug}/jobs/{id}?questions=true
```

Each job carries `title`, `location.name`, `content` (HTML), `absolute_url`, and sometimes
`pay_input_ranges` (`{min_cents, max_cents, currency, title}`). If that is absent, the salary is
usually in the `content` HTML.

**The `offices` array is not a statement of remote eligibility.** It is per-job and can disagree
with `location.name`. Read `location.name` and the posting body; do not infer remote from `offices`.
Getting this wrong told Andre a Boston/NY-only req was NJ-remote on 2026-08-21.
