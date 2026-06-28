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
				Inputs: map[string]InputSpec{"repo": {Requires: []Requirement{{Predicate: "clean_worktree"}}}},
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

func TestMergePoliciesMostRestrictive(t *testing.T) {
	project := Policy{Permissions: map[string]interface{}{
		"network":      map[string]interface{}{"access": "allowed"},
		"filesystem":   map[string]interface{}{"write": "allowed"},
		"credentials":  map[string]interface{}{"use": "forbidden"},
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
	if err := exec.Command("git", "commit", "-m", "initial").Run(); err != nil {
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
