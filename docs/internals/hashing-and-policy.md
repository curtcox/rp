# Determinism: hashing, policy & budgets

This page covers the machinery that makes a run *reproducible* and *bounded*: how
`rp` fingerprints its inputs, how it merges policies most-restrictively, how it
caps confidence by source, and how it enforces cost budgets.

## Canonical config hash

Every run records the hash of the config it ran under. `canonicalHash`:

1. Marshals the `Config` to JSON.
2. Re-parses it into a generic tree and `canonicalize`s it: **map keys are sorted
   recursively**, while array order is preserved.
3. Marshals the canonical tree and takes its SHA-256.

The practical consequence: reordering keys in your YAML (or the order imports
happen to define resources) does **not** change the hash, but reordering an
ordered list (e.g. a command's `argv`, or a list of `requires_evidence`) does —
because order is meaningful there. The short form you see as `Config: <hash>` is
this value truncated.

The hash is the run's **identity check**. `rp replan <run-id>` refuses to resume a
run when the current config hash differs from the one recorded at run start, so
you never silently continue against changed inputs. Saved plans (`rp plan` → `rp
exec`) are stamped the same way.

## Config loading and merge

`loadProject` finds the project root by walking up for a `.rp/` directory
(`findProjectRoot`), then `loadConfig` loads the entry config and resolves local
`imports`. `mergeConfig` merges imported configs map-by-map (resources,
capabilities, policies, goals, defaults) where a **later definition wins** for a
duplicate key, and `version` is taken from the first that sets it.

Unknown top-level and nested keys are **rejected** during validation
(`validateYAMLDocument`) — the escape hatch is an `x-` prefix, which is allowed
through (`isExtensionKey`) for your own annotations.

## Policy merge: most-restrictive wins

The active policy is `defaults.policy` from the project config. An optional user
policy at `~/.config/rp/policy.yaml` is layered on top by `applyUserPolicy` →
`mergePolicies`, and the merge is deliberately **conservative — the stricter
setting always wins**:

| Policy area | Merge rule (`mergePolicies`) |
| --- | --- |
| `permissions` | per key, `mostRestrictivePermission`: `forbidden` < `approval_required` < `allowed` |
| `environment` | `inherit` is logical AND; `allow` is the **intersection** of the two lists |
| `evidence` | `source_limits` and `final_goal_rules` lists are **appended** (union of rules) |
| `max_cost` | per key, the **smaller** budget wins (`minCostString` for `Nm` durations) |

So a user policy can only ever *tighten* the project policy, never loosen it —
adding a user policy is always safe.

The policy itself is fingerprinted separately by `hashPolicy` (SHA-256 of the
active policy) and recorded alongside the config hash.

## Confidence caps

A capability declares the confidence each assertion *would* have, but policy gets
the final say. `capConfidenceByPolicy` looks up the policy's `source_limits`
entry for the assertion's evidence source and, if that ceiling is lower than the
declared confidence, lowers the recorded value to the ceiling. This is how a
policy says, e.g., "claims from `process_exit` may rise no higher than
`observed`," regardless of what a capability asserts.

A separate rule set, `final_goal_rules`, governs whether a given source may
satisfy a **final goal** requirement at all (`sourceMaySatisfyRequiredEvidence`),
independent of confidence — useful for accepting a human attestation as a
precondition while still requiring machine evidence to close the goal.

## Cost budgets

The effective budget for a run is the active policy's `max_cost` merged with the
goal's `constraints.max_cost`, taking the **smaller** limit per dimension
(`effectiveMaxCost` / `minCostLimit`). It is enforced at two levels:

- **Per capability** during planning — `capabilityExceedsMaxCost` drops any
  single capability that blows the budget on its own.
- **Per plan** — `validatePlanMaxCost` sums the surviving steps and fails if the
  total exceeds the limit.

Each dimension is compared on its own scale:

| Dimension | How a capability's cost is read |
| --- | --- |
| `time` (`Nm`) | `cost.time` mapped to minutes: `cheap`=1, `moderate`=2, `expensive`=8 (`capabilityEstimatedMinutes`) |
| `money_usd` | numeric `cost.money_usd`, compared directly |
| `tokens` | numeric `cost.tokens`, compared directly |
| `human_attention` | ranked `none` < `approval_if_required` < `low` < `medium` < `high` (`humanAttentionRank`) |

## Plan-change confirmation

When JIT re-planning changes the plan, `rp` decides whether to interrupt you via
`planRevisionNeedsConfirmation`, scored against the dimensions your policy listed
under `execution.plan_changes.allow_without_confirmation_if_not_increasing`
(`permissions`, `risk`, `cost_class`). If the new plan scores higher on any
listed dimension, `rp` asks (unless `--yes`). Scoring uses `planDimensionScore`:

- `permissions`: network/credential use = 3, filesystem writes / write-approval =
  2, other external effects = 1.
- `risk`: `cost.risk` of low/medium/high → 1/2/3.
- `cost_class`: `cost.time` of cheap/moderate/expensive → 1/2/3.

See the worked behavior in [execution](execution.md#when-a-re-plan-stops-to-ask-you).

## How to use this well

- **Treat the config hash as the run's name.** If `rp replan` refuses to resume,
  the config changed since the run started — re-`achieve` or `rerun` rather than
  forcing it.
- **Layer a personal user policy freely.** Because the merge only tightens,
  `~/.config/rp/policy.yaml` is a safe place to add stricter network/credential
  rules that apply across all your projects.
- **Set caps where the risk is.** Use `source_limits` to bound how much you trust
  a class of evidence, and `final_goal_rules` to decide which sources can *close*
  a goal versus merely unblock a step.
- **Budget at the goal when it's goal-specific.** Policy `max_cost` is the floor
  for everyone; a tighter `goal.constraints.max_cost` narrows it for one goal
  without loosening the policy.
