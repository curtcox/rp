#!/usr/bin/env python3
"""render-docs.py — render docs/*.md to a themed HTML site under <out>.

Dependency-free (Python 3 stdlib only). Supports the Markdown subset used by the
rp docs: ATX headings, fenced code blocks, pipe tables, blockquotes, ordered and
unordered lists, horizontal rules, paragraphs, and inline code/bold/links.
Relative `.md` links are rewritten to `.html`. A curated sidebar (built from
NAV below) and a link back to the Reports dashboard are added to every page.

Usage: render-docs.py <docs_dir> <out_dir>
  <out_dir> is typically <site>/docs; the Reports dashboard is <site>/index.html.
"""

import html
import os
import re
import sys

# Curated navigation: (section title, [(relative md path, label), ...]).
NAV = [
    ("Overview", [
        ("index.md", "Home"),
        ("getting-started.md", "Getting started"),
    ]),
    ("CLI reference", [
        ("cli/index.md", "Overview"),
        ("cli/init.md", "init"),
        ("cli/scaffold.md", "scaffold (capability/goal/policy/add)"),
        ("cli/resources.md", "resources / resource"),
        ("cli/plan.md", "plan"),
        ("cli/achieve.md", "achieve"),
        ("cli/exec.md", "exec"),
        ("cli/evidence.md", "evidence"),
        ("cli/why.md", "why"),
        ("cli/trace.md", "trace"),
        ("cli/observe.md", "observe"),
        ("cli/attest.md", "attest"),
        ("cli/runs.md", "runs (audit/replay/replan/rerun)"),
    ]),
    ("Concepts", [
        ("concepts/overview.md", "The model"),
        ("concepts/glossary.md", "Glossary"),
    ]),
    ("Under the hood", [
        ("internals/index.md", "How rp works"),
        ("internals/planning.md", "Backward planning"),
        ("internals/execution.md", "Execution & JIT re-planning"),
        ("internals/evidence.md", "Evidence & confidence"),
        ("internals/hashing-and-policy.md", "Hashing, policy & budgets"),
    ]),
    ("Config", [
        ("config/reference.md", ".rp/ reference"),
        ("config/policy.md", "Policy"),
    ]),
    ("Tutorials", [
        ("tutorials/bugfix-walkthrough.md", "Bugfix walkthrough"),
        ("tutorials/reproducible-build.md", "Reproducible build"),
        ("tutorials/writing-a-capability.md", "Writing a capability"),
        ("tutorials/defining-a-goal.md", "Defining a goal"),
    ]),
    ("Contributing", [
        ("README.md", "Docs & examples"),
    ]),
]

PAGE = """<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{title} — rp docs</title>
<style>
  :root {{ color-scheme: dark; }}
  * {{ box-sizing: border-box; }}
  body {{ margin: 0; background: #0d1117; color: #c9d1d9;
         font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif; }}
  a {{ color: #58a6ff; text-decoration: none; }}
  a:hover {{ text-decoration: underline; }}
  .layout {{ display: grid; grid-template-columns: 280px 1fr; min-height: 100vh; }}
  nav {{ border-right: 1px solid #30363d; background: #161b22; padding: 20px 16px;
        position: sticky; top: 0; align-self: start; height: 100vh; overflow-y: auto; }}
  nav .brand {{ font-weight: 700; font-size: 18px; margin: 0 0 4px; }}
  nav .reports {{ font-size: 13px; margin: 0 0 18px; }}
  nav h3 {{ font-size: 11px; text-transform: uppercase; letter-spacing: .06em;
           color: #8b949e; margin: 18px 0 6px; }}
  nav ul {{ list-style: none; margin: 0; padding: 0; }}
  nav li {{ margin: 2px 0; }}
  nav a {{ display: block; padding: 4px 8px; border-radius: 6px; font-size: 14px; color: #c9d1d9; }}
  nav a:hover {{ background: #21262d; text-decoration: none; }}
  nav a.active {{ background: #1f6feb; color: #fff; }}
  main {{ padding: 32px 48px; max-width: 900px; }}
  main h1 {{ margin-top: 0; }}
  h1, h2, h3 {{ line-height: 1.25; }}
  h2 {{ border-bottom: 1px solid #21262d; padding-bottom: 6px; margin-top: 32px; }}
  code {{ background: #161b22; border: 1px solid #30363d; border-radius: 4px;
         padding: 1px 5px; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
         font-size: 85%; }}
  pre {{ background: #161b22; border: 1px solid #30363d; border-radius: 8px;
        padding: 14px 16px; overflow-x: auto; }}
  pre code {{ background: none; border: 0; padding: 0; font-size: 13px; line-height: 1.5; }}
  table {{ border-collapse: collapse; margin: 16px 0; display: block; overflow-x: auto; }}
  th, td {{ border: 1px solid #30363d; padding: 6px 12px; text-align: left; }}
  th {{ background: #161b22; }}
  blockquote {{ border-left: 3px solid #30363d; margin: 16px 0; padding: 2px 16px; color: #8b949e; }}
  hr {{ border: 0; border-top: 1px solid #30363d; margin: 28px 0; }}
  footer {{ margin-top: 48px; padding-top: 16px; border-top: 1px solid #21262d;
           color: #6e7681; font-size: 12px; }}
  @media (max-width: 720px) {{ .layout {{ grid-template-columns: 1fr; }}
    nav {{ position: static; height: auto; }} main {{ padding: 24px; }} }}
</style></head><body>
<div class="layout">
<nav>
  <p class="brand">rp docs</p>
  <p class="reports"><a href="{reports}">&larr; Test &amp; analysis reports</a></p>
  {navhtml}
</nav>
<main>
{content}
<footer>Generated {generated} from commit <code>{commit}</code>.</footer>
</main>
</div>
</body></html>
"""

INLINE_CODE = re.compile(r"`([^`]+)`")
BOLD = re.compile(r"\*\*(.+?)\*\*")
LINK = re.compile(r"\[(.+?)\]\(([^)]+)\)")


def rewrite_link(url):
    if url.startswith("http://") or url.startswith("https://") or url.startswith("#"):
        return url
    if ".md" in url:
        anchor = ""
        if "#" in url:
            url, anchor = url.split("#", 1)
            anchor = "#" + anchor
        if url.endswith(".md"):
            url = url[:-3] + ".html"
        return url + anchor
    return url


def inline(text):
    text = html.escape(text, quote=False)
    text = INLINE_CODE.sub(lambda m: "<code>" + m.group(1) + "</code>", text)
    text = BOLD.sub(r"<strong>\1</strong>", text)
    text = LINK.sub(lambda m: '<a href="%s">%s</a>' % (rewrite_link(m.group(2)), m.group(1)), text)
    return text


def is_table_sep(line):
    s = line.strip().strip("|")
    return bool(s) and all(c in "-:| " for c in s)


def split_row(line):
    return [c.strip() for c in line.strip().strip("|").split("|")]


def render(md):
    lines = md.split("\n")
    out = []
    i, n = 0, len(lines)
    while i < n:
        line = lines[i]
        stripped = line.strip()

        # Fenced code block (any indent). A run of 3+ backticks opens; the close
        # must be a backtick run at least as long (so ````md blocks that contain
        # ``` are handled correctly).
        fence = re.match(r"(`{3,})(.*)", stripped)
        if fence:
            marker, lang = fence.group(1), fence.group(2).strip()
            closer = re.compile(r"`{%d,}\s*$" % len(marker))
            i += 1
            buf = []
            while i < n and not closer.fullmatch(lines[i].strip()):
                buf.append(lines[i])
                i += 1
            i += 1  # closing fence
            cls = ' class="language-%s"' % lang if lang else ""
            body = html.escape("\n".join(buf), quote=False)
            out.append("<pre><code%s>%s</code></pre>" % (cls, body))
            continue

        if not stripped:
            i += 1
            continue

        if stripped in ("---", "***", "___"):
            out.append("<hr>")
            i += 1
            continue

        m = re.match(r"(#{1,6})\s+(.*)", stripped)
        if m:
            level = len(m.group(1))
            text = m.group(2)
            slug = re.sub(r"[^a-z0-9]+", "-", text.lower()).strip("-")
            out.append('<h%d id="%s">%s</h%d>' % (level, slug, inline(text), level))
            i += 1
            continue

        # Pipe table.
        if "|" in stripped and i + 1 < n and is_table_sep(lines[i + 1]):
            header = split_row(line)
            i += 2
            rows = []
            while i < n and "|" in lines[i] and lines[i].strip():
                rows.append(split_row(lines[i]))
                i += 1
            out.append("<table><thead><tr>" +
                       "".join("<th>%s</th>" % inline(c) for c in header) +
                       "</tr></thead><tbody>")
            for r in rows:
                out.append("<tr>" + "".join("<td>%s</td>" % inline(c) for c in r) + "</tr>")
            out.append("</tbody></table>")
            continue

        # Blockquote.
        if stripped.startswith(">"):
            buf = []
            while i < n and lines[i].strip().startswith(">"):
                buf.append(lines[i].strip()[1:].lstrip())
                i += 1
            out.append("<blockquote>%s</blockquote>" % inline(" ".join(buf)))
            continue

        # Unordered list.
        if re.match(r"[-*]\s+", stripped):
            buf = []
            while i < n and re.match(r"\s*[-*]\s+", lines[i]):
                buf.append(re.sub(r"\s*[-*]\s+", "", lines[i], count=1))
                i += 1
            out.append("<ul>" + "".join("<li>%s</li>" % inline(x) for x in buf) + "</ul>")
            continue

        # Ordered list.
        if re.match(r"\d+\.\s+", stripped):
            buf = []
            while i < n and re.match(r"\s*\d+\.\s+", lines[i]):
                buf.append(re.sub(r"\s*\d+\.\s+", "", lines[i], count=1))
                i += 1
            out.append("<ol>" + "".join("<li>%s</li>" % inline(x) for x in buf) + "</ol>")
            continue

        # Paragraph (gather until blank or block start).
        buf = []
        while i < n and lines[i].strip() and not lines[i].strip().startswith(("```", "#", ">")) \
                and not re.match(r"[-*]\s+|\d+\.\s+", lines[i].strip()):
            buf.append(lines[i].strip())
            i += 1
        out.append("<p>%s</p>" % inline(" ".join(buf)))
    return "\n".join(out)


def page_title(md, fallback):
    for line in md.split("\n"):
        m = re.match(r"#\s+(.*)", line.strip())
        if m:
            return m.group(1)
    return fallback


def build_nav(docs_dir, current_rel):
    cur_html = current_rel[:-3] + ".html"
    sections = []
    for title, items in NAV:
        lis = []
        for md_path, label in items:
            if not os.path.exists(os.path.join(docs_dir, md_path)):
                continue
            target_html = md_path[:-3] + ".html"
            href = os.path.relpath(target_html, os.path.dirname(cur_html)) or "."
            active = " active" if md_path[:-3] == current_rel[:-3] else ""
            lis.append('<li><a class="%s" href="%s">%s</a></li>' %
                       (active.strip(), href, html.escape(label)))
        if lis:
            sections.append("<h3>%s</h3><ul>%s</ul>" % (html.escape(title), "".join(lis)))
    return "\n  ".join(sections)


def main():
    if len(sys.argv) != 3:
        sys.exit("usage: render-docs.py <docs_dir> <out_dir>")
    docs_dir, out_dir = sys.argv[1], sys.argv[2]
    generated = os.environ.get("GENERATED_AT", "")
    commit = os.environ.get("COMMIT", "unknown")

    md_files = []
    for root, _, files in os.walk(docs_dir):
        for f in sorted(files):
            if f.endswith(".md"):
                md_files.append(os.path.relpath(os.path.join(root, f), docs_dir))

    for rel in md_files:
        with open(os.path.join(docs_dir, rel), encoding="utf-8") as fh:
            md = fh.read()
        out_rel = rel[:-3] + ".html"
        # Reports dashboard lives one level above out_dir (the site root).
        depth = out_rel.count("/") + 1
        reports = "../" * depth + "index.html"
        page = PAGE.format(
            title=html.escape(page_title(md, rel)),
            reports=reports,
            navhtml=build_nav(docs_dir, rel),
            content=render(md),
            generated=html.escape(generated),
            commit=html.escape(commit),
        )
        dest = os.path.join(out_dir, out_rel)
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        with open(dest, "w", encoding="utf-8") as fh:
            fh.write(page)

    print("rendered %d doc pages to %s" % (len(md_files), out_dir))


if __name__ == "__main__":
    main()
