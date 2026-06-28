package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAssertionMatches(t *testing.T) {
	as := AssertionSpec{When: map[string]interface{}{"exit_code": 0, "stdout_matches": "^$"}}
	if !assertionMatches(as, 0, "") {
		t.Fatal("expected matching assertion")
	}
	if assertionMatches(as, 1, "") {
		t.Fatal("exit code should not match")
	}
	if assertionMatches(as, 0, "dirty\n") {
		t.Fatal("stdout should not match")
	}
}

func TestRenderPlan(t *testing.T) {
	plan := []PlanStep{{Capability: "observe_git_status"}, {Capability: "run_tests"}}
	if got := renderMermaid(plan); got == "" || got[:12] != "flowchart TD" {
		t.Fatalf("unexpected mermaid: %q", got)
	}
}

func TestCheckCapabilityPolicyRejectsForbiddenNetwork(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"network": map[string]interface{}{"access": "forbidden"},
			},
		}},
	}
	capability := Capability{Effects: EffectSpec{Network: map[string]interface{}{"access": true}}}
	if err := checkCapabilityPolicy(cfg, capability); err == nil {
		t.Fatal("expected forbidden network error")
	}
}

func TestNeedsWriteApprovalUsesPolicy(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"filesystem": map[string]interface{}{"write": "approval_required"},
			},
		}},
	}
	capability := Capability{Effects: EffectSpec{Filesystem: map[string][]string{"writes": []string{"out.txt"}}}}
	if !needsWriteApproval(cfg, capability) {
		t.Fatal("expected write approval")
	}
}

func TestGoalSatisfiedHonorsAllowedEvidenceSources(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-1","evidence_source":"llm_claim","action_id":"act-1"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	goal := Goal{RequiresEvidence: []Requirement{{
		Subject:       "patch",
		Predicate:     "applies_cleanly",
		MinConfidence: "observed",
		AnySourceType: []string{"process_exit"},
	}}}
	if goalSatisfied(runDir, goal) {
		t.Fatal("llm_claim should not satisfy process_exit-only requirement")
	}
	goal.RequiresEvidence[0].AnySourceType = []string{"llm_claim"}
	if !goalSatisfied(runDir, goal) {
		t.Fatal("matching source should satisfy requirement")
	}
}

func TestMissingEvidenceHonorsFinalGoalSourceRules(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-1","evidence_source":"llm_claim","action_id":"act-1"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	goal := Goal{RequiresEvidence: []Requirement{{
		Subject: "patch", Predicate: "applies_cleanly", MinConfidence: "observed",
	}}}
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Evidence: map[string]interface{}{
				"final_goal_rules": []interface{}{
					map[string]interface{}{"source_type": "llm_claim", "may_satisfy_required_evidence": false},
				},
			},
		}},
	}
	if len(missingEvidenceWithConfig(runDir, goal, cfg)) != 1 {
		t.Fatal("llm_claim should not satisfy final evidence when policy forbids it")
	}
	cfg.Policies["local_safe"] = Policy{}
	if len(missingEvidenceWithConfig(runDir, goal, cfg)) != 0 {
		t.Fatal("same assertion should satisfy evidence without a forbidding final goal rule")
	}
}

func TestPolicySourceLimitsCapConfidence(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Evidence: map[string]interface{}{
				"source_limits": []interface{}{
					map[string]interface{}{"source_type": "llm_claim", "max_confidence": "claimed"},
				},
			},
		}},
	}
	if got := capConfidenceByPolicy(cfg, "llm_claim", "observed"); got != "claimed" {
		t.Fatalf("expected llm_claim confidence to be capped at claimed, got %q", got)
	}
	if got := capConfidenceByPolicy(cfg, "process_exit", "observed"); got != "observed" {
		t.Fatalf("expected process_exit confidence to remain observed, got %q", got)
	}
}

func TestMissingEvidenceListsUnsatisfiedRequirements(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-1","evidence_source":"process_exit","action_id":"act-1"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	goal := Goal{RequiresEvidence: []Requirement{
		{Subject: "patch", Predicate: "applies_cleanly", MinConfidence: "observed"},
		{Subject: "patched_repo", Predicate: "tests_pass", MinConfidence: "observed"},
	}}
	missing := missingEvidence(runDir, goal)
	if len(missing) != 1 {
		t.Fatalf("expected one missing requirement, got %+v", missing)
	}
	if missing[0].Subject != "patched_repo" || missing[0].Predicate != "tests_pass" {
		t.Fatalf("unexpected missing requirement: %+v", missing[0])
	}
}

func TestObserveGitStatusDoesNotClaimDirtyWorktreeClean(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "init").Run(); err != nil {
		t.Skipf("git init failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(".rp", "runs"), 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
resources:
  repo:
    type: GitRepo
    realizations:
      - id: repo.local
        kind: local_path
        uri: file://.
        media_type: inode/directory
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	if err := os.WriteFile(filepath.Join(".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("untracked.txt", []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := cmdObserve([]string{"repo", "--with", "git_status"}); err != nil {
		t.Fatal(err)
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, as := range assertions {
		if as.Subject == "repo" && as.Predicate == "clean_worktree" {
			t.Fatalf("dirty worktree should not be recorded as clean: %+v", as)
		}
	}
}

func TestParseAssertionTarget(t *testing.T) {
	subject, predicate, err := parseAssertionTarget("patch.addresses_bug", "")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "patch" || predicate != "addresses_bug" {
		t.Fatalf("unexpected target: %s.%s", subject, predicate)
	}
	subject, predicate, err = parseAssertionTarget("tests_pass", "patched_repo")
	if err != nil {
		t.Fatal(err)
	}
	if subject != "patched_repo" || predicate != "tests_pass" {
		t.Fatalf("unexpected override target: %s.%s", subject, predicate)
	}
}

func TestAppendManualAssertionCreatesRunEvidence(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(".rp", "runs"), 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
resources: {}
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	if err := os.WriteFile(filepath.Join(".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
	if err := appendManualAssertion("patch", "addresses_bug", "", "claimed", "human_review", "looks right", "manual-test"); err != nil {
		t.Fatal(err)
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(assertions) != 1 {
		t.Fatalf("expected one assertion, got %d", len(assertions))
	}
	if assertions[0].Subject != "patch" || assertions[0].Predicate != "addresses_bug" || assertions[0].EvidenceSource != "human_review" {
		t.Fatalf("unexpected assertion: %+v", assertions[0])
	}
	events, err := readEvents(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if !eventMatches(events[len(events)-1], "manual-test") {
		t.Fatal("expected trace matching for manual action")
	}
}

func TestPlanSavesSnapshotAndExecRunsIt(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(".rp", 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
resources: {}
capabilities:
  say_ok:
    purpose: observe
    kind: command
    inputs: {}
    outputs:
      result:
        type: CommandResult
        assertions:
          - subject: result
            predicate: completed
            confidence: observed
            when:
              exit_code: 0
            evidence_source: process_exit
    command:
      cwd: "."
      argv:
        - /bin/sh
        - -c
        - 'printf ok'
    effects:
      external: local_process
      filesystem:
        writes: []
goals:
  smoke:
    requires_evidence:
      - subject: result
        predicate: completed
        min_confidence: observed
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	if err := os.WriteFile(filepath.Join(".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
	if err := cmdPlan([]string{"smoke"}); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(".rp", "cache", "plans", "plan-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one saved plan, got %d", len(matches))
	}
	planFile := filepath.Base(matches[0])
	planID := planFile[:len(planFile)-len(".json")]
	if err := cmdExec([]string{planID}); err != nil {
		t.Fatal(err)
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(assertions) != 1 || assertions[0].Subject != "result" || assertions[0].Predicate != "completed" {
		t.Fatalf("unexpected assertions: %+v", assertions)
	}
	events, err := readEvents(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if !eventMatches(events[0], planID) {
		t.Fatal("run_started should reference the executed saved plan")
	}
}
