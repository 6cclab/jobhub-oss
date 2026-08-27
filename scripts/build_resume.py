#!/usr/bin/env python3
"""Render a tailored resume.md into resume.html + resume.pdf using templates/.

Replaces the throwaway per-company build scripts. Reads the markdown, converts it
to the template's HTML structure, inlines the CSS, renders with WeasyPrint, and
reports page count and last-page fill so the length rule in resume-style.md can be
checked rather than guessed.

Usage:
    python3 scripts/build_resume.py user/tailored/headway/senior-fullstack-software-engineer
    python3 scripts/build_resume.py <dir> --check    # report pages/fill, write nothing

Expected markdown shape:
    # Name
    **Title** | Location | email | phone
    links
    ## Summary          -> paragraph
    ## Skills           -> "- **Label:** a, b, c" lines
    ## Experience       -> "### Company — Location", bold title/date lines, "- " bullets
    ## <Other>          -> section with "- " bullets (e.g. Independent Engineering)
    ## Education        -> paragraph
"""

import re
import sys
from pathlib import Path

from weasyprint import HTML

ROOT = Path(__file__).resolve().parent.parent
EXTRA_CSS = """
.subsection { font-size: 10pt; font-weight: 700; font-style: italic;
              margin-top: 5pt; margin-bottom: 0; }
    .nb { white-space: nowrap; }
"""


def inline(text):
    """Markdown inline -> HTML. Bold, links, and bare em-dashes preserved."""
    text = re.sub(r"\[([^\]]+)\]\(([^)]+)\)", r'<a href="\2">\1</a>', text)
    text = re.sub(r"\*\*(.+?)\*\*", r"<strong>\1</strong>", text)
    text = keep_hyphens_intact(text)
    return text


def keep_hyphens_intact(text):
    """Stop the renderer breaking a line at a hyphen.

    A hyphen is a break opportunity, so "cross-functionally" can render as
    "cross-" / "functionally" across two lines. That looks fine on the page, but
    PDF text extraction rejoins it *without* the hyphen -- an ATS then reads
    "crossfunctionally" and the keyword match for "cross-functional" fails.
    Found 2026-08-20 across 9 resumes, including "on-call" on two already-submitted
    applications that named on-call as a requirement.

    Wrapping each hyphenated token in white-space:nowrap keeps the ASCII hyphen
    (so extraction is correct) and removes the break opportunity.
    """
    return re.sub(
        r"(?<![>\w])([A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+)(?![\w<])",
        r'<span class="nb">\1</span>',
        text,
    )


def parse(md):
    """Split the markdown into ordered (kind, payload) blocks."""
    blocks, section, buf = [], None, []

    def flush():
        if section and buf:
            blocks.append((section, list(buf)))
        buf.clear()

    for line in md.split("\n"):
        if line.startswith("## "):
            flush()
            section = line[3:].strip()
        elif section:
            buf.append(line)
    flush()
    return blocks


# A role line is "Title — dates", with or without bold, identified by the year in the
# dates half. Requiring a year keeps a subsection label that happens to contain an
# em-dash from being mistaken for a job title.
ROLE_LINE = re.compile(r"—.*\b(?:19|20)\d{2}\b")


def render_experience(lines):
    out, bullets, dropped = [], [], []

    def flush():
        if bullets:
            out.append("<ul>" + "".join(f"<li>{inline(b)}</li>" for b in bullets) + "</ul>")
            bullets.clear()

    for line in lines:
        s = line.strip()
        if not s or s == "---":
            continue
        if s.startswith("### "):
            flush()
            head = s[4:]
            company, _, loc = head.partition("—")
            out.append(
                f'<div class="job-header"><span class="company">{company.strip()}</span>'
                f'<span class="location">{loc.strip()}</span></div>'
            )
            out.append('<div class="job-titles">')
        elif s.startswith("- "):
            bullets.append(s[2:])
        elif ROLE_LINE.search(s):
            # A role line: "**Title** — dates" OR "Title — dates".
            #
            # **Bold is optional, and that is the whole point.** This branch used to
            # require `s.startswith("**")`, because master-resume.md bolds these lines.
            # Every *tailored* resume writes them unbolded, so they matched no branch and
            # were dropped in silence -- shipping PDFs with no job titles and no
            # employment dates at all. Found 2026-08-21 on the Courier Health and Datadog
            # resumes, both of which had already passed the preflight. Do not narrow this
            # back to bold-only.
            title, _, dates = s.partition("—")
            title = re.sub(r"\*+", "", title).strip()
            out.append(
                f'<div class="title-row"><span class="title">{title}</span>'
                f'<span class="dates">{dates.strip()}</span></div>'
            )
        elif s.startswith("*"):
            # A subsection label, bold (**Product engineering**) or italic
            # (*Product engineering*). Italic is what the tailored resumes use, and it
            # was silently dropped for the same reason as the role lines above.
            #
            # flush() FIRST: a subsection ends the preceding bullet list. Without it,
            # bullets accumulated across every subsection and were emitted as one <ul>
            # at the end, so all five labels stacked together above a single flat list.
            #
            # NB: this was r"[*]{{2}}" -- a doubled brace in a raw string matches the
            # literal text "{{2}}", not two asterisks, so the ** leaked into the PDF.
            # Fixed 2026-08-20.
            # REFUSED as of 2026-08-21. Andre: "Again adding boldened sub sections."
            # "Again" because he had said it before, nothing was written down, and it came
            # back on every new resume -- six live resumes were carrying eighteen of them
            # when it was finally caught, one of them already sent.
            #
            # This raises rather than skipping the line, because silently dropping it
            # would reintroduce the exact failure that shipped two resumes with no job
            # titles (see the ROLE_LINE branch above). Refusing to build is the control:
            # a rule that only lives in a style file has already proven it does not hold.
            raise SystemExit(
                f"{'':>2}subsection labels are not used -- delete this line: {s!r}\n"
                f"{'':>2}bullets run continuously under the company and title block.\n"
                f"{'':>2}see user/resume-style.md, 'No subsection labels inside Experience'"
            )
        else:
            # Anything unrecognised. NEVER drop it quietly -- silent drops are exactly
            # how the missing-dates defect survived for a whole session.
            dropped.append(s)
    flush()

    if dropped:
        print("WARNING: lines in ## Experience matched no rule and were NOT rendered:",
              file=sys.stderr)
        for s in dropped:
            print(f"    {s[:100]}", file=sys.stderr)
    # balance job-titles divs
    html = "\n".join(out)
    html = html.replace('<div class="job-titles">', '<div class="job-titles">', 1)
    opens = html.count('<div class="job-titles">')
    html += "</div>" * 0  # title rows are self-contained; wrapper closed below
    # Close the job-titles wrapper at the first thing that follows the title rows. That is
    # a subsection label when the resume uses them, not always a <ul> -- without the
    # subsection alternative here, every label was swallowed into the titles block.
    return re.sub(
        r'(<div class="job-titles">)((?:(?!</div>).)*?)'
        r'(?=<ul>|<div class="subsection">|<div class="job-header">|$)',
        r"\1\2</div>", html, flags=re.S)


def build(target: Path):
    md = (target / "resume.md").read_text()
    cfg = {}
    for line in (ROOT / "user" / "config.yaml").read_text().split("\n"):
        m = re.match(r'^(\w+):\s*"?([^"#]*?)"?\s*(?:#.*)?$', line)
        if m and m.group(2):
            cfg[m.group(1)] = m.group(2).strip()
    css = (ROOT / "templates" / "resume.css").read_text() + EXTRA_CSS
    shell = (ROOT / "templates" / "resume.html").read_text()

    # The headline. master-resume.md writes "**Staff Software Engineer** | New Jersey | ...",
    # tailored resumes write "**Senior Software Engineer**" alone on its own line. The
    # trailing pipe used to be mandatory, so every tailored resume fell through to the
    # default and shipped with the WRONG headline -- "Software Engineer" on a Senior req,
    # while preferences.md requires the headline to match the posting. Found 2026-08-21.
    title_line = re.search(r"^\*\*(.+?)\*\*\s*(?:\||$)", md, re.M)
    if not title_line:
        print("ERROR: no headline found. Expected a line like '**Senior Software Engineer**'"
              " near the top of resume.md.", file=sys.stderr)
        raise SystemExit(1)
    title = title_line.group(1).strip()

    parts = []
    for name, lines in parse(md):
        body = [l for l in lines if l.strip() and l.strip() != "---"]
        parts.append(f'<div class="section-title">{name}</div>')
        if name.lower() == "experience":
            parts.append(render_experience(lines))
        elif name.lower() == "skills":
            rows = "".join(f"<p>{inline(l.strip()[2:])}</p>" for l in body if l.strip().startswith("- "))
            parts.append(f'<div class="skills">{rows}</div>')
        elif name.lower() == "education":
            parts.append(f'<div class="education"><p>{inline(" ".join(body))}</p></div>')
        elif any(l.strip().startswith("- ") for l in body):
            items = "".join(f"<li>{inline(l.strip()[2:])}</li>" for l in body if l.strip().startswith("- "))
            parts.append(f"<ul>{items}</ul>")
        else:
            parts.append(f"<p>{inline(' '.join(body))}</p>")

    links = (f'<a href="https://{cfg["linkedin"]}">{cfg["linkedin"]}</a> | '
             f'<a href="https://{cfg["github"]}">{cfg["github"]}</a>')
    html = shell.replace('<link rel="stylesheet" href="resume.css">', f"<style>\n{css}\n</style>")
    for k, v in {
        "{{NAME}}": cfg["name"], "{{TITLE}}": title, "{{LOCATION}}": cfg["location"],
        "{{EMAIL}}": cfg["email"], "{{PHONE}}": cfg["phone"], "{{LINKS}}": links,
        "{{CONTENT}}": "\n".join(parts),
    }.items():
        html = html.replace(k, v)
    return html


def report(html_path: Path):
    doc = HTML(filename=str(html_path)).render()
    n = len(doc.pages)
    fill = None
    if n > 1:
        page, ys = doc.pages[-1], []

        def walk(b):
            for c in getattr(b, "children", []) or []:
                if hasattr(c, "text") and c.text:
                    ys.append(getattr(c, "position_y", 0) + getattr(c, "height", 0))
                walk(c)

        walk(page._page_box)
        if ys:
            fill = max(ys) / page.height * 100
    return n, fill


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 1
    target = Path(sys.argv[1])
    if not (target / "resume.md").exists():
        print(f"ERROR: no resume.md in {target}", file=sys.stderr)
        return 1

    html = build(target)
    (target / "resume.html").write_text(html)
    pages, fill = report(target / "resume.html")

    if "--check" not in sys.argv:
        HTML(filename=str(target / "resume.html")).write_pdf(str(target / "resume.pdf"))

    msg = f"{target.name}: {pages} page(s)"
    if fill is not None:
        msg += f", last page {fill:.0f}% full"
        if fill < 33:
            msg += "  <- BELOW the one-third rule in resume-style.md; cut to one page or add content"
    print(msg)
    return 0


if __name__ == "__main__":
    sys.exit(main())
