#!/usr/bin/env python3
"""Regression tests for scripts/jobhub_db.py, the local funnel store.

These pin the properties that made the networked server worth replacing, not
just "the inserts work":

  - init is idempotent, so a scan never has to care whether the file exists
  - CHECK constraints reject bad enum values at the store, not at a handler
  - return shapes match what the HTTP API returned, so call sites port by
    swapping a function for a curl and nothing else
  - fit signals can be DELETED, which is the one deliberate improvement over
    the API (jobhub-api.md: signal arrays were create-only, "a wrong one
    cannot be removed")
  - below_floor is gone and stays gone

Every case runs against a temp file. Nothing here touches user/jobhub.db.

Usage:
    python3 scripts/jobhub_db_cases.py
"""
import sqlite3
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import jobhub_db as db  # noqa: E402


def _conn():
    """A fresh store in a temp dir. Caller does not clean up; the OS does."""
    d = tempfile.mkdtemp(prefix='jobhub-cases-')
    return db.connect(Path(d) / 'test.db')


BATCH = {
    'ran_at': '2026-09-11T07:15:00+00:00',
    'board_count': 110,
    'raw_count': 2400,
    'location_filter': 'remote-US + Philadelphia/South Jersey/Delaware + NYC/North NJ',
}

RESULT = {
    'company': 'Acme Systems',
    'role_title': 'Senior Software Engineer',
    'location': 'Remote (US)',
    'is_remote': True,
    'salary_min': 180000,
    'salary_max': 220000,
    'salary_disclosed': True,
    'posting_url': 'https://example.invalid/jobs/1',
    'fit_tier': 'strong',
    'tags': ['devex', 'senior'],
    'level_tag': 'senior',
    'domain_tag': 'devex',
}

FIT = {
    'company': 'Acme Systems',
    'role_title': 'Senior Software Engineer',
    'location': 'Remote (US)',
    'level': 'senior',
    'posting_url': 'https://example.invalid/jobs/1',
    'verdict': 'strong',
    'verdict_summary': 'Direct overlap on developer platform work.',
    'why_apply': '<p>Direct overlap on developer platform work.</p>',
    'match_signals': [
        {'requirement': 'Internal tooling', 'evidence': 'Account Tools rebuild'},
        {'requirement': 'CI/CD', 'evidence': 'Argo Rollouts canary work'},
    ],
    'gap_signals': [{'requirement': 'Vector search', 'evidence': 'No production exposure'}],
    'flag_signals': [],
}


# --- schema / lifecycle -----------------------------------------------------

def case_connect_creates_the_schema():
    c = _conn()
    got = {r[0] for r in c.execute(
        "SELECT name FROM sqlite_master WHERE type='table'")}
    for t in ('companies', 'boards', 'search_batches', 'search_results',
              'fit_reports', 'fit_signals'):
        assert t in got, f'{t} missing from {sorted(got)}'


def case_connect_is_idempotent():
    d = tempfile.mkdtemp(prefix='jobhub-cases-')
    p = Path(d) / 'test.db'
    c1 = db.connect(p)
    db.upsert_company(c1, 'Acme Systems')
    c1.close()
    c2 = db.connect(p)          # must not wipe or raise
    assert len(db.list_companies(c2)) == 1, 'reconnect lost data or re-ran DDL'


def case_wal_is_on():
    c = _conn()
    mode = c.execute('PRAGMA journal_mode').fetchone()[0]
    assert mode.lower() == 'wal', f'journal_mode={mode}'


def case_foreign_keys_are_enforced():
    c = _conn()
    try:
        c.execute("INSERT INTO search_results (id, search_batch_id, company_id,"
                  " role_title, posting_url, fit_tier, created_at)"
                  " VALUES ('x','nope','nope','r','u','strong','t')")
        c.commit()
    except sqlite3.IntegrityError:
        return
    raise AssertionError('orphan row was accepted; PRAGMA foreign_keys is off')


def case_jobhub_db_env_redirects_the_store():
    """Without this, testing a call site writes fake companies into the real funnel."""
    import importlib
    import os
    d = tempfile.mkdtemp(prefix='jobhub-cases-')
    target = Path(d) / 'redirected.db'
    old = os.environ.get('JOBHUB_DB')
    os.environ['JOBHUB_DB'] = str(target)
    try:
        importlib.reload(db)
        assert db.DEFAULT_PATH == target, db.DEFAULT_PATH
        db.connect().close()
        assert target.exists(), 'connect() ignored the override'
    finally:
        if old is None:
            del os.environ['JOBHUB_DB']
        else:
            os.environ['JOBHUB_DB'] = old
        importlib.reload(db)   # leave the module as the other cases expect


def case_below_floor_is_gone():
    c = _conn()
    cols = {r[1] for r in c.execute('PRAGMA table_info(search_results)')}
    assert 'below_floor' not in cols, 'below_floor came back'


# --- companies --------------------------------------------------------------

def case_upsert_company_is_stable_by_slug():
    c = _conn()
    a = db.upsert_company(c, 'Acme Systems')
    b = db.upsert_company(c, 'acme   systems')   # same slug, different spelling
    assert a == b, f'{a} != {b}; slug collapse failed'
    assert len(db.list_companies(c)) == 1


def case_company_keeps_first_display_name():
    c = _conn()
    db.upsert_company(c, 'Acme Systems')
    db.upsert_company(c, 'ACME SYSTEMS')
    assert db.list_companies(c)[0]['name'] == 'Acme Systems'


# --- boards -----------------------------------------------------------------

def case_boards_round_trip_in_api_shape():
    c = _conn()
    db.upsert_board(c, slug='acme', name='Acme', ats='greenhouse',
                    tags=['fintech'], status='tracked')
    out = db.list_boards(c)
    assert set(out) == {'tracked', 'discovery'}, sorted(out)
    b = out['tracked'][0]
    assert b['tags'] == ['fintech'], b['tags']
    assert b['slug'] == 'acme' and b['ats'] == 'greenhouse'
    assert 'last_probed_at' in b and 'created_at' in b


def case_board_upsert_updates_rather_than_duplicates():
    c = _conn()
    db.upsert_board(c, slug='acme', name='Acme', ats='greenhouse',
                    tags=[], status='discovery')
    db.upsert_board(c, slug='acme', name='Acme Systems', ats='greenhouse',
                    tags=['fintech'], status='tracked')
    out = db.list_boards(c)
    assert out['discovery'] == [], 'old status row survived'
    assert len(out['tracked']) == 1
    assert out['tracked'][0]['name'] == 'Acme Systems'


def case_list_boards_filtered_by_status_returns_a_flat_list():
    """scan.py calls this path on every run. It shipped broken once; pin it."""
    c = _conn()
    db.upsert_board(c, slug='acme', name='Acme', ats='greenhouse',
                    tags=['fintech'], status='tracked')
    db.upsert_board(c, slug='globex', name='Globex', ats='ashby',
                    tags=[], status='dead')
    out = db.list_boards(c, status='tracked')
    assert isinstance(out, list), f'expected a list, got {type(out).__name__}'
    assert [b['slug'] for b in out] == ['acme'], out
    assert out[0]['tags'] == ['fintech']
    assert db.list_boards(c, status='dead')[0]['slug'] == 'globex'
    assert db.list_boards(c, status='discovery') == []


def case_dead_boards_are_in_neither_bucket():
    """'dead' must not leak into tracked; a dead board is one that stopped resolving."""
    c = _conn()
    db.upsert_board(c, slug='gone', name='Gone', ats='lever', tags=[], status='dead')
    out = db.list_boards(c)
    assert out['tracked'] == [] and out['discovery'] == [], out


def case_bad_ats_is_rejected():
    c = _conn()
    try:
        db.upsert_board(c, slug='x', name='X', ats='workday',
                        tags=[], status='tracked')
    except sqlite3.IntegrityError:
        return
    raise AssertionError('ats=workday was accepted')


# --- search results ---------------------------------------------------------

def case_post_search_results_returns_batch_id_and_rows():
    c = _conn()
    bid = db.post_search_results(c, BATCH, [RESULT])
    assert bid, 'no batch id'
    rows = db.list_search_results(c, bid)
    assert len(rows) == 1, rows
    r = rows[0]
    assert r['company'] == 'Acme Systems'
    assert r['tags'] == ['devex', 'senior'], r['tags']
    assert r['is_remote'] is True and r['salary_disclosed'] is True
    assert r['salary_min'] == 180000


def case_posting_results_creates_the_company():
    c = _conn()
    db.post_search_results(c, BATCH, [RESULT])
    assert len(db.list_companies(c)) == 1


def case_empty_result_list_still_records_the_batch():
    """A scan that found nothing is a fact worth keeping, not a no-op."""
    c = _conn()
    bid = db.post_search_results(c, BATCH, [])
    assert bid
    assert db.list_search_results(c, bid) == []


def case_bad_fit_tier_is_rejected():
    c = _conn()
    bad = dict(RESULT, fit_tier='amazing')
    try:
        db.post_search_results(c, BATCH, [bad])
    except sqlite3.IntegrityError:
        return
    raise AssertionError('fit_tier=amazing was accepted')


def case_deleting_a_batch_takes_its_results():
    c = _conn()
    bid = db.post_search_results(c, BATCH, [RESULT])
    c.execute('DELETE FROM search_batches WHERE id=?', (bid,))
    c.commit()
    n = c.execute('SELECT count(*) FROM search_results').fetchone()[0]
    assert n == 0, f'{n} orphan result row(s); ON DELETE CASCADE is not wired'


# --- fit reports ------------------------------------------------------------

def case_create_fit_report_writes_signals_with_kinds():
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    rep = db.get_fit_report(c, fid)
    assert rep['verdict'] == 'strong'
    kinds = [s['kind'] for s in rep['signals']]
    assert kinds.count('match') == 2 and kinds.count('gap') == 1, kinds
    assert rep['signals'][0]['requirement'] == 'Internal tooling'


def case_fit_signal_sort_order_is_preserved():
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    match = [s for s in db.get_fit_report(c, fid)['signals'] if s['kind'] == 'match']
    assert [s['requirement'] for s in match] == ['Internal tooling', 'CI/CD']


def case_patch_fit_report_updates_narrative_fields():
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    db.patch_fit_report(c, fid, why_apply='<p>Rewritten.</p>')
    rep = db.get_fit_report(c, fid)
    assert rep['why_apply'] == '<p>Rewritten.</p>'
    assert rep['verdict_summary'] == FIT['verdict_summary'], 'patch clobbered a field'


def case_patch_bumps_updated_at():
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    before = db.get_fit_report(c, fid)['updated_at']
    c.execute("UPDATE fit_reports SET updated_at='2000-01-01T00:00:00+00:00'"
              ' WHERE id=?', (fid,))
    c.commit()
    db.patch_fit_report(c, fid, why_apply='<p>x</p>')
    assert db.get_fit_report(c, fid)['updated_at'] != '2000-01-01T00:00:00+00:00'


def case_patch_rejects_unknown_field():
    """Typos must fail loudly. A silent no-op patch is how state goes stale."""
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    try:
        db.patch_fit_report(c, fid, verdic='strong')
    except (ValueError, KeyError):
        return
    raise AssertionError('unknown field was silently accepted')


def case_bad_verdict_is_rejected():
    c = _conn()
    try:
        db.create_fit_report(c, dict(FIT, verdict='maybe'))
    except sqlite3.IntegrityError:
        return
    raise AssertionError('verdict=maybe was accepted')


def case_a_wrong_signal_can_be_deleted():
    """The deliberate improvement over the HTTP API. See jobhub-api.md."""
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    sig = db.get_fit_report(c, fid)['signals'][0]
    db.delete_fit_signal(c, sig['id'])
    left = [s['id'] for s in db.get_fit_report(c, fid)['signals']]
    assert sig['id'] not in left, 'signal survived delete'
    assert len(left) == 2, left


def case_deleting_a_fit_report_takes_its_signals():
    c = _conn()
    fid = db.create_fit_report(c, FIT)
    c.execute('DELETE FROM fit_reports WHERE id=?', (fid,))
    c.commit()
    n = c.execute('SELECT count(*) FROM fit_signals').fetchone()[0]
    assert n == 0, f'{n} orphan signal(s)'


def case_no_research_brief_id_column():
    """Research is a file now. The column fed a sidebar that no longer exists."""
    c = _conn()
    cols = {r[1] for r in c.execute('PRAGMA table_info(fit_reports)')}
    assert 'research_brief_id' not in cols


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
            print(f'FAIL {name}: {type(e).__name__}: {e}')
    print(f'\n{len(cases) - len(failures)}/{len(cases)} passed')
    if failures:
        print('FAILED: ' + ', '.join(failures), file=sys.stderr)
        return 1
    return 0


if __name__ == '__main__':
    sys.exit(main())
