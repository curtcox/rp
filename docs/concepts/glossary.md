# Glossary

The domain vocabulary, drawn from `spec-v01.md` §2. The spec is authoritative for
*intent*; `cmd/rp/main.go` is authoritative for *behavior*.

| Term | Meaning |
| ---- | ------- |
| **Resource** | An abstract typed entity (`GitRepo`, `BugReport`, `Patch`). |
| **Realization** | A concrete backing instance of a resource — a file, a path, a URI. |
| **Capability** | A contract describing how an action *derives* or *observes* resources, including its command, inputs, outputs, preconditions, and effects. |
| **Action** | One concrete invocation of a capability during a run. |
| **Assertion** | A claim about a subject/predicate with evidence and a confidence level. An output is never a bare fact. |
| **Evidence** | The recorded basis for an assertion (process exit, command result, test report, human attestation…). |
| **Goal** | What you want to be true: resources to **produce** plus **evidence required** to believe it. |
| **Constraint** | A limit a goal imposes (permissions, `max_cost`). |
| **Effect** | What a capability does to the world (filesystem writes, external processes, side effects). |

## The confidence ladder

Assertions carry a confidence level. Higher requires stronger evidence
(`internal/model/confidence.go`):

```
unsupported < claimed < observed < attested < reproduced < independently_reproduced
```

- **unsupported** — asserted with no backing.
- **claimed** — stated by the producing process.
- **observed** — backed by a direct observation (e.g. a process exit code).
- **attested** — a human or named source vouches for it (see [attest](../cli/attest.md)).
- **reproduced** — reproduced independently within the system.
- **independently_reproduced** — reproduced by an independent party.

A goal's `requires_evidence` sets a **`min_confidence`** per requirement; the
goal is met only when each observed assertion is *at least* that strong.

## Illustrations

<!-- rp-example: id=glossary-confidence cwd=fixture status=todo -->
```console
$ rp why patched_repo.tests_pass
# (output to be captured — note the confidence level on the assertion)
```

<!-- rp-example: id=glossary-attest cwd=fixture status=todo -->
```console
$ rp attest patched_repo.tests_pass --source human_review --note "reviewed"
# (output to be captured — confidence rises to attested)
```
