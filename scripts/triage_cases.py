#!/usr/bin/env python3
"""Regression test for the triage screen. Run it after ANY edit to screen-profile.md.

This exists because it caught a real bug. An earlier screen profile listed
"AI roles" as an exclusion without qualification, and the model dropped the
Hims & Hers *Senior SWE, Developer Platform* req -- a role Andre actually
applied to on 2026-08-17 -- because the words "AI-assisted" appeared in it.
`preferences.md` is explicit that the line is the team's charter, not the
vocabulary in the posting. The `charter_*` cases below pin that down.

The screen profile is a prompt, so it has no type system and no compiler. This
file is the only thing standing between an edit and a silently narrower funnel.

Usage:
    python3 scripts/triage_cases.py
    python3 scripts/triage_cases.py --model qwen3:14b
"""
import argparse
import json
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import triage  # noqa: E402
import vetoes  # noqa: E402

# (name, expect_keep, posting). expect_keep is what a correct screen produces.
CASES = [
    ('charter_dx_with_ai', True, {
        'company': 'Hims & Hers', 'title': 'Senior Software Engineer, Developer Platform',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Build internal tooling and developer platform capabilities including '
                       'AI-assisted development workflows. TypeScript, NestJS, Kubernetes, AWS. '
                       'Own CI/CD and the internal SDK surface. 5+ years experience.'}),
    ('charter_agentic_platform', True, {
        'company': 'Comcast', 'title': 'Senior Platform Engineer, Agentic Systems',
        'location': 'Philadelphia, PA', 'is_remote': False,
        'description': 'Own the internal developer platform. Go, Kubernetes, CI/CD pipelines. '
                       'Some agent tooling for internal engineering workflows. 5+ years.'}),
    ('charter_true_ai_org', False, {
        'company': 'Scale AI', 'title': 'Senior Machine Learning Engineer, Applied AI',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Own model training and inference infrastructure. Design evaluation '
                       'harnesses for foundation models. PyTorch, CUDA. 6+ years in ML.'}),
    ('charter_ai_platform_team', False, {
        'company': 'Rippling', 'title': 'Senior Engineer, AI Platform',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Build model serving and evaluation infrastructure for LLM-powered '
                       'features. Own inference latency and model rollout. 5+ years.'}),
    ('geo_nyc_onsite_ok', True, {
        'company': 'Ramp', 'title': 'Senior Product Engineer',
        'location': 'New York, NY', 'is_remote': False,
        'description': 'React, TypeScript, Node, Postgres. Own a product surface end to end '
                       'from API through UI. 3 days per week in our NYC office. 5+ years.'}),
    ('geo_philly_onsite_ok', True, {
        'company': 'Vanguard', 'title': 'Senior Software Engineer, Developer Experience',
        'location': 'Malvern, PA', 'is_remote': False,
        'description': 'Internal tooling, CI/CD, Go and Java services. 3 days onsite. 6+ years.'}),
    ('geo_sf_relocation_out', False, {
        'company': 'Figma', 'title': 'Senior Software Engineer, Design Systems',
        'location': 'San Francisco, CA', 'is_remote': False,
        'description': 'React and TypeScript design tooling. 4 days per week onsite in SF. '
                       'No remote option. 5+ years experience.'}),
    ('fullstack_remote_keep', True, {
        'company': 'Roadie', 'title': 'Senior Full Stack Engineer',
        'location': 'Remote, US', 'is_remote': True,
        'description': 'Node, TypeScript, React, Postgres. Own the API, the data model and the '
                       'UI. Logistics domain. 5+ years experience.'}),
    # --- veto cases: these must be killed by vetoes.py, not by the model ---
    ('veto_defense', False, {
        'company': 'Anduril Industries', 'title': 'Senior Software Engineer',
        'location': 'Costa Mesa, CA', 'is_remote': False,
        'description': 'Autonomous defense systems. C++ and Rust. 5+ years.'}),
    ('veto_ten_years', False, {
        'company': 'Affirm', 'title': 'Senior Software Engineer, Web Infrastructure',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Requires 10+ years of industry experience plus prior project and '
                       'people management. Own frontend platform direction.'}),
    # Added 2026-08-21 with the Staff hard filter. The three cases above were
    # written with Staff titles when Staff was in scope; the new veto fired
    # before their own rule could, so geo_philly_onsite_ok failed outright and
    # charter_true_ai_org and veto_ten_years kept passing without exercising
    # the charter rule or the years check at all. They are Senior now, and the
    # Staff filter gets its own cases here instead of leaking into theirs.
    ('veto_staff_title', False, {
        'company': 'Airbnb', 'title': 'Staff Software Engineer, Build (Bazel)',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Own Bazel build infrastructure and developer experience. '
                       'Monorepo tooling, remote caching, CI. 7+ years.'}),
    ('veto_staff_strong_fit', False, {
        'company': 'Anthropic', 'title': 'Staff Software Engineer, Claude Code',
        'location': 'New York City', 'is_remote': False,
        'description': 'Build developer tooling and agentic coding workflows in '
                       'TypeScript and Node. Product and DX ownership. 7+ years.'}),
    ('multiband_senior_or_staff_keep', True, {
        'company': 'Linear', 'title': 'Senior / Staff Fullstack Engineer',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Own product surfaces end to end in TypeScript and React '
                       'over a Node API. Levelled during the process. 6+ years.'}),
    ('veto_management', False, {
        'company': 'Datadog', 'title': 'Engineering Manager, Platform',
        'location': 'New York, NY', 'is_remote': False,
        'description': 'Lead a team of 8 engineers. No IC coding expected. 5+ years managing.'}),
    ('veto_closed_company', False, {
        'company': 'Seeq', 'title': 'Staff Software Engineer',
        'location': 'Remote US', 'is_remote': True,
        'description': 'Full stack, TypeScript and Python. Industrial analytics. 5+ years.'}),
]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument('--model', default=triage.DEFAULT_MODEL)
    ap.add_argument('--verbose', action='store_true')
    a = ap.parse_args()

    if not triage.PROFILE.exists():
        print(f'FATAL: {triage.PROFILE} missing', file=sys.stderr)
        return 1
    profile = triage.PROFILE.read_text()

    print(f'model {a.model}  |  {len(CASES)} cases\n')
    from concurrent.futures import ThreadPoolExecutor
    with ThreadPoolExecutor(max_workers=4) as ex:
        scored = list(ex.map(triage.score_one,
                             [(p, profile, a.model, 2) for _, _, p in CASES]))

    failures = []
    for (name, expect, _), r in zip(CASES, scored):
        veto = vetoes.check(r)
        keep = bool(r.get('keep')) and not veto
        src = 'veto' if veto else 'model'
        ok = keep == expect
        if not ok:
            failures.append(name)
        # A veto case that the model would have kept anyway is still a pass --
        # the veto is precisely the safety net. But say which fired.
        flag = 'ok  ' if ok else 'FAIL'
        print(f'{flag} {name:26} keep={str(keep):5} via {src:5} '
              f'{(veto or r.get("reason") or "")[:58]}')
        if a.verbose and not ok:
            print(f'       raw: {json.dumps({k: r.get(k) for k in ("keep","tier","reason")})}')

    print(f'\n{len(CASES)-len(failures)}/{len(CASES)} passed')
    if failures:
        print('FAILED: ' + ', '.join(failures), file=sys.stderr)
        print('\nThe screen profile changed behaviour. Either fix '
              'user/screen-profile.md or, if the new behaviour is intended, '
              'update the expectation in this file and say why.', file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
