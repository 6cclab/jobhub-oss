#!/usr/bin/env python3
"""Stage 2: score candidates with a local Ollama model. Zero token cost.

    scan.py  ->  triage.py  ->  appeal.py  ->  POST to JobHub

Reads `user/screen-profile.md` as the system prompt and asks a local model to
keep or drop each posting. Hard deal-breakers are applied afterwards in Python
(`vetoes.py`) as an override the model cannot argue with.

The screen profile carries a deliberate **keep-when-uncertain** bias: a wrongly
kept role costs one line of human review, a wrongly dropped one is invisible.
Drops that came from the model's judgement (not a veto) are re-checked by
`appeal.py`.

Usage:
    python3 scripts/triage.py candidates.json --out triaged.json
    python3 scripts/triage.py candidates.json --limit 20      # sample first
    python3 scripts/triage.py candidates.json --model qwen3:14b

Requires OLLAMA_HOST (e.g. http://192.168.3.168:11434).

Throughput, measured 2026-08-21 against qwen3.8 with real 2200-char postings:
**~7.3s per posting, and it does not improve with more client workers** -- 40
postings took 4.9 min at both 6 and 12 concurrent, so the server is serializing.
That makes the first scan (~845 postings) a ~100 minute overnight job, and every
scan after it a few minutes, because only unseen postings reach this stage.
To go faster, raise OLLAMA_NUM_PARALLEL on the Ollama host.
"""
import argparse
import json
import os
import sys
import time
import urllib.error
import urllib.request
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import vetoes  # noqa: E402

REPO = Path(__file__).resolve().parent.parent
PROFILE = REPO / 'user' / 'screen-profile.md'

OLLAMA = os.environ.get('OLLAMA_HOST', 'http://localhost:11434').rstrip('/')
DEFAULT_MODEL = 'qwen3.8:latest'

# `tier` deliberately has no "drop" member. An earlier schema allowed it and the
# model returned keep=true with tier="drop" on the same object; removing the
# value makes the contradiction unrepresentable rather than merely discouraged.
SCHEMA = {
    'type': 'object',
    'properties': {
        'keep': {'type': 'boolean'},
        'tier': {'type': 'string', 'enum': ['strong', 'good']},
        'reason': {'type': 'string'},
        'level_tag': {'type': 'string', 'enum': ['senior', 'staff', 'other']},
        'domain_tag': {'type': 'string',
                       'enum': ['platform', 'devplatform', 'fullstack', 'product',
                                'infra', 'observability', 'identity', 'other']},
    },
    'required': ['keep', 'tier', 'reason', 'level_tag', 'domain_tag'],
}

# Descriptions run to tens of thousands of characters. The screen needs the
# shape of the role, not the benefits section.
DESC_CHARS = 2200


def posting_prompt(p):
    desc = (p.get('description') or '')[:DESC_CHARS]
    loc = p.get('location') or 'unspecified'
    if p.get('is_remote'):
        loc += ' [remote]'
    return (f"Company: {p.get('company')}\n"
            f"Title: {p.get('title')}\n"
            f"Location: {loc}\n\n"
            f"{desc if desc else '(no description available from this ATS)'}\n\n"
            "Keep or drop for this candidate? reason = one sentence, under 20 words.")


def score_one(args):
    p, profile, model, retries = args
    body = json.dumps({
        'model': model, 'stream': False, 'think': False, 'keep_alive': '30m',
        'format': SCHEMA,
        'options': {'temperature': 0},
        'messages': [{'role': 'system', 'content': profile},
                     {'role': 'user', 'content': posting_prompt(p)}],
    }).encode()

    last = None
    for attempt in range(retries):
        try:
            req = urllib.request.Request(OLLAMA + '/api/chat', data=body,
                                         headers={'Content-Type': 'application/json'})
            # Generous: a cold model load on a 27B is ~2 min before any tokens.
            r = json.load(urllib.request.urlopen(req, timeout=300))
            verdict = json.loads(r['message']['content'])
            out = dict(p)
            out.update({k: verdict.get(k) for k in
                        ('keep', 'tier', 'reason', 'level_tag', 'domain_tag')})
            return out
        except Exception as e:
            last = f'{type(e).__name__}: {str(e)[:80]}'
            time.sleep(1.5 * (attempt + 1))

    # A model failure is not a drop. Keep it and flag it, so a broken Ollama
    # box degrades into "review these by hand" rather than silently emptying
    # the funnel.
    out = dict(p)
    out.update({'keep': True, 'tier': 'good', 'level_tag': 'other',
                'domain_tag': 'other', 'reason': f'triage failed, kept for review ({last})',
                'triage_error': last})
    return out


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('candidates')
    ap.add_argument('--out', default='triaged.json')
    ap.add_argument('--model', default=DEFAULT_MODEL)
    # Measured 2026-08-21: 40 postings took 4.9 min at BOTH 6 and 12 workers.
    # The Ollama server serializes requests (OLLAMA_NUM_PARALLEL), so client-side
    # concurrency past ~4 buys nothing. Raise it on the server, not here.
    ap.add_argument('--workers', type=int, default=4)
    ap.add_argument('--limit', type=int)
    ap.add_argument('--retries', type=int, default=2)
    a = ap.parse_args()

    if not PROFILE.exists():
        print(f'FATAL: {PROFILE} missing -- triage has no screening criteria', file=sys.stderr)
        return 1
    profile = PROFILE.read_text()

    data = json.loads(Path(a.candidates).read_text())
    cands = data['candidates'][:a.limit] if a.limit else data['candidates']
    if not cands:
        print('no candidates to triage')
        Path(a.out).write_text(json.dumps({**data, 'kept': [], 'dropped': []}, indent=1))
        return 0

    # Warm the model once rather than paying the cold load inside a worker,
    # where it would burn the timeout budget of whichever posting drew it.
    print(f'warming {a.model} on {OLLAMA} ...', end='', flush=True)
    t0 = time.time()
    try:
        warm = json.dumps({'model': a.model, 'stream': False, 'keep_alive': '30m',
                           'messages': [{'role': 'user', 'content': 'ok'}]}).encode()
        urllib.request.urlopen(urllib.request.Request(
            OLLAMA + '/api/chat', data=warm,
            headers={'Content-Type': 'application/json'}), timeout=600)
        print(f' {time.time()-t0:.0f}s')
    except Exception as e:
        print(f'\nFATAL: Ollama unreachable at {OLLAMA} -- {type(e).__name__}: {e}',
              file=sys.stderr)
        return 1

    print(f'triaging {len(cands)} postings, {a.workers} concurrent')
    t0 = time.time()
    scored = []
    with ThreadPoolExecutor(max_workers=a.workers) as ex:
        for i, r in enumerate(ex.map(score_one,
                                     [(p, profile, a.model, a.retries) for p in cands]), 1):
            scored.append(r)
            if i % 25 == 0 or i == len(cands):
                rate = i / max(time.time() - t0, 0.01)
                print(f'  {i}/{len(cands)}  ({rate:.1f}/s, {(len(cands)-i)/max(rate,0.01)/60:.0f} min left)')

    # Vetoes run last and win. `veto` also marks the drop as certain, which is
    # what tells appeal.py not to spend money re-checking it.
    kept, dropped, vetoed_keeps = [], [], 0
    for r in scored:
        veto = vetoes.check(r)
        if veto:
            if r.get('keep'):
                vetoed_keeps += 1
            r['keep'] = False
            r['veto'] = veto
            r['reason'] = veto
            dropped.append(r)
        elif r.get('keep'):
            kept.append(r)
        else:
            r['veto'] = None
            dropped.append(r)

    errs = sum(1 for r in scored if r.get('triage_error'))
    appealable = sum(1 for r in dropped if not r.get('veto'))
    print(f'\nkept {len(kept)}  dropped {len(dropped)}'
          f'  (vetoed {len(dropped)-appealable}, appealable {appealable})')
    print(f'vetoes overrode {vetoed_keeps} model keeps')
    if errs:
        print(f'WARNING: {errs} postings failed triage and were kept for manual review',
              file=sys.stderr)
    print(f'elapsed {(time.time()-t0)/60:.1f} min')

    Path(a.out).write_text(json.dumps(
        {**{k: v for k, v in data.items() if k != 'candidates'},
         'model': a.model, 'kept': kept, 'dropped': dropped}, indent=1))
    print(f'wrote {a.out}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
