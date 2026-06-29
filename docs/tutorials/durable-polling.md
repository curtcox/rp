# Tutorial: durable polling goal

> **Future work.** `rp achieve` runs synchronously to completion or failure.
> Durable timers, background polling, and long-running orchestration are out of
> scope for v0.1 (`spec-v01.md`). There is no `rp resume` command today (only
> internal resume hooks for replan). This walkthrough describes how a deploy
> goal should pause, schedule a poll, and resume later when a remote job
> completes.

The scenario: `deploy_verified` triggers a deploy, then must observe
`deployment.ready` on a remote job resource that returns `pending` initially.
The run pauses with a durable poll recorded; the user (or a scheduler) resumes
later when the deployment finishes.

## 1. Goal and remote job resource

```yaml
# .rp/planner.yaml (excerpt — future)
resources:
  staging_env:
    type: DeployTarget
    realizations:
      - id: staging.api
        kind: uri
        uri: https://staging.example.internal
        media_type: application/json

# .rp/goals/deploy.yaml (excerpt — future)
deploy_verified:
  given:
    staging_env: staging_env
  requires_evidence:
    - subject: deployment
      predicate: ready
      min_confidence: observed
      any_source_type:
        - api_response
  constraints:
    polling:
      interval: 30s
      timeout: 15m
      durable: true
```

```yaml
# .rp/capabilities/deploy.yaml (excerpt — future)
trigger_deploy:
  purpose: derive
  outputs:
    deployment:
      type: RemoteJob
      realization:
        kind: uri
        uri: "${outputs.job_url}"
  command:
    argv: ["./scripts/trigger_deploy.sh"]

poll_deployment:
  purpose: observe
  inputs:
    deployment:
      type: RemoteJob
  outputs:
    poll_result:
      assertions:
        - subject: deployment
          predicate: ready
          confidence: observed
          when:
            json_path: ".status"
            equals: "ready"
          evidence_source: api_response
  command:
    argv: ["./scripts/poll_deployment.sh", "${inputs.deployment.uri}"]
```

## 2. Achieve pauses with a durable poll scheduled

The deploy step succeeds and returns a job URL. The first poll sees
`status: pending`; the run pauses instead of blocking the terminal for 15
minutes:

<!-- rp-example: id=poll-achieve-pause cwd=empty status=todo -->
```console
$ rp achieve deploy_verified --yes
Plan for deploy_verified (2 steps):
  1. trigger_deploy — derive deployment resource
  2. poll_deployment — observe deployment.ready

Effect summary:
  external: network, local_process
  approval required: trigger_deploy
run run-<id> paused (durable poll scheduled)
Poll: deployment.ready every 30s (timeout 15m, next at 2026-06-28T21:37:00Z)
Resume with: rp resume run-<id>
<root>/.rp/runs/run-<id>
```

Event log should record the pause:

<!-- rp-example: id=poll-run-status cwd=empty status=todo -->
```console
$ rp runs --goal deploy_verified
run-<id>  deploy_verified  paused  poll scheduled (deployment.ready)
```

## 3. Resume when the deployment is ready

After the remote job transitions to `ready`, resume continues polling (or
observes cached state), records observation evidence, and satisfies the goal:

<!-- rp-example: id=poll-resume cwd=empty status=todo -->
```console
$ rp resume run-<id>
Resuming run-<id> (deploy_verified)
Poll deployment: status ready (was pending)
Recorded: deployment.ready observed via api_response
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
$ rp evidence deploy_verified
Goal deploy_verified (run run-<id>)
Satisfied: true

Required evidence:
  [ok] deployment.ready >= observed — observed via api_response (as-step-02-poll_deployment-deployment-ready)
```

If the timeout elapses before `ready`, resume should fail with a clear reason
and leave the run resumable only via replan:

<!-- rp-example: id=poll-resume-timeout cwd=empty status=todo exit=1 -->
```console
$ rp resume run-<id>
Resuming run-<id> (deploy_verified)
Poll deployment: status pending (elapsed 15m, timeout exceeded)
run run-<id> failed (poll timeout)
rp: deployment.ready not observed within polling.timeout 15m
```

## What exists today

- `rp achieve` blocks until all steps finish or fail; no pause/poll state.
- No `polling:` goal constraints or `rp resume` subcommand.
- Observations are normalized for command/process exits and `git_status`, not
  polled API JSON paths.

## Implementation notes

When this lands, expect:

- Run state `paused` with a `poll_scheduled` event (interval, timeout, next_at,
  subject/predicate).
- `rp resume RUN_ID` to continue from the last incomplete step (distinct from
  `rp replan`, which rebuilds the plan).
- Optional background mode (daemon or cron-friendly `rp poll --due`) that resumes
  due runs without blocking a terminal.
- Policy gates on network polling capabilities (reuse existing network
  permission checks).

## Next

- [`rp replan`](../cli/runs.md) — rebuild and continue after failure (today).
- [Release gate](release-gate.md) — synchronous multi-step goals today.
