# Concepts: the model end to end

`rp` treats *getting work done* the way Make treats *building files* — but the
unit is not a file with a timestamp, it is a **resource** with **evidence**.

## The chain

1. You declare **resources** (typed entities) and **capabilities** (contracts
   for actions that derive or observe resources).
2. You declare a **goal**: what to **produce** and what **evidence** is required
   to believe the goal is met.
3. `rp` **plans backward** from the goal: which capability produces each needed
   resource or assertion, recursively, until it reaches what you already have.
4. `rp` **executes** the plan one step at a time. Each step runs a real command,
   with preconditions checked at execution time and **just-in-time replanning**
   between steps.
5. Each output becomes an **assertion** — a claim with **evidence** at a
   **confidence** level, never a bare fact.
6. The goal is satisfied only when every required output exists and every
   required assertion meets its minimum confidence. `rp` then writes a goal
   **attestation bundle**.

## Evidence is append-only

Runs record events to an append-only `events.jsonl`. Nothing is overwritten; when
a claim is corrected, the new assertion **supersedes** the old one and both
remain in the record. That is what makes a run *auditable*: you can always
reconstruct how `rp` arrived at a conclusion (see [replay / audit](../cli/runs.md)).

## Policy governs blast radius

A **policy** (project-local, merged most-restrictively with an optional user
policy) governs permissions (network, credentials, filesystem), hashing,
execution behavior (auto-repair, on-failure), and cost budgets. Steps that a
policy forbids are filtered out of the plan. See
[config: policy](../config/policy.md).

## Where to look next

- [Glossary](glossary.md) — precise definitions and the confidence ladder.
- [Config reference](../config/reference.md) — how the model is written in YAML.
- [Tutorials](../tutorials/bugfix-walkthrough.md) — the model in action.
