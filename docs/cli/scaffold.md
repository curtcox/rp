# Scaffolding: capability / goal / policy / add

Commands that write new config into `.rp/`. After scaffolding, edit the
generated YAML — see the [config reference](../config/reference.md).

```
rp capability init command NAME
rp goal init NAME
rp policy init NAME
rp add resource NAME --type TYPE (--uri URI | --file PATH) [--media-type TYPE]
rp add assertion SUBJECT.PREDICATE [--subject SUBJECT] [--confidence LEVEL]
```

## rp capability init command NAME

Scaffold a `command`-kind capability — the contract for an action that runs a
host command.

<!-- rp-example: id=capability-init cwd=empty status=todo -->
```console
$ rp capability init command run_linter
# (output to be captured)
```

## rp goal init NAME

Scaffold a goal: what to produce and the evidence required to believe it.

<!-- rp-example: id=goal-init cwd=empty status=todo -->
```console
$ rp goal init ship_release
# (output to be captured)
```

## rp policy init NAME

Scaffold a policy (permissions, hashing, execution, budgets).

<!-- rp-example: id=policy-init cwd=empty status=todo -->
```console
$ rp policy init local_safe
# (output to be captured)
```

## rp add resource

Declare a resource from the command line.

<!-- rp-example: id=add-resource cwd=empty status=todo -->
```console
$ rp add resource spec --type Document --file spec.md --media-type text/markdown
# (output to be captured)
```

## rp add assertion

Record a fresh assertion about a subject/predicate.

<!-- rp-example: id=add-assertion cwd=fixture status=todo -->
```console
$ rp add assertion repo.clean_worktree --confidence observed
# (output to be captured)
```

## See also

- [config reference](../config/reference.md) — the schema for what you scaffold.
- [Tutorials: writing a capability](../tutorials/writing-a-capability.md),
  [defining a goal](../tutorials/defining-a-goal.md).
