#!/usr/bin/env bash
# Wire JobHub's harness-neutral prompts into a specific coding agent.
#
#   scripts/install-harness.sh claude|codex|cursor|gemini|opencode|all
#
# prompts/commands/*.md is the source of truth. This script only creates the
# per-harness plumbing each tool expects. Re-run it after adding a command.
#
# Nothing here is required: every command is a plain Markdown prompt, so you can
# always just paste prompts/commands/job.md into a chat and it works.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SRC="$REPO_ROOT/prompts/commands"
cd "$REPO_ROOT"

if [ ! -d "$SRC" ]; then
  echo "error: $SRC not found — run this from inside the JobHub repo." >&2
  exit 1
fi

commands() { find "$SRC" -maxdepth 1 -name '*.md' -exec basename {} .md \; | sort; }

# Strip YAML frontmatter (harnesses that don't parse it choke on the --- fences).
strip_frontmatter() {
  awk 'NR==1 && $0=="---" {fm=1; next} fm && $0=="---" {fm=0; next} !fm {print}' "$1"
}

# Pull `description:` out of frontmatter for harnesses that want it separately.
description_of() {
  awk -F': *' '/^description:/ {sub(/^description: */,""); print; exit}' "$1"
}

link_dir() { # link_dir <target-dir-relative-to-repo> <dest>
  local target="$1" dest="$2"
  mkdir -p "$(dirname "$dest")"
  if [ -e "$dest" ] && [ ! -L "$dest" ]; then
    echo "  skip  $dest already exists and is not a symlink — move it aside first"
    return
  fi
  rm -f "$dest"
  ln -s "$target" "$dest"
  echo "  link  $dest -> $target"
}

install_claude() {
  echo "claude:"
  link_dir "../prompts/commands" ".claude/commands"
  link_dir "../prompts/rules"    ".claude/rules"
  echo "  note  CLAUDE.md points at AGENTS.md; nothing else to do"
}

install_cursor() {
  echo "cursor:"
  link_dir "../prompts/commands" ".cursor/commands"
  echo "  note  Cursor also reads AGENTS.md at the repo root"
}

install_opencode() {
  echo "opencode:"
  link_dir "../prompts/commands" ".opencode/command"
  echo "  note  opencode also reads AGENTS.md at the repo root"
}

install_codex() {
  # Codex reads prompts from ~/.codex/prompts (global, not per-repo), so these are
  # copies rather than links into the repo. Re-run after editing a command.
  echo "codex:"
  local dest="${CODEX_HOME:-$HOME/.codex}/prompts"
  mkdir -p "$dest"
  local n=0
  while read -r name; do
    strip_frontmatter "$SRC/$name.md" > "$dest/$name.md"
    n=$((n + 1))
  done < <(commands)
  echo "  copy  $n prompt(s) -> $dest"
  echo "  note  Codex prompts are global. Re-run this after editing prompts/commands/."
  echo "  note  Codex reads AGENTS.md at the repo root automatically."
}

install_gemini() {
  # Gemini CLI uses TOML, one file per command, with the body as `prompt`.
  echo "gemini:"
  local dest=".gemini/commands"
  mkdir -p "$dest"
  local n=0
  while read -r name; do
    {
      printf 'description = "%s"\n' "$(description_of "$SRC/$name.md" | sed 's/"/\\"/g')"
      printf 'prompt = """\n'
      # Escape backslashes for TOML, and translate the $ARGUMENTS placeholder
      # (Claude Code / Cursor / opencode / Codex) into Gemini's {{args}}.
      strip_frontmatter "$SRC/$name.md" | sed -e 's/\\/\\\\/g' -e 's/\$ARGUMENTS/{{args}}/g'
      printf '\n"""\n'
    } > "$dest/$name.toml"
    n=$((n + 1))
  done < <(commands)
  echo "  gen   $n command(s) -> $dest/*.toml"
  echo "  note  Generated from prompts/commands/. Re-run after editing them."
}

target="${1:-}"
case "$target" in
  claude)   install_claude ;;
  codex)    install_codex ;;
  cursor)   install_cursor ;;
  gemini)   install_gemini ;;
  opencode) install_opencode ;;
  all)      install_claude; install_cursor; install_opencode; install_codex; install_gemini ;;
  *)
    echo "usage: scripts/install-harness.sh claude|codex|cursor|gemini|opencode|all"
    echo
    echo "Available commands:"
    commands | sed 's/^/  /'
    exit 1
    ;;
esac

echo
echo "Done. Commands available: $(commands | tr '\n' ' ')"
