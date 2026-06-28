# AGENTS.md

`rp` is a local, terminal-first, **evidence-auditable resource planner** — a Go
CLI. Think "Make for resources, capabilities, goals, and evidence" instead of
files and timestamps: it starts from a goal, plans backward, executes approved
host commands one step at a time, and records append-only evidence until the
goal is satisfied. See `spec-v01.md` for the full design — it is the source of
truth for *intent*, but the code may have drifted, so trust the code for
*behavior* (see "Spec vs. code" below).

## Build / test / run loop

- `make check` — the gate: gofmt + go vet + golangci-lint + go test. Run before every commit.
- `make test` — tests only. `make test-race` — with the race detector (CI also runs this).
- `make build` — `go build ./...`.
- `make fmt` — `gofmt -s -w .`. gofmt-clean is **gating**; CI fails on drift.
- `make help` — list all targets.
- Fast loop: `go test ./cmd/rp -run TestName` for a single test; the suite is small and fast.
- `make tools` installs pinned analyzers (golangci-lint, gocyclo, gocognit) into GOBIN. Needed for `make lint`/`make check` locally.
- No credentials, network, or external services are needed to build or test.

## Where the code lives

- `cmd/rp/main.go` — the entire CLI: dispatch, planning, execution, evidence. ~3,500 lines. See `cmd/rp/AGENTS.md` for a function-group map.
- `cmd/rp/main_test.go` — end-to-end tests driven through `run([]string)`.
- `cmd/rp/doctest_test.go` — `TestDocExamples`: runs the `console` examples embedded in `docs/` and fails on drift (`make doctest`). Reuses `copyDir` from `main_test.go`.
- `internal/model/` — core data types (`Config`, `Capability`, `Goal`, `Event`, ...) and the confidence ladder. **Types only, no logic.**
- `internal/gitrepo/` — validates that `GitRepo` resources are independent git repos before execution.
- `example-project/` — a **test fixture AND the tutorial** (the spec's Milestone 5 walkthrough). Do not "tidy" it; tests depend on its layout.
- `scripts/gen-site.sh` — builds the HTML site published to GitHub Pages (reports + docs).
- `scripts/render-docs.py` — stdlib-only Markdown→HTML renderer for `docs/` (called by `gen-site.sh`).
- `docs/` — prose documentation (Markdown). Its `console` examples are **runnable tests** (see `cmd/rp/doctest_test.go` and `docs/README.md`); `status=ready` blocks are executed and asserted, `status=todo` blocks are counted placeholders. Keep examples in sync with behavior, or `make doctest` fails.
- `spec-v01.md` — design spec and glossary (intent, not a behavior contract).

## Glossary (domain terms)

Defined in `spec-v01.md` section 2. Key ones: **Resource** (an abstract typed
entity), **Realization** (a concrete backing instance of a resource),
**Capability** (a contract describing how an action derives/observes
resources), **Action** (one concrete invocation of a capability), **Assertion**
(an output is never a fact — it is a claim with evidence), **Evidence**,
**Goal**, **Constraint**, **Effect**.

Confidence ladder (`internal/model/confidence.go`), low → high:
`unsupported < claimed < observed < attested < reproduced < independently_reproduced`.

## Conventions an agent must not infer wrong

- **Types live in `internal/model`; `cmd/rp/main.go` re-aliases them** (`type Config = model.Config`, see `cmd/rp/main.go:29-51`) so the command code uses short names. Add new types in `internal/model`, not `cmd`. Logic lives in `main.go`, not in `model`.
- **`.rp/` YAML rejects unknown keys** unless they are prefixed `x-` (`isExtensionKey`, the `validate*` functions in `main.go`). When you add a config field, update **both** the struct tag in `internal/model` **and** the matching `validate*` allow-list in `main.go`, or loading will reject it.
- gofmt-clean and golangci-lint-clean are gating; do not introduce findings.

## Safety / blast radius

- `rp achieve` and `rp exec` **run real host commands** (each capability's `command.argv`) and write under `.rp/runs/<run-id>/`. The run log is append-only, but the commands themselves DO execute and can mutate a worktree. Tests sandbox this through the fixture; never point these at a real repo casually.
- `example-project/scripts/*.sh` mutate the git worktree (apply a patch, run pytest).
- CI gate = `make check` + `make test-race`. A red gate blocks merge.

## Spec vs. code

`spec-v01.md` describes the v0.1 design and milestones. Where it disagrees with
`cmd/rp/main.go`, the code is authoritative for current behavior. Use the spec
for vocabulary and intent, not as a guarantee of what is implemented.

## Links

- CI / test & static-analysis dashboard: <https://curtcox.github.io/rp/>
- CI status: the badge in `README.md`, or `gh run list`.
