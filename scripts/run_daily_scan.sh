#!/usr/bin/env bash
# Daily automated job scan: scan -> triage -> appeal -> post.
#
# Designed to run unattended from launchd. Every stage failure is written to the
# run log AND surfaced in the notification, because a scan that silently stops
# working looks exactly like a quiet job market.
#
# Install:  see scripts/launchd/README.md
# Manual:   ./scripts/run_daily_scan.sh
#           ./scripts/run_daily_scan.sh --dry-run
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO" || exit 1

DRY_RUN=""
[[ "${1:-}" == "--dry-run" ]] && DRY_RUN="--dry-run"

STAMP="$(date -u +%Y-%m-%d)"
WORK="$(mktemp -d "${TMPDIR:-/tmp}/jobhub-scan.XXXXXX")"
LOGDIR="$REPO/user/search-results/logs"
LOG="$LOGDIR/$STAMP.log"
mkdir -p "$LOGDIR"

# launchd gives a minimal PATH; Homebrew python3 and the claude CLI both live
# outside it. Without this the cron silently no-ops on "command not found".
export PATH="/opt/homebrew/bin:/usr/local/bin:$HOME/.local/bin:$PATH"

log() { echo "[$(date -u +%H:%M:%S)] $*" | tee -a "$LOG"; }

notify() {
  local title="$1" msg="$2"
  command -v terminal-notifier >/dev/null 2>&1 \
    && terminal-notifier -title "$title" -message "$msg" -group jobhub-scan 2>/dev/null
  # osascript works without an extra dependency; ignore failure under launchd.
  command -v osascript >/dev/null 2>&1 \
    && osascript -e "display notification \"${msg//\"/}\" with title \"${title//\"/}\"" 2>/dev/null
  return 0
}

fail() {
  log "FAILED: $*"
  notify "JobHub scan failed" "$*"
  echo "$LOG"
  exit 1
}

: "${JOBHUB_URL:?JOBHUB_URL is not set — launchd needs it in the plist EnvironmentVariables}"
: "${OLLAMA_HOST:?OLLAMA_HOST is not set — launchd needs it in the plist EnvironmentVariables}"

log "repo=$REPO work=$WORK"
log "jobhub=$JOBHUB_URL ollama=$OLLAMA_HOST"

# ---------------------------------------------------------------- 1. scan
log "stage 1/4 scan"
python3 -u scripts/scan.py --out "$WORK/candidates.json" >>"$LOG" 2>&1 \
  || fail "scan.py failed — see $LOG"

COUNT="$(python3 -c "import json,sys;print(len(json.load(open(sys.argv[1]))['candidates']))" \
         "$WORK/candidates.json" 2>/dev/null || echo 0)"
log "scan produced $COUNT new candidates"

if [[ "$COUNT" -eq 0 ]]; then
  log "nothing new; stopping before the model stages"
  notify "JobHub scan" "No new roles today."
  exit 0
fi

# --------------------------------------------------------------- 2. triage
log "stage 2/4 triage (local ollama, ~7s per posting)"
python3 -u scripts/triage.py "$WORK/candidates.json" --out "$WORK/triaged.json" >>"$LOG" 2>&1 \
  || fail "triage.py failed — is Ollama up at $OLLAMA_HOST? see $LOG"

# --------------------------------------------------------------- 3. appeal
# A failed appeal must not sink the run: an un-appealed queue is degraded but
# still useful, and the digest records that the pass did not happen.
log "stage 3/4 appeal (batched haiku)"
if ! python3 -u scripts/appeal.py "$WORK/triaged.json" --out "$WORK/appealed.json" >>"$LOG" 2>&1; then
  log "WARNING: appeal.py failed — continuing with un-appealed results"
  cp "$WORK/triaged.json" "$WORK/appealed.json"
fi

# ----------------------------------------------------------------- 4. post
log "stage 4/4 post"
DIGEST="$REPO/user/search-results/$STAMP-scan.md"
POST_RC=0
python3 -u scripts/post_results.py "$WORK/appealed.json" --digest "$DIGEST" $DRY_RUN >>"$LOG" 2>&1 \
  || POST_RC=$?

# post_results.py refuses to overwrite an existing digest and writes
# {date}-scan-{HHMM}.md instead, so $DIGEST is not necessarily where it landed.
# Take the path it reported. Under launchd the notification is the only thing
# seen, and pointing it at a stale digest is worse than not sending one.
ACTUAL="$(grep -a '^digest: ' "$LOG" | tail -1 | sed 's/^digest: //')"
[[ -n "$ACTUAL" && -f "$ACTUAL" ]] && DIGEST="$ACTUAL"

KEPT="$(python3 -c "import json,sys;print(len(json.load(open(sys.argv[1])).get('kept',[])))" \
        "$WORK/appealed.json" 2>/dev/null || echo '?')"
OVER="$(python3 -c "import json,sys;print((json.load(open(sys.argv[1])).get('appeal') or {}).get('overturned',0))" \
        "$WORK/appealed.json" 2>/dev/null || echo 0)"

if [[ "$POST_RC" -ne 0 ]]; then
  log "server POST failed; digest written to $DIGEST"
  notify "JobHub scan — server POST failed" "$KEPT roles found but NOT on the dashboard. See $DIGEST"
else
  log "done: $KEPT kept ($OVER rescued by appeal)"
  notify "JobHub scan" "$KEPT roles kept, $OVER rescued by appeal."
fi

rm -rf "$WORK"
log "log: $LOG"
log "digest: $DIGEST"
exit "$POST_RC"
