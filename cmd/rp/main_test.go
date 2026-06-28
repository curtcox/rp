package main

import (
	"os"
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
