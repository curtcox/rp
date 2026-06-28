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
	"strconv"
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
	External            string                 `yaml:"external,omitempty" json:"external,omitempty"`
	Planner             string                 `yaml:"planner,omitempty" json:"planner,omitempty"`
	Filesystem          map[string][]string    `yaml:"filesystem,omitempty" json:"filesystem,omitempty"`
	Network             map[string]interface{} `yaml:"network,omitempty" json:"network,omitempty"`
	ExternalSideEffects map[string]interface{} `yaml:"external_side_effects,omitempty" json:"external_side_effects,omitempty"`
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

type SavedPlan struct {
	ID         string     `json:"id"`
	Goal       string     `json:"goal"`
	ConfigHash string     `json:"config_hash"`
	PolicyHash string     `json:"policy_hash"`
	CreatedAt  string     `json:"created_at"`
	Steps      []PlanStep `json:"steps"`
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
	Supersedes     string `json:"supersedes,omitempty"`
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
	case "trace":
		return cmdTrace(args[1:])
	case "observe":
		return cmdObserve(args[1:])
	case "attest":
		return cmdAttest(args[1:])
	case "audit":
		return cmdAudit(args[1:])
	case "replay":
		return cmdReplay(args[1:])
	case "replan":
		return cmdReplan(args[1:])
	case "rerun":
		return cmdRerun(args[1:])
	case "exec":
		return cmdExec(args[1:])
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
  rp achieve GOAL [--dry-run] [--step] [--yes] [--auto-repair] [--max-attempts N]
  rp exec PLAN_ID [--dry-run] [--step] [--yes]
  rp evidence GOAL
  rp why SUBJECT.PREDICATE
  rp trace QUERY
  rp observe RESOURCE --with git_status
  rp attest SUBJECT.PREDICATE --source SOURCE [--note NOTE]
  rp add assertion SUBJECT.PREDICATE [--subject SUBJECT] [--confidence LEVEL]
  rp audit RUN_ID
  rp replay RUN_ID
  rp replan RUN_ID [--yes] [--step]
  rp rerun RUN_ID [--yes] [--step] [--auto-repair] [--max-attempts N]`)
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
      external_side_effects:
        create_pull_request: forbidden
        send_message: forbidden
        deploy: forbidden
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
	if len(args) > 0 && args[0] == "assertion" {
		return cmdAddAssertion(args[1:])
	}
	if len(args) < 2 || args[0] != "resource" {
		return errors.New("usage: rp add resource NAME --type TYPE (--uri URI | --file PATH) [--media-type TYPE] OR rp add assertion SUBJECT.PREDICATE")
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

func cmdAddAssertion(args []string) error {
	fs := flag.NewFlagSet("add assertion", flag.ContinueOnError)
	subjectFlag := fs.String("subject", "", "assertion subject override")
	object := fs.String("object", "", "assertion object")
	confidence := fs.String("confidence", "claimed", "confidence level")
	source := fs.String("source", "manual_entry", "evidence source type")
	note := fs.String("note", "", "note")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"subject": true, "object": true, "confidence": true, "source": true, "note": true}, map[string]bool{})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp add assertion SUBJECT.PREDICATE [--subject SUBJECT] [--confidence LEVEL]")
	}
	subject, predicate, err := parseAssertionTarget(fs.Arg(0), *subjectFlag)
	if err != nil {
		return err
	}
	return appendManualAssertion(subject, predicate, *object, *confidence, *source, *note, "manual-assertion")
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
	speculative := fs.Bool("speculative", false, "show assumed preconditions without saving a plan snapshot")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"format": true}, map[string]bool{"explain": true, "speculative": true})); err != nil {
		return err
	}
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
	var saved SavedPlan
	if !*speculative {
		saved, err = savePlan(root, fs.Arg(0), hash, hashPolicy(cfg), plan)
		if err != nil {
			return err
		}
	}
	switch *format {
	case "json":
		if *speculative {
			return printJSON(map[string]interface{}{
				"goal": fs.Arg(0), "speculative": true, "steps": plan,
				"assumptions": planAssumptions(cfg, plan), "effects": summarizePlanEffects(cfg, plan),
			})
		}
		return printJSON(saved)
	case "dot":
		fmt.Print(renderDOT(plan))
	case "mermaid":
		fmt.Print(renderMermaid(plan))
	case "text":
		if *speculative {
			fmt.Printf("Goal: %s (speculative — plan not saved)\nConfig: %s\nRoot: %s\n\n", fs.Arg(0), hash[:12], root)
		} else {
			fmt.Printf("Goal: %s\nConfig: %s\nRoot: %s\nSaved plan: %s\n\n", fs.Arg(0), hash[:12], root, saved.ID)
		}
		for i, step := range plan {
			fmt.Printf("%d. %s\n   capability: %s\n   reason: %s\n", i+1, step.ID, step.Capability, step.Reason)
			if *explain {
				fmt.Printf("   inputs: %v\n", step.Inputs)
			}
		}
		if *speculative {
			printPlanAssumptions(cfg, plan)
		}
		printEffectSummary(cfg, plan)
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
	autoRepair := fs.Bool("auto-repair", false, "retry failed steps by replanning")
	maxAttempts := fs.Int("max-attempts", 0, "max failure retries when auto-repair is enabled (0 uses policy default)")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"max-attempts": true}, map[string]bool{"dry-run": true, "step": true, "yes": true, "auto-repair": true})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp achieve GOAL")
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	plan, err := buildPlan(cfg, fs.Arg(0))
	if err != nil {
		return err
	}
	repairEnabled, repairMax := autoRepairSettings(cfg, *autoRepair, *maxAttempts)
	return runPlan(root, cfg, configHash, fs.Arg(0), plan, "", *dryRun, *stepMode, *yes, true, repairEnabled, repairMax, "")
}

func runPlan(root string, cfg Config, configHash, goalName string, plan []PlanStep, planID string, dryRun, stepMode, yes, jit bool, autoRepair bool, maxAttempts int, resumeRunID string) error {
	if dryRun {
		fmt.Printf("Plan for %s (%d steps) [dry-run — no execution]\n", goalName, len(plan))
		for i, step := range plan {
			fmt.Printf("  %d. %s — %s\n", i+1, step.Capability, step.Reason)
		}
		printEffectSummary(cfg, plan)
		return nil
	}
	var ctx RunContext
	var err error
	resuming := resumeRunID != ""
	if resuming {
		ctx, err = openRun(root, resumeRunID)
		if err != nil {
			return err
		}
		if ctx.ConfigHash != configHash {
			return fmt.Errorf("run %s config hash %s does not match current config %s; replan with matching config or re-achieve", resumeRunID, shortHash(ctx.ConfigHash), shortHash(configHash))
		}
	} else {
		ctx, err = newRun(root, configHash, hashPolicy(cfg))
		if err != nil {
			return err
		}
	}
	defer ctx.Events.Close()
	if !resuming {
		startData := map[string]interface{}{"goal": goalName, "config_hash": configHash}
		if planID != "" {
			startData["plan_id"] = planID
		}
		appendEvent(ctx, "run_started", "", startData)
		planData := map[string]interface{}{"steps": plan}
		if planID != "" {
			planData["plan_id"] = planID
		}
		appendEvent(ctx, "plan_proposed", "", planData)
		fmt.Printf("Plan for %s (%d steps):\n", goalName, len(plan))
		for i, step := range plan {
			fmt.Printf("  %d. %s — %s\n", i+1, step.Capability, step.Reason)
		}
		printEffectSummary(cfg, plan)
	} else {
		appendEvent(ctx, "plan_revised", "", map[string]interface{}{"steps": plan, "reason": "replan from prior run"})
		fmt.Printf("Replan for %s in %s (%d steps):\n", goalName, ctx.RunID, len(plan))
		for i, step := range plan {
			fmt.Printf("  %d. %s — %s\n", i+1, step.Capability, step.Reason)
		}
	}
	resources := map[string]string{}
	for k := range cfg.Resources {
		resources[k] = k
	}
	goal := cfg.Goals[goalName]
	executedCaps := map[string]bool{}
	stepNum := 0
	if resuming {
		executedCaps = executedCapabilitiesFromEvents(ctx.RunDir)
		events, _ := readEvents(ctx.RunDir)
		for _, ev := range events {
			if ev.Type == "action_completed" || ev.Type == "action_failed" {
				stepNum++
			}
		}
	}
	failureCount := 0
	lastPlanCaps := planCapabilities(plan)
	for {
		if len(missingEvidenceWithConfig(ctx.RunDir, goal, cfg)) == 0 {
			break
		}
		var steps []PlanStep
		if jit {
			steps, err = buildPlan(cfg, goalName)
			if err != nil {
				appendEvent(ctx, "run_stopped", "", map[string]interface{}{"reason": err.Error()})
				writeSummary(ctx, goalName, false, err.Error(), missingEvidenceWithConfig(ctx.RunDir, goal, cfg), missingProduce(ctx.RunDir, goal))
				return err
			}
			newCaps := planCapabilities(steps)
			if stepNum > 0 && !sameCapabilities(lastPlanCaps, newCaps) {
				if confirm, reason := planRevisionNeedsConfirmation(cfg, lastPlanCaps, newCaps); confirm && !yes {
					if !ask("plan revised ("+reason+"), continue?") {
						appendEvent(ctx, "run_stopped", "", map[string]interface{}{"reason": "plan revision denied"})
						return errors.New("plan revision denied")
					}
				}
				appendEvent(ctx, "plan_revised", "", map[string]interface{}{"steps": steps, "reason": "evidence gap replan"})
				fmt.Printf("Plan revised (%d steps):\n", len(steps))
				for i, s := range steps {
					if executedCaps[s.Capability] {
						continue
					}
					fmt.Printf("  %d. %s — %s\n", i+1, s.Capability, s.Reason)
				}
			}
			lastPlanCaps = newCaps
		} else {
			steps = plan
		}
		var step *PlanStep
		for i := range steps {
			if !executedCaps[steps[i].Capability] {
				step = &steps[i]
				break
			}
		}
		if step == nil {
			break
		}
		stepNum++
		step.ID = fmt.Sprintf("step-%02d", stepNum)
		if stepMode && !ask("execute "+step.Capability+"?") {
			appendEvent(ctx, "run_stopped", "", map[string]interface{}{"reason": "step denied"})
			return errors.New("stopped by user")
		}
		capability := cfg.Capabilities[step.Capability]
		if err := checkCapabilityPolicy(cfg, capability); err != nil {
			appendEvent(ctx, "run_stopped", step.ID, map[string]interface{}{"reason": err.Error()})
			writeSummary(ctx, goalName, false, err.Error(), missingEvidenceWithConfig(ctx.RunDir, goal, cfg), missingProduce(ctx.RunDir, goal))
			return err
		}
		if err := checkGoalConstraints(goal, capability); err != nil {
			appendEvent(ctx, "run_stopped", step.ID, map[string]interface{}{"reason": err.Error()})
			writeSummary(ctx, goalName, false, err.Error(), missingEvidenceWithConfig(ctx.RunDir, goal, cfg), missingProduce(ctx.RunDir, goal))
			return err
		}
		if err := checkStepPreconditions(ctx.RunDir, cfg, *step); err != nil {
			appendEvent(ctx, "plan_invalidated", step.ID, map[string]interface{}{"reason": err.Error(), "suggest": "replan"})
			stopData, summaryReason := failureStopData(cfg, ctx.RunID, err.Error())
			appendEvent(ctx, "run_stopped", "", stopData)
			writeSummary(ctx, goalName, false, summaryReason, missingEvidenceWithConfig(ctx.RunDir, goal, cfg), missingProduce(ctx.RunDir, goal))
			return err
		}
		if perm := capabilityApprovalPermission(cfg, capability); perm != "" && !yes {
			appendEvent(ctx, "approval_requested", step.ID, map[string]interface{}{"permission": perm})
			if !ask("approve "+perm+" for "+step.Capability+"?") {
				appendEvent(ctx, "approval_denied", step.ID, nil)
				return errors.New("approval denied")
			}
			appendEvent(ctx, "approval_granted", step.ID, nil)
		}
		if err := executeStep(ctx, cfg, *step, resources); err != nil {
			failureCount++
			appendEvent(ctx, "plan_invalidated", step.ID, map[string]interface{}{"reason": err.Error(), "suggest": "replan"})
			if autoRepair && failureCount < maxAttempts {
				appendEvent(ctx, "auto_repair_attempted", step.ID, map[string]interface{}{
					"attempt": failureCount, "max_attempts": maxAttempts, "capability": step.Capability,
				})
				fmt.Printf("auto-repair: replanning after %s failure (%d/%d)\n", step.Capability, failureCount, maxAttempts)
				continue
			}
			stopData, summaryReason := failureStopData(cfg, ctx.RunID, err.Error())
			appendEvent(ctx, "run_stopped", "", stopData)
			writeSummary(ctx, goalName, false, summaryReason, missingEvidenceWithConfig(ctx.RunDir, goal, cfg), missingProduce(ctx.RunDir, goal))
			return err
		}
		executedCaps[step.Capability] = true
		missing := missingEvidenceWithConfig(ctx.RunDir, goal, cfg)
		appendEvent(ctx, "goal_gap_evaluated", step.ID, map[string]interface{}{
			"missing_count": len(missing),
			"missing":       missing,
		})
		if len(missing) == 0 {
			appendEvent(ctx, "run_stopped", step.ID, map[string]interface{}{"reason": "goal evidence requirements satisfied"})
			break
		}
		if !jit {
			continue
		}
	}
	missing := missingEvidenceWithConfig(ctx.RunDir, cfg.Goals[goalName], cfg)
	produceGaps := missingProduce(ctx.RunDir, goal)
	satisfied := len(missing) == 0 && len(produceGaps) == 0
	appendEvent(ctx, "goal_satisfied", "", map[string]interface{}{"satisfied": satisfied})
	reason := "goal evidence requirements satisfied"
	if !satisfied {
		switch {
		case len(missing) > 0 && len(produceGaps) > 0:
			reason = "goal evidence and produce requirements not fully satisfied"
		case len(produceGaps) > 0:
			reason = "goal produce requirements not fully satisfied"
		default:
			reason = "goal evidence requirements not fully satisfied"
		}
	} else {
		recordGoalAttestation(ctx, cfg, goal)
	}
	if err := writeSummary(ctx, goalName, satisfied, reason, missing, produceGaps); err != nil {
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
	goalName := args[0]
	_, cfg, _, err := loadProject()
	if err != nil {
		return err
	}
	goal, ok := cfg.Goals[goalName]
	if !ok {
		return fmt.Errorf("goal %q not found", goalName)
	}
	runDir, err := latestRunDir()
	if err != nil {
		return err
	}
	summary := loadRunSummary(runDir)
	if g, _ := summary["goal"].(string); g != "" && g != goalName {
		fmt.Printf("note: latest run goal is %q, not %q\n", g, goalName)
	}
	satisfied := goalSatisfied(runDir, goal, cfg)
	fmt.Printf("Goal %s (run %s)\n", goalName, filepath.Base(runDir))
	fmt.Printf("Satisfied: %v\n\n", satisfied)
	if len(goal.Produce) > 0 {
		fmt.Println("Required outputs:")
		realizations := realizationsFromRun(runDir)
		for _, name := range sortedKeys(goal.Produce) {
			spec := goal.Produce[name]
			status := "missing"
			detail := ""
			if rec, ok := realizations[name]; ok {
				if reason := produceSpecMismatch(runDir, spec, rec); reason == "" {
					status = "ok"
					if rec.MediaType != "" {
						detail = rec.MediaType
					}
					if rec.Artifact != "" {
						if detail != "" {
							detail += " at "
						}
						detail += rec.Artifact
					}
				} else {
					status = "mismatch"
					detail = reason
				}
			}
			line := fmt.Sprintf("  [%s] %s (%s)", status, name, spec.Type)
			if detail != "" {
				line += " — " + detail
			}
			fmt.Println(line)
		}
		fmt.Println()
	}
	fmt.Println("Required evidence:")
	assertions, _ := assertionsFromRun(runDir)
	effective := effectiveAssertions(assertions)
	for _, req := range goal.RequiresEvidence {
		status := "missing"
		detail := ""
		for _, as := range effective {
			if as.Subject == req.Subject && as.Predicate == req.Predicate &&
				confidenceAtLeast(as.Confidence, req.MinConfidence) &&
				sourceAllowed(as.EvidenceSource, req.AnySourceType) &&
				sourceMaySatisfyRequiredEvidence(cfg, as.EvidenceSource) {
				status = "ok"
				detail = fmt.Sprintf("%s via %s (%s)", as.Confidence, as.EvidenceSource, as.ID)
				break
			}
		}
		fmt.Printf("  [%s] %s.%s >= %s", status, req.Subject, req.Predicate, req.MinConfidence)
		if detail != "" {
			fmt.Printf(" — %s", detail)
		}
		fmt.Println()
	}
	missing := missingEvidenceWithConfig(runDir, goal, cfg)
	if len(missing) > 0 {
		fmt.Printf("\nMissing requirements: %d\n", len(missing))
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
	for _, a := range effectiveAssertions(assertions) {
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

func cmdTrace(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp trace QUERY")
	}
	runDir, err := latestRunDir()
	if err != nil {
		return err
	}
	events, err := readEvents(runDir)
	if err != nil {
		return err
	}
	query := args[0]
	found := false
	for _, ev := range events {
		if eventMatches(ev, query) {
			found = true
			b, _ := json.MarshalIndent(ev, "", "  ")
			fmt.Println(string(b))
		}
	}
	if !found {
		fmt.Printf("no trace entries for %q in latest run %s\n", query, filepath.Base(runDir))
	}
	return nil
}

func cmdObserve(args []string) error {
	fs := flag.NewFlagSet("observe", flag.ContinueOnError)
	with := fs.String("with", "", "observer")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"with": true}, map[string]bool{})); err != nil {
		return err
	}
	if fs.NArg() != 1 || *with == "" {
		return errors.New("usage: rp observe RESOURCE --with git_status")
	}
	if *with != "git_status" {
		return fmt.Errorf("unsupported observer %q", *with)
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	res, ok := cfg.Resources[fs.Arg(0)]
	if !ok {
		return fmt.Errorf("resource %q not found", fs.Arg(0))
	}
	path := pathFromURI(root, realizationURI(res))
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path
	cmd.Env = environmentFor(cfg)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	exitCode := 0
	if err != nil {
		if ee := new(exec.ExitError); errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}
	ctx, err := newManualRun(root, configHash, hashPolicy(cfg), "observe "+fs.Arg(0)+" with git_status")
	if err != nil {
		return err
	}
	defer ctx.Events.Close()
	actionID := "manual-observe-" + safeName(fs.Arg(0)+"-git-status")
	obsID := "obs-" + actionID
	evidenceID := "ev-" + actionID
	appendEvent(ctx, "action_started", actionID, map[string]interface{}{"observer": "git_status", "resource": fs.Arg(0)})
	obsData := map[string]interface{}{
		"id": obsID, "source_type": "process_exit", "exit_code": exitCode,
	}
	if policyHashing(cfg, "command_stdout") {
		obsData["stdout_sha256"] = sha(stdout.Bytes())
	}
	if policyHashing(cfg, "command_stderr") {
		obsData["stderr_sha256"] = sha(stderr.Bytes())
	}
	appendEvent(ctx, "observation_recorded", actionID, obsData)
	appendEvent(ctx, "evidence_recorded", actionID, map[string]interface{}{
		"id": evidenceID, "source_type": "process_exit", "observation_id": obsID,
		"confidence_contribution": "observed",
	})
	if exitCode == 0 && stdout.Len() == 0 {
		rec := AssertionRecord{
			ID: "as-" + actionID + "-clean-worktree", Subject: fs.Arg(0), Predicate: "clean_worktree",
			Confidence: "observed", EvidenceID: evidenceID, EvidenceSource: "process_exit", ActionID: actionID,
		}
		recordAssertion(ctx, actionID, rec)
	}
	appendEvent(ctx, "action_completed", actionID, map[string]interface{}{"exit_code": exitCode})
	if err := writeSummary(ctx, "manual_observe", exitCode == 0, "manual git_status observation recorded", nil, nil); err != nil {
		return err
	}
	fmt.Println(ctx.RunDir)
	if exitCode != 0 {
		return fmt.Errorf("git_status observer failed with exit code %d", exitCode)
	}
	return nil
}

func cmdAttest(args []string) error {
	fs := flag.NewFlagSet("attest", flag.ContinueOnError)
	subjectFlag := fs.String("subject", "", "assertion subject override")
	source := fs.String("source", "human_review", "evidence source type")
	note := fs.String("note", "", "note")
	confidence := fs.String("confidence", "attested", "confidence level")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"subject": true, "source": true, "note": true, "confidence": true}, map[string]bool{})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp attest SUBJECT.PREDICATE --source SOURCE [--note NOTE]")
	}
	subject, predicate, err := parseAssertionTarget(fs.Arg(0), *subjectFlag)
	if err != nil {
		return err
	}
	return appendManualAssertion(subject, predicate, "", *confidence, *source, *note, "manual-attestation")
}

func cmdAudit(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp audit RUN_ID")
	}
	runID := args[0]
	runDir := filepath.Join(".rp", "runs", runID)
	events, err := readEvents(runDir)
	if err != nil {
		return err
	}
	summary := loadRunSummary(runDir)
	fmt.Printf("Audit %s\n", runID)
	if goal, _ := summary["goal"].(string); goal != "" {
		fmt.Printf("Goal: %s\n", goal)
	}
	if satisfied, ok := summary["satisfied"].(bool); ok {
		fmt.Printf("Satisfied: %v\n", satisfied)
	}
	if reason, _ := summary["reason"].(string); reason != "" {
		fmt.Printf("Reason: %s\n", reason)
	}
	if h, _ := summary["config_hash"].(string); h != "" {
		fmt.Printf("Config: %s\n", shortHash(h))
	}
	if h, _ := summary["policy_hash"].(string); h != "" {
		fmt.Printf("Policy: %s\n", shortHash(h))
	}
	counts := map[string]int{}
	var attestationID string
	for _, ev := range events {
		counts[ev.Type]++
		if ev.Type == "attestation_recorded" {
			attestationID, _ = ev.Data["id"].(string)
		}
	}
	if len(counts) > 0 {
		fmt.Printf("Events: %d\n", len(events))
		for _, typ := range sortedKeys(counts) {
			fmt.Printf("  %s: %d\n", typ, counts[typ])
		}
	}
	if attestationID != "" {
		fmt.Printf("Attestation: %s\n", attestationID)
	}
	fmt.Println()
	for _, ev := range events {
		fmt.Printf("%s\t%s\t%s\n", ev.Time, ev.Type, ev.ActionID)
	}
	return nil
}

func cmdReplay(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: rp replay RUN_ID")
	}
	runID := args[0]
	runDir := filepath.Join(".rp", "runs", runID)
	events, err := readEvents(runDir)
	if err != nil {
		return err
	}
	summary := map[string]interface{}{}
	if b, err := os.ReadFile(filepath.Join(runDir, "summary.json")); err == nil {
		_ = json.Unmarshal(b, &summary)
	}
	fmt.Printf("Replay %s\n", runID)
	if goal, _ := summary["goal"].(string); goal != "" {
		fmt.Printf("Goal: %s\n", goal)
	}
	if satisfied, ok := summary["satisfied"].(bool); ok {
		fmt.Printf("Satisfied: %v\n", satisfied)
	}
	if reason, _ := summary["reason"].(string); reason != "" {
		fmt.Printf("Reason: %s\n", reason)
	}
	fmt.Println()
	for _, ev := range events {
		switch ev.Type {
		case "run_started":
			fmt.Printf("[%s] run started", shortTime(ev.Time))
			if g, _ := ev.Data["goal"].(string); g != "" {
				fmt.Printf(" goal=%s", g)
			}
			fmt.Println()
		case "plan_proposed", "plan_revised":
			label := "plan proposed"
			if ev.Type == "plan_revised" {
				label = "plan revised"
			}
			n := planStepCount(ev.Data["steps"])
			fmt.Printf("[%s] %s (%d steps)\n", shortTime(ev.Time), label, n)
		case "approval_requested":
			fmt.Printf("[%s] approval requested for %s (%v)\n", shortTime(ev.Time), ev.ActionID, ev.Data["permission"])
		case "approval_granted":
			fmt.Printf("[%s] approval granted for %s\n", shortTime(ev.Time), ev.ActionID)
		case "approval_denied":
			fmt.Printf("[%s] approval denied for %s\n", shortTime(ev.Time), ev.ActionID)
		case "action_started":
			capName, _ := ev.Data["capability"].(string)
			fmt.Printf("[%s] action started: %s (%s)\n", shortTime(ev.Time), ev.ActionID, capName)
		case "action_completed":
			fmt.Printf("[%s] action completed: %s exit=%v\n", shortTime(ev.Time), ev.ActionID, ev.Data["exit_code"])
		case "action_failed":
			fmt.Printf("[%s] action failed: %s exit=%v\n", shortTime(ev.Time), ev.ActionID, ev.Data["exit_code"])
		case "observation_recorded":
			fmt.Printf("[%s] observation %v exit=%v\n", shortTime(ev.Time), ev.Data["id"], ev.Data["exit_code"])
		case "evidence_recorded":
			fmt.Printf("[%s] evidence %v (%v)\n", shortTime(ev.Time), ev.Data["id"], ev.Data["source_type"])
		case "assertion_recorded":
			subject, _ := ev.Data["subject"].(string)
			predicate, _ := ev.Data["predicate"].(string)
			confidence, _ := ev.Data["confidence"].(string)
			source, _ := ev.Data["evidence_source"].(string)
			fmt.Printf("[%s] assertion %s.%s confidence=%s source=%s\n", shortTime(ev.Time), subject, predicate, confidence, source)
		case "assertion_superseded":
			fmt.Printf("[%s] assertion superseded: %v by %v\n", shortTime(ev.Time), ev.Data["id"], ev.Data["superseded_by"])
		case "resource_realization_recorded":
			fmt.Printf("[%s] resource realization %v\n", shortTime(ev.Time), ev.Data["resource"])
		case "artifact_recorded":
			fmt.Printf("[%s] artifact %v\n", shortTime(ev.Time), ev.Data["path"])
		case "goal_gap_evaluated":
			fmt.Printf("[%s] goal gap: %v missing requirement(s)\n", shortTime(ev.Time), ev.Data["missing_count"])
		case "goal_satisfied":
			fmt.Printf("[%s] goal satisfied=%v\n", shortTime(ev.Time), ev.Data["satisfied"])
		case "attestation_recorded":
			fmt.Printf("[%s] attestation %v (%d assertions)\n", shortTime(ev.Time), ev.Data["id"], lenSlice(ev.Data["assertion_ids"]))
		case "plan_invalidated":
			fmt.Printf("[%s] plan invalidated: %v\n", shortTime(ev.Time), ev.Data["reason"])
		case "run_stopped":
			fmt.Printf("[%s] run stopped: %v\n", shortTime(ev.Time), ev.Data["reason"])
		default:
			fmt.Printf("[%s] %s\n", shortTime(ev.Time), ev.Type)
		}
	}
	assertions, err := assertionsFromRun(runDir)
	if err == nil && len(assertions) > 0 {
		fmt.Println("\nAssertions:")
		for _, as := range effectiveAssertions(assertions) {
			line := fmt.Sprintf("  %s.%s %s (via %s, action %s)", as.Subject, as.Predicate, as.Confidence, as.EvidenceSource, as.ActionID)
			if as.Supersedes != "" {
				line += fmt.Sprintf(" supersedes %s", as.Supersedes)
			}
			fmt.Println(line)
		}
	}
	return nil
}

func cmdReplan(args []string) error {
	fs := flag.NewFlagSet("replan", flag.ContinueOnError)
	stepMode := fs.Bool("step", false, "confirm every step")
	yes := fs.Bool("yes", false, "approve writes and continue execution in the prior run")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{}, map[string]bool{"step": true, "yes": true})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp replan RUN_ID [--yes] [--step]")
	}
	runID := fs.Arg(0)
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	summary := loadRunSummary(filepath.Join(".rp", "runs", runID))
	goal, _ := summary["goal"].(string)
	if goal == "" {
		return errors.New("run summary has no goal")
	}
	if _, ok := cfg.Goals[goal]; !ok {
		return fmt.Errorf("goal %q from run is not in current config", goal)
	}
	missing := missingEvidenceWithConfig(filepath.Join(".rp", "runs", runID), cfg.Goals[goal], cfg)
	fmt.Printf("Prior run %s goal=%s satisfied=%v\n", runID, goal, len(missing) == 0)
	if len(missing) > 0 {
		fmt.Println("Missing evidence:")
		for _, req := range missing {
			fmt.Printf("  - %s.%s >= %s\n", req.Subject, req.Predicate, req.MinConfidence)
		}
		fmt.Println()
	}
	plan, err := buildPlan(cfg, goal)
	if err != nil {
		return err
	}
	fmt.Printf("Revised plan (%d steps):\n", len(plan))
	for i, step := range plan {
		fmt.Printf("  %d. %s — %s\n", i+1, step.Capability, step.Reason)
		if *yes {
			fmt.Printf("     inputs: %v\n", step.Inputs)
		}
	}
	if !*yes {
		fmt.Println("\nUse --yes to continue execution in the prior run.")
		return nil
	}
	repairEnabled, repairMax := autoRepairSettings(cfg, false, 0)
	return runPlan(root, cfg, configHash, goal, plan, "", false, *stepMode, true, true, repairEnabled, repairMax, runID)
}

func cmdRerun(args []string) error {
	fs := flag.NewFlagSet("rerun", flag.ContinueOnError)
	stepMode := fs.Bool("step", false, "confirm every step")
	yes := fs.Bool("yes", false, "approve required filesystem writes")
	autoRepair := fs.Bool("auto-repair", false, "retry failed steps by replanning")
	maxAttempts := fs.Int("max-attempts", 0, "max failure retries when auto-repair is enabled (0 uses policy default)")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{"max-attempts": true}, map[string]bool{"step": true, "yes": true, "auto-repair": true})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp rerun RUN_ID [--yes] [--step] [--auto-repair] [--max-attempts N]")
	}
	runID := fs.Arg(0)
	goal, err := goalFromRun(runID)
	if err != nil {
		return err
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	plan, err := buildPlan(cfg, goal)
	if err != nil {
		return err
	}
	repairEnabled, repairMax := autoRepairSettings(cfg, *autoRepair, *maxAttempts)
	return runPlan(root, cfg, configHash, goal, plan, "", false, *stepMode, *yes, true, repairEnabled, repairMax, "")
}

func cmdExec(args []string) error {
	fs := flag.NewFlagSet("exec", flag.ContinueOnError)
	stepMode := fs.Bool("step", false, "confirm every step")
	yes := fs.Bool("yes", false, "approve required filesystem writes")
	dryRun := fs.Bool("dry-run", false, "show saved plan without execution")
	if err := fs.Parse(normalizeFlagArgs(args, map[string]bool{}, map[string]bool{"step": true, "yes": true, "dry-run": true})); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("usage: rp exec PLAN_ID")
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	plan, err := loadSavedPlan(root, fs.Arg(0))
	if err != nil {
		return err
	}
	if plan.ConfigHash != configHash {
		return fmt.Errorf("saved plan %s was built for config %s, current config is %s; replan before exec", plan.ID, shortHash(plan.ConfigHash), shortHash(configHash))
	}
	repairEnabled, repairMax := autoRepairSettings(cfg, false, 0)
	return runPlan(root, cfg, configHash, plan.Goal, plan.Steps, plan.ID, *dryRun, *stepMode, *yes, false, repairEnabled, repairMax, "")
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
	cfg = applyUserPolicy(cfg)
	hash, err := canonicalHash(cfg)
	if err != nil {
		return "", Config{}, "", err
	}
	return root, cfg, hash, nil
}

func applyUserPolicy(cfg Config) Config {
	user, ok, err := loadUserPolicy()
	if err != nil || !ok {
		return cfg
	}
	policyName := cfg.Defaults["policy"]
	if policyName == "" {
		return cfg
	}
	if cfg.Policies == nil {
		cfg.Policies = map[string]Policy{}
	}
	project := cfg.Policies[policyName]
	cfg.Policies[policyName] = mergePolicies(project, user)
	return cfg
}

func loadUserPolicy() (Policy, bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Policy{}, false, err
	}
	path := filepath.Join(home, ".config", "rp", "policy.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Policy{}, false, nil
		}
		return Policy{}, false, err
	}
	var doc struct {
		Version  string             `yaml:"version"`
		Policies map[string]Policy  `yaml:"policies,omitempty"`
		Policy   Policy             `yaml:"policy,omitempty"`
	}
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return Policy{}, false, fmt.Errorf("%s: %w", path, err)
	}
	if doc.Version != "" && doc.Version != version {
		return Policy{}, false, fmt.Errorf("%s: unsupported version %q", path, doc.Version)
	}
	if len(doc.Policies) > 0 {
		for _, name := range sortedKeys(doc.Policies) {
			return doc.Policies[name], true, nil
		}
	}
	if doc.Policy.Description != "" || len(doc.Policy.Permissions) > 0 {
		return doc.Policy, true, nil
	}
	return Policy{}, false, nil
}

func mergePolicies(project, user Policy) Policy {
	out := project
	out.Permissions = mergePermissionMaps(project.Permissions, user.Permissions)
	if len(user.Environment.Allow) > 0 {
		out.Environment.Inherit = project.Environment.Inherit && user.Environment.Inherit
		out.Environment.Allow = intersectStrings(project.Environment.Allow, user.Environment.Allow)
	}
	out.Evidence = mergeEvidenceMaps(project.Evidence, user.Evidence)
	out.MaxCost = mergeCostMaps(project.MaxCost, user.MaxCost)
	return out
}

func mergePermissionMaps(project, user map[string]interface{}) map[string]interface{} {
	if len(project) == 0 {
		return user
	}
	if len(user) == 0 {
		return project
	}
	out := map[string]interface{}{}
	for _, section := range sortedKeys(unionMapKeys(project, user)) {
		pSection, _ := project[section].(map[string]interface{})
		uSection, _ := user[section].(map[string]interface{})
		if pSection == nil && uSection == nil {
			continue
		}
		if pSection == nil {
			out[section] = uSection
			continue
		}
		if uSection == nil {
			out[section] = pSection
			continue
		}
		merged := map[string]interface{}{}
		for _, key := range sortedKeys(unionMapKeys(pSection, uSection)) {
			pVal, _ := pSection[key].(string)
			uVal, _ := uSection[key].(string)
			switch {
			case pVal == "":
				merged[key] = uSection[key]
			case uVal == "":
				merged[key] = pSection[key]
			default:
				merged[key] = mostRestrictivePermission(pVal, uVal)
			}
		}
		out[section] = merged
	}
	return out
}

func mergeEvidenceMaps(project, user map[string]interface{}) map[string]interface{} {
	if len(user) == 0 {
		return project
	}
	if len(project) == 0 {
		return user
	}
	out := map[string]interface{}{}
	for k, v := range project {
		out[k] = v
	}
	for k, v := range user {
		switch k {
		case "source_limits", "final_goal_rules":
			out[k] = appendEvidenceRules(out[k], v)
		default:
			out[k] = v
		}
	}
	return out
}

func appendEvidenceRules(existing, extra interface{}) []interface{} {
	var rules []interface{}
	if existing != nil {
		if items, ok := existing.([]interface{}); ok {
			rules = append(rules, items...)
		}
	}
	if extra != nil {
		if items, ok := extra.([]interface{}); ok {
			rules = append(rules, items...)
		}
	}
	return rules
}

func mergeCostMaps(project, user map[string]interface{}) map[string]interface{} {
	if len(user) == 0 {
		return project
	}
	if len(project) == 0 {
		return user
	}
	out := map[string]interface{}{}
	for k, v := range project {
		out[k] = v
	}
	for k, uVal := range user {
		pVal, ok := out[k]
		if !ok {
			out[k] = uVal
			continue
		}
		if pStr, ok := pVal.(string); ok {
			if uStr, ok := uVal.(string); ok {
				out[k] = minCostString(pStr, uStr)
			}
		}
	}
	return out
}

func mostRestrictivePermission(a, b string) string {
	rank := map[string]int{"forbidden": 0, "approval_required": 1, "allowed": 2}
	if rank[a] <= rank[b] {
		return a
	}
	return b
}

func minCostString(a, b string) string {
	if strings.HasSuffix(a, "m") && strings.HasSuffix(b, "m") {
		aMin, errA := strconv.Atoi(strings.TrimSuffix(a, "m"))
		bMin, errB := strconv.Atoi(strings.TrimSuffix(b, "m"))
		if errA == nil && errB == nil && bMin < aMin {
			return b
		}
	}
	return a
}

func unionMapKeys(a, b map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for k := range a {
		out[k] = nil
	}
	for k := range b {
		out[k] = nil
	}
	return out
}

func intersectStrings(a, b []string) []string {
	if len(a) == 0 {
		return b
	}
	if len(b) == 0 {
		return a
	}
	set := map[string]bool{}
	for _, s := range a {
		set[s] = true
	}
	var out []string
	for _, s := range b {
		if set[s] {
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
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
	var root yaml.Node
	if err := yaml.Unmarshal(b, &root); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := validateYAMLDocument(&root, path); err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if cfg.Version != "" && cfg.Version != version {
		return Config{}, fmt.Errorf("%s: unsupported version %q", path, cfg.Version)
	}
	return cfg, nil
}

func checkStepPreconditions(runDir string, cfg Config, step PlanStep) error {
	capability := cfg.Capabilities[step.Capability]
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		return err
	}
	for inputName, input := range capability.Inputs {
		for _, req := range input.Requires {
			subject := req.Subject
			if subject == "" {
				subject = step.Inputs[inputName]
			}
			if subject == "" {
				subject = inputName
			}
			if !assertionRequirementMet(effectiveAssertions(assertions), req, subject, cfg) {
				return fmt.Errorf("precondition %s.%s>=%s not satisfied for %s", subject, req.Predicate, req.MinConfidence, step.Capability)
			}
		}
	}
	for _, req := range capability.Preconditions {
		subject := req.Subject
		if subject == "" {
			subject = step.Inputs["repo"]
		}
		if !assertionRequirementMet(effectiveAssertions(assertions), req, subject, cfg) {
			return fmt.Errorf("precondition %s.%s>=%s not satisfied for %s", subject, req.Predicate, req.MinConfidence, step.Capability)
		}
	}
	return nil
}

func assertionRequirementMet(assertions []AssertionRecord, req Requirement, subject string, cfg Config) bool {
	for _, as := range assertions {
		if as.Subject == subject && as.Predicate == req.Predicate &&
			confidenceAtLeast(as.Confidence, req.MinConfidence) &&
			sourceAllowed(as.EvidenceSource, req.AnySourceType) &&
			sourceMaySatisfyRequiredEvidence(cfg, as.EvidenceSource) {
			return true
		}
	}
	return false
}

func planCapabilities(plan []PlanStep) []string {
	caps := make([]string, len(plan))
	for i, step := range plan {
		caps[i] = step.Capability
	}
	return caps
}

func sameCapabilities(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func planStepCount(raw interface{}) int {
	items, ok := raw.([]interface{})
	if !ok {
		return 0
	}
	return len(items)
}

func lenSlice(raw interface{}) int {
	items, ok := raw.([]interface{})
	if !ok {
		return 0
	}
	return len(items)
}

func shortTime(ts string) string {
	if len(ts) >= 19 {
		return ts[:19]
	}
	return ts
}

var configRootKeys = map[string]bool{
	"version": true, "imports": true, "resources": true, "capabilities": true,
	"policies": true, "goals": true, "defaults": true,
}

var resourceKeys = map[string]bool{"type": true, "realizations": true}
var realizationKeys = map[string]bool{"id": true, "kind": true, "uri": true, "media_type": true, "hash": true, "metadata": true}
var capabilityKeys = map[string]bool{
	"purpose": true, "kind": true, "inputs": true, "outputs": true, "preconditions": true,
	"command": true, "approval": true, "always_record_result": true, "effects": true,
	"nondeterminism": true, "idempotence": true, "cost": true,
}
var inputSpecKeys = map[string]bool{"type": true, "realization": true, "requires": true}
var requirementKeys = map[string]bool{"subject": true, "predicate": true, "min_confidence": true, "any_source_type": true}
var outputSpecKeys = map[string]bool{"type": true, "required_realization": true, "realization": true, "assertions": true}
var assertionSpecKeys = map[string]bool{"subject": true, "predicate": true, "object": true, "confidence": true, "when": true, "evidence_source": true}
var commandKeys = map[string]bool{"cwd": true, "argv": true, "stdout": true, "stderr": true}
var streamKeys = map[string]bool{"save_as": true, "save_as_artifact": true, "media_type": true}
var saveAsKeys = map[string]bool{"resource": true, "artifact_path": true, "media_type": true}
var effectKeys = map[string]bool{"external": true, "planner": true, "filesystem": true, "network": true, "external_side_effects": true}
var policyKeys = map[string]bool{
	"description": true, "permissions": true, "environment": true, "evidence": true,
	"hashing": true, "execution": true, "max_cost": true,
}
var goalKeys = map[string]bool{"description": true, "given": true, "produce": true, "requires_evidence": true, "constraints": true}

func validateYAMLDocument(root *yaml.Node, path string) error {
	doc := root
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		doc = doc.Content[0]
	}
	return validateMapping(doc, path, configRootKeys, map[string]map[string]bool{
		"resources":    resourceKeys,
		"capabilities": capabilityKeys,
		"policies":     policyKeys,
		"goals":        goalKeys,
	})
}

func validateMapping(node *yaml.Node, path string, allowed map[string]bool, nested map[string]map[string]bool) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		key := keyNode.Value
		if isExtensionKey(key) {
			continue
		}
		if !allowed[key] {
			return fmt.Errorf("%s: unknown field %q (use x-* prefix for extensions)", path, key)
		}
		childPath := path + "." + key
		if valNode.Kind == yaml.MappingNode {
			if _, ok := nested[key]; ok {
				if key == "resources" || key == "capabilities" || key == "policies" || key == "goals" {
					if err := validateNamedItems(valNode, childPath, nested[key], nil); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func validateNamedItems(node *yaml.Node, path string, itemAllowed map[string]bool, _ map[string]map[string]bool) error {
	for i := 0; i < len(node.Content); i += 2 {
		name := node.Content[i].Value
		item := node.Content[i+1]
		itemPath := path + "." + name
		if err := validateMapping(item, itemPath, itemAllowed, nil); err != nil {
			return err
		}
		if item.Kind == yaml.MappingNode {
			if err := validateCapabilityBody(item, itemPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCapabilityBody(node *yaml.Node, path string) error {
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		childPath := path + "." + key
		switch key {
		case "inputs", "outputs":
			if val.Kind == yaml.MappingNode {
				itemKeys := inputSpecKeys
				if key == "outputs" {
					itemKeys = outputSpecKeys
				}
				for j := 0; j < len(val.Content); j += 2 {
					if err := validateMapping(val.Content[j+1], childPath+"."+val.Content[j].Value, itemKeys, map[string]map[string]bool{
						"realization": realizationKeys, "requires": requirementKeys, "assertions": assertionSpecKeys,
						"required_realization": realizationKeys,
					}); err != nil {
						return err
					}
					if err := validateInputOutputBody(val.Content[j+1], childPath+"."+val.Content[j].Value); err != nil {
						return err
					}
				}
			}
		case "command":
			if err := validateMapping(val, childPath, commandKeys, map[string]map[string]bool{
				"stdout": streamKeys, "stderr": streamKeys,
			}); err != nil {
				return err
			}
			for j := 0; j < len(val.Content); j += 2 {
				if val.Content[j].Value == "stdout" || val.Content[j].Value == "stderr" {
					if err := validateStream(val.Content[j+1], childPath+"."+val.Content[j].Value); err != nil {
						return err
					}
				}
			}
		case "effects":
			if err := validateMapping(val, childPath, effectKeys, nil); err != nil {
				return err
			}
			for j := 0; j < len(val.Content); j += 2 {
				if val.Content[j].Value == "filesystem" && val.Content[j+1].Kind == yaml.MappingNode {
					if err := validateOpenMapping(val.Content[j+1], childPath+".filesystem", map[string]bool{"writes": true, "reads": true}); err != nil {
						return err
					}
				}
			}
		case "preconditions":
			if val.Kind == yaml.SequenceNode {
				for _, item := range val.Content {
					if err := validateMapping(item, childPath+"[]", requirementKeys, nil); err != nil {
						return err
					}
				}
			}
		case "realizations":
			if val.Kind == yaml.SequenceNode {
				for _, item := range val.Content {
					if err := validateMapping(item, childPath+"[]", realizationKeys, nil); err != nil {
						return err
					}
				}
			}
		case "permissions", "environment", "evidence", "hashing", "execution", "max_cost", "constraints", "given", "produce", "cost", "approval":
			// semi-structured policy/goal sections; top-level keys already checked
		}
	}
	return nil
}

func validateOpenMapping(node *yaml.Node, path string, allowed map[string]bool) error {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		if isExtensionKey(key) {
			continue
		}
		if !allowed[key] {
			return fmt.Errorf("%s: unknown field %q (use x-* prefix for extensions)", path, key)
		}
	}
	return nil
}

func validateInputOutputBody(node *yaml.Node, path string) error {
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		switch key {
		case "requires":
			if val.Kind == yaml.SequenceNode {
				for _, item := range val.Content {
					if err := validateMapping(item, path+".requires[]", requirementKeys, nil); err != nil {
						return err
					}
				}
			}
		case "assertions":
			if val.Kind == yaml.SequenceNode {
				for _, item := range val.Content {
					if err := validateMapping(item, path+".assertions[]", assertionSpecKeys, nil); err != nil {
						return err
					}
				}
			}
		case "realization", "required_realization":
			if err := validateMapping(val, path+"."+key, realizationKeys, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateStream(node *yaml.Node, path string) error {
	if err := validateMapping(node, path, streamKeys, map[string]map[string]bool{"save_as": saveAsKeys}); err != nil {
		return err
	}
	for i := 0; i < len(node.Content); i += 2 {
		if node.Content[i].Value == "save_as" {
			if err := validateMapping(node.Content[i+1], path+".save_as", saveAsKeys, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func isExtensionKey(key string) bool {
	return strings.HasPrefix(key, "x-")
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
	steps = filterPlanByPolicy(cfg, steps)
	if len(steps) == 0 {
		return nil, errors.New("no plan found under active policy")
	}
	steps = filterPlanByConstraints(cfg, goal, steps)
	if len(steps) == 0 {
		return nil, errors.New("no plan found under goal constraints")
	}
	if err := validatePlanMaxCost(cfg, goal, steps); err != nil {
		return nil, err
	}
	sort.Slice(steps, func(i, j int) bool {
		pi, pj := planStepPriority(cfg, steps[i]), planStepPriority(cfg, steps[j])
		if pi != pj {
			return pi < pj
		}
		return steps[i].Capability < steps[j].Capability
	})
	for i := range steps {
		steps[i].ID = fmt.Sprintf("step-%02d", i+1)
	}
	return steps, nil
}

func planStepPriority(cfg Config, step PlanStep) int {
	capability, ok := cfg.Capabilities[step.Capability]
	if !ok {
		return 100
	}
	for _, out := range capability.Outputs {
		for _, as := range out.Assertions {
			switch as.Predicate {
			case "clean_worktree":
				return 10
			case "applies_cleanly":
				return 30
			case "tests_pass":
				return 50
			}
		}
	}
	for name := range capability.Outputs {
		switch name {
		case "patch":
			return 20
		case "patched_repo":
			return 40
		}
	}
	if capability.Purpose == "observe" {
		return 25
	}
	if capability.Purpose == "derive" {
		return 35
	}
	return 45
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
	saveStreams(ctx, cfg, capability, step, resources, stdout.Bytes(), stderr.Bytes())
	obsData := map[string]interface{}{
		"id": obsID, "source_type": "process_exit", "exit_code": exitCode,
	}
	if policyHashing(cfg, "command_stdout") {
		obsData["stdout_sha256"] = sha(stdout.Bytes())
	}
	if policyHashing(cfg, "command_stderr") {
		obsData["stderr_sha256"] = sha(stderr.Bytes())
	}
	appendEvent(ctx, "observation_recorded", actionID, obsData)
	appendEvent(ctx, "evidence_recorded", actionID, map[string]interface{}{
		"id": evidenceID, "source_type": "process_exit", "observation_id": obsID,
		"confidence_contribution": "observed",
	})
	for outName, out := range capability.Outputs {
		if capability.Command.Stdout.SaveAs.Resource == outName {
			resources[outName] = outName
			realizationData := map[string]interface{}{
				"resource": outName, "artifact": capability.Command.Stdout.SaveAs.ArtifactPath,
				"media_type": capability.Command.Stdout.SaveAs.MediaType,
			}
			if kind := out.Realization.Kind; kind != "" {
				realizationData["kind"] = kind
			} else if capability.Command.Stdout.SaveAs.Resource != "" {
				realizationData["kind"] = "file"
			}
			if policyHashing(cfg, "file_backed_realizations") {
				realizationData["sha256"] = sha(stdout.Bytes())
			}
			appendEvent(ctx, "resource_realization_recorded", actionID, realizationData)
		}
		for _, as := range out.Assertions {
			if assertionMatches(as, exitCode, stdout.String()) {
				subj := resolveSubject(as.Subject, step)
				evidenceSource := as.EvidenceSource
				if evidenceSource == "" {
					evidenceSource = "process_exit"
				}
				confidence := capConfidenceByPolicy(cfg, evidenceSource, as.Confidence)
				rec := AssertionRecord{
					ID:      "as-" + actionID + "-" + safeName(subj+"-"+as.Predicate),
					Subject: subj, Predicate: as.Predicate, Object: as.Object,
					Confidence: confidence, EvidenceID: evidenceID, EvidenceSource: evidenceSource, ActionID: actionID,
				}
				recordAssertion(ctx, actionID, rec)
			}
		}
	}
	if exitCode != 0 {
		appendEvent(ctx, "action_failed", actionID, map[string]interface{}{"exit_code": exitCode})
		msg := fmt.Sprintf("%s failed with exit code %d", step.Capability, exitCode)
		if capability.AlwaysRecordResult {
			msg += " (result recorded)"
		}
		return errors.New(msg)
	}
	appendEvent(ctx, "action_completed", actionID, map[string]interface{}{"exit_code": exitCode})
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

func saveStreams(ctx RunContext, cfg Config, cap Capability, step PlanStep, resources map[string]string, stdout, stderr []byte) {
	if p := cap.Command.Stdout.SaveAs.ArtifactPath; p != "" {
		writeArtifact(ctx, cfg, p, stdout)
	}
	if p := cap.Command.Stdout.SaveAsArtifact; p != "" {
		writeArtifact(ctx, cfg, p, stdout)
	}
	if p := cap.Command.Stderr.SaveAsArtifact; p != "" {
		writeArtifact(ctx, cfg, p, stderr)
	}
}

func writeArtifact(ctx RunContext, cfg Config, rel string, b []byte) {
	path := filepath.Join(ctx.RunDir, rel)
	if !strings.HasPrefix(rel, "artifacts/") {
		path = filepath.Join(ctx.Artifacts, rel)
	}
	_ = os.MkdirAll(filepath.Dir(path), 0755)
	_ = os.WriteFile(path, b, 0644)
	data := map[string]interface{}{"path": path}
	if policyHashing(cfg, "file_backed_realizations") {
		data["sha256"] = sha(b)
	}
	appendEvent(ctx, "artifact_recorded", "", data)
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

func openRun(root, runID string) (RunContext, error) {
	runDir := filepath.Join(root, ".rp", "runs", runID)
	summary := loadRunSummary(runDir)
	configHash, _ := summary["config_hash"].(string)
	policyHash, _ := summary["policy_hash"].(string)
	if configHash == "" {
		return RunContext{}, fmt.Errorf("run %s has no config_hash in summary", runID)
	}
	artifacts := filepath.Join(runDir, "artifacts")
	if err := os.MkdirAll(artifacts, 0755); err != nil {
		return RunContext{}, err
	}
	f, err := os.OpenFile(filepath.Join(runDir, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return RunContext{}, err
	}
	return RunContext{Root: root, RPDir: filepath.Join(root, ".rp"), RunID: runID, RunDir: runDir, Artifacts: artifacts, ConfigHash: configHash, PolicyHash: policyHash, Events: f}, nil
}

func loadRunSummary(runDir string) map[string]interface{} {
	summary := map[string]interface{}{}
	b, err := os.ReadFile(filepath.Join(runDir, "summary.json"))
	if err != nil {
		return summary
	}
	_ = json.Unmarshal(b, &summary)
	return summary
}

func executedCapabilitiesFromEvents(runDir string) map[string]bool {
	events, err := readEvents(runDir)
	if err != nil {
		return map[string]bool{}
	}
	capByAction := map[string]string{}
	executed := map[string]bool{}
	for _, ev := range events {
		if ev.Type == "action_started" {
			if cap, ok := ev.Data["capability"].(string); ok {
				capByAction[ev.ActionID] = cap
			}
		}
		if ev.Type == "action_completed" {
			if cap, ok := capByAction[ev.ActionID]; ok {
				executed[cap] = true
			}
		}
	}
	return executed
}

func recordGoalAttestation(ctx RunContext, cfg Config, goal Goal) {
	assertions, err := assertionsFromRun(ctx.RunDir)
	if err != nil {
		return
	}
	effective := effectiveAssertions(assertions)
	var assertionIDs, evidenceIDs []string
	seenEvidence := map[string]bool{}
	for _, req := range goal.RequiresEvidence {
		for _, as := range effective {
			if as.Subject != req.Subject || as.Predicate != req.Predicate {
				continue
			}
			if !confidenceAtLeast(as.Confidence, req.MinConfidence) {
				continue
			}
			assertionIDs = append(assertionIDs, as.ID)
			if as.EvidenceID != "" && !seenEvidence[as.EvidenceID] {
				evidenceIDs = append(evidenceIDs, as.EvidenceID)
				seenEvidence[as.EvidenceID] = true
			}
			break
		}
	}
	if len(assertionIDs) == 0 {
		return
	}
	appendEvent(ctx, "attestation_recorded", "", map[string]interface{}{
		"id":            "att-goal-" + ctx.RunID,
		"assertion_ids": assertionIDs,
		"evidence_ids":  evidenceIDs,
		"input_hashes":  inputHashesForGoal(ctx.Root, cfg, goal),
		"config_hash":   ctx.ConfigHash,
		"policy_hash":   ctx.PolicyHash,
		"run_id":        ctx.RunID,
	})
}

func newManualRun(root, configHash, policyHash, reason string) (RunContext, error) {
	ctx, err := newRun(root, configHash, policyHash)
	if err != nil {
		return RunContext{}, err
	}
	appendEvent(ctx, "run_started", "", map[string]interface{}{"goal": "manual", "config_hash": configHash, "reason": reason})
	return ctx, nil
}

func appendEvent(ctx RunContext, typ, actionID string, data map[string]interface{}) {
	ev := Event{Type: typ, Time: time.Now().UTC().Format(time.RFC3339Nano), RunID: ctx.RunID, ActionID: actionID, Data: data}
	b, _ := json.Marshal(ev)
	_, _ = ctx.Events.Write(append(b, '\n'))
}

func writeSummary(ctx RunContext, goal string, satisfied bool, reason string, missing []Requirement, produceGaps []string) error {
	summary := map[string]interface{}{"run_id": ctx.RunID, "goal": goal, "satisfied": satisfied, "reason": reason, "config_hash": ctx.ConfigHash, "policy_hash": ctx.PolicyHash}
	if len(missing) > 0 {
		summary["missing_evidence"] = missing
	}
	if len(produceGaps) > 0 {
		summary["missing_produce"] = produceGaps
	}
	b, _ := json.MarshalIndent(summary, "", "  ")
	return os.WriteFile(filepath.Join(ctx.RunDir, "summary.json"), append(b, '\n'), 0644)
}

func goalFromRun(runID string) (string, error) {
	summaryPath := filepath.Join(".rp", "runs", runID, "summary.json")
	b, err := os.ReadFile(summaryPath)
	if err != nil {
		return "", err
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(b, &summary); err != nil {
		return "", err
	}
	goal, _ := summary["goal"].(string)
	if goal == "" || strings.HasPrefix(goal, "manual_") || goal == "manual" {
		return "", errors.New("run summary has no rerunnable goal")
	}
	return goal, nil
}

func savePlan(root, goalName, configHash, policyHash string, steps []PlanStep) (SavedPlan, error) {
	plan := SavedPlan{
		ID:         "plan-" + time.Now().UTC().Format("20060102T150405.000000000Z"),
		Goal:       goalName,
		ConfigHash: configHash,
		PolicyHash: policyHash,
		CreatedAt:  time.Now().UTC().Format(time.RFC3339Nano),
		Steps:      steps,
	}
	dir := filepath.Join(root, ".rp", "cache", "plans")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return SavedPlan{}, err
	}
	b, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return SavedPlan{}, err
	}
	if err := os.WriteFile(filepath.Join(dir, plan.ID+".json"), append(b, '\n'), 0644); err != nil {
		return SavedPlan{}, err
	}
	return plan, nil
}

func loadSavedPlan(root, planID string) (SavedPlan, error) {
	if !strings.HasPrefix(planID, "plan-") {
		return SavedPlan{}, errors.New("plan id must start with plan-")
	}
	path := filepath.Join(root, ".rp", "cache", "plans", planID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return SavedPlan{}, err
	}
	var plan SavedPlan
	if err := json.Unmarshal(b, &plan); err != nil {
		return SavedPlan{}, err
	}
	if plan.ID != planID {
		return SavedPlan{}, fmt.Errorf("saved plan id mismatch: file requested %s but contained %s", planID, plan.ID)
	}
	if plan.Goal == "" || len(plan.Steps) == 0 {
		return SavedPlan{}, errors.New("saved plan is missing goal or steps")
	}
	return plan, nil
}

func goalSatisfied(runDir string, goal Goal, cfg Config) bool {
	return len(missingEvidenceWithConfig(runDir, goal, cfg)) == 0 && len(missingProduce(runDir, goal)) == 0
}

type ProduceRecord struct {
	Resource  string
	Artifact  string
	MediaType string
	Kind      string
}

func realizationsFromRun(runDir string) map[string]ProduceRecord {
	events, err := readEvents(runDir)
	if err != nil {
		return nil
	}
	out := map[string]ProduceRecord{}
	for _, ev := range events {
		if ev.Type != "resource_realization_recorded" {
			continue
		}
		res, _ := ev.Data["resource"].(string)
		if res == "" {
			continue
		}
		rec := ProduceRecord{Resource: res}
		rec.Artifact, _ = ev.Data["artifact"].(string)
		rec.MediaType, _ = ev.Data["media_type"].(string)
		rec.Kind, _ = ev.Data["kind"].(string)
		out[res] = rec
	}
	return out
}

func missingProduce(runDir string, goal Goal) []string {
	if len(goal.Produce) == 0 {
		return nil
	}
	realizations := realizationsFromRun(runDir)
	var gaps []string
	for _, name := range sortedKeys(goal.Produce) {
		spec := goal.Produce[name]
		rec, ok := realizations[name]
		if !ok {
			gaps = append(gaps, name+": not produced")
			continue
		}
		if reason := produceSpecMismatch(runDir, spec, rec); reason != "" {
			gaps = append(gaps, name+": "+reason)
		}
	}
	return gaps
}

func produceSpecMismatch(runDir string, spec OutputSpec, rec ProduceRecord) string {
	req := spec.RequiredRealization
	if req.MediaType == "" && req.Kind == "" {
		return ""
	}
	if req.MediaType != "" && rec.MediaType != "" && req.MediaType != rec.MediaType {
		return fmt.Sprintf("media_type want %s got %s", req.MediaType, rec.MediaType)
	}
	if req.Kind == "file" {
		if rec.Artifact == "" {
			return "missing file artifact"
		}
		path := filepath.Join(runDir, rec.Artifact)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			return "artifact file not found"
		}
	}
	return ""
}

func missingEvidence(runDir string, goal Goal) []Requirement {
	return missingEvidenceWithConfig(runDir, goal, Config{})
}

func missingEvidenceWithConfig(runDir string, goal Goal, cfg Config) []Requirement {
	assertions, err := assertionsFromRun(runDir)
	if err != nil {
		return goal.RequiresEvidence
	}
	effective := effectiveAssertions(assertions)
	var missing []Requirement
	for _, req := range goal.RequiresEvidence {
		ok := false
		for _, as := range effective {
			if as.Subject == req.Subject && as.Predicate == req.Predicate && confidenceAtLeast(as.Confidence, req.MinConfidence) && sourceAllowed(as.EvidenceSource, req.AnySourceType) && sourceMaySatisfyRequiredEvidence(cfg, as.EvidenceSource) {
				ok = true
			}
		}
		if !ok {
			missing = append(missing, req)
		}
	}
	return missing
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

func effectiveAssertions(assertions []AssertionRecord) []AssertionRecord {
	superseded := map[string]bool{}
	for _, as := range assertions {
		if as.Supersedes != "" {
			superseded[as.Supersedes] = true
		}
	}
	var out []AssertionRecord
	for _, as := range assertions {
		if !superseded[as.ID] {
			out = append(out, as)
		}
	}
	return out
}

func recordAssertion(ctx RunContext, actionID string, rec AssertionRecord) {
	assertions, _ := assertionsFromRun(ctx.RunDir)
	if prev := latestAssertion(assertions, rec.Subject, rec.Predicate); prev.ID != "" {
		rec.Supersedes = prev.ID
		appendEvent(ctx, "assertion_superseded", actionID, map[string]interface{}{
			"id": prev.ID, "superseded_by": rec.ID,
		})
	}
	appendEvent(ctx, "assertion_recorded", actionID, structToMap(rec))
}

func latestAssertion(assertions []AssertionRecord, subject, predicate string) AssertionRecord {
	effective := effectiveAssertions(assertions)
	var best AssertionRecord
	for _, as := range effective {
		if as.Subject == subject && as.Predicate == predicate {
			best = as
		}
	}
	return best
}

func autoRepairSettings(cfg Config, flagEnabled bool, flagMax int) (bool, int) {
	pol := activePolicy(cfg)
	maxAttempts := 1
	if raw, ok := pol.Execution["auto_repair"].(map[string]interface{}); ok {
		if enabled, ok := boolField(raw, "enabled"); ok && enabled {
			flagEnabled = true
		}
		if n, ok := intField(raw, "max_attempts"); ok && n > 0 {
			maxAttempts = n
		}
	}
	if flagMax > 0 {
		maxAttempts = flagMax
	}
	return flagEnabled, maxAttempts
}

func intField(m map[string]interface{}, key string) (int, bool) {
	switch v := m[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
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

func appendManualAssertion(subject, predicate, object, confidence, source, note, actionID string) error {
	if !knownConfidence(confidence) {
		return fmt.Errorf("unknown confidence %q", confidence)
	}
	root, cfg, configHash, err := loadProject()
	if err != nil {
		return err
	}
	ctx, err := newManualRun(root, configHash, hashPolicy(cfg), "manual assertion")
	if err != nil {
		return err
	}
	defer ctx.Events.Close()
	actionID = actionID + "-" + safeName(subject+"-"+predicate)
	obsID := "obs-" + actionID
	evidenceID := "ev-" + actionID
	appendEvent(ctx, "action_started", actionID, map[string]interface{}{"manual": true})
	appendEvent(ctx, "observation_recorded", actionID, map[string]interface{}{
		"id": obsID, "source_type": source, "note": note,
	})
	appendEvent(ctx, "evidence_recorded", actionID, map[string]interface{}{
		"id": evidenceID, "source_type": source, "observation_id": obsID,
		"confidence_contribution": confidence,
	})
	rec := AssertionRecord{
		ID: "as-" + actionID, Subject: subject, Predicate: predicate, Object: object,
		Confidence: capConfidenceByPolicy(cfg, source, confidence), EvidenceID: evidenceID, EvidenceSource: source, ActionID: actionID,
	}
	recordAssertion(ctx, actionID, rec)
	appendEvent(ctx, "attestation_recorded", actionID, map[string]interface{}{
		"id": "att-" + actionID, "assertion_ids": []string{rec.ID}, "evidence_ids": []string{evidenceID},
		"config_hash": ctx.ConfigHash, "policy_hash": ctx.PolicyHash, "run_id": ctx.RunID,
	})
	appendEvent(ctx, "action_completed", actionID, map[string]interface{}{"manual": true})
	if err := writeSummary(ctx, "manual_assertion", true, "manual assertion recorded", nil, nil); err != nil {
		return err
	}
	fmt.Println(ctx.RunDir)
	return nil
}

func parseAssertionTarget(expr, subjectOverride string) (string, string, error) {
	subject, predicate, ok := strings.Cut(expr, ".")
	if subjectOverride != "" {
		if ok {
			predicate = predicatePart(predicate)
		} else {
			predicate = expr
		}
		return subjectOverride, predicate, nil
	}
	if ok {
		return subject, predicatePart(predicate), nil
	}
	return expr, "attested", nil
}

func predicatePart(predicate string) string {
	if predicate == "" {
		return "attested"
	}
	return predicate
}

func knownConfidence(confidence string) bool {
	_, ok := confidenceRank[confidence]
	return ok
}

func eventMatches(ev Event, query string) bool {
	if strings.Contains(ev.Type, query) || strings.Contains(ev.ActionID, query) || strings.Contains(ev.RunID, query) {
		return true
	}
	b, _ := json.Marshal(ev.Data)
	return strings.Contains(string(b), query) || strings.Contains(filepath.Base(string(b)), query)
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
				if as.Predicate != predicate {
					continue
				}
				if as.Subject == subject || as.Subject == "" || subject == "" {
					return name
				}
			}
		}
	}
	return ""
}

func filterPlanByPolicy(cfg Config, steps []PlanStep) []PlanStep {
	var out []PlanStep
	for _, step := range steps {
		capability := cfg.Capabilities[step.Capability]
		if err := checkCapabilityPolicy(cfg, capability); err != nil {
			continue
		}
		out = append(out, step)
	}
	return out
}

func filterPlanByConstraints(cfg Config, goal Goal, steps []PlanStep) []PlanStep {
	limits := effectiveMaxCost(cfg, goal)
	var out []PlanStep
	for _, step := range steps {
		capability := cfg.Capabilities[step.Capability]
		if err := checkGoalConstraints(goal, capability); err != nil {
			continue
		}
		if capabilityExceedsMaxCost(capability, limits) {
			continue
		}
		out = append(out, step)
	}
	return out
}

func effectiveMaxCost(cfg Config, goal Goal) map[string]interface{} {
	out := map[string]interface{}{}
	for k, v := range activePolicy(cfg).MaxCost {
		out[k] = v
	}
	if raw, ok := goal.Constraints["max_cost"].(map[string]interface{}); ok {
		for k, v := range raw {
			if existing, ok := out[k]; ok {
				out[k] = minCostLimit(existing, v)
			} else {
				out[k] = v
			}
		}
	}
	return out
}

func minCostLimit(a, b interface{}) interface{} {
	if aStr, ok := a.(string); ok {
		if bStr, ok := b.(string); ok {
			if strings.HasSuffix(aStr, "m") && strings.HasSuffix(bStr, "m") {
				return minCostString(aStr, bStr)
			}
			if humanAttentionRank(aStr) > 0 && humanAttentionRank(bStr) > 0 {
				if humanAttentionRank(bStr) < humanAttentionRank(aStr) {
					return bStr
				}
				return aStr
			}
		}
	}
	if aNum, ok := numericCost(a); ok {
		if bNum, ok := numericCost(b); ok && bNum < aNum {
			return b
		}
	}
	return a
}

func numericCost(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case float64:
		return x, true
	default:
		return 0, false
	}
}

func parseDurationBudget(s string) (time.Duration, bool) {
	if s == "" {
		return 0, false
	}
	if strings.HasSuffix(s, "m") {
		n, err := strconv.Atoi(strings.TrimSuffix(s, "m"))
		if err != nil {
			return 0, false
		}
		return time.Duration(n) * time.Minute, true
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false
	}
	return d, true
}

func capabilityEstimatedMinutes(cap Capability) int {
	switch costString(cap.Cost["time"]) {
	case "cheap":
		return 1
	case "expensive":
		return 8
	default:
		return 2
	}
}

func costString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func humanAttentionRank(v interface{}) int {
	switch costString(v) {
	case "none":
		return 0
	case "approval_if_required":
		return 1
	case "low":
		return 2
	case "medium":
		return 3
	case "high":
		return 4
	default:
		return -1
	}
}

func capabilityExceedsMaxCost(cap Capability, limits map[string]interface{}) bool {
	if len(limits) == 0 {
		return false
	}
	if budget, ok := limits["time"].(string); ok {
		if dur, ok := parseDurationBudget(budget); ok {
			if time.Duration(capabilityEstimatedMinutes(cap))*time.Minute > dur {
				return true
			}
		}
	}
	if budget, ok := numericCost(limits["money_usd"]); ok {
		if capCost, ok := numericCost(cap.Cost["money_usd"]); ok && capCost > budget {
			return true
		}
	}
	if budget, ok := numericCost(limits["tokens"]); ok {
		if capCost, ok := numericCost(cap.Cost["tokens"]); ok && capCost > budget {
			return true
		}
	}
	if budget, ok := limits["human_attention"].(string); ok {
		if humanAttentionRank(cap.Cost["human_attention"]) > humanAttentionRank(budget) {
			return true
		}
	}
	return false
}

func validatePlanMaxCost(cfg Config, goal Goal, plan []PlanStep) error {
	limits := effectiveMaxCost(cfg, goal)
	if len(limits) == 0 {
		return nil
	}
	totalMinutes := 0
	totalMoney := 0.0
	totalTokens := 0.0
	for _, step := range plan {
		cap := cfg.Capabilities[step.Capability]
		if capabilityExceedsMaxCost(cap, limits) {
			return fmt.Errorf("capability %s exceeds max_cost budget", step.Capability)
		}
		totalMinutes += capabilityEstimatedMinutes(cap)
		if v, ok := numericCost(cap.Cost["money_usd"]); ok {
			totalMoney += v
		}
		if v, ok := numericCost(cap.Cost["tokens"]); ok {
			totalTokens += v
		}
	}
	if budget, ok := limits["time"].(string); ok {
		if dur, ok := parseDurationBudget(budget); ok {
			if time.Duration(totalMinutes)*time.Minute > dur {
				return fmt.Errorf("plan estimated time %dm exceeds max_cost budget %s", totalMinutes, budget)
			}
		}
	}
	if budget, ok := numericCost(limits["money_usd"]); ok && totalMoney > budget {
		return fmt.Errorf("plan estimated cost $%.2f exceeds max_cost budget", totalMoney)
	}
	if budget, ok := numericCost(limits["tokens"]); ok && totalTokens > budget {
		return fmt.Errorf("plan estimated tokens %.0f exceeds max_cost budget", totalTokens)
	}
	return nil
}

func checkGoalConstraints(goal Goal, cap Capability) error {
	perms, ok := goal.Constraints["permissions"].(map[string]interface{})
	if !ok {
		return nil
	}
	if val, _ := perms["network"].(string); val == "forbidden" && declaresNetworkEffect(cap) {
		return errors.New("goal constraint forbids network access")
	}
	if val, _ := perms["credentials"].(string); val == "forbidden" && declaresCredentialUse(cap) {
		return errors.New("goal constraint forbids credential use")
	}
	return nil
}

func checkCapabilityPolicy(cfg Config, cap Capability) error {
	policyName := cfg.Defaults["policy"]
	pol := cfg.Policies[policyName]
	processExec := permissionValue(pol.Permissions, "process", "execute")
	if processExec == "forbidden" {
		return errors.New("policy forbids process execution")
	}
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
	if err := checkExternalSideEffectsPolicy(pol, cap); err != nil {
		return err
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
	if usesCredentialRefs(cap) {
		return true
	}
	return strings.Contains(strings.ToLower(cap.Effects.External), "credential")
}

func usesCredentialRefs(cap Capability) bool {
	for _, in := range cap.Inputs {
		if in.Type == "CredentialRef" {
			return true
		}
	}
	return false
}

func declaredExternalSideEffects(cap Capability) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for name, raw := range cap.Effects.ExternalSideEffects {
		if boolFieldValue(raw) {
			add(name)
		}
	}
	ext := strings.ToLower(cap.Effects.External)
	for _, name := range []string{"create_pull_request", "send_message", "deploy"} {
		if strings.Contains(ext, name) {
			add(name)
		}
	}
	sort.Strings(out)
	return out
}

func checkExternalSideEffectsPolicy(pol Policy, cap Capability) error {
	perms, ok := pol.Permissions["external_side_effects"].(map[string]interface{})
	if !ok {
		return nil
	}
	for _, effect := range declaredExternalSideEffects(cap) {
		val, _ := perms[effect].(string)
		if val == "forbidden" {
			return fmt.Errorf("policy forbids external side effect %s", effect)
		}
	}
	return nil
}

func boolFieldValue(v interface{}) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		return x == "true" || x == "yes"
	default:
		return false
	}
}

func failureBehavior(cfg Config) string {
	pol := activePolicy(cfg)
	if behavior, ok := pol.Execution["on_failure"].(string); ok && behavior != "" {
		return behavior
	}
	return "stop_and_suggest"
}

func failureStopData(cfg Config, runID, reason string) (map[string]interface{}, string) {
	data := map[string]interface{}{"reason": reason, "on_failure": failureBehavior(cfg)}
	summary := reason
	if failureBehavior(cfg) == "stop_and_suggest" {
		data["suggest_replan"] = true
		summary += "; try: rp replan " + runID
	}
	return data, summary
}

func capabilityApprovalPermission(cfg Config, cap Capability) string {
	if capabilityNeedsDestructiveApproval(cfg, cap) {
		return "filesystem.destructive_write"
	}
	if capabilityNeedsWriteApproval(cfg, cap) {
		return "filesystem.write"
	}
	return ""
}

func capabilityNeedsApproval(cfg Config, cap Capability) bool {
	return capabilityApprovalPermission(cfg, cap) != ""
}

func capabilityNeedsDestructiveApproval(cfg Config, cap Capability) bool {
	if cap.Idempotence != "non_idempotent" || len(cap.Effects.Filesystem["writes"]) == 0 {
		return false
	}
	return permissionValue(activePolicy(cfg).Permissions, "filesystem", "destructive_write") == "approval_required"
}

func capabilityNeedsWriteApproval(cfg Config, cap Capability) bool {
	if capabilityDeclaresWriteApproval(cap) {
		if permissionValue(activePolicy(cfg).Permissions, "filesystem", "write") == "approval_required" {
			return true
		}
	}
	if len(cap.Effects.Filesystem["writes"]) == 0 {
		return false
	}
	return permissionValue(activePolicy(cfg).Permissions, "filesystem", "write") == "approval_required"
}

func capabilityDeclaresWriteApproval(cap Capability) bool {
	items, ok := cap.Approval["required_if"].([]interface{})
	if !ok {
		return false
	}
	for _, item := range items {
		rule, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		if perm, _ := rule["permission"].(string); perm == "filesystem.write" {
			return true
		}
	}
	return false
}

func planChangeAutoAllowDimensions(cfg Config) []string {
	raw, ok := activePolicy(cfg).Execution["plan_changes"].(map[string]interface{})
	if !ok {
		return nil
	}
	items, ok := raw["allow_without_confirmation_if_not_increasing"].([]interface{})
	if !ok {
		return nil
	}
	var out []string
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func planRevisionNeedsConfirmation(cfg Config, oldCaps, newCaps []string) (bool, string) {
	allowed := planChangeAutoAllowDimensions(cfg)
	if len(allowed) == 0 {
		return false, ""
	}
	var reasons []string
	for _, dim := range []string{"permissions", "risk", "cost_class"} {
		if !containsString(allowed, dim) {
			continue
		}
		if planDimensionScore(cfg, dim, newCaps) > planDimensionScore(cfg, dim, oldCaps) {
			reasons = append(reasons, dim)
		}
	}
	if len(reasons) == 0 {
		return false, ""
	}
	return true, strings.Join(reasons, ", ") + " increased"
}

func planDimensionScore(cfg Config, dim string, capNames []string) int {
	score := 0
	for _, name := range capNames {
		cap := cfg.Capabilities[name]
		switch dim {
		case "permissions":
			if declaresNetworkEffect(cap) || declaresCredentialUse(cap) {
				score = maxInt(score, 3)
			}
			if len(cap.Effects.Filesystem["writes"]) > 0 || capabilityDeclaresWriteApproval(cap) {
				score = maxInt(score, 2)
			}
			if cap.Effects.External != "" {
				score = maxInt(score, 1)
			}
		case "risk":
			score = maxInt(score, rankString(cap.Cost["risk"], map[string]int{"low": 1, "medium": 2, "high": 3}))
		case "cost_class":
			score = maxInt(score, rankString(cap.Cost["time"], map[string]int{"cheap": 1, "moderate": 2, "expensive": 3}))
		}
	}
	return score
}

func rankString(v interface{}, ranks map[string]int) int {
	s, _ := v.(string)
	if r, ok := ranks[s]; ok {
		return r
	}
	return 0
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
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

type PlanEffectSummary struct {
	External            []string `json:"external,omitempty"`
	ExternalSideEffects []string `json:"external_side_effects,omitempty"`
	FilesystemWrites    []string `json:"filesystem_writes,omitempty"`
	NeedsApproval       []string `json:"needs_approval,omitempty"`
}

func summarizePlanEffects(cfg Config, plan []PlanStep) PlanEffectSummary {
	var summary PlanEffectSummary
	seenWrites := map[string]bool{}
	seenExt := map[string]bool{}
	for _, step := range plan {
		capability := cfg.Capabilities[step.Capability]
		for _, w := range capability.Effects.Filesystem["writes"] {
			if w != "" && !seenWrites[w] {
				seenWrites[w] = true
				summary.FilesystemWrites = append(summary.FilesystemWrites, w)
			}
		}
		if capability.Effects.External != "" && !seenExt[capability.Effects.External] {
			seenExt[capability.Effects.External] = true
			summary.External = append(summary.External, capability.Effects.External)
		}
		for _, effect := range declaredExternalSideEffects(capability) {
			summary.ExternalSideEffects = append(summary.ExternalSideEffects, effect)
		}
		if capabilityNeedsApproval(cfg, capability) {
			summary.NeedsApproval = append(summary.NeedsApproval, step.Capability)
		}
	}
	sort.Strings(summary.FilesystemWrites)
	sort.Strings(summary.External)
	sort.Strings(summary.ExternalSideEffects)
	sort.Strings(summary.NeedsApproval)
	summary.ExternalSideEffects = dedupeStrings(summary.ExternalSideEffects)
	return summary
}

func dedupeStrings(items []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func printEffectSummary(cfg Config, plan []PlanStep) {
	summary := summarizePlanEffects(cfg, plan)
	if len(summary.FilesystemWrites) == 0 && len(summary.External) == 0 && len(summary.ExternalSideEffects) == 0 && len(summary.NeedsApproval) == 0 {
		return
	}
	fmt.Println("\nEffect summary:")
	if len(summary.External) > 0 {
		fmt.Printf("  external: %s\n", strings.Join(summary.External, ", "))
	}
	if len(summary.ExternalSideEffects) > 0 {
		fmt.Printf("  external side effects: %s\n", strings.Join(summary.ExternalSideEffects, ", "))
	}
	if len(summary.FilesystemWrites) > 0 {
		fmt.Println("  filesystem writes:")
		for _, w := range summary.FilesystemWrites {
			fmt.Printf("    - %s\n", w)
		}
	}
	if len(summary.NeedsApproval) > 0 {
		fmt.Printf("  approval required: %s\n", strings.Join(summary.NeedsApproval, ", "))
	}
}

func planAssumptions(cfg Config, plan []PlanStep) []string {
	var out []string
	seen := map[string]bool{}
	for _, step := range plan {
		capability := cfg.Capabilities[step.Capability]
		for inputName, input := range capability.Inputs {
			for _, req := range input.Requires {
				subject := req.Subject
				if subject == "" {
					subject = step.Inputs[inputName]
				}
				if subject == "" {
					subject = inputName
				}
				line := fmt.Sprintf("%s.%s>=%s (required by %s)", subject, req.Predicate, req.MinConfidence, step.Capability)
				if !seen[line] {
					seen[line] = true
					out = append(out, line)
				}
			}
		}
		for _, req := range capability.Preconditions {
			subject := req.Subject
			if subject == "" {
				subject = step.Inputs["repo"]
			}
			line := fmt.Sprintf("%s.%s>=%s (required by %s)", subject, req.Predicate, req.MinConfidence, step.Capability)
			if !seen[line] {
				seen[line] = true
				out = append(out, line)
			}
		}
	}
	sort.Strings(out)
	return out
}

func printPlanAssumptions(cfg Config, plan []PlanStep) {
	assumptions := planAssumptions(cfg, plan)
	if len(assumptions) == 0 {
		return
	}
	fmt.Println("\nAssumed preconditions (speculative):")
	for _, line := range assumptions {
		fmt.Printf("  - %s\n", line)
	}
}

func policyHashing(cfg Config, key string) bool {
	pol := activePolicy(cfg)
	if pol.Hashing == nil {
		return false
	}
	v, ok := pol.Hashing[key]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
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
	b, err = json.Marshal(canonicalize(normalized))
	if err != nil {
		return "", err
	}
	return sha(b), nil
}

func canonicalize(v interface{}) interface{} {
	switch x := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(x))
		for _, k := range sortedKeys(x) {
			out[k] = canonicalize(x[k])
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(x))
		for i, item := range x {
			out[i] = canonicalize(item)
		}
		return out
	default:
		return v
	}
}

func hashPolicy(cfg Config) string {
	name := cfg.Defaults["policy"]
	b, _ := json.Marshal(cfg.Policies[name])
	return sha(b)
}

func activePolicy(cfg Config) Policy {
	return cfg.Policies[cfg.Defaults["policy"]]
}

func capConfidenceByPolicy(cfg Config, sourceType, confidence string) string {
	maxConfidence := policySourceMaxConfidence(activePolicy(cfg), sourceType)
	if maxConfidence == "" || confidenceAtLeast(maxConfidence, confidence) {
		return confidence
	}
	return maxConfidence
}

func policySourceMaxConfidence(pol Policy, sourceType string) string {
	for _, rule := range policyEvidenceRules(pol, "source_limits") {
		if stringField(rule, "source_type") == sourceType {
			if max := stringField(rule, "max_confidence"); knownConfidence(max) {
				return max
			}
		}
	}
	return ""
}

func sourceMaySatisfyRequiredEvidence(cfg Config, sourceType string) bool {
	pol := activePolicy(cfg)
	for _, rule := range policyEvidenceRules(pol, "final_goal_rules") {
		if stringField(rule, "source_type") == sourceType {
			if may, ok := boolField(rule, "may_satisfy_required_evidence"); ok {
				return may
			}
		}
	}
	return true
}

func policyEvidenceRules(pol Policy, key string) []map[string]interface{} {
	raw, ok := pol.Evidence[key]
	if !ok {
		return nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	var rules []map[string]interface{}
	for _, item := range items {
		if rule, ok := item.(map[string]interface{}); ok {
			rules = append(rules, rule)
		}
	}
	return rules
}

func stringField(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

func boolField(m map[string]interface{}, key string) (bool, bool) {
	v, ok := m[key].(bool)
	return v, ok
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

func inputHashesForGoal(root string, cfg Config, goal Goal) map[string]string {
	hashes := map[string]string{}
	for inputName, resName := range goal.Given {
		res, ok := cfg.Resources[resName]
		if !ok {
			continue
		}
		if len(res.Realizations) > 0 && res.Realizations[0].Hash != "" {
			hashes[inputName] = res.Realizations[0].Hash
			continue
		}
		path := pathFromURI(root, realizationURI(res))
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		b, err := os.ReadFile(path)
		if err == nil {
			hashes[inputName] = sha(b)
		}
	}
	return hashes
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

func shortHash(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
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
