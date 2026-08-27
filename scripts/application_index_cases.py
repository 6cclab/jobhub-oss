#!/usr/bin/env python3
"""Regression tests for the generated user/applications.md index.

The index is a projection: it must never be the only home for a fact. These
cases pin that it renders every record, sorts newest-first, and REFUSES to
build when a record is malformed rather than quietly omitting it -- silent
omission is the failure this whole design exists to remove.

Placeholder companies: this file is published to the public mirror.

Usage:
    python3 scripts/application_index_cases.py
"""
import datetime
import re
import shutil
import sys
import tempfile
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

import application_record as ar  # noqa: E402
import build_application_index as bai  # noqa: E402
import migrate_applications as mig  # noqa: E402

ACME = """---
company: Acme
role: Backend Engineer
status: phone_screen
source: recruiter-inbound
submitted: false
applied: 2026-08-24
---

Acme body.
"""

GLOBEX = """---
company: Globex
role: Platform Engineer
status: applied
source: greenhouse
submitted: true
applied: 2026-08-26
---

Globex body.
"""


def _dir(**files):
    d = Path(tempfile.mkdtemp())
    for name, text in files.items():
        (d / f'{name}.md').write_text(text)
    return d


def case_renders_every_record():
    out = bai.build(_dir(acme=ACME, globex=GLOBEX))
    assert 'Acme' in out and 'Globex' in out, out


def case_sorts_newest_first():
    out = bai.build(_dir(acme=ACME, globex=GLOBEX))
    assert out.index('Globex') < out.index('Acme'), out


def case_carries_the_do_not_edit_header():
    out = bai.build(_dir(acme=ACME))
    assert 'DO NOT EDIT' in out.split('\n')[0].upper() or 'DO NOT EDIT' in out[:400], out


def case_renders_submitted_column():
    out = bai.build(_dir(acme=ACME, globex=GLOBEX))
    acme_row = [l for l in out.split('\n') if 'Acme' in l][0]
    globex_row = [l for l in out.split('\n') if 'Globex' in l][0]
    assert 'no' in acme_row.lower(), acme_row
    assert 'yes' in globex_row.lower(), globex_row


def case_links_to_the_record_file():
    out = bai.build(_dir(acme=ACME))
    assert 'applications/acme.md' in out, out


def case_malformed_record_fails_the_build():
    bad = ACME.replace('status: phone_screen', 'status: bogus')
    try:
        bai.build(_dir(acme=ACME, broken=bad))
    except ar.RecordError as e:
        assert 'bogus' in str(e), str(e)
        return
    raise AssertionError('a malformed record was skipped instead of failing the build')


def case_empty_dir_produces_a_header_and_no_rows():
    out = bai.build(_dir())
    assert 'DO NOT EDIT' in out.upper()
    assert '| 20' not in out, out


def case_pipe_in_role_does_not_split_the_row():
    piped = ACME.replace('role: Backend Engineer',
                         'role: Backend | Platform Engineer')
    out = bai.build(_dir(acme=piped))
    rows = [l for l in out.split('\n') if 'Backend' in l]
    assert len(rows) == 1, rows
    row = rows[0]
    assert 'Backend \\| Platform Engineer' in row, row
    header_row = [l for l in bai.HEADER.split('\n') if l.startswith('| Date')][0]
    # Count only UNESCAPED pipes -- an escaped '\|' inside a cell is data,
    # not a column separator, and must not inflate the column count.
    unescaped = len(re.findall(r'(?<!\\)\|', row))
    assert unescaped == header_row.count('|'), (row, header_row)


TABLE = """# Application Tracker

Some prose that is not a row.

| Date | Company | Role | Source | Status | Age | Resume | Notes |
|------|---------|------|--------|--------|-----|--------|-------|
| 2026-08-24 | Acme | Backend Engineer | Recruiter inbound | **phone_screen** | 2d | acme/r.pdf | Long **notes** with a | pipe? no. |
| 2026-08-26 | Globex | Platform Engineer | Greenhouse | **applied** | 0d | globex/r.pdf | Short note. |

Trailing prose, also not a row.
"""


def case_table_rows_are_parsed():
    rows = mig.parse_table(TABLE)
    assert len(rows) == 2, rows
    assert rows[0]['Company'] == 'Acme', rows[0]
    assert rows[1]['Company'] == 'Globex'


def case_notes_cell_is_kept_verbatim():
    rows = mig.parse_table(TABLE)
    assert rows[1]['Notes'] == 'Short note.', repr(rows[1]['Notes'])


def case_non_row_prose_is_collected():
    lines = mig.non_row_lines(TABLE)
    joined = '\n'.join(lines)
    assert 'Some prose that is not a row.' in joined
    assert 'Trailing prose, also not a row.' in joined
    assert 'Acme' not in joined, joined


# The real file is three tables with three different schemas under three
# '## ' sections, not one table -- one of them has no Status column at all.
# This fixture has the same shape: two sections, two column counts, the
# second missing Status entirely.
MULTI_TABLE = """# Tracker

## Live / awaiting response

| Date | Company | Role | Source | Status | Age | Resume | Notes |
|------|---------|------|--------|--------|-----|--------|-------|
| 2026-08-24 | Acme | Backend Engineer | Recruiter inbound | **phone_screen** | 2d | acme/r.pdf | Live note. |

## Started but never submitted

| Date | Company | Role | Stage reached | Notes |
|------|---------|------|----------------|-------|
| 2026-08-20 | Globex | Platform Engineer | Draft only | Abandoned before submitting. |
"""


def case_each_row_parses_against_its_own_table_header():
    rows = mig.parse_table(MULTI_TABLE)
    assert len(rows) == 1, rows
    assert rows[0]['Company'] == 'Acme', rows[0]
    assert rows[0]['source_table'] == 'Live / awaiting response', rows[0]


def case_status_less_row_is_excluded_from_application_rows():
    rows = mig.parse_table(MULTI_TABLE)
    companies = {r['Company'] for r in rows}
    assert 'Globex' not in companies, rows


def case_status_less_row_notes_reach_the_pipeline():
    lines = mig.non_row_lines(MULTI_TABLE)
    joined = '\n'.join(lines)
    assert 'Abandoned before submitting.' in joined, joined


def case_slug_matches_packet_naming():
    assert mig.slug('Acme', 'Software Engineer, Backend (Mid/Senior)') == \
        'acme-software-engineer-backend-mid-senior', \
        mig.slug('Acme', 'Software Engineer, Backend (Mid/Senior)')


def case_slug_is_stable_for_two_roles_at_one_company():
    a = mig.slug('Acme', 'Backend Engineer')
    b = mig.slug('Acme', 'Platform Engineer')
    assert a != b, (a, b)


def case_row_becomes_a_valid_record_with_verbatim_body():
    rows = mig.parse_table(TABLE)
    rec = mig.row_to_record(rows[1])
    assert rec['company'] == 'Globex'
    assert rec['status'] == 'applied', rec['status']
    assert rec['body'].strip() == 'Short note.', repr(rec['body'])
    ar.parse(ar.serialise(rec))  # must round-trip through the real parser


def case_bold_status_markers_are_stripped():
    rows = mig.parse_table(TABLE)
    rec = mig.row_to_record(rows[0])
    assert rec['status'] == 'phone_screen', rec['status']


def case_unmappable_status_raises():
    row = dict(mig.parse_table(TABLE)[0])
    row['Status'] = '**interviewing**'
    try:
        mig.row_to_record(row)
    except SystemExit:
        return
    raise AssertionError('an unmappable status was silently accepted')


class _MigWorkdir:
    """Point migrate_applications at a throwaway user/ dir for the duration
    of a `with` block, restoring the real module globals on exit. Never
    used against the real user/applications.md.
    """

    def __enter__(self):
        self.tmp = Path(tempfile.mkdtemp())
        self.user = self.tmp / 'user'
        self.user.mkdir()
        self._orig = (mig.ROOT, mig.TABLE, mig.RECORDS, mig.PIPELINE)
        mig.ROOT = self.tmp
        mig.TABLE = self.user / 'applications.md'
        mig.RECORDS = self.user / 'applications'
        mig.PIPELINE = self.user / 'pipeline.md'
        return self

    def __exit__(self, *exc):
        mig.ROOT, mig.TABLE, mig.RECORDS, mig.PIPELINE = self._orig
        shutil.rmtree(self.tmp, ignore_errors=True)


class _Args:
    def __init__(self, force=False):
        self.force = force


def case_migrate_stops_on_slug_collision():
    collide = """# Application Tracker

| Date | Company | Role | Source | Status | Age | Resume | Notes |
|------|---------|------|--------|--------|-----|--------|-------|
| 2026-08-24 | Acme | Backend/Platform Engineer | Greenhouse | **applied** | 1d | acme/r.pdf | First note. |
| 2026-08-25 | Acme | Backend-Platform Engineer | Greenhouse | **applied** | 1d | acme/r2.pdf | Second note. |
"""
    with _MigWorkdir() as w:
        mig.TABLE.write_text(collide)
        rc = mig.cmd_migrate(_Args())
        assert rc == 1, rc
        assert not mig.RECORDS.exists() or not any(mig.RECORDS.glob('*.md')), \
            'a colliding slug still wrote a record'


def case_migrate_never_overwrites_an_existing_backup():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(TABLE)
        assert mig.cmd_migrate(_Args()) == 0

        stamp = datetime.date.today().isoformat()
        first_backup = w.user / f'_backup-{stamp}' / 'applications.md'
        assert first_backup.exists(), first_backup
        original_bytes = first_backup.read_bytes()

        # Simulate the trigger the reviewer found: row_to_record raised on a
        # bad status after the first backup was written, the operator
        # hand-edited applications.md, and re-ran the same day.
        mig.TABLE.write_text(TABLE.replace('Short note.', 'Edited note.'))
        assert mig.cmd_migrate(_Args(force=True)) == 0

        assert first_backup.read_bytes() == original_bytes, \
            'a same-day rerun overwrote the first backup'
        second_backup = w.user / f'_backup-{stamp}-2' / 'applications.md'
        assert second_backup.exists(), 'rerun did not use a distinct backup path'
        assert b'Edited note.' in second_backup.read_bytes(), second_backup.read_bytes()


def case_migrate_refuses_to_overwrite_pipeline_without_force():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(TABLE)
        assert mig.cmd_migrate(_Args()) == 0
        curated = mig.PIPELINE.read_text() + '\nHand-added strategy note.\n'
        mig.PIPELINE.write_text(curated)

        # Isolate the pipeline guard from the records guard: remove the
        # records directory so only pipeline.md's own existence can block
        # a force-less rerun.
        shutil.rmtree(mig.RECORDS)

        rc = mig.cmd_migrate(_Args())
        assert rc == 1, rc
        assert mig.PIPELINE.read_text() == curated, \
            'pipeline.md was overwritten by a force-less rerun'


def case_migrate_backs_up_curated_pipeline_before_a_forced_rerun():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(TABLE)
        assert mig.cmd_migrate(_Args()) == 0
        curated = mig.PIPELINE.read_text() + '\nHand-added strategy note.\n'
        mig.PIPELINE.write_text(curated)

        # A --force rerun for an unrelated reason (bad status, collision,
        # partial failure) must not lose the curated pipeline.md: it should
        # land in the run's backup directory before being overwritten.
        assert mig.cmd_migrate(_Args(force=True)) == 0

        stamp = datetime.date.today().isoformat()
        backup_pipeline = w.user / f'_backup-{stamp}-2' / 'pipeline.md'
        assert backup_pipeline.exists(), 'curated pipeline.md was not backed up'
        assert backup_pipeline.read_text() == curated, \
            'the backed-up pipeline.md does not match the curated content'
        # And the live file now holds the freshly regenerated content, not
        # the curated prose -- the backup is where curation survives.
        assert 'Hand-added strategy note.' not in mig.PIPELINE.read_text()


def case_migrate_backs_up_records_before_a_forced_rerun():
    # The Critical finding this pins: cmd_migrate backed up applications.md
    # and pipeline.md before overwriting them, but never backed up
    # user/applications/ itself before the write loop overwrote files in
    # it by slug. A --force rerun could silently wipe real, hand-curated
    # record bodies with nothing to recover from (user/ has no git
    # history). Confirmed this fails without the RECORDS backup: reverting
    # that hunk makes `backup_record.exists()` below fail, because nothing
    # ever copies user/applications/ into the backup directory.
    with _MigWorkdir() as w:
        mig.TABLE.write_text(TABLE)
        assert mig.cmd_migrate(_Args()) == 0

        acme_record = mig.RECORDS / 'acme-backend-engineer.md'
        assert acme_record.exists(), acme_record
        curated = ar.load(acme_record)
        curated['body'] = 'Hand-curated prose that must survive.\n'
        acme_record.write_text(ar.serialise(curated))

        # A --force rerun for an unrelated reason (bad status, collision,
        # a hand-edit to applications.md) is about to overwrite every
        # record file the table maps a slug to -- including this one.
        assert mig.cmd_migrate(_Args(force=True)) == 0

        stamp = datetime.date.today().isoformat()
        backup_record = (w.user / f'_backup-{stamp}-2' / 'applications'
                          / 'acme-backend-engineer.md')
        assert backup_record.exists(), \
            'user/applications/ was not backed up before the rerun overwrote it'
        backed_up = ar.load(backup_record)
        assert 'Hand-curated prose that must survive.' in backed_up['body'], \
            backed_up['body']
        # And the live file now holds the freshly regenerated content, not
        # the curated prose -- same shape as the pipeline.md guard above:
        # the backup is where curation survives, not the live file.
        live = ar.load(acme_record)
        assert 'Hand-curated prose that must survive.' not in live['body']


def case_migrate_refuses_to_parse_the_generated_index():
    # Once migrate has run, user/applications.md is regenerated by
    # build_application_index.py and carries its "DO NOT EDIT" marker. It
    # still has a Status column, so parse_table's row classifier accepts
    # every row in it, and row_to_record would set every body to just the
    # Notes cell -- wiping every real record.
    #
    # The real generated index has no Notes column at all, so
    # _assert_has_notes_column would also catch it -- this fixture adds a
    # Notes column on purpose, so this case exercises ONLY the marker
    # guard and isn't masked by the other one. Confirmed this fails
    # without _assert_not_generated: cmd_migrate then runs to completion
    # (return 0, having written a record) instead of raising, so the
    # `raise AssertionError` below fires.
    with _MigWorkdir() as w:
        widened = bai.HEADER.replace(
            '| Date | Company | Role | Status | Sent | Source | Record |',
            '| Date | Company | Role | Status | Sent | Source | Record | Notes |'
        ).replace(
            '|------|---------|------|--------|------|--------|--------|',
            '|------|---------|------|--------|------|--------|--------|-------|'
        )
        assert 'Notes' in widened and 'DO NOT EDIT' in widened, widened
        generated = widened + (
            '| 2026-08-24 | Acme | Backend Engineer | applied | yes '
            '| greenhouse | [acme-backend-engineer]'
            '(applications/acme-backend-engineer.md) | Some note. |\n')
        mig.TABLE.write_text(generated)
        try:
            mig.cmd_migrate(_Args())
        except SystemExit:
            pass
        else:
            raise AssertionError('migrate accepted the generated index')
        assert not mig.RECORDS.exists() or not any(mig.RECORDS.glob('*.md')), \
            'the generated index was parsed and records were written from it'


def case_reconcile_refuses_to_parse_the_generated_index():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(bai.HEADER)
        try:
            mig.cmd_reconcile(_Args())
        except SystemExit:
            pass
        else:
            raise AssertionError('reconcile accepted the generated index')


def case_migrate_refuses_a_table_with_no_notes_column():
    # Even without the marker, a source table with a Status column but no
    # Notes column can only produce empty record bodies -- the same
    # silent-wipe failure through a different door. Confirmed this fails
    # without the guard: without it, cmd_migrate writes a record (body is
    # just '\n') and returns 0 instead of raising, so the RECORDS glob is
    # non-empty and the `raise AssertionError` below fires.
    no_notes = """# Application Tracker

| Date | Company | Role | Source | Status | Age | Resume |
|------|---------|------|--------|--------|-----|--------|
| 2026-08-24 | Acme | Backend Engineer | Greenhouse | **applied** | 1d | acme/r.pdf |
"""
    with _MigWorkdir() as w:
        mig.TABLE.write_text(no_notes)
        try:
            mig.cmd_migrate(_Args())
        except SystemExit:
            pass
        else:
            raise AssertionError('migrate accepted a table with no Notes column')
        assert not mig.RECORDS.exists() or not any(mig.RECORDS.glob('*.md')), \
            'a table with no Notes column still wrote records'


def case_verify_detects_a_dropped_notes_cell():
    d = Path(tempfile.mkdtemp())
    (d / 'applications.md').write_text(TABLE)
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(TABLE)
    # Write only ONE of the two records: the second row's notes go missing.
    rec = mig.row_to_record(rows[0])
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    missing = mig.missing_notes(TABLE, recs)
    assert any('Short note.' in m for m in missing), missing


def case_verify_passes_when_every_cell_landed():
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    for row in mig.parse_table(TABLE):
        rec = mig.row_to_record(row)
        (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(
            ar.serialise(rec))
    assert mig.missing_notes(TABLE, recs) == [], mig.missing_notes(TABLE, recs)


def case_missing_notes_pipe_cell_survives_its_own_rejoin_spacing():
    # An embedded '|' in a Notes cell is split and rejoined with ' | ' by
    # _cells. The record body and this check must derive that value from
    # the SAME parse (parse_table), or a spacing mismatch could produce a
    # false failure here. No row in the live file has this shape today, but
    # TABLE's Acme row does, so this exercises the path directly.
    rows = mig.parse_table(TABLE)
    piped_row = [r for r in rows if r['Company'] == 'Acme'][0]
    assert ' | ' in piped_row['Notes'], piped_row['Notes']
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rec = mig.row_to_record(piped_row)
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    missing = mig.missing_notes(TABLE, recs)
    assert not any('pipe' in m for m in missing), missing


def case_missing_notes_detects_a_dropped_statusless_note():
    # Population 2: a row from a table with NO 'Status' column. Its Notes
    # never reach a record; they must reach pipeline_text instead. Give a
    # pipeline_text that does NOT contain it, and confirm it is flagged.
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(MULTI_TABLE)
    rec = mig.row_to_record(rows[0])
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    missing = mig.missing_notes(MULTI_TABLE, recs, pipeline_text='unrelated text')
    assert any('Abandoned before submitting.' in m for m in missing), missing


def case_missing_notes_statusless_note_passes_when_it_reached_pipeline():
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(MULTI_TABLE)
    rec = mig.row_to_record(rows[0])
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    pipeline_text = '\n'.join(mig.non_row_lines(MULTI_TABLE))
    missing = mig.missing_notes(MULTI_TABLE, recs, pipeline_text=pipeline_text)
    assert missing == [], missing


def case_missing_notes_statusless_population_is_opt_in():
    # Omitting pipeline_text checks population 1 only -- documented as
    # deliberate in missing_notes' docstring, pinned here so a future edit
    # cannot silently make the statusless check mandatory (or silently drop
    # it) without a test noticing.
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(MULTI_TABLE)
    rec = mig.row_to_record(rows[0])
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    # The statusless row's note is nowhere -- no pipeline.md was written at
    # all -- yet passing no pipeline_text must not flag it.
    missing = mig.missing_notes(MULTI_TABLE, recs)
    assert missing == [], missing


# Two rows whose Notes overlap on purpose: Acme's note contains Globex's
# note as a literal substring. Used to prove a fully-dropped record can hide
# behind another record's text under a blob-wide containment check.
OVERLAP_TABLE = """# Application Tracker

| Date | Company | Role | Source | Status | Age | Resume | Notes |
|------|---------|------|--------|--------|-----|--------|-------|
| 2026-08-24 | Acme | Backend Engineer | Greenhouse | **applied** | 1d | acme/r.pdf | Short note. And then some. |
| 2026-08-25 | Globex | Platform Engineer | Greenhouse | **applied** | 1d | globex/r.pdf | Short note. |
"""


def case_missing_notes_reports_a_record_file_that_was_never_written():
    # A plain missing record: Globex's row is in the table but no file was
    # ever written for it, and there is no overlap with any other body.
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(TABLE)
    acme = [r for r in rows if r['Company'] == 'Acme'][0]
    rec = mig.row_to_record(acme)
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    # Globex's record is never written -- an entirely dropped row.
    missing = mig.missing_notes(TABLE, recs)
    assert any('Short note.' in m for m in missing), missing


def case_missing_notes_catches_a_dropped_row_hiding_behind_another_record():
    # The hole a global blob-containment check has: Globex's record is
    # NEVER written, but its Notes text ("Short note.") is also a literal
    # substring of Acme's body ("Short note. And then some.") in
    # OVERLAP_TABLE. A check that asks "does this text appear ANYWHERE in
    # the concatenation of all bodies" says yes and misses the dropped
    # record entirely. Each row must be checked against its OWN record
    # file, resolved by slug(), not against the blob -- absence of that
    # file is itself a finding, independent of any text search.
    d = Path(tempfile.mkdtemp())
    recs = d / 'applications'
    recs.mkdir()
    rows = mig.parse_table(OVERLAP_TABLE)
    acme = [r for r in rows if r['Company'] == 'Acme'][0]
    rec = mig.row_to_record(acme)
    (recs / (mig.slug(rec['company'], rec['role']) + '.md')).write_text(ar.serialise(rec))
    # Globex's record is never written. Its slug'd file must not exist.
    globex_file = recs / (mig.slug('Globex', 'Platform Engineer') + '.md')
    assert not globex_file.exists()
    missing = mig.missing_notes(OVERLAP_TABLE, recs)
    assert any('Short note.' in m for m in missing), \
        f'a fully dropped record hid behind another record\'s text: {missing}'


class _VerifyArgs:
    def __init__(self, backup=None):
        self.backup = backup


def case_verify_passes_end_to_end_on_a_clean_migration():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(MULTI_TABLE)
        assert mig.cmd_migrate(_Args()) == 0
        assert mig.cmd_verify(_VerifyArgs()) == 0


def case_verify_catches_a_statusless_note_dropped_after_migration():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(MULTI_TABLE)
        assert mig.cmd_migrate(_Args()) == 0
        # Simulate the loss this task exists to catch: pipeline.md gets
        # truncated (by a bad edit, a bad re-render, anything) after a
        # correct migration wrote it.
        mig.PIPELINE.write_text('# Pipeline\n\nnothing relevant here\n')
        assert mig.cmd_verify(_VerifyArgs()) == 1


def case_verify_catches_a_record_body_dropped_after_migration():
    with _MigWorkdir() as w:
        mig.TABLE.write_text(TABLE)
        assert mig.cmd_migrate(_Args()) == 0
        # Simulate an application record losing its body after migration.
        globex = mig.RECORDS / 'globex-platform-engineer.md'
        rec = ar.load(globex)
        rec['body'] = ''
        globex.write_text(ar.serialise(rec))
        assert mig.cmd_verify(_VerifyArgs()) == 1


def case_verify_reports_missing_backup_instead_of_crashing():
    with _MigWorkdir() as w:
        assert mig.cmd_verify(_VerifyArgs(backup='2000-01-01')) == 1


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
