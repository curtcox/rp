# rp plan

```
rp plan GOAL [--explain] [--format text|json|dot|mermaid] [--speculative]
```

Plan backward from `GOAL` to an ordered sequence of capability invocations.
`plan` **does not execute anything** — it resolves config, merges policy, filters
steps by policy and goal constraints, and (unless `--speculative`) saves a plan
snapshot under `.rp/cache/plans/` that `rp exec` can later run.

## Flags

| Flag | Effect |
| ---- | ------ |
| `--explain` | Print the reason for each step and an effect summary. |
| `--format` | Output as `text` (default), `json`, `dot`, or `mermaid`. |
| `--speculative` | Show assumed preconditions without saving a snapshot. |

## Example: explain the bugfix plan

Run inside a project (here, the `example-project` fixture):

<!-- rp-example: id=plan-explain cwd=fixture status=ready -->
```console
$ rp plan bugfix_patch --explain
Goal: bugfix_patch
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: observe_git_status
   reason: observe precondition clean_worktree
   inputs: map[repo:repo]
2. step-02
   capability: propose_patch_with_script
   reason: produce patch resource
   inputs: map[bug_report:bug_report repo:repo]
3. step-03
   capability: check_patch_applies
   reason: observe patch.applies_cleanly
   inputs: map[patch:patch repo:repo]
4. step-04
   capability: apply_patch_to_worktree
   reason: derive patched_repo
   inputs: map[patch:patch repo:repo]
5. step-05
   capability: run_tests
   reason: observe patched_repo.tests_pass
   inputs: map[repo:patched_repo]

Effect summary:
  external: local_filesystem_write, local_process
  filesystem writes:
    - ${inputs.repo.path}
    - .rp/runs/${run.id}/artifacts/proposed.patch
  approval required: apply_patch_to_worktree, propose_patch_with_script
```

## Example: machine-readable output

<!-- rp-example: id=plan-json cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --format json
# (output to be captured)
```

## Example: a graph for docs or review

<!-- rp-example: id=plan-mermaid cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --format mermaid
# (output to be captured)
```

<!-- rp-example: id=plan-dot cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --format dot
# (output to be captured)
```

## Example: speculative planning

Show the plan with assumed preconditions, without saving a snapshot:

<!-- rp-example: id=plan-speculative cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --speculative
# (output to be captured)
```

## See also

- [achieve](achieve.md) — plan and execute in one step.
- [exec](exec.md) — run a saved plan snapshot.
