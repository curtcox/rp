# Runs: audit / replay / replan / rerun

Every `achieve`/`exec` invocation creates a run under `.rp/runs/<run-id>/` with
an append-only `events.jsonl` and a `summary.json`. These commands inspect and
continue past runs.

```
rp audit RUN_ID
rp replay RUN_ID
rp replan RUN_ID [--yes] [--step]
rp rerun RUN_ID [--yes] [--step] [--auto-repair] [--max-attempts N]
```

## rp audit — raw timeline

Print the raw event timeline for a run.

<!-- rp-example: id=audit-run cwd=fixture status=todo -->
```console
$ rp audit run-20260628T213615.913017000Z
# (output to be captured)
```

## rp replay — narrative reconstruction

A human-readable reconstruction of what happened.

<!-- rp-example: id=replay-run cwd=fixture status=todo -->
```console
$ rp replay run-20260628T213615.913017000Z
# (output to be captured)
```

## rp replan — continue a run

Continue execution in the prior run, replanning what remains.

<!-- rp-example: id=replan-run cwd=fixture status=todo -->
```console
$ rp replan run-20260628T213615.913017000Z --yes
# (output to be captured)
```

## rp rerun — run again

Re-execute a run from the start.

<!-- rp-example: id=rerun-run cwd=fixture status=todo -->
```console
$ rp rerun run-20260628T213615.913017000Z --yes
# (output to be captured)
```

## See also

- [achieve](achieve.md) — produces the runs these commands inspect.
- [evidence](evidence.md) / [why](why.md) — interpret a run's result.
