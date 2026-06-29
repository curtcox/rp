# Tutorial: translation with independent verification

The [data conformance](data-conform.md) and
[bugfix](bugfix-walkthrough.md) walkthroughs reach `observed` confidence from
process output alone. This tutorial exercises **trust policy**: a `translate`
capability emits `translation_valid` at `claimed` from `llm_claim`, but the
goal requires `observed` evidence from a non-LLM source. Only the independent
`check_translation` verifier satisfies the goal; `rp why` shows the LLM
assertion exists but does not count.

It is driven through
[`translate-doc`](https://github.com/curtcox/rp/tree/main/translate-doc).

## 1. Look at the project

<!-- rp-example: id=translate-resources cwd=translate status=ready -->
```console
$ rp resources
source_doc	SourceDoc	file://readme.en.md
```

Policy `trust_limited` caps `llm_claim` at `claimed` and sets
`final_goal_rules` so an LLM claim may not satisfy required evidence.

## 2. Plan backward from the goal

The planner selects `check_translation` (not `translate`) for the required
evidence because only `command_result` at `observed` can satisfy the goal:

<!-- rp-example: id=translate-plan cwd=translate status=ready -->
```console
$ rp plan translated_readme
Goal: translated_readme
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: translate
   reason: produce translated_doc resource
2. step-02
   capability: check_translation
   reason: observe translated_doc.translation_valid

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/readme.fr.md
  approval required: translate
```

## 3. Achieve, evidence, and why

The translate step records a `claimed` / `llm_claim` assertion, but the goal
is satisfied only after the verifier runs:

<!-- rp-example: id=translate-loop cwd=translate status=ready -->
```console
$ rp achieve translated_readme --yes
Plan for translated_readme (2 steps):
  1. translate — produce translated_doc resource
  2. check_translation — observe translated_doc.translation_valid

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/readme.fr.md
  approval required: translate
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
$ rp evidence translated_readme
Goal translated_readme (run run-<id>)
Satisfied: true

Required outputs:
  [ok] translated_doc (TranslatedDoc) — text/markdown at artifacts/readme.fr.md

Required evidence:
  [ok] translated_doc.translation_valid >= observed — observed via command_result (as-step-02-check_translation-translated_doc-translation_valid)
$ rp why translated_doc.translation_valid
translated_doc.translation_valid is observed
supported by assertion as-step-02-check_translation-translated_doc-translation_valid from action step-02-check_translation and evidence ev-step-02-check_translation (command_result)
```

## Next

- [Release gate & policy](release-gate.md) — when a forbidden side effect blocks the plan.
- [Evidence & confidence](../internals/evidence.md) — how the ladder is enforced.
