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
