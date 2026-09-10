package protocol

import (
	"encoding/json"
	"fmt"
)

// Message is a decoded JSONL line from qoderclicn stdout. Each concrete type
// reports its wire "type" discriminant via Type().
type Message interface {
	MessageType() string
}

// ParseMessage decodes one JSONL line into the matching concrete Message type.
// Unknown types yield a *UnknownMessage that preserves the raw bytes.
func ParseMessage(line []byte) (Message, error) {
	var probe struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(line, &probe); err != nil {
		return nil, fmt.Errorf("qoder: decode message: %w", err)
	}
	switch probe.Type {
	case "assistant":
		return decode[AssistantMessage](line)
	case "user":
		return decode[UserMessage](line)
	case "result":
		return decode[ResultMessage](line)
	case "system":
		return decode[SystemMessage](line)
	case "stream_event":
		return decode[PartialAssistantMessage](line)
	case "command_lifecycle":
		return decode[CommandLifecycleMessage](line)
	case "prompt_suggestion":
		return decode[PromptSuggestionMessage](line)
	case "cloud_agent_event":
		return decode[CloudAgentEventMessage](line)
	case "control_request":
		return decode[ControlRequest](line)
	case "control_response":
		return decode[ControlResponse](line)
	case "control_cancel", "control_cancel_request":
		return decode[ControlCancel](line)
	case "keep_alive":
		return decode[KeepAlive](line)
	case "transcript_mirror":
		return decode[TranscriptMirror](line)
	default:
		return &UnknownMessage{typeStr: probe.Type, Raw: append([]byte(nil), line...)}, nil
	}
}

func decode[T Message](line []byte) (*T, error) {
	var m T
	if err := json.Unmarshal(line, &m); err != nil {
		return nil, fmt.Errorf("qoder: decode %s: %w", probeType(line), err)
	}
	return &m, nil
}

func probeType(line []byte) string {
	var probe struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(line, &probe)
	return probe.Type
}

// UnknownMessage preserves an unrecognized JSONL line verbatim.
type UnknownMessage struct {
	typeStr string
	Raw     json.RawMessage
}

func (m UnknownMessage) MessageType() string { return m.typeStr }

// --- Agent messages ---

// AssistantMessage is a model response turn.
type AssistantMessage struct {
	Type              string                    `json:"type"` // "assistant"
	Message           BetaMessage               `json:"message"`
	ParentToolUseID   *string                   `json:"parent_tool_use_id"`
	IsAPIErrorMessage bool                      `json:"isApiErrorMessage,omitempty"`
	RequestID         string                    `json:"request_id,omitempty"`
	Error             *SDKAssistantMessageError `json:"error,omitempty"`
	Aborted           bool                      `json:"aborted,omitempty"`
	UUID              UUID                      `json:"uuid"`
	SessionID         string                    `json:"session_id"`
}

func (m AssistantMessage) MessageType() string { return "assistant" }

// SDKToolNonExecutionKind labels a tool result that did not run to completion.
type SDKToolNonExecutionKind string

// SDKToolResultMeta is display-only metadata for a non-executed tool result.
type SDKToolResultMeta struct {
	ID               string                  `json:"id"`
	NonExecutionKind SDKToolNonExecutionKind `json:"non_execution_kind"`
	UserFeedback     string                  `json:"user_feedback,omitempty"`
}

// SDKMessageOrigin identifies what triggered a message.
type SDKMessageOrigin struct {
	Kind         string `json:"kind"` // task-notification | peer | human | bridge
	From         string `json:"from,omitempty"`
	SenderTaskID string `json:"senderTaskId,omitempty"`
	Name         string `json:"name,omitempty"`
	Body         string `json:"body,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
}

// SDKFileAttachment references a persisted file.
type SDKFileAttachment struct {
	FileID       string `json:"file_id"`
	RelativePath string `json:"relative_path"`
}

// UserMessage is a user-side message (input or replay).
type UserMessage struct {
	Type            string              `json:"type"` // "user"
	Message         MessageParam        `json:"message"`
	ParentToolUseID *string             `json:"parent_tool_use_id"`
	CustomContext   map[string]string   `json:"custom_context,omitempty"`
	ClientComposed  bool                `json:"client_composed,omitempty"`
	FileAttachments []SDKFileAttachment `json:"file_attachments,omitempty"`
	IsSynthetic     bool                `json:"isSynthetic,omitempty"`
	Origin          *SDKMessageOrigin   `json:"origin,omitempty"`
	ToolUseResult   json.RawMessage     `json:"tool_use_result,omitempty"`
	ToolResultMeta  []SDKToolResultMeta `json:"tool_result_meta,omitempty"`
	Priority        string              `json:"priority,omitempty"` // now|next|later
	ShouldQuery     *bool               `json:"shouldQuery,omitempty"`
	Timestamp       string              `json:"timestamp,omitempty"`
	UUID            UUID                `json:"uuid"`
	SessionID       string              `json:"session_id"`
	IsReplay        bool                `json:"isReplay,omitempty"`
}

func (m UserMessage) MessageType() string { return "user" }

// SDKPermissionDenial records a denied tool call.
type SDKPermissionDenial struct {
	ToolName           string                     `json:"tool_name"`
	ToolUseID          string                     `json:"tool_use_id"`
	ToolInput          map[string]json.RawMessage `json:"tool_input"`
	Message            string                     `json:"message,omitempty"`
	DecisionReasonType string                     `json:"decision_reason_type,omitempty"`
	DecisionReason     string                     `json:"decision_reason,omitempty"`
}

// ResultMessage is the terminal result of a task round (success or error).
type ResultMessage struct {
	Type              string                `json:"type"`    // "result"
	Subtype           string                `json:"subtype"` // success | error_during_execution | error_max_turns | error_max_budget_usd
	DurationMs        int                   `json:"duration_ms"`
	DurationAPIMs     int                   `json:"duration_api_ms"`
	IsError           bool                  `json:"is_error"`
	NumTurns          int                   `json:"num_turns"`
	Result            string                `json:"result,omitempty"` // success only
	StopReason        *string               `json:"stop_reason"`
	TotalCostUSD      float64               `json:"total_cost_usd"`
	TotalCredits      *float64              `json:"total_credits,omitempty"`
	Usage             NonNullableUsage      `json:"usage"`
	ModelUsage        map[string]ModelUsage `json:"modelUsage"`
	PermissionDenials []SDKPermissionDenial `json:"permission_denials"`
	Errors            []string              `json:"errors,omitempty"` // error only
	ErrorCode         *int                  `json:"error_code,omitempty"`
	TerminalReason    *string               `json:"terminal_reason,omitempty"`
	FastModeState     *FastModeState        `json:"fast_mode_state,omitempty"`
	Origin            *SDKMessageOrigin     `json:"origin,omitempty"`
	UUID              UUID                  `json:"uuid"`
	SessionID         string                `json:"session_id"`
}

func (m ResultMessage) MessageType() string { return "result" }

// IsSuccess reports whether this is a success result.
func (m ResultMessage) IsSuccess() bool { return m.Subtype == "success" }

// SDKStatus is a transient session status (compacting or null).
type SDKStatus string

// SystemMessage is a "system"-typed message, discriminated by Subtype. It is a
// flattened union: set Subtype selects the meaningful fields. Fields not
// relevant to the active subtype are ignored. Unknown subtypes are preserved
// in Raw for forward compatibility.
type SystemMessage struct {
	Type    string          `json:"type"` // "system"
	Subtype string          `json:"subtype"`
	Raw     json.RawMessage `json:"-"`

	// init
	Agents          []string      `json:"agents,omitempty"`
	APIKeySource    *ApiKeySource `json:"apiKeySource,omitempty"`
	QodercliVersion string        `json:"qodercli_version,omitempty"`
	ProtocolVersion string        `json:"protocol_version,omitempty"`
	CWD             string        `json:"cwd,omitempty"`
	Tools           []string      `json:"tools,omitempty"`
	McpServers      []struct {
		Name   string `json:"name"`
		Status string `json:"status"`
	} `json:"mcp_servers,omitempty"`
	Model          string         `json:"model,omitempty"`
	PermissionMode PermissionMode `json:"permissionMode,omitempty"`
	SlashCommands  []SlashCommand `json:"slash_commands,omitempty"`
	OutputStyle    string         `json:"output_style,omitempty"`
	Skills         []string       `json:"skills,omitempty"`
	Plugins        []PluginInfo   `json:"plugins,omitempty"`
	Capabilities   []string       `json:"capabilities,omitempty"`
	FastModeState  *FastModeState `json:"fast_mode_state,omitempty"`

	// status / task_*
	Status string `json:"status,omitempty"` // system/status: compacting|null; task_*: completed|failed|stopped|pending|running|...

	// api_retry
	Attempt      int                       `json:"attempt,omitempty"`
	MaxRetries   int                       `json:"max_retries,omitempty"`
	RetryDelayMs int                       `json:"retry_delay_ms,omitempty"`
	ErrorStatus  *int                      `json:"error_status,omitempty"`
	SystemError  *SDKAssistantMessageError `json:"error,omitempty"`

	// model_queue_status / control_request_progress
	RequestID          string `json:"request_id,omitempty"`
	RequestSetID       string `json:"request_set_id,omitempty"`
	ModelKey           string `json:"model_key,omitempty"`
	QueueType          string `json:"queue_type,omitempty"`
	QueueCount         *int   `json:"queue_count,omitempty"`
	WaitTimeMs         *int   `json:"wait_time_ms,omitempty"`
	QueueWaitElapsedMs *int   `json:"queue_wait_elapsed_ms,omitempty"`
	QueueMaxWaitMs     *int   `json:"queue_max_wait_ms,omitempty"`
	ServiceAvailable   *bool  `json:"service_available,omitempty"`

	// hook_*
	HookID       string `json:"hook_id,omitempty"`
	HookName     string `json:"hook_name,omitempty"`
	HookEvent    string `json:"hook_event,omitempty"`
	HookStdout   string `json:"stdout,omitempty"`
	HookStderr   string `json:"stderr,omitempty"`
	HookOutput   string `json:"output,omitempty"`
	HookExitCode *int   `json:"exit_code,omitempty"`
	HookOutcome  string `json:"outcome,omitempty"` // success|error|cancelled

	// task_*
	TaskID          string `json:"task_id,omitempty"`
	TaskOutputFile  string `json:"output_file,omitempty"`
	TaskSummary     string `json:"summary,omitempty"`
	TaskDescription string `json:"description,omitempty"`
	SubagentType    string `json:"subagent_type,omitempty"`
	TaskType        string `json:"task_type,omitempty"`
	WorkflowName    string `json:"workflow_name,omitempty"`
	TaskPrompt      string `json:"prompt,omitempty"`
	TaskUsage       *struct {
		TotalTokens *int `json:"total_tokens,omitempty"`
		ToolUses    int  `json:"tool_uses"`
		DurationMs  int  `json:"duration_ms"`
	} `json:"usage,omitempty"`
	LastToolName string `json:"last_tool_name,omitempty"`
	TaskPatch    *struct {
		Status         string `json:"status,omitempty"`
		Description    string `json:"description,omitempty"`
		EndTime        *int   `json:"end_time,omitempty"`
		TotalPausedMs  *int   `json:"total_paused_ms,omitempty"`
		Error          string `json:"error,omitempty"`
		IsBackgrounded *bool  `json:"is_backgrounded,omitempty"`
	} `json:"patch,omitempty"`
	BackgroundTasks []struct {
		TaskID      string `json:"task_id"`
		TaskType    string `json:"task_type"`
		Description string `json:"description"`
	} `json:"tasks,omitempty"`

	// session_state_changed
	SessionState string `json:"state,omitempty"` // idle|running|requires_action

	// plan_mode_changed / goal_*
	PlanMode   *PlanModeSnapshot `json:"plan_mode,omitempty"`
	Goal       *GoalSnapshot     `json:"goal,omitempty"`
	GoalID     string            `json:"goal_id,omitempty"`
	GoalReason string            `json:"reason,omitempty"`

	// session_title_changed
	Title       string `json:"title,omitempty"`
	TitleSource string `json:"source,omitempty"` // ai|custom
	Revision    *int   `json:"revision,omitempty"`

	// files_persisted
	FilesPersisted []struct {
		Filename string `json:"filename"`
		FileID   string `json:"file_id"`
	} `json:"files,omitempty"`
	FilesFailed []struct {
		Filename string `json:"filename"`
		Error    string `json:"error"`
	} `json:"failed,omitempty"`
	ProcessedAt string `json:"processed_at,omitempty"`

	// elicitation_complete / permission_denied
	McpServerName     string `json:"mcp_server_name,omitempty"`
	ElicitationID     string `json:"elicitation_id,omitempty"`
	PermDeniedTool    string `json:"tool_name,omitempty"`
	ToolUseID         string `json:"tool_use_id,omitempty"`
	PermDeniedMessage string `json:"message,omitempty"`
	AgentID           string `json:"agent_id,omitempty"`

	// artifacts_update / available_models_update / commands_changed
	Artifacts    []SDKArtifactInfo `json:"artifacts,omitempty"`
	Models       []ModelInfo       `json:"models,omitempty"`
	CurrentModel string            `json:"currentModel,omitempty"`
	Commands     []SlashCommand    `json:"commands,omitempty"`

	// memory_generation / memory_consumption / skill_evolution
	MemoryResult json.RawMessage `json:"result,omitempty"`

	UUID      UUID   `json:"uuid"`
	SessionID string `json:"session_id"`
}

func (m SystemMessage) MessageType() string { return "system" }

// SDKArtifactInfo describes a changed or presented artifact.
type SDKArtifactInfo struct {
	Path        string `json:"path"`
	DisplayPath string `json:"display_path"`
	Name        string `json:"name"`
	Group       *struct {
		Key  string `json:"key"`
		Name string `json:"name"`
	} `json:"group,omitempty"`
	RelativePath string `json:"relative_path,omitempty"`
	Size         *int64 `json:"size,omitempty"`
	MIME         string `json:"mime,omitempty"`
	Mtime        *int64 `json:"mtime,omitempty"`
	Kind         string `json:"kind,omitempty"` // changed | presented
	Additions    int    `json:"additions,omitempty"`
	Deletions    int    `json:"deletions,omitempty"`
	IsNew        bool   `json:"is_new,omitempty"`
}

// PartialAssistantMessage is an incremental stream event.
type PartialAssistantMessage struct {
	Type            string                    `json:"type"` // "stream_event"
	Event           BetaRawMessageStreamEvent `json:"event"`
	ParentToolUseID *string                   `json:"parent_tool_use_id"`
	UUID            UUID                      `json:"uuid"`
	SessionID       string                    `json:"session_id"`
}

func (m PartialAssistantMessage) MessageType() string { return "stream_event" }

// CommandLifecycleMessage reports delivery state for a queued user command.
type CommandLifecycleMessage struct {
	Type        string `json:"type"` // "command_lifecycle"
	CommandUUID string `json:"command_uuid"`
	State       string `json:"state"` // queued|started|completed|cancelled|discarded
	UUID        UUID   `json:"uuid"`
	SessionID   string `json:"session_id"`
}

func (m CommandLifecycleMessage) MessageType() string { return "command_lifecycle" }

// PromptSuggestionMessage carries a suggested follow-up prompt.
type PromptSuggestionMessage struct {
	Type       string `json:"type"` // "prompt_suggestion"
	Suggestion string `json:"suggestion"`
	UUID       UUID   `json:"uuid"`
	SessionID  string `json:"session_id"`
}

func (m PromptSuggestionMessage) MessageType() string { return "prompt_suggestion" }

// CloudAgentEventMessage forwards a cloud-agent event.
type CloudAgentEventMessage struct {
	Type      string          `json:"type"` // "cloud_agent_event"
	Event     string          `json:"event"`
	ID        string          `json:"id,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	UUID      UUID            `json:"uuid"`
	SessionID string          `json:"session_id"`
}

func (m CloudAgentEventMessage) MessageType() string { return "cloud_agent_event" }

// KeepAlive is a transport-level keep-alive frame.
type KeepAlive struct {
	Type string `json:"type"` // "keep_alive"
}

func (m KeepAlive) MessageType() string { return "keep_alive" }

// TranscriptMirrorMessage is an SDK-internal post-commit transcript frame.
type TranscriptMirror struct {
	Type      string                  `json:"type"` // "transcript_mirror"
	SessionID string                  `json:"session_id,omitempty"`
	FilePath  string                  `json:"filePath"`
	Entries   []TranscriptMirrorEntry `json:"entries"`
}

// TranscriptMirrorEntry is one mirrored transcript entry.
type TranscriptMirrorEntry struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"-"`
}

func (m TranscriptMirror) MessageType() string { return "transcript_mirror" }
