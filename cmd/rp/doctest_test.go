package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Doctest harness: the prose docs under ./docs carry runnable examples that are
// both illustrations and tests. Each example is a fenced `console` transcript
// preceded by an `rp-example` HTML comment, e.g.:
//
//	<!-- rp-example: id=version cwd=empty status=ready -->
//	```console
//	$ rp version
//	rp rp.dev/v0.1
//	```
//
// Attributes:
//   - id      unique identifier (used as the subtest name)
//   - cwd     "empty" (fresh temp dir), "fixture" (a sandboxed copy of
//             example-project, git-initialised), "repro" (a sandboxed copy of
//             the reproducible-build fixture), "conform" (data-conform),
//             "translate" (translate-doc), "gate" (release-gate), or
//             "flaky" (flaky-fix); defaults to "empty"
//   - status  "ready" (executed and asserted) or "todo" (a placeholder that is
//             counted and skipped); defaults to "ready"
//   - exit    expected exit code of the LAST command in the block (default 0)
//
// Inside the block, lines beginning with "$ " are commands; the leading token
// `rp` is rewired to the freshly built binary. Lines that follow a command are
// its expected combined (stdout+stderr) output, until the next "$ " or the end
// of the block. Output is normalised (see redact) so volatile tokens — run and
// plan ids, the sandbox root, config hashes — do not make examples brittle.
//
// To promote a placeholder: capture the real output, paste it in, and flip
// status=todo to status=ready. See docs/README.md.

var rpBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "rp-doctest-bin")
	if err != nil {
		fmt.Fprintln(os.Stderr, "doctest: mkdtemp:", err)
		os.Exit(1)
	}
	bin := filepath.Join(dir, "rp")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "doctest: build failed: %v\n%s", err, out)
		os.Exit(1)
	}
	rpBinary = bin
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

type docCommand struct {
	args []string
	want string
}

type docExample struct {
	id       string
	cwd      string
	status   string
	exit     int
	file     string
	line     int
	commands []docCommand
}

var exampleAttrRe = regexp.MustCompile(`<!--\s*rp-example:\s*(.*?)\s*-->`)

func TestDocExamples(t *testing.T) {
	root := filepath.Join("..", "..", "docs")
	if _, err := os.Stat(root); err != nil {
		t.Skip("docs/ not present")
	}
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var ready, todo int
	seen := map[string]string{}
	for _, f := range files {
		examples, perr := parseDocExamples(f)
		if perr != nil {
			t.Errorf("%s: %v", f, perr)
			continue
		}
		for _, ex := range examples {
			if prev, dup := seen[ex.id]; dup {
				t.Errorf("%s:%d: duplicate example id %q (also in %s)", ex.file, ex.line, ex.id, prev)
				continue
			}
			seen[ex.id] = fmt.Sprintf("%s:%d", ex.file, ex.line)
			ex := ex
			t.Run(ex.id, func(t *testing.T) {
				switch ex.status {
				case "todo":
					todo++
					t.Skipf("placeholder (status=todo) at %s:%d", ex.file, ex.line)
				case "ready":
					ready++
					runDocExample(t, ex)
				default:
					t.Fatalf("%s:%d: unknown status %q", ex.file, ex.line, ex.status)
				}
			})
		}
	}
	t.Logf("doc examples: %d ready, %d todo placeholders across %d files", ready, todo, len(files))
}

// parseDocExamples extracts every rp-example block from a markdown file.
func parseDocExamples(file string) ([]docExample, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	var out []docExample
	for i := 0; i < len(lines); i++ {
		m := exampleAttrRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		ex, err := attrsToExample(m[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", i+1, err)
		}
		ex.file = file
		ex.line = i + 1
		// Find the opening fence, skipping blank lines.
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
			return nil, fmt.Errorf("line %d: example %q is not followed by a fenced block", i+1, ex.id)
		}
		// Fences inside list items are indented; strip that indent from the body
		// so `$ ` command detection works regardless of nesting.
		indent := lines[j][:len(lines[j])-len(strings.TrimLeft(lines[j], " "))]
		j++
		var body []string
		for j < len(lines) && strings.TrimSpace(lines[j]) != "```" {
			body = append(body, strings.TrimPrefix(lines[j], indent))
			j++
		}
		ex.commands = parseTranscript(body)
		if len(ex.commands) == 0 {
			return nil, fmt.Errorf("line %d: example %q has no `$` command lines", i+1, ex.id)
		}
		out = append(out, ex)
		i = j
	}
	return out, nil
}

func attrsToExample(attrs string) (docExample, error) {
	ex := docExample{cwd: "empty", status: "ready"}
	for _, field := range strings.Fields(attrs) {
		k, v, ok := strings.Cut(field, "=")
		if !ok {
			return ex, fmt.Errorf("malformed attribute %q (want key=value)", field)
		}
		switch k {
		case "id":
			ex.id = v
		case "cwd":
			ex.cwd = v
		case "status":
			ex.status = v
		case "exit":
			n, err := parseIntStrict(v)
			if err != nil {
				return ex, fmt.Errorf("bad exit %q: %w", v, err)
			}
			ex.exit = n
		default:
			return ex, fmt.Errorf("unknown attribute %q", k)
		}
	}
	if ex.id == "" {
		return ex, fmt.Errorf("missing id")
	}
	return ex, nil
}

func parseIntStrict(s string) (int, error) {
	n := 0
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(r-'0')
	}
	return n, nil
}

// parseTranscript splits a console block into commands plus their expected output.
func parseTranscript(body []string) []docCommand {
	var cmds []docCommand
	var cur *docCommand
	var want []string
	flush := func() {
		if cur != nil {
			cur.want = strings.TrimRight(strings.Join(want, "\n"), "\n")
			cmds = append(cmds, *cur)
		}
		want = nil
	}
	for _, ln := range body {
		if strings.HasPrefix(ln, "$ ") {
			flush()
			c := docCommand{args: splitArgs(strings.TrimPrefix(ln, "$ "))}
			cur = &c
			continue
		}
		if cur != nil {
			want = append(want, ln)
		}
	}
	flush()
	return cmds
}

// splitArgs is a minimal shell-ish splitter: whitespace separates tokens, and
// double quotes group a token (the only quoting the docs use).
func splitArgs(s string) []string {
	var args []string
	var b strings.Builder
	inQuote, has := false, false
	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			has = true
		case r == ' ' && !inQuote:
			if has {
				args = append(args, b.String())
				b.Reset()
				has = false
			}
		default:
			b.WriteRune(r)
			has = true
		}
	}
	if has {
		args = append(args, b.String())
	}
	return args
}

func runDocExample(t *testing.T, ex docExample) {
	t.Helper()
	dir := exampleSandbox(t, ex.cwd)
	for idx, c := range ex.commands {
		if len(c.args) == 0 || c.args[0] != "rp" {
			t.Fatalf("%s:%d: command %d must start with `rp`, got %v", ex.file, ex.line, idx+1, c.args)
		}
		cmd := exec.Command(rpBinary, c.args[1:]...)
		cmd.Dir = dir
		raw, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			ee := &exec.ExitError{}
			if ok := asExitError(err, ee); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("%s:%d: running %v: %v", ex.file, ex.line, c.args, err)
			}
		}
		last := idx == len(ex.commands)-1
		wantCode := 0
		if last {
			wantCode = ex.exit
		}
		if code != wantCode {
			t.Fatalf("%s:%d: command %v exited %d, want %d\noutput:\n%s",
				ex.file, ex.line, c.args, code, wantCode, raw)
		}
		got := redact(string(raw), dir)
		want := redact(c.want, dir)
		if got != want {
			t.Errorf("%s:%d: output mismatch for %v\n--- want ---\n%s\n--- got ---\n%s",
				ex.file, ex.line, c.args, want, got)
		}
	}
}

func asExitError(err error, into *exec.ExitError) bool {
	if ee, ok := err.(*exec.ExitError); ok {
		*into = *ee
		return true
	}
	return false
}

// exampleSandbox builds the working directory an example runs in.
func exampleSandbox(t *testing.T, cwd string) string {
	t.Helper()
	dir := t.TempDir()
	switch cwd {
	case "empty":
		// nothing to do
	case "fixture":
		exampleRoot := filepath.Join("..", "..", "example-project")
		if _, err := os.Stat(filepath.Join(exampleRoot, ".rp", "planner.yaml")); err != nil {
			t.Skip("example-project fixture not present")
		}
		copyDir(t, exampleRoot, dir)
		gitInitFixture(t, dir)
	case "repro":
		exampleRoot := filepath.Join("..", "..", "reproducible-build")
		if _, err := os.Stat(filepath.Join(exampleRoot, ".rp", "planner.yaml")); err != nil {
			t.Skip("reproducible-build fixture not present")
		}
		copyDir(t, exampleRoot, dir)
	case "conform":
		copyExampleFixture(t, dir, "data-conform")
	case "translate":
		copyExampleFixture(t, dir, "translate-doc")
	case "gate":
		copyExampleFixture(t, dir, "release-gate")
	case "flaky":
		copyExampleFixture(t, dir, "flaky-fix")
	default:
		t.Fatalf("unknown cwd %q (want empty|fixture|repro|conform|translate|gate|flaky)", cwd)
	}
	return dir
}

func copyExampleFixture(t *testing.T, dir, name string) {
	t.Helper()
	exampleRoot := filepath.Join("..", "..", name)
	if _, err := os.Stat(filepath.Join(exampleRoot, ".rp", "planner.yaml")); err != nil {
		t.Skipf("%s fixture not present", name)
	}
	copyDir(t, exampleRoot, dir)
}

func gitInitFixture(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init"},
		{"add", "-A"},
		{"-c", "user.email=rp-test@example.com", "-c", "user.name=rp test", "commit", "-m", "initial"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v failed: %v\n%s", args, err, out)
		}
	}
}

var (
	runIDRe  = regexp.MustCompile(`run-\d{8}T\d{6}\.\d+Z`)
	planIDRe = regexp.MustCompile(`plan-\d{8}T\d{6}\.\d+Z`)
	hashRe   = regexp.MustCompile(`Config: [0-9a-f]{8,}`)
)

// redact normalises volatile tokens so examples stay stable across machines and
// runs. The sandbox root is collapsed to <root>; ids and hashes to placeholders.
func redact(s, dir string) string {
	if dir != "" {
		s = strings.ReplaceAll(s, dir, "<root>")
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			s = strings.ReplaceAll(s, resolved, "<root>")
		}
	}
	s = runIDRe.ReplaceAllString(s, "run-<id>")
	s = planIDRe.ReplaceAllString(s, "plan-<id>")
	s = hashRe.ReplaceAllString(s, "Config: <hash>")
	return strings.TrimRight(s, "\n")
}
