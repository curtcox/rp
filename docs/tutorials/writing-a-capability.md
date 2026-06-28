# Tutorial: writing a capability

A **capability** is the contract for an action — what it consumes, what it
produces, the command it runs, and the evidence it yields. This tutorial builds
one from scratch. See the [config reference](../config/reference.md) for the full
schema.

## 1. Scaffold

<!-- rp-example: id=cap-scaffold cwd=empty status=todo -->
```console
$ rp capability init command run_tests
# (output to be captured)
```

## 2. Describe inputs and outputs

A capability `derives` or `observes` resources. An *observer* produces an
assertion rather than a new resource:

```yaml
capabilities:
  run_tests:
    purpose: observe
    kind: command
    inputs:
      repo:
        type: GitRepo
    outputs:
      test_result:
        type: TestReport
        assertions:
          - subject: patched_repo
            predicate: tests_pass
            confidence: observed
            evidence_source: process_exit
    command:
      cwd: "${inputs.repo.path}"
      argv: ["./scripts/run_tests.sh"]
      stdout:
        save_as_artifact: artifacts/pytest.stdout
    effects:
      external: local_process
```

## 3. Declare effects and cost honestly

The planner uses `effects`, `nondeterminism`, `idempotence`, and `cost` to filter
by policy and to build the effect summary. Under-declaring effects defeats the
auditability `rp` is for.

## 4. See it in a plan

Once the capability produces something a goal needs, it appears in the plan:

<!-- rp-example: id=cap-in-plan cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --explain
# (run_tests appears as the step producing patched_repo.tests_pass)
```

## See also

- [Defining a goal](defining-a-goal.md)
- [Config reference: capabilities](../config/reference.md)
