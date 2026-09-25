#!/usr/bin/env python3
"""The funnel store: a local SQLite file, replacing the JobHub HTTP API.

Design: docs/sqlite-funnel-design.md. Schema was translated from the Go
server's migrations, which were deleted with it on 2026-09-18; this file is now
the only definition of the funnel schema.

WHAT LIVES HERE and what does not, because getting this backwards has already
cost a day (see prompts/rules/where-funnel-work-goes.md):

    here   boards, search batches, search results, fit reports, fit signals
           -- machine-generated, machine-read, voluminous, regenerable
    files  applications, research briefs, evals, resumes
           -- hand-written, diffable, authoritative, NEVER copied in here

Four tables the server had are deliberately NOT recreated: research_briefs,
applications, application_events, eval_results. They moved to files on
2026-08-26 and recreating them would rebuild the problem that move solved.

Return shapes match what the HTTP API returned, so call sites port by swapping
a function call for a curl and changing nothing else.

Usage:
    import jobhub_db as db
    conn = db.connect()                 # user/jobhub.db, WAL, schema applied
    cid  = db.upsert_company(conn, 'Acme')
    bid  = db.post_search_results(conn, batch, results)
"""
import json
import os
import re
import sqlite3
import uuid
from datetime import datetime, timezone
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent

# $JOBHUB_DB redirects the store, the way $JOBHUB_URL used to redirect the
# server. Use it to develop against a throwaway file instead of writing fake
# companies into the real funnel. See prompts/rules/where-funnel-work-goes.md.
DEFAULT_PATH = Path(os.environ.get('JOBHUB_DB') or (REPO / 'user' / 'jobhub.db'))

ATS = ('greenhouse', 'lever', 'ashby')
BOARD_STATUS = ('tracked', 'discovery', 'dead')
FIT_TIERS = ('strong', 'good')
VERDICTS = ('strong', 'worth', 'stretch', 'skip')
SIGNAL_KINDS = ('match', 'gap', 'flag')

# Narrative fields only. verdict is NOT patchable: changing a verdict after the
# fact rewrites history the digest already reported. Create a new report.
PATCHABLE = ('verdict_summary', 'why_apply', 'level', 'location', 'posting_url')

SCHEMA = """
CREATE TABLE IF NOT EXISTS companies (
    id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, slug TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS boards (
    id TEXT PRIMARY KEY, slug TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
    ats TEXT NOT NULL CHECK (ats IN ('greenhouse','lever','ashby')),
    tags TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('tracked','discovery','dead')),
    last_probed_at TEXT, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS search_batches (
    id TEXT PRIMARY KEY, ran_at TEXT NOT NULL, board_count INTEGER NOT NULL DEFAULT 0,
    raw_count INTEGER NOT NULL DEFAULT 0, location_filter TEXT, notes TEXT,
    created_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS search_results (
    id TEXT PRIMARY KEY,
    search_batch_id TEXT NOT NULL REFERENCES search_batches(id) ON DELETE CASCADE,
    company_id TEXT NOT NULL REFERENCES companies(id),
    role_title TEXT NOT NULL, location TEXT, is_remote INTEGER NOT NULL DEFAULT 0,
    salary_min INTEGER, salary_max INTEGER, salary_disclosed INTEGER NOT NULL DEFAULT 0,
    posting_url TEXT NOT NULL,
    fit_tier TEXT NOT NULL CHECK (fit_tier IN ('strong','good')),
    tags TEXT NOT NULL DEFAULT '', level_tag TEXT, domain_tag TEXT,
    created_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS fit_reports (
    id TEXT PRIMARY KEY, company_id TEXT NOT NULL REFERENCES companies(id),
    role_title TEXT NOT NULL, location TEXT, level TEXT, posting_url TEXT,
    verdict TEXT NOT NULL CHECK (verdict IN ('strong','worth','stretch','skip')),
    verdict_summary TEXT NOT NULL, why_apply TEXT,
    created_at TEXT NOT NULL, updated_at TEXT NOT NULL);

CREATE TABLE IF NOT EXISTS fit_signals (
    id TEXT PRIMARY KEY,
    fit_report_id TEXT NOT NULL REFERENCES fit_reports(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('match','gap','flag')),
    requirement TEXT NOT NULL, evidence TEXT NOT NULL, source TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0);

CREATE INDEX IF NOT EXISTS idx_results_batch ON search_results(search_batch_id);
CREATE INDEX IF NOT EXISTS idx_signals_report ON fit_signals(fit_report_id);
CREATE INDEX IF NOT EXISTS idx_reports_company ON fit_reports(company_id);
CREATE INDEX IF NOT EXISTS idx_batches_ran_at ON search_batches(ran_at DESC);
"""


def now():
    """ISO-8601 UTC, the shape the API already returned. SQLite has no timestamptz."""
    return datetime.now(timezone.utc).isoformat()


def new_id():
    return str(uuid.uuid4())


def slugify(name):
    s = re.sub(r'[^a-z0-9]+', '-', (name or '').lower()).strip('-')
    return s or 'unknown'


def _tags_out(raw):
    return [t for t in (raw or '').split(',') if t]


def _tags_in(tags):
    if isinstance(tags, str):
        tags = [tags]
    return ','.join(str(t).strip() for t in (tags or []) if str(t).strip())


def connect(path=None):
    """Open the store, applying the schema if absent. Safe to call repeatedly."""
    p = Path(path) if path else DEFAULT_PATH
    p.parent.mkdir(parents=True, exist_ok=True)
    conn = sqlite3.connect(p)
    conn.row_factory = sqlite3.Row
    # Both are per-connection, not stored in the file. They must be set every
    # time, not once at creation -- a reconnect without them silently accepts
    # orphan rows.
    conn.execute('PRAGMA foreign_keys = ON')
    conn.execute('PRAGMA journal_mode = WAL')
    conn.executescript(SCHEMA)
    conn.commit()
    return conn


# --- companies --------------------------------------------------------------

def upsert_company(conn, name):
    """-> company_id, keyed on slug. First display name wins."""
    slug = slugify(name)
    row = conn.execute('SELECT id FROM companies WHERE slug=?', (slug,)).fetchone()
    if row:
        return row['id']
    cid, t = new_id(), now()
    conn.execute('INSERT INTO companies (id,name,slug,created_at,updated_at)'
                 ' VALUES (?,?,?,?,?)', (cid, name or slug, slug, t, t))
    conn.commit()
    return cid


def list_companies(conn):
    return [dict(r) for r in conn.execute('SELECT * FROM companies ORDER BY name')]


# --- boards -----------------------------------------------------------------

def upsert_board(conn, slug, name, ats, tags=(), status='tracked',
                 last_probed_at=None):
    """-> board_id. Updates in place on slug; never duplicates."""
    t = now()
    row = conn.execute('SELECT id FROM boards WHERE slug=?', (slug,)).fetchone()
    if row:
        conn.execute('UPDATE boards SET name=?, ats=?, tags=?, status=?,'
                     ' last_probed_at=COALESCE(?, last_probed_at), updated_at=?'
                     ' WHERE id=?',
                     (name, ats, _tags_in(tags), status, last_probed_at, t, row['id']))
        conn.commit()
        return row['id']
    bid = new_id()
    conn.execute('INSERT INTO boards (id,slug,name,ats,tags,status,last_probed_at,'
                 'created_at,updated_at) VALUES (?,?,?,?,?,?,?,?,?)',
                 (bid, slug, name, ats, _tags_in(tags), status, last_probed_at, t, t))
    conn.commit()
    return bid


def list_boards(conn, status=None):
    """Matches GET /api/boards: {"tracked": [...], "discovery": [...]}.

    Passing status returns a flat list instead, for callers that want one bucket.
    """
    q = 'SELECT * FROM boards'
    args = ()
    if status:
        q += ' WHERE status=?'
        args = (status,)
    rows = []
    for r in conn.execute(q + ' ORDER BY name', args):
        d = dict(r)
        d['tags'] = _tags_out(d['tags'])
        rows.append(d)
    if status:
        return rows
    return {'tracked': [r for r in rows if r['status'] == 'tracked'],
            'discovery': [r for r in rows if r['status'] == 'discovery']}


# --- search results ---------------------------------------------------------

def post_search_results(conn, batch, results):
    """-> search_batch_id. Mirrors POST /api/search-results.

    Batch and rows land in ONE transaction: a crash mid-write must not leave a
    batch claiming 200 roles with 40 rows under it.
    """
    bid, t = new_id(), now()
    try:
        conn.execute('INSERT INTO search_batches (id,ran_at,board_count,raw_count,'
                     'location_filter,notes,created_at) VALUES (?,?,?,?,?,?,?)',
                     (bid, batch.get('ran_at') or t, batch.get('board_count', 0),
                      batch.get('raw_count', 0), batch.get('location_filter'),
                      batch.get('notes'), t))
        for r in results or []:
            conn.execute(
                'INSERT INTO search_results (id,search_batch_id,company_id,role_title,'
                'location,is_remote,salary_min,salary_max,salary_disclosed,posting_url,'
                'fit_tier,tags,level_tag,domain_tag,created_at)'
                ' VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)',
                (new_id(), bid, upsert_company(conn, r.get('company')),
                 r.get('role_title') or 'Unknown', r.get('location'),
                 1 if r.get('is_remote') else 0, r.get('salary_min'), r.get('salary_max'),
                 1 if r.get('salary_disclosed') else 0, r.get('posting_url') or '',
                 r.get('fit_tier') or 'good', _tags_in(r.get('tags')),
                 r.get('level_tag'), r.get('domain_tag'), t))
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    return bid


def list_search_results(conn, batch_id):
    rows = []
    for r in conn.execute(
            'SELECT sr.*, c.name AS company FROM search_results sr'
            ' JOIN companies c ON c.id = sr.company_id'
            ' WHERE sr.search_batch_id=? ORDER BY sr.fit_tier, c.name', (batch_id,)):
        d = dict(r)
        d['tags'] = _tags_out(d['tags'])
        d['is_remote'] = bool(d['is_remote'])
        d['salary_disclosed'] = bool(d['salary_disclosed'])
        rows.append(d)
    return rows


def latest_batch(conn):
    r = conn.execute('SELECT * FROM search_batches ORDER BY ran_at DESC'
                     ' LIMIT 1').fetchone()
    return dict(r) if r else None


# --- fit reports ------------------------------------------------------------

def create_fit_report(conn, payload):
    """-> fit_report_id. Mirrors POST /api/fit-reports, signals included."""
    fid, t = new_id(), now()
    try:
        conn.execute(
            'INSERT INTO fit_reports (id,company_id,role_title,location,level,'
            'posting_url,verdict,verdict_summary,why_apply,created_at,updated_at)'
            ' VALUES (?,?,?,?,?,?,?,?,?,?,?)',
            (fid, upsert_company(conn, payload.get('company')),
             payload.get('role_title') or 'unknown', payload.get('location'),
             payload.get('level'), payload.get('posting_url'),
             payload.get('verdict'), payload.get('verdict_summary') or '',
             payload.get('why_apply'), t, t))
        order = 0
        for kind in SIGNAL_KINDS:
            for s in payload.get(f'{kind}_signals') or []:
                conn.execute('INSERT INTO fit_signals (id,fit_report_id,kind,'
                             'requirement,evidence,source,sort_order)'
                             ' VALUES (?,?,?,?,?,?,?)',
                             (new_id(), fid, kind, s.get('requirement') or '',
                              s.get('evidence') or '', s.get('source'), order))
                order += 1
        conn.commit()
    except Exception:
        conn.rollback()
        raise
    return fid


def get_fit_report(conn, fit_report_id):
    r = conn.execute('SELECT fr.*, c.name AS company FROM fit_reports fr'
                     ' JOIN companies c ON c.id = fr.company_id'
                     ' WHERE fr.id=?', (fit_report_id,)).fetchone()
    if not r:
        return None
    d = dict(r)
    d['signals'] = [dict(s) for s in conn.execute(
        'SELECT * FROM fit_signals WHERE fit_report_id=? ORDER BY sort_order',
        (fit_report_id,))]
    return d


def patch_fit_report(conn, fit_report_id, **fields):
    """Narrative fields only. An unknown key raises rather than no-opping."""
    bad = [k for k in fields if k not in PATCHABLE]
    if bad:
        raise ValueError(f'not patchable: {", ".join(sorted(bad))};'
                         f' allowed: {", ".join(PATCHABLE)}')
    if not fields:
        return
    sets = ', '.join(f'{k}=?' for k in fields)
    conn.execute(f'UPDATE fit_reports SET {sets}, updated_at=? WHERE id=?',
                 (*fields.values(), now(), fit_report_id))
    conn.commit()


def delete_fit_signal(conn, signal_id):
    """The one deliberate improvement over the HTTP API, which had no DELETE:
    jobhub-api.md warned that "a wrong one cannot be removed". That was an HTTP
    surface limitation, not a data one."""
    conn.execute('DELETE FROM fit_signals WHERE id=?', (signal_id,))
    conn.commit()


def list_fit_reports(conn, verdict=None, limit=100):
    q = ('SELECT fr.*, c.name AS company FROM fit_reports fr'
         ' JOIN companies c ON c.id = fr.company_id')
    args = []
    if verdict:
        q += ' WHERE fr.verdict=?'
        args.append(verdict)
    q += ' ORDER BY fr.created_at DESC LIMIT ?'
    args.append(limit)
    return [dict(r) for r in conn.execute(q, args)]


def load_boards_export(conn, path):
    """Load a GET /api/boards JSON dump. Returns the number of boards written.

    This is how the 110 hand-curated slugs survive the server retirement. It is
    an upsert, so re-running it is harmless.
    """
    data = json.loads(Path(path).read_text())
    n = 0
    for bucket in ('tracked', 'discovery'):
        for b in data.get(bucket) or []:
            upsert_board(conn, slug=b['slug'], name=b.get('name') or b['slug'],
                         ats=b.get('ats') or 'greenhouse', tags=b.get('tags') or [],
                         status=b.get('status') or bucket,
                         last_probed_at=b.get('last_probed_at'))
            n += 1
    return n


if __name__ == '__main__':
    c = connect()
    print(f'{DEFAULT_PATH}')
    for t in ('companies', 'boards', 'search_batches', 'search_results',
              'fit_reports', 'fit_signals'):
        print(f'  {t:16} {c.execute(f"SELECT count(*) FROM {t}").fetchone()[0]:>6}')
