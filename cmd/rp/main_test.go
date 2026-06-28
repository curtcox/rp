package main

import "testing"

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
