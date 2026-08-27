#!/usr/bin/env python3
"""Regression test for scripts/vetoes.py.

The vetoes are the rules a quantized model is never asked to weigh -- deal-breakers
and band filters that must hold absolutely. That makes them worth testing, and the
Staff filter in particular got two wrong regex implementations before this file
existed:

  \\bsr\\.?\\b(?!\\s+staff)   -- backtracked the optional period away, ended the match
                            at "Sr", saw ". Staff" (not \\s+staff) and let
                            "Sr. Staff Software Engineer" through as a Senior req.
  \\s*(?!staff)            -- \\s* matched zero characters, so the lookahead landed
                            on " Staff" rather than "Staff" and let
                            "Senior Staff Software Engineer" through.

Both looked right. Both were caught only by running them against real titles off
the 2026-08-21 scan. Run this after any change to vetoes.py:

    python3 scripts/vetoes_cases.py
"""
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
from vetoes import check  # noqa: E402

# (title, company, description, expect_veto). Titles marked [real] came off the
# 2026-08-21 scan, where 92 of 266 kept roles were Staff-titled.
CASES = [
    # -- Staff and above: OUT. preferences.md 2026-08-21, hard filter.
    ("Staff Software Engineer, Build (Bazel)", "Airbnb", "", True),          # [real]
    ("Staff Backend Engineer - Ads Platform", "Airbnb", "", True),           # [real]
    ("Staff Software Engineer, Claude Code", "Anthropic", "", True),         # [real]
    ("Staff+ Software Engineer, Backend", "Anthropic", "", True),            # [real]
    ("Staff Software Engineer, Billing Platform", "Anthropic", "", True),    # [real]
    ("Senior Staff Software Engineer", "Acme", "", True),
    ("Sr. Staff Software Engineer, Platform", "Acme", "", True),
    ("Sr Staff Engineer", "Acme", "", True),
    ("Principal Engineer", "Acme", "", True),
    ("Distinguished Engineer", "Acme", "", True),
    ("Software Engineer L6", "Acme", "", True),

    # -- Multi-band reqs survive: levelling during the process is not applying to Staff.
    ("Senior / Staff Software Engineer", "Acme", "", False),
    ("Staff / Senior Software Engineer", "Acme", "", False),
    ("Senior or Staff Full-Stack Engineer", "Acme", "", False),

    # -- Senior: the target band.
    ("Senior Software Engineer", "Acme", "", False),
    ("Senior Fullstack Software Engineer, Growth", "Algolia", "", False),    # [real]
    ("Senior Software Engineer II, Frontend Platform", "Alloy", "", False),  # [real]
    ("Sr. Software Engineer, Developer Platform", "Acme", "", False),
    ("Senior Platform Engineer", "Acme", "", False),
    ("Software Engineer, Product", "Acme", "", False),

    # -- IAM as the team's charter: OUT (2026-08-21). Titles marked [real] had
    # already surfaced when Andre called it. A bare "Identity" must NOT match --
    # adjacent identity work is still in scope.
    ("Senior Software Engineer 2, IAM", "Drata", "", True),                          # [real]
    ("Senior Backend Engineer, IAM", "Reddit", "", True),                            # [real]
    ("Senior Backend Engineer, Identity & Access Management", "LaunchDarkly", "", True),   # [real]
    ("Senior Software Engineer, Identity and Access Management", "MongoDB", "", True),     # [real]
    ("Senior Software Engineer, Identity Platform", "Acme", "", False),
    ("Senior Software Engineer, Session Infrastructure", "Acme", "", False),
    # \biam\b must not fire on a city. Locations are never read, but prove it anyway.
    ("Senior Software Engineer, Payments", "Acme", "Based in Miami, FL.", False),
    ("Site Reliability Engineer (Senior or Staff), Storage", "MongoDB", "Miami; Toronto", False),

    # -- Other absolute rules.
    ("Senior Software Engineer", "Seeq", "", True),
    ("Senior Software Engineer", "Lockheed Martin", "", True),
    ("Engineering Manager, Platform", "Acme", "", True),
    ("Director of Engineering", "Acme", "", True),
    ("Senior iOS Engineer", "Acme", "", True),
    ("Senior Software Engineer", "Acme", "We require 12+ years of experience.", True),
    ("Senior Software Engineer", "Acme", "8-12 years of experience required.", True),
    ("Senior Software Engineer", "Acme", "5+ years of experience required.", False),
]


def main():
    failures = []
    for title, company, desc, expect_veto in CASES:
        reason = check({"company": company, "title": title, "description": desc})
        got = bool(reason)
        ok = got == expect_veto
        if not ok:
            failures.append((title, company, expect_veto, reason))
        mark = "ok  " if ok else "FAIL"
        verdict = f"veto: {reason}" if reason else "keep"
        print(f"  {mark}  {company:18} {title[:46]:46} {verdict}")

    print()
    if failures:
        print(f"{len(failures)} FAILED:")
        for title, company, expect_veto, reason in failures:
            want = "veto" if expect_veto else "keep"
            print(f"  wanted {want}: {company} — {title}  (got {reason or 'keep'})")
        return 1
    print(f"all {len(CASES)} passed")
    return 0


if __name__ == "__main__":
    sys.exit(main())
