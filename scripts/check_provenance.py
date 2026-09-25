#!/usr/bin/env python3
"""
The master resume is the law.

A tailored resume is a SELECTION from user/master-resume.md, not an authorship. Every
bullet and every summary sentence must cite the line(s) of the law it derives from, and
every figure and technical term it asserts must appear in those cited lines.

If a claim is true but the law does not say it, THE FIX IS TO AMEND THE LAW FIRST.
That ordering is the whole point. On 2026-08-28 an LLM review judge asserted that
"analysis-gated canaries with automated rollback" was a cross-bullet conflation. It was
not -- master-resume.md:134 says the canaries halt a bad release themselves, which IS
automated rollback -- but the finding was accepted without reading line 134, a true claim
was stripped from two resumes, and a rule was written into claim-rules.json that would
have stripped it from every future resume. Nothing in the pipeline objected, because every
gate checked what was ADDED and nothing checked what was REMOVED or whether what remained
still traced to the record.

This script closes that hole from the other side: nothing appears on a tailored resume
unless it traces to the law, and if the law changes underneath a resume, the citation
goes stale and this fails. Same staleness property review.json's summary hash already has,
applied to every claim instead of just the summary.

Usage:
    python3 scripts/check_provenance.py <dir>              # verify
    python3 scripts/check_provenance.py <dir> --propose    # draft provenance.json by best match
    python3 scripts/check_provenance.py --all
"""
import sys, re, json, hashlib, difflib
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
LAW = ROOT / "user" / "master-resume.md"
ALSO = ROOT / "user" / "personal-projects.md"

RED, GRN, YEL, DIM, OFF = "\033[31m", "\033[32m", "\033[33m", "\033[2m", "\033[0m"

# Generic connective/rewording vocabulary. Tailoring may re-order and re-word; it may not
# introduce CONTENT. These carry no claim on their own, so they are not required to appear
# in the cited source. Deliberately does NOT include outcome verbs (led, owned, drove,
# partnered) -- verb escalation is a real failure mode and those must trace to the law.
ALLOW = set("""
a an the and or but of to in on at for with from by as is are was were be been being that this
these those it its into over under across through per via not no than then so such which who whom
whose what when where while how why if each every both all any some more most other another same
i my me we our us he his him they their them you your it's dont don't
where their than about between during after before against within without upon
""".split())

FIG = re.compile(r"\$?\d[\d,]*\.?\d*\s*(?:%|\+|gb|mb|kb|k\b|x\b|min(?:ute)?s?|sec(?:ond)?s?|ms|milliseconds?|hours?|days?|weeks?|months?|years?)?", re.I)


def norm(s: str) -> str:
    """Normalize for comparison. Paraphrase is legal; content is not."""
    s = s.lower()
    s = s.replace("→", " to ").replace("—", " ").replace("–", " ")
    s = s.replace("&nbsp;", " ").replace("’", "'")
    s = re.sub(r"[*_`>#\[\]()]", " ", s)
    s = re.sub(r"(\d),(\d)", r"\1\2", s)              # 40,000 -> 40000
    # Ranges first: the law writes "30-40k daily users"; a resume writes "30,000-40,000".
    # Without this the leading 30 never expands and reads as an unsourced figure.
    s = re.sub(r"\b(\d+)\s*-\s*(\d+)k\b",
               lambda m: f"{int(m.group(1))*1000} to {int(m.group(2))*1000}", s)
    s = re.sub(r"\$?(\d+(?:\.\d+)?)k\b", lambda m: str(int(float(m.group(1)) * 1000)), s)  # $5k -> 5000
    s = re.sub(r"\bminutes?\b", "min", s)
    s = re.sub(r"\bseconds?\b", "s", s)
    s = re.sub(r"\bmilliseconds?\b", "ms", s)
    s = re.sub(r"\bpercent\b", "%", s)
    s = re.sub(r"\s+", " ", s)
    return s.strip()


def figures(s: str):
    out = set()
    for m in FIG.finditer(norm(s)):
        t = re.sub(r"\s+", "", m.group(0)).strip(",.").lstrip("$")
        if re.search(r"\d", t):
            out.add(t)
    return out


def words(s: str):
    # Trailing punctuation is not content. "crash loops." must match the law's "crash loops,"
    # -- reporting `loops.` as unsourced is a defect in this script, not in the resume.
    out = []
    for w in re.findall(r"[a-z][a-z0-9'\-\.]*", norm(s)):
        w = w.strip(".'-")
        if w and w not in ALLOW and len(w) > 2:
            out.append(w)
    return out


def sha(s: str) -> str:
    """FROZEN minimal normalization -- whitespace and case only.

    Deliberately does NOT use norm(). Hashing the comparison-normalized text means every
    stored citation goes stale the moment norm() is tuned, which is a defect in this
    script masquerading as a change in the law. Observed immediately: adding $5k -> 5000
    expansion invalidated a correct citation to master-resume.md:182. Keep this frozen."""
    return hashlib.sha256(re.sub(r"\s+", " ", s).strip().lower().encode()).hexdigest()[:16]


def law_lines():
    """Every citable line of the law, 1-indexed, keyed by line number."""
    out = {}
    for src in (LAW, ALSO):
        if not src.exists():
            continue
        rel = src.relative_to(ROOT).as_posix()
        for i, line in enumerate(src.read_text().split("\n"), 1):
            t = line.strip()
            if len(t) > 20:
                out[f"{rel}:{i}"] = t
    return out


def claims_of(md: str):
    """Every unit of a tailored resume that asserts something: summary sentences and bullets."""
    out = []
    if "## Summary" in md:
        summ = md.split("## Summary")[1].split("##")[0].strip()
        for sent in re.split(r"(?<=[.!?])\s+(?=[A-Z])", summ):
            if len(sent.strip()) > 20:
                out.append(("summary", sent.strip()))
    body = md
    for head in ("## Experience", "## Independent Engineering"):
        if head in body:
            for chunk in body.split(head)[1:]:
                chunk = chunk.split("\n## ")[0]
                for line in chunk.split("\n"):
                    t = line.strip()
                    if t.startswith("- ") and len(t) > 30:
                        out.append(("bullet", t[2:].strip()))
    # de-dup, preserve order
    seen, uniq = set(), []
    for k, t in out:
        if sha(t) not in seen:
            seen.add(sha(t))
            uniq.append((k, t))
    return uniq


def propose(d: Path):
    md = (d / "resume.md").read_text()
    law = law_lines()
    keys = list(law)
    entries = []
    for kind, text in claims_of(md):
        scored = sorted(
            ((difflib.SequenceMatcher(None, norm(text), norm(law[k])).ratio(), k) for k in keys),
            reverse=True)[:3]
        srcs = [{"ref": k, "sha": sha(law[k])} for r, k in scored if r > 0.25] or \
               [{"ref": scored[0][1], "sha": sha(law[scored[0][1]])}]
        entries.append({"kind": kind, "text_sha": sha(text),
                        "text": text[:70], "sources": srcs})
    out = {"law": LAW.relative_to(ROOT).as_posix(),
           "note": "DRAFT proposed by best-match. Every entry MUST be read and corrected by hand -- "
                   "a fuzzy match is not a citation.",
           "entries": entries}
    (d / "provenance.json").write_text(json.dumps(out, indent=2) + "\n")
    return len(entries)


def restamp(d: Path):
    """Update line numbers where the CONTENT still matches. Never changes a stored SHA.

    Absorbs reflow only. A citation whose content genuinely changed stays stale and must be
    re-verified by a human against the law -- that is the finding, not a chore to automate away.
    """
    pv_p = d / "provenance.json"
    if not pv_p.exists():
        return 0
    pv = json.loads(pv_p.read_text())
    law = law_lines()
    by_content = {}
    for k, v in law.items():
        by_content.setdefault(sha(v), k)
    n = 0
    for e in pv.get("entries", []):
        for s in e.get("sources", []):
            want, ref = s.get("sha"), s.get("ref")
            now = by_content.get(want)
            if now and now != ref:
                s["ref"] = now
                n += 1
    if n:
        pv_p.write_text(json.dumps(pv, indent=2) + "\n")
    return n


def check(d: Path):
    findings = []
    md_p, pv_p = d / "resume.md", d / "provenance.json"
    if not md_p.exists():
        return [("NO_RESUME", "", "")]
    if not pv_p.exists():
        return [("NO_PROVENANCE", "provenance.json missing",
                 "the master resume is the law -- every claim must cite it. Run --propose, then correct by hand")]

    pv = json.loads(pv_p.read_text())
    law = law_lines()
    by_sha = {e["text_sha"]: e for e in pv.get("entries", [])}

    for kind, text in claims_of(md_p.read_text()):
        e = by_sha.get(sha(text))
        if not e:
            findings.append(("UNCITED_CLAIM", text[:75],
                             f"this {kind} traces to nothing in the law -- add a citation or remove it"))
            continue
        # The stored SHA is authoritative; the line number is only a hint.
        #
        # Citing by line number alone is brittle to reflow: on 2026-08-28, deleting six lines
        # of commentary from master-resume.md shifted every line below it and marked citations
        # to :177, :185 and :250 stale even though their CONTENT was untouched. A staleness
        # signal that fires on unrelated edits is noise, and noise is what trains people to
        # ignore a gate. So resolve by content: if a line with the cited SHA still exists
        # anywhere in the law, the citation holds -- and report where it moved to.
        by_content = {}
        for k, v in law.items():
            by_content.setdefault(sha(v), k)
        pool, stale, moved = [], [], []
        for s in e.get("sources", []):
            ref, want = s.get("ref"), s.get("sha")
            if ref in law and sha(law[ref]) == want:
                pool.append(law[ref])
            elif want in by_content:
                now = by_content[want]
                pool.append(law[now])
                if now != ref:
                    moved.append(f"{ref} -> {now}")
            elif ref not in law:
                stale.append(f"{ref} no longer exists")
            else:
                stale.append(f"{ref} changed since it was cited")
        if moved:
            findings.append(("SOURCE_MOVED", text[:75],
                             "the law reflowed; same content, new line: " + "; ".join(moved)
                             + " -- run --restamp to update the citations"))
        if stale:
            findings.append(("STALE_SOURCE", text[:75],
                             "; ".join(stale) + " -- the law moved, re-verify this claim"))
        if not pool:
            continue
        joined = norm(" || ".join(pool))
        flat = re.sub(r"\s+", "", joined).replace("$", "")
        miss_f = sorted(f for f in figures(text) if f not in flat)
        if miss_f:
            findings.append(("UNSOURCED_FIGURE", text[:75],
                             f"figures not in the cited lines: {', '.join(miss_f)}"))
        # Deviation from the law's wording must be DECLARED, not assumed. A bare synonym
        # ("move" for "migrate") is a one-line declaration; a content addition forces you to
        # write down why it is claimable, in the artifact, where it can be read later. That
        # asymmetry is the point -- on 2026-08-28 "automated rollback" was silently REMOVED
        # on a judge's say-so and nothing recorded the reasoning either way.
        declared = {k.lower() for k in (e.get("reworded") or {})}
        miss_w = sorted({w for w in words(text) if w not in joined and w not in declared})
        if miss_w:
            findings.append(("UNDECLARED_TERM", text[:75],
                             f"not in the cited lines and not declared in \"reworded\": "
                             f"{', '.join(miss_w[:12])}"
                             + (f" (+{len(miss_w)-12} more)" if len(miss_w) > 12 else "")
                             + " -- either cite a line that supports it, or declare it with a reason"))
        empty = sorted(k for k, v in (e.get("reworded") or {}).items() if not str(v).strip())
        if empty:
            findings.append(("EMPTY_DECLARATION", text[:75],
                             f"declared with no reason: {', '.join(empty)} -- a blank reason is not a declaration"))
    return findings


def main():
    args = sys.argv[1:]
    if "--all" in args:
        dirs = sorted(p.parent for p in (ROOT / "user" / "tailored").rglob("resume.md"))
    else:
        dirs = [Path(a) if Path(a).is_absolute() else ROOT / a for a in args if not a.startswith("--")]
    if not dirs:
        print(__doc__)
        return 2
    bad = 0
    for d in dirs:
        name = d.relative_to(ROOT / "user" / "tailored").as_posix()
        if "--restamp" in args:
            n = restamp(d)
            print(f"{GRN if n else DIM}RESTAMP{OFF} {name}: {n} citation(s) re-pointed after reflow")
            continue
        if "--propose" in args:
            print(f"{YEL}DRAFT{OFF} {name}: {propose(d)} entries -> provenance.json  {DIM}(must be corrected by hand){OFF}")
            continue
        f = check(d)
        if not f:
            print(f"{GRN}PASS{OFF}  {name}")
        else:
            bad += 1
            print(f"{RED}FAIL{OFF}  {name}  ({len(f)} finding(s))")
            for code, where, why in f:
                print(f"    {RED}[{code}]{OFF} {where}")
                if why:
                    print(f"        {DIM}{why}{OFF}")
    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
