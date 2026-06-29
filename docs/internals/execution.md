# Execution & just-in-time re-planning

`rp achieve` runs the plan **one step at a time**, re-checking the world between
steps. The loop lives in `runPlan`
([`cmd/rp/main.go`](https://github.com/curtcox/rp/blob/main/cmd/rp/main.go)); a
single step is run by `executeStep`. This page walks both.

## The step loop

After emitting `run_started` and `plan_proposed` and validating that every
`GitRepo` the goal uses is its own repository root
(`gitrepo.ValidateGoalGitRepos`), `rp` enters this loop:

1. **Done?** If `missingEvidenceWithConfig` returns nothing, the goal's evidence
   is already satisfied — stop.
2. **Re-plan (JIT).** If just-in-time planning is on (the default for `achieve`),
   call `buildPlan` again against the *current* config. If the set of
   capabilities changed from the last iteration, `rp` may pause to confirm (see
   below) and records a `plan_revised` event. With `rp exec` the saved plan is
   used as-is and this step is skipped.
3. **Pick the next step.** The first step whose capability has **not** already
   run this run (`executedCaps`). If every capability has run, stop.
4. **Pre-flight checks**, in order — any failure stops the run and writes a
   summary explaining why:
   - `checkCapabilityPolicy` — the capability's effects are allowed by policy.
   - `checkGoalConstraints` — the capability satisfies the goal's constraints.
   - `checkStepPreconditions` — every input `requires` and capability
     `precondition` is met **by recorded evidence** (see next section). A failure
     here emits `plan_invalidated` with `suggest: replan`.
5. **Approval.** If the capability needs approval and you did not pass `--yes`,
   `rp` emits `approval_requested` and prompts. Denial stops the run.
6. **Execute** via `executeStep`. On a non-zero exit (or any error), `rp` records
   `plan_invalidated` and either **auto-repairs** or stops (see below).
7. **Evaluate the gap.** Mark the capability run, recompute the evidence gap, and
   emit `goal_gap_evaluated` with the remaining requirements. If the gap is
   closed, stop.

When the loop ends, `rp` computes the final evidence gap **and** the produce gap
(`missingProduce`); the goal is `satisfied` only when *both* are empty. It emits
`goal_satisfied`, and on success writes the [attestation bundle](evidence.md).

## Preconditions are checked against evidence, at execution time

This is the key difference from a static build tool. `checkStepPreconditions`
does not inspect the filesystem — it reads the run's recorded assertions
(`assertionsFromRun`), drops superseded ones, and asks whether each requirement
is met: matching subject + predicate, confidence at or above the required
minimum, an allowed evidence source, and a source the policy permits for this use
(`assertionRequirementMet`). A subject defaults from the step's wired inputs, or
the input/`repo` name, when the requirement does not name one explicitly.

Because this runs *between* steps, a precondition that was true at plan time but
false now (or never established) is caught right before the dependent command —
not assumed from the original plan.

## Auto-repair

When a step fails and auto-repair is enabled, `rp` does **not** just retry the
same command — it loops back to step 2 and **re-plans** from the current
evidence, which may pick a different remedy:

- A failure increments `failureCount` and emits `plan_invalidated`.
- If auto-repair is on and `failureCount < maxAttempts`, `rp` emits
  `auto_repair_attempted` and `continue`s the loop (re-plan + retry).
- Otherwise the run stops with a summary.

Whether auto-repair is on, and how many attempts, is resolved by
`autoRepairSettings`: the policy's `execution.auto_repair` (`enabled`,
`max_attempts`) provides the baseline, and the `--auto-repair` / `--max-attempts`
flags override it. Default `max_attempts` is 1 (i.e. no extra attempt) unless
policy or flag raises it.

## When a re-plan stops to ask you

JIT re-planning can change the plan as evidence accrues. `rp` only interrupts you
when the change is *riskier*, and only if your policy opted in. The logic is
`planRevisionNeedsConfirmation`:

- Policy `execution.plan_changes.allow_without_confirmation_if_not_increasing`
  lists the dimensions you've agreed may grow silently — chosen from
  `permissions`, `risk`, and `cost_class`.
- For each listed dimension, `rp` scores the old and new capability sets
  (`planDimensionScore`). If the new plan scores **higher** on any of them, `rp`
  asks for confirmation (unless `--yes`), naming the dimension that increased.

Scoring, briefly: `permissions` ranks network/credential use (3) above
filesystem writes (2) above other external effects (1); `risk` and `cost_class`
read the capability's `cost.risk` (low/medium/high) and `cost.time`
(cheap/moderate/expensive). See
[policy](hashing-and-policy.md#plan-change-confirmation).

## Inside `executeStep`

For the selected step, `executeStep`:

1. Emits `action_started`, then resolves the command. `substitute` expands
   `${run.id}` and `${inputs.<name>.path}` tokens — the latter resolves to a
   resource realization's path (with built-in handling so `patch` points at the
   run's `artifacts/proposed.patch` and `patched_repo` resolves to the repo).
2. Runs `argv` in the resolved `cwd` with an environment built by
   `environmentFor` (governed by the policy's `environment.inherit`/`allow`).
3. Captures stdout/stderr, writes any declared artifacts (`saveStreams`), and
   records an `observation_recorded` event (`source_type: process_exit`, the exit
   code, and — when policy `hashing` enables it — `stdout_sha256`/`stderr_sha256`)
   plus an `evidence_recorded` event contributing confidence `observed`.
4. For each declared output: records a `resource_realization_recorded` event for
   any saved-as resource, and for each assertion spec whose `when` matches the
   exit code / stdout (`assertionMatches`), records an assertion — its confidence
   first **capped by policy** (`capConfidenceByPolicy`) and superseding any prior
   claim for the same subject/predicate. See [evidence](evidence.md).
5. On non-zero exit, emits `action_failed` and returns an error (noting
   `result recorded` if the capability set `always_record_result`). Otherwise
   emits `action_completed`.

## Resuming a run

`rp replan <run-id>` continues a prior run in place. It reopens the run, replays
events to learn which capabilities already completed (`executedCapabilitiesFromEvents`)
and the step count, and **refuses to continue if the config hash changed** since
the run started. `rp rerun <run-id>` instead starts a fresh run for the same
goal. (`rp exec` runs a saved plan with JIT off.)

## How to use this well

- **Prefer `achieve` while iterating, `exec` for repeatability.** `achieve`
  adapts via JIT re-planning; `exec` runs a frozen, hash-stamped snapshot.
- **Model preconditions as evidence, not assumptions.** A capability's
  `inputs.*.requires` and `preconditions` are enforced from recorded assertions
  at execution time — so add an observer capability that *produces* the assertion
  a consumer needs.
- **Tune auto-repair in policy, not by re-running by hand.** Set
  `execution.auto_repair.enabled` / `max_attempts`; auto-repair re-plans rather
  than blindly retrying, so an alternate path can be taken.
- **Opt into silent re-plans deliberately.** List only the dimensions you're
  comfortable letting grow under
  `execution.plan_changes.allow_without_confirmation_if_not_increasing`; leave the
  rest to prompt you.
- **Read the trail when something stops.** `rp audit <run-id>` shows the raw
  events (look for `plan_invalidated`, `run_stopped`, `approval_denied`); `rp
  replay <run-id>` narrates them.
