# CLI reference

Every `rp` verb, grouped by what it does. The leading token is written `rp`
throughout; from a checkout use `go run ./cmd/rp ...`.

## Commands

| Command | Page | Purpose |
| ------- | ---- | ------- |
| `init` | [init](init.md) | Create a `.rp/` project. |
| `capability init command`, `goal init`, `policy init`, `add` | [scaffold](scaffold.md) | Scaffold config and add resources/assertions. |
| `resources`, `resource` | [resources](resources.md) | List and inspect declared resources. |
| `plan` | [plan](plan.md) | Plan backward from a goal (no execution). |
| `achieve` | [achieve](achieve.md) | Execute toward a goal with JIT replanning. |
| `exec` | [exec](exec.md) | Execute a previously saved plan snapshot. |
| `evidence` | [evidence](evidence.md) | Show required vs. observed evidence for a goal. |
| `why` | [why](why.md) | Explain one assertion from the latest run. |
| `trace` | [trace](trace.md) | Trace how a resource/assertion was produced. |
| `observe`, `attest` | [observe](observe.md), [attest](attest.md) | Record manual observations and attestations. |
| `audit`, `replay`, `replan`, `rerun` | [runs](runs.md) | Inspect and continue past runs. |
| `version` | — | Print the version. |

## Conventions

- **Goals, resources, capabilities** are referenced by their config name.
- **Assertions** are referenced as `SUBJECT.PREDICATE` (e.g.
  `patched_repo.tests_pass`).
- Mutating commands (`achieve`, `exec`) prompt for approval unless `--yes` is
  given; `--step` walks one step at a time; `--dry-run` plans without executing.

## Unknown commands

`rp` rejects unknown verbs with a non-zero exit:

<!-- rp-example: id=cli-unknown cwd=empty status=ready exit=1 -->
```console
$ rp frobnicate
rp: unknown command "frobnicate"
```
