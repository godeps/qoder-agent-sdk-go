package protocol

import "encoding/json"

// HookEvent enumerates the hook lifecycle events.
type HookEvent string

const (
	HookPreToolUse         HookEvent = "PreToolUse"
	HookPostToolUse        HookEvent = "PostToolUse"
	HookPostToolUseFailure HookEvent = "PostToolUseFailure"
	HookUserPromptSubmit   HookEvent = "UserPromptSubmit"
	HookSessionStart       HookEvent = "SessionStart"
	HookSessionEnd         HookEvent = "SessionEnd"
	HookStop               HookEvent = "Stop"
	HookStopFailure        HookEvent = "StopFailure"
	HookSubagentStart      HookEvent = "SubagentStart"
	HookSubagentStop       HookEvent = "SubagentStop"
	HookPreCompact         HookEvent = "PreCompact"
	HookPostCompact        HookEvent = "PostCompact"
	HookPermissionRequest  HookEvent = "PermissionRequest"
	HookPermissionDenied   HookEvent = "PermissionDenied"
	HookSetup              HookEvent = "Setup"
	HookTeammateIdle       HookEvent = "TeammateIdle"
	HookTaskCreated        HookEvent = "TaskCreated"
	HookTaskCompleted      HookEvent = "TaskCompleted"
	HookElicitation        HookEvent = "Elicitation"
	HookElicitationResult  HookEvent = "ElicitationResult"
	HookConfigChange       HookEvent = "ConfigChange"
	HookWorktreeCreate     HookEvent = "WorktreeCreate"
	HookWorktreeRemove     HookEvent = "WorktreeRemove"
	HookInstructionsLoaded HookEvent = "InstructionsLoaded"
	HookCwdChanged         HookEvent = "CwdChanged"
	HookFileChanged        HookEvent = "FileChanged"
)

// HookEvents is the full set of hook events (matches the TS HOOK_EVENTS export).
var HookEvents = []HookEvent{
	HookPreToolUse, HookPostToolUse, HookPostToolUseFailure, HookUserPromptSubmit,
	HookSessionStart, HookSessionEnd, HookStop, HookSubagentStart, HookSubagentStop,
	HookPreCompact, HookPostCompact, HookPermissionRequest, HookPermissionDenied,
	HookSetup, HookTeammateIdle, HookTaskCreated, HookTaskCompleted, HookElicitation,
	HookElicitationResult, HookConfigChange, HookWorktreeCreate, HookWorktreeRemove,
	HookInstructionsLoaded, HookCwdChanged, HookFileChanged,
}

// BaseHookInput is the common prefix of every hook input payload.
type BaseHookInput struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
	CWD            string `json:"cwd"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
	AgentID        string `json:"agent_id,omitempty"`
	AgentType      string `json:"agent_type,omitempty"`
}

// HookInput is the wire payload the CLI sends to a host hook callback. It is a
// flattened union: HookEventName selects which fields are meaningful. Unknown
// or event-specific fields beyond the common set are ignored on decode.
type HookInput struct {
	BaseHookInput
	HookEventName         HookEvent          `json:"hook_event_name"`
	ToolName              string             `json:"tool_name,omitempty"`
	ToolInput             json.RawMessage    `json:"tool_input,omitempty"`
	ToolResponse          json.RawMessage    `json:"tool_response,omitempty"`
	ToolUseID             string             `json:"tool_use_id,omitempty"`
	Error                 string             `json:"error,omitempty"`
	IsInterrupt           *bool              `json:"is_interrupt,omitempty"`
	Prompt                string             `json:"prompt,omitempty"`
	ImageURLs             []string           `json:"image_urls,omitempty"`
	Source                string             `json:"source,omitempty"` // startup|resume|clear|compact ; init|maintenance
	Reason                ExitReason         `json:"reason,omitempty"`
	StopHookActive        *bool              `json:"stop_hook_active,omitempty"`
	LastAssistantMessage  string             `json:"last_assistant_message,omitempty"`
	Trigger               string             `json:"trigger,omitempty"` // manual|auto
	CustomInstructions    *string            `json:"custom_instructions,omitempty"`
	CompactSummary        string             `json:"compact_summary,omitempty"`
	PermissionSuggestions []PermissionUpdate `json:"permission_suggestions,omitempty"`
	DenialReason          string             `json:"denial_reason,omitempty"`
	AgentTranscriptPath   string             `json:"agent_transcript_path,omitempty"`
	TaskID                string             `json:"task_id,omitempty"`
	TaskSubject           string             `json:"task_subject,omitempty"`
	TaskDescription       string             `json:"task_description,omitempty"`
	TeammateName          string             `json:"teammate_name,omitempty"`
	TeamName              string             `json:"team_name,omitempty"`
}

// HookJSONOutput is the structured result a host returns from a hook callback.
type HookJSONOutput struct {
	Continue          *bool  `json:"continue,omitempty"`
	StopReason        string `json:"stopReason,omitempty"`
	SuppressOutput    *bool  `json:"suppressOutput,omitempty"`
	Decision          string `json:"decision,omitempty"` // allow|block|ask
	Reason            string `json:"reason,omitempty"`
	AdditionalContext string `json:"additionalContext,omitempty"`
}
