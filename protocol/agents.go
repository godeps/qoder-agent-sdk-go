package protocol

import "encoding/json"

// AgentMemoryScope is the persistent-memory scope for an agent.
type AgentMemoryScope string

const (
	AgentMemoryUser    AgentMemoryScope = "user"
	AgentMemoryProject AgentMemoryScope = "project"
	AgentMemoryLocal   AgentMemoryScope = "local"
)

// AgentMcpServerSpec is either a server name (string) or a map of name→config.
// It is preserved as raw JSON because the two shapes are disjoint on the wire.
type AgentMcpServerSpec json.RawMessage

// AgentDefinition defines a reusable subagent.
type AgentDefinition struct {
	Description     string               `json:"description"`
	Tools           []string             `json:"tools,omitempty"`
	DisallowedTools []string             `json:"disallowedTools,omitempty"`
	Prompt          string               `json:"prompt"`
	Model           string               `json:"model,omitempty"`
	McpServers      []AgentMcpServerSpec `json:"mcpServers,omitempty"`
	Skills          []string             `json:"skills,omitempty"`
	InitialPrompt   string               `json:"initialPrompt,omitempty"`
	MaxTurns        *int                 `json:"maxTurns,omitempty"`
	Effort          EffortLevel          `json:"effort,omitempty"`
	PermissionMode  PermissionMode       `json:"permissionMode,omitempty"`
	Background      *bool                `json:"background,omitempty"`
	Memory          AgentMemoryScope     `json:"memory,omitempty"`
	Isolation       string               `json:"isolation,omitempty"` // "worktree"
}

// AgentInfo is the CLI's report of a registered agent.
type AgentInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Model       string `json:"model,omitempty"`
}
