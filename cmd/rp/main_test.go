package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildPlanBugfixOrdering(t *testing.T) {
	cfg := Config{
		Resources: map[string]Resource{
			"repo":       {Type: "GitRepo"},
			"bug_report": {Type: "BugReport"},
		},
		Capabilities: map[string]Capability{
			"run_tests": {
				Purpose: "observe",
				Outputs: map[string]OutputSpec{"test_result": {Assertions: []AssertionSpec{{Predicate: "tests_pass"}}}},
			},
			"apply_patch_to_worktree": {
				Purpose: "derive",
				Outputs: map[string]OutputSpec{"patched_repo": {}},
			},
			"check_patch_applies": {
				Purpose: "observe",
				Outputs: map[string]OutputSpec{"patch_apply_result": {Assertions: []AssertionSpec{{Predicate: "applies_cleanly"}}}},
			},
			"propose_patch_with_script": {
				Purpose: "derive",
				Inputs:  map[string]InputSpec{"repo": {Requires: []Requirement{{Predicate: "clean_worktree"}}}},
				Outputs: map[string]OutputSpec{"patch": {}},
			},
			"observe_git_status": {
				Purpose: "observe",
				Outputs: map[string]OutputSpec{"status": {Assertions: []AssertionSpec{{Predicate: "clean_worktree"}}}},
			},
		},
		Goals: map[string]Goal{
			"bugfix_patch": {
				Given:   map[string]string{"repo": "repo", "bug_report": "bug_report"},
				Produce: map[string]OutputSpec{"patch": {}},
				RequiresEvidence: []Requirement{
					{Subject: "patch", Predicate: "applies_cleanly", MinConfidence: "observed"},
					{Subject: "patched_repo", Predicate: "tests_pass", MinConfidence: "observed"},
				},
			},
		},
	}
	plan, err := buildPlan(cfg, "bugfix_patch")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"observe_git_status",
		"propose_patch_with_script",
		"check_patch_applies",
		"apply_patch_to_worktree",
		"run_tests",
	}
	if len(plan) != len(want) {
		t.Fatalf("expected %d steps, got %d: %+v", len(want), len(plan), plan)
	}
	for i, name := range want {
		if plan[i].Capability != name {
			t.Fatalf("step %d: expected %s, got %s", i+1, name, plan[i].Capability)
		}
	}
}

func TestBuildPlanGeneralizesBeyondBugfix(t *testing.T) {
	// A non-bugfix goal: produce a build artifact, require it to be reproduced,
	// and require a human attestation that no capability can satisfy. The planner
	// must plan the producing capability (not just the asserting one), order the
	// producer before its consumer, and add no step for the manual attestation.
	cfg := Config{
		Resources: map[string]Resource{
			"source": {Type: "SourceTree"},
		},
		Capabilities: map[string]Capability{
			"verify_reproducible": {
				Purpose: "observe",
				Inputs: map[string]InputSpec{
					"source": {Type: "SourceTree"},
					"binary": {Type: "BuildArtifact"},
				},
				Outputs: map[string]OutputSpec{"check": {Assertions: []AssertionSpec{
					{Subject: "binary", Predicate: "build_reproducible"},
				}}},
			},
			"build_artifact": {
				Purpose: "derive",
				Inputs:  map[string]InputSpec{"source": {Type: "SourceTree"}},
				Outputs: map[string]OutputSpec{"binary": {Assertions: []AssertionSpec{
					{Subject: "binary", Predicate: "built"},
				}}},
			},
		},
		Goals: map[string]Goal{
			"reproducible_release": {
				Given:   map[string]string{"source": "source"},
				Produce: map[string]OutputSpec{"binary": {}},
				RequiresEvidence: []Requirement{
					{Subject: "binary", Predicate: "build_reproducible", MinConfidence: "reproduced"},
					{Subject: "binary", Predicate: "release_approved", MinConfidence: "attested"},
				},
			},
		},
	}
	plan, err := buildPlan(cfg, "reproducible_release")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"build_artifact", "verify_reproducible"}
	if len(plan) != len(want) {
		t.Fatalf("expected %d steps, got %d: %+v", len(want), len(plan), plan)
	}
	for i, name := range want {
		if plan[i].Capability != name {
			t.Fatalf("step %d: expected %s, got %s", i+1, name, plan[i].Capability)
		}
	}
	// verify_reproducible must know its binary input is the produced resource.
	if got := plan[1].Inputs["binary"]; got != "binary" {
		t.Fatalf("verify_reproducible binary input = %q, want %q", got, "binary")
	}
}

func TestMergePoliciesMostRestrictive(t *testing.T) {
	project := Policy{Permissions: map[string]interface{}{
		"network":     map[string]interface{}{"access": "allowed"},
		"filesystem":  map[string]interface{}{"write": "allowed"},
		"credentials": map[string]interface{}{"use": "forbidden"},
	}}
	user := Policy{Permissions: map[string]interface{}{
		"network":    map[string]interface{}{"access": "forbidden"},
		"filesystem": map[string]interface{}{"write": "approval_required"},
	}}
	merged := mergePolicies(project, user)
	if permissionValue(merged.Permissions, "network", "access") != "forbidden" {
		t.Fatal("network should be forbidden")
	}
	if permissionValue(merged.Permissions, "filesystem", "write") != "approval_required" {
		t.Fatal("filesystem write should require approval")
	}
	if permissionValue(merged.Permissions, "credentials", "use") != "forbidden" {
		t.Fatal("project credential rule should remain")
	}
}

func TestExampleProjectBugfixAchieve(t *testing.T) {
	exampleRoot := filepath.Join("..", "..", "example-project")
	if _, err := os.Stat(filepath.Join(exampleRoot, ".rp", "planner.yaml")); err != nil {
		t.Skip("example-project fixture not present")
	}
	dir := t.TempDir()
	copyDir(t, exampleRoot, dir)
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
	if err := exec.Command("git", "add", "-A").Run(); err != nil {
		t.Fatal(err)
	}
	// CI runners have no global git identity; commit must supply one locally.
	if err := exec.Command("git", "-c", "user.email=rp-test@example.com", "-c", "user.name=rp test", "commit", "-m", "initial").Run(); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"achieve", "bugfix_patch", "--yes"}); err != nil {
		t.Fatal(err)
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"artifacts/proposed.patch", "artifacts/pytest.stdout", "events.jsonl", "summary.json"} {
		if _, err := os.Stat(filepath.Join(runDir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	summaryBytes, err := os.ReadFile(filepath.Join(runDir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summaryBytes), `"satisfied": true`) {
		t.Fatalf("expected satisfied summary, got %s", summaryBytes)
	}
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		t.Fatal(err)
	}
	foundApply, foundTests := false, false
	for _, as := range assertions {
		if as.Subject == "patch" && as.Predicate == "applies_cleanly" {
			foundApply = true
		}
		if as.Subject == "patched_repo" && as.Predicate == "tests_pass" {
			foundTests = true
		}
	}
	if !foundApply || !foundTests {
		t.Fatalf("missing goal assertions: apply=%v tests=%v all=%+v", foundApply, foundTests, assertions)
	}
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(filepath.Separator)) {
			return nil
		}
		if strings.HasPrefix(rel, ".rp"+string(filepath.Separator)+"runs") || strings.HasPrefix(rel, ".rp"+string(filepath.Separator)+"cache") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
	if err != nil {
		t.Fatal(err)
	}
}

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

func TestSummarizePlanEffects(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {Permissions: map[string]interface{}{
			"filesystem": map[string]interface{}{"write": "approval_required"},
		}}},
		Capabilities: map[string]Capability{
			"observe_git_status": {Effects: EffectSpec{External: "local_process", Filesystem: map[string][]string{"writes": {}}}},
			"apply_patch_to_worktree": {
				Effects: EffectSpec{
					External:   "local_filesystem_write",
					Filesystem: map[string][]string{"writes": {"${inputs.repo.path}"}},
				},
			},
		},
	}
	plan := []PlanStep{
		{Capability: "observe_git_status"},
		{Capability: "apply_patch_to_worktree"},
	}
	summary := summarizePlanEffects(cfg, plan)
	if len(summary.External) != 2 {
		t.Fatalf("expected two external effects, got %+v", summary.External)
	}
	if len(summary.FilesystemWrites) != 1 {
		t.Fatalf("expected one filesystem write, got %+v", summary.FilesystemWrites)
	}
	if len(summary.NeedsApproval) != 1 || summary.NeedsApproval[0] != "apply_patch_to_worktree" {
		t.Fatalf("unexpected approval list: %+v", summary.NeedsApproval)
	}
}

func TestPolicyHashingHonorsPolicy(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {Hashing: map[string]interface{}{
			"command_stdout": true, "file_backed_realizations": false,
		}}},
	}
	if !policyHashing(cfg, "command_stdout") {
		t.Fatal("command_stdout should be enabled")
	}
	if policyHashing(cfg, "file_backed_realizations") {
		t.Fatal("file_backed_realizations should be disabled")
	}
	if policyHashing(cfg, "command_stderr") {
		t.Fatal("command_stderr should default to disabled")
	}
}

func TestPlanAssumptionsListsPreconditions(t *testing.T) {
	cfg := Config{
		Capabilities: map[string]Capability{
			"propose_patch_with_script": {
				Inputs: map[string]InputSpec{
					"repo": {Requires: []Requirement{{Predicate: "clean_worktree", MinConfidence: "observed"}}},
				},
			},
			"apply_patch_to_worktree": {
				Inputs: map[string]InputSpec{
					"patch": {Requires: []Requirement{{Predicate: "applies_cleanly", MinConfidence: "observed"}}},
				},
			},
		},
	}
	plan := []PlanStep{
		{Capability: "propose_patch_with_script", Inputs: map[string]string{"repo": "repo"}},
		{Capability: "apply_patch_to_worktree", Inputs: map[string]string{"repo": "repo", "patch": "patch"}},
	}
	assumptions := planAssumptions(cfg, plan)
	if len(assumptions) != 2 {
		t.Fatalf("expected two assumptions, got %+v", assumptions)
	}
}

func TestSpeculativePlanDoesNotSave(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(dir, ".rp", "cache", "plans"), 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
capabilities:
  ok:
    purpose: observe
    kind: command
    inputs: {}
    outputs:
      result:
        type: CommandResult
    command:
      cwd: "."
      argv: [echo, ok]
    effects:
      external: local_process
      filesystem:
        writes: []
goals:
  smoke:
    requires_evidence: []
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	if err := os.WriteFile(filepath.Join(dir, ".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cmdPlan([]string{"smoke", "--speculative"})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "speculative") {
		t.Fatalf("expected speculative output, got %q", buf.String())
	}
	entries, err := os.ReadDir(filepath.Join(dir, ".rp", "cache", "plans"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("speculative plan should not save snapshot, found %d files", len(entries))
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
	capability := Capability{Effects: EffectSpec{Filesystem: map[string][]string{"writes": {"out.txt"}}}}
	if !capabilityNeedsApproval(cfg, capability) {
		t.Fatal("expected write approval")
	}
}

func TestCapabilityApprovalRequiredIf(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"filesystem": map[string]interface{}{"write": "approval_required"},
			},
		}},
	}
	capability := Capability{
		Approval: map[string]interface{}{
			"required_if": []interface{}{
				map[string]interface{}{"permission": "filesystem.write"},
			},
		},
	}
	if !capabilityNeedsApproval(cfg, capability) {
		t.Fatal("expected approval from required_if")
	}
}

func TestCheckGoalConstraintsRejectsNetwork(t *testing.T) {
	goal := Goal{Constraints: map[string]interface{}{
		"permissions": map[string]interface{}{"network": "forbidden"},
	}}
	cap := Capability{Effects: EffectSpec{Network: map[string]interface{}{"access": true}}}
	if err := checkGoalConstraints(goal, cap); err == nil {
		t.Fatal("expected goal network constraint error")
	}
}

func TestCheckCapabilityPolicyRejectsForbiddenProcess(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"process": map[string]interface{}{"execute": "forbidden"},
			},
		}},
	}
	capability := Capability{Effects: EffectSpec{External: "local_process"}}
	if err := checkCapabilityPolicy(cfg, capability); err == nil {
		t.Fatal("expected forbidden process execution error")
	}
}

func TestPlanRevisionNeedsConfirmation(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Execution: map[string]interface{}{
				"plan_changes": map[string]interface{}{
					"allow_without_confirmation_if_not_increasing": []interface{}{"permissions", "risk", "cost_class"},
				},
			},
		}},
		Capabilities: map[string]Capability{
			"observe": {Effects: EffectSpec{External: "local_process"}, Cost: map[string]interface{}{"risk": "low", "time": "cheap"}},
			"apply": {
				Effects: EffectSpec{External: "local_filesystem_write", Filesystem: map[string][]string{"writes": {"."}}},
				Cost:    map[string]interface{}{"risk": "medium", "time": "cheap"},
			},
		},
	}
	confirm, reason := planRevisionNeedsConfirmation(cfg, []string{"observe"}, []string{"observe", "apply"})
	if !confirm || !strings.Contains(reason, "permissions") {
		t.Fatalf("expected permissions increase confirmation, got confirm=%v reason=%q", confirm, reason)
	}
	confirm, _ = planRevisionNeedsConfirmation(cfg, []string{"observe"}, []string{"observe"})
	if confirm {
		t.Fatal("unchanged plan should not require confirmation")
	}
}

func TestFilterPlanByPolicyRemovesForbiddenNetwork(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"network": map[string]interface{}{"access": "forbidden"},
			},
		}},
		Capabilities: map[string]Capability{
			"ok":      {Effects: EffectSpec{External: "local_process"}},
			"network": {Effects: EffectSpec{Network: map[string]interface{}{"access": true}}},
		},
	}
	steps := filterPlanByPolicy(cfg, []PlanStep{{Capability: "ok"}, {Capability: "network"}})
	if len(steps) != 1 || steps[0].Capability != "ok" {
		t.Fatalf("expected only ok step, got %+v", steps)
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
	if goalSatisfied(runDir, goal, Config{}) {
		t.Fatal("llm_claim should not satisfy process_exit-only requirement")
	}
	goal.RequiresEvidence[0].AnySourceType = []string{"llm_claim"}
	if !goalSatisfied(runDir, goal, Config{}) {
		t.Fatal("matching source should satisfy requirement")
	}
}

func TestEvidenceAggregatesAcrossRuns(t *testing.T) {
	// Evidence accumulated in separate runs (e.g. an achieve run plus a later
	// attest run) must combine to satisfy a goal.
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
	mkRun := func(id, events string) {
		runDir := filepath.Join(dir, ".rp", "runs", id)
		if err := os.MkdirAll(runDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), []byte(events), 0644); err != nil {
			t.Fatal(err)
		}
	}
	mkRun("run-20260101T000000.000000000Z",
		`{"type":"assertion_recorded","data":{"id":"as-1","subject":"binary","predicate":"build_reproducible","confidence":"reproduced","evidence_id":"ev-1","evidence_source":"command_result","action_id":"act-1"}}`+"\n")
	mkRun("run-20260102T000000.000000000Z",
		`{"type":"assertion_recorded","data":{"id":"as-2","subject":"binary","predicate":"release_approved","confidence":"attested","evidence_id":"ev-2","evidence_source":"human_review","action_id":"act-2"}}`+"\n")
	if err := os.WriteFile(filepath.Join(dir, ".rp", "planner.yaml"), []byte("version: rp.dev/v0.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	goal := Goal{RequiresEvidence: []Requirement{
		{Subject: "binary", Predicate: "build_reproducible", MinConfidence: "reproduced"},
		{Subject: "binary", Predicate: "release_approved", MinConfidence: "attested"},
	}}
	if missing := missingEvidenceAcross(goal, Config{}); len(missing) != 0 {
		t.Fatalf("expected no missing evidence across runs, got %+v", missing)
	}
	// A single run only carries half the evidence.
	latest, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	if missing := missingEvidenceWithConfig(latest, goal, Config{}); len(missing) != 1 {
		t.Fatalf("expected one requirement missing within a single run, got %+v", missing)
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

func TestStrictYAMLRejectsUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	body := []byte(`version: rp.dev/v0.1
capabilities:
  bad:
    purpose: observe
    kind: command
    not_a_real_field: true
    inputs: {}
    outputs: {}
    command:
      cwd: "."
      argv: [echo, ok]
    effects:
      external: local_process
      filesystem:
        writes: []
`)
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSingleConfig(path); err == nil {
		t.Fatal("expected unknown field error")
	} else if !strings.Contains(err.Error(), "not_a_real_field") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStrictYAMLAllowsExtensionFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ext.yaml")
	body := []byte(`version: rp.dev/v0.1
x-notes: tutorial only
capabilities:
  ok:
    purpose: observe
    kind: command
    x-custom: true
    inputs: {}
    outputs:
      result:
        type: CommandResult
    command:
      cwd: "."
      argv: [echo, ok]
    effects:
      external: local_process
      filesystem:
        writes: []
`)
	if err := os.WriteFile(path, body, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadSingleConfig(path); err != nil {
		t.Fatalf("extension fields should be allowed: %v", err)
	}
}

func TestFailedActionEmitsActionFailedEvent(t *testing.T) {
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
	if err := os.MkdirAll(".rp/runs", 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
capabilities:
  fail_cmd:
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
    command:
      cwd: "."
      argv: [/bin/sh, -c, "exit 7"]
    always_record_result: true
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
	if err := os.WriteFile(".rp/planner.yaml", planner, 0644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"achieve", "smoke"})
	if err == nil {
		t.Fatal("expected achieve to fail")
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	events, err := readEvents(runDir)
	if err != nil {
		t.Fatal(err)
	}
	foundFailed, foundObservation := false, false
	for _, ev := range events {
		if ev.Type == "action_failed" {
			foundFailed = true
		}
		if ev.Type == "observation_recorded" && ev.ActionID != "" {
			foundObservation = true
		}
	}
	if !foundFailed {
		t.Fatal("expected action_failed event")
	}
	if !foundObservation {
		t.Fatal("expected observation even on failure with always_record_result")
	}
}

func TestCheckStepPreconditions(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"repo","predicate":"clean_worktree","confidence":"observed","evidence_id":"ev-1","evidence_source":"process_exit","action_id":"act-1"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Capabilities: map[string]Capability{
			"needs_clean": {
				Inputs: map[string]InputSpec{
					"repo": {Requires: []Requirement{{Predicate: "clean_worktree", MinConfidence: "observed"}}},
				},
			},
		},
	}
	step := PlanStep{Capability: "needs_clean", Inputs: map[string]string{"repo": "repo"}}
	if err := checkStepPreconditions(runDir, cfg, step); err != nil {
		t.Fatalf("precondition should be satisfied: %v", err)
	}
	step = PlanStep{Capability: "needs_clean", Inputs: map[string]string{"repo": "other"}}
	if err := checkStepPreconditions(runDir, cfg, step); err == nil {
		t.Fatal("expected missing precondition for other repo")
	}
}

func TestAssertionSupersessionKeepsLatestEffective(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"claimed","evidence_id":"ev-1","evidence_source":"llm_claim","action_id":"act-1"}}
{"type":"assertion_recorded","data":{"id":"as-2","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-2","evidence_source":"process_exit","action_id":"act-2","supersedes":"as-1"}}
{"type":"assertion_superseded","data":{"id":"as-1","superseded_by":"as-2"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	all, err := assertionsFromRun(runDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("expected two recorded assertions, got %d", len(all))
	}
	effective := effectiveAssertions(all)
	if len(effective) != 1 {
		t.Fatalf("expected one effective assertion, got %+v", effective)
	}
	if effective[0].ID != "as-2" || effective[0].Confidence != "observed" {
		t.Fatalf("unexpected effective assertion: %+v", effective[0])
	}
	goal := Goal{RequiresEvidence: []Requirement{{
		Subject: "patch", Predicate: "applies_cleanly", MinConfidence: "observed",
		AnySourceType: []string{"process_exit"},
	}}}
	if len(missingEvidenceWithConfig(runDir, goal, Config{})) != 0 {
		t.Fatal("superseding observed assertion should satisfy evidence requirement")
	}
}

func TestAutoRepairSettings(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Execution: map[string]interface{}{
				"auto_repair": map[string]interface{}{"enabled": true, "max_attempts": 3},
			},
		}},
	}
	enabled, max := autoRepairSettings(cfg, false, 0)
	if !enabled || max != 3 {
		t.Fatalf("policy auto_repair should enable with max 3, got enabled=%v max=%d", enabled, max)
	}
	enabled, max = autoRepairSettings(cfg, false, 5)
	if !enabled || max != 5 {
		t.Fatalf("policy enabled with flag max override, got enabled=%v max=%d", enabled, max)
	}
	enabled, max = autoRepairSettings(Config{}, true, 2)
	if !enabled || max != 2 {
		t.Fatalf("flag auto-repair should enable with max 2, got enabled=%v max=%d", enabled, max)
	}
}

func TestAutoRepairRetriesBeforeStopping(t *testing.T) {
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
	if err := os.MkdirAll(".rp/runs", 0755); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
capabilities:
  fail_cmd:
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
    command:
      cwd: "."
      argv: [/bin/sh, -c, "exit 7"]
    always_record_result: true
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
	if err := os.WriteFile(".rp/planner.yaml", planner, 0644); err != nil {
		t.Fatal(err)
	}
	err = run([]string{"achieve", "smoke", "--auto-repair", "--max-attempts", "2"})
	if err == nil {
		t.Fatal("expected achieve to fail after retries exhausted")
	}
	runDir, err := latestRunDir()
	if err != nil {
		t.Fatal(err)
	}
	events, err := readEvents(runDir)
	if err != nil {
		t.Fatal(err)
	}
	repairAttempts := 0
	failCmdRuns := 0
	for _, ev := range events {
		if ev.Type == "auto_repair_attempted" {
			repairAttempts++
		}
		if ev.Type == "action_started" && ev.Data["capability"] == "fail_cmd" {
			failCmdRuns++
		}
	}
	if repairAttempts != 1 {
		t.Fatalf("expected one auto_repair_attempted event, got %d", repairAttempts)
	}
	if failCmdRuns != 2 {
		t.Fatalf("expected fail_cmd to run twice with max-attempts 2, got %d runs", failCmdRuns)
	}
}

func TestReplayPrintsNarrative(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".rp", "runs", "run-test")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	events := []byte(`{"type":"run_started","time":"2026-06-28T12:00:00Z","data":{"goal":"bugfix_patch"}}
{"type":"assertion_recorded","time":"2026-06-28T12:00:01Z","action_id":"step-01","data":{"subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_source":"process_exit"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	summary := []byte(`{"goal":"bugfix_patch","satisfied":true,"reason":"goal evidence requirements satisfied"}
`)
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), summary, 0644); err != nil {
		t.Fatal(err)
	}
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
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cmdReplay([]string{"run-test"})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Replay run-test") || !strings.Contains(out, "patch.applies_cleanly") {
		t.Fatalf("unexpected replay output: %q", out)
	}
}

func TestEvidenceReportShowsRequirementStatus(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".rp", "runs", "run-evidence")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-1","evidence_source":"process_exit","action_id":"act-1"}}
{"type":"assertion_recorded","data":{"id":"as-2","subject":"patched_repo","predicate":"tests_pass","confidence":"observed","evidence_id":"ev-2","evidence_source":"process_exit","action_id":"act-2"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	summary := []byte(`{"goal":"bugfix_patch","satisfied":true,"reason":"goal evidence requirements satisfied"}
`)
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), summary, 0644); err != nil {
		t.Fatal(err)
	}
	planner := []byte(`version: rp.dev/v0.1
goals:
  bugfix_patch:
    requires_evidence:
      - subject: patch
        predicate: applies_cleanly
        min_confidence: observed
      - subject: patched_repo
        predicate: tests_pass
        min_confidence: observed
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.MkdirAll(filepath.Join(dir, ".rp"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cmdEvidence([]string{"bugfix_patch"})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Satisfied: true") || !strings.Contains(out, "[ok] patch.applies_cleanly") || !strings.Contains(out, "[ok] patched_repo.tests_pass") {
		t.Fatalf("unexpected evidence output: %q", out)
	}
}

func TestExecutedCapabilitiesFromEvents(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"action_started","action_id":"step-01-observe","data":{"capability":"observe_git_status"}}
{"type":"action_completed","action_id":"step-01-observe","data":{"exit_code":0}}
{"type":"action_started","action_id":"step-02-fail","data":{"capability":"run_tests"}}
{"type":"action_failed","action_id":"step-02-fail","data":{"exit_code":1}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	executed := executedCapabilitiesFromEvents(runDir)
	if !executed["observe_git_status"] {
		t.Fatal("observe_git_status should count as executed")
	}
	if executed["run_tests"] {
		t.Fatal("failed capability should not count as executed")
	}
}

func TestRecordGoalAttestation(t *testing.T) {
	dir := t.TempDir()
	ctx, err := newRun(dir, "cfg-hash", "pol-hash")
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Events.Close()
	goal := Goal{RequiresEvidence: []Requirement{
		{Subject: "patch", Predicate: "applies_cleanly", MinConfidence: "observed"},
	}}
	events := []byte(`{"type":"assertion_recorded","data":{"id":"as-1","subject":"patch","predicate":"applies_cleanly","confidence":"observed","evidence_id":"ev-1","evidence_source":"process_exit","action_id":"act-1"}}
`)
	if err := os.WriteFile(filepath.Join(ctx.RunDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Events.Close()
	f, err := os.OpenFile(filepath.Join(ctx.RunDir, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	ctx.Events = f
	recordGoalAttestation(ctx, Config{
		Resources: map[string]Resource{
			"bug_report": {Realizations: []Realization{{URI: "file://bug.md"}}},
		},
	}, goal)
	all, err := readEvents(ctx.RunDir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ev := range all {
		if ev.Type == "attestation_recorded" && ev.Data["id"] == "att-goal-"+ctx.RunID {
			found = true
		}
	}
	if !found {
		t.Fatal("expected goal attestation event")
	}
}

func TestCapabilityDestructiveWriteApproval(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"filesystem": map[string]interface{}{
					"write":             "allowed",
					"destructive_write": "approval_required",
				},
			},
		}},
	}
	capability := Capability{
		Idempotence: "non_idempotent",
		Effects:     EffectSpec{Filesystem: map[string][]string{"writes": {"."}}},
	}
	if got := capabilityApprovalPermission(cfg, capability); got != "filesystem.destructive_write" {
		t.Fatalf("expected destructive_write approval, got %q", got)
	}
}

func TestValidatePlanMaxCostRejectsExpensivePlan(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			MaxCost: map[string]interface{}{"time": "1m"},
		}},
		Capabilities: map[string]Capability{
			"cheap":     {Cost: map[string]interface{}{"time": "cheap"}},
			"expensive": {Cost: map[string]interface{}{"time": "expensive"}},
		},
	}
	goal := Goal{Constraints: map[string]interface{}{
		"max_cost": map[string]interface{}{"time": "1m"},
	}}
	plan := []PlanStep{{Capability: "cheap"}, {Capability: "expensive"}}
	if err := validatePlanMaxCost(cfg, goal, plan); err == nil {
		t.Fatal("expected plan to exceed 1m budget")
	}
}

func TestFilterPlanByConstraintsRemovesForbiddenNetwork(t *testing.T) {
	goal := Goal{Constraints: map[string]interface{}{
		"permissions": map[string]interface{}{"network": "forbidden"},
	}}
	cfg := Config{
		Capabilities: map[string]Capability{
			"ok":      {Effects: EffectSpec{External: "local_process"}},
			"network": {Effects: EffectSpec{Network: map[string]interface{}{"access": true}}},
		},
	}
	steps := filterPlanByConstraints(cfg, goal, []PlanStep{{Capability: "ok"}, {Capability: "network"}})
	if len(steps) != 1 || steps[0].Capability != "ok" {
		t.Fatalf("expected only ok step, got %+v", steps)
	}
}

func TestInputHashesForGoalHashesFileResources(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bug.md")
	if err := os.WriteFile(path, []byte("bug"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		Resources: map[string]Resource{
			"bug_report": {Realizations: []Realization{{URI: "file://" + path}}},
		},
	}
	goal := Goal{Given: map[string]string{"bug_report": "bug_report"}}
	hashes := inputHashesForGoal(dir, cfg, goal)
	if len(hashes) != 1 || hashes["bug_report"] == "" {
		t.Fatalf("expected bug_report hash, got %+v", hashes)
	}
}

func TestCheckCapabilityPolicyRejectsForbiddenExternalSideEffect(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"external_side_effects": map[string]interface{}{
					"create_pull_request": "forbidden",
				},
			},
		}},
	}
	capability := Capability{Effects: EffectSpec{
		ExternalSideEffects: map[string]interface{}{"create_pull_request": true},
	}}
	if err := checkCapabilityPolicy(cfg, capability); err == nil {
		t.Fatal("expected forbidden external side effect error")
	}
}

func TestFilterPlanByPolicyRemovesForbiddenExternalSideEffect(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"external_side_effects": map[string]interface{}{"deploy": "forbidden"},
			},
		}},
		Capabilities: map[string]Capability{
			"ok":     {Effects: EffectSpec{External: "local_process"}},
			"deploy": {Effects: EffectSpec{ExternalSideEffects: map[string]interface{}{"deploy": true}}},
		},
	}
	steps := filterPlanByPolicy(cfg, []PlanStep{{Capability: "ok"}, {Capability: "deploy"}})
	if len(steps) != 1 || steps[0].Capability != "ok" {
		t.Fatalf("expected only ok step, got %+v", steps)
	}
}

func TestCheckCapabilityPolicyRejectsCredentialRefInput(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Permissions: map[string]interface{}{
				"credentials": map[string]interface{}{"use": "forbidden"},
			},
		}},
	}
	capability := Capability{
		Inputs: map[string]InputSpec{"token": {Type: "CredentialRef"}},
	}
	if err := checkCapabilityPolicy(cfg, capability); err == nil {
		t.Fatal("expected forbidden credential use for CredentialRef input")
	}
}

func TestFailureStopDataHonorsOnFailurePolicy(t *testing.T) {
	cfg := Config{
		Defaults: map[string]string{"policy": "local_safe"},
		Policies: map[string]Policy{"local_safe": {
			Execution: map[string]interface{}{"on_failure": "stop"},
		}},
	}
	data, summary := failureStopData(cfg, "run-123", "boom")
	if _, ok := data["suggest_replan"]; ok {
		t.Fatal("stop mode should not suggest replan")
	}
	if strings.Contains(summary, "replan") {
		t.Fatalf("stop mode summary should not mention replan: %q", summary)
	}
	cfg.Policies["local_safe"] = Policy{
		Execution: map[string]interface{}{"on_failure": "stop_and_suggest"},
	}
	data, summary = failureStopData(cfg, "run-123", "boom")
	if data["suggest_replan"] != true {
		t.Fatal("stop_and_suggest should suggest replan")
	}
	if !strings.Contains(summary, "rp replan run-123") {
		t.Fatalf("expected replan hint in summary, got %q", summary)
	}
}

func TestMissingProduceRequiresRealization(t *testing.T) {
	runDir := t.TempDir()
	events := []byte(`{"type":"resource_realization_recorded","data":{"resource":"patch","artifact":"artifacts/proposed.patch","media_type":"text/plain","kind":"file"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(runDir, "artifacts"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(runDir, "artifacts/proposed.patch"), []byte("diff"), 0644); err != nil {
		t.Fatal(err)
	}
	goal := Goal{Produce: map[string]OutputSpec{
		"patch": {Type: "Patch", RequiredRealization: Realization{Kind: "file", MediaType: "text/x-diff"}},
	}}
	gaps := missingProduce(runDir, goal)
	if len(gaps) != 1 || !strings.Contains(gaps[0], "media_type") {
		t.Fatalf("expected media_type mismatch, got %+v", gaps)
	}
	events = []byte(`{"type":"resource_realization_recorded","data":{"resource":"patch","artifact":"artifacts/proposed.patch","media_type":"text/x-diff","kind":"file"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	if len(missingProduce(runDir, goal)) != 0 {
		t.Fatal("matching realization should satisfy produce requirement")
	}
}

func TestDryRunPrintsEffectSummary(t *testing.T) {
	dir := t.TempDir()
	planner := []byte(`version: rp.dev/v0.1
capabilities:
  ok:
    purpose: observe
    kind: command
    outputs:
      result:
        type: CommandResult
    command:
      cwd: "."
      argv: [echo, ok]
    effects:
      external: local_process
      filesystem:
        writes: []
goals:
  smoke:
    requires_evidence: []
policies:
  local_safe:
    permissions: {}
defaults:
  policy: local_safe
`)
	if err := os.MkdirAll(filepath.Join(dir, ".rp"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".rp", "planner.yaml"), planner, 0644); err != nil {
		t.Fatal(err)
	}
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
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cmdAchieve([]string{"smoke", "--dry-run"})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "dry-run") || !strings.Contains(out, "Effect summary:") {
		t.Fatalf("expected dry-run effect summary, got %q", out)
	}
}

func TestAuditPrintsSummaryHeader(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, ".rp", "runs", "run-audit")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatal(err)
	}
	events := []byte(`{"type":"run_started","time":"2026-01-01T00:00:00Z","data":{"goal":"bugfix_patch"}}
{"type":"attestation_recorded","time":"2026-01-01T00:00:01Z","data":{"id":"att-goal-run-audit"}}
`)
	if err := os.WriteFile(filepath.Join(runDir, "events.jsonl"), events, 0644); err != nil {
		t.Fatal(err)
	}
	summary := []byte(`{"goal":"bugfix_patch","satisfied":true,"reason":"ok","config_hash":"abc","policy_hash":"def"}
`)
	if err := os.WriteFile(filepath.Join(runDir, "summary.json"), summary, 0644); err != nil {
		t.Fatal(err)
	}
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
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = cmdAudit([]string{"run-audit"})
	w.Close()
	os.Stdout = oldStdout
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Audit run-audit") || !strings.Contains(out, "Attestation: att-goal-run-audit") || !strings.Contains(out, "run_started: 1") {
		t.Fatalf("unexpected audit output: %q", out)
	}
}
