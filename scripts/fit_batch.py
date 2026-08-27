#!/usr/bin/env python3
"""Fit-report a screened queue against Andre's actual record.

The scan pipeline (scan -> triage -> appeal -> post_results) produces a SCREEN:
title, location and ~700 characters of description judged against
`user/screen-profile.md`. It never reads his history, so a `strong` tier there
means "this looks like the kind of role he wants", not "he matches it".

This closes that gap. For each posting it produces the same artifact `/job`
produces by hand -- match signals with the evidence behind each one, honest gaps,
flags against his preferences, and a verdict -- and POSTs it to
`/api/fit-reports`, so the dashboard ranking is built on his record rather than
on a title match.

    python3 scripts/fit_batch.py roles.json --out fit.json

Deliberately NOT part of run_daily_scan.sh. The screen is cheap and runs daily;
this is heavier and is worth running against a queue you actually intend to work
through.
"""
import argparse
import json
import os
import re
import subprocess
import sys
import time
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
JOBHUB = os.environ.get('JOBHUB_URL', 'http://localhost:8080')
TOKEN = os.environ.get('JOBHUB_API_TOKEN', '')

# Small on purpose. The appeal batch-size experiment (2026-08-21) measured recall
# collapsing as batch size grew -- 4 rescues at 25, 13 at 5, 25 at 1 on one
# identical drop list -- on a task far simpler than this one. A fit report has to
# hold his whole record against a whole posting, so if a keep/drop judgement
# degrades at 25, this degrades sooner. Three is a compromise with wall-clock,
# not a claim that three is safe.
BATCH = 3
DESC_CHARS = 6000

VERDICTS = ('strong', 'worth', 'stretch', 'skip')

PROMPT_HEAD = """You are evaluating job postings against ONE candidate's verified record. Be accurate, not encouraging: a wrong `strong` costs him a day of tailoring, and an inflated match signal ends up on a resume and then in an interview he cannot back up.

=== THE CANDIDATE'S RECORD (the only source of truth about him) ===
{record}

=== HIS PREFERENCES AND DEAL-BREAKERS ===
{profile}

=== HARD RULES ===
- Every match signal MUST quote or closely paraphrase something in the record above. If you cannot point to it, it is not a match signal.
- These are CONFIRMED GAPS. Never report them as matches, and never soften them: Kafka (dabbled, never shipped), Temporal the workflow engine (never used), ledger accounting, credit decisioning, underwriting (never touched), Java/JVM/Kotlin/Scala (never shipped).
- MySQL is NOT a gap. He treats it as interchangeable with PostgreSQL and his relational depth is real.
- Go is real but PERSONAL-PROJECT ONLY -- never professional. Say so when a posting wants production Go.
- Staff+ titled roles are out of scope unless Senior is offered alongside.
- A term missing from his record is only a gap if the CAPABILITY is missing. Do not report a phrase as a gap when the evidence is there under different words.
- If a posting has no description text, return verdict "skip" with verdict_summary "insufficient posting text to evaluate" and leave the signal arrays empty. Do NOT guess from the title.

=== OUTPUT ===
Return ONLY a JSON array, one object per posting, no prose outside it:
[{{"n": 1,
   "verdict": "strong|worth|stretch|skip",
   "verdict_summary": "one sentence, concrete",
   "level": "senior|staff|mid|unclear",
   "match": [{{"requirement": "what they ask for", "evidence": "how he maps, naming the specific work"}}],
   "gap": [{{"requirement": "what they ask for", "evidence": "honest assessment"}}],
   "flag": [{{"requirement": "the conflict", "evidence": "why it matters"}}]}}]

Verdict scale: `strong` = 70%+ match, gaps learnable, no flags. `worth` = solid overlap, real gaps. `stretch` = significant gaps, only worth it for a specific reason. `skip` = misaligned on level, domain, location or deal-breaker.

Aim for 2-5 match signals, 1-4 gaps, and only real flags. An empty flag array is the normal case.

=== POSTINGS ===
{postings}"""


def build_record():
    """His record, trimmed to what a fit judgement needs.

    Sent whole rather than retrieved per-requirement: retrieval is what the KB
    path does for a single role, and 181 roles times 6 queries is both slower and
    worse -- a missed chunk becomes a fabricated gap, which is the failure mode
    this whole script exists to avoid.
    """
    parts = []
    for name in ('master-resume.md', 'personal-projects.md'):
        path = REPO / 'user' / name
        if path.exists():
            parts.append(f'--- {name} ---\n{path.read_text()}')
    return '\n\n'.join(parts)


def build_prompt(batch, record, profile):
    lines = []
    for i, p in enumerate(batch, 1):
        desc = re.sub(r'\s+', ' ', (p.get('description') or ''))[:DESC_CHARS]
        loc = p.get('location') or 'unstated'
        salary = p.get('salary') or 'not disclosed'
        lines.append(f"{i}. {p.get('title')} | {p.get('company')} | {loc} | {salary}\n"
                     f"   posting: {desc or '(NO DESCRIPTION AVAILABLE)'}")
    return PROMPT_HEAD.format(record=record, profile=profile, postings='\n\n'.join(lines))


def parse_reply(text, n):
    """Pull the JSON array out of a reply that may be fenced or chatty."""
    if not text:
        return None
    m = re.search(r'\[.*\]', text, re.S)
    if not m:
        return None
    try:
        arr = json.loads(m.group(0))
    except json.JSONDecodeError:
        return None
    if not isinstance(arr, list):
        return None
    out = {}
    for item in arr:
        if not isinstance(item, dict):
            continue
        try:
            idx = int(item['n'])
        except (KeyError, TypeError, ValueError):
            continue
        if 1 <= idx <= n:
            out[idx] = item
    return out


def run_batch(batch, record, profile, model, timeout):
    prompt = build_prompt(batch, record, profile)
    try:
        proc = subprocess.run(
            ['claude', '-p', '--model', model, '--output-format', 'json'],
            input=prompt, capture_output=True, text=True, timeout=timeout)
    except FileNotFoundError:
        return None, 0.0, 'claude CLI not found on PATH'
    except subprocess.TimeoutExpired:
        return None, 0.0, f'timed out after {timeout}s'
    if proc.returncode != 0:
        return None, 0.0, (proc.stderr or 'non-zero exit').strip()[:200]
    try:
        env = json.loads(proc.stdout)
        text, cost = env.get('result', ''), float(env.get('total_cost_usd') or 0)
    except (json.JSONDecodeError, ValueError):
        text, cost = proc.stdout, 0.0
    verdicts = parse_reply(text, len(batch))
    if verdicts is None:
        return None, cost, 'could not parse a JSON array from the reply'
    return verdicts, cost, None


def post_fit_report(role, v):
    """POST one fit report. Returns (id_or_None, error_or_None)."""
    body = {
        'company': role.get('company') or 'unknown',
        'role_title': role.get('title') or 'unknown',
        'location': role.get('location') or '',
        'level': v.get('level') or 'unclear',
        'posting_url': role.get('url') or '',
        'verdict': v.get('verdict'),
        'verdict_summary': v.get('verdict_summary') or '',
        'why_apply': f"<p>{v.get('verdict_summary') or ''}</p>",
        'match_signals': v.get('match') or [],
        'gap_signals': v.get('gap') or [],
        'flag_signals': v.get('flag') or [],
    }
    cmd = ['curl', '-s', '-X', 'POST', f'{JOBHUB}/api/fit-reports',
           '-H', 'Content-Type: application/json']
    if TOKEN:
        cmd += ['-H', f'Authorization: Bearer {TOKEN}']
    cmd += ['-d', json.dumps(body)]
    try:
        proc = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
    except subprocess.TimeoutExpired:
        return None, 'POST timed out'
    try:
        return json.loads(proc.stdout).get('id'), None
    except json.JSONDecodeError:
        return None, (proc.stdout or 'empty response').strip()[:160]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('roles', help='JSON list of postings, or {"candidates": [...]}')
    ap.add_argument('--out', required=True)
    ap.add_argument('--batch', type=int, default=BATCH)
    ap.add_argument('--model', default='sonnet',
                    help='a fit judgement is not a keep/drop call; do not drop this to haiku '
                         'without measuring what it costs in accuracy')
    ap.add_argument('--timeout', type=int, default=300)
    ap.add_argument('--limit', type=int)
    ap.add_argument('--no-post', action='store_true',
                    help='evaluate and write the file, but do not POST to the dashboard')
    a = ap.parse_args()

    raw = json.loads(Path(a.roles).read_text())
    roles = raw['candidates'] if isinstance(raw, dict) else raw
    if a.limit:
        roles = roles[:a.limit]

    record = build_record()
    profile_path = REPO / 'user' / 'screen-profile.md'
    profile = profile_path.read_text() if profile_path.exists() else '(no screen profile)'

    results, cost, failures = [], 0.0, []
    batches = [roles[i:i + a.batch] for i in range(0, len(roles), a.batch)]
    print(f'{len(roles)} roles in {len(batches)} batches of {a.batch}, model={a.model}', flush=True)

    for bi, batch in enumerate(batches, 1):
        verdicts, c, err = run_batch(batch, record, profile, a.model, a.timeout)
        cost += c
        if err:
            # Named and counted, never silently dropped -- a role missing from the
            # ranking because a batch failed looks identical to one that scored
            # badly, and that is the difference between a queue and a guess.
            for role in batch:
                failures.append({'url': role.get('url'), 'company': role.get('company'),
                                 'title': role.get('title'), 'error': err})
            print(f'  batch {bi}/{len(batches)}: FAILED — {err}', flush=True)
            continue
        for i, role in enumerate(batch, 1):
            v = verdicts.get(i)
            if not v or v.get('verdict') not in VERDICTS:
                failures.append({'url': role.get('url'), 'company': role.get('company'),
                                 'title': role.get('title'), 'error': 'no usable verdict returned'})
                continue
            results.append({**{k: role.get(k) for k in
                               ('url', 'company', 'title', 'location', 'salary', 'tier')},
                            **v})
        done = len(results)
        print(f'  batch {bi}/{len(batches)}: {done} evaluated, {len(failures)} failed', flush=True)

    if not a.no_post:
        posted = 0
        for r in results:
            fid, err = post_fit_report(r, r)
            if fid:
                r['fit_report_id'] = fid
                posted += 1
            else:
                r['post_error'] = err
            time.sleep(0.05)
        print(f'posted {posted}/{len(results)} fit reports to {JOBHUB}', flush=True)

    order = {v: i for i, v in enumerate(VERDICTS)}
    results.sort(key=lambda r: (order.get(r.get('verdict'), 9), -len(r.get('match') or [])))
    Path(a.out).write_text(json.dumps(
        {'evaluated': len(results), 'failed': len(failures), 'cost': round(cost, 4),
         'results': results, 'failures': failures}, indent=1))

    counts = {v: sum(1 for r in results if r.get('verdict') == v) for v in VERDICTS}
    print(f'\nevaluated {len(results)}, failed {len(failures)}')
    print('  ' + '  '.join(f'{v}: {counts[v]}' for v in VERDICTS))
    print(f'wrote {a.out}')
    return 1 if failures and not results else 0


if __name__ == '__main__':
    sys.exit(main())
