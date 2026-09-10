package protocol

import (
	"encoding/json"
	"fmt"
)

// ControlRequest is the wire envelope the CLI sends to the SDK (and vice versa)
// for bidirectional control operations outside the agentic loop.
type ControlRequest struct {
	Type      string          `json:"type"` // "control_request"
	RequestID string          `json:"request_id"`
	SessionID string          `json:"session_id,omitempty"`
	Request   json.RawMessage `json:"request"` // ControlRequestInner, decode via Inner()
}

func (m ControlRequest) MessageType() string { return "control_request" }

// Inner decodes the request payload into a ControlRequestInner.
func (m ControlRequest) Inner() (ControlRequestInner, error) {
	return ParseControlRequestInner(m.Request)
}

// ControlRequestInner is a decoded control request payload, discriminated by
// type or subtype. Concrete types implement it via innerMarker().
type ControlRequestInner interface {
	innerMarker()
}

// ParseControlRequestInner decodes a control request payload by its type (or,
// for legacy variants, subtype) discriminant.
func ParseControlRequestInner(raw json.RawMessage) (ControlRequestInner, error) {
	var probe struct {
		Type    string `json:"type"`
		Subtype string `json:"subtype"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("qoder: decode control request: %w", err)
	}
	// Type-discriminated first; fall back to subtype for legacy variants.
	switch probe.Type {
	case "initialize":
		return decodeInner[InitializeRequest](raw)
	case "interrupt":
		return decodeInner[InterruptRequest](raw)
	case "set_permission_mode":
		return decodeInner[SetPermissionModeRequest](raw)
	case "set_model":
		return decodeInner[SetModelRequest](raw)
	case "set_proxy":
		return decodeInner[SetProxyRequest](raw)
	case "generate_session_title":
		return decodeInner[GenerateSessionTitleRequest](raw)
	case "set_max_thinking_tokens":
		return decodeInner[SetMaxThinkingTokensRequest](raw)
	case "mcp_status":
		return decodeInner[McpStatusRequest](raw)
	case "get_context_usage":
		return decodeInner[GetContextUsageRequest](raw)
	case "get_usage_info":
		return decodeInner[GetUsageInfoRequest](raw)
	case "get_model_policy":
		return decodeInner[GetModelPolicyRequest](raw)
	case "account_info":
		return decodeInner[AccountInfoRequest](raw)
	case "rewind_files":
		return decodeInner[RewindFilesRequest](raw)
	case "rewind":
		return decodeInner[RewindRequest](raw)
	case "seed_read_state":
		return decodeInner[SeedReadStateRequest](raw)
	case "mcp_set_servers":
		return decodeInner[McpSetServersRequest](raw)
	case "reload_plugins":
		return decodeInner[ReloadPluginsRequest](raw)
	case "reload_skills":
		return decodeInner[ReloadSkillsRequest](raw)
	case "mcp_reconnect":
		return decodeInner[McpReconnectRequest](raw)
	case "mcp_toggle":
		return decodeInner[McpToggleRequest](raw)
	case "apply_flag_settings":
		return decodeInner[ApplyFlagSettingsRequest](raw)
	case "get_settings":
		return decodeInner[GetSettingsRequest](raw)
	case "end_session":
		return decodeInner[EndSessionRequest](raw)
	case "channel_enable":
		return decodeInner[ChannelEnableRequest](raw)
	case "enable_remote_projection":
		return decodeInner[EnableRemoteProjectionRequest](raw)
	case "disable_remote_projection":
		return decodeInner[DisableRemoteProjectionRequest](raw)
	case "mcp_inject_token":
		return decodeInner[McpInjectTokenRequest](raw)
	case "mcp_authenticate":
		return decodeInner[McpAuthenticateRequest](raw)
	case "mcp_oauth_callback_url":
		return decodeInner[McpOAuthCallbackUrlRequest](raw)
	case "mcp_clear_auth":
		return decodeInner[McpClearAuthRequest](raw)
	case "get_byok_config":
		return decodeInner[GetByokConfigRequest](raw)
	case "validate_byok_model":
		return decodeInner[ValidateByokModelRequest](raw)
	case "list_byok_configs":
		return decodeInner[ListByokConfigsRequest](raw)
	case "create_byok_config":
		return decodeInner[CreateByokConfigRequest](raw)
	case "update_byok_config":
		return decodeInner[UpdateByokConfigRequest](raw)
	case "delete_byok_config":
		return decodeInner[DeleteByokConfigRequest](raw)
	case "check_byok_config":
		return decodeInner[CheckByokConfigRequest](raw)
	case "list_plugins":
		return decodeInner[ListPluginsRequest](raw)
	case "memory_should_generate":
		return decodeInner[MemoryShouldGenerateRequest](raw)
	case "flush_memory":
		return decodeInner[FlushMemoryRequest](raw)
	case "refresh_memory":
		return decodeInner[RefreshMemoryRequest](raw)
	case "flush_skill_evolution":
		return decodeInner[FlushSkillEvolutionRequest](raw)
	case "elicitation_response":
		return decodeInner[ElicitationResponseRequest](raw)
	}
	// Subtype-discriminated (legacy/variant) requests.
	switch probe.Subtype {
	case "can_use_tool":
		return decodeInner[CanUseToolRequest](raw)
	case "hook_callback":
		return decodeInner[HookCallbackRequest](raw)
	case "mcp_message":
		return decodeInner[McpMessageRequest](raw)
	case "side_question":
		return decodeInner[SideQuestionRequest](raw)
	case "resolve_session_artifacts":
		return decodeInner[ResolveSessionArtifactsRequest](raw)
	case "skill_evolution_should_review":
		return decodeInner[SkillEvolutionShouldReviewRequest](raw)
	case "cancel_async_message":
		return decodeInner[CancelAsyncMessageRequest](raw)
	case "stop_task":
		return decodeInner[StopTaskRequest](raw)
	case "background_tasks":
		return decodeInner[BackgroundTasksRequest](raw)
	case "set_plan_mode":
		return decodeInner[SetPlanModeRequest](raw)
	case "get_plan_mode":
		return decodeInner[GetPlanModeRequest](raw)
	case "set_goal":
		return decodeInner[SetGoalRequest](raw)
	case "set_goal_max_turns":
		return decodeInner[SetGoalMaxTurnsRequest](raw)
	case "get_goal":
		return decodeInner[GetGoalRequest](raw)
	case "clear_goal":
		return decodeInner[ClearGoalRequest](raw)
	case "elicitation":
		return decodeInner[ElicitationRequest](raw)
	}
	return &UnknownControlRequest{Type: probe.Type, Subtype: probe.Subtype, Raw: append([]byte(nil), raw...)}, nil
}

func decodeInner[T ControlRequestInner](raw json.RawMessage) (ControlRequestInner, error) {
	var m T
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("qoder: decode control inner: %w", err)
	}
	return m, nil
}

// UnknownControlRequest preserves an unrecognized control request.
type UnknownControlRequest struct {
	Type    string
	Subtype string
	Raw     json.RawMessage
}

func (m UnknownControlRequest) innerMarker() {}

// ControlSuccessResponse is the success body of a control response.
type ControlSuccessResponse struct {
	Subtype   string                     `json:"subtype"` // "success"
	RequestID string                     `json:"request_id"`
	Response  map[string]json.RawMessage `json:"response,omitempty"`
}

// ControlErrorResponse is the error body of a control response.
type ControlErrorResponse struct {
	Subtype                   string                     `json:"subtype"` // "error"
	RequestID                 string                     `json:"request_id"`
	Error                     string                     `json:"error"`
	Code                      string                     `json:"code,omitempty"`
	Retryable                 *bool                      `json:"retryable,omitempty"`
	Details                   map[string]json.RawMessage `json:"details,omitempty"`
	PendingPermissionRequests []ControlRequest           `json:"pending_permission_requests,omitempty"`
}

// ControlResponse is the wire envelope for a control response (either direction).
type ControlResponse struct {
	Type      string          `json:"type"` // "control_response"
	SessionID string          `json:"session_id,omitempty"`
	Response  json.RawMessage `json:"response"` // ControlSuccessResponse | ControlErrorResponse
}

func (m ControlResponse) MessageType() string { return "control_response" }

// IsSuccess reports whether the response is a success.
func (m ControlResponse) IsSuccess() bool {
	var probe struct {
		Subtype string `json:"subtype"`
	}
	_ = json.Unmarshal(m.Response, &probe)
	return probe.Subtype == "success"
}

// Success decodes a success response body. Returns nil for error responses.
func (m ControlResponse) Success() (*ControlSuccessResponse, error) {
	if !m.IsSuccess() {
		return nil, nil
	}
	var s ControlSuccessResponse
	if err := json.Unmarshal(m.Response, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// Error decodes an error response body. Returns nil for success responses.
func (m ControlResponse) Error() (*ControlErrorResponse, error) {
	if m.IsSuccess() {
		return nil, nil
	}
	var e ControlErrorResponse
	if err := json.Unmarshal(m.Response, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// NewSuccessResponse builds a control_response success envelope.
func NewSuccessResponse(requestID string, response map[string]json.RawMessage) ControlResponse {
	body, _ := json.Marshal(ControlSuccessResponse{Subtype: "success", RequestID: requestID, Response: response})
	return ControlResponse{Type: "control_response", Response: body}
}

// NewErrorResponse builds a control_response error envelope.
func NewErrorResponse(requestID, message string) ControlResponse {
	body, _ := json.Marshal(ControlErrorResponse{Subtype: "error", RequestID: requestID, Error: message})
	return ControlResponse{Type: "control_response", Response: body}
}

// ControlCancel is the wire envelope for cancelling an in-flight control request.
type ControlCancel struct {
	Type      string `json:"type"` // "control_cancel" | "control_cancel_request"
	RequestID string `json:"request_id"`
	SessionID string `json:"session_id,omitempty"`
}

func (m ControlCancel) MessageType() string { return "control_cancel" }

// --- Inner request types (type-discriminated) ---

// InitializeRequest is the SDK→CLI handshake payload.
type InitializeRequest struct {
	Type                           string                                       `json:"type"` // "initialize"
	Model                          string                                       `json:"model,omitempty"`
	CWD                            string                                       `json:"cwd,omitempty"`
	AllowedTools                   []string                                     `json:"allowedTools,omitempty"`
	DisallowedTools                []string                                     `json:"disallowedTools,omitempty"`
	McpServers                     map[string]McpServerConfig                   `json:"mcpServers,omitempty"`
	Agents                         map[string]AgentDefinition                   `json:"agents,omitempty"`
	Skills                         []string                                     `json:"skills,omitempty"`
	Plugins                        []SdkPluginConfig                            `json:"plugins,omitempty"`
	SystemPrompt                   string                                       `json:"systemPrompt,omitempty"`
	AppendSystemPrompt             string                                       `json:"appendSystemPrompt,omitempty"`
	ModelRequestPatches            json.RawMessage                              `json:"modelRequestPatches,omitempty"`
	ExcludeDynamicSections         *bool                                        `json:"excludeDynamicSections,omitempty"`
	PermissionMode                 PermissionMode                               `json:"permissionMode,omitempty"`
	IndependentPlanModeControls    *bool                                        `json:"independentPlanModeControls,omitempty"`
	MaxThinkingTokens              *int                                         `json:"maxThinkingTokens,omitempty"`
	EnableFileCheckpointing        *bool                                        `json:"enableFileCheckpointing,omitempty"`
	SdkMcpServers                  []string                                     `json:"sdkMcpServers,omitempty"`
	SdkMcpToolOverrides            map[string]map[string]McpToolRuntimeOverride `json:"sdkMcpToolOverrides,omitempty"`
	PromptSuggestions              *bool                                        `json:"promptSuggestions,omitempty"`
	GoalMaxTurns                   *int                                         `json:"goalMaxTurns,omitempty"`
	SupportsCatalogReadyInitialize *bool                                        `json:"supportsCatalogReadyInitialize,omitempty"`
	SupportsAvailableModelsUpdate  *bool                                        `json:"supportsAvailableModelsUpdate,omitempty"`
	SupportsCommandsChanged        *bool                                        `json:"supportsCommandsChanged,omitempty"`
	ResolveSessionArtifacts        *bool                                        `json:"resolveSessionArtifacts,omitempty"`
	InitializeTimeoutMs            *int                                         `json:"initializeTimeoutMs,omitempty"`
	AgentProgressSummaries         *bool                                        `json:"agentProgressSummaries,omitempty"`
	Memory                         json.RawMessage                              `json:"memory,omitempty"`
	Evolution                      json.RawMessage                              `json:"evolution,omitempty"`
	Hooks                          map[HookEvent][]HookSpec                     `json:"hooks,omitempty"`
	Scene                          string                                       `json:"scene,omitempty"`
}

// HookSpec is one hook registration in initialize.
type HookSpec struct {
	HookCallbackIDs []string `json:"hookCallbackIds"`
	Timeout         *int     `json:"timeout,omitempty"`
	Matcher         string   `json:"matcher,omitempty"`
}

func (m InitializeRequest) innerMarker() {}

// InterruptRequest asks the CLI to stop the current task.
type InterruptRequest struct {
	Type         string `json:"type"` // "interrupt"
	CancelQueued *bool  `json:"cancel_queued,omitempty"`
}

func (m InterruptRequest) innerMarker() {}

// SetPermissionModeRequest changes the active permission mode.
type SetPermissionModeRequest struct {
	Type             string         `json:"type"` // "set_permission_mode"
	Mode             PermissionMode `json:"mode"`
	PreservePlanMode *bool          `json:"preservePlanMode,omitempty"`
}

func (m SetPermissionModeRequest) innerMarker() {}

// SetModelRequest changes the active model.
type SetModelRequest struct {
	Type  string `json:"type"` // "set_model"
	Model string `json:"model"`
}

func (m SetModelRequest) innerMarker() {}

// SetProxyRequest sets or clears an outbound proxy.
type SetProxyRequest struct {
	Type  string  `json:"type"` // "set_proxy"
	Proxy *string `json:"proxy"`
}

func (m SetProxyRequest) innerMarker() {}

// GenerateSessionTitleRequest asks the CLI to generate a session title.
type GenerateSessionTitleRequest struct {
	Type        string `json:"type"` // "generate_session_title"
	Description string `json:"description"`
	Persist     *bool  `json:"persist,omitempty"`
}

func (m GenerateSessionTitleRequest) innerMarker() {}

// SetMaxThinkingTokensRequest adjusts the thinking-token budget.
type SetMaxThinkingTokensRequest struct {
	Type              string `json:"type"` // "set_max_thinking_tokens"
	MaxThinkingTokens *int   `json:"max_thinking_tokens"`
}

func (m SetMaxThinkingTokensRequest) innerMarker() {}

// McpStatusRequest queries MCP server status.
type McpStatusRequest struct {
	Type string `json:"type"` // "mcp_status"
}

func (m McpStatusRequest) innerMarker() {}

// GetContextUsageRequest queries context-window usage.
type GetContextUsageRequest struct {
	Type string `json:"type"` // "get_context_usage"
}

func (m GetContextUsageRequest) innerMarker() {}

// GetUsageInfoRequest queries account/session usage.
type GetUsageInfoRequest struct {
	Type string `json:"type"` // "get_usage_info"
}

func (m GetUsageInfoRequest) innerMarker() {}

// GetModelPolicyRequest is the CLI→SDK per-LLM-call model selection query.
type GetModelPolicyRequest struct {
	Type                  string            `json:"type"` // "get_model_policy"
	Purpose               QoderModelPurpose `json:"purpose"`
	SessionID             string            `json:"sessionId"`
	TurnIndex             int               `json:"turnIndex"`
	AgentID               string            `json:"agentId,omitempty"`
	AgentType             string            `json:"agentType,omitempty"`
	ResolvedModel         string            `json:"resolvedModel,omitempty"`
	ContextUsage          *float64          `json:"contextUsage,omitempty"`
	EstimatedInputTokens  *int              `json:"estimatedInputTokens,omitempty"`
	EstimatedOutputTokens *int              `json:"estimatedOutputTokens,omitempty"`
	Retry                 *struct {
		Attempt     int    `json:"attempt"`
		LastError   string `json:"lastError"`
		LastModelID string `json:"lastModelId"`
	} `json:"retry,omitempty"`
	Models []ModelInfo `json:"models,omitempty"`
}

func (m GetModelPolicyRequest) innerMarker() {}

// AccountInfoRequest queries the authenticated account.
type AccountInfoRequest struct {
	Type string `json:"type"` // "account_info"
}

func (m AccountInfoRequest) innerMarker() {}

// RewindFilesRequest rewinds file state to a user message.
type RewindFilesRequest struct {
	Type          string `json:"type"` // "rewind_files"
	UserMessageID string `json:"user_message_id"`
	DryRun        *bool  `json:"dry_run,omitempty"`
}

func (m RewindFilesRequest) innerMarker() {}

// RewindRequest rewinds conversation and/or file state.
type RewindRequest struct {
	Type          string  `json:"type"` // "rewind"
	UserMessageID string  `json:"user_message_id"`
	Scope         *string `json:"scope,omitempty"` // conversation | files | both
	DryRun        *bool   `json:"dry_run,omitempty"`
}

func (m RewindRequest) innerMarker() {}

// SeedReadStateRequest seeds the CLI's file-read state cache.
type SeedReadStateRequest struct {
	Type  string `json:"type"` // "seed_read_state"
	Path  string `json:"path"`
	Mtime int64  `json:"mtime"`
}

func (m SeedReadStateRequest) innerMarker() {}

// McpSetServersRequest replaces the MCP server set.
type McpSetServersRequest struct {
	Type    string                     `json:"type"` // "mcp_set_servers"
	Servers map[string]McpServerConfig `json:"servers"`
}

func (m McpSetServersRequest) innerMarker() {}

// ReloadPluginsRequest asks the CLI to reload plugins.
type ReloadPluginsRequest struct {
	Type string `json:"type"`
}

func (m ReloadPluginsRequest) innerMarker() {}

// ReloadSkillsRequest asks the CLI to reload skills.
type ReloadSkillsRequest struct {
	Type string `json:"type"`
}

func (m ReloadSkillsRequest) innerMarker() {}

// McpReconnectRequest asks the CLI to reconnect an MCP server.
type McpReconnectRequest struct {
	Type       string `json:"type"` // "mcp_reconnect"
	ServerName string `json:"serverName"`
}

func (m McpReconnectRequest) innerMarker() {}

// McpToggleRequest enables/disables an MCP server.
type McpToggleRequest struct {
	Type       string `json:"type"` // "mcp_toggle"
	ServerName string `json:"serverName"`
	Enabled    bool   `json:"enabled"`
}

func (m McpToggleRequest) innerMarker() {}

// ApplyFlagSettingsRequest applies CLI flag settings.
type ApplyFlagSettingsRequest struct {
	Type     string                     `json:"type"` // "apply_flag_settings"
	Settings map[string]json.RawMessage `json:"settings"`
}

func (m ApplyFlagSettingsRequest) innerMarker() {}

// GetSettingsRequest reads resolved CLI settings.
type GetSettingsRequest struct {
	Type string `json:"type"`
}

func (m GetSettingsRequest) innerMarker() {}

// EndSessionRequest ends the session.
type EndSessionRequest struct {
	Type   string `json:"type"` // "end_session"
	Reason string `json:"reason,omitempty"`
}

func (m EndSessionRequest) innerMarker() {}

// ChannelEnableRequest toggles a channel.
type ChannelEnableRequest struct {
	Type    string `json:"type"` // "channel_enable"
	Channel string `json:"channel"`
	Enabled bool   `json:"enabled"`
}

func (m ChannelEnableRequest) innerMarker() {}

// EnableRemoteProjectionRequest enables remote-control projection.
type EnableRemoteProjectionRequest struct {
	Type                           string `json:"type"` // "enable_remote_projection"
	RemoteSessionID                string `json:"remote_session_id"`
	SuppressInitialIdleStateEvents *bool  `json:"suppress_initial_idle_state_events,omitempty"`
	SkipInitialFlush               *bool  `json:"skip_initial_flush,omitempty"`
	FromSequenceNum                *int   `json:"from_sequence_num,omitempty"`
}

func (m EnableRemoteProjectionRequest) innerMarker() {}

// DisableRemoteProjectionRequest disables remote-control projection.
type DisableRemoteProjectionRequest struct {
	Type string `json:"type"`
}

func (m DisableRemoteProjectionRequest) innerMarker() {}

// McpInjectTokenRequest injects an OAuth token into an MCP server.
type McpInjectTokenRequest struct {
	Type       string     `json:"type"` // "mcp_inject_token"
	ServerName string     `json:"serverName"`
	Token      OAuthToken `json:"token"`
}

func (m McpInjectTokenRequest) innerMarker() {}

// McpAuthenticateRequest starts MCP OAuth.
type McpAuthenticateRequest struct {
	Type        string `json:"type"` // "mcp_authenticate"
	ServerName  string `json:"serverName"`
	RedirectURI string `json:"redirectUri,omitempty"`
}

func (m McpAuthenticateRequest) innerMarker() {}

// McpOAuthCallbackUrlRequest supplies an OAuth callback URL.
type McpOAuthCallbackUrlRequest struct {
	Type        string `json:"type"` // "mcp_oauth_callback_url"
	ServerName  string `json:"serverName"`
	CallbackURL string `json:"callbackUrl"`
}

func (m McpOAuthCallbackUrlRequest) innerMarker() {}

// McpClearAuthRequest clears MCP auth for a server.
type McpClearAuthRequest struct {
	Type       string `json:"type"` // "mcp_clear_auth"
	ServerName string `json:"serverName"`
}

func (m McpClearAuthRequest) innerMarker() {}

// GetByokConfigRequest queries the BYOK catalog.
type GetByokConfigRequest struct {
	Type string `json:"type"`
}

func (m GetByokConfigRequest) innerMarker() {}

// ValidateByokModelRequest validates a BYOK model+key.
type ValidateByokModelRequest struct {
	Type     string `json:"type"` // "validate_byok_model"
	Provider string `json:"provider"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
	URL      string `json:"url,omitempty"`
	Style    string `json:"style,omitempty"`
}

func (m ValidateByokModelRequest) innerMarker() {}

// ListByokConfigsRequest lists BYOK configs.
type ListByokConfigsRequest struct {
	Type string `json:"type"`
}

func (m ListByokConfigsRequest) innerMarker() {}

// CreateByokConfigRequest creates a BYOK config (Config preserved raw).
type CreateByokConfigRequest struct {
	Type   string          `json:"type"` // "create_byok_config"
	Config json.RawMessage `json:"config"`
}

func (m CreateByokConfigRequest) innerMarker() {}

// UpdateByokConfigRequest updates a BYOK config.
type UpdateByokConfigRequest struct {
	Type   string          `json:"type"` // "update_byok_config"
	Config json.RawMessage `json:"config"`
}

func (m UpdateByokConfigRequest) innerMarker() {}

// DeleteByokConfigRequest deletes a BYOK config.
type DeleteByokConfigRequest struct {
	Type   string          `json:"type"` // "delete_byok_config"
	Target json.RawMessage `json:"target"`
}

func (m DeleteByokConfigRequest) innerMarker() {}

// CheckByokConfigRequest checks a BYOK config.
type CheckByokConfigRequest struct {
	Type   string          `json:"type"` // "check_byok_config"
	Config json.RawMessage `json:"config"`
}

func (m CheckByokConfigRequest) innerMarker() {}

// ListPluginsRequest lists plugins.
type ListPluginsRequest struct {
	Type string `json:"type"`
}

func (m ListPluginsRequest) innerMarker() {}

// MemoryShouldGenerateRequest is a memory gate callback from the CLI.
type MemoryShouldGenerateRequest struct {
	Type       string          `json:"type"` // "memory_should_generate"
	CallbackID string          `json:"callbackId"`
	Input      json.RawMessage `json:"input"`
}

func (m MemoryShouldGenerateRequest) innerMarker() {}

// FlushMemoryRequest flushes pending memory writes.
type FlushMemoryRequest struct {
	Type string `json:"type"`
}

func (m FlushMemoryRequest) innerMarker() {}

// RefreshMemoryRequest refreshes memory.
type RefreshMemoryRequest struct {
	Type string `json:"type"`
}

func (m RefreshMemoryRequest) innerMarker() {}

// FlushSkillEvolutionRequest flushes pending skill-evolution reviews.
type FlushSkillEvolutionRequest struct {
	Type string `json:"type"`
}

func (m FlushSkillEvolutionRequest) innerMarker() {}

// --- Inner request types (subtype-discriminated) ---

// CanUseToolRequest is the CLI→SDK permission prompt for a tool call.
type CanUseToolRequest struct {
	Subtype               string                       `json:"subtype"` // "can_use_tool"
	ToolName              string                       `json:"tool_name"`
	Input                 map[string]json.RawMessage   `json:"input"`
	PermissionSuggestions []PermissionUpdate           `json:"permission_suggestions,omitempty"`
	BlockedPath           string                       `json:"blocked_path,omitempty"`
	DecisionReason        string                       `json:"decision_reason,omitempty"`
	DecisionReasonType    string                       `json:"decision_reason_type,omitempty"`
	ClassifierApprovable  *bool                        `json:"classifier_approvable,omitempty"`
	Title                 string                       `json:"title,omitempty"`
	DisplayName           string                       `json:"display_name,omitempty"`
	Description           string                       `json:"description,omitempty"`
	ToolUseID             string                       `json:"tool_use_id"`
	AgentID               string                       `json:"agent_id,omitempty"`
	PermissionKind        string                       `json:"permission_kind,omitempty"`
	Options               []CanUseToolPermissionOption `json:"options,omitempty"`
	Details               *CanUseToolPermissionDetails `json:"details,omitempty"`
}

func (m CanUseToolRequest) innerMarker() {}

// HookCallbackRequest is the CLI→SDK hook execution request.
type HookCallbackRequest struct {
	Subtype    string    `json:"subtype,omitempty"` // "hook_callback"
	CallbackID string    `json:"callback_id,omitempty"`
	Input      HookInput `json:"input"`
	ToolUseID  string    `json:"tool_use_id,omitempty"`
	// Alternate form (type-discriminated as "hook_callback"): hook_id+hook_event.
	Type      HookEvent `json:"type,omitempty"`
	HookID    string    `json:"hook_id,omitempty"`
	HookEvent HookEvent `json:"hook_event,omitempty"`
}

func (m HookCallbackRequest) innerMarker() {}

// McpMessageRequest is the CLI→SDK in-process MCP call.
type McpMessageRequest struct {
	Subtype    string          `json:"subtype,omitempty"` // "mcp_message"
	ServerName string          `json:"server_name"`
	Message    json.RawMessage `json:"message"` // JSONRPCMessage
}

func (m McpMessageRequest) innerMarker() {}

// SideQuestionRequest asks the CLI a side question without interrupting.
type SideQuestionRequest struct {
	Subtype  string                     `json:"subtype"` // "side_question"
	Question string                     `json:"question"`
	History  []SideQuestionHistoryEntry `json:"history,omitempty"`
}

// SideQuestionHistoryEntry is one prior side-question exchange.
type SideQuestionHistoryEntry struct {
	Question       string `json:"question"`
	Response       string `json:"response"`
	FallbackNotice string `json:"fallback_notice,omitempty"`
}

func (m SideQuestionRequest) innerMarker() {}

// ResolveSessionArtifactsRequest resolves session artifacts.
type ResolveSessionArtifactsRequest struct {
	Subtype        string `json:"subtype"` // "resolve_session_artifacts"
	LocalSessionID string `json:"local_session_id"`
	CWD            string `json:"cwd"`
}

func (m ResolveSessionArtifactsRequest) innerMarker() {}

// SkillEvolutionShouldReviewRequest is a skill-evolution gate callback.
type SkillEvolutionShouldReviewRequest struct {
	Subtype    string          `json:"subtype"` // "skill_evolution_should_review"
	CallbackID string          `json:"callbackId"`
	Input      json.RawMessage `json:"input"`
}

func (m SkillEvolutionShouldReviewRequest) innerMarker() {}

// CancelAsyncMessageRequest cancels a queued async message.
type CancelAsyncMessageRequest struct {
	Subtype     string `json:"subtype"` // "cancel_async_message"
	MessageUUID string `json:"message_uuid"`
}

func (m CancelAsyncMessageRequest) innerMarker() {}

// StopTaskRequest stops a running task by ID.
type StopTaskRequest struct {
	Subtype string `json:"subtype"` // "stop_task"
	TaskID  string `json:"task_id"`
}

func (m StopTaskRequest) innerMarker() {}

// BackgroundTasksRequest moves foreground executions to the background.
type BackgroundTasksRequest struct {
	Subtype   string `json:"subtype"` // "background_tasks"
	ToolUseID string `json:"tool_use_id,omitempty"`
}

func (m BackgroundTasksRequest) innerMarker() {}

// SetPlanModeRequest enters or leaves plan mode.
type SetPlanModeRequest struct {
	Subtype string `json:"subtype"` // "set_plan_mode"
	Active  bool   `json:"active"`
}

func (m SetPlanModeRequest) innerMarker() {}

// GetPlanModeRequest reads the plan-mode state.
type GetPlanModeRequest struct {
	Subtype string `json:"subtype"`
}

func (m GetPlanModeRequest) innerMarker() {}

// SetGoalRequest creates, redirects, transitions, or re-budgets the session goal.
type SetGoalRequest struct {
	Subtype       string     `json:"subtype"` // "set_goal"
	Objective     string     `json:"objective,omitempty"`
	Status        GoalStatus `json:"status,omitempty"`
	CreditsBudget *int       `json:"credits_budget,omitempty"`
}

func (m SetGoalRequest) innerMarker() {}

// SetGoalMaxTurnsRequest updates the max-turns copied into future goals.
type SetGoalMaxTurnsRequest struct {
	Subtype  string `json:"subtype"` // "set_goal_max_turns"
	MaxTurns *int   `json:"max_turns"`
}

func (m SetGoalMaxTurnsRequest) innerMarker() {}

// GetGoalRequest reads the current goal.
type GetGoalRequest struct {
	Subtype string `json:"subtype"`
}

func (m GetGoalRequest) innerMarker() {}

// ClearGoalRequest clears the current goal.
type ClearGoalRequest struct {
	Subtype string `json:"subtype"`
}

func (m ClearGoalRequest) innerMarker() {}

// ElicitationRequest is the CLI→SDK elicitation prompt.
type ElicitationRequest struct {
	Subtype         string                     `json:"subtype"` // "elicitation"
	McpServerName   string                     `json:"mcp_server_name"`
	Message         string                     `json:"message"`
	Mode            string                     `json:"mode,omitempty"` // form | url
	URL             string                     `json:"url,omitempty"`
	ElicitationID   string                     `json:"elicitation_id,omitempty"`
	RequestedSchema map[string]json.RawMessage `json:"requested_schema,omitempty"`
	Title           string                     `json:"title,omitempty"`
	DisplayName     string                     `json:"display_name,omitempty"`
	Description     string                     `json:"description,omitempty"`
}

func (m ElicitationRequest) innerMarker() {}

// ElicitationResponseRequest is the host→CLI elicitation response.
type ElicitationResponseRequest struct {
	Type          string   `json:"type"` // "elicitation_response"
	ElicitationID string   `json:"elicitation_id"`
	Prompt        string   `json:"prompt"`
	Options       []string `json:"options,omitempty"`
	ToolUseID     string   `json:"tool_use_id,omitempty"`
}

func (m ElicitationResponseRequest) innerMarker() {}

// --- Snapshot / purpose types ---

// PlanModeSnapshot is the structured plan-mode state.
type PlanModeSnapshot struct {
	Active bool `json:"active"`
}

// GoalStatus is the lifecycle status of a goal.
type GoalStatus string

const (
	GoalActive        GoalStatus = "active"
	GoalPaused        GoalStatus = "paused"
	GoalBlocked       GoalStatus = "blocked"
	GoalUsageLimited  GoalStatus = "usage_limited"
	GoalBudgetLimited GoalStatus = "budget_limited"
	GoalComplete      GoalStatus = "complete"
)

// GoalSnapshot is the structured goal state.
type GoalSnapshot struct {
	ID              string     `json:"id"`
	Objective       string     `json:"objective"`
	Status          GoalStatus `json:"status"`
	TurnsUsed       int        `json:"turns_used"`
	MaxTurns        *int       `json:"max_turns,omitempty"`
	TimeUsedSeconds int        `json:"time_used_seconds"`
	CreditsBudget   *float64   `json:"credits_budget,omitempty"`
	CreditsUsed     *float64   `json:"credits_used,omitempty"`
	CreatedAt       int64      `json:"created_at"`
	UpdatedAt       int64      `json:"updated_at"`
}

// QoderModelPurpose labels the purpose of an LLM call.
type QoderModelPurpose string

const (
	PurposeMain        QoderModelPurpose = "main"
	PurposePlan        QoderModelPurpose = "plan"
	PurposeTask        QoderModelPurpose = "task"
	PurposeCompact     QoderModelPurpose = "compact"
	PurposeTitle       QoderModelPurpose = "title"
	PurposeSuggestion  QoderModelPurpose = "suggestion"
	PurposeGenerate    QoderModelPurpose = "generate"
	PurposeHookPrompt  QoderModelPurpose = "hook_prompt"
	PurposeSubagent    QoderModelPurpose = "subagent"
	PurposeWebFetch    QoderModelPurpose = "web_fetch"
	PurposeImageGen    QoderModelPurpose = "image_gen"
	PurposeCompression QoderModelPurpose = "compression"
	PurposeUtility     QoderModelPurpose = "utility"
)
