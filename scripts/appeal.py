#!/usr/bin/env python3
"""Stage 3: give every judgement drop a second opinion from Haiku.

    scan.py  ->  triage.py  ->  appeal.py  ->  POST to JobHub

The errors here are not symmetric. A wrongly KEPT role costs one line of review
on a dashboard. A wrongly DROPPED role is invisible -- it never reaches the
queue and nobody ever learns it existed. The local model is a quantized 27B and
it does make mistakes, so a stronger model re-checks every drop it made.

Two rules keep this cheap:

1. **Vetoed drops are skipped.** A veto from `vetoes.py` is a rule, not a
   judgement -- re-checking "Anduril is a defense contractor" spends money to
   confirm something that was never uncertain.

2. **Appeals are batched, 25 per call.** Measured 2026-08-20: one posting costs
   $0.026 and eight postings cost $0.027, because the CLI's system prompt
   dominates. Unbatched this runs $80-150/month; batched it is ~$0.16/day.
   Batching is load-bearing, not an optimization.

Auth comes from the `claude` CLI's existing credentials -- no API key needed.

Usage:
    python3 scripts/appeal.py triaged.json --out appealed.json
    python3 scripts/appeal.py triaged.json --skip     # pass through untouched
"""
import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
PROFILE = REPO / 'user' / 'screen-profile.md'

# Lowered from 25 on 2026-08-21, on measurement rather than judgement.
#
# 25 was chosen purely on cost, back when cost was a constraint. It no longer is
# (preferences.md), so the batch size was tested on one identical 84-posting drop
# list, all three arms appealing the same roles:
#
#   batch=25 -> 4 rescued (4.8%),  $0.28
#   batch=5  -> 13 rescued (15.5%), $0.81
#   batch=1  -> 25 rescued (29.8%), $3.01
#
# Roles that only the smaller batches recovered include Cloudflare, GitLab,
# Grafana, Confluent, Sentry, Plaid, Coinbase and MongoDB — and several were
# genuine triage errors, not marginal calls: two were dropped on misread
# `[remote]` locations, and one on confusing a company's AI positioning with an
# AI team charter.
#
# 5 rather than 1 because the rescue count is NOT a clean quality metric: some of
# what batch=1 recovers comes from the prompt's "when uncertain, KEEP" rule
# firing more often on thinner evidence, not from better reading. 5 takes most of
# the real recall at a quarter of the leniency. Raising this number back up will
# silently drop roles — that is what the numbers above measure.
BATCH = 5
DESC_CHARS = 700  # enough to judge charter and level; the rest is boilerplate

PROMPT_HEAD = """You are an appeal reviewer for a job search. A small local model DROPPED the postings below. Your job is to catch the ones it got WRONG.

Screening criteria:
{profile}

The local model is known to fail in two specific ways, so look hardest for these:
1. Dropping a platform/DX/product role because the words "AI" or "ML" appear, when the team's CHARTER is not AI.
2. Dropping an in-scope location (NYC, Philadelphia, New Jersey, Delaware, remote-US) because the posting says onsite or hybrid. In-office days are NOT a filter.

Uphold the drop when it is right. Overturn it when it is wrong. Be decisive.

POSTINGS:
{postings}

Reply with ONLY a minified JSON array, one object per numbered posting, no prose and no code fence:
[{{"n":1,"drop_upheld":true,"reason":"under 15 words"}}]"""


def build_prompt(batch, profile):
    lines = []
    for i, p in enumerate(batch, 1):
        loc = p.get('location') or 'unspecified'
        if p.get('is_remote'):
            loc += ' [remote]'
        desc = re.sub(r'\s+', ' ', (p.get('description') or ''))[:DESC_CHARS]
        lines.append(f"{i}. {p.get('title')} | {p.get('company')} | {loc}\n"
                     f"   dropped because: {p.get('reason') or 'unstated'}\n"
                     f"   posting: {desc or '(no description)'}")
    return PROMPT_HEAD.format(profile=profile, postings='\n\n'.join(lines))


def parse_reply(text, n):
    """Pull the JSON array out of a CLI reply that may be fenced or chatty."""
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
        try:
            idx = int(item['n'])
        except (KeyError, TypeError, ValueError):
            continue
        if 1 <= idx <= n:
            out[idx] = item
    return out


def run_batch(batch, profile, model, timeout):
    """Returns (verdicts_by_index, cost_usd, error)."""
    prompt = build_prompt(batch, profile)
    try:
        proc = subprocess.run(
            ['claude', '-p', '--model', model, '--output-format', 'json'],
            input=prompt, capture_output=True, text=True, timeout=timeout)
    except FileNotFoundError:
        return None, 0.0, 'claude CLI not found on PATH'
    except subprocess.TimeoutExpired:
        return None, 0.0, f'timed out after {timeout}s'

    if proc.returncode != 0:
        return None, 0.0, f'exit {proc.returncode}: {(proc.stderr or "")[:160]}'
    try:
        env = json.loads(proc.stdout)
    except json.JSONDecodeError:
        return None, 0.0, f'unparseable CLI envelope: {proc.stdout[:160]}'

    cost = float(env.get('total_cost_usd') or 0.0)
    verdicts = parse_reply(env.get('result'), len(batch))
    if verdicts is None:
        return None, cost, f'no JSON array in reply: {str(env.get("result"))[:160]}'
    return verdicts, cost, None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('triaged')
    ap.add_argument('--out', default='appealed.json')
    ap.add_argument('--model', default='haiku')
    ap.add_argument('--batch', type=int, default=BATCH)
    ap.add_argument('--timeout', type=int, default=240)
    ap.add_argument('--limit', type=int, help='appeal only the first N drops (testing)')
    ap.add_argument('--skip', action='store_true', help='pass through with no appeals')
    a = ap.parse_args()

    data = json.loads(Path(a.triaged).read_text())
    kept, dropped = data.get('kept', []), data.get('dropped', [])

    if a.skip:
        Path(a.out).write_text(json.dumps(
            {**data, 'appeal': {'skipped': True}}, indent=1))
        print('--skip: no appeals run')
        return 0

    # A veto is certain. Paying a cloud model to re-confirm a rule is waste.
    appealable = [p for p in dropped if not p.get('veto')]
    vetoed = len(dropped) - len(appealable)
    if a.limit:
        appealable = appealable[:a.limit]

    if not appealable:
        print(f'nothing to appeal ({vetoed} vetoed drops skipped)')
        Path(a.out).write_text(json.dumps(
            {**data, 'appeal': {'appealed': 0, 'overturned': 0, 'cost_usd': 0.0}}, indent=1))
        return 0

    profile = PROFILE.read_text() if PROFILE.exists() else '(screen profile missing)'
    batches = [appealable[i:i + a.batch] for i in range(0, len(appealable), a.batch)]
    print(f'appealing {len(appealable)} drops in {len(batches)} batch(es) of {a.batch}'
          f'  ({vetoed} vetoed drops skipped)')

    overturned, cost, errors = [], 0.0, []
    for bi, batch in enumerate(batches, 1):
        verdicts, c, err = run_batch(batch, profile, a.model, a.timeout)
        cost += c
        if err:
            errors.append(f'batch {bi}: {err}')
            print(f'  batch {bi}/{len(batches)}: FAILED -- {err}', file=sys.stderr)
            continue
        n_over = 0
        for i, p in enumerate(batch, 1):
            v = verdicts.get(i)
            if v is None:
                # Missing verdict is not an upheld drop. Say so rather than
                # letting a truncated reply quietly bury roles.
                errors.append(f'batch {bi}: no verdict for #{i} ({p.get("title")})')
                continue
            if not v.get('drop_upheld', True):
                p['keep'] = True
                p['appealed'] = True
                p['appeal_reason'] = v.get('reason', '')
                p['tier'] = p.get('tier') or 'good'
                overturned.append(p)
                n_over += 1
        print(f'  batch {bi}/{len(batches)}: {n_over} overturned  (${cost:.4f} so far)')

    ov_urls = {p['url'] for p in overturned}
    new_kept = kept + overturned
    new_dropped = [p for p in dropped if p.get('url') not in ov_urls]

    print(f'\noverturned {len(overturned)} of {len(appealable)} appealable drops')
    print(f'kept {len(kept)} -> {len(new_kept)}')
    print(f'cost ${cost:.4f}' + (f' ({cost/len(appealable):.5f}/posting)' if appealable else ''))
    if errors:
        print(f'\n{len(errors)} appeal error(s) -- these drops were NOT reviewed:', file=sys.stderr)
        for e in errors[:10]:
            print(f'  {e}', file=sys.stderr)

    Path(a.out).write_text(json.dumps({
        **data, 'kept': new_kept, 'dropped': new_dropped,
        'appeal': {'appealed': len(appealable), 'overturned': len(overturned),
                   'vetoed_skipped': vetoed, 'cost_usd': round(cost, 4),
                   'model': a.model, 'errors': errors},
    }, indent=1))
    print(f'wrote {a.out}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
