# Policy reference

A **policy** governs what a plan is allowed to do. `rp` merges the project
policy with an optional user policy from `~/.config/rp/policy.yaml` using
**most-restrictive-wins** semantics, computes a **policy hash**, and filters the
plan accordingly. Steps a policy forbids never run.

## Policy keys

| Key | Purpose |
| --- | ------- |
| `description` | Human description. |
| `permissions` | Filesystem / process / network / credentials / external-side-effect rules. |
| `environment` | `inherit` plus an `allow` list of env vars passed to commands. |
| `evidence` | Confidence ordering, per-source limits, final-goal rules. |
| `hashing` | What gets hashed (file realizations, stdout/stderr, directories, credentials). |
| `execution` | `on_failure`, `auto_repair`, and `plan_changes` behavior. |
| `max_cost` | Budgets: `time`, `money_usd`, `tokens`, `human_attention`. |

## Example: a conservative local policy

```yaml
policies:
  local_safe:
    description: Local-only development policy with conservative trust defaults.
    permissions:
      filesystem: { read: allowed, write: approval_required, destructive_write: approval_required }
      process:    { execute: allowed }
      network:    { access: forbidden }
      credentials:{ use: forbidden }
      external_side_effects: { create_pull_request: forbidden, send_message: forbidden, deploy: forbidden }
    environment:
      inherit: false
      allow: [PATH, HOME]
    hashing:
      file_backed_realizations: true
      command_stdout: true
      command_stderr: true
      directories: false
      credentials: false
    execution:
      on_failure: stop_and_suggest
      auto_repair: { enabled: false, max_attempts: 1 }
    max_cost: { time: 10m, money_usd: 0, tokens: 0, human_attention: low }
```

## Most-restrictive-wins merge

When the user policy and project policy disagree, the **more restrictive** value
wins (e.g. `allowed` < `approval_required` < `forbidden`; lower cost caps win).
This means a user can tighten, but never loosen, a project's policy.

## Effect on planning

Because policy filtering happens during planning, `plan --explain` shows which
steps need approval and which effects they carry:

<!-- rp-example: id=policy-plan-approval cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --explain
# (see the "approval required" line in the effect summary)
```

## See also

- [Config reference](reference.md) — the rest of the `.rp/` schema.
- [Concepts: policy governs blast radius](../concepts/overview.md).
