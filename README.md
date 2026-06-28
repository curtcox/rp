# rp

[![CI](https://github.com/curtcox/rp/actions/workflows/ci.yml/badge.svg)](https://github.com/curtcox/rp/actions/workflows/ci.yml)

A local, terminal-first, evidence-auditable resource planner.

📊 **[Test & static-analysis reports](https://curtcox.github.io/rp/)** — coverage,
golangci-lint, and complexity dashboards, published from `main` by CI.

## v0.1 CLI

This repository contains a Go implementation of the v0.1 vertical slice
described in `spec-v01.md`. Core types live in `internal/model/`; GitRepo
validation is in `internal/gitrepo/`.

```sh
go run ./cmd/rp version
go run ./cmd/rp init
go run ./cmd/rp plan bugfix_patch --explain
go run ./cmd/rp exec plan-20260628T120000.000000000Z --yes
go run ./cmd/rp achieve bugfix_patch --yes
go run ./cmd/rp evidence bugfix_patch
go run ./cmd/rp why patched_repo.tests_pass
go run ./cmd/rp trace proposed.patch
go run ./cmd/rp observe repo --with git_status
go run ./cmd/rp attest patched_repo.tests_pass --source human_review --note "reviewed"
```

Implemented commands include:

- `init`
- `capability init command`
- `goal init`
- `policy init`
- `add resource`, `resources`, `resource`
- `plan` with `text`, `json`, `dot`, and `mermaid` output plus saved plan snapshots (`--speculative` shows assumed preconditions without saving)
- `exec` for saved plan execution
- `achieve` with `--dry-run`, `--step`, `--yes`, `--auto-repair`, and `--max-attempts`
- `evidence`, `why`, `trace`, `audit`, `replay`, `replan`, and `rerun`
- manual `observe`, `attest`, and `add assertion`

The runtime loads project-local `.rp/` YAML, resolves local imports, validates
declared fields (unknown keys are rejected unless prefixed with `x-`), merges an
optional user policy from `~/.config/rp/policy.yaml` (most-restrictive wins),
computes a canonical JSON config hash, plans backward from goals with just-in-time
replanning during `achieve`, validates GitRepo resources are independent git
repositories before execution, executes command capabilities serially with
execution-time precondition checks, supports `--auto-repair` retries governed by
policy `execution.auto_repair`, records assertion supersession when evidence is
corrected, writes a goal attestation bundle when evidence requirements are met,
honors policy `hashing` rules for command output and file artifacts, enforces goal
constraints, `max_cost` budgets, `external_side_effects` and `CredentialRef` policy
checks, and `execution.plan_changes` / `execution.on_failure` during JIT replan, prints an
effect summary with plans and runs, writes artifacts under
`.rp/runs/<run-id>/artifacts`, records append-only JSONL events (including
`action_failed` for non-zero exits when `always_record_result` is set), and
explains assertions from the latest run. `rp evidence` reports both required outputs
(`goal.produce`) and required evidence. Use `rp replay RUN_ID` for a narrative
reconstruction of a run; `rp audit RUN_ID` prints the raw event timeline.
`rp replan RUN_ID --yes` continues execution in the prior run; `rp rerun RUN_ID`
starts a fresh run for the same goal.

## Tutorial: bugfix patch

The `example-project/` directory is the Milestone 5 walkthrough from the spec: a
local Git repo, a Markdown bug report, and capabilities that propose a patch,
verify it applies, apply it to the worktree, and run tests.

```sh
cd example-project
git init
git add -A
git commit -m "initial"
go run ../cmd/rp plan bugfix_patch --explain
go run ../cmd/rp achieve bugfix_patch --yes
go run ../cmd/rp why patch.applies_cleanly
go run ../cmd/rp why patched_repo.tests_pass
go run ../cmd/rp evidence bugfix_patch
go run ../cmd/rp audit "$(ls -1 .rp/runs | tail -1)"
go run ../cmd/rp replay "$(ls -1 .rp/runs | tail -1)"
```

The project must have its own `git init` in `example-project/` so Git commands target
that directory rather than a parent repository.

After a successful run you should see:

- `.rp/runs/<run-id>/artifacts/proposed.patch`
- `.rp/runs/<run-id>/artifacts/pytest.stdout` and `pytest.stderr`
- JSONL events and a `summary.json` with `"satisfied": true`
- `rp why` output showing `observed` evidence for patch apply and test pass

The example uses `./scripts/propose_patch.sh` to emit a unified diff and
`./scripts/run_tests.sh`, which runs `pytest` when available and otherwise
falls back to a small Python assertion so the tutorial works without extra
setup.

User policy example (`~/.config/rp/policy.yaml`):

```yaml
version: rp.dev/v0.1
policy:
  permissions:
    network:
      access: forbidden
```

This layers on top of the project policy; the stricter permission wins.

## Development

All checks are driven by the [`Makefile`](Makefile) so that local runs and CI
behave identically. Run `make help` to list targets.

```sh
make tools      # install pinned analysis tools (golangci-lint, gocyclo, gocognit)
make check      # gating suite: gofmt + go vet + golangci-lint + go test
make test       # run tests
make coverage   # write coverage.out and print total coverage
make complexity # cyclomatic (gocyclo) + cognitive (gocognit) complexity
make reports    # build the HTML report site under ./site
make clean      # remove generated artifacts
```

### Continuous integration

[`.github/workflows/ci.yml`](.github/workflows/ci.yml) defines two jobs:

- **checks** (every push and pull request) — runs `make check` plus the race
  detector. This is the gate; lint findings, formatting drift, vet errors, or
  test failures fail the build. Linting is configured in
  [`.golangci.yml`](.golangci.yml).
- **pages** (pushes to `main`) — builds the report site with `make reports` and
  publishes it to **GitHub Pages** at <https://curtcox.github.io/rp/>. The
  dashboard links to test output, the HTML coverage report, golangci-lint
  results, and cyclomatic/cognitive complexity reports. Coverage and complexity
  are informational (published, not gating).

> [!NOTE]
> Pages publishing requires the repository's **Settings → Pages → Build and
> deployment → Source** to be set to **GitHub Actions** (one-time setup).
