# rp trace

```
rp trace QUERY
```

Trace how a resource or assertion came to be: the chain of actions, inputs, and
evidence that produced `QUERY` (for example `proposed.patch`). Where `why`
explains one assertion, `trace` follows the provenance.

## Example

<!-- rp-example: id=trace-patch cwd=fixture status=todo -->
```console
$ rp trace proposed.patch
# (output to be captured)
```

## See also

- [why](why.md) — explain a single assertion.
- [runs](runs.md) — `audit` / `replay` the full timeline.
