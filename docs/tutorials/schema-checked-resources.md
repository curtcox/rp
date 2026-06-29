# Tutorial: structural resource schema validation

> **Future work.** Resource types are named strings today; `produceSpecMismatch`
> checks realization **kind** and **media_type** but not JSON/YAML structure.
> This walkthrough describes structural schema validation for realizations — a
> capability may exit 0 and record process evidence, while the produced resource
> is rejected as schema-invalid and the goal stays unsatisfied.

The scenario: a `release_manifest` resource must match a declared JSON Schema.
One capability writes incomplete JSON; a corrected capability writes a valid
manifest. `rp evidence` should distinguish **process success** from **schema-valid
realization**.

## 1. Declared resource and schema

```yaml
# .rp/planner.yaml (excerpt — future)
resources:
  codebase:
    type: SourceTree
    realizations:
      - id: codebase.local
        kind: local_path
        uri: file://src

# .rp/schemas/release_manifest.json (future)
{
  "type": "object",
  "required": ["version", "artifacts", "checksums"],
  "properties": {
    "version": { "type": "string" },
    "artifacts": { "type": "array", "minItems": 1 },
    "checksums": { "type": "object" }
  }
}
```

```yaml
# .rp/goals/release.yaml (excerpt — future)
release_manifest_valid:
  given:
    codebase: codebase
  produce:
    release_manifest:
      type: ReleaseManifest
      required_realization:
        kind: file
        media_type: application/json
        schema: schemas/release_manifest.json
  requires_evidence:
    - subject: release_manifest
      predicate: schema_valid
      min_confidence: observed
      any_source_type:
        - schema_check
```

## 2. Capability succeeds but realization is schema-invalid

`write_manifest_draft` exits 0 and writes JSON missing `checksums`:

```yaml
# .rp/capabilities/manifest.yaml (excerpt — future)
write_manifest_draft:
  purpose: derive
  outputs:
    release_manifest:
      type: ReleaseManifest
      realization:
        kind: file
        media_type: application/json
      assertions:
        - subject: release_manifest
          predicate: written
          confidence: observed
          when:
            exit_code: 0
          evidence_source: command_result
  command:
    argv: ["./scripts/write_manifest_draft.sh"]
```

<!-- rp-example: id=schema-invalid-achieve cwd=gate status=todo exit=1 -->
```console
$ rp achieve release_manifest_valid --yes
Plan for release_manifest_valid (1 steps):
  1. write_manifest_draft — produce release_manifest resource

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/manifest.json
run run-<id> goal produce requirements not fully satisfied
<root>/.rp/runs/run-<id>
```

The command succeeded; `rp` recorded process evidence. The realization failed
schema validation:

<!-- rp-example: id=schema-invalid-evidence cwd=gate status=todo -->
```console
$ rp evidence release_manifest_valid
Goal release_manifest_valid (run run-<id>)
Satisfied: false

Required outputs:
  [schema-invalid] release_manifest (ReleaseManifest) — application/json at artifacts/manifest.json — missing required field "checksums"

Required evidence:
  [missing] release_manifest.schema_valid >= observed
  [ok] release_manifest.written >= observed — observed via command_result (as-step-01-write_manifest_draft-release_manifest-written)

Process vs realization:
  command write_manifest_draft: exit 0 (process success recorded)
  realization release_manifest: schema check failed (schema_check ev-step-01-schema-invalid)
```

## 3. Corrected capability satisfies the schema

A second capability writes the complete manifest; the planner selects it on
replan (or the project replaces the draft writer):

```yaml
write_manifest_final:
  purpose: derive
  outputs:
    release_manifest:
      type: ReleaseManifest
      realization:
        kind: file
        media_type: application/json
        schema: schemas/release_manifest.json
      assertions:
        - subject: release_manifest
          predicate: schema_valid
          confidence: observed
          evidence_source: schema_check
```

<!-- rp-example: id=schema-valid-achieve cwd=gate status=todo -->
```console
$ rp replan run-<id> --yes
Plan for release_manifest_valid (1 steps):
  1. write_manifest_final — produce release_manifest resource
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
$ rp evidence release_manifest_valid
Goal release_manifest_valid (run run-<id>)
Satisfied: true

Required outputs:
  [ok] release_manifest (ReleaseManifest) — application/json at artifacts/manifest.json

Required evidence:
  [ok] release_manifest.schema_valid >= observed — observed via schema_check (as-step-01-write_manifest_final-release_manifest-schema_valid)
```

## What exists today

`rp evidence` reports `[mismatch]` when realization **kind** or **media_type**
does not match `required_realization`. There is no `schema:` field, no
`schema_check` evidence source, and no `[schema-invalid]` output status. A
command that exits 0 and writes a file satisfies `produce` if kind/media_type
match, regardless of file contents.

## Implementation notes

When this lands, expect:

- Optional `schema:` on `required_realization` and capability output
  `realization` blocks (JSON Schema or a referenced file under `.rp/schemas/`).
- Post-action validation hook: after artifact write, validate structure before
  marking the realization effective.
- New evidence source type `schema_check` distinct from `command_result` /
  `process_exit`.
- Evidence report lines that show both process outcome and schema outcome
  (as in the example above).

## Next

- [Data conformance](data-conform.md) — schema validation via command capabilities
  today (external script, not structural typing in `rp`).
- [Config reference](../config/reference.md) — `required_realization` fields.
