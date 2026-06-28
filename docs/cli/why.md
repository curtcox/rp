# rp why

```
rp why SUBJECT.PREDICATE
```

Explain a single assertion from the latest run: what claimed it, the evidence
backing it, its confidence level, and any assertion it superseded. `why` answers
"why do we believe `patched_repo.tests_pass`?".

## Example

<!-- rp-example: id=why-assertion cwd=fixture status=todo -->
```console
$ rp why patched_repo.tests_pass
# (output to be captured)
```

## Example: an unmet claim

<!-- rp-example: id=why-missing cwd=fixture status=todo -->
```console
$ rp why patch.applies_cleanly
# (output to be captured)
```

## See also

- [evidence](evidence.md) — the whole goal's evidence status.
- [trace](trace.md) — the production chain behind a resource.
