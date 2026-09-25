---
description: How to drive Greenhouse and Ashby forms — applies only when the user has asked you to submit an application
---

# Driving ATS Portals

Applies **only when the user has asked you to submit an application.** The default everywhere else
is packets, not submission — see `prompts/commands/job-auto.md`.

## Never

- **Never bypass or complete a CAPTCHA, and never enter an emailed "confirm you're a human"
  code** — not when the user asks, not when the user supplies the code, not when the batch is
  already authorised. Say so plainly and hand the tab back.
- Never enter passwords, government IDs, or payment details.
- Never create an account.
- Never answer voluntary EEO/demographic questions unless the user has said otherwise.
- **Never reach for the devtools MCP when the user's browser cannot complete a form.** Hand it to
  the user instead.

## Work in the user's own browser

- Use the Chrome extension. It cannot reach into a cross-origin iframe or upload a file into one;
  accept that limit.
- When a form cannot be completed in the user's own browser, hand it to the user. That is the
  outcome, not a reason to switch tools.

## When a company board embeds Greenhouse

- **Try `job-boards.greenhouse.io/{slug}/jobs/{id}` first.** Direct Greenhouse works completely.
- Expect `find` to return nothing and `form_input` and `file_upload` to be unavailable inside the
  iframe. The resume is required, so this blocks the submission.
- **Screenshot after the first field** to confirm clicks and typing are landing. Never fill twenty
  fields and then check.
- Offer Greenhouse's **"Enter manually"** resume box. Never assume it — it discards the PDF the
  preflight pipeline produced.

## Custom comboboxes

Greenhouse and Ashby dropdowns are not `<select>` elements.

- Never trust a `form_input` success message. Confirm visually.
- Never trust `invalid="true"` in the accessibility tree. It is stale validation state.
- **To commit a value:** click the toggle, take a fresh snapshot, click the `option` by its uid.
  Typing plus `Enter` does not commit and often clears the field.
- **To verify:** click Submit and read which field the form jumps to. Nothing else is trustworthy.
- Screenshot immediately before any coordinate click on a long form, or use ref/uid clicks.

## Answer from the record

Fill only what `user/config.yaml`, `master-resume.md` and `personal-projects.md` support, plus what
the user has stated in session. **Stop and ask** for anything else — home city, salary expectation,
relocation plans, post-employment restrictions, citizenship for export-control questions, favourite
video game.

Record any narrowing answer in `user/applications/{company}-{role}.md` so a later call is not
blindsided.

## Log it, and only if it was sent

After a confirmed submission: write or update `user/applications/{company}-{role}.md`, set
`submitted: true`, append an `events:` entry dated today, attach the PDF path in `resume:`, and
run `python3 scripts/build_application_index.py`.

**A form that was filled but not submitted keeps `submitted: false`.** That is what the field is
for.
