# Scheduling the daily scan

`run_daily_scan.sh` chains scan → triage → appeal → post. This wires it to launchd so it runs
every morning without being asked.

## Install

```bash
REPO="$HOME/job-search"

sed -e "s|__REPO__|$REPO|g" \
    -e "s|__HOME__|$HOME|g" \
    -e "s|__OLLAMA_HOST__|$OLLAMA_HOST|g" \
    "$REPO/scripts/launchd/com.jobhub.dailyscan.plist" \
    > ~/Library/LaunchAgents/com.jobhub.dailyscan.plist

launchctl bootstrap "gui/$(id -u)" ~/Library/LaunchAgents/com.jobhub.dailyscan.plist
```

**No API token any more.** The server was retired on 2026-09-11 and the funnel is
`user/jobhub.db`, a local file found relative to the repo. `JOBHUB_URL` and `JOBHUB_API_TOKEN` are
gone from the template; if your installed copy still has them, re-install from the template above.

The store must exist before the first run. `run_daily_scan.sh` fails loudly if it does not:

```bash
python3 -c "import sys;sys.path.insert(0,'$REPO/scripts');import jobhub_db as d;\
print(d.load_boards_export(d.connect(), '$REPO/server/export/boards-<date>.json'), 'boards')"
```

## Verify it works

Run it once by hand before trusting the schedule:

```bash
launchctl kickstart -k "gui/$(id -u)/com.jobhub.dailyscan"
tail -f user/search-results/logs/$(date -u +%F).log
```

You should get a notification, a digest at `user/search-results/YYYY-MM-DD-scan.md`, and a batch
on the dashboard.

## Manage

```bash
launchctl print "gui/$(id -u)/com.jobhub.dailyscan"      # status, last exit code
launchctl kickstart -k "gui/$(id -u)/com.jobhub.dailyscan"  # run now
launchctl bootout "gui/$(id -u)/com.jobhub.dailyscan"    # uninstall
```

## Things that will bite you

**The laptop must be awake at 07:15.** launchd skips a missed calendar interval on a sleeping
Mac rather than deferring it — if the machine was closed, that day simply does not run. Check
the digest dates if the funnel looks dry. `caffeinate` or a `pmset repeat wake` entry fixes it
if you care; otherwise just run it by hand after opening the lid.

**The first run takes about 100 minutes.** It triages every posting on all 77 boards because
nothing has been seen yet. Every run after that only sees new postings and takes a few minutes.

**launchd does not inherit your shell environment.** `OLLAMA_HOST` comes from the plist's
`EnvironmentVariables`, and `PATH` is set explicitly so Homebrew `python3` and the `claude` CLI are
findable. If you change it in your shell profile, update the plist too — the script exits
immediately with a clear error rather than running against the wrong host. The funnel store needs
no variable at all; it is found relative to the repo.

**Appeals cost money.** Roughly $0.004 per drop reviewed. A normal day is a few cents; the first
backfill is a couple of dollars. Every run records its spend in the digest.
