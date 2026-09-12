package protocol

import "encoding/json"

// Response payloads for SDK→CLI control requests. Field shapes mirror the
// TypeScript SDK's protocol/control.d.ts (wire-verified against qodercli
// 1.1.49).

// InterruptResponse is the response to an interrupt request.
type InterruptResponse struct {
	// StillQueued lists UUIDs of queued messages that survive the interrupt.
	StillQueued []string `json:"still_queued"`
	// Cancelled lists UUIDs removed by cancel_queued=true.
	Cancelled []string `json:"cancelled,omitempty"`
}

// SetModelResponse is the response to set_model (empty object on success;
// the switch takes effect from the next model call).
type SetModelResponse struct{}

// SetPermissionModeResponse is the response to set_permission_mode.
type SetPermissionModeResponse struct{}

// ContextUsageCategory is one /context-style usage bucket.
type ContextUsageCategory struct {
	Type       string  `json:"type"` // system_prompt | system_tools | skills | messages | other | free_space | auto_compact
	Percentage float64 `json:"percentage"`
}

// ContextSkillItem is one skill's context footprint.
type ContextSkillItem struct {
	Name                string  `json:"name"`
	Source              string  `json:"source"` // project | user | built-in | plugin
	PercentageOfContext float64 `json:"percentageOfContext"`
}

// DuplicateFileRead is a file read more than once in the session.
type DuplicateFileRead struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// GetContextUsageResponse is the response to get_context_usage.
type GetContextUsageResponse struct {
	Model         string `json:"model"`
	ContextWindow struct {
		UsedPercentage float64 `json:"usedPercentage"`
	} `json:"contextWindow"`
	Categories  []ContextUsageCategory `json:"categories"`
	AutoCompact struct {
		Enabled             bool    `json:"enabled"`
		ThresholdPercentage float64 `json:"thresholdPercentage"`
	} `json:"autoCompact"`
	Skills struct {
		Count               int                `json:"count"`
		PercentageOfContext float64            `json:"percentageOfContext"`
		Items               []ContextSkillItem `json:"items"`
	} `json:"skills"`
	DuplicateFileReads []DuplicateFileRead `json:"duplicateFileReads"`
	Session            struct {
		MessageCount int `json:"messageCount"`
		PromptCount  int `json:"promptCount"`
		ToolCalls    struct {
			Total     int `json:"total"`
			Succeeded int `json:"succeeded"`
			Failed    int `json:"failed"`
		} `json:"toolCalls"`
		LinesChanged struct {
			Added   int `json:"added"`
			Removed int `json:"removed"`
		} `json:"linesChanged"`
	} `json:"session"`
}

// UsageQuotaBucket is one account quota bucket.
type UsageQuotaBucket struct {
	Total      *float64 `json:"total,omitempty"`
	Used       *float64 `json:"used,omitempty"`
	Remaining  *float64 `json:"remaining,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	DetailURL  string   `json:"detailUrl,omitempty"` // add-on buckets only
}

// UsageOrgResourcePackage is an organization resource package quota.
type UsageOrgResourcePackage struct {
	Used       *float64 `json:"used,omitempty"`
	Cap        *float64 `json:"cap,omitempty"`
	Remaining  *float64 `json:"remaining,omitempty"`
	Percentage *float64 `json:"percentage,omitempty"`
	Available  *bool    `json:"available,omitempty"`
	Unit       string   `json:"unit,omitempty"`
}

// SessionCreditsUsage is session-local credit consumption (independent of
// account quota).
type SessionCreditsUsage struct {
	TotalCredits float64                       `json:"total_credits"`
	ModelUsage   map[string]SessionModelCredit `json:"model_usage,omitempty"`
}

// SessionModelCredit is per-model credit usage within a session.
type SessionModelCredit struct {
	Credits float64 `json:"credits"`
}

// UsageInfo is the account usage summary.
type UsageInfo struct {
	UserID               string                   `json:"userId,omitempty"`
	UserType             string                   `json:"userType,omitempty"`
	TotalUsagePercentage *float64                 `json:"totalUsagePercentage,omitempty"`
	IsHighestTier        *bool                    `json:"isHighestTier,omitempty"`
	ExpiresAt            *int64                   `json:"expiresAt,omitempty"`
	UpgradeURL           string                   `json:"upgradeUrl,omitempty"`
	UserQuota            *UsageQuotaBucket        `json:"userQuota,omitempty"`
	AddOnQuota           *UsageQuotaBucket        `json:"addOnQuota,omitempty"`
	IsQuotaExceeded      *bool                    `json:"isQuotaExceeded,omitempty"`
	IsPlanQuotaProrated  *bool                    `json:"isPlanQuotaProrated,omitempty"`
	OrgResourcePackage   *UsageOrgResourcePackage `json:"orgResourcePackage,omitempty"`
	Session              *SessionCreditsUsage     `json:"session,omitempty"`
}

// GetUsageInfoResponse is the response to get_usage_info.
type GetUsageInfoResponse struct {
	Usage      *UsageInfo           `json:"usage"`
	Session    *SessionCreditsUsage `json:"session,omitempty"`
	UsageError string               `json:"usage_error,omitempty"`
}

// McpStatusResponse is the response to mcp_status.
type McpStatusResponse struct {
	Servers []McpServerStatus `json:"servers"`
}

// RewindScope selects what a rewind rolls back.
type RewindScope string

const (
	RewindScopeConversation RewindScope = "conversation"
	RewindScopeFiles        RewindScope = "files"
	RewindScopeBoth         RewindScope = "both"
)

// RewindFileFailure is one file that could not be restored.
type RewindFileFailure struct {
	Path  string `json:"path"`
	Error string `json:"error"`
}

// RewindResult is the response to rewind. Status is one of
// ready (successful dry run) | success | partial | rejected.
type RewindResult struct {
	Status              RewindStatus        `json:"status"`
	TargetUserMessageID string              `json:"targetUserMessageId"`
	Scope               RewindScope         `json:"scope"`
	RetainedLeafID      *string             `json:"retainedLeafId,omitempty"`
	RemovedMessageIDs   []string            `json:"removedMessageIds,omitempty"`
	FilesChanged        []string            `json:"filesChanged,omitempty"`
	Insertions          *int                `json:"insertions,omitempty"`
	Deletions           *int                `json:"deletions,omitempty"`
	FailedFiles         []RewindFileFailure `json:"failedFiles,omitempty"`
	Error               string              `json:"error,omitempty"`
}

// RewindStatus is the outcome of a rewind request.
type RewindStatus string

const (
	RewindStatusReady    RewindStatus = "ready"
	RewindStatusSuccess  RewindStatus = "success"
	RewindStatusPartial  RewindStatus = "partial"
	RewindStatusRejected RewindStatus = "rejected"
)

// RewindFilesResult is the response to rewind_files.
type RewindFilesResult struct {
	CanRewind    bool     `json:"canRewind"`
	Error        string   `json:"error,omitempty"`
	FilesChanged []string `json:"filesChanged,omitempty"`
	Insertions   *int     `json:"insertions,omitempty"`
	Deletions    *int     `json:"deletions,omitempty"`
}

// --- BYOK response types ---

// BYOKLocalizedName is a server-provided bilingual name.
type BYOKLocalizedName struct {
	EnUS string `json:"en_us,omitempty"`
	CnZh string `json:"cn_zh,omitempty"`
}

// BYOKFieldInfo is one credential field a provider requires.
type BYOKFieldInfo struct {
	Key             string             `json:"key"`
	DisplayName     string             `json:"display_name"`
	DisplayNameI18n *BYOKLocalizedName `json:"display_name_i18n,omitempty"`
	Type            string             `json:"type"`
	Mandatory       *bool              `json:"mandatory,omitempty"`
}

// BYOKModelInfo is one model offered by a BYOK provider.
type BYOKModelInfo struct {
	Key              string   `json:"key"`
	DisplayName      string   `json:"display_name"`
	IsVL             bool     `json:"is_vl"`
	IsReasoning      bool     `json:"is_reasoning"`
	Format           string   `json:"format"`
	MaxInputTokens   int      `json:"max_input_tokens"`
	Efforts          []string `json:"efforts,omitempty"`
	SupportsDisabled *bool    `json:"supports_disabled,omitempty"`
}

// BYOKModelTypeInfo is a model group within a provider.
type BYOKModelTypeInfo struct {
	Key             string             `json:"key,omitempty"`
	DisplayName     string             `json:"display_name"`
	DisplayNameI18n *BYOKLocalizedName `json:"display_name_i18n,omitempty"`
	Models          []BYOKModelInfo    `json:"models"`
}

// BYOKProviderInfo is one third-party provider in the BYOK catalog.
type BYOKProviderInfo struct {
	Key             string              `json:"key"`
	DisplayName     string              `json:"display_name"`
	DisplayNameI18n *BYOKLocalizedName  `json:"display_name_i18n,omitempty"`
	Source          string              `json:"source,omitempty"`
	APIKeyURL       string              `json:"api_key_url"`
	URL             string              `json:"url"`
	Fields          []BYOKFieldInfo     `json:"fields"`
	Types           []BYOKModelTypeInfo `json:"types"`
}

// GetByokConfigResponse is the response to get_byok_config.
type GetByokConfigResponse struct {
	Providers []BYOKProviderInfo `json:"providers"`
}

// ValidateByokModelResponse is the response to validate_byok_model.
type ValidateByokModelResponse struct {
	Success bool `json:"success"`
}

// ByokModelConfigInfo is a persisted per-model BYOK config (secret-free).
type ByokModelConfigInfo struct {
	Key                       string   `json:"key,omitempty"`
	DisplayName               string   `json:"displayName,omitempty"`
	Provider                  string   `json:"provider,omitempty"`
	Model                     string   `json:"model,omitempty"`
	Type                      string   `json:"type,omitempty"`
	BaseURL                   string   `json:"baseUrl,omitempty"`
	Style                     string   `json:"style,omitempty"`
	Vision                    bool     `json:"vision,omitempty"`
	Reasoning                 bool     `json:"reasoning,omitempty"`
	MaxInputTokens            int      `json:"maxInputTokens,omitempty"`
	ReasoningEfforts          []string `json:"reasoningEfforts,omitempty"`
	SupportsDisabledReasoning *bool    `json:"supportsDisabledReasoning,omitempty"`
	ReasoningEffort           string   `json:"reasoningEffort,omitempty"`
	AvailableContextWindows   []int    `json:"availableContextWindows,omitempty"`
	ContextWindow             int      `json:"contextWindow,omitempty"`

	// Custom-provider variant fields.
	ProviderID     string            `json:"providerId,omitempty"`
	ProviderType   string            `json:"providerType,omitempty"`
	Protocol       string            `json:"protocol,omitempty"`
	AuthType       string            `json:"authType,omitempty"`
	DefaultModelID string            `json:"defaultModelId,omitempty"`
	Models         []json.RawMessage `json:"models,omitempty"`
}

// ListByokConfigsResponse is the response to list_byok_configs.
type ListByokConfigsResponse struct {
	Configs []ByokModelConfigInfo `json:"configs"`
}

// ByokConfigReference identifies a created BYOK config.
type ByokConfigReference struct {
	Key        string `json:"key,omitempty"`
	ProviderID string `json:"providerId,omitempty"`
}

// CreateByokConfigResponse is the response to create_byok_config.
type CreateByokConfigResponse struct {
	Config ByokConfigReference `json:"config"`
}

// ByokConfigCheckDetails carries failure diagnostics.
type ByokConfigCheckDetails struct {
	Field        string `json:"field,omitempty"`
	HTTPStatus   *int   `json:"httpStatus,omitempty"`
	UpstreamCode string `json:"upstreamCode,omitempty"`
}

// ByokConfigCheckResult is the response to check_byok_config.
type ByokConfigCheckResult struct {
	Success   bool                    `json:"success"`
	Error     string                  `json:"error,omitempty"`
	Code      string                  `json:"code,omitempty"`
	RequestID string                  `json:"requestId,omitempty"`
	Details   *ByokConfigCheckDetails `json:"details,omitempty"`
}

// AccountInfoResponse is the response to account_info.
type AccountInfoResponse struct {
	Raw json.RawMessage `json:"-"`
}

// ReloadPluginsResult is the response to reload_plugins.
type ReloadPluginsResult struct {
	Commands   []SlashCommand      `json:"commands"`
	Agents     []AgentInfo         `json:"agents"`
	Plugins    []ReloadedPluginRef `json:"plugins"`
	McpServers []McpServerStatus   `json:"mcpServers"`
	ErrorCount int                 `json:"error_count"`
}

// ReloadedPluginRef identifies one loaded plugin after a reload.
type ReloadedPluginRef struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

// ReloadSkillsResult is the response to reload_skills.
type ReloadSkillsResult struct {
	Skills []SlashCommand `json:"skills"`
}
