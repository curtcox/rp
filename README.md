# rp

A local, terminal-first, evidence-auditable resource planner.

## v0.1 CLI

This repository contains a Go implementation of the v0.1 vertical slice
described in `spec-v01.md`.

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
- `plan` with `text`, `json`, `dot`, and `mermaid` output plus saved plan snapshots
- `exec` for saved plan execution
- `achieve` with `--dry-run`, `--step`, and `--yes`
- `evidence`, `why`, `trace`, `audit`, `replay`, `replan`, and `rerun`
- manual `observe`, `attest`, and `add assertion`

The runtime loads project-local `.rp/` YAML, resolves local imports, merges an
optional user policy from `~/.config/rp/policy.yaml` (most-restrictive wins),
computes a canonical JSON config hash, executes command capabilities serially,
writes artifacts under `.rp/runs/<run-id>/artifacts`, records append-only JSONL
events, and explains assertions from the latest run.

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
```

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
