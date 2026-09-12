# Driving ATS Portals

**Applies only when the user has asked you to submit an application.** The default everywhere else
is still packets, not submission — see `prompts/commands/job-auto.md`.

Everything below was established on 2026-08-23 across twelve Greenhouse and Ashby forms. Nine
submitted; three did not. The three failures were not bad luck, they were structural, and this file
exists so the next attempt does not rediscover them.

## The hard stop, first

**Never bypass or complete a CAPTCHA, and never enter an emailed "confirm you're a human" code.**
This holds when the user explicitly asks, supplies the code, or has already authorised the batch.
Both ClickHouse and MongoDB produced one; the answer was the same each time. Say so plainly and hand
the tab back.

Also standing: never enter passwords, government IDs, or payment details; never create an account;
leave voluntary EEO/demographic questions blank unless the user has said otherwise.

## Use the user's own browser. The devtools MCP is not a workaround.

| | Chrome extension (user's browser) | devtools MCP (own Chrome profile) |
|---|---|---|
| Reaches into a cross-origin iframe | **No** | Yes, via CDP |
| Uploads a file into one | **No** | Yes |
| Human-verification code demanded | **0 of 9 submissions** | **2 of 2 submissions** |

The devtools path looks like the answer to the iframe problem and is not. A fresh, cookie-less
profile appears to trip Greenhouse's bot heuristics — inferred from the 9-vs-2 split, not verified —
and it dead-ends at a code you are not allowed to enter. It also drives a browser the user cannot
see, so "I fill it, you finish it" is not available there.

**Consequence: if a form cannot be completed in the user's own browser, the honest outcome is to
hand it to the user, not to reach for CDP.**

## Company-hosted boards embed Greenhouse cross-origin

`asana.com/jobs/apply/…`, `mongodb.com/careers/…`, `betterment.com/careers/…` all render the real
Greenhouse form inside an iframe from another origin. The accessibility tree stops at the iframe
boundary, so `find` returns nothing, there are no refs, and `form_input` and `file_upload` are both
unavailable. **The resume is a required field, so this blocks the whole submission.**

- **Try `job-boards.greenhouse.io/{slug}/jobs/{id}` first.** Direct Greenhouse works completely.
  Company URLs generally redirect back to themselves, so this is a cheap test, not a fix.
- Coordinate clicking and typing sometimes still reach the iframe (Asana accepted both) and
  sometimes do not (MongoDB and Betterment ignored every click). Verify with a screenshot after the
  first field; do not fill twenty and then discover none landed.
- Greenhouse offers **"Enter manually"** for the resume, which takes pasted plain text. It is a real
  fallback but it discards the PDF the whole preflight pipeline produced — offer it, do not assume it.

## Custom comboboxes lie, in both directions

Greenhouse and Ashby dropdowns are not `<select>` elements. Two independent failure modes:

1. **`form_input` reports success and leaves the field empty.** It returned `Set text value to "US"`
   on Mercury's Country field while the field stayed blank. Always confirm visually.
2. **The accessibility tree reports `invalid="true"` on fields that are correctly set.** The flag is
   stale validation state, not truth. Three separate mechanisms were abandoned as "not working" when
   they had in fact worked.

**What actually commits:** click the toggle → take a fresh snapshot → click the `option` element by
its uid. Typing plus `Enter` does not commit and often clears the field.

**The only trustworthy verification is clicking Submit and reading which field the form jumps to.**
A failed submit is cheap and diagnostic. Use it deliberately rather than trusting any flag.

## Long forms scroll under you

Coordinate clicks go stale between calls on long Greenhouse forms — the page re-renders and shifts.
Screenshot immediately before a coordinate click, or prefer ref/uid clicks where the tree reaches.

## Answer from the record, never from inference

Every field is a factual claim the user has to stand behind in an interview. Fill only what
`user/config.yaml`, `master-resume.md` and `personal-projects.md` support, plus what the user has
stated in the session. **Stop and ask** for anything else. Real examples that required asking rather
than guessing: home city, salary expectation, relocation plans within 90 days, post-employment
restrictions, citizenship for export-control questions, and "what's your favourite video game."

Two answers given on 2026-08-23 are worth knowing were given, because they set up a screen:
Astronomer was told Go is **independent work only, not professional**, and Kubernetes is **service-owner
experience, not cluster-operator**. Both true, both narrowing. Record answers like these in
`user/applications/{company}-{role}.md` so a later call is not blindsided.

## Log it, and only if it was actually sent

After a confirmed submission: write or update `user/applications/{company}-{role}.md`, set
`submitted: true`, append an `events:` entry dated today, attach the PDF path in `resume:`, and run
`python3 scripts/build_application_index.py`. **A form that was filled but not submitted keeps
`submitted: false`** — that is what the field is for.
