# Backward planning

`rp plan` and the first iteration of `rp achieve` both call `buildPlan`, which
turns a **goal** into an ordered list of **capabilities** to run. This page
describes that algorithm and the three filters that prune it.

## The search

Planning is *backward*: it starts from what the goal needs and works toward
capabilities that produce it. `buildPlan` (in
[`cmd/rp/main.go`](https://github.com/curtcox/rp/blob/main/cmd/rp/main.go))
proceeds in this order, adding each capability at most once (`seen` set):

1. **Resolve the working resources.** The repo is `goal.given.repo`, or the first
   `GitRepo` resource if unset; the bug report is `goal.given.bug_report`.
2. **Satisfy declared preconditions.** If any capability's inputs `require` a
   `clean_worktree` predicate, `rp` adds the capability that *observes*
   `repo.clean_worktree` so the precondition can be met before the consumer runs.
3. **Produce required outputs.** If the goal's `produce:` map asks for a `patch`,
   `rp` adds the capability whose outputs include `patch`
   (`findOutputCapability`).
4. **Produce required evidence.** For each entry in the goal's
   `requires_evidence:`, `rp` adds the capability that yields that
   subject/predicate assertion (`findAssertionCapability`). A few subjects are
   handled specially:
   - `patch` + `applies_cleanly` → the patch-verification capability.
   - `patched_repo` → first *derive* `patched_repo` from `repo` + `patch`, then
     observe the predicate on `patched_repo`.
   - any other `subject` + `predicate` → the matching observer, generically.
5. **Fallback.** If nothing matched and there is exactly one capability defined,
   that capability is the plan. Otherwise planning fails with `no plan found`.

> [!NOTE]
> Steps 2–4 contain the **bugfix-specialized** cases (the literal predicate and
> resource names). The generic machinery is `findOutputCapability` /
> `findAssertionCapability`, which match a capability by what its `outputs:`
> declare. When you write your own capabilities, make their `outputs` and
> `assertions` name the subjects/predicates your goal's `requires_evidence`
> asks for — that is what lets the generic path find them.

## Pruning: policy, constraints, budget

Before the plan is returned it passes through three filters, in this order. Any
of them can empty the plan, which is reported as a distinct error so you know
*why* there is no plan:

| Filter | Function | Removes a step when… | Error if empty |
| --- | --- | --- | --- |
| Policy | `filterPlanByPolicy` | the capability uses a permission the active policy marks `forbidden` | `no plan found under active policy` |
| Constraints | `filterPlanByConstraints` | the capability violates a `goal.constraints` rule | `no plan found under goal constraints` |
| Budget | `validatePlanMaxCost` | the plan's total cost exceeds the effective `max_cost` | a cost-specific error |

Budget enforcement happens at **two granularities**: a single capability that
exceeds the limit on its own is dropped during filtering
(`capabilityExceedsMaxCost`), and the *sum* across the surviving plan is checked
by `validatePlanMaxCost`. See [budgets](hashing-and-policy.md#cost-budgets) for
how each cost dimension (time, money, tokens, human attention) is compared.

## Ordering

The surviving steps are sorted by a fixed **priority**, then by capability name
for stability (`planStepPriority`). Lower number runs first:

| Priority | What earns it |
| --- | --- |
| 10 | asserts `clean_worktree` (establish a clean base first) |
| 20 | output named `patch` |
| 25 | capability `purpose: observe` |
| 30 | asserts `applies_cleanly` |
| 35 | capability `purpose: derive` |
| 40 | output named `patched_repo` |
| 45 | anything else (default) |
| 50 | asserts `tests_pass` (verify last) |

The intent is the natural bugfix order: confirm a clean worktree → produce the
patch → check it applies → apply it → run the tests. Because ordering is by
*priority class* rather than a dependency graph, two capabilities in the same
class are ordered by name — keep that in mind if you add several capabilities at
the same altitude.

After sorting, step IDs are re-assigned `step-01`, `step-02`, … so the printed
and saved plan reads top to bottom.

## Saved plans vs. just-in-time plans

- `rp plan … ` prints the plan and (unless `--speculative`) saves a **snapshot**
  (`SavedPlan`) under `.rp/plans/` stamped with the config and policy hash. `rp
  exec <plan-id>` later runs *that frozen plan* without re-planning.
- `rp achieve` re-plans **just in time** between steps by default — see
  [execution](execution.md). The plan you see at the start can change as evidence
  accumulates.

## How to use this well

- **Name outputs and assertions to match your goal.** The generic finders match
  on declared `outputs`/`assertions`. If `rp plan` says `no plan found`, the
  usual cause is a `requires_evidence` subject/predicate that no capability
  advertises.
- **Read `rp plan --explain` before `achieve`.** It shows the chosen order and
  the per-step *reason* string, which tells you which branch of `buildPlan`
  selected each capability.
- **Use `--speculative` to explore.** It prints assumed preconditions without
  writing a snapshot, so you can iterate on a goal definition without
  accumulating plan files.
- **If a step you expect is missing, suspect a filter.** The three "no plan
  found under …" messages point straight at policy, constraints, or budget.
