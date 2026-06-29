# Under the hood: how `rp` works

This section explains the algorithms `rp` runs so you can reason about *why* it
plans what it plans, *when* it re-plans, and *how* it decides a goal is met. None
of this is required to use the tool — the [CLI reference](../cli/index.md) and
[tutorials](../tutorials/bugfix-walkthrough.md) stand on their own — but knowing
the mechanics is the fastest way to write capabilities and policies that behave
the way you expect.

Everything described here is implemented in
[`cmd/rp/main.go`](https://github.com/curtcox/rp/blob/main/cmd/rp/main.go) and the
helpers under [`internal/`](https://github.com/curtcox/rp/tree/main/internal).
Where it helps, the pages name the exact function so you can read the source.

## The pipeline at a glance

Every `rp achieve` (and `rp plan`/`rp exec`) flows through the same stages:

```text
  load + merge config          loadProject → loadConfig → mergeConfig
        │                       validate unknown keys (x- escape hatch)
        ▼
  apply user policy            applyUserPolicy (most-restrictive merge)
        │
        ▼
  hash config + policy         canonicalHash, hashPolicy   ← run identity
        │
        ▼
  build plan (backward)        buildPlan → filter by policy/constraints/cost
        │                              → sort by priority
        ▼
  execute step-by-step  ┌───►  pick first un-run step
        │               │      check policy / constraints / preconditions
        │               │      run command, record observation + evidence
        │               │      derive assertions (confidence capped by policy)
        │               │
        │               │      ┌── evidence gap remains? ──┐
        │               └──────┤  JIT re-plan, maybe ask   │
        │                      └── gap closed ─────────────┘
        ▼
  evaluate goal                missingEvidence + missingProduce
        │
        ▼
  attest (if satisfied)        recordGoalAttestation, writeSummary
```

The detail pages break this down:

- **[Backward planning](planning.md)** — how a goal becomes an ordered list of
  capabilities, and how policy, constraints, and budgets prune it.
- **[Execution & JIT re-planning](execution.md)** — the step loop,
  execution-time preconditions, auto-repair, and when `rp` stops to ask you.
- **[Evidence, confidence & attestation](evidence.md)** — how outputs become
  assertions, the confidence ladder, supersession, and the satisfied/not-satisfied
  decision.
- **[Determinism: hashing, policy & budgets](hashing-and-policy.md)** — the
  canonical config hash, most-restrictive policy merge, confidence caps, and cost
  enforcement.

## Two invariants worth internalizing

Two properties shape almost every algorithm below:

1. **The event log is the source of truth.** `rp` does not keep a mutable
   in-memory model of "what is true." It appends events to
   `.rp/runs/<run-id>/events.jsonl` and *re-reads them* to answer questions like
   "is this precondition met?" or "is the goal satisfied?" (`assertionsFromRun`,
   `missingEvidenceWithConfig`). Nothing is overwritten; corrections
   **supersede**. This is what makes a run replayable and auditable — and it is
   why `rp why` and `rp replay` can reconstruct a conclusion exactly.

2. **Identity is a hash, not a path.** A run records the canonical hash of the
   config and policy it ran under. Resuming a run (`rp replan`) refuses to
   proceed if the current config hash differs, so you never silently continue a
   run against changed inputs. See [hashing](hashing-and-policy.md).

> [!NOTE]
> The v0.1 planner is deliberately a **vertical slice**: its backward search
> understands the bugfix workflow's resource and predicate names (`patch`,
> `patched_repo`, `clean_worktree`, `applies_cleanly`, `tests_pass`) as built-in
> cases, with a generic fallback for everything else. The
> [planning page](planning.md) is explicit about which parts are general and
> which are specialized, so you know what you can lean on today.
