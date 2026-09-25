#!/usr/bin/env python3
"""Scan non-Greenhouse ATS platforms for roles in a commute radius.

Greenhouse covers tech companies. Large regional employers -- banks, insurers,
health systems, manufacturers -- run Workday, Oracle Fusion or iCIMS instead,
and none of them appear in the tracked Greenhouse board list. This scans them.

Usage:
    python3 scripts/ats_scan.py                 # all employers, default region
    python3 scripts/ats_scan.py --all-locations # skip the region filter
    python3 scripts/ats_scan.py --json out.json

This is the standalone human-readable probe. The employer list, fetchers and
screening regexes live in `ats_sources.py`, shared with `scan.py` -- add new
employers there.
"""
import argparse
import json
import sys
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from ats_sources import EMPLOYERS, fetch_employer, prefilter  # noqa: E402


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--all-locations', action='store_true')
    ap.add_argument('--json')
    a = ap.parse_args()

    results, errors, total_seen = {}, {}, 0
    with ThreadPoolExecutor(max_workers=6) as ex:
        for name, jobs, err in ex.map(fetch_employer, EMPLOYERS):
            if err:
                errors[name] = err
                continue
            total_seen += len(jobs)
            results[name] = [j for j in jobs if prefilter(j, a.all_locations)]

    print(f'Scanned {len(EMPLOYERS)-len(errors)} employers, {total_seen} total reqs\n')
    hits = 0
    for name, jobs in sorted(results.items(), key=lambda kv: -len(kv[1])):
        if not jobs:
            continue
        print(f'== {name} ({len(jobs)})')
        for j in sorted(jobs, key=lambda x: str(x['posted']), reverse=True):
            print(f"   {j['title']}")
            print(f"      {j['location']}  |  {j['posted']}")
            print(f"      {j['url']}")
            hits += 1
        print()
    empty = [n for n, v in results.items() if not v]
    if empty:
        print('No regional senior+ engineering roles: ' + ', '.join(sorted(empty)))
    if errors:
        print('\nERRORS (endpoint drift -- re-discover the tenant/site):')
        for n, e in errors.items():
            print(f'  {n}: {e}')
    print(f'\nTOTAL: {hits}')
    if a.json:
        json.dump(results, open(a.json, 'w'), indent=1)
        print(f'wrote {a.json}')


if __name__ == '__main__':
    main()
