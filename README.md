# rp
a local, terminal-first, evidence-auditable resource planner

## v0.1 CLI

This repository now contains a small Go implementation of the v0.1 vertical
slice described in `spec-v01.md`.

```sh
go run ./cmd/rp version
go run ./cmd/rp init
go run ./cmd/rp plan bugfix_patch --explain
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
- `plan` with `text`, `json`, `dot`, and `mermaid` output
- `achieve` with `--dry-run`, `--step`, and `--yes`
- `evidence`, `why`, `trace`, `audit`, `replay`, `replan`, and `rerun`
- manual `observe`, `attest`, and `add assertion`

The runtime loads project-local `.rp/` YAML, resolves local imports, computes a
canonical JSON config hash, executes command capabilities serially, writes
artifacts under `.rp/runs/<run-id>/artifacts`, records append-only JSONL events,
and explains assertions from the latest run.
