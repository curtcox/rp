// Package model defines the core types and helpers shared across the rp
// planner: resources, capabilities, goals, plans, events, and run state.
package model

import "os"

const Version = "rp.dev/v0.1"

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

type ProduceRecord struct {
	Resource  string
	Artifact  string
	MediaType string
	Kind      string
}

type PlanEffectSummary struct {
	External            []string `json:"external,omitempty"`
	ExternalSideEffects []string `json:"external_side_effects,omitempty"`
	FilesystemWrites    []string `json:"filesystem_writes,omitempty"`
	NeedsApproval       []string `json:"needs_approval,omitempty"`
}
