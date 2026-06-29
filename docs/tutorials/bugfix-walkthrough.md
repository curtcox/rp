# Tutorial: the bugfix walkthrough

This is the end-to-end walkthrough from the spec's Milestone 5, driven through
the [`example-project`](https://github.com/curtcox/rp/tree/main/example-project)
fixture. The goal `bugfix_patch` produces a patch for a bug report and evidence
that the configured tests pass.

> The fixture's `achieve`/`exec` steps run real commands (apply a patch, run
> pytest). Work in a throwaway copy.

## 1. Look at the project

The project declares two resources — the repo and the bug report:

<!-- rp-example: id=tut-resources cwd=fixture status=ready -->
```console
$ rp resources
bug_report	BugReport	file://bug.md
repo	GitRepo	file://.
```

<!-- rp-example: id=tut-bugreport cwd=fixture status=ready -->
```console
$ rp resource bug_report
{
  "type": "BugReport",
  "realizations": [
    {
      "id": "bug_report.markdown",
      "kind": "file",
      "uri": "file://bug.md",
      "media_type": "text/markdown"
    }
  ]
}
```

## 2. Plan backward from the goal

`rp` figures out the five steps needed to satisfy the goal, and saves the plan:

<!-- rp-example: id=tut-plan cwd=fixture status=ready -->
```console
$ rp plan bugfix_patch
Goal: bugfix_patch
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: observe_git_status
   reason: observe precondition clean_worktree
2. step-02
   capability: propose_patch_with_script
   reason: produce patch resource
3. step-03
   capability: check_patch_applies
   reason: observe patch.applies_cleanly
4. step-04
   capability: apply_patch_to_worktree
   reason: derive patched_repo
5. step-05
   capability: run_tests
   reason: observe patched_repo.tests_pass

Effect summary:
  external: local_filesystem_write, local_process
  filesystem writes:
    - ${inputs.repo.path}
    - .rp/runs/${run.id}/artifacts/proposed.patch
  approval required: apply_patch_to_worktree, propose_patch_with_script
```

Add `--explain` to see each step's resolved `inputs`.

## 3. Achieve the goal

Execute the plan, approving all steps:

<!-- rp-example: id=tut-achieve cwd=fixture status=todo -->
```console
$ rp achieve bugfix_patch --yes
# (output to be captured)
```

## 4. Check the evidence

<!-- rp-example: id=tut-evidence cwd=fixture status=todo -->
```console
$ rp evidence bugfix_patch
# (output to be captured)
```

## 5. Ask why a claim holds

<!-- rp-example: id=tut-why cwd=fixture status=todo -->
```console
$ rp why patched_repo.tests_pass
# (output to be captured)
```

## 6. Replay the run

<!-- rp-example: id=tut-replay cwd=fixture status=todo -->
```console
$ rp replay run-20260628T213615.913017000Z
# (output to be captured)
```

## Next

- [Reproducible build](reproducible-build.md) — climb past `observed` to
  `reproduced` and `attested`.
- [Writing a capability](writing-a-capability.md)
- [Defining a goal](defining-a-goal.md)
