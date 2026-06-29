# Tutorial: solver chooses the cheapest valid plan

> **Future work.** Today `rp plan` uses a deterministic vertical-slice search: it
> picks the first capability that satisfies each evidence gap, sorts by a fixed
> priority, and filters by policy and goal constraints. It is **not** a
> constraint solver or optimizer. This walkthrough describes the intended
> `--planner solver` mode from `spec-v01.md` — choose among multiple valid plan
> candidates by minimizing cost while satisfying goal constraints.

This scenario extends the
[`release-gate`](https://github.com/curtcox/rp/tree/main/release-gate) domain.
The goal `release_candidate` still needs lint, test, and scan evidence before
producing a release artifact, but **two capabilities can observe each gate**
with different cost/risk profiles. A goal-level `max_cost` constraint rejects
some combinations; deterministic first-match planning can pick a valid-but-expensive
path, while the solver should pick the cheapest valid candidate.

## 1. Competing capabilities for the same evidence

Two observers can satisfy `codebase.tests_pass`:

```yaml
# .rp/capabilities/gate.yaml (excerpt — future)
run_tests:
  purpose: observe
  outputs:
    test_result:
      assertions:
        - subject: codebase
          predicate: tests_pass
          confidence: observed
  cost:
    time: cheap          # ~2m
    risk: low

run_integration_suite:
  purpose: observe
  outputs:
    integration_result:
      assertions:
        - subject: codebase
          predicate: tests_pass
          confidence: observed
  cost:
    time: moderate       # ~12m
    risk: moderate
```

Two observers can satisfy `codebase.scan_clean`:

```yaml
run_scan:
  cost:
    time: cheap
    risk: low

run_pentest:
  cost:
    time: moderate
    risk: high
```

The goal keeps a tight budget and forbids high risk:

```yaml
# .rp/goals/release.yaml (excerpt — future)
release_candidate:
  requires_evidence:
    - subject: codebase
      predicate: lint_clean
      min_confidence: observed
    - subject: codebase
      predicate: tests_pass
      min_confidence: observed
    - subject: codebase
      predicate: scan_clean
      min_confidence: observed
  constraints:
    max_cost:
      time: 10m
      risk: low
```

## 2. Why deterministic planning is insufficient

With the current planner, `findAssertionCapability` returns the **first**
capability whose outputs declare the required subject/predicate. If
`run_integration_suite` appears before `run_tests` in the merged config, the
deterministic plan includes the integration suite even though a cheaper path
exists. Sorting by fixed priority does not compare **total plan cost** across
alternatives.

A solver-backed planner should enumerate valid combinations (lint is unique;
tests and scan each have two choices → four candidates before produce step),
score each surviving plan under `max_cost`, and select the minimum.

## 3. Plan with the solver and explain rejections

<!-- rp-example: id=solver-plan-explain cwd=gate status=todo -->
```console
$ rp plan release_candidate --planner solver --explain
Goal: release_candidate
Config: <hash>
Root: <root>
Planner: solver (minimize cost under constraints)
Saved plan: plan-<id>

Plan candidates (4 enumerated):
  candidate A: run_lint + run_integration_suite + run_scan + create_pull_request
    rejected: total time 14m exceeds max_cost.time 10m
  candidate B: run_lint + run_integration_suite + run_pentest + create_pull_request
    rejected: total time 14m exceeds max_cost.time 10m; run_pentest risk high exceeds max_cost.risk low
  candidate C: run_lint + run_tests + run_pentest + create_pull_request
    rejected: run_pentest risk high exceeds max_cost.risk low
  candidate D: run_lint + run_tests + run_scan + create_pull_request
    selected: total time 6m, risk low — cheapest valid candidate

1. step-01
   capability: run_lint
   reason: observe codebase.lint_clean
2. step-02
   capability: run_tests
   reason: observe codebase.tests_pass
   chosen over: run_integration_suite (saves ~10m; both satisfy tests_pass)
3. step-03
   capability: run_scan
   reason: observe codebase.scan_clean
   chosen over: run_pentest (risk high forbidden; both satisfy scan_clean)
4. step-04
   capability: create_pull_request
   reason: produce release_candidate resource

Effect summary:
  external: create_pull_request, local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/release.json
  approval required: create_pull_request
  estimated total cost: time 6m, risk low
```

## 4. Achieve the solver-selected plan

<!-- rp-example: id=solver-achieve cwd=gate status=todo -->
```console
$ rp achieve release_candidate --planner solver --yes
Plan for release_candidate (4 steps, solver-selected):
  1. run_lint — observe codebase.lint_clean
  2. run_tests — observe codebase.tests_pass
  3. run_scan — observe codebase.scan_clean
  4. create_pull_request — produce release_candidate resource

Effect summary:
  external: create_pull_request, local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/release.json
  approval required: create_pull_request
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
```

## Implementation notes

When this lands, expect:

- A `planner.mode` (or `--planner`) switch: `deterministic` (default) vs `solver`.
- Candidate generation from **multiple capabilities per evidence slot**, not only
  the first match.
- Plan-level cost aggregation using existing `cost:` fields on capabilities.
- Rejection reasons surfaced in `--explain` output (same shape as policy blocks
  today).
- Integration with existing `filterPlanByConstraints` and `validatePlanMaxCost`
  — the solver runs **after** capability discovery but **before** saving the
  plan snapshot.

## Next

- [Release gate (today)](release-gate.md) — branching DAG and policy blocks with
  the current deterministic planner.
- [Planning internals](../internals/planning.md) — how the vertical-slice search
  works now.
