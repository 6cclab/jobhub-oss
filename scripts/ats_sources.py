#!/usr/bin/env python3
"""Shared ATS fetchers and screening regexes.

Every job source in JobHub normalizes to the same posting dict:

    {'source', 'company', 'title', 'location', 'is_remote',
     'posted', 'url', 'description'}

Two families live here:

- **Board ATSs** (Greenhouse, Ashby, Lever) — one board per company, slug-keyed.
  The tracked list comes from the JobHub server, not from this file.
- **Enterprise ATSs** (Workday, Oracle) — large regional employers that never
  appear on a Greenhouse board. The employer list IS in this file, because
  each one has to be discovered by hand.

`ats_scan.py` and `scan.py` both import from here. Keep it dependency-free:
stdlib only, so it runs under a bare `python3` from launchd.
"""
import html
import json
import re
import time
import urllib.error
import urllib.request

UA = {'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)',
      'Accept': 'application/json', 'Content-Type': 'application/json'}

TIMEOUT = 30

# ---------------------------------------------------------------- screening

# Commutable metros for the configured home base (see `location` in
# user/config.yaml). The South Jersey and Philadelphia entries are the close
# tier at 15-30 minutes; the New York entries are 90-120 minutes each way and
# are listed deliberately as a separate, lower tier rather than dropped.
#
# Keep the home town itself out of this file -- it is published to the public
# mirror, where the scrub gate in .github/workflows/mirror.yaml rejects
# personal contact details. Metro names are fine; a home address is not.
REGION = re.compile(
    r'Philadelphia|Camden|Cherry Hill|Mount Laurel|Marlton|Moorestown|Voorhees|Mt Laurel'
    r'|Malvern|Wayne, PA|Radnor|Conshohocken|King of Prussia|Exton|Berwyn|Oaks, PA'
    r'|Wilmington|Newark, DE|Delaware'
    r'|New Jersey|NJ\b|Pennsylvania|PA\b'
    r'|New York|NYC|Jersey City|Newark, NJ', re.I)

REMOTE = re.compile(r'\bremote\b|\bdistributed\b|work from home|anywhere in the us', re.I)

ENG = re.compile(r'engineer|developer|software|platform|devops|sre\b|architect', re.I)
LEVEL = re.compile(r'senior|staff|sr\.?\b|lead\b|principal|\bII\b|\bIII\b', re.I)
EXCLUDE = re.compile(
    r'\bintern\b|internship|campus|graduate program|apprentice'
    r'|sales|account executive|marketing|recruit|nurse|clinical|physician|therapist'
    r'|technician|driver|warehouse|custodian|security officer|call center'
    r'|manager,|director|vice president|\bvp\b', re.I)

# Salary patterns, most specific first. Group semantics differ per pattern, so
# each is paired with a parser rather than assumed to be (min, max).
_SALARY_PATTERNS = [
    re.compile(r'\$\s*([\d,]{5,9})\s*(?:-|–|—|to)\s*\$?\s*([\d,]{5,9})'),
    re.compile(r'\$\s*(\d{2,3})\s*[kK]\s*(?:-|–|—|to)\s*\$?\s*(\d{2,3})\s*[kK]'),
]


def parse_salary(text):
    """Best-effort (min, max) annual USD from posting text. (None, None) if absent.

    Deliberately conservative: anything outside a plausible annual salary band
    is discarded rather than guessed at, because a wrong number on a dashboard
    is worse than a blank one.
    """
    if not text:
        return None, None
    for pat in _SALARY_PATTERNS:
        for m in pat.finditer(text):
            a, b = m.group(1), m.group(2)
            lo = int(a.replace(',', ''))
            hi = int(b.replace(',', ''))
            if 'k' in m.group(0).lower() and lo < 1000:
                lo, hi = lo * 1000, hi * 1000
            if lo > hi:
                lo, hi = hi, lo
            if 40_000 <= lo <= 900_000 and lo <= hi <= 1_500_000:
                return lo, hi
    return None, None


_TAG = re.compile(r'<[^>]+>')


def strip_html(s):
    if not s:
        return ''
    return re.sub(r'\s+', ' ', html.unescape(_TAG.sub(' ', s))).strip()


def _get(url, data=None):
    req = urllib.request.Request(url, data=data, headers=UA)
    return json.load(urllib.request.urlopen(req, timeout=TIMEOUT))


# ------------------------------------------------------------ board fetchers

def greenhouse(slug, company=None):
    d = _get(f'https://boards-api.greenhouse.io/v1/boards/{slug}/jobs?content=true')
    out = []
    for j in d.get('jobs', []):
        loc = (j.get('location') or {}).get('name') or ''
        desc = strip_html(j.get('content'))
        out.append({
            'source': 'greenhouse', 'company': company or j.get('company_name') or slug,
            'title': j.get('title') or '', 'location': loc,
            'is_remote': bool(REMOTE.search(loc) or REMOTE.search(desc[:400])),
            'posted': (j.get('first_published') or j.get('updated_at') or '')[:10],
            'url': j.get('absolute_url') or '', 'description': desc,
        })
    return out


def ashby(slug, company=None):
    d = _get(f'https://api.ashbyhq.com/posting-api/job-board/{slug}')
    out = []
    for j in d.get('jobs', []):
        if j.get('isListed') is False:
            continue
        out.append({
            'source': 'ashby', 'company': company or slug,
            'title': j.get('title') or '', 'location': j.get('location') or '',
            'is_remote': bool(j.get('isRemote')),
            'posted': (j.get('publishedAt') or '')[:10],
            'url': j.get('jobUrl') or j.get('applyUrl') or '',
            'description': j.get('descriptionPlain') or strip_html(j.get('descriptionHtml')),
        })
    return out


def lever(slug, company=None):
    d = _get(f'https://api.lever.co/v0/postings/{slug}')
    out = []
    for j in d:
        cats = j.get('categories') or {}
        loc = cats.get('location') or ''
        posted = ''
        if j.get('createdAt'):
            posted = time.strftime('%Y-%m-%d', time.gmtime(int(j['createdAt']) / 1000))
        out.append({
            'source': 'lever', 'company': company or slug,
            'title': j.get('text') or '', 'location': loc,
            'is_remote': (j.get('workplaceType') or '').lower() == 'remote' or bool(REMOTE.search(loc)),
            'posted': posted,
            'url': j.get('hostedUrl') or j.get('applyUrl') or '',
            'description': j.get('descriptionPlain') or strip_html(j.get('description')),
        })
    return out


BOARD_FETCHERS = {'greenhouse': greenhouse, 'ashby': ashby, 'lever': lever}


# ------------------------------------------------------- enterprise fetchers

def workday(p, company=None, limit=20):  # Workday rejects limit > 20 with a bare HTTP 400
    out, off = [], 0
    while True:
        url = f"https://{p['tenant']}.{p['wd']}.myworkdayjobs.com/wday/cxs/{p['tenant']}/{p['site']}/jobs"
        body = json.dumps({"appliedFacets": {}, "limit": limit, "offset": off, "searchText": ""}).encode()
        d = _get(url, data=body)
        posts = d.get('jobPostings', [])
        for j in posts:
            loc = j.get('locationsText') or ''
            out.append({
                'source': 'workday', 'company': company or p['tenant'],
                'title': j.get('title') or '', 'location': loc,
                'is_remote': bool(REMOTE.search(loc)),
                'posted': j.get('postedOn') or '',
                'url': f"https://{p['tenant']}.{p['wd']}.myworkdayjobs.com/en-US/{p['site']}{j.get('externalPath','')}",
                'description': '',  # list endpoint carries no body; fetched on demand
            })
        off += limit
        if not posts or off >= d.get('total', 0) or off >= 2000:
            return out
        time.sleep(0.2)


def oracle(p, company=None, limit=100):
    out, off = [], 0
    while True:
        finder = f"findReqs;siteNumber={p['site']},limit={limit},offset={off}"
        url = (f"https://{p['host']}/hcmRestApi/resources/latest/recruitingCEJobRequisitions"
               f"?onlyData=true&expand=requisitionList.secondaryLocations&finder={finder}")
        d = _get(url)
        it = d['items'][0]
        reqs = it.get('requisitionList', [])
        for r in reqs:
            sec = ' '.join((s.get('LocationName') or '') for s in (r.get('secondaryLocations') or []))
            loc = f"{r.get('PrimaryLocation') or ''} {sec}".strip()
            out.append({
                'source': 'oracle', 'company': company or p['site'],
                'title': r.get('Title') or '', 'location': loc,
                'is_remote': bool(REMOTE.search(loc)),
                'posted': str(r.get('PostedDate'))[:10],
                'url': f"https://{p['host']}/hcmUI/CandidateExperience/en/sites/{p['site']}/job/{r.get('Id')}",
                'description': '',
            })
        off += limit
        if not reqs or off >= it.get('TotalJobsCount', 0) or off >= 2000:
            return out
        time.sleep(0.2)


# name -> (kind, params). Verified working 2026-08-20.
# Adding an employer: fetch their careers page and grep for myworkdayjobs.com /
# oraclecloud.com / icims.com, then add a row. Workday site names are NOT the
# locale in the URL -- /en-US/Foo means site=Foo.
EMPLOYERS = {
    'Comcast':           ('workday', {'tenant': 'comcast', 'site': 'Comcast_Careers', 'wd': 'wd5'}),
    'TD Bank':           ('workday', {'tenant': 'td', 'site': 'TD_Bank_Careers', 'wd': 'wd3'}),
    'Crown Holdings':    ('workday', {'tenant': 'crownholdings', 'site': 'CrownHoldings', 'wd': 'wd501'}),
    'Holman':            ('workday', {'tenant': 'holmanautogroup', 'site': 'HolmanEnterprisesCareers', 'wd': 'wd1'}),
    'SEI':               ('workday', {'tenant': 'seic', 'site': 'SEI_Global_Services', 'wd': 'wd1'}),
    'Jefferson Health':  ('workday', {'tenant': 'jeffersonhealth', 'site': 'ThomasJeffersonExternal', 'wd': 'wd5'}),
    'Chubb':             ('oracle',  {'host': 'fa-ewgu-saasfaprod1.fa.ocs.oraclecloud.com', 'site': 'CX_1'}),
    'Subaru of America': ('oracle',  {'host': 'hcal.fa.us2.oraclecloud.com', 'site': 'CX_1'}),
    'American Express':  ('oracle',  {'host': 'egug.fa.us2.oraclecloud.com', 'site': 'CX_1'}),
    'Vanguard':          ('workday', {'tenant': 'vanguard', 'site': 'Vanguard_External', 'wd': 'wd5'}),
    'Cigna':             ('workday', {'tenant': 'cigna', 'site': 'CignaCareers', 'wd': 'wd5'}),
    'Wawa':              ('workday', {'tenant': 'wawa', 'site': 'Careers', 'wd': 'wd1'}),
}

ENTERPRISE_FETCHERS = {'workday': workday, 'oracle': oracle}


def fetch_employer(name):
    """Fetch one enterprise employer. Returns (name, postings, error)."""
    kind, params = EMPLOYERS[name]
    try:
        return name, ENTERPRISE_FETCHERS[kind](params, company=name), None
    except Exception as e:
        return name, [], f'{type(e).__name__}: {str(e)[:70]}'


def fetch_board(ats, slug, company=None):
    """Fetch one board. Returns (label, postings, error)."""
    label = company or slug
    fn = BOARD_FETCHERS.get((ats or '').lower())
    if not fn:
        return label, [], f'unsupported ats: {ats}'
    try:
        return label, fn(slug, company=company), None
    except urllib.error.HTTPError as e:
        return label, [], f'HTTP {e.code}'
    except Exception as e:
        return label, [], f'{type(e).__name__}: {str(e)[:70]}'


def prefilter(p, all_locations=False):
    """Cheap deterministic screen. True = worth spending a model call on.

    Runs before any LLM sees the posting and cuts thousands of reqs to hundreds
    at zero cost. Intentionally crude -- it screens on title and location only,
    and leaves every judgement call to triage.
    """
    title = p.get('title') or ''
    if not ENG.search(title) or EXCLUDE.search(title) or not LEVEL.search(title):
        return False
    if all_locations:
        return True
    return bool(p.get('is_remote') or REGION.search(p.get('location') or ''))
