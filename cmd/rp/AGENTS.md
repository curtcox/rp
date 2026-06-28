# cmd/rp — code map

Almost everything is in `main.go` (~3,500 lines). Functions are grouped by
concern below with approximate line locations — grep the function name to
confirm, since line numbers drift.

- **Dispatch / usage** — `main`, `run`, `usage` (~53–136). `run` is a `switch` on the verb; each `case` calls a `cmdXxx` handler.
- **Command handlers** (`cmdXxx`) — `cmdInit`, `cmdCapability`, `cmdGoal`, `cmdPolicy`, `cmdAdd`, `cmdAddAssertion`, `cmdResources`, `cmdResource`, `cmdPlan`, `cmdAchieve`, `cmdEvidence`, `cmdWhy`, `cmdTrace`, `cmdObserve`, `cmdAttest`, `cmdAudit`, `cmdReplay`, `cmdReplan`, `cmdRerun`, `cmdExec` (~138–1130).
- **Project & config loading** — `loadProject`, `findProjectRoot`, `loadConfig`, `loadSingleConfig`, `mergeConfig` (~1130, 1365–1430, 1734).
- **Policy merge** (most-restrictive-wins) — `applyUserPolicy`, `loadUserPolicy`, `mergePolicies`, `mergePermissionMaps`, `mostRestrictivePermission`, `minCostString` (~1147–1365).
- **YAML schema validation** (unknown-key rejection) — `validateYAMLDocument`, `validateMapping`, `validateNamedItems`, `validateCapabilityBody`, `validateInputOutputBody`, `isExtensionKey` (~1538–1733).
- **Planning** (backward chaining) — `buildPlan`, `planStepPriority`, `filterPlanByPolicy`, `filterPlanByConstraints`, `planCapabilities` (~1770–1900, 2549–2577).
- **Execution** — `runPlan`, `executeStep`, `substitute`, `saveStreams`, `writeArtifact` (~436, 1897–2045).
- **Runs & events** (append-only JSONL) — `newRun`, `openRun`, `appendEvent`, `writeSummary`, `readEvents`, `recordGoalAttestation`, `latestRunDir` (~2045–2200, 2417, 2507).
- **Goal satisfaction / evidence** — `goalSatisfied`, `realizationsFromRun`, `missingProduce`, `missingEvidence` (~2237–2330).
- **Assertions** — `assertionsFromRun`, `effectiveAssertions`, `recordAssertion`, `latestAssertion`, `appendManualAssertion` (~2330–2500).
- **Cost / budgets** — `effectiveMaxCost`, `minCostLimit`, `parseDurationBudget`, `capabilityEstimatedMinutes`, `costString` (~2577–2660).

## Editing tips

- A new CLI verb = add a `case` in `run` + a `cmdXxx` handler.
- A new config field = add the struct tag in `internal/model` **and** the
  matching `validate*` allow-list here, or YAML loading will reject it (unknown
  keys are rejected unless prefixed `x-`).
- The `type (...)` alias block at `main.go:29-51` re-exports `internal/model`
  types under short names; add new shared types in `internal/model`.
- Tests in `main_test.go` drive the CLI through `run([]string{...})` end to end;
  prefer adding to that style over unit-testing private helpers.
