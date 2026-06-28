Below is the consolidated v0.1 specification package from the interview.

rp v0.1 Specification Draft

1. One-page product thesis

rp is a terminal-first resource planner for recursive just-in-time decision and execution built on existing tools and evidence.

It sits between Make/Bazel, PDDL/HTN planning, constraint systems, workflow engines, rule systems, and LLM agent frameworks. Its central purpose is not to run a predefined DAG. Its purpose is to start from a goal, inspect available resources and capabilities, infer what needs to be done, execute approved actions, observe results, and accumulate evidence until the goal is satisfied or the run stops.

The developer shorthand is:

Make for resources, capabilities, goals, and evidence rather than files and timestamps.

Where Make/Bazel ask, “What files must be rebuilt?”, rp asks:

What resources, assertions, and evidence are needed to satisfy this goal under the active policy?

rp does not treat action outputs as facts. Every output is an assertion with evidence. Assertions may be unsupported, claimed, observed, attested, reproduced, or independently reproduced. LLM claims, command outputs, process exits, test reports, human approvals, hashes, API responses, and signed attestations are all evidence sources with different trust implications.

The v0.1 system is intentionally local, CLI-first, inspectable, and small. It executes directly on the host by default. Safety comes from declared effects, policy checks, dry runs, approval gates, provenance, and append-only run logs, not from sandboxing. Sandboxing, remote execution, durable orchestration, distributed workers, built-in LLM providers, and full constraint solving are deferred.

The core runtime loop is:

Plan enough to explain.
Execute one approved step.
Observe.
Record evidence.
Revise the plan.
Repeat.

The first concrete tutorial target is:

Given a local Git repo and a local Markdown bug report, produce a patch artifact and evidence that configured tests pass.

The output is not merely a patch. The output is a patch plus an auditable explanation of why the system believes the patch satisfies the goal.

⸻

2. Glossary

Resource

An abstract typed entity the planner can reason about.

A resource may represent a file, directory, Git repository, bug report, patch, command result, test result, API response, credential reference, human approval, LLM output, observation, or attestation.

A resource is not necessarily a file. It may have zero or more concrete realizations.

Realization

A concrete backing instance of a resource.

Examples:

Resource: repo
Realization: local path file://. at commit abc123
Resource: patch
Realization: .rp/runs/run-123/artifacts/proposed.patch
Resource: bug_report
Realization: bug.md with media type text/markdown

Resource identity and realization identity are distinct core concepts.

Capability

A planner-visible contract describing how an action may derive or observe resources, assertions, or evidence.

A capability declares:

inputs
preconditions
outputs
assertions emitted
evidence emitted
cost
risk
effects
permissions
nondeterminism
idempotence
failure behavior

Action

One concrete invocation of a capability.

Example:

Capability: run_tests
Action: run pytest in repo realization repo.local during run run-123

Goal

A desired resource/artifact plus required evidence and constraints.

Example:

Produce a Patch[text/x-diff]
with evidence:
  patch applies cleanly >= observed
  patched repo tests pass >= observed
under policy:
  no network
  no credentials
  max cost 10 minutes

Constraint

A restriction on valid plans or executions.

Examples:

no network
no credentials
max cost
required evidence level
allowed filesystem writes
required clean Git worktree

Effect

A declared consequence of an action.

Effects may affect the planner state or the outside world.

Planner state is append-only. The outside world may be mutated by effectful actions.

Examples:

append event log entry
write artifact under .rp/runs
modify local worktree
delete file
call network API
use credential
open pull request

Observation

Evidence acquired from the world.

Examples:

process exit code
stdout/stderr hash
file hash
git status output
test report
API response
human approval

Assertion

A claim about a resource, realization, goal, or observation.

Assertions are never accepted as unqualified fact. They are supported by evidence.

Example:

subject: patched_repo
predicate: tests_pass
confidence: observed
evidence: pytest exit code 0

Evidence

Support for an assertion.

Evidence has a source type, provenance, and confidence implications.

Example source types:

llm_claim
process_exit
process_output
filesystem_hash
test_report
api_response
human_review
signed_attestation

Attestation

A durable evidence bundle tying an assertion to inputs, actions, observations, environment, hashes, timestamps, and policy context.

Policy

Rules governing planning, execution, evidence requirements, permissions, costs, approval gates, environment inheritance, hashing, and plan changes.

There are project policy and user policy layers. The most restrictive rule wins unless an explicit override mechanism is later added.

World state

The mutable external world: files, processes, APIs, humans, databases, services, credentials, clocks, random sources.

Planner state

The append-only internal record: resources, realizations, assertions, observations, attestations, actions, runs, plans, and event logs.

rp is evidence-auditable, not truth-sound. It records why assertions are believed, not metaphysical truth.

⸻

3. v0.1 scope

In scope

v0.1 supports:

local CLI workflow
project-local .rp/ directory
YAML authoring files
canonical JSON internal representation
local imports only
built-in primitive resource types
command capabilities
optional built-in observers if simpler
simple backward-chaining planner
provisional full plan display
recursive just-in-time replanning
serial execution
step mode
dry runs
policy-controlled approvals
host execution by default
append-only JSONL event logs
generated run summaries
resource/evidence/provenance inspection
DOT and Mermaid plan export
manual resource/assertion/attestation entry
user and project policy layers
minimal canonical config hashing

Out of scope for v0.1

v0.1 does not require:

sandboxing
remote execution
daemon/server mode
parallel execution
durable timers
background polling
distributed workers
built-in LLM provider abstraction
built-in Git semantics
full structural type system
full constraint solver
SAT/SMT/ASP integration
general expression language
custom programming language
GUI/editor
remote capability packs
stable library API

Deferred but reserved

The model should leave room for:

Docker/Podman sandboxing
Nix/devbox execution environments
Git worktree/temp-copy sandboxing
remote runners
constraint solver integration
ASP/SAT/SMT/MiniZinc planners
LLM-assisted planner plugins
SQLite materialized indexes
signed attestations
shared capability packs
richer structural schemas
long-running orchestration
parallel executor

⸻

4. Minimal formal model

Entities

Resource(id, type, metadata)
Realization(id, resource_id, kind, uri, media_type, hash?, metadata)
Capability(
  id,
  purpose,
  inputs,
  outputs,
  preconditions,
  effects,
  cost,
  risk,
  nondeterminism,
  idempotence
)
Action(id, capability_id, input_bindings, run_id)
Assertion(
  id,
  subject,
  predicate,
  object?,
  confidence,
  evidence_ids,
  supersedes?
)
Evidence(
  id,
  source_type,
  observation_id?,
  action_id?,
  provenance,
  confidence_contribution
)
Observation(
  id,
  source_type,
  payload_ref?,
  hashes?,
  timestamp,
  action_id?
)
Attestation(
  id,
  assertion_ids,
  evidence_ids,
  input_hashes,
  config_hash,
  policy_hash,
  run_id
)
Goal(
  id,
  given,
  produce,
  requires_evidence,
  constraints
)
Policy(
  id,
  permissions,
  approval_rules,
  evidence_rules,
  cost_rules,
  hashing_rules,
  execution_rules
)
Run(
  id,
  goal_id,
  config_hash,
  policy_hash,
  event_log
)

Confidence order

v0.1 core confidence levels:

unsupported
claimed
observed
attested
reproduced
independently_reproduced

These are evidence-status levels, not truth levels.

Open-world semantics

Absence of evidence does not imply falsity.

No evidence that tests pass
means
tests_pass is unknown
not
tests_pass is false

A goal or policy may reject unknown status.

Preconditions

A precondition requires sufficient evidence, not absolute truth.

Capability precondition:
  tests_pass >= observed
Means:
  there exists an assertion supporting tests_pass
  with confidence at least observed
  under the active policy

If evidence is insufficient, the planner may insert observation/validation actions.

State monotonicity

Planner state is append-only.

Actions may mutate the world, but planner history records this by appending new observations, realizations, assertions, and events.

Corrections are represented by supersession, not mutation.

Planning semantics

v0.1 planner:

1. Start from goal outputs and evidence requirements.
2. Backward-chain to find capabilities that can produce missing resources/assertions.
3. Insert observation actions when evidence is missing or insufficient.
4. Filter by policy, permissions, simple constraints, and cost class.
5. Produce a provisional full plan.
6. During execution, invalidate or revise the remaining plan after each observation.

Execution semantics

rp achieve:

load project config
load user and project policies
compute canonical config hash
resolve goal
produce provisional plan
display plan and effect summary
request approvals as policy requires
execute serially
record every action/observation/assertion/resource realization
after each observation, re-evaluate goal gap
continue, replan, stop, or ask according to policy

⸻

5. CLI sketch

Initialization

rp init

Creates:

.rp/
  planner.yaml
  capabilities/
  policies/
  goals/
  runs/      # ignored
  cache/     # ignored

Scaffolding

rp capability init command run_tests
rp capability init command propose_patch
rp goal init bugfix_patch
rp policy init local_safe

Resource management

rp add resource repo \
  --type GitRepo \
  --uri file://.
rp add resource bug_report \
  --type BugReport \
  --file bug.md \
  --media-type text/markdown
rp resources
rp resource repo

Planning

rp plan bugfix_patch
rp plan bugfix_patch --explain
rp plan bugfix_patch --speculative
rp plan bugfix_patch --format json
rp plan bugfix_patch --format dot > plan.dot
rp plan bugfix_patch --format mermaid > plan.mmd

Achieving goals

rp achieve bugfix_patch
rp achieve bugfix_patch --dry-run
rp achieve bugfix_patch --step
rp achieve bugfix_patch --auto-repair --max-attempts 2

Saved plans and runs

rp exec plan-123
rp replan run-123
rp replay run-123
rp rerun run-123

replay reconstructs/explains an event log.
rerun re-executes, subject to policy.

Evidence and provenance

rp evidence bugfix_patch
rp why tests_pass
rp trace patch.out
rp audit run-123

Manual evidence

rp observe repo --with git_status
rp attest tests_pass \
  --source human_review \
  --note "Reviewed test output manually"
rp add assertion patch.addresses_bug \
  --subject patch.out \
  --confidence claimed

⸻

6. Example project

Directory layout

example-project/
  bug.md
  scripts/
    propose_patch.sh
  .rp/
    planner.yaml
    capabilities/
      git.yaml
      patch.yaml
      tests.yaml
    policies/
      local-safe.yaml
    goals/
      bugfix.yaml
    runs/       # ignored
    cache/      # ignored

.rp/planner.yaml

version: rp.dev/v0.1
imports:
  - capabilities/git.yaml
  - capabilities/patch.yaml
  - capabilities/tests.yaml
  - policies/local-safe.yaml
  - goals/bugfix.yaml
resources:
  repo:
    type: GitRepo
    realizations:
      - id: repo.local
        kind: local_path
        uri: file://.
        media_type: inode/directory
  bug_report:
    type: BugReport
    realizations:
      - id: bug_report.markdown
        kind: file
        uri: file://bug.md
        media_type: text/markdown
defaults:
  policy: local_safe

.rp/policies/local-safe.yaml

version: rp.dev/v0.1
policies:
  local_safe:
    description: Local-only development policy with conservative trust defaults.
    permissions:
      filesystem:
        read: allowed
        write: approval_required
        destructive_write: approval_required
      process:
        execute: allowed
      network:
        access: forbidden
      credentials:
        use: forbidden
      external_side_effects:
        create_pull_request: forbidden
        send_message: forbidden
        deploy: forbidden
    environment:
      inherit: false
      allow:
        - PATH
        - HOME
    evidence:
      confidence_order:
        - unsupported
        - claimed
        - observed
        - attested
        - reproduced
        - independently_reproduced
      source_limits:
        - source_type: llm_claim
          max_confidence: claimed
      final_goal_rules:
        - source_type: llm_claim
          may_satisfy_required_evidence: false
    hashing:
      file_backed_realizations: true
      command_stdout: true
      command_stderr: true
      directories: false
      credentials: false
    execution:
      on_failure: stop_and_suggest
      auto_repair:
        enabled: false
        max_attempts: 1
      plan_changes:
        allow_without_confirmation_if_not_increasing:
          - permissions
          - risk
          - cost_class
    max_cost:
      time: 10m
      money_usd: 0
      tokens: 0
      human_attention: low

.rp/goals/bugfix.yaml

version: rp.dev/v0.1
goals:
  bugfix_patch:
    description: Produce a patch for the bug report and evidence that configured tests pass.
    given:
      repo: repo
      bug_report: bug_report
    produce:
      patch:
        type: Patch
        required_realization:
          kind: file
          media_type: text/x-diff
    requires_evidence:
      - subject: patch
        predicate: applies_cleanly
        min_confidence: observed
        any_source_type:
          - process_exit
          - command_result
      - subject: patched_repo
        predicate: tests_pass
        min_confidence: observed
        any_source_type:
          - process_exit
          - test_report
    constraints:
      permissions:
        network: forbidden
        credentials: forbidden
      max_cost:
        time: 10m
        money_usd: 0

.rp/capabilities/git.yaml

version: rp.dev/v0.1
capabilities:
  observe_git_status:
    purpose: observe
    kind: command
    inputs:
      repo:
        type: GitRepo
    outputs:
      status:
        type: CommandResult
        assertions:
          - subject: repo
            predicate: clean_worktree
            confidence: observed
            when:
              exit_code: 0
              stdout_matches: "^$"
            evidence_source: process_exit
    command:
      cwd: "${inputs.repo.path}"
      argv:
        - git
        - status
        - --porcelain
    effects:
      external: local_process
      planner: append_only
      filesystem:
        writes: []
    nondeterminism:
      - external_process
    idempotence: idempotent
    cost:
      time: cheap
      money_usd: 0
      risk: low
      human_attention: none

.rp/capabilities/patch.yaml

version: rp.dev/v0.1
capabilities:
  propose_patch_with_script:
    purpose: derive
    kind: command
    inputs:
      repo:
        type: GitRepo
        requires:
          - predicate: clean_worktree
            min_confidence: observed
      bug_report:
        type: BugReport
        realization:
          media_type: text/markdown
    outputs:
      patch:
        type: Patch
        realization:
          kind: file
          media_type: text/x-diff
        assertions:
          - subject: patch
            predicate: addresses_bug
            confidence: claimed
            evidence_source: process_output
    command:
      cwd: "${inputs.repo.path}"
      argv:
        - ./scripts/propose_patch.sh
        - "${inputs.bug_report.path}"
      stdout:
        save_as:
          resource: patch
          artifact_path: artifacts/proposed.patch
          media_type: text/x-diff
    effects:
      external: local_process
      filesystem:
        writes:
          - ".rp/runs/${run.id}/artifacts/proposed.patch"
    nondeterminism:
      - external_process
    idempotence: unknown
    cost:
      time: moderate
      money_usd: 0
      risk: medium
      human_attention: none

.rp/capabilities/tests.yaml

version: rp.dev/v0.1
capabilities:
  check_patch_applies:
    purpose: observe
    kind: command
    inputs:
      repo:
        type: GitRepo
      patch:
        type: Patch
        realization:
          media_type: text/x-diff
    outputs:
      patch_apply_result:
        type: CommandResult
        assertions:
          - subject: patch
            predicate: applies_cleanly
            confidence: observed
            when:
              exit_code: 0
            evidence_source: process_exit
    command:
      cwd: "${inputs.repo.path}"
      argv:
        - git
        - apply
        - --check
        - "${inputs.patch.path}"
    always_record_result: true
    effects:
      external: local_process
      filesystem:
        writes: []
    nondeterminism:
      - external_process
    idempotence: idempotent
    cost:
      time: cheap
      money_usd: 0
      risk: low
      human_attention: none
  apply_patch_to_worktree:
    purpose: derive
    kind: command
    inputs:
      repo:
        type: GitRepo
      patch:
        type: Patch
        requires:
          - predicate: applies_cleanly
            min_confidence: observed
    outputs:
      patched_repo:
        type: GitRepo
        realization:
          kind: local_path
        assertions:
          - subject: patched_repo
            predicate: derived_from
            object: repo
            confidence: observed
            evidence_source: command_result
    command:
      cwd: "${inputs.repo.path}"
      argv:
        - git
        - apply
        - "${inputs.patch.path}"
    approval:
      required_if:
        - permission: filesystem.write
    always_record_result: true
    effects:
      external: local_filesystem_write
      filesystem:
        writes:
          - "${inputs.repo.path}"
    nondeterminism:
      - external_process
    idempotence: non_idempotent
    cost:
      time: cheap
      money_usd: 0
      risk: medium
      human_attention: approval_if_required
  run_tests:
    purpose: observe
    kind: command
    inputs:
      repo:
        type: GitRepo
    outputs:
      test_result:
        type: TestResult
        assertions:
          - subject: repo
            predicate: tests_pass
            confidence: observed
            when:
              exit_code: 0
            evidence_source: process_exit
    command:
      cwd: "${inputs.repo.path}"
      argv:
        - pytest
      stdout:
        save_as_artifact: pytest.stdout
        media_type: text/plain
      stderr:
        save_as_artifact: pytest.stderr
        media_type: text/plain
    always_record_result: true
    effects:
      external: local_process
      filesystem:
        writes: []
    nondeterminism:
      - external_process
    idempotence: unknown
    cost:
      time: moderate
      money_usd: 0
      risk: low
      human_attention: none

⸻

7. Explicit non-goals

v0.1 should not prematurely become any adjacent system wholesale.

Non-goals:

not a universal workflow engine
not a general programming language
not an autonomous AI agent framework
not a full constraint solver
not a distributed execution system
not a durable orchestration platform
not a CI/CD system
not a secrets manager
not a sandbox runtime
not a GUI workflow builder
not a replacement for Make/Bazel/Airflow/Temporal/Dagster/LangGraph

However, borrowing concepts from any of those systems is acceptable if it simplifies the implementation.

The rule is:

Include only what is needed to recursively choose, execute, observe, and replan toward resource/evidence goals.

⸻

8. Comparison against adjacent systems

System	Primary abstraction	What it does well	What rp borrows	How rp differs
Make	Files, targets, timestamps	Simple local dependency builds	Dependency inference from declared rules	Generalizes beyond files; goals require evidence, not just outputs
Bazel	Hermetic build actions, artifacts	Reproducible large-scale builds	Action contracts, declared inputs/outputs	v0.1 is not hermetic, not distributed, and models assertions/evidence
PDDL / STRIPS	States, predicates, actions	Goal-directed planning	Backward planning from desired state	rp executes real host tools and records evidence/provenance
HTN planners	Tasks decomposed into methods	Procedural decomposition	Capability decomposition patterns	v0.1 is less procedural; no full HTN method language
MiniZinc / OR-Tools	Variables and constraints	Optimization and constraint satisfaction	Future constraint/planner integration	v0.1 uses simple filtering, not a solver
Airflow	Scheduled DAGs	Batch workflow orchestration	Visible DAGs and task logs	rp infers steps from goals; no daemon/scheduler in v0.1
Temporal	Durable workflows	Long-running reliable orchestration	Durable event-history inspiration	rp v0.1 is local CLI, not a durable worker platform
Dagster	Data assets and materializations	Data pipeline asset modeling	Asset/resource lineage	rp is more general and terminal-first; evidence is central
Argo	Kubernetes workflows	Container-native DAG execution	Declarative workflow execution model	rp v0.1 has no Kubernetes/remote execution
Datalog	Facts and rules	Declarative inference	Facts/assertions and derivation	rp actions are real tool invocations with effects/evidence
Prolog	Goals and backward chaining	Logical goal solving	Backward chaining	rp is open-world and evidence-aware, not pure logic programming
ASP	Stable models	Complex combinatorial reasoning	Future planner possibility	Not in v0.1
LangGraph	LLM/agent workflow graph	Agentic control flows	Graph visibility, stepwise execution	rp is not LLM-first; LLM claims cannot satisfy final evidence by default

⸻

9. Likely implementation architecture

Preferred implementation language: Go.

Distribution model:

single rp binary
CLI plus internal modules
no stable library API in v0.1

Internal modules:

cmd/
  rp CLI commands
config/
  YAML loading
  import resolution
  strict schema validation
  canonical JSON generation
  config hashing
model/
  resources
  realizations
  capabilities
  goals
  policies
  assertions
  evidence
  observations
  events
policy/
  project/user policy layering
  permission checks
  approval decisions
  evidence sufficiency checks
  cost/risk comparison
planner/
  planner interface
  v0.1 backward-chaining planner
  goal-gap analysis
  provisional plan generation
  validation action insertion
executor/
  serial action execution
  host command runner
  environment sanitization
  approval prompts
  step mode
  failure handling
evidence/
  assertion recording
  observation normalization
  confidence ordering
  source limits
  attestation bundle generation
store/
  JSONL append-only event log
  run summary generation
  artifact path management
  cache hooks
render/
  terminal plan display
  explanation output
  JSON output
  DOT output
  Mermaid output
inspect/
  why
  trace
  evidence
  audit
  replay

Capability execution

v0.1 capabilities are command capabilities, plus optional built-in observers only where they simplify the vertical slice.

Git is not built in as semantic magic. Git support is provided through command capability templates.

Config and hashing

Authoring files are YAML.

Internal form is canonical JSON.

v0.1 should define enough canonicalization to produce reproducible config hashes:

resolve local imports
parse YAML
reject unknown non-x-* keys
normalize to internal JSON object
sort object keys
use UTF-8
no insignificant whitespace
hash with SHA-256

Run records

Normative concepts:

run started
plan proposed
approval requested
approval granted/denied
action started
action completed
action failed
observation recorded
assertion recorded
resource realization recorded
artifact recorded
plan invalidated/revised
goal satisfied
run stopped

Exact JSON shape may evolve during v0.1.

Initial storage:

.rp/runs/<run-id>/
  events.jsonl
  summary.json
  artifacts/

⸻

10. First five development milestones

Milestone 1 — Minimal vertical slice

Goal:

rp init → load YAML → plan one goal → execute one command → record evidence → explain why

Deliverables:

Go CLI skeleton
.rp/ directory creation
YAML parser
strict schema validation
local imports
simple resource/capability/goal model
one command capability
JSONL event log
rp plan
rp achieve
rp why

Success test:

A fixture project can declare a goal requiring a CommandResult assertion,
run a command, record process exit evidence, and explain the satisfied goal.

⸻

Milestone 2 — Resource/evidence model

Deliverables:

Resource versus realization identity
assertions
observations
evidence source types
confidence ordering
always-record failed action results
supersession/correction model
artifact recording under .rp/runs

Success test:

A failed test command creates a TestResult resource and evidence record,
then rp why explains that the goal is not satisfied because tests_pass is unsupported or failed.

⸻

Milestone 3 — Policy and approval gates

Deliverables:

project policy
user policy
most-restrictive-wins resolution
permission classes
network default forbidden
CredentialRef model
sanitized environment
approval prompts grouped by effect class
dry run
step mode

Success test:

A capability declaring filesystem writes prompts for approval.
A capability declaring network is rejected by local-safe policy.
A command receives only the allowed environment.

⸻

Milestone 4 — Provisional planning and replanning

Deliverables:

backward-chaining planner
evidence gap detection
validation action insertion
provisional full plan
plan invalidation after observations
policy-controlled plan changes
failure behavior: record, stop, suggest replan
rp replan
DOT and Mermaid output

Success test:

A bugfix goal creates a provisional plan:
observe clean tree → propose patch → check patch applies → apply patch → run tests.
If patch apply fails, rp records the failure and suggests a repair/replan path if one exists.

⸻

Milestone 5 — Tutorial bugfix project

Deliverables:

example repo
bug.md
scripts/propose_patch.sh
capability templates for Git-as-commands
patch apply check
test runner
local-safe policy
complete docs walkthrough
audit/replay commands

Success test:

rp achieve bugfix_patch produces:
  .rp/runs/<run>/artifacts/proposed.patch
  pytest stdout/stderr artifacts
  event log
  summary
  evidence that patch applies
  evidence that tests pass
  rp evidence bugfix_patch output
  rp audit run output

⸻

11. Open tensions to keep visible

Evidence model versus complexity

The evidence model is the project’s differentiator, but it can easily become a research project. v0.1 should keep ordinal confidence and source limits, not probabilistic inference.

Host execution versus safety

v0.1 executes directly on the host. This is pragmatic, but every capability must honestly declare effects. The system is auditable, not intrinsically safe.

Plans versus recursive JIT control

Users need a full plan to inspect. But execution must allow observations to invalidate the plan. The spec should keep both: provisional plan display plus stepwise reevaluation.

Generality versus tutorial clarity

The system is general-purpose, but the first tutorial should be concrete and boring: repo + bug report → patch + evidence.

YAML simplicity versus expressiveness

No expression language in v0.1. Fixed substitution only. This will feel restrictive, but it prevents the config format from becoming a programming language too early.

⸻

12. Condensed v0.1 definition

rp v0.1 is a local, terminal-first, evidence-auditable resource planner.

It loads YAML declarations of resources, capabilities, goals, and policies; compiles them to canonical JSON; plans backward from a goal; executes host commands serially under policy; records all observations and assertions in an append-only event log; and explains whether a goal is satisfied by available evidence.

It does not know the world is true. It knows what was claimed, what was observed, what evidence supports each assertion, and what policy judged sufficient.