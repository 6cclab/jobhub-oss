#!/usr/bin/env python3
"""Regression tests for the application record format.

Run after ANY edit to scripts/application_record.py. The record files under
user/applications/ are the authoritative job-search record and are NOT in git
(.gitignore:1), so a parser that silently drops a field loses data with no way
to recover it. Every case below pins a failure mode that would do that.

Fixtures use placeholder companies on purpose: this file is published to the
public mirror (.github/workflows/mirror.yaml).

Usage:
    python3 scripts/application_record_cases.py
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import application_record as ar  # noqa: E402

GOOD = """---
company: Acme
role: Software Engineer, Backend (Mid/Senior)
status: phone_screen
source: recruiter-inbound
submitted: false
applied: 2026-08-24
resume: acme/software-engineer-backend/resume.pdf
packet: user/tailored/acme/software-engineer-backend
server_id: 00000000-0000-0000-0000-000000000000
fit: strong
events:
  - date: 2026-08-24
    to: inbound
    note: recruiter outbound
  - date: 2026-08-26
    to: phone_screen
    note: screen passed; behavioral then technical
---

Body text. **Bold** survives.
"""


def case_parses_scalars():
    r = ar.parse(GOOD)
    assert r['company'] == 'Acme', r['company']
    assert r['status'] == 'phone_screen'
    assert r['submitted'] is False, r['submitted']
    assert r['applied'] == '2026-08-24'


def case_role_keeps_its_colon_free_commas():
    r = ar.parse(GOOD)
    assert r['role'] == 'Software Engineer, Backend (Mid/Senior)', r['role']


def case_parses_events_in_order():
    r = ar.parse(GOOD)
    assert len(r['events']) == 2, r['events']
    assert r['events'][0] == {'date': '2026-08-24', 'to': 'inbound',
                              'note': 'recruiter outbound'}
    assert r['events'][1]['to'] == 'phone_screen'


def case_note_may_contain_a_semicolon_and_colon():
    text = GOOD.replace('note: screen passed; behavioral then technical',
                        'note: passed; next: behavioral, then "technical"')
    r = ar.parse(text)
    assert r['events'][1]['note'] == 'passed; next: behavioral, then "technical"', \
        r['events'][1]['note']


def case_body_is_preserved_verbatim():
    r = ar.parse(GOOD)
    assert r['body'] == 'Body text. **Bold** survives.\n', repr(r['body'])


def case_unknown_key_is_an_error():
    text = GOOD.replace('fit: strong', 'fit: strong\nsalary: 200000')
    try:
        ar.parse(text)
    except ar.RecordError as e:
        assert 'salary' in str(e), str(e)
        return
    raise AssertionError('unknown key was accepted')


def case_missing_required_key_is_an_error():
    text = GOOD.replace('source: recruiter-inbound\n', '')
    try:
        ar.parse(text)
    except ar.RecordError as e:
        assert 'source' in str(e), str(e)
        return
    raise AssertionError('missing required key was accepted')


def case_bad_status_is_an_error():
    text = GOOD.replace('status: phone_screen', 'status: interviewing')
    try:
        ar.parse(text)
    except ar.RecordError as e:
        assert 'interviewing' in str(e), str(e)
        return
    raise AssertionError('unknown status was accepted')


def case_bad_submitted_is_an_error():
    text = GOOD.replace('submitted: false', 'submitted: nope')
    try:
        ar.parse(text)
    except ar.RecordError:
        return
    raise AssertionError('non-boolean submitted was accepted')


def case_bad_date_is_an_error():
    text = GOOD.replace('applied: 2026-08-24', 'applied: 24/08/2026')
    try:
        ar.parse(text)
    except ar.RecordError:
        return
    raise AssertionError('malformed date was accepted')


def case_empty_required_value_is_an_error():
    text = GOOD.replace('company: Acme', 'company:')
    try:
        ar.parse(text)
    except ar.RecordError as e:
        assert 'company' in str(e), str(e)
        return
    raise AssertionError('empty required value was accepted')


def case_duplicate_key_is_an_error():
    text = GOOD.replace('status: phone_screen',
                        'status: phone_screen\nstatus: onsite')
    try:
        ar.parse(text)
    except ar.RecordError as e:
        assert 'status' in str(e), str(e)
        return
    raise AssertionError('duplicate key was accepted')


def case_missing_frontmatter_is_an_error():
    try:
        ar.parse('no frontmatter here\n')
    except ar.RecordError:
        return
    raise AssertionError('file without frontmatter was accepted')


def case_round_trips():
    r = ar.parse(GOOD)
    again = ar.parse(ar.serialise(r))
    assert again == r, f'{again}\n!=\n{r}'


def main():
    cases = [v for k, v in sorted(globals().items()) if k.startswith('case_')]
    failures = []
    for fn in cases:
        name = fn.__name__[len('case_'):]
        try:
            fn()
            print(f'ok   {name}')
        except Exception as e:  # noqa: BLE001 - a case failure is a test result
            failures.append(name)
            print(f'FAIL {name}: {e}')
    print(f'\n{len(cases) - len(failures)}/{len(cases)} passed')
    if failures:
        print('FAILED: ' + ', '.join(failures), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
