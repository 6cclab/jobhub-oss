#!/usr/bin/env python3
"""Regression cases for the rejection carve-out in scan.py.

`status: rejected` hides a company from every future scan, company-wide and
permanently. `user/rejections.json` carves out the administrative closes. The
whole mechanism turns on one safety property:

    DEFAULT IS EXCLUDE.

A missing file, a malformed file, a missing key, an unknown company -- every one
of those must leave the company omitted. Only a deliberate `exclude: false` may
widen the funnel, and it has to show up in the digest when it does.

That direction matters because the two failures are not symmetric. A wrongly
omitted company is invisible: nothing in a digest can show roles that stopped
appearing. A wrongly kept company costs one line of review.

Run: python3 scripts/rejection_cases.py
"""
import importlib.util
import json
import sys
import tempfile
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

spec = importlib.util.spec_from_file_location('scan', REPO / 'scripts' / 'scan.py')
scan = importlib.util.module_from_spec(spec)
spec.loader.exec_module(scan)


def carveouts_from(payload, raw=None):
    """Run load_rejection_carveouts() against a temp user/rejections.json."""
    with tempfile.TemporaryDirectory() as td:
        root = Path(td)
        (root / 'user').mkdir()
        target = root / 'user' / 'rejections.json'
        if raw is not None:
            target.write_text(raw)
        elif payload is not None:
            target.write_text(json.dumps(payload))
        orig = scan.REPO
        scan.REPO = root
        try:
            return scan.load_rejection_carveouts()
        finally:
            scan.REPO = orig


CASES = []


def case(name):
    def deco(fn):
        CASES.append((name, fn))
        return fn
    return deco


@case('missing_file_excludes_everything')
def _():
    got = carveouts_from(None)
    assert got == {}, got


@case('malformed_json_excludes_everything')
def _():
    got = carveouts_from(None, raw='{not json at all')
    assert got == {}, got


@case('exclude_false_is_a_carveout')
def _():
    got = carveouts_from({'rejections': [
        {'company': "The Farmer's Dog", 'exclude': False, 'reason': 'role filled'}]})
    assert got == {"the farmer's dog": (False, 'role filled')}, got


@case('exclude_true_stays_excluded')
def _():
    got = carveouts_from({'rejections': [
        {'company': 'Seeq', 'exclude': True, 'reason': 'assessed and passed'}]})
    assert got['seeq'][0] is True, got


@case('missing_exclude_key_defaults_to_exclude')
def _():
    # The safety property. An entry written without the key must NOT widen.
    got = carveouts_from({'rejections': [{'company': 'Acme', 'reason': 'unclear'}]})
    assert got['acme'][0] is True, got


@case('missing_reason_is_labelled_not_blank')
def _():
    got = carveouts_from({'rejections': [{'company': 'Acme', 'exclude': False}]})
    assert got['acme'] == (False, 'unstated'), got


@case('company_name_is_normalised')
def _():
    got = carveouts_from({'rejections': [
        {'company': '  MiXeD Case  ', 'exclude': False, 'reason': 'req closed'}]})
    assert 'mixed case' in got, got


@case('blank_company_is_dropped')
def _():
    got = carveouts_from({'rejections': [{'company': '   ', 'exclude': False}]})
    assert got == {}, got


@case('empty_rejections_list_is_fine')
def _():
    assert carveouts_from({'rejections': []}) == {}


@case('carveout_only_applies_to_actually_rejected_companies')
def _():
    # Subtraction logic from load_rejected_companies(). A carve-out for a company
    # that never rejected anything must not appear as a kept-visible exception,
    # or the digest fills with noise about companies that were never omitted.
    rejected = {'affirm', 'reddit', "the farmer's dog"}
    carve = {"the farmer's dog": (False, 'role filled'),
             'seeq': (True, 'assessed and passed'),
             'never-applied-here': (False, 'req closed')}
    kept = [(n, r) for n, (ex, r) in carve.items() if not ex and n in rejected]
    remaining = rejected - {n for n, _ in kept}
    assert kept == [("the farmer's dog", 'role filled')], kept
    assert remaining == {'affirm', 'reddit'}, remaining


@case('live_rejections_json_parses_and_carves_the_farmers_dog')
def _():
    # The real file, not a fixture -- catches a hand-edit that breaks the schema.
    got = scan.load_rejection_carveouts()
    assert got, 'user/rejections.json produced no entries'
    assert got.get("the farmer's dog", (True, ''))[0] is False, got.get("the farmer's dog")
    for name in ('seeq', 'headway', 'affirm', 'reddit'):
        assert got.get(name, (False, ''))[0] is True, (name, got.get(name))


def main():
    failed = 0
    for name, fn in CASES:
        try:
            fn()
            print(f'ok   {name}')
        except AssertionError as e:
            failed += 1
            print(f'FAIL {name}: {e}')
        except Exception as e:
            failed += 1
            print(f'ERROR {name}: {type(e).__name__}: {e}')
    total = len(CASES)
    print(f'\n{total - failed}/{total} passed')
    return 1 if failed else 0


if __name__ == '__main__':
    sys.exit(main())
