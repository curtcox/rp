# Tutorial: flaky service and auto-repair

This walkthrough uses
[`flaky-fix`](https://github.com/curtcox/rp/tree/main/flaky-fix). A service
config starts with a wrong port; the first `run_tests` fails deterministically.
Policy enables `execution.auto_repair` (`max_attempts: 2`); after
`action_failed`, JIT replanning adds `repair_config`, fixes the value, and the
retry passes.

## 1. Plan the goal

The initial plan only needs the test observer:

<!-- rp-example: id=flaky-plan cwd=flaky status=ready -->
```console
$ rp plan service_green
Goal: service_green
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: run_tests
   reason: observe service.tests_pass

Effect summary:
  external: local_process
```

## 2. Achieve with auto-repair

Policy enables repair; no `--auto-repair` flag is required:

<!-- rp-example: id=flaky-achieve cwd=flaky status=ready -->
```console
$ rp achieve service_green --yes
Plan for service_green (1 steps):
  1. run_tests — observe service.tests_pass

Effect summary:
  external: local_process
auto-repair: replanning after run_tests failure (1/2)
Plan revised (2 steps):
  1. repair_config — repair after failure
  2. run_tests — observe service.tests_pass
Plan revised (1 steps):
  1. run_tests — observe service.tests_pass
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
$ rp evidence service_green
Goal service_green (run run-<id>)
Satisfied: true

Required evidence:
  [ok] service.tests_pass >= observed — observed via process_exit (as-step-03-run_tests-service-tests_pass)
```

Inspect the run log for `action_failed`, `auto_repair_attempted`, and the
recovery path with `rp audit run-<id>` or `rp replay run-<id>`.

## Next

- [Execution & JIT re-planning](../internals/execution.md) — the step loop and repair semantics.
- [Bugfix walkthrough](bugfix-walkthrough.md) — the baseline without auto-repair.
