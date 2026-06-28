# Getting started

## Requirements

- Go 1.22+ (to build and run the CLI).
- `git` on your `PATH` (`rp` validates `GitRepo` resources as real git repos).
- No network, credentials, or external services are needed.

## Build and run

`rp` is a single Go module. Run it straight from source:

<!-- rp-example: id=gs-version cwd=empty status=ready -->
```console
$ rp version
rp rp.dev/v0.1
```

> In these docs the command is written `rp`; from a checkout use
> `go run ./cmd/rp ...`, or `go build -o rp ./cmd/rp` once and then `./rp ...`.

With no arguments, `rp` prints its usage:

<!-- rp-example: id=gs-usage cwd=empty status=ready -->
```console
$ rp
rp v0.1

Usage:
  rp init
  rp capability init command NAME
  rp goal init NAME
  rp policy init NAME
  rp add resource NAME --type TYPE (--uri URI | --file PATH) [--media-type TYPE]
  rp resources
  rp resource NAME
  rp plan GOAL [--explain] [--format text|json|dot|mermaid]
  rp achieve GOAL [--dry-run] [--step] [--yes] [--auto-repair] [--max-attempts N]
  rp exec PLAN_ID [--dry-run] [--step] [--yes]
  rp evidence GOAL
  rp why SUBJECT.PREDICATE
  rp trace QUERY
  rp observe RESOURCE --with git_status
  rp attest SUBJECT.PREDICATE --source SOURCE [--note NOTE]
  rp add assertion SUBJECT.PREDICATE [--subject SUBJECT] [--confidence LEVEL]
  rp audit RUN_ID
  rp replay RUN_ID
  rp replan RUN_ID [--yes] [--step]
  rp rerun RUN_ID [--yes] [--step] [--auto-repair] [--max-attempts N]
```

## Initialize a project

`rp` reads project-local configuration from a `.rp/` directory. Create one with:

<!-- rp-example: id=gs-init cwd=empty status=todo -->
```console
$ rp init
# (output to be captured)
```

## The core loop

The everyday workflow is **plan → execute → check evidence → ask why**:

1. **Plan** — see what `rp` would do, without running anything:

   <!-- rp-example: id=gs-plan cwd=fixture status=todo -->
   ```console
   $ rp plan bugfix_patch --explain
   # (output to be captured — see the CLI reference for a ready example)
   ```

2. **Achieve** — execute the plan, one approved step at a time:

   <!-- rp-example: id=gs-achieve cwd=fixture status=todo -->
   ```console
   $ rp achieve bugfix_patch --yes
   # (output to be captured)
   ```

3. **Evidence** — check what is required and what has been observed:

   <!-- rp-example: id=gs-evidence cwd=fixture status=todo -->
   ```console
   $ rp evidence bugfix_patch
   # (output to be captured)
   ```

4. **Why** — explain a single assertion from the latest run:

   <!-- rp-example: id=gs-why cwd=fixture status=todo -->
   ```console
   $ rp why patched_repo.tests_pass
   # (output to be captured)
   ```

## Safety note

`rp achieve` and `rp exec` **run real host commands** and can mutate your
worktree. Try them first against a throwaway copy of
[`example-project`](tutorials/bugfix-walkthrough.md), not a repo you care about.

## Next

- [CLI reference](cli/index.md) — every command in detail.
- [Concepts](concepts/overview.md) — the model behind the loop.
