# rp observe

```
rp observe RESOURCE --with git_status
```

Record a manual observation of a resource. `observe` runs an observer (such as
`git_status`) and records the result as evidence, without running a full plan.
Use it to seed or refresh an observed fact — for example, that a worktree is
clean.

## Example

<!-- rp-example: id=observe-git-status cwd=fixture status=todo -->
```console
$ rp observe repo --with git_status
# (output to be captured)
```

## See also

- [attest](attest.md) — record a human attestation.
- [evidence](evidence.md) — see how observations satisfy a goal.
- [Concepts: the confidence ladder](../concepts/glossary.md).
