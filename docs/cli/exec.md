# rp exec

```
rp exec PLAN_ID [--dry-run] [--step] [--yes]
```

Execute a plan snapshot previously saved by [`rp plan`](plan.md). Where
`achieve` plans and runs in one shot, `exec` runs a *specific, frozen* plan —
useful for review-then-run workflows where the plan is approved before execution.

## Flags

| Flag | Effect |
| ---- | ------ |
| `--dry-run` | Report what would run without executing. |
| `--step` | Pause for approval before each step. |
| `--yes` | Approve all steps non-interactively. |

## Example: plan, then execute the snapshot

<!-- rp-example: id=exec-saved cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch
$ rp exec plan-20260628T223322.212258000Z --yes
# (output to be captured)
```

## See also

- [plan](plan.md) — produce the snapshot.
- [achieve](achieve.md) — plan and execute together.
