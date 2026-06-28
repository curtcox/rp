# Tutorial: defining a goal

A **goal** states what you want to be true: the resources to **produce** and the
**evidence required** to believe the goal is met. `rp` plans backward from it.

## 1. Scaffold

<!-- rp-example: id=goal-scaffold cwd=empty status=todo -->
```console
$ rp goal init bugfix_patch
# (output to be captured)
```

## 2. Say what to produce

```yaml
goals:
  bugfix_patch:
    description: Produce a patch for the bug report and evidence that tests pass.
    given: { repo: repo, bug_report: bug_report }
    produce:
      patch:
        type: Patch
        required_realization: { kind: file, media_type: text/x-diff }
```

## 3. Require evidence, not just output

The difference between `rp` and a build tool: a goal is not done because a file
exists — it is done when the **evidence** meets a minimum confidence.

```yaml
    requires_evidence:
      - subject: patch
        predicate: applies_cleanly
        min_confidence: observed
        any_source_type: [process_exit, command_result]
      - subject: patched_repo
        predicate: tests_pass
        min_confidence: observed
        any_source_type: [process_exit, test_report]
```

See the [confidence ladder](../concepts/glossary.md#the-confidence-ladder).

## 4. Constrain blast radius

```yaml
    constraints:
      permissions: { network: forbidden, credentials: forbidden }
      max_cost: { time: 10m, money_usd: 0 }
```

## 5. Plan it

<!-- rp-example: id=goal-plan cwd=fixture status=todo -->
```console
$ rp plan bugfix_patch --explain
# (output to be captured)
```

## 6. Check evidence after a run

<!-- rp-example: id=goal-evidence cwd=fixture status=todo -->
```console
$ rp evidence bugfix_patch
# (output to be captured)
```

## See also

- [Writing a capability](writing-a-capability.md)
- [Config reference: goals](../config/reference.md)
