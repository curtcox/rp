# Tutorial: a reproducible build with human sign-off

The [bugfix walkthrough](bugfix-walkthrough.md) only ever reaches `observed`
confidence — a single process exit. This tutorial climbs higher up the
[confidence ladder](../concepts/glossary.md#the-confidence-ladder). Its goal,
`reproducible_release`, asks for two very different kinds of evidence about the
same artifact:

- **`build_reproducible` at `reproduced`** — the system must build the project
  *twice, independently,* and confirm the two artifacts hash identically. That
  is stronger than observing one build succeed.
- **`release_approved` at `attested`** — a human must vouch for the release.
  No capability can produce this; it is met only with [`rp attest`](../cli/attest.md).

It is driven through the
[`reproducible-build`](https://github.com/curtcox/rp/tree/main/reproducible-build)
fixture. Unlike the bugfix project, it has no `GitRepo` resource and needs no
`git init`: the only resource is a `SourceTree`.

> The `achieve` step runs real commands (it builds the project twice). Work in a
> throwaway copy if you are experimenting.

## 1. Look at the project

One resource — the source tree to be built:

<!-- rp-example: id=repro-resources cwd=repro status=ready -->
```console
$ rp resources
source	SourceTree	file://src
```

The deterministic build lives in `scripts/build.sh`: it bundles every file
under `src/` in a stable order, with no timestamps or host paths, so identical
inputs always produce identical bytes. `scripts/compare_builds.sh` runs it twice
and exits non-zero if the hashes ever differ.

## 2. Plan backward from the goal

`rp` plans two steps: build the artifact, then verify it reproduces. The build
is planned first because the verification consumes the `binary` it produces.

<!-- rp-example: id=repro-plan cwd=repro status=ready -->
```console
$ rp plan reproducible_release
Goal: reproducible_release
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: build_artifact
   reason: produce binary resource
2. step-02
   capability: verify_reproducible
   reason: observe binary.build_reproducible

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/app.bundle
  approval required: build_artifact
```

## 3. Achieve, sign off, and check the evidence

`achieve` builds the artifact and verifies reproducibility, recording a
`reproduced` assertion. It stops short of *satisfied*, because the release still
needs a human: `release_approved` can only come from [`rp attest`](../cli/attest.md).
Because the policy's `final_goal_rules` forbid an `llm_claim` from satisfying
required evidence but allow `human_review`, only a human attestation counts.

Evidence is an append-only, project-wide store, so the reproducibility evidence
(from the `achieve` run) and the approval (from the `attest` run) combine to
satisfy the goal:

<!-- rp-example: id=repro-loop cwd=repro status=ready -->
```console
$ rp achieve reproducible_release --yes
Plan for reproducible_release (2 steps):
  1. build_artifact — produce binary resource
  2. verify_reproducible — observe binary.build_reproducible

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/app.bundle
  approval required: build_artifact
run run-<id> goal evidence requirements not fully satisfied
<root>/.rp/runs/run-<id>
$ rp attest binary.release_approved --source human_review --note "release reviewed"
<root>/.rp/runs/run-<id>
$ rp evidence reproducible_release
Goal reproducible_release (run run-<id>)
Satisfied: true

Required outputs:
  [ok] binary (BuildArtifact) — application/octet-stream at artifacts/app.bundle

Required evidence:
  [ok] binary.build_reproducible >= reproduced — reproduced via command_result (as-step-02-verify_reproducible-binary-build_reproducible)
  [ok] binary.release_approved >= attested — attested via human_review (as-manual-attestation-binary-release_approved)
```

## 4. Ask why a claim is trusted

`rp why` reaches across runs to find the strongest assertion for a claim — the
same query, two different rungs of the ladder. (This block re-runs the loop so
it stands alone.)

<!-- rp-example: id=repro-why cwd=repro status=ready -->
```console
$ rp achieve reproducible_release --yes
Plan for reproducible_release (2 steps):
  1. build_artifact — produce binary resource
  2. verify_reproducible — observe binary.build_reproducible

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/app.bundle
  approval required: build_artifact
run run-<id> goal evidence requirements not fully satisfied
<root>/.rp/runs/run-<id>
$ rp attest binary.release_approved --source human_review --note "release reviewed"
<root>/.rp/runs/run-<id>
$ rp why binary.build_reproducible
binary.build_reproducible is reproduced
supported by assertion as-step-02-verify_reproducible-binary-build_reproducible from action step-02-verify_reproducible and evidence ev-step-02-verify_reproducible (command_result)
$ rp why binary.release_approved
binary.release_approved is attested
supported by assertion as-manual-attestation-binary-release_approved from action manual-attestation-binary-release_approved and evidence ev-manual-attestation-binary-release_approved (human_review)
```

## Next

- [Bugfix walkthrough](bugfix-walkthrough.md) — the `observed`-only baseline.
- [Translation & trust](translate-doc.md) — when LLM claims cannot satisfy a goal.
- [Evidence & confidence](../internals/evidence.md) — how the ladder is enforced.
- [`rp attest`](../cli/attest.md) — recording human judgement.
