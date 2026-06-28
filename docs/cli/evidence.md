# rp evidence

```
rp evidence GOAL
```

Report a goal's evidence status: both the **required outputs** (`goal.produce`)
and the **required evidence** (`goal.requires_evidence`), and which of them the
latest run has satisfied. This is how you tell *why a goal is or isn't done*
without reading raw event logs.

## Example

<!-- rp-example: id=evidence-goal cwd=fixture status=todo -->
```console
$ rp evidence bugfix_patch
# (output to be captured)
```

## See also

- [why](why.md) — explain a single assertion.
- [trace](trace.md) — trace how something was produced.
- [Concepts: evidence & the confidence ladder](../concepts/glossary.md).
