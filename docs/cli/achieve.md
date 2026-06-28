# rp achieve

```
rp achieve GOAL [--dry-run] [--step] [--yes] [--auto-repair] [--max-attempts N]
```

Drive a project toward `GOAL`. `achieve` plans backward, then executes capability
commands serially with **execution-time precondition checks** and **just-in-time
replanning**: after each step it re-evaluates what remains. It writes artifacts
under `.rp/runs/<run-id>/artifacts/`, records append-only JSONL events, and stops
when the goal's evidence requirements are met (writing a goal attestation
bundle) — or explains what is missing.

> **Safety:** this runs real host commands and can mutate your worktree. Use a
> throwaway copy of [`example-project`](../tutorials/bugfix-walkthrough.md).

## Flags

| Flag | Effect |
| ---- | ------ |
| `--dry-run` | Plan and report effects without executing. |
| `--step` | Pause for approval before each step. |
| `--yes` | Approve all steps non-interactively. |
| `--auto-repair` | Retry failed steps where policy `execution.auto_repair` allows. |
| `--max-attempts N` | Cap repair attempts per step. |

## Example: run to completion

<!-- rp-example: id=achieve-yes cwd=fixture status=todo -->
```console
$ rp achieve bugfix_patch --yes
# (output to be captured)
```

## Example: preview effects only

<!-- rp-example: id=achieve-dry-run cwd=fixture status=todo -->
```console
$ rp achieve bugfix_patch --dry-run
# (output to be captured)
```

## Example: step through with repair

<!-- rp-example: id=achieve-auto-repair cwd=fixture status=todo -->
```console
$ rp achieve bugfix_patch --yes --auto-repair --max-attempts 2
# (output to be captured)
```

## See also

- [plan](plan.md) — preview without executing.
- [evidence](evidence.md) / [why](why.md) — inspect the result.
- [runs](runs.md) — `replan` / `rerun` a previous run.
