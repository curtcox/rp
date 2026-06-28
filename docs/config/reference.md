# Config reference (`.rp/` YAML)

`rp` loads project-local configuration from a `.rp/` directory: a top-level
`planner.yaml` plus files it `imports`. Local imports are resolved relative to the
importing file, declared fields are validated, and a canonical JSON **config
hash** is computed over the merged result.

> **Unknown keys are rejected.** Any key not in the schema below is an error,
> *unless* it is prefixed `x-` (the extension escape hatch). When you add a field
> in code you must update **both** the struct tag in `internal/model` **and** the
> matching allow-list in `cmd/rp/main.go`.

## Top level

| Key | Type | Notes |
| --- | ---- | ----- |
| `version` | string | Schema version, e.g. `rp.dev/v0.1`. |
| `imports` | list | Relative paths to other config files to merge. |
| `resources` | map | Named [resources](#resources). |
| `capabilities` | map | Named [capabilities](#capabilities). |
| `policies` | map | Named [policies](../config/policy.md). |
| `goals` | map | Named [goals](#goals). |
| `defaults` | map | Defaults, e.g. `policy: local_safe`. |

## resources

Each resource has a `type` and a list of `realizations`.

```yaml
resources:
  repo:
    type: GitRepo
    realizations:
      - id: repo.local
        kind: local_path
        uri: file://.
        media_type: inode/directory
```

- **resource** keys: `type`, `realizations`.
- **realization** keys: `id`, `kind`, `uri`, `media_type`, `hash`, `metadata`.

## capabilities

A capability is the contract for an action.

```yaml
capabilities:
  propose_patch_with_script:
    purpose: derive          # derive | observe
    kind: command
    inputs:
      repo:
        type: GitRepo
        requires:
          - predicate: clean_worktree
            min_confidence: observed
    outputs:
      patch:
        type: Patch
        realization: { kind: file, media_type: text/x-diff }
        assertions:
          - subject: patch
            predicate: addresses_bug
            confidence: claimed
            evidence_source: process_output
    command:
      cwd: "${inputs.repo.path}"
      argv: ["./scripts/propose_patch.sh", "${inputs.bug_report.path}"]
      stdout:
        save_as: { resource: patch, artifact_path: artifacts/proposed.patch, media_type: text/x-diff }
    effects:
      external: local_process
      filesystem:
        writes: [".rp/runs/${run.id}/artifacts/proposed.patch"]
    nondeterminism: [external_process]
    idempotence: unknown
    cost: { time: moderate, money_usd: 0, risk: medium, human_attention: none }
```

- **capability** keys: `purpose`, `kind`, `inputs`, `outputs`, `preconditions`,
  `command`, `approval`, `always_record_result`, `effects`, `nondeterminism`,
  `idempotence`, `cost`.
- **input** keys: `type`, `realization`, `requires`.
- **requirement** keys: `subject`, `predicate`, `min_confidence`, `any_source_type`.
- **output** keys: `type`, `required_realization`, `realization`, `assertions`.
- **assertion** keys: `subject`, `predicate`, `object`, `confidence`, `when`,
  `evidence_source`.
- **command** keys: `cwd`, `argv`, `stdout`, `stderr`.
- **stream** (`stdout`/`stderr`) keys: `save_as`, `save_as_artifact`, `media_type`.
- **save_as** keys: `resource`, `artifact_path`, `media_type`.
- **effects** keys: `external`, `planner`, `filesystem`, `network`,
  `external_side_effects`.

### Substitutions

`command.cwd`/`argv` support `${inputs.<name>.path}` and `${run.id}`
placeholders, resolved at execution time.

## goals

```yaml
goals:
  bugfix_patch:
    description: Produce a patch and evidence that tests pass.
    given: { repo: repo, bug_report: bug_report }
    produce:
      patch:
        type: Patch
        required_realization: { kind: file, media_type: text/x-diff }
    requires_evidence:
      - subject: patched_repo
        predicate: tests_pass
        min_confidence: observed
        any_source_type: [process_exit, test_report]
    constraints:
      permissions: { network: forbidden, credentials: forbidden }
      max_cost: { time: 10m, money_usd: 0 }
```

- **goal** keys: `description`, `given`, `produce`, `requires_evidence`,
  `constraints`.

## Extension keys

Anything you need that the schema does not cover must be `x-` prefixed:

```yaml
goals:
  bugfix_patch:
    x-owner: platform-team   # allowed: ignored by the validator
```

## Valid vs. invalid

A field that is neither in the schema nor `x-` prefixed is rejected on load:

<!-- rp-example: id=config-reject-unknown cwd=empty status=todo -->
```console
$ rp resources
# with `owner: platform-team` (no x- prefix) in a goal, loading fails:
# rp: .rp/...: unknown field "owner" (use x-* prefix for extensions)
```

## See also

- [Policy reference](policy.md) — permissions, hashing, execution, budgets.
- [Tutorials](../tutorials/writing-a-capability.md) — build these from scratch.
