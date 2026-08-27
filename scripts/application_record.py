#!/usr/bin/env python3
"""Read and write one application record: user/applications/<company>-<role>.md.

This is the ONLY code that knows the record format. Everything else goes
through parse()/serialise().

The format is a deliberately narrow subset of YAML because this repo has no
PyYAML (see scripts/build_resume.py:184, which regexes config.yaml for the same
reason). A value is everything after the first ': ' to end of line, stripped --
no quote processing and no escaping, so colons, quotes and pipes inside a note
are safe and there is nothing to get wrong.

Unknown keys are an ERROR. These records are the authoritative job-search record
and are not in git, so silently dropping a field loses data permanently.
"""
import re
from pathlib import Path

STATUSES = ('applied', 'phone_screen', 'onsite', 'offer', 'rejected',
            'ghosted', 'withdrawn')
REQUIRED = ('company', 'role', 'status', 'source', 'submitted', 'applied')
OPTIONAL = ('resume', 'packet', 'server_id', 'fit', 'events')
EVENT_KEYS = ('date', 'to', 'note')
_DATE = re.compile(r'^\d{4}-\d{2}-\d{2}$')
_ORDER = REQUIRED + OPTIONAL


class RecordError(Exception):
    """A record file is malformed. Never swallow this -- fail loudly."""


def _split(text):
    if not text.startswith('---\n'):
        raise RecordError('file does not start with a "---" frontmatter fence')
    end = text.find('\n---\n', 3)
    if end == -1:
        raise RecordError('frontmatter is not closed by a "---" line')
    return text[4:end + 1], text[end + 5:].lstrip('\n')


def parse(text):
    """Return a dict of the frontmatter fields plus 'body'. Raise RecordError."""
    front, body = _split(text)
    rec, events, in_events = {}, [], False

    for lineno, line in enumerate(front.split('\n'), start=2):
        if not line.strip():
            continue
        if line.startswith('  '):
            if not in_events:
                raise RecordError(f'line {lineno}: indented line outside "events:"')
            stripped = line.strip()
            if stripped.startswith('- '):
                events.append({})
                stripped = stripped[2:]
            if not events:
                raise RecordError(f'line {lineno}: event field before any "- "')
            key, _, val = stripped.partition(': ')
            if key not in EVENT_KEYS:
                raise RecordError(f'line {lineno}: unknown event key "{key}"')
            events[-1][key] = val.strip()
            continue

        in_events = False
        key, sep, val = line.partition(':')
        if not sep:
            raise RecordError(f'line {lineno}: not a "key: value" line: {line!r}')
        key, val = key.strip(), val.strip()
        if key == 'events':
            if val:
                raise RecordError(f'line {lineno}: "events:" takes no inline value')
            in_events = True
            continue
        if key not in _ORDER:
            raise RecordError(f'line {lineno}: unknown key "{key}"')
        if key in rec:
            raise RecordError(f'line {lineno}: duplicate key "{key}"')
        rec[key] = val

    missing = [k for k in REQUIRED if k not in rec or not rec[k]]
    if missing:
        raise RecordError(f'missing required key(s): {", ".join(missing)}')
    if rec['status'] not in STATUSES:
        raise RecordError(f'unknown status "{rec["status"]}"; '
                          f'expected one of {", ".join(STATUSES)}')
    if rec['submitted'] not in ('true', 'false'):
        raise RecordError(f'submitted must be true or false, got "{rec["submitted"]}"')
    rec['submitted'] = rec['submitted'] == 'true'
    if not _DATE.match(rec['applied']):
        raise RecordError(f'applied must be YYYY-MM-DD, got "{rec["applied"]}"')

    for i, ev in enumerate(events):
        for k in ('date', 'to'):
            if k not in ev:
                raise RecordError(f'event {i}: missing "{k}"')
        if not _DATE.match(ev['date']):
            raise RecordError(f'event {i}: date must be YYYY-MM-DD, got "{ev["date"]}"')
    if events:
        rec['events'] = events
    rec['body'] = body
    return rec


def serialise(rec):
    """Inverse of parse(). Deterministic key order so diffs stay readable."""
    out = ['---']
    for key in _ORDER:
        if key == 'events' or key not in rec:
            continue
        val = rec[key]
        if isinstance(val, bool):
            val = 'true' if val else 'false'
        out.append(f'{key}: {val}')
    if rec.get('events'):
        events_block = ['events:']
        for ev in rec['events']:
            events_block.append(f'  - date: {ev["date"]}')
            events_block.append(f'    to: {ev["to"]}')
            if ev.get('note'):
                events_block.append(f'    note: {ev["note"]}')
        out.extend(events_block)
    out.append('---')
    return '\n'.join(out) + '\n\n' + rec.get('body', '')


def load(path):
    """Parse a record file, prefixing RecordError with the path."""
    path = Path(path)
    try:
        return parse(path.read_text())
    except RecordError as e:
        raise RecordError(f'{path}: {e}') from e
