#!/usr/bin/env python3
"""Stage 4: publish the kept roles to JobHub and write the flat-file fallback.

    scan.py  ->  triage.py  ->  appeal.py  ->  post_results.py

Follows the dual-write rule in `prompts/commands/job.md`: POST to the server,
then write the flat file regardless. If the server is unreachable the file is
still written and the failure is reported loudly -- nothing is silently lost,
and nobody is told a record landed on the dashboard when it did not.

Usage:
    python3 scripts/post_results.py appealed.json
    python3 scripts/post_results.py appealed.json --digest user/search-results/x.md
    python3 scripts/post_results.py appealed.json --dry-run
"""
import argparse
import json
import os
import sys
import urllib.error
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from scan import REPO, SEEN, record_seen  # noqa: E402

REPO = Path(__file__).resolve().parent.parent
OUTDIR = REPO / 'user' / 'search-results'

JOBHUB_URL = os.environ.get('JOBHUB_URL', 'http://localhost:8080').rstrip('/')
JOBHUB_TOKEN = os.environ.get('JOBHUB_API_TOKEN', '')

TIER_ORDER = {'strong': 0, 'good': 1}


def to_result(p):
    """Map a triaged posting onto the /api/search-results row shape."""
    return {
        'company': p.get('company') or 'Unknown',
        'role_title': p.get('title') or 'Unknown',
        'location': p.get('location') or None,
        'is_remote': bool(p.get('is_remote')),
        'salary_min': p.get('salary_min'),
        'salary_max': p.get('salary_max'),
        'salary_disclosed': p.get('salary_min') is not None,
        'below_floor': False,  # the comp floor is withdrawn; never gate on comp
        'posting_url': p.get('url') or '',
        'fit_tier': p.get('tier') if p.get('tier') in TIER_ORDER else 'good',
        'tags': [t for t in [p.get('domain_tag'), p.get('level_tag'),
                             'appealed' if p.get('appealed') else None] if t],
        'level_tag': p.get('level_tag') if p.get('level_tag') in ('senior', 'staff') else None,
        'domain_tag': p.get('domain_tag') if p.get('domain_tag') != 'other' else None,
    }


def post(payload):
    req = urllib.request.Request(
        JOBHUB_URL + '/api/search-results',
        data=json.dumps(payload).encode(),
        headers={'Content-Type': 'application/json'})
    if JOBHUB_TOKEN:
        req.add_header('Authorization', f'Bearer {JOBHUB_TOKEN}')
    return json.load(urllib.request.urlopen(req, timeout=60))


def write_digest(path, data, kept, server_url, server_err):
    ap = data.get('appeal') or {}
    lines = [
        f"# Job scan — {datetime.now(timezone.utc).strftime('%Y-%m-%d %H:%M UTC')}",
        '',
        f"- Boards scanned: **{data.get('board_count', '?')}**",
        f"- Raw postings: **{data.get('raw_count', '?')}** → "
        f"prefiltered **{data.get('prefiltered_count', '?')}** → "
        f"new **{len(kept) + len(data.get('dropped', []))}**",
        f"- Kept: **{len(kept)}**",
        f"- Appeals: {ap.get('appealed', 0)} reviewed, **{ap.get('overturned', 0)} overturned**, "
        f"{ap.get('vetoed_skipped', 0)} vetoed drops skipped, cost **${ap.get('cost_usd', 0):.4f}**",
        f"- Triage model: `{data.get('model', '?')}`",
        '',
    ]
    if server_err:
        lines += [f'> **Server POST FAILED — nothing reached the dashboard.** `{server_err}`',
                  '> This file is the only record of the scan. Fix and re-post.', '']
    elif server_url:
        lines += [f'Dashboard: {server_url}', '']

    om = data.get('rejected_omitted') or {}
    if om:
        total = sum(om.values())
        lines += [f'**Omitted {total} posting(s)** at {len(om)} company(ies) that already '
                  'rejected an application: '
                  + ', '.join(f'{c} ({n})' for c, n in sorted(om.items(), key=lambda kv: -kv[1]))
                  + '.', '',
                  '_Run `scan.py --include-rejected` to see them._', '']

    # scan.py already prints these to the run log, but the digest is what actually
    # gets read -- and a carve-out that only exists in a log file is exactly the
    # invisible-exception problem the carve-out mechanism was built to prevent.
    carve = data.get('rejection_carveouts') or []
    if carve:
        lines += ['**Kept visible despite a rejection** — '
                  + ', '.join(f"{c.get('company')} ({c.get('reason') or 'unstated'})"
                              for c in carve)
                  + '.', '',
                  '_These companies rejected an application but formed no view of the candidate, '
                  'so their postings are still scanned. See `user/rejections.json`._', '']

    if data.get('errors'):
        lines += ['## Sources that failed', '',
                  '_Endpoint drift. Re-discover the tenant/site or fix the slug._', '']
        for label, err in sorted(data['errors'].items()):
            lines.append(f'- `{label}` — {err}')
        lines.append('')

    lines += ['## Kept roles', '']
    if not kept:
        lines.append('_Nothing cleared the screen this run._')
    else:
        lines += ['| Tier | Company | Role | Location | Salary | Why |',
                  '|---|---|---|---|---|---|']
        for p in kept:
            sal = (f"${p['salary_min']//1000}k-${p['salary_max']//1000}k"
                   if p.get('salary_min') else '—')
            why = (p.get('appeal_reason') or p.get('reason') or '').replace('|', '/')[:70]
            mark = ' ⤴' if p.get('appealed') else ''
            lines.append(f"| {p.get('tier','?')}{mark} | {p.get('company')} | "
                         f"[{(p.get('title') or '')[:52]}]({p.get('url')}) | "
                         f"{(p.get('location') or '')[:34]} | {sal} | {why} |")
        lines += ['', '_⤴ = rescued by the appeal pass after the local model dropped it._']

    lines += ['', '## Next', '',
              'Pick 5-8 and run `/job-auto` to build application packets.', '']
    path.write_text('\n'.join(lines))


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('appealed')
    ap.add_argument('--digest')
    ap.add_argument('--dry-run', action='store_true')
    a = ap.parse_args()

    data = json.loads(Path(a.appealed).read_text())
    kept = data.get('kept', [])

    # scan.py already omits rejected companies, so this normally removes nothing.
    # It is a backstop for queues produced before that filter existed -- a
    # rejected company reaching the dashboard is the one outcome Andre asked
    # not to see.
    late = {}
    survivors = []
    for p in kept:
        if p.get('prior_rejection'):
            c = p.get('company') or '?'
            late[c] = late.get(c, 0) + 1
        else:
            survivors.append(p)
    if late:
        data.setdefault('rejected_omitted', {})
        for c, n in late.items():
            data['rejected_omitted'][c] = data['rejected_omitted'].get(c, 0) + n
        print(f'dropped {sum(late.values())} kept role(s) at rejected companies: '
              + ', '.join(sorted(late)))

    kept = sorted(survivors,
                  key=lambda p: (TIER_ORDER.get(p.get('tier'), 9), p.get('company', '')))

    payload = {
        'ran_at': data.get('ran_at') or datetime.now(timezone.utc).isoformat(),
        'board_count': data.get('board_count', 0),
        'raw_count': data.get('raw_count', 0),
        'location_filter': 'remote-US + Philadelphia/South Jersey/Delaware + NYC/North NJ',
        'results': [to_result(p) for p in kept],
    }

    server_url, server_err = None, None
    if a.dry_run:
        print(f'--dry-run: would POST {len(payload["results"])} results to {JOBHUB_URL}')
    elif not payload['results']:
        print('no kept roles; skipping POST')
    else:
        try:
            r = post(payload)
            server_url = JOBHUB_URL + (r.get('url') or '')
            print(f'posted {len(payload["results"])} results -> {server_url}')
        except urllib.error.HTTPError as e:
            server_err = f'HTTP {e.code}: {e.read()[:200].decode(errors="replace")}'
        except Exception as e:
            server_err = f'{type(e).__name__}: {e}'
        if server_err:
            # Surface it. The flat file below is written either way.
            print(f'SERVER POST FAILED: {server_err}', file=sys.stderr)

    OUTDIR.mkdir(parents=True, exist_ok=True)
    digest = Path(a.digest) if a.digest else (
        OUTDIR / f"{datetime.now(timezone.utc).strftime('%Y-%m-%d')}-scan.md")

    # Refuse to destroy an existing digest. On 2026-08-21 a second scan
    # overwrote the morning's 266-role queue with a 4-role one, and the roles
    # could not be recovered by re-running because they were already marked
    # seen. Never clobber; write alongside.
    if digest.exists():
        alt = digest.with_name(
            f"{digest.stem}-{datetime.now(timezone.utc).strftime('%H%M')}{digest.suffix}")
        print(f'{digest.name} exists; writing {alt.name} instead so nothing is lost')
        digest = alt

    write_digest(digest, data, kept, server_url, server_err)
    print(f'digest: {digest}')

    # The dedup ledger is written HERE, not in scan.py, and only now that the
    # digest is on disk. The ledger means "this reached Andre", not "this was
    # fetched" -- so a crash anywhere upstream leaves these unrecorded and they
    # come back on the next run instead of vanishing.
    #
    # Both kept and dropped are recorded: a drop is a decision that was made and
    # re-triaging it tomorrow would spend model time to reach the same answer.
    # Vetoed and rejected-company postings never get here, which is intended --
    # re-checking those costs a regex, and it means a veto rule change resurfaces
    # everything the old rule excluded.
    decided = [p for p in (kept + data.get('dropped', [])) if p.get('url')]
    if a.dry_run:
        print(f'--dry-run: {len(decided)} posting(s) NOT recorded in the dedup ledger')
    else:
        record_seen(decided, data.get('ran_at') or datetime.now(timezone.utc).isoformat())
        # Display only -- relative_to() raises when the ledger is not under the
        # repo, and this line runs after the write, so a cosmetic path problem must
        # not abort the stage.
        try:
            where = SEEN.relative_to(REPO)
        except ValueError:
            where = SEEN
        print(f'recorded {len(decided)} posting(s) in {where} '
              f'({len(kept)} kept, {len(decided) - len(kept)} dropped)')

    return 1 if server_err else 0


if __name__ == '__main__':
    sys.exit(main())
