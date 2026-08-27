#!/usr/bin/env python3
"""Hard gate over a tailored resume. Exits non-zero on any violation.

Why this exists: the rules it enforces all existed as prose in master-resume.md,
resume-style.md and prompts/rules/ -- and were violated anyway, repeatedly, in the
same session that wrote them. Andre, 2026-08-20: "you keep leaving notes and then
forget. They should be enforced." A note is advisory; a non-zero exit is not.

Checks, in order:
  CLAIMS      -- fabricated outcomes, missing qualifiers, not-claimable items,
                 superseded figures, verb escalation, banned phrases
  PROSE       -- vague back-references ("that assistance"), dangling openers,
                 first-person-plural ownership in the summary, repeated openers
  ARTIFACT    -- PDF exists, is newer than the markdown, has no raw markdown left
                 in it, and loses no hyphens in text extraction
  REVIEW      -- evidence that the MANDATORY judge panel ran against THIS summary
                 (review.json, hash-bound so an edit invalidates it)
  GAPS        -- every posting term you are about to call "missing" is checked
                 against BOTH master-resume.md and personal-projects.md, and any
                 term outside the primary stack is listed as MUST-ASK

Usage:
    python3 scripts/resume_preflight.py user/tailored/<company>/<role>
    python3 scripts/resume_preflight.py <dir> --missing Azure,Kafka,"browser extension"
    python3 scripts/resume_preflight.py --all          # every tailored resume

Rules are data, in user/claim-rules.json. Add one the day you learn the fact.
"""
import json
import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
RULES_PATH = ROOT / "user" / "claim-rules.json"

RED, YEL, GRN, DIM, OFF = "\033[31m", "\033[33m", "\033[32m", "\033[2m", "\033[0m"


def load_rules():
    if not RULES_PATH.exists():
        print(f"{YEL}no user/claim-rules.json -- claim checks skipped{OFF}")
        return {}
    return json.loads(RULES_PATH.read_text())


def ctx(text, start, end, pad=45):
    return re.sub(r"\s+", " ", text[max(0, start - pad):end + pad]).strip()


def check_claims(text, rules):
    """Claim rules run over prose only. Skills lines are inventories, not claims --
    running verb-escalation over 'canary deployments' in a skills list only ever
    produced false positives."""
    v, low = [], text.lower()

    for pat, why in rules.get("forbidden_outcomes", []):
        for m in re.finditer(pat, low):
            v.append(("FABRICATION", ctx(text, m.start(), m.end()), why))

    for pat, need, window, why in rules.get("required_qualifiers", []):
        for m in re.finditer(pat, low):
            w = low[max(0, m.start() - window):m.end() + window]
            if not re.search(need, w):
                v.append(("UNQUALIFIED", ctx(text, m.start(), m.end()), why))

    for pat, bad, window, why in rules.get("forbidden_near", []):
        for m in re.finditer(pat, low):
            w = low[max(0, m.start() - window):m.end() + window]
            if re.search(bad, w):
                v.append(("SCOPE", ctx(text, m.start(), m.end()), why))

    for pat, why in rules.get("not_claimable", []):
        for m in re.finditer(pat, low):
            v.append(("NOT_CLAIMABLE", ctx(text, m.start(), m.end()), why))

    for pat, why in rules.get("superseded_figures", []):
        for m in re.finditer(pat, low):
            v.append(("SUPERSEDED", ctx(text, m.start(), m.end()), why))

    for p in rules.get("banned_phrases", []):
        for m in re.finditer(re.escape(p), low):
            v.append(("BANNED_PHRASE", ctx(text, m.start(), m.end()), f"'{p}' is banned in resume prose"))

    return v


VAGUE_NOUNS = r"assistance|work|effort|approach|initiative|capability|experience|thing|stuff|piece"
DANGLING_OPENERS = r"^(alongside it|along with it|with that|because of that|on top of that|off the back of that)\b"


def check_prose(md):
    """Readability, not truth. The claim rules and the artifact checks both pass on a
    sentence nobody can parse -- Andre, 2026-08-20, on 'I contributed to consolidating
    our web-session API into a single service in C# on modern .NET with that assistance':
    "Are you reviewing what you write? Very ambiguous." Nothing in this script read it.
    These rules catch the specific ways that sentence failed."""
    v = []
    summary = md.split("## Summary")[1].split("##")[0].strip() if "## Summary" in md else ""
    body = md
    if "## Skills" in body and "## Experience" in body:
        pre, rest = body.split("## Skills", 1)
        body = pre + "## Experience" + rest.split("## Experience", 1)[1]
    units = [("summary", s.strip()) for s in re.split(r"(?<=[.!?])\s+", summary) if s.strip()]
    units += [("bullet", b.strip()) for b in re.findall(r"^- (.+)$", body, re.M)]

    for where, s in units:
        low = s.lower()
        for m in re.finditer(rf"\b(that|those|this|these)\s+({VAGUE_NOUNS})\b", low):
            v.append(("VAGUE_REFERENCE", s[:90],
                      f"'{m.group(0)}' points back at nothing specific -- name the thing ({where})"))
        if re.search(DANGLING_OPENERS, low):
            v.append(("DANGLING_OPENER", s[:90],
                      f"opens with a back-reference to an unnamed antecedent ({where})"))
        if where == "summary" and re.search(r"\b(our|we|us)\b", low):
            v.append(("AMBIGUOUS_OWNERSHIP", s[:90],
                      "first-person plural in the summary blurs what he did vs what the team did"))

    # Employer tenure in the summary. resume-style.md, 2026-08-23: sentence one carries
    # career years, function and domain -- never a tenure split like "nearly five at
    # LegalZoom". Employment dates already live in Experience, where they are verifiable;
    # repeating a slice of them up top spends the highest-value line in the document on
    # something the reader is about to be told anyway.
    #
    # Scoped to the summary on purpose. A bullet may legitimately say "over three years
    # with the platform team", and the negative lookahead on `the` keeps "roughly two
    # years of the migration" out of it. The trailing capital is what makes this a
    # *tenure at an employer* rather than any duration.
    # The number and qualifier are matched case-insensitively so a sentence-initial
    # "Ten years at ..." is caught, but the employer's leading [A-Z] stays
    # case-sensitive -- that capital is the whole signal that this is an employer
    # and not a common noun.
    TENURE = (r"\b(?i:(nearly|almost|over|about|roughly|more than|~)?\s*"
              r"(one|two|three|four|five|six|seven|eight|nine|ten|\d+(?:\.\d+)?)"
              r"[\s-]*(years?|yrs?)?\s*(at|with|of))\s+(?!the\b)[A-Z][\w.&'-]*")
    for where, s in units:
        if where != "summary":
            continue
        for m in re.finditer(TENURE, s):
            v.append(("SUMMARY_TENURE", m.group(0).strip(),
                      "employer tenure does not belong in the summary -- sentence one "
                      "carries career years, function and domain, never a tenure split "
                      "(resume-style.md, 2026-08-23). Employment dates live in Experience"))

    # Subsection labels inside Experience. Andre, 2026-08-21: "Again adding boldened sub
    # sections" -- "again" because he had said it before and nothing was written down, so it
    # kept coming back on every new resume. build_resume.py renders `*Label*` at font-weight
    # 700, which puts a bolded heading in direct competition with the job titles beneath it.
    # He chose to drop them entirely rather than unbold them: bullets sit under the company
    # and title block in one continuous list. This lives in the script, not in a style note,
    # because a note that has to be remembered is not a control.
    # Match BOTH *italic* and **bold** forms, and tolerate trailing whitespace. The first
    # version of this check used ^\*([^*\n]+)\*$ and missed **Bold Label** entirely --
    # which is the exact form Andre complained about, so the guard would have let the
    # thing it was written for walk straight through. Caught 2026-08-21 by testing it
    # against the bold form instead of assuming.
    exp = md.split("## Experience", 1)[1] if "## Experience" in md else ""
    for label in re.findall(r"(?m)^\*{1,2}([^*\n]+)\*{1,2}[ \t]*$", exp):
        v.append(("SUBSECTION_LABEL", label.strip(),
                  "subsection labels are not used -- delete the line and let the bullets run"))

    # Repeated sentence openers read as a list, not an argument.
    openers = {}
    for where, s in units:
        if where != "summary":
            continue
        key = " ".join(s.lower().split()[:2])
        openers.setdefault(key, []).append(s)
    for key, group in openers.items():
        if len(group) > 1:
            v.append(("REPEATED_OPENER", f'"{key}..." x{len(group)}',
                      "two summary sentences open the same way -- vary them or merge"))
    return v


def prose_of(md):
    """Summary + bullets. Excludes the Skills block and headers."""
    summary = md.split("## Summary")[1].split("##")[0] if "## Summary" in md else ""
    body = md
    if "## Skills" in body and "## Experience" in body:
        pre, rest = body.split("## Skills", 1)
        body = pre + "## Experience" + rest.split("## Experience", 1)[1]
    bullets = re.findall(r"^- (.+)$", body, re.M)
    return summary + "\n" + "\n".join(bullets)


def summary_text(md):
    return re.sub(r"\s+", " ", md.split("## Summary")[1].split("##")[0]).strip() if "## Summary" in md else ""


def check_review(d: Path, md_text):
    """Require evidence that the MANDATORY judge panel in resume-style.md actually ran,
    against THIS version of the summary.

    A script cannot perform the review and cannot make anyone think. What it can do is
    refuse to pass while the evidence is missing or stale, and make the skip visible to
    Andre rather than silent. That matters because the review loop has been marked
    MANDATORY in resume-style.md since 2026-08-09 and was skipped on all three resumes
    built on 2026-08-20 -- Andre: "Ok, so how do we enforce it?"

    review.json must sit beside resume.md and contain:
        {"summary_sha256": "<sha of the summary reviewed>",
         "lenses": [{"lens": "...", "verdict": "pass"|"fail", "findings": [...]}, ...]}

    The hash is what makes it tamper-evident: edit the summary and the review goes stale
    automatically, exactly like the PDF does.
    """
    import hashlib
    p = d / "review.json"
    want = hashlib.sha256(summary_text(md_text).encode()).hexdigest()
    if not p.exists():
        return [("NO_REVIEW", "review.json missing",
                 "the MANDATORY summary review loop (resume-style.md) has not run -- "
                 "run the judge panel and write review.json")]
    try:
        r = json.loads(p.read_text())
    except Exception as e:
        return [("REVIEW_UNREADABLE", str(e), "review.json is not valid JSON")]

    v = []
    if r.get("summary_sha256") != want:
        v.append(("REVIEW_STALE", "summary changed since review",
                  "the summary was edited after it was reviewed -- re-run the panel"))
    lenses = r.get("lenses") or []
    names = {str(l.get("lens", "")).lower() for l in lenses}
    if len(lenses) < 3:
        v.append(("REVIEW_THIN", f"{len(lenses)} lens(es)",
                  "resume-style.md requires 3-4 judges with DIFFERENT lenses"))
    if not any("honest" in n or "evidence" in n for n in names):
        v.append(("REVIEW_MISSING_LENS", "evidence/honesty",
                  "resume-style.md: the honesty lens 'catches the most and must be included'"))
    for l in lenses:
        if str(l.get("verdict", "")).lower() != "pass":
            v.append(("REVIEW_FAILED", f"{l.get('lens')}: {l.get('verdict')}",
                      "; ".join(str(x) for x in (l.get("findings") or []))[:160]))
    return v


def check_artifact(d: Path):
    v = []
    md, pdf = d / "resume.md", d / "resume.pdf"
    if not pdf.exists():
        return [("NO_PDF", str(pdf), "no PDF built -- run scripts/build_resume.py")]
    if md.stat().st_mtime > pdf.stat().st_mtime + 1:
        v.append(("STALE_PDF", pdf.name,
                  "resume.md is newer than resume.pdf -- rebuild before sending"))
    try:
        txt = subprocess.run(["pdftotext", str(pdf), "-"], capture_output=True,
                             text=True, timeout=60).stdout
    except Exception as e:
        return v + [("PDF_UNREADABLE", str(e), "could not extract PDF text")]

    flat = re.sub(r"\s+", " ", txt)
    if "**" in flat:
        v.append(("RAW_MARKDOWN", "**", "literal ** in the PDF -- a markdown line the template did not parse"))

    src = md.read_text()
    for w in sorted({w for w in re.findall(r"\b[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+\b", src)}):
        if w not in flat and w.replace("-", "") in flat:
            v.append(("HYPHEN_LOST", w,
                      f"renders as '{w.replace('-', '')}' to a text parser -- ATS keyword match fails"))

    # Every block the markdown declares must actually survive into the PDF.
    #
    # **Why this exists.** build_resume.py silently dropped any line in ## Experience it
    # did not recognise. Tailored resumes write role lines and subsection labels without
    # bold, which matched no branch, so the Courier Health and Datadog PDFs shipped with
    # NO job titles and NO employment dates at all -- and passed this preflight clean,
    # because nothing here checked that the employment history was present. The renderer
    # is fixed; this is the check that makes a recurrence loud instead of invisible.
    # Found 2026-08-21 only by looking at the rendered page.
    for company in re.findall(r"^###\s+(.+?)\s*(?:—|$)", src, re.M):
        if company.strip() and company.strip() not in flat:
            v.append(("COMPANY_MISSING", company.strip(),
                      "declared in resume.md but absent from the PDF -- the renderer dropped it"))

    md_roles = re.findall(r"^\**(.+?)\**\s+—\s+(\w+ \d{4}\s*-\s*\w+ \d{4})\s*$", src, re.M)
    for title, dates in md_roles:
        title = title.strip().strip("*")
        if title and title not in flat:
            v.append(("TITLE_MISSING", title,
                      "job title in resume.md is absent from the PDF -- the renderer dropped it"))
        norm = re.sub(r"\s*-\s*", " - ", dates.strip())
        if norm not in re.sub(r"\s*-\s*", " - ", flat):
            v.append(("DATES_MISSING", f"{title}: {dates}",
                      "employment dates in resume.md are absent from the PDF -- the renderer dropped them"))
    if not md_roles:
        v.append(("NO_ROLE_LINES", "## Experience",
                  "no 'Title — Mon YYYY - Mon YYYY' lines in resume.md -- a resume with no dates"))

    # Subsection labels (*Product engineering*) are dropped by the same class of bug.
    for label in re.findall(r"^\*([^*\n]{3,60})\*\s*$", src, re.M):
        if label.strip() not in flat:
            v.append(("SUBSECTION_MISSING", label.strip(),
                      "subsection label in resume.md is absent from the PDF -- the renderer dropped it"))

    n = subprocess.run(["pdfinfo", str(pdf)], capture_output=True, text=True).stdout
    m = re.search(r"Pages:\s+(\d+)", n)
    if m and int(m.group(1)) > 2:
        v.append(("TOO_LONG", f"{m.group(1)} pages", "over two pages"))
    return v


def check_gaps(missing, rules):
    """A term is only a gap if BOTH source files lack it. Anything outside the
    primary stack must be put to Andre before it is called a gap at all."""
    out = []
    src = {}
    for f in ("master-resume.md", "personal-projects.md"):
        p = ROOT / "user" / f
        src[f] = p.read_text().lower() if p.exists() else ""
    primary = set(rules.get("primary_stack", []))

    for term in missing:
        t = term.strip().lower()
        if not t:
            continue
        # Word-boundary match. A naive substring check reported "Java" as present
        # because master-resume.md says "JavaScript" -- observed 2026-08-20 on a
        # Home Depot req whose preferred stack is Java/Spring Boot.
        pat = re.compile(r"(?<![a-z0-9])" + re.escape(t) + r"(?![a-z0-9])", re.I)
        hits = []
        for f, body in src.items():
            for line in body.split("\n"):
                if pat.search(line):
                    hits.append((f, re.sub(r"\s+", " ", line).strip()))
        if hits:
            # Deliberately does NOT conclude. A substring match cannot tell evidence
            # ("shipped a React Native app") from a note recording its absence
            # ("Kafka is dabbling only -- never list it"). Both look identical here.
            # Observed 2026-08-20: Azure/Kafka/Angular were reported NOT_A_GAP purely
            # because the rule notes denying them mention the word. Surfacing the
            # matching lines forces the reader to judge instead of trusting the match.
            why = "term appears in the record -- READ these lines and decide; do not assume either way:"
            for f, line in hits[:3]:
                why += f"\n            {DIM}{f}:{OFF} {line[:150]}"
            out.append(("REVIEW_EVIDENCE", term, why))
        elif t not in primary:
            out.append(("MUST_ASK", term,
                        "outside the primary stack and absent from both files -- ASK Andre before "
                        "calling it a gap. Absence from the record is not absence from Andre."))
    return out


def run(d: Path, missing, rules):
    md = d / "resume.md"
    if not md.exists():
        print(f"{RED}no resume.md in {d}{OFF}")
        return 1
    raw = md.read_text()
    v = check_claims(prose_of(raw), rules)
    v += check_prose(raw)
    v += check_review(d, raw)
    v += check_artifact(d)
    v += check_gaps(missing, rules)

    label = str(d.relative_to(ROOT / "user" / "tailored"))
    if not v:
        print(f"{GRN}PASS{OFF}  {label}")
        return 0

    hard = [x for x in v if x[0] != "MUST_ASK"]
    print(f"{RED if hard else YEL}{'FAIL' if hard else 'ASK '}{OFF}  {label}  ({len(v)} finding(s))")
    for kind, quote, why in v:
        colour = YEL if kind == "MUST_ASK" else RED
        print(f"    {colour}[{kind}]{OFF} {quote[:88]}")
        print(f"        {DIM}{why}{OFF}")
    return 1


def main():
    args = [a for a in sys.argv[1:]]
    missing = []
    if "--missing" in args:
        i = args.index("--missing")
        missing = args[i + 1].split(",")
        del args[i:i + 2]
    rules = load_rules()

    if "--all" in args:
        dirs = sorted(p.parent for p in (ROOT / "user" / "tailored").rglob("resume.md")
                      if "_archive" not in p.parts)
    elif args:
        dirs = [Path(args[0]).resolve()]
    else:
        print(__doc__)
        return 1

    rc = 0
    for d in dirs:
        rc |= run(d, missing, rules)
    return rc


if __name__ == "__main__":
    sys.exit(main())
