# rp attest

```
rp attest SUBJECT.PREDICATE --source SOURCE [--note NOTE]
```

Record a human **attestation** for an assertion — raising its confidence on the
ladder (e.g. from `observed` to `attested`) with a named source and optional
note. Attestations are recorded as append-only evidence like any other claim.

## Flags

| Flag | Effect |
| ---- | ------ |
| `--source SOURCE` | Who/what is attesting (e.g. `human_review`). |
| `--note NOTE` | Free-text note stored with the attestation. |

## Example

<!-- rp-example: id=attest-tests-pass cwd=fixture status=todo -->
```console
$ rp attest patched_repo.tests_pass --source human_review --note "reviewed"
# (output to be captured)
```

## See also

- [add assertion](scaffold.md) — record a fresh assertion.
- [why](why.md) — see the resulting evidence.
- [Concepts: the confidence ladder](../concepts/glossary.md).
