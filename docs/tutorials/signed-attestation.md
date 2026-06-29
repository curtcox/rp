# Tutorial: signed goal attestation

> **Future work.** Satisfied goals already emit an `attestation_recorded` event
> with assertion ids, evidence ids, input hashes, config hash, policy hash, and
> run id (see `recordGoalAttestation` in `cmd/rp/main.go`). **Cryptographic
> signing and offline verification are not implemented.** This walkthrough
> describes the intended `--attest-sign` and `rp verify-attestation` commands
> from `spec-v01.md`.

After a goal is satisfied, `rp` should write a portable **attestation bundle**
under the run directory and optionally sign it with a named key reference. Third
parties verify the bundle without re-running the plan.

This example uses the
[`example-project`](https://github.com/curtcox/rp/tree/main/example-project)
bugfix goal.

## 1. Achieve with signing enabled

<!-- rp-example: id=signed-achieve cwd=fixture status=todo -->
```console
$ rp achieve bugfix_patch --attest-sign key://local-dev --yes
Plan for bugfix_patch (5 steps):
  1. observe_git_status — observe precondition clean_worktree
  2. propose_patch_with_script — produce patch resource
  3. check_patch_applies — observe patch.applies_cleanly
  4. apply_patch_to_worktree — derive patched_repo
  5. run_tests — observe patched_repo.tests_pass

Effect summary:
  external: local_filesystem_write, local_process
  filesystem writes:
    - ${inputs.repo.path}
    - .rp/runs/${run.id}/artifacts/proposed.patch
  approval required: apply_patch_to_worktree, propose_patch_with_script
run run-<id> goal evidence requirements satisfied
Attestation: att-goal-run-<id>
Signed with: key://local-dev (ed25519)
Wrote: <root>/.rp/runs/run-<id>/attestation.json
<root>/.rp/runs/run-<id>
```

## 2. Attestation bundle contents

The bundle is a JSON document capturing everything needed to audit the satisfied
goal:

<!-- rp-example: id=signed-bundle cwd=fixture status=todo -->
```console
$ cat .rp/runs/run-<id>/attestation.json
{
  "version": "rp.dev/v0.1",
  "run_id": "run-<id>",
  "goal_id": "bugfix_patch",
  "config_hash": "<hash>",
  "policy_hash": "<hash>",
  "input_hashes": {
    "repo": "<hash>",
    "bug_report": "<hash>"
  },
  "resource_hashes": {
    "patch": "<hash>",
    "patched_repo": "<hash>"
  },
  "assertion_ids": [
    "as-step-01-observe_git_status-repo-clean_worktree",
    "as-step-03-check_patch_applies-patch-applies_cleanly",
    "as-step-05-run_tests-patched_repo-tests_pass"
  ],
  "evidence_ids": [
    "ev-step-01-observe_git_status",
    "ev-step-03-check_patch_applies",
    "ev-step-05-run_tests"
  ],
  "satisfied_at": "2026-06-28T21:36:15Z",
  "signature": {
    "key_ref": "key://local-dev",
    "algorithm": "ed25519",
    "signed_fields": [
      "run_id",
      "goal_id",
      "config_hash",
      "policy_hash",
      "input_hashes",
      "resource_hashes",
      "assertion_ids",
      "evidence_ids"
    ],
    "value": "<sig>"
  }
}
```

## 3. Verify offline

Verification re-hashes referenced artifacts, checks the signature, and confirms
the bundle matches the event log:

<!-- rp-example: id=signed-verify cwd=fixture status=todo -->
```console
$ rp verify-attestation .rp/runs/run-<id>/attestation.json
Attestation: att-goal-run-<id>
Goal: bugfix_patch (run run-<id>)
Signature: valid (key://local-dev, ed25519)
Config hash: <hash> (matches bundle)
Policy hash: <hash> (matches bundle)
Assertions: 3 referenced, 3 found in event log
Evidence: 3 referenced, 3 found in event log
Input hashes: repo <hash>, bug_report <hash> (match current project)
Resource hashes: patch <hash>, patched_repo <hash> (match run artifacts)
Result: attestation verified
```

A tampered bundle should fail clearly:

<!-- rp-example: id=signed-verify-fail cwd=fixture status=todo exit=1 -->
```console
$ rp verify-attestation .rp/runs/run-<id>/attestation.json
Attestation: att-goal-run-<id>
Signature: invalid (key://local-dev, ed25519)
Result: attestation verification failed: signature mismatch
```

## What exists today

On a satisfied run, `rp achieve` already appends an `attestation_recorded` event
with `id`, `assertion_ids`, `evidence_ids`, `input_hashes`, `config_hash`,
`policy_hash`, and `run_id`. There is no `attestation.json` file, no
`--attest-sign` flag, and no `rp verify-attestation` command.

## Implementation notes

When this lands, expect:

- `--attest-sign KEY_REF` on `rp achieve` (and optionally `rp exec`) writing
  `attestation.json` beside the run summary.
- Key references resolved through a small local keystore (e.g. `key://local-dev`
  → file under `~/.config/rp/keys/`).
- `rp verify-attestation PATH` that validates signature + cross-checks ids
  against `.rp/runs/<id>/events.jsonl` and artifact hashes.
- The unsigned event log record remains the source of truth; the bundle is an
  export format for sharing.

## Next

- [`rp attest`](../cli/attest.md) — manual confidence attestation (today).
- [Confidence ladder](../concepts/glossary.md) — where `attested` sits.
