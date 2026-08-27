---
description: Review application text for tone violations — old-org shots, project pitching, self-deprecation, banned phrases, em-dash overuse
---

Review the provided text (or the most recently drafted application answer in this conversation) for tone violations.

Delegate this to a fast, cheap model if your harness supports subagents; otherwise run it inline in this conversation. The check is what matters, not the mechanism. See **Delegation** in `AGENTS.md`.

Use this prompt:

```
You are a tone reviewer for job application text. Check for these violations:

1. OLD-ORG SHOTS: Any phrase that implies the current/previous employer is worse than the target company. Patterns: "not a cost center," "not a footnote," "not an afterthought," "not bolted on," "somewhere that actually X," "a team that's already bought in," any "not [negative thing]" that implies the old org was that negative thing. Test: would the writer's current manager be uncomfortable reading this?

2. PROJECT PITCHING: Does the answer read like a showcase of what the writer built rather than an answer to the question asked? The question should drive the answer; projects/experience are supporting evidence, not the headline.

3. SELF-DEPRECATION: Does any personal anecdote frame the writer as having a shortcoming? Aim frustration at tools/systems being bad, not at the writer.

4. BANNED PHRASES: "passionate about," "leveraging," "excited to," "robust solutions," "synergy," "track record of," "proven ability to," "demonstrated experience in," "results-driven," "self-motivated," "detail-oriented"

5. EXCESSIVE EM-DASHES: More than 1 em-dash in the text.

For each violation found, quote the offending text and explain why it fails. Be strict — if it's borderline, flag it.

Return JSON: {"pass": true|false, "violations": [{"type": "old_org_shot|project_pitch|self_deprecation|banned_phrase|em_dash", "quote": "...", "reason": "..."}]}
```

**TEXT TO REVIEW:**
```
{the text provided as $ARGUMENTS, or the most recent application answer in conversation}
```

Present the result clearly:
- If PASS: "Clean — no violations found."
- If FAIL: List each violation with the quoted text and a suggested fix. Then present a corrected version.

$ARGUMENTS
