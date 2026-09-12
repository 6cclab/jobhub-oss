#!/usr/bin/env python3
"""Deal-breakers enforced in Python, never by a model.

`preferences.md` records these as absolute. A quantized local model must not be
the thing standing between a defense contractor and a job application, so they
are checked here in code, after the model has spoken, as a veto it cannot
override.

A veto is also a statement of *certainty*, which is what makes it useful
downstream: `appeal.py` skips vetoed drops entirely rather than paying a cloud
model to re-confirm a rule that was never in doubt.

Everything requiring judgement -- above all the AI charter question -- stays
with the model. See `user/screen-profile.md`.
"""
import re

# Rejected after a phone screen 2026-08-19. Andre: "Seeq was a rejection so omit."
# They keep posting Staff/Principal roles publicly, so this fires often.
CLOSED_COMPANIES = {'seeq'}

DEFENSE = re.compile(
    r'\banduril\b|\bpalantir\b|lockheed|raytheon|northrop|general dynamics'
    r'|\bbae systems\b|l3harris|booz allen|leidos|\bsaic\b|mitre|draper'
    r'|huntington ingalls|\brtx\b|sierra nevada corp|shield ai|epirus', re.I)

# Explicit partisan / advocacy alignment. ActBlue is the reference case.
POLITICAL = re.compile(r'\bactblue\b|\bwinred\b|democratic national|republican national'
                       r'|campaign committee|\bpac\b\s|political action committee', re.I)

# Management-only. "Manager, X" and "Director" are already cut by the title
# prefilter; this catches the rest and the no-IC-coding phrasings.
MGMT_TITLE = re.compile(r'^\s*(engineering|software|technical)\s+manager\b'
                        r'|^\s*(senior\s+)?manager\b|head of engineering'
                        r'|\bvp\b|vice president|^\s*director\b', re.I)

# Staff and above. HARD FILTER as of 2026-08-21, per preferences.md: "Do not
# surface, evaluate, or write fit reports for Staff-titled reqs. Do not argue
# for an exception."
#
# This lives in Python, not in the screen profile, deliberately. The screen
# profile said "Staff is in scope when the fit is strong" and the 2026-08-21
# scan kept 92 Staff-titled roles out of 266 -- a third of the queue was roles
# Andre had already decided not to apply to. A quantized model asked to weigh
# "fit" against a ban will find strong fits, because strong fit is exactly when
# the ban is tempting. So it is not asked.
#
# The filter is also evidence-backed, not just a preference: Seeq rejected him
# from a Staff req saying they expected someone more technical for the level.
TOO_SENIOR = re.compile(r'\b(staff|principal|distinguished|fellow)\b|\bl[5-9]\b', re.I)

# "Senior Staff" is ONE band, not a Senior offer. Two regex attempts at this in
# a single lookahead both failed -- \bsr\.?\b(?!\s+staff) backtracked the period
# away and let "Sr. Staff" through, and \s*(?!staff) matched zero characters and
# let "Senior Staff" through. The logic is stated plainly instead: strike the
# compound band out of the title first, then ask whether a Senior offer remains.
SENIOR_STAFF = re.compile(r'\b(?:senior|sr)\.?\s+staff\b', re.I)
SENIOR_BAND = re.compile(r'\b(?:senior|sr)\.?\b', re.I)


def offers_senior(title):
    """True when the title advertises Senior as a band in its own right.

    "Senior / Staff Software Engineer" does. "Senior Staff Software Engineer"
    does not -- that is a single band above Staff.
    """
    return bool(SENIOR_BAND.search(SENIOR_STAFF.sub(' ', title or '')))


# Both bounds are captured so a range takes its UPPER value: "8-12 years" is a
# 12-year posting, and reading it as 8 would slip past the 10-year veto.
YEARS = re.compile(r'(\d{1,2})\s*(?:\+|\s*(?:-|–|to)\s*(\d{1,2}))?\s*(?:\+\s*)?(?:years|yrs)', re.I)

MOBILE_ONLY = re.compile(r'\b(ios|android)\s+engineer\b|\bmobile engineer\b'
                         r'|\bswift\b.*\bkotlin\b', re.I)

LEGACY_STACK = re.compile(r'\b(asp\.net|vb\.net|webforms|sharepoint|coldfusion)\b', re.I)

# Identity and Access Management as the team's CHARTER. Andre, 2026-08-21:
# "Ignore IAM roles." Eight had surfaced and one was sitting in a live queue.
#
# Deliberately narrow. Matches IAM and "identity and/& access", NOT a bare
# "identity" -- he has real identity experience (event-driven MFA delivery,
# Customer Session Refresh to 100% of production traffic, SSO Proxy
# observability) and adjacent identity work is still in scope at lower weight.
# It is the charter that is out, not the subject matter.
#
# \biam\b is safe next to "Miami" because of the word boundaries, and this only
# ever reads the title, never the location list.
IAM_CHARTER = re.compile(r'\biam\b|\bidentity\s*(?:and|&)\s*access\b', re.I)


def max_years_required(text):
    """Highest 'N years' figure in the requirements. None if unstated.

    Deliberately takes the max: a posting saying "5 years backend, 10 years
    overall" is a 10-year posting.
    """
    if not text:
        return None
    vals = []
    for m in YEARS.finditer(text):
        for g in (m.group(1), m.group(2)):
            if g and int(g) <= 30:
                vals.append(int(g))
    return max(vals) if vals else None


def check(posting):
    """Return a veto reason string, or None if the posting survives.

    Only absolute, non-judgement rules belong here. If a call needs reading
    comprehension, it belongs in the screen profile instead.
    """
    company = (posting.get('company') or '').strip()
    title = posting.get('title') or ''
    desc = posting.get('description') or ''
    blob = f'{company} {title} {desc}'

    if company.lower() in CLOSED_COMPANIES:
        return f'closed: {company} rejected the candidate, do not resurface'

    if DEFENSE.search(company) or DEFENSE.search(title):
        return 'deal-breaker: defense/military contractor'

    if POLITICAL.search(company) or POLITICAL.search(blob[:600]):
        return 'deal-breaker: politically aligned or advocacy organization'

    if MGMT_TITLE.search(title):
        return 'out of scope: management-only title'

    # Staff+ titles are out. The one survivor is a genuine multi-band req that
    # also offers Senior ("Senior / Staff Software Engineer") -- levelling during
    # the process is not the same as applying to Staff, and preferences.md says
    # to surface those.
    if TOO_SENIOR.search(title) and not offers_senior(title):
        return 'out of scope: Staff+ title, and Senior is not offered alongside'

    yrs = max_years_required(desc)
    if yrs is not None and yrs >= 10:
        return f'out of scope: requires {yrs}+ years'

    if MOBILE_ONLY.search(title):
        return 'out of scope: purely mobile role'

    if LEGACY_STACK.search(title):
        return 'out of scope: legacy .NET stack'

    if IAM_CHARTER.search(title):
        return 'out of scope: IAM-charter role'

    return None
