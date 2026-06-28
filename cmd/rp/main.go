package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const version = "rp.dev/v0.1"

var confidenceRank = map[string]int{
	"unsupported":              0,
	"claimed":                  1,
	"observed":                 2,
	"attested":                 3,
	"reproduced":               4,
	"independently_reproduced": 5,
}

type Config struct {
	Version      string                 `yaml:"version" json:"version"`
	Imports      []string               `yaml:"imports,omitempty" json:"imports,omitempty"`
	Resources    map[string]Resource    `yaml:"resources,omitempty" json:"resources,omitempty"`
	Capabilities map[string]Capability  `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Policies     map[string]Policy      `yaml:"policies,omitempty" json:"policies,omitempty"`
	Goals        map[string]Goal        `yaml:"goals,omitempty" json:"goals,omitempty"`
	Defaults     map[string]string      `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	X            map[string]interface{} `yaml:",inline" json:"-"`
}

type Resource struct {
	Type         string        `yaml:"type" json:"type"`
	Realizations []Realization `yaml:"realizations,omitempty" json:"realizations,omitempty"`
}

type Realization struct {
	ID        string                 `yaml:"id" json:"id"`
	Kind      string                 `yaml:"kind" json:"kind"`
	URI       string                 `yaml:"uri" json:"uri"`
	MediaType string                 `yaml:"media_type,omitempty" json:"media_type,omitempty"`
	Hash      string                 `yaml:"hash,omitempty" json:"hash,omitempty"`
	Metadata  map[string]interface{} `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

type Capability struct {
	Purpose            string                 `yaml:"purpose" json:"purpose"`
	Kind               string                 `yaml:"kind" json:"kind"`
	Inputs             map[string]InputSpec   `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs            map[string]OutputSpec  `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Preconditions      []Requirement          `yaml:"preconditions,omitempty" json:"preconditions,omitempty"`
	Command            CommandSpec            `yaml:"command,omitempty" json:"command,omitempty"`
	Approval           map[string]interface{} `yaml:"approval,omitempty" json:"approval,omitempty"`
	AlwaysRecordResult bool                   `yaml:"always_record_result,omitempty" json:"always_record_result,omitempty"`
	Effects            EffectSpec             `yaml:"effects,omitempty" json:"effects,omitempty"`
	Nondeterminism     []string               `yaml:"nondeterminism,omitempty" json:"nondeterminism,omitempty"`
	Idempotence        string                 `yaml:"idempotence,omitempty" json:"idempotence,omitempty"`
	Cost               map[string]interface{} `yaml:"cost,omitempty" json:"cost,omitempty"`
}

type InputSpec struct {
	Type        string        `yaml:"type" json:"type"`
	Realization Realization   `yaml:"realization,omitempty" json:"realization,omitempty"`
	Requires    []Requirement `yaml:"requires,omitempty" json:"requires,omitempty"`
}

type Requirement struct {
	Subject       string   `yaml:"subject,omitempty" json:"subject,omitempty"`
	Predicate     string   `yaml:"predicate" json:"predicate"`
	MinConfidence string   `yaml:"min_confidence" json:"min_confidence"`
	AnySourceType []string `yaml:"any_source_type,omitempty" json:"any_source_type,omitempty"`
}

type OutputSpec struct {
	Type                string          `yaml:"type" json:"type"`
	RequiredRealization Realization     `yaml:"required_realization,omitempty" json:"required_realization,omitempty"`
	Realization         Realization     `yaml:"realization,omitempty" json:"realization,omitempty"`
	Assertions          []AssertionSpec `yaml:"assertions,omitempty" json:"assertions,omitempty"`
}

type AssertionSpec struct {
	Subject        string                 `yaml:"subject" json:"subject"`
	Predicate      string                 `yaml:"predicate" json:"predicate"`
	Object         string                 `yaml:"object,omitempty" json:"object,omitempty"`
	Confidence     string                 `yaml:"confidence" json:"confidence"`
	When           map[string]interface{} `yaml:"when,omitempty" json:"when,omitempty"`
	EvidenceSource string                 `yaml:"evidence_source,omitempty" json:"evidence_source,omitempty"`
}

type CommandSpec struct {
	CWD    string     `yaml:"cwd,omitempty" json:"cwd,omitempty"`
	Argv   []string   `yaml:"argv,omitempty" json:"argv,omitempty"`
	Stdout StreamSpec `yaml:"stdout,omitempty" json:"stdout,omitempty"`
	Stderr StreamSpec `yaml:"stderr,omitempty" json:"stderr,omitempty"`
}

type StreamSpec struct {
	SaveAs         SaveAsSpec `yaml:"save_as,omitempty" json:"save_as,omitempty"`
	SaveAsArtifact string     `yaml:"save_as_artifact,omitempty" json:"save_as_artifact,omitempty"`
	MediaType      string     `yaml:"media_type,omitempty" json:"media_type,omitempty"`
}

type SaveAsSpec struct {
	Resource     string `yaml:"resource,omitempty" json:"resource,omitempty"`
	ArtifactPath string `yaml:"artifact_path,omitempty" json:"artifact_path,omitempty"`
	MediaType    string `yaml:"media_type,omitempty" json:"media_type,omitempty"`
}

type EffectSpec struct {
	External   string                 `yaml:"external,omitempty" json:"external,omitempty"`
	Planner    string                 `yaml:"planner,omitempty" json:"planner,omitempty"`
	Filesystem map[string][]string    `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`
	Network    map[string]interface{} `yaml:"network,omitempty" json:"network,omitempty"`
}

type Policy struct {
	Description string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Permissions map[string]interface{} `yaml:"permissions,omitempty" json:"permissions,omitempty"`
	Environment struct {
		Inherit bool     `yaml:"inherit" json:"inherit"`
		Allow   []string `yaml:"allow,omitempty" json:"allow,omitempty"`
	} `yaml:"environment,omitempty" json:"environment,omitempty"`
	Evidence  map[string]interface{} `yaml:"evidence,omitempty" json:"evidence,omitempty"`
	Hashing   map[string]interface{} `yaml:"hashing,omitempty" json:"hashing,omitempty"`
	Execution map[string]interface{} `yaml:"execution,omitempty" json:"execution,omitempty"`
	MaxCost   map[string]interface{} `yaml:"max_cost,omitempty" json:"max_cost,omitempty"`
}

type Goal struct {
	Description      string                 `yaml:"description,omitempty" json:"description,omitempty"`
	Given            map[string]string      `yaml:"given,omitempty" json:"given,omitempty"`
	Produce          map[string]OutputSpec  `yaml:"produce,omitempty" json:"produce,omitempty"`
	RequiresEvidence []Requirement          `yaml:"requires_evidence,omitempty" json:"requires_evidence,omitempty"`
	Constraints      map[string]interface{} `yaml:"constraints,omitempty" json:"constraints,omitempty"`
}

type PlanStep struct {
	ID         string            `json:"id"`
	Capability string            `json:"capability"`
	Reason     string            `json:"reason"`
	Inputs     map[string]string `json:"inputs"`
}

type Event struct {
	Type     string                 `json:"type"`
	Time     string                 `json:"time"`
	RunID    string                 `json:"run_id,omitempty"`
	ActionID string                 `json:"action_id,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
}

type RunContext struct {
	Root       string
	RPDir      string
	RunID      string
	RunDir     string
	Artifacts  string
	ConfigHash string
	PolicyHash string
	Events     *os.File
}

type AssertionRecord struct {
	ID             string `json:"id"`
	Subject        string `json:"subject"`
	Predicate      string `json:"predicate"`
	Object         string `json:"object,omitempty"`
	Confidence     string `json:"confidence"`
	EvidenceID     string `json:"evidence_id"`
	EvidenceSource string `json:"evidence_source"`
	ActionID       string `json:"action_id"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "rp:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:])
	case "capability":
		return cmdCapability(args[1:])
	case "goal":
		return cmdGoal(args[1:])
	case "policy":
		return cmdPolicy(args[1:])
	case "add":
		return cmdAdd(args[1:])
	case "resources":
		return cmdResources(args[1:])
	case "resource":
		return cmdResource(args[1:])
	case "plan":
		return cmdPlan(args[1:])
	case "achieve":
		return cmdAchieve(args[1:])
	case "evidence":
		return cmdEvidence(args[1:])
	case "why":
		return cmdWhy(args[1:])
	case "audit", "replay":
		return cmdAudit(args[1:])
	case "replan":
		return cmdReplan(args[1:])
	case "version":
		fmt.Println("rp", version)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Println(`rp v0.1

Usage:
  rp init
  rp capability init command NAME
  rp goal init NAME
  rp policy init NAME
  rp add resource NAME --type TYPE (--uri URI | --file PATH) [--media-type TYPE]
  rp resources
  rp resource NAME
  rp plan GOAL [--explain] [--format text|json|dot|mermaid]
  rp achieve GOAL [--dry-run] [--step] [--yes]
  rp evidence GOAL
  rp why SUBJECT.PREDICATE
  rp audit RUN_ID
  rp replay RUN_ID
  rp replan RUN_ID`)
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	dirs := []string{".rp", ".rp/capabilities", ".rp/policies", ".rp/goals", ".rp/runs", ".rp/cache"}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	writeIfMissing(".rp/planner.yaml", `version: rp.dev/v0.1
imports: []
resources: {}
defaults: {}
`)
	writeIfMissing(".rp/.gitignore", "runs/\ncache/\n")
	fmt.Println("initialized .rp/")
	return nil
}

func cmdCapability(args []string) error {
	if len(args) != 3 || args[0] != "init" || args[1] != "command" {
		return errors.New("usage: rp capability init command NAME")
	}
	path := filepath.Join(".rp", "capabilities", args[2]+".yaml")
	body := fmt.Sprintf(`version: rp.dev/v0.1
capabilities:
  %s:
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
        - echo
        - ok
    effects:
      external: local_process
      filesystem:
        writes: []
    idempotence: idempotent
`, args[2])
	if err := writeIfMissing(path, body); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func cmdGoal(args []string) error {
	if len(args) != 2 || args[0] != "init" {
		return errors.New("usage: rp goal init NAME")
	}
	path := filepath.Join(".rp", "goals", args[1]+".yaml")
	body := fmt.Sprintf(`version: rp.dev/v0.1
goals:
  %s:
    description: Fill in this goal.
    given: {}
    produce: {}
    requires_evidence: []
    constraints: {}
`, args[1])
	if err := writeIfMissing(path, body); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func cmdPolicy(args []string) error {
	if len(args) != 2 || args[0] != "init" {
		return errors.New("usage: rp policy init NAME")
	}
	path := filepath.Join(".rp", "policies", args[1]+".yaml")
	body := fmt.Sprintf(`version: rp.dev/v0.1
policies:
  %s:
    description: Local conservative policy.
    permissions:
      filesystem:
        read: allowed
        write: approval_required
      process:
        execute: allowed
      network:
        access: forbidden
      credentials:
        use: forbidden
    environment:
      inherit: false
      allow:
        - PATH
        - HOME
`, args[1])
	if err := writeIfMissing(path, body); err != nil {
		return err
	}
	fmt.Println(path)
	return nil
}

func cmdAdd(args []string) error {
	if len(args) < 2 || args[0] != "resource" {
		return errors.New("usage: rp add resource NAME --type TYPE (--uri URI | --file PATH) [--media-type TYPE]")
	}
	fs := flag.NewFlagSet("add resource", flag.ContinueOnError)
	typ := fs.String("type", "", "resource type")
	uri := fs.String("uri", "", "resource uri")
	file := fs.String("file", "", "file path")
	media := fs.String("media-type", "", "media type")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	if *typ == "" {
		return errors.New("--type is required")
	}
	if *uri == "" && *file == "" {
		return errors.New("--uri or --file is required")
	}
	actualURI := *uri
	kind := "local_path"
	if *file != "" {
		actualURI = "file://" + *file
		kind = "file"
	}
	path := ".rp/planner.yaml"
	cfg, err := loadSingleConfig(path)
	if err != nil {
		return err
	}
	if cfg.Resources == nil {
		cfg.Resources = map[string]Resource{}
	}
	name := args[1]
	cfg.Resources[name] = Resource{Type: *typ, Realizations: []Realization{{
		ID: name + ".local", Kind: kind, URI: actualURI, MediaType: *media,
	}}}
	return writeYAML(path, cfg)
}

func cmdResources(args []string) error {
	root, cfg, _, err := loadProject()
	if err != nil {
		return err
	}
	_ = root
	names := sortedKeys(cfg.Resources)
	for _, name := range names {
		fmt.Printf("%s\t%s\t%s\n", name, cfg.Resources[name].Type, realizationURI(cfg.Resources[name]))
	}
	return nil
}

func cmdResource(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp resource NAME")
	}
	_, cfg, _, err := loadProject()
	if err != nil {
		return err
	}
	res, ok := cfg.Resources[args[0]]
	if !ok {
		return fmt.Errorf("resource %q not found", args[0])
	}
	return printJSON(res)
}

func cmdPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)
	explain := fs.Bool("explain", false, "explain plan")
	format := fs.String("format", "text", "text|json|dot|mermaid")
	speculative := fs.Bool("speculative", false, "accepted for v0.1 compatibility")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"format": true}, map[string]bool{"explain": true, "speculative": true})); err != nil {
		return err
	}
	_ = speculative
	if fs.NArg() != 1 {
		return errors.New("usage: rp plan GOAL")
	}
	root, cfg, hash, err := loadProject()
	if err != nil {
		return err
	}
	plan, err := buildPlan(cfg, fs.Arg(0))
	if err != nil {
		return err
	}
	switch *format {
	case "json":
		return printJSON(map[string]interface{}{"goal": fs.Arg(0), "config_hash": hash, "steps": plan})
	case "dot":
		fmt.Print(renderDOT(plan))
	case "mermaid":
		fmt.Print(renderMermaid(plan))
	case "text":
		fmt.Printf("Goal: %s\nConfig: %s\nRoot: %s\n\n", fs.Arg(0), hash[:12], root)
		for i, step := range plan {
			fmt.Printf("%d. %s\n   capability: %s\n   reason: %s\n", i+1, step.ID, step.Capability, step.Reason)
			if *explain {
				fmt.Printf("   inputs: %v\n", step.Inputs)
			}
		}
	default:
		return fmt.Errorf("unsupported format %q", *format)
	}
	return nil
}

func cmdAchieve(args []string) error {
	fs := flag.NewFlagSet("achieve", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show and record no execution")
	stepMode := fs.Bool("step", false, "confirm every step")
	yes := fs.Bool("yes", false, "approve required filesystem writes")
	autoRepair := fs.Bool("auto-repair", false, "accepted for v0.1 compatibility")
	maxAttempts := fs.Int("max-attempts", 1, "accepted for v0.1 compatibility")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"max-attempts": true}, map[string]bool{"dry-run": true, "step": true, "yes": true, "auto-repair": true})); err != nil {
		return err
	}
	_, _ = autoRepair, maxAttempts
	if fs.NArg() != 1 {
		return errors.New("usage: rp achieve GOAL")
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	goalName := fs.Arg(0)
	plan, err := buildPlan(cfg, goalName)
	if err != nil {
		return err
	}
	if *dryRun {
		for i, step := range plan {
			fmt.Printf("%d. %s (%s)\n", i+1, step.Capability, step.Reason)
		}
		return nil
	}
	ctx, err := newRun(root, configHash, hashPolicy(cfg))
	if err != nil {
		return err
	}
	defer ctx.Events.Close()
	appendEvent(ctx, "run_started", "", map[string]interface{}{"goal": goalName, "config_hash": configHash})
	appendEvent(ctx, "plan_proposed", "", map[string]interface{}{"steps": plan})
	resources := map[string]string{}
	for k := range cfg.Resources {
		resources[k] = k
	}
	for _, step := range plan {
		if *stepMode && !ask("execute "+step.Capability+"?") {
			appendEvent(ctx, "run_stopped", "", map[string]interface{}{"reason": "step denied"})
			return errors.New("stopped by user")
		}
		capability := cfg.Capabilities[step.Capability]
		if err := checkCapabilityPolicy(cfg, capability); err != nil {
			appendEvent(ctx, "run_stopped", step.ID, map[string]interface{}{"reason": err.Error()})
			writeSummary(ctx, goalName, false, err.Error())
			return err
		}
		if needsWriteApproval(cfg, capability) && !*yes {
			appendEvent(ctx, "approval_requested", step.ID, map[string]interface{}{"permission": "filesystem.write"})
			if !ask("approve filesystem write for " + step.Capability + "?") {
				appendEvent(ctx, "approval_denied", step.ID, nil)
				return errors.New("approval denied")
			}
			appendEvent(ctx, "approval_granted", step.ID, nil)
		}
		if err := executeStep(ctx, cfg, step, resources); err != nil {
			appendEvent(ctx, "run_stopped", "", map[string]interface{}{"reason": err.Error()})
			writeSummary(ctx, goalName, false, err.Error())
			return err
		}
	}
	satisfied := goalSatisfied(ctx.RunDir, cfg.Goals[goalName])
	appendEvent(ctx, "goal_satisfied", "", map[string]interface{}{"satisfied": satisfied})
	reason := "goal evidence requirements satisfied"
	if !satisfied {
		reason = "goal evidence requirements not fully satisfied"
	}
	if err := writeSummary(ctx, goalName, satisfied, reason); err != nil {
		return err
	}
	fmt.Printf("run %s %s\n", ctx.RunID, reason)
	fmt.Println(ctx.RunDir)
	return nil
}

func cmdEvidence(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp evidence GOAL")
	}
	runDir, err := latestRunDir()
	if err != nil {
		return err
	}
	events, err := readEvents(runDir)
	if err != nil {
		return err
	}
	for _, ev := range events {
		if ev.Type == "assertion_recorded" || ev.Type == "observation_recorded" || ev.Type == "evidence_recorded" {
			b, _ := json.MarshalIndent(ev, "", "  ")
			fmt.Println(string(b))
		}
	}
	return nil
}

func cmdWhy(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp why SUBJECT.PREDICATE")
	}
	subject, predicate, ok := strings.Cut(args[0], ".")
	if !ok {
		return errors.New("why expects SUBJECT.PREDICATE")
	}
	runDir, err := latestRunDir()
	if err != nil {
		return err
	}
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		return err
	}
	best := AssertionRecord{Confidence: "unsupported"}
	for _, a := range assertions {
		if a.Subject == subject && a.Predicate == predicate && confidenceAtLeast(a.Confidence, best.Confidence) {
			best = a
		}
	}
	if best.ID == "" {
		fmt.Printf("%s.%s is unsupported in latest run %s\n", subject, predicate, filepath.Base(runDir))
		return nil
	}
	fmt.Printf("%s.%s is %s\nsupported by assertion %s from action %s and evidence %s (%s)\n", subject, predicate, best.Confidence, best.ID, best.ActionID, best.EvidenceID, best.EvidenceSource)
	return nil
}

func cmdAudit(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp audit RUN_ID")
	}
	runDir := filepath.Join(".rp", "runs", args[0])
	events, err := readEvents(runDir)
	if err != nil {
		return err
	}
	for _, ev := range events {
		fmt.Printf("%s\t%s\t%s\n", ev.Time, ev.Type, ev.ActionID)
	}
	return nil
}

func cmdReplan(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp replan RUN_ID")
	}
	summaryPath := filepath.Join(".rp", "runs", args[0], "summary.json")
	b, err := os.ReadFile(summaryPath)
	if err != nil {
		return err
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(b, &summary); err != nil {
		return err
	}
	goal, _ := summary["goal"].(string)
	if goal == "" {
		return errors.New("run summary has no goal")
	}
	return cmdPlan([]string{"--explain", goal})
}

func loadProject() (string, Config, string, error) {
	root, err := findProjectRoot()
	if err != nil {
		return "", Config{}, "", err
	}
	cfg, err := loadConfig(filepath.Join(root, ".rp", "planner.yaml"))
	if err != nil {
		return "", Config{}, "", err
	}
	hash, err := canonicalHash(cfg)
	if err != nil {
		return "", Config{}, "", err
	}
	return root, cfg, hash, nil
}

func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".rp", "planner.yaml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("no .rp/planner.yaml found")
		}
		dir = parent
	}
}

func loadConfig(path string) (Config, error) {
	base, err := loadSingleConfig(path)
	if err != nil {
		return Config{}, err
	}
	root := filepath.Dir(path)
	merged := base
	merged.Capabilities = map[string]Capability{}
	merged.Policies = map[string]Policy{}
	merged.Goals = map[string]Goal{}
	for _, imp := range base.Imports {
		child, err := loadSingleConfig(filepath.Join(root, imp))
		if err != nil {
			return Config{}, err
		}
		mergeConfig(&merged, child)
	}
	mergeConfig(&merged, base)
	return merged, nil
}

func loadSingleConfig(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version != "" && cfg.Version != version {
		return Config{}, fmt.Errorf("%s: unsupported version %q", path, cfg.Version)
	}
	return cfg, nil
}

func mergeConfig(dst *Config, src Config) {
	if dst.Version == "" {
		dst.Version = src.Version
	}
	if dst.Resources == nil {
		dst.Resources = map[string]Resource{}
	}
	for k, v := range src.Resources {
		dst.Resources[k] = v
	}
	if dst.Capabilities == nil {
		dst.Capabilities = map[string]Capability{}
	}
	for k, v := range src.Capabilities {
		dst.Capabilities[k] = v
	}
	if dst.Policies == nil {
		dst.Policies = map[string]Policy{}
	}
	for k, v := range src.Policies {
		dst.Policies[k] = v
	}
	if dst.Goals == nil {
		dst.Goals = map[string]Goal{}
	}
	for k, v := range src.Goals {
		dst.Goals[k] = v
	}
	if dst.Defaults == nil {
		dst.Defaults = map[string]string{}
	}
	for k, v := range src.Defaults {
		dst.Defaults[k] = v
	}
}

func buildPlan(cfg Config, goalName string) ([]PlanStep, error) {
	goal, ok := cfg.Goals[goalName]
	if !ok {
		return nil, fmt.Errorf("goal %q not found", goalName)
	}
	var steps []PlanStep
	seen := map[string]bool{}
	add := func(capName, reason string, inputs map[string]string) {
		if seen[capName] {
			return
		}
		seen[capName] = true
		steps = append(steps, PlanStep{ID: fmt.Sprintf("step-%02d", len(steps)+1), Capability: capName, Reason: reason, Inputs: inputs})
	}
	repoName := goal.Given["repo"]
	if repoName == "" {
		repoName = firstResourceOfType(cfg, "GitRepo")
	}
	bugName := goal.Given["bug_report"]
	for capName, cap := range cfg.Capabilities {
		for _, in := range cap.Inputs {
			for _, req := range in.Requires {
				if req.Predicate == "clean_worktree" {
					if obs := findAssertionCapability(cfg, "repo", "clean_worktree"); obs != "" {
						add(obs, "observe precondition clean_worktree", map[string]string{"repo": repoName})
					}
					_ = capName
				}
			}
		}
	}
	if _, wantsPatch := goal.Produce["patch"]; wantsPatch {
		if capName := findOutputCapability(cfg, "patch"); capName != "" {
			add(capName, "produce patch resource", map[string]string{"repo": repoName, "bug_report": bugName})
		}
	}
	for _, req := range goal.RequiresEvidence {
		switch {
		case req.Subject == "patch" && req.Predicate == "applies_cleanly":
			if capName := findAssertionCapability(cfg, "patch", "applies_cleanly"); capName != "" {
				add(capName, "observe patch.applies_cleanly", map[string]string{"repo": repoName, "patch": "patch"})
			}
		case req.Subject == "patched_repo" || req.Subject == "repo":
			if req.Subject == "patched_repo" {
				if capName := findOutputCapability(cfg, "patched_repo"); capName != "" {
					add(capName, "derive patched_repo", map[string]string{"repo": repoName, "patch": "patch"})
				}
			}
			if capName := findAssertionCapability(cfg, "repo", req.Predicate); capName != "" {
				subject := repoName
				if req.Subject == "patched_repo" {
					subject = "patched_repo"
				}
				add(capName, "observe "+req.Subject+"."+req.Predicate, map[string]string{"repo": subject})
			}
		default:
			if capName := findAssertionCapability(cfg, req.Subject, req.Predicate); capName != "" {
				add(capName, "observe "+req.Subject+"."+req.Predicate, map[string]string{req.Subject: req.Subject})
			}
		}
	}
	if len(steps) == 0 && len(cfg.Capabilities) == 1 {
		for name := range cfg.Capabilities {
			add(name, "single available capability", map[string]string{})
		}
	}
	if len(steps) == 0 {
		return nil, errors.New("no plan found")
	}
	return steps, nil
}

func executeStep(ctx RunContext, cfg Config, step PlanStep, resources map[string]string) error {
	capability := cfg.Capabilities[step.Capability]
	actionID := step.ID + "-" + safeName(step.Capability)
	appendEvent(ctx, "action_started", actionID, map[string]interface{}{"capability": step.Capability, "inputs": step.Inputs})
	cwd := substitute(capability.Command.CWD, ctx, cfg, step, resources)
	if cwd == "" {
		cwd = "."
	}
	if !filepath.IsAbs(cwd) {
		cwd = filepath.Join(ctx.Root, cwd)
	}
	argv := make([]string, len(capability.Command.Argv))
	for i, a := range capability.Command.Argv {
		argv[i] = substitute(a, ctx, cfg, step, resources)
	}
	if len(argv) == 0 {
		return fmt.Errorf("%s has no argv", step.Capability)
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = environmentFor(cfg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee := new(exec.ExitError); errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	obsID := "obs-" + actionID
	evidenceID := "ev-" + actionID
	saveStreams(ctx, capability, step, resources, stdout.Bytes(), stderr.Bytes())
	appendEvent(ctx, "observation_recorded", actionID, map[string]interface{}{
		"id": obsID, "source_type": "process_exit", "exit_code": exitCode,
		"stdout_sha256": sha(stdout.Bytes()), "stderr_sha256": sha(stderr.Bytes()),
	})
	appendEvent(ctx, "evidence_recorded", actionID, map[string]interface{}{
		"id": evidenceID, "source_type": "process_exit", "observation_id": obsID,
		"confidence_contribution": "observed",
	})
	for outName, out := range capability.Outputs {
		if capability.Command.Stdout.SaveAs.Resource == outName {
			resources[outName] = outName
			appendEvent(ctx, "resource_realization_recorded", actionID, map[string]interface{}{
				"resource": outName, "artifact": capability.Command.Stdout.SaveAs.ArtifactPath,
				"media_type": capability.Command.Stdout.SaveAs.MediaType,
			})
		}
		for _, as := range out.Assertions {
			if assertionMatches(as, exitCode, stdout.String()) {
				subj := resolveSubject(as.Subject, step)
				evidenceSource := as.EvidenceSource
				if evidenceSource == "" {
					evidenceSource = "process_exit"
				}
				rec := AssertionRecord{
					ID:      "as-" + actionID + "-" + safeName(subj+"-"+as.Predicate),
					Subject: subj, Predicate: as.Predicate, Object: as.Object,
					Confidence: as.Confidence, EvidenceID: evidenceID, EvidenceSource: evidenceSource, ActionID: actionID,
				}
				appendEvent(ctx, "assertion_recorded", actionID, structToMap(rec))
			}
		}
	}
	appendEvent(ctx, "action_completed", actionID, map[string]interface{}{"exit_code": exitCode})
	if exitCode != 0 {
		return fmt.Errorf("%s failed with exit code %d", step.Capability, exitCode)
	}
	return nil
}

func substitute(s string, ctx RunContext, cfg Config, step PlanStep, resources map[string]string) string {
	out := strings.ReplaceAll(s, "${run.id}", ctx.RunID)
	re := regexp.MustCompile(`\$\{inputs\.([a-zA-Z0-9_-]+)\.path\}`)
	out = re.ReplaceAllStringFunc(out, func(token string) string {
		m := re.FindStringSubmatch(token)
		name := step.Inputs[m[1]]
		if name == "" {
			name = m[1]
		}
		if name == "patch" {
			return filepath.Join(ctx.Artifacts, "proposed.patch")
		}
		if name == "patched_repo" {
			name = step.Inputs["repo"]
			if name == "patched_repo" {
				name = "repo"
			}
		}
		if res, ok := cfg.Resources[name]; ok {
			return pathFromURI(ctx.Root, realizationURI(res))
		}
		return name
	})
	return out
}

func saveStreams(ctx RunContext, cap Capability, step PlanStep, resources map[string]string, stdout, stderr []byte) {
	if p := cap.Command.Stdout.SaveAs.ArtifactPath; p != "" {
		writeArtifact(ctx, p, stdout)
	}
	if p := cap.Command.Stdout.SaveAsArtifact; p != "" {
		writeArtifact(ctx, p, stdout)
	}
	if p := cap.Command.Stderr.SaveAsArtifact; p != "" {
		writeArtifact(ctx, p, stderr)
	}
}

func writeArtifact(ctx RunContext, rel string, b []byte) {
	path := filepath.Join(ctx.RunDir, rel)
	if !strings.HasPrefix(rel, "artifacts/") {
		path = filepath.Join(ctx.Artifacts, rel)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, b, 0644)
	appendEvent(ctx, "artifact_recorded", "", map[string]interface{}{"path": path, "sha256": sha(b)})
}

func newRun(root, configHash, policyHash string) (RunContext, error) {
	runID := "run-" + time.Now().UTC().Format("20060102T150405.000000000Z")
	runDir := filepath.Join(root, ".rp", "runs", runID)
	artifacts := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artifacts, 0755); err != nil {
		return RunContext{}, err
	}
	f, err := os.OpenFile(filepath.Join(runDir, "events.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return RunContext{}, err
	}
	return RunContext{Root: root, RPDir: filepath.Join(root, ".rp"), RunID: runID, RunDir: runDir, Artifacts: artifacts, ConfigHash: configHash, PolicyHash: policyHash, Events: f}, nil
}

func appendEvent(ctx RunContext, typ, actionID string, data map[string]interface{}) {
	ev := Event{Type: typ, Time: time.Now().UTC().Format(time.RFC3339Nano), RunID: ctx.RunID, ActionID: actionID, Data: data}
	b, _ := json.Marshal(ev)
	_, _ = ctx.Events.Write(append(b, '\n'))
}

func writeSummary(ctx RunContext, goal string, satisfied bool, reason string) error {
	summary := map[string]interface{}{"run_id": ctx.RunID, "goal": goal, "satisfied": satisfied, "reason": reason, "config_hash": ctx.ConfigHash, "policy_hash": ctx.PolicyHash}
	b, _ := json.MarshalIndent(summary, "", "  ")
	return os.WriteFile(filepath.Join(ctx.RunDir, "summary.json"), append(b, '\n'), 0644)
}

func goalSatisfied(runDir string, goal Goal) bool {
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		return false
	}
	for _, req := range goal.RequiresEvidence {
		ok := false
		for _, as := range assertions {
			if as.Subject == req.Subject && as.Predicate == req.Predicate && confidenceAtLeast(as.Confidence, req.MinConfidence) && sourceAllowed(as.EvidenceSource, req.AnySourceType) {
				ok = true
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

func assertionsFromRun(runDir string) ([]AssertionRecord, error) {
	events, err := readEvents(runDir)
	if err != nil {
		return nil, err
	}
	var out []AssertionRecord
	for _, ev := range events {
		if ev.Type != "assertion_recorded" {
			continue
		}
		b, _ := json.Marshal(ev.Data)
		var rec AssertionRecord
		if err := json.Unmarshal(b, &rec); err == nil {
			out = append(out, rec)
		}
	}
	return out, nil
}

func readEvents(runDir string) ([]Event, error) {
	f, err := os.Open(filepath.Join(runDir, "events.jsonl"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var events []Event
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var ev Event
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	return events, scanner.Err()
}

func latestRunDir() (string, error) {
	root, err := findProjectRoot()
	if err != nil {
		return "", err
	}
	matches, err := filepath.Glob(filepath.Join(root, ".rp", "runs", "run-*"))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", errors.New("no runs found")
	}
	sort.Strings(matches)
	return matches[len(matches)-1], nil
}

func findOutputCapability(cfg Config, output string) string {
	for _, name := range sortedKeys(cfg.Capabilities) {
		if _, ok := cfg.Capabilities[name].Outputs[output]; ok {
			return name
		}
	}
	return ""
}

func findAssertionCapability(cfg Config, subject, predicate string) string {
	for _, name := range sortedKeys(cfg.Capabilities) {
		cap := cfg.Capabilities[name]
		for _, out := range cap.Outputs {
			for _, as := range out.Assertions {
				if as.Predicate == predicate && (as.Subject == subject || subject == "") {
					return name
				}
			}
		}
	}
	return ""
}

func checkCapabilityPolicy(cfg Config, cap Capability) error {
	policyName := cfg.Defaults["policy"]
	pol := cfg.Policies[policyName]
	networkAccess := permissionValue(pol.Permissions, "network", "access")
	if networkAccess == "" {
		networkAccess = "forbidden"
	}
	if networkAccess == "forbidden" && declaresNetworkEffect(cap) {
		return errors.New("policy forbids network access")
	}
	credentialUse := permissionValue(pol.Permissions, "credentials", "use")
	if credentialUse == "" {
		credentialUse = "forbidden"
	}
	if credentialUse == "forbidden" && declaresCredentialUse(cap) {
		return errors.New("policy forbids credential use")
	}
	return nil
}

func declaresNetworkEffect(cap Capability) bool {
	if len(cap.Effects.Network) > 0 {
		return true
	}
	return strings.Contains(strings.ToLower(cap.Effects.External), "network")
}

func declaresCredentialUse(cap Capability) bool {
	return strings.Contains(strings.ToLower(cap.Effects.External), "credential")
}

func needsWriteApproval(cfg Config, cap Capability) bool {
	if len(cap.Effects.Filesystem["writes"]) == 0 {
		return false
	}
	policyName := cfg.Defaults["policy"]
	pol := cfg.Policies[policyName]
	return permissionValue(pol.Permissions, "filesystem", "write") == "approval_required"
}

func permissionValue(permissions map[string]interface{}, section, key string) string {
	rawSection, ok := permissions[section]
	if !ok {
		return ""
	}
	values, ok := rawSection.(map[string]interface{})
	if !ok {
		return ""
	}
	value, _ := values[key].(string)
	return value
}

func environmentFor(cfg Config) []string {
	policyName := cfg.Defaults["policy"]
	pol := cfg.Policies[policyName]
	if pol.Environment.Inherit {
		return os.Environ()
	}
	allowed := pol.Environment.Allow
	if len(allowed) == 0 {
		allowed = []string{"PATH", "HOME"}
	}
	var env []string
	for _, key := range allowed {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	return env
}

func assertionMatches(as AssertionSpec, exitCode int, stdout string) bool {
	if len(as.When) == 0 {
		return true
	}
	if want, ok := as.When["exit_code"]; ok {
		if intFromAny(want) != exitCode {
			return false
		}
	}
	if pat, ok := as.When["stdout_matches"].(string); ok {
		matched, err := regexp.MatchString(pat, stdout)
		if err != nil || !matched {
			return false
		}
	}
	return true
}

func resolveSubject(subject string, step PlanStep) string {
	if subject == "repo" && step.Inputs["repo"] == "patched_repo" {
		return "patched_repo"
	}
	return subject
}

func renderDOT(plan []PlanStep) string {
	var b strings.Builder
	b.WriteString("digraph rp_plan {\n")
	for i, step := range plan {
		fmt.Fprintf(&b, "  n%d [label=\"%s\"];\n", i, step.Capability)
		if i > 0 {
			fmt.Fprintf(&b, "  n%d -> n%d;\n", i-1, i)
		}
	}
	b.WriteString("}\n")
	return b.String()
}

func renderMermaid(plan []PlanStep) string {
	var b strings.Builder
	b.WriteString("flowchart TD\n")
	for i, step := range plan {
		fmt.Fprintf(&b, "  n%d[\"%s\"]\n", i, step.Capability)
		if i > 0 {
			fmt.Fprintf(&b, "  n%d --> n%d\n", i-1, i)
		}
	}
	return b.String()
}

func canonicalHash(cfg Config) (string, error) {
	b, err := json.Marshal(cfg)
	if err != nil {
		return "", err
	}
	var normalized interface{}
	if err := json.Unmarshal(b, &normalized); err != nil {
		return "", err
	}
	b, err = json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	return sha(b), nil
}

func hashPolicy(cfg Config) string {
	name := cfg.Defaults["policy"]
	b, _ := json.Marshal(cfg.Policies[name])
	return sha(b)
}

func writeIfMissing(path, body string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0644)
}

func writeYAML(path string, v interface{}) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}

func printJSON(v interface{}) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func realizationURI(res Resource) string {
	if len(res.Realizations) == 0 {
		return ""
	}
	return res.Realizations[0].URI
}

func pathFromURI(root, uri string) string {
	if strings.HasPrefix(uri, "file://") {
		p := strings.TrimPrefix(uri, "file://")
		if p == "." {
			return root
		}
		if filepath.IsAbs(p) {
			return p
		}
		return filepath.Join(root, p)
	}
	return uri
}

func firstResourceOfType(cfg Config, typ string) string {
	for _, name := range sortedKeys(cfg.Resources) {
		if cfg.Resources[name].Type == typ {
			return name
		}
	}
	return ""
}

func safeName(s string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]+`)
	return strings.Trim(re.ReplaceAllString(s, "-"), "-")
}

func sha(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func confidenceAtLeast(got, min string) bool {
	return confidenceRank[got] >= confidenceRank[min]
}

func sourceAllowed(got string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, source := range allowed {
		if got == source {
			return true
		}
	}
	return false
}

func intFromAny(v interface{}) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func structToMap(v interface{}) map[string]interface{} {
	b, _ := json.Marshal(v)
	var out map[string]interface{}
	_ = json.Unmarshal(b, &out)
	return out
}

func ask(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

func normalizeFlagArgs(args []string, valueFlags, boolFlags map[string]bool) []string {
	var flags []string
	var positionals []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positionals = append(positionals, arg)
			continue
		}
		flags = append(flags, arg)
		name := strings.TrimLeft(arg, "-")
		if before, _, ok := strings.Cut(name, "="); ok {
			name = before
		}
		if boolFlags[name] {
			continue
		}
		if valueFlags[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, positionals...)
}
