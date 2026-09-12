# JobHub

**Project instructions live in [`AGENTS.md`](../AGENTS.md) at the repo root. Read that file.**

It is harness-neutral so the same instructions work in Claude Code, Codex, Cursor, Gemini CLI,
opencode, and anything else. This file exists only so Claude Code finds its way there.

## Claude Code specifics

- `.claude/commands` and `.claude/rules` are **symlinks** into `prompts/commands` and
  `prompts/rules`. Slash commands work normally. **Edit `prompts/` directly** — editing
  "through" the symlink works but is confusing in diffs, and other harnesses read `prompts/`.
- Machine-local settings belong in `.claude/settings.local.json`, which is gitignored.
  `.claude/settings.json` is shared — keep personal hosts and tokens out of it.
