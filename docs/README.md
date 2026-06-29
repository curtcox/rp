# Documentation source

This directory holds the prose documentation for `rp`. It is rendered to HTML by
`scripts/gen-site.sh` and published to <https://curtcox.github.io/rp/> alongside
the test and static-analysis dashboards.

- `index.md` — site landing page.
- `getting-started.md` — install, build, and the core loop.
- `cli/` — one page per command.
- `concepts/` — the model and the glossary.
- `internals/` — under-the-hood algorithms (planning, execution, evidence,
  hashing/policy) and how to use them well.
- `config/` — the `.rp/` YAML reference and policy semantics.
- `tutorials/` — end-to-end walkthroughs.
  - **Implemented** — bugfix, data conform, translation, release gate, flaky fix,
    reproducible build.
  - **Future (`status=todo`)** — [solver-backed release](tutorials/solver-backed-release.md),
    [signed attestation](tutorials/signed-attestation.md),
    [schema-checked resources](tutorials/schema-checked-resources.md),
    [durable polling](tutorials/durable-polling.md). These document proposed
    capabilities; console examples are placeholders until the features land.

## Runnable examples (the doctest convention)

Examples in these pages are **executed as tests** by `cmd/rp/doctest_test.go`
(`make doctest`). They both illustrate usage and guard against drift: if the
documented output stops matching reality, the build fails.

An example is a fenced `console` block preceded by an `rp-example` comment:

````markdown
<!-- rp-example: id=readme-version cwd=empty status=ready -->
```console
$ rp version
rp rp.dev/v0.1
```
````

Attributes:

| attr     | values                | meaning                                                        |
| -------- | --------------------- | -------------------------------------------------------------- |
| `id`     | unique slug           | subtest name; must be unique across all docs                   |
| `cwd`    | `empty` \| `fixture` \| `repro` \| `conform` \| `translate` \| `gate` \| `flaky` | fresh temp dir, a sandboxed copy of `example-project` (git-init), `reproducible-build`, `data-conform`, `translate-doc`, `release-gate`, or `flaky-fix` |
| `status` | `ready` \| `todo`     | `ready` is executed and asserted; `todo` is a counted placeholder |
| `exit`   | integer (default `0`) | expected exit code of the **last** command in the block        |

Inside the block, lines starting with `$ ` are commands; everything after a
command (until the next `$ ` or the end of the block) is its expected combined
stdout+stderr output. The leading `rp` token is rewired to a freshly built
binary, so you write examples exactly as a user would type them.

### Placeholders

Most examples ship as `status=todo` — they show the shape of a command but are
not yet asserted. They are **counted and reported** (not silently ignored) so
the backlog of examples-to-capture stays visible:

```
$ make doctest
...
    doctest_test.go:NNN: doc examples: 6 ready, 41 todo placeholders across 12 files
```

### Promoting a placeholder to a test

1. Run the command in a sandbox (`cwd=empty`, or a throwaway copy of
   `example-project` for `cwd=fixture`).
2. Paste the real output under the `$ ` line.
3. Replace volatile tokens with their redacted forms (see below).
4. Flip `status=todo` to `status=ready` and run `make doctest`.

### Output normalization (redaction)

Volatile tokens are normalized before comparison, so examples stay stable across
machines and runs. Use these placeholders in expected output:

| real token                         | use instead    |
| ---------------------------------- | -------------- |
| the sandbox/project root path      | `<root>`       |
| `run-20260628T213615.913017000Z`   | `run-<id>`     |
| `plan-20260628T223322.212258000Z`  | `plan-<id>`    |
| `Config: 885429622b4f`             | `Config: <hash>` |

If an example produces other nondeterministic output (counts, durations), keep
it `status=todo` or extend the `redact` rules in `cmd/rp/doctest_test.go`.
