# Tutorial: conforming a messy dataset

This walkthrough uses the
[`data-conform`](https://github.com/curtcox/rp/tree/main/data-conform)
fixture — a non-code data domain. A messy CSV and a YAML schema spec are
normalized into a `CleanedDataset` with `conforms_to_schema` evidence at
`observed` from a `command_result` verifier. File-backed realization hashing
is enabled by policy.

> Commands run locally under `.rp/runs/`. Work in a throwaway copy if you are
> experimenting.

## 1. Look at the project

Two declared resources — the raw dataset and the schema:

<!-- rp-example: id=conform-resources cwd=conform status=ready -->
```console
$ rp resources
dataset	Dataset	file://data/raw.csv
schema	SchemaSpec	file://schema/spec.yaml
```

## 2. Plan backward from the goal

`rp` derives the cleaned CSV first, then plans schema validation:

<!-- rp-example: id=conform-plan cwd=conform status=ready -->
```console
$ rp plan conforming_dataset
Goal: conforming_dataset
Config: <hash>
Root: <root>
Saved plan: plan-<id>

1. step-01
   capability: normalize_rows
   reason: produce cleaned_dataset resource
2. step-02
   capability: validate_schema
   reason: observe cleaned_dataset.conforms_to_schema

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/cleaned.csv
  approval required: normalize_rows
```

## 3. Achieve and check evidence

<!-- rp-example: id=conform-loop cwd=conform status=ready -->
```console
$ rp achieve conforming_dataset --yes
Plan for conforming_dataset (2 steps):
  1. normalize_rows — produce cleaned_dataset resource
  2. validate_schema — observe cleaned_dataset.conforms_to_schema

Effect summary:
  external: local_process
  filesystem writes:
    - .rp/runs/${run.id}/artifacts/cleaned.csv
  approval required: normalize_rows
run run-<id> goal evidence requirements satisfied
<root>/.rp/runs/run-<id>
$ rp evidence conforming_dataset
Goal conforming_dataset (run run-<id>)
Satisfied: true

Required outputs:
  [ok] cleaned_dataset (CleanedDataset) — text/csv at artifacts/cleaned.csv

Required evidence:
  [ok] cleaned_dataset.conforms_to_schema >= observed — observed via command_result (as-step-02-validate_schema-cleaned_dataset-conforms_to_schema)
```

## Next

- [Translation & trust](translate-doc.md) — when an LLM claim is not enough.
- [Bugfix walkthrough](bugfix-walkthrough.md) — the code-domain baseline.
