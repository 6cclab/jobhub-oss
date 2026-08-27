#!/usr/bin/env python3
"""Stage 1 of the automated funnel: fetch every reachable board, screen, dedup.

    scan.py  ->  triage.py  ->  appeal.py  ->  POST to JobHub

Pulls the tracked board list from the JobHub server, fetches each board plus the
enterprise employers in `ats_sources.EMPLOYERS`, applies the cheap deterministic
prefilter, drops anything seen in a previous scan, and writes the survivors as
JSON for `triage.py` to score.

No model is called here. This stage exists to make the model stage small.

Usage:
    python3 scripts/scan.py --out candidates.json
    python3 scripts/scan.py --dry-run          # report counts, touch nothing
    python3 scripts/scan.py --all-locations    # ignore the commute radius
    python3 scripts/scan.py --no-dedup         # re-emit everything

Exit codes: 0 ok (even with zero new roles), 1 hard failure (no board list).
"""
import argparse
import json
import os
import re
import sys
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timedelta, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from ats_sources import EMPLOYERS, fetch_board, fetch_employer, parse_salary, prefilter  # noqa: E402
from vetoes import check as veto_check  # noqa: E402

REPO = Path(__file__).resolve().parent.parent
SEEN = REPO / 'user' / 'search-results' / '.seen.jsonl'

JOBHUB_URL = os.environ.get('JOBHUB_URL', 'http://localhost:8080').rstrip('/')
JOBHUB_TOKEN = os.environ.get('JOBHUB_API_TOKEN', '')


def api_get(path):
    req = urllib.request.Request(JOBHUB_URL + path)
    if JOBHUB_TOKEN:
        req.add_header('Authorization', f'Bearer {JOBHUB_TOKEN}')
    return json.load(urllib.request.urlopen(req, timeout=30))


def load_boards():
    """Tracked boards from the server. Falls back to nothing -- see caller."""
    d = api_get('/api/boards')
    return [b for b in d.get('tracked', []) if b.get('slug')]


def load_rejected_companies():
    """Companies that already rejected the candidate, from the tracker.

    These are OMITTED, not flagged. Andre, 2026-08-21: "if previously rejected
    then omit." An earlier version surfaced them with a warning marker and he
    asked for them gone.

    Dropping them here rather than at triage also skips the model call and the
    appeal cost for every one. The omission is reported in the digest -- a
    filter nobody can see is how a funnel narrows without anyone noticing.

    `--include-rejected` overrides this for a one-off run.

    Companies carved out by `user/rejections.json` are subtracted -- a rejection
    only hides a company when the company actually formed a view. See that file.

    Returns (omitted, kept_visible): the second is for the digest, so an
    exception is as visible as the omission it overrides.
    """
    try:
        d = api_get('/api/applications?status=rejected')
    except Exception:
        return set(), []  # advisory only; never fail a scan over it
    rejected = {(a.get('company') or '').strip().lower()
                for a in d.get('applications', []) if a.get('company')}
    carve = load_rejection_carveouts()
    kept = [(name, reason) for name, (exclude, reason) in carve.items()
            if not exclude and name in rejected]
    return rejected - {n for n, _ in kept}, kept


def load_rejection_carveouts():
    """`user/rejections.json` as {company_lower: (exclude, reason)}.

    DEFAULT IS EXCLUDE. A company absent from the file is omitted exactly as
    before, so a stale or forgotten entry can never silently widen the funnel --
    only a deliberate `exclude: false` does, and that one shows up in the digest.

    A missing or malformed file is not fatal. Failing a scan over a data file
    would be a worse outcome than falling back to the old behaviour, which is
    also the safe direction: everything stays excluded.
    """
    path = REPO / 'user' / 'rejections.json'
    if not path.exists():
        return {}
    try:
        data = json.loads(path.read_text())
    except (json.JSONDecodeError, OSError) as e:
        print(f'warning: could not read {path.name} ({e}); '
              f'every rejected company stays omitted', file=sys.stderr)
        return {}
    out = {}
    for row in data.get('rejections', []):
        name = (row.get('company') or '').strip().lower()
        if name:
            out[name] = (bool(row.get('exclude', True)), row.get('reason') or 'unstated')
    return out


def _norm_title(t):
    return re.sub(r'\s+', ' ', re.sub(r'[^a-z0-9 ]', ' ', (t or '').lower())).strip()


def collapse_roles(postings):
    """Collapse one role posted to many locations into a single candidate.

    Greenhouse and Ashby emit a separate posting per office for the same req --
    ClickHouse alone contributes six rows for one Java streaming role. Triaging
    each is a wasted model call, and six identical dashboard rows is noise.

    Keeps the first posting's URL as canonical, merges the locations, and marks
    the role remote if ANY of its postings were.
    """
    groups = {}
    for p in postings:
        key = (p.get('company', '').lower(), _norm_title(p.get('title')))
        if key in groups:
            g = groups[key]
            loc = p.get('location')
            if loc and loc not in g['_locs']:
                g['_locs'].append(loc)
            g['is_remote'] = g['is_remote'] or p.get('is_remote', False)
            # Prefer whichever copy actually carried a description.
            if not g.get('description') and p.get('description'):
                g['description'] = p['description']
        else:
            g = dict(p)
            g['_locs'] = [p['location']] if p.get('location') else []
            groups[key] = g

    out = []
    for g in groups.values():
        locs = g.pop('_locs')
        if len(locs) > 1:
            g['location'] = '; '.join(locs[:4]) + (f' (+{len(locs)-4} more)' if len(locs) > 4 else '')
            g['location_count'] = len(locs)
        out.append(g)
    return out


# Andre, 2026-08-21: "we should never re-fetch listings within 30 days."
# Entries older than this expire, so a req still open after a month can come
# back round -- it may have been reposted, rescoped, or simply missed the first
# time. Before this the ledger was permanent and nothing ever returned.
SEEN_TTL_DAYS = 30


def load_seen(ttl_days=SEEN_TTL_DAYS):
    """url -> first_seen, for entries inside the TTL window.

    Entries past the window are omitted, so the caller treats them as new. The
    file is not rewritten -- expiry is a read-time decision, which keeps the
    ledger an append-only record of what was actually surfaced and when.
    """
    if not SEEN.exists():
        return {}
    cutoff = datetime.now(timezone.utc) - timedelta(days=ttl_days)
    out, expired = {}, 0
    for line in SEEN.read_text().splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            r = json.loads(line)
            url = r['url']
        except (json.JSONDecodeError, KeyError):
            continue  # a corrupt line must not sink the scan
        stamp = r.get('first_seen', '')
        try:
            when = datetime.fromisoformat(stamp)
            if when.tzinfo is None:
                when = when.replace(tzinfo=timezone.utc)
        except ValueError:
            # No parseable date: treat as permanently seen rather than
            # resurfacing it on every run.
            out[url] = stamp
            continue
        if when < cutoff:
            expired += 1
            continue
        out[url] = stamp
    if expired:
        print(f'{expired} ledger entr(ies) older than {ttl_days}d expired; eligible again')
    return out


def record_seen(postings, when, reason=None):
    """Mark postings as seen. Call this ONLY after they have reached a digest.

    Recording at scan time -- which this did until 2026-08-21 -- loses roles
    silently. That day a second scan overwrote the morning digest, and because
    all 845 postings had already been marked seen at fetch time, the 262 roles
    that vanished with it could never reappear in a future scan. Andre:
    "no silent losses. If I've seen it then dont go get it again."

    So the ledger now means "this reached him", not "this was fetched". A crash
    or a lost digest leaves the postings unrecorded and they come back tomorrow.

    Vetoed and rejected-company postings are deliberately NOT recorded. Re-checking
    them costs a regex, and leaving them out means a veto rule change resurfaces
    everything it used to exclude -- which is how the Staff filter change on
    2026-08-21 should have behaved and could not.
    """
    SEEN.parent.mkdir(parents=True, exist_ok=True)
    with SEEN.open('a') as f:
        for p in postings:
            row = {'url': p['url'], 'first_seen': when,
                   'company': p['company'], 'title': p['title']}
            r = reason or p.get('_seen_reason')
            if r:
                row['reason'] = r
            f.write(json.dumps(row) + '\n')


def forget(substr):
    """Expire reason-tagged ledger entries matching substr. Returns the count.

    Only entries carrying a `reason` are eligible -- those are the deterministic
    exclusions (vetoes, rejected companies). Roles that actually reached a digest
    have no reason field and are never removed by this, because forgetting those
    would resurface things Andre has already reviewed.
    """
    if not SEEN.exists():
        return 0
    keep, dropped = [], 0
    for line in SEEN.read_text().splitlines():
        s = line.strip()
        if not s:
            continue
        try:
            r = json.loads(s)
        except json.JSONDecodeError:
            keep.append(s)      # preserve unparseable lines rather than eating them
            continue
        reason = r.get('reason')
        if reason and (substr == 'all' or substr.lower() in reason.lower()):
            dropped += 1
            continue
        keep.append(s)
    if dropped:
        SEEN.write_text('\n'.join(keep) + ('\n' if keep else ''))
    return dropped


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--out', default='candidates.json')
    ap.add_argument('--dry-run', action='store_true')
    ap.add_argument('--all-locations', action='store_true')
    ap.add_argument('--no-dedup', action='store_true')
    ap.add_argument('--forget', metavar='SUBSTR',
                    help='drop ledger entries whose reason contains SUBSTR, making them '
                         'eligible again immediately. Use after changing a veto rule: '
                         "--forget 'Staff+ title'. Use --forget all to clear every "
                         'reason-tagged entry. Kept/dropped entries carry no reason and '
                         'are never touched.')
    ap.add_argument('--record', action='store_true',
                    help='mark candidates seen at scan time. Off by default: the '
                         'ledger is written by post_results.py once a digest exists.')
    ap.add_argument('--no-record', action='store_true',
                    help='write the output file but do not mark postings as seen')
    ap.add_argument('--boards-only', action='store_true')
    ap.add_argument('--no-veto', action='store_true',
                    help='skip the deterministic vetoes; useful for auditing what they remove')
    ap.add_argument('--include-rejected', action='store_true',
                    help='do not omit companies that already rejected an application')
    ap.add_argument('--workers', type=int, default=8)
    a = ap.parse_args()

    if a.forget:
        n = forget(a.forget)
        print(f"forgot {n} ledger entr(ies) matching {a.forget!r}; "
              f"they are eligible on the next scan")
        return 0

    try:
        boards = load_boards()
    except Exception as e:
        # Without the board list there is no scan. Say so rather than silently
        # scanning only the dozen hardcoded enterprise employers.
        print(f'FATAL: could not read {JOBHUB_URL}/api/boards -- {type(e).__name__}: {e}',
              file=sys.stderr)
        return 1

    jobs = [(b.get('ats'), b['slug'], b.get('name')) for b in boards]
    print(f'{len(jobs)} boards from {JOBHUB_URL}'
          + ('' if a.boards_only else f' + {len(EMPLOYERS)} enterprise employers'))

    raw, errors = [], {}
    with ThreadPoolExecutor(max_workers=a.workers) as ex:
        futures = [ex.submit(fetch_board, ats, slug, name) for ats, slug, name in jobs]
        if not a.boards_only:
            futures += [ex.submit(fetch_employer, n) for n in EMPLOYERS]
        for fut in futures:
            label, postings, err = fut.result()
            if err:
                errors[label] = err
            else:
                raw += postings

    kept = [p for p in raw if prefilter(p, a.all_locations)]
    kept = collapse_roles(kept)

    seen = {} if a.no_dedup else load_seen()
    fresh = [p for p in kept if p['url'] and p['url'] not in seen]

    if a.include_rejected:
        rejected, carved = set(), []
    else:
        rejected, carved = load_rejected_companies()
    omitted = {}
    # Deterministic vetoes run HERE, before the model, not only in triage.py.
    # They are absolute rules -- Staff+ titles, defense, 10+ years, closed
    # companies -- so nothing is gained by paying a model to look at them first.
    # Measured 2026-08-21: 310 of 739 candidates (42%) were vetoable, 284 of them
    # Staff-titled, cutting a triage pass from ~17 to ~10 minutes.
    #
    # triage.py still applies the same vetoes after the model. That is deliberate
    # belt-and-braces, not a leftover: it costs nothing and it means a candidate
    # reaching triage by some other path is still checked.
    vetoed = {}
    survivors = []
    excluded = []            # vetoed + rejected-company, recorded so they are
                             # not re-processed on every run inside the TTL
    for p in fresh:
        company = (p.get('company') or '').strip()
        if company.lower() in rejected:
            omitted[company] = omitted.get(company, 0) + 1
            p['_seen_reason'] = f'rejected company: {company}'
            excluded.append(p)
            continue
        reason = None if a.no_veto else veto_check(p)
        if reason:
            vetoed[reason] = vetoed.get(reason, 0) + 1
            p['_seen_reason'] = reason
            excluded.append(p)
            continue
        lo, hi = parse_salary(p.get('description'))
        p['salary_min'], p['salary_max'] = lo, hi
        survivors.append(p)
    fresh = survivors

    n_omitted = sum(omitted.values())
    n_vetoed = sum(vetoed.values())
    print(f'raw {len(raw)}  ->  prefilter {len(kept)}  ->  new {len(fresh)}'
          f'  ({len(kept)-len(fresh)-n_omitted-n_vetoed} already seen,'
          f' {n_omitted} omitted at rejected companies,'
          f' {n_vetoed} vetoed)')
    if omitted:
        print('  omitted: ' + ', '.join(f'{c} ({n})' for c, n in
                                        sorted(omitted.items(), key=lambda kv: -kv[1])))
    # An EXCEPTION nobody can see is the same failure as a filter nobody can see,
    # in the other direction: a company staying visible after a rejection is a
    # deliberate carve-out and should be as legible as the omission it overrides.
    for name, reason in sorted(carved):
        print(f'  kept visible despite a rejection: {name} ({reason})')
    # A filter nobody can see is how a funnel narrows without anyone noticing.
    for reason, n in sorted(vetoed.items(), key=lambda kv: -kv[1]):
        print(f'  vetoed {n:4}  {reason}')

    if errors:
        # Endpoint drift is normal and must be visible, not swallowed.
        print(f'\n{len(errors)} source(s) failed:', file=sys.stderr)
        for label, err in sorted(errors.items()):
            print(f'  {label}: {err}', file=sys.stderr)

    if a.dry_run:
        print('\n--dry-run: nothing written')
        for p in fresh[:15]:
            print(f"  [{p['source']:10}] {p['company']:22} {p['title'][:58]}")
        if len(fresh) > 15:
            print(f'  ... and {len(fresh)-15} more')
        return 0

    now = datetime.now(timezone.utc).isoformat()
    payload = {'ran_at': now, 'board_count': len(jobs),
               'raw_count': len(raw), 'prefiltered_count': len(kept),
               'rejected_omitted': omitted,
               'rejection_carveouts': [{'company': n, 'reason': r} for n, r in carved],
               'vetoed': vetoed,
               'errors': errors, 'candidates': fresh}
    Path(a.out).write_text(json.dumps(payload, indent=1))
    # Vetoed and rejected-company postings are recorded HERE rather than after a
    # digest, because they are deterministic exclusions that will never reach a
    # digest at all -- leaving them out meant re-processing 310 of them on every
    # run, which is the re-fetching Andre asked to stop. They are not a silent
    # loss: the exclusion is a stated rule, and the reason is written alongside.
    #
    # The cost is that a veto RULE change no longer resurfaces affected roles for
    # up to SEEN_TTL_DAYS. Use --forget to clear them deliberately when that
    # happens; that is what it is for.
    if excluded and not a.no_record and not a.dry_run:
        record_seen(excluded, now)
        print(f'recorded {len(excluded)} excluded posting(s) '
              f'({sum(vetoed.values())} vetoed, {n_omitted} rejected-company)')

    if a.record and not a.no_record:
        record_seen(fresh, now)
        print(f'wrote {a.out} ({len(fresh)} candidates); --record: written to '
              f'{SEEN.relative_to(REPO)} before any digest exists')
    else:
        print(f'wrote {a.out} ({len(fresh)} candidates); '
              f'kept/dropped go in the ledger via post_results.py after the digest')
    return 0


if __name__ == '__main__':
    sys.exit(main())
