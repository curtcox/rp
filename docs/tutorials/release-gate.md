# Tutorial: release gate with a policy block

This walkthrough uses
[`release-gate`](https://github.com/curtcox/rp/tree/main/release-gate). The
goal `release_candidate` requires three **independent** gate checks —
`lint_clean`, `tests_pass`, and `scan_clean` — converging before a
`create_pull_request` side effect produces the release artifact. Policy forbids
`external_side_effects.create_pull_request`, so `rp plan` renders the full
branching DAG **and** annotates the blocked step; `rp achieve` runs the checks
then stops with a clear policy error before the forbidden effect.

## 1. Plan the branching DAG

Three observe steps have no dependencies on each other; `create_pull_request`
depends on all three preconditions:

<!-- rp-example: id=gate-plan cwd=gate status=ready -->
```console
$ rp plan release_candidate
Goal: release_candidate
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: run_lint
   reason: observe codebase.lint_clean
2. step-02
   capability: run_scan
   reason: observe codebase.scan_clean
3. step-03
   capability: run_tests
   reason: observe codebase.tests_pass
4. step-04
   capability: create_pull_request
   reason: produce release_candidate resource
   policy blocked: policy forbids external side effect create_pull_request

Effect summary:
  external: create_pull_request, local_process
  external side effects: create_pull_request
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/release.json
  approval required: create_pull_request

Policy blocks:
  - create_pull_request (policy forbids external side effect create_pull_request)
```

Add `--format mermaid` or `--format dot` to export the same step graph.

## 2. Achieve until policy blocks

Gate checks succeed; the forbidden PR step is reported before it runs:

<!-- rp-example: id=gate-achieve cwd=gate status=ready exit=1 -->
```console
$ rp achieve release_candidate --yes
Plan for release_candidate (4 steps):
  1. run_lint — observe codebase.lint_clean
  2. run_scan — observe codebase.scan_clean
  3. run_tests — observe codebase.tests_pass
  4. create_pull_request — produce release_candidate resource

Effect summary:
  external: create_pull_request, local_process
  external side effects: create_pull_request
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/release.json
  approval required: create_pull_request

Policy blocks:
  - create_pull_request (policy forbids external side effect create_pull_request)
rp: policy forbids external side effect create_pull_request
```

## 3. Evidence after the block

After the policy stop, gate evidence is already recorded — only the artifact
is missing. (This block re-runs the achieve step; it exits non-zero when the
forbidden side effect is reached, then queries evidence.)

<!-- rp-example: id=gate-evidence cwd=gate status=todo -->
```console
$ rp achieve release_candidate --yes
# exits 1: policy forbids external side effect create_pull_request
$ rp evidence release_candidate
Goal release_candidate (run run-<id>)
Satisfied: false

Required outputs:
  [missing] release_candidate (ReleaseCandidate)

Required evidence:
  [ok] codebase.lint_clean >= observed — observed via command_result (as-step-01-run_lint-codebase-lint_clean)
  [ok] codebase.tests_pass >= observed — observed via process_exit (as-step-03-run_tests-codebase-tests_pass)
  [ok] codebase.scan_clean >= observed — observed via command_result (as-step-02-run_scan-codebase-scan_clean)
```

## Next

- [Flaky fix & auto-repair](flaky-fix.md) — recover from `action_failed` events.
- [Policy](../config/policy.md) — side-effect permissions.
