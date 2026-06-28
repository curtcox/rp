#!/usr/bin/env bash
#
# gen-site.sh — build a static HTML report site under $SITE_DIR.
#
# Produces a dashboard (index.html) linking to:
#   - test output
#   - Go HTML coverage report + per-function coverage
#   - golangci-lint results
#   - cyclomatic & cognitive complexity reports
#
# The lint and complexity steps are reported even when they find issues; this
# script always exits 0 so report generation never blocks on findings. Gating
# happens separately in `make check` / the CI "checks" job.
set -uo pipefail

SITE_DIR="${SITE_DIR:-site}"
COVERPROFILE="${COVERPROFILE:-coverage.out}"
GOCYCLO_OVER="${GOCYCLO_OVER:-15}"
GOCOGNIT_OVER="${GOCOGNIT_OVER:-20}"
GOLANGCI_LINT="${GOLANGCI_LINT:-golangci-lint}"
GOCYCLO="${GOCYCLO:-gocyclo}"
GOCOGNIT="${GOCOGNIT:-gocognit}"

mkdir -p "$SITE_DIR"
GENERATED_AT="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo unknown)"

# HTML-escape stdin.
esc() { python3 -c 'import html,sys; sys.stdout.write(html.escape(sys.stdin.read()))'; }

# Wrap raw text output in a standalone themed HTML page.
# usage: text_page <title> <outfile> < input
text_page() {
	local title="$1" out="$2"
	{
		cat <<HTML
<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>${title} — rp</title>
<style>
  body { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
         margin: 0; background: #0d1117; color: #c9d1d9; }
  header { padding: 16px 24px; border-bottom: 1px solid #30363d; background: #161b22; }
  header a { color: #58a6ff; text-decoration: none; }
  h1 { font-size: 16px; margin: 0; }
  pre { padding: 16px 24px; white-space: pre-wrap; word-break: break-word;
        font-size: 13px; line-height: 1.5; margin: 0; }
</style></head><body>
<header><h1><a href="index.html">&larr; rp reports</a> / ${title}</h1></header>
<pre>
HTML
		esc
		cat <<HTML
</pre></body></html>
HTML
	} >"$out"
}

echo ">> tests + coverage"
go test -covermode=atomic -coverprofile="$COVERPROFILE" ./... >"$SITE_DIR/_test.txt" 2>&1
TEST_STATUS=$?
text_page "Tests" "$SITE_DIR/tests.html" <"$SITE_DIR/_test.txt"

COVERAGE_TOTAL="n/a"
if [ -f "$COVERPROFILE" ]; then
	go tool cover -html="$COVERPROFILE" -o "$SITE_DIR/coverage.html" || true
	go tool cover -func="$COVERPROFILE" >"$SITE_DIR/_coverfunc.txt" 2>&1 || true
	text_page "Coverage by function" "$SITE_DIR/coverage-func.html" <"$SITE_DIR/_coverfunc.txt"
	COVERAGE_TOTAL="$(grep -E '^total:' "$SITE_DIR/_coverfunc.txt" | awk '{print $NF}')"
fi

echo ">> golangci-lint"
"$GOLANGCI_LINT" run ./... >"$SITE_DIR/_lint.txt" 2>&1
LINT_STATUS=$?
if [ "$LINT_STATUS" -eq 0 ] && [ ! -s "$SITE_DIR/_lint.txt" ]; then
	echo "No issues reported by golangci-lint." >"$SITE_DIR/_lint.txt"
fi
text_page "golangci-lint" "$SITE_DIR/lint.html" <"$SITE_DIR/_lint.txt"

echo ">> complexity"
{
	echo "== Cyclomatic complexity (gocyclo), functions over ${GOCYCLO_OVER} =="
	"$GOCYCLO" -over "$GOCYCLO_OVER" -avg . 2>&1 || true
	echo
	echo "== Top 15 by cyclomatic complexity =="
	"$GOCYCLO" -top 15 -avg . 2>&1 || true
} >"$SITE_DIR/_cyclo.txt"
text_page "Cyclomatic complexity" "$SITE_DIR/complexity-cyclomatic.html" <"$SITE_DIR/_cyclo.txt"

{
	echo "== Cognitive complexity (gocognit), functions over ${GOCOGNIT_OVER} =="
	"$GOCOGNIT" -over "$GOCOGNIT_OVER" -avg . 2>&1 || true
	echo
	echo "== Top 15 by cognitive complexity =="
	"$GOCOGNIT" -top 15 -avg . 2>&1 || true
} >"$SITE_DIR/_cognit.txt"
text_page "Cognitive complexity" "$SITE_DIR/complexity-cognitive.html" <"$SITE_DIR/_cognit.txt"

# Status helpers for the dashboard cards.
test_badge() { [ "$TEST_STATUS" -eq 0 ] && echo "pass ok" || echo "fail bad"; }
lint_badge() { [ "$LINT_STATUS" -eq 0 ] && echo "pass ok" || echo "issues bad"; }

read -r TEST_LABEL TEST_CLASS <<<"$(test_badge)"
read -r LINT_LABEL LINT_CLASS <<<"$(lint_badge)"

echo ">> dashboard"
cat >"$SITE_DIR/index.html" <<HTML
<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>rp — test &amp; analysis reports</title>
<style>
  :root { color-scheme: dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
         margin: 0; background: #0d1117; color: #c9d1d9; }
  .wrap { max-width: 900px; margin: 0 auto; padding: 40px 24px; }
  h1 { margin: 0 0 4px; font-size: 28px; }
  .sub { color: #8b949e; margin: 0 0 32px; font-size: 14px; }
  .grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
  .card { display: block; padding: 20px; border: 1px solid #30363d; border-radius: 10px;
          background: #161b22; text-decoration: none; color: inherit; transition: border-color .15s; }
  .card:hover { border-color: #58a6ff; }
  .card h2 { margin: 0 0 6px; font-size: 16px; color: #58a6ff; }
  .card p { margin: 0; font-size: 13px; color: #8b949e; }
  .badge { display: inline-block; margin-top: 12px; padding: 2px 10px; border-radius: 20px;
           font-size: 12px; font-weight: 600; }
  .ok  { background: #1f6f3f; color: #fff; }
  .bad { background: #b62324; color: #fff; }
  .num { background: #1f6feb; color: #fff; }
  footer { margin-top: 40px; color: #6e7681; font-size: 12px; }
  footer code { color: #8b949e; }
</style></head><body>
<div class="wrap">
  <h1>rp — documentation &amp; reports</h1>
  <p class="sub">A local, terminal-first, evidence-auditable resource planner.</p>
  <div class="grid">
    <a class="card" href="docs/index.html">
      <h2>Documentation</h2>
      <p>Guides, CLI reference, concepts, config, and tutorials.</p>
      <span class="badge num">docs</span>
    </a>
    <a class="card" href="tests.html">
      <h2>Tests</h2>
      <p>Full <code>go test ./...</code> output.</p>
      <span class="badge ${TEST_CLASS}">${TEST_LABEL}</span>
    </a>
    <a class="card" href="coverage.html">
      <h2>Coverage</h2>
      <p>Annotated HTML source coverage. <a href="coverage-func.html">Per-function&nbsp;&rarr;</a></p>
      <span class="badge num">${COVERAGE_TOTAL}</span>
    </a>
    <a class="card" href="lint.html">
      <h2>golangci-lint</h2>
      <p>Static analysis (govet, staticcheck, ineffassign, unused, misspell, unconvert).</p>
      <span class="badge ${LINT_CLASS}">${LINT_LABEL}</span>
    </a>
    <a class="card" href="complexity-cyclomatic.html">
      <h2>Cyclomatic complexity</h2>
      <p>gocyclo — functions over ${GOCYCLO_OVER}, plus top offenders.</p>
    </a>
    <a class="card" href="complexity-cognitive.html">
      <h2>Cognitive complexity</h2>
      <p>gocognit — functions over ${GOCOGNIT_OVER}, plus top offenders.</p>
    </a>
  </div>
  <footer>
    Generated ${GENERATED_AT} from commit <code>${COMMIT}</code>.
  </footer>
</div>
</body></html>
HTML

# Render the prose documentation (docs/*.md) into $SITE_DIR/docs.
DOCS_DIR="${DOCS_DIR:-docs}"
RENDER_DOCS="${RENDER_DOCS:-scripts/render-docs.py}"
if [ -d "$DOCS_DIR" ]; then
	echo ">> docs"
	GENERATED_AT="$GENERATED_AT" COMMIT="$COMMIT" \
		python3 "$RENDER_DOCS" "$DOCS_DIR" "$SITE_DIR/docs"
fi

# Clean up intermediate text files.
rm -f "$SITE_DIR"/_*.txt

echo "Site written to $SITE_DIR/ (coverage: $COVERAGE_TOTAL, tests: $TEST_LABEL, lint: $LINT_LABEL)"
exit 0
