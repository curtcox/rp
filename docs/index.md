# rp — evidence-auditable resource planner

`rp` is a local, terminal-first **resource planner**. Think "Make for resources,
capabilities, goals, and evidence" instead of files and timestamps: you declare
a **goal**, `rp` plans backward to a sequence of approved host commands, executes
them one step at a time, and records **append-only evidence** until the goal is
satisfied — or explains exactly why it is not.

```console
$ rp version
rp rp.dev/v0.1
```

<!-- rp-example: id=index-version cwd=empty status=ready -->
```console
$ rp version
rp rp.dev/v0.1
```

## The mental model

- **Resource** — an abstract typed entity (a repo, a bug report, a patch).
- **Realization** — a concrete backing instance of a resource (a file, a path).
- **Capability** — a contract describing how an action derives or observes
  resources, including the command it runs.
- **Action** — one concrete invocation of a capability.
- **Assertion** — an output is never a bare fact; it is a *claim* with evidence
  and a confidence level.
- **Goal** — what you want to be true, plus the evidence required to believe it.

See [Concepts](concepts/overview.md) and the [Glossary](concepts/glossary.md).

## Where to go next

- **[Getting started](getting-started.md)** — install, build, and run the loop.
- **[CLI reference](cli/index.md)** — every command, flag by flag.
- **[Config reference](config/reference.md)** — the `.rp/` YAML schema.
- **[Tutorials](tutorials/bugfix-walkthrough.md)** — end-to-end walkthroughs.
- **[Under the hood](internals/index.md)** — the planning, execution, and
  evidence algorithms, and how to use them well.

The test, coverage, lint, and complexity dashboards live under
[Reports](../index.html).

## A taste of the loop

```console
$ rp init
$ rp plan bugfix_patch --explain
$ rp achieve bugfix_patch --yes
$ rp evidence bugfix_patch
$ rp why patched_repo.tests_pass
```

Each of those is documented, with runnable examples, in the
[CLI reference](cli/index.md).
