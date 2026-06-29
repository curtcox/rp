# Evidence, confidence & attestation

In `rp` an output is never a bare fact — it is an **assertion**: a claim about a
subject, with a predicate, backed by **evidence**, at a **confidence** level.
This page covers how assertions are created, ranked, corrected, and finally rolled
up into a satisfied/not-satisfied decision.

## The confidence ladder

Confidence is an ordered ladder (`internal/model/confidence.go`). Higher beats
lower, and comparisons are by rank, not string:

| Rank | Confidence | Roughly means |
| --- | --- | --- |
| 0 | `unsupported` | no evidence |
| 1 | `claimed` | asserted without verification |
| 2 | `observed` | derived from a real observation (e.g. a process exit) |
| 3 | `attested` | a human or trusted source vouched for it |
| 4 | `reproduced` | reproduced once |
| 5 | `independently_reproduced` | reproduced by an independent source |

`ConfidenceAtLeast(got, min)` is the single predicate used everywhere a
requirement is checked: a requirement asking for `observed` is met by `observed`,
`attested`, `reproduced`, … but not by `claimed`.

## From a command to an assertion

When a step runs (`executeStep`), each of the capability's declared output
`assertions` is evaluated against the result:

1. **Match.** `assertionMatches` checks the assertion's `when` against the exit
   code and/or stdout. Only matching assertions are recorded — e.g. "tests pass"
   is asserted only when the test command exits 0.
2. **Cap by policy.** The declared confidence is passed through
   `capConfidenceByPolicy`, which lowers it to the policy's `source_limits`
   ceiling for that evidence source if the policy is stricter. A capability cannot
   claim more confidence than policy allows for its source — see
   [policy](hashing-and-policy.md#confidence-caps).
3. **Record with provenance.** The resulting `AssertionRecord` carries the
   subject, predicate, object, (capped) confidence, the evidence id and source
   type, and the action id — so every claim traces back to the observation that
   produced it. The underlying observation also records `process_exit` evidence
   contributing `observed`.

## Supersession: corrections without deletion

Evidence is **append-only**. When a new assertion is recorded for a
subject/predicate that already has one, `recordAssertion` does not overwrite it —
it sets `supersedes` to the prior assertion's id and emits an
`assertion_superseded` event. Both records stay in `events.jsonl`.

When `rp` needs the current truth, `effectiveAssertions` filters out every
assertion that something else supersedes, leaving only the live claims. This is
how a re-run or a correction (including a human `rp attest`) updates a conclusion
while the full history remains auditable — which is exactly what `rp why` and `rp
replay` reconstruct.

## Deciding the goal is met

`runPlan` treats a goal as satisfied only when two independent gaps are both
empty:

- **Evidence gap** (`missingEvidenceWithConfig`). For each
  `goal.requires_evidence` requirement, there must be a live assertion that
  matches the subject + predicate, clears the minimum confidence
  (`ConfidenceAtLeast`), comes from an allowed source (`any_source_type`), and
  comes from a source the policy permits for final-goal use
  (`sourceMaySatisfyRequiredEvidence`, from policy `final_goal_rules`).
- **Produce gap** (`missingProduce`). For each `goal.produce` entry, a matching
  resource realization must have been recorded, and its kind/media type must match
  what the goal required (`produceSpecMismatch`).

`rp evidence <goal>` reports both sides so you can see exactly what is still
outstanding. The summary's `reason` distinguishes the three failure shapes
(evidence only, produce only, or both).

## Attestation bundle

When — and only when — both gaps are empty, `recordGoalAttestation` writes the
goal **attestation bundle** and the run summary records `"satisfied": true`. This
is the durable, signed-off artifact that the goal was achieved with the evidence
to back it, captured under a known config and policy hash.

## Manual evidence: `observe` and `attest`

Not all evidence comes from planned steps. `rp observe` records an observation
directly, and `rp attest` records a human assertion (e.g. confidence `attested`)
through the same append-only path (`appendManualAssertion`), including
supersession. Whether such a source can satisfy a *final goal* requirement is
governed by policy `final_goal_rules` — a policy can, for example, accept
`human_review` for preconditions but require machine evidence for the final goal.

## How to use this well

- **Ask for the lowest confidence that actually convinces you.** A
  `requires_evidence` minimum of `observed` is met by a clean command exit;
  demand `reproduced`/`independently_reproduced` only when you mean it, since each
  rung needs more work to reach.
- **Let corrections supersede — don't edit the log.** Re-run or `rp attest` to
  update a claim; the history is the point. Use `rp why <subject.predicate>` to
  see which assertion is currently live and what it superseded.
- **Remember the policy ceiling.** If an assertion lands lower than its capability
  declared, check `source_limits`: policy capped it. Likewise a satisfied-looking
  precondition that won't satisfy the *goal* is usually a `final_goal_rules`
  restriction.
- **Watch both gaps.** A green-looking run can still be unsatisfied because of a
  `produce` mismatch (wrong kind/media type), not missing evidence. `rp evidence`
  separates the two.
