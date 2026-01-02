package crd

import "time"

// APIVersion and Kind constants
const (
	APIVersion = "xgent.ai/v1"
)

// ResourceKind represents CRD resource types
type ResourceKind string

const (
	KindSoul          ResourceKind = "Soul"
	KindMind          ResourceKind = "Mind"
	KindCraft         ResourceKind = "Craft"
	KindRobot         ResourceKind = "Robot"
	KindTeam          ResourceKind = "Team"
	KindCollaboration ResourceKind = "Collaboration"
)

// Resource is the base interface for all CRD resources
type Resource interface {
	GetKind() ResourceKind
	GetMetadata() Metadata
	Validate() error
}

// Metadata contains resource metadata
type Metadata struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at,omitempty" json:"created_at,omitempty"`
	UpdatedAt   time.Time         `yaml:"updated_at,omitempty" json:"updated_at,omitempty"`
}

// Soul represents an agent's personality and behavior (essence)
type Soul struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind `yaml:"kind" json:"kind"`
	Metadata   Metadata     `yaml:"metadata" json:"metadata"`
	Spec       SoulSpec     `yaml:"spec" json:"spec"`
}

type SoulSpec struct {
	Personality  string   `yaml:"personality" json:"personality"`
	Capabilities []string `yaml:"capabilities,omitempty" json:"capabilities,omitempty"`
	Constraints  []string `yaml:"constraints,omitempty" json:"constraints,omitempty"`
	Examples     []string `yaml:"examples,omitempty" json:"examples,omitempty"`
}

func (s *Soul) GetKind() ResourceKind { return KindSoul }
func (s *Soul) GetMetadata() Metadata { return s.Metadata }
func (s *Soul) Validate() error {
	if s.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	if s.Spec.Personality == "" {
		return ErrInvalidSpec
	}
	return nil
}

// Mind represents LLM model configuration (cognitive ability)
type Mind struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind `yaml:"kind" json:"kind"`
	Metadata   Metadata     `yaml:"metadata" json:"metadata"`
	Spec       MindSpec     `yaml:"spec" json:"spec"`
}

type MindSpec struct {
	Provider    string            `yaml:"provider" json:"provider"` // openai, anthropic, custom
	ModelID     string            `yaml:"model_id" json:"model_id"`
	APIKey      string            `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	BaseURL     string            `yaml:"base_url,omitempty" json:"base_url,omitempty"`
	Temperature float32           `yaml:"temperature,omitempty" json:"temperature,omitempty"`
	MaxTokens   int               `yaml:"max_tokens,omitempty" json:"max_tokens,omitempty"`
	Headers     map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
}

func (m *Mind) GetKind() ResourceKind { return KindMind }
func (m *Mind) GetMetadata() Metadata { return m.Metadata }
func (m *Mind) Validate() error {
	if m.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	if m.Spec.Provider == "" || m.Spec.ModelID == "" {
		return ErrInvalidSpec
	}
	return nil
}

// Craft represents an agent's tools and capabilities (skills)
type Craft struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind `yaml:"kind" json:"kind"`
	Metadata   Metadata     `yaml:"metadata" json:"metadata"`
	Spec       CraftSpec    `yaml:"spec" json:"spec"`
}

type CraftSpec struct {
	Tools        []ToolConfig      `yaml:"tools,omitempty" json:"tools,omitempty"`
	Instructions string            `yaml:"instructions,omitempty" json:"instructions,omitempty"`
	Environment  map[string]string `yaml:"environment,omitempty" json:"environment,omitempty"`
	MCP          *MCPConfig        `yaml:"mcp,omitempty" json:"mcp,omitempty"`
}

type ToolConfig struct {
	Name    string                 `yaml:"name" json:"name"`
	Type    string                 `yaml:"type" json:"type"` // builtin, custom, mcp
	Config  map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	Enabled bool                   `yaml:"enabled" json:"enabled"`
}

type MCPConfig struct {
	Servers []MCPServer `yaml:"servers,omitempty" json:"servers,omitempty"`
}

type MCPServer struct {
	Name    string            `yaml:"name" json:"name"`
	Command string            `yaml:"command" json:"command"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

func (c *Craft) GetKind() ResourceKind { return KindCraft }
func (c *Craft) GetMetadata() Metadata { return c.Metadata }
func (c *Craft) Validate() error {
	if c.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	return nil
}

// Robot represents an agent instance (Soul + Mind + Craft)
type Robot struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind `yaml:"kind" json:"kind"`
	Metadata   Metadata     `yaml:"metadata" json:"metadata"`
	Spec       RobotSpec    `yaml:"spec" json:"spec"`
}

type RobotSpec struct {
	Soul       string `yaml:"soul" json:"soul"`                       // Reference to Soul resource
	Mind       string `yaml:"mind" json:"mind"`                       // Reference to Mind resource
	Craft      string `yaml:"craft,omitempty" json:"craft,omitempty"` // Reference to Craft resource
	SessionID  string `yaml:"session_id,omitempty" json:"session_id,omitempty"`
	MaxHistory int    `yaml:"max_history,omitempty" json:"max_history,omitempty"`
}

func (r *Robot) GetKind() ResourceKind { return KindRobot }
func (r *Robot) GetMetadata() Metadata { return r.Metadata }
func (r *Robot) Validate() error {
	if r.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	if r.Spec.Soul == "" || r.Spec.Mind == "" {
		return ErrInvalidSpec
	}
	return nil
}

// Team represents a collaborative team of robots
type Team struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind `yaml:"kind" json:"kind"`
	Metadata   Metadata     `yaml:"metadata" json:"metadata"`
	Spec       TeamSpec     `yaml:"spec" json:"spec"`
}

type TeamSpec struct {
	Leader      string            `yaml:"leader,omitempty" json:"leader,omitempty"` // Reference to Robot
	Members     []string          `yaml:"members" json:"members"`                   // References to Robots
	Mode        CollaborationMode `yaml:"mode" json:"mode"`
	Craft       string            `yaml:"craft,omitempty" json:"craft,omitempty"` // Shared craft
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
}

type CollaborationMode string

const (
	ModeCoordinate  CollaborationMode = "coordinate"
	ModeCollaborate CollaborationMode = "collaborate"
	ModeRoute       CollaborationMode = "route"
)

func (t *Team) GetKind() ResourceKind { return KindTeam }
func (t *Team) GetMetadata() Metadata { return t.Metadata }
func (t *Team) Validate() error {
	if t.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	if len(t.Spec.Members) == 0 {
		return ErrInvalidSpec
	}
	return nil
}

// Collaboration defines custom collaboration patterns
type Collaboration struct {
	APIVersion string            `yaml:"apiVersion" json:"apiVersion"`
	Kind       ResourceKind      `yaml:"kind" json:"kind"`
	Metadata   Metadata          `yaml:"metadata" json:"metadata"`
	Spec       CollaborationSpec `yaml:"spec" json:"spec"`
}

type CollaborationSpec struct {
	Type       string                 `yaml:"type" json:"type"` // sequential, parallel, conditional
	Steps      []CollaborationStep    `yaml:"steps" json:"steps"`
	Conditions map[string]interface{} `yaml:"conditions,omitempty" json:"conditions,omitempty"`
}

type CollaborationStep struct {
	Name      string   `yaml:"name" json:"name"`
	Agent     string   `yaml:"agent" json:"agent"`
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Condition string   `yaml:"condition,omitempty" json:"condition,omitempty"`
}

func (c *Collaboration) GetKind() ResourceKind { return KindCollaboration }
func (c *Collaboration) GetMetadata() Metadata { return c.Metadata }
func (c *Collaboration) Validate() error {
	if c.Metadata.Name == "" {
		return ErrInvalidMetadata
	}
	return nil
}

// Errors
var (
	ErrInvalidMetadata = &ValidationError{Message: "invalid metadata"}
	ErrInvalidSpec     = &ValidationError{Message: "invalid spec"}
)

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
