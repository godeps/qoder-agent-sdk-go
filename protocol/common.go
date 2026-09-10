package protocol

import "encoding/json"

// UUID is a wire-level UUID; it is a plain string on the wire.
type UUID string

// Base64ImageSource is a base64-encoded image input.
type Base64ImageSource struct {
	Type      string `json:"type"` // "base64"
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// URLImageSource is a URL-referenced image input.
type URLImageSource struct {
	Type string `json:"type"` // "url"
	URL  string `json:"url"`
}

// ContentBlock is a generic content block (text, tool_use, tool_result, image, ...).
// The wire shape is open-ended; unknown fields are ignored on decode.
type ContentBlock struct {
	Type      string          `json:"type"`
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`
	Name      string          `json:"name,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Content   json.RawMessage `json:"content,omitempty"`
	Source    json.RawMessage `json:"source,omitempty"`
	ToolUseID string          `json:"tool_use_id,omitempty"`
	IsError   bool            `json:"is_error,omitempty"`
}

// MessageParam is a user-side message envelope.
type MessageParam struct {
	Role    string          `json:"role"`    // "user"
	Content json.RawMessage `json:"content"` // string or []ContentBlock
}

// BetaCacheCreation is the cache-creation breakdown within BetaUsage.
type BetaCacheCreation struct {
	Ephemeral1HInputTokens *int `json:"ephemeral_1h_input_tokens,omitempty"`
	Ephemeral5MInputTokens *int `json:"ephemeral_5m_input_tokens,omitempty"`
}

// BetaServerToolUse is the server-side tool-use breakdown within BetaUsage.
type BetaServerToolUse struct {
	WebFetchRequests  *int `json:"web_fetch_requests,omitempty"`
	WebSearchRequests *int `json:"web_search_requests,omitempty"`
}

// BetaUsage is the model usage block on a BetaMessage (nullable upstream).
type BetaUsage struct {
	CacheCreation            *BetaCacheCreation `json:"cache_creation,omitempty"`
	CacheCreationInputTokens *int               `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int               `json:"cache_read_input_tokens,omitempty"`
	InferenceGeo             *string            `json:"inference_geo,omitempty"`
	InputTokens              *int               `json:"input_tokens,omitempty"`
	Iterations               json.RawMessage    `json:"iterations,omitempty"`
	OutputTokens             *int               `json:"output_tokens,omitempty"`
	ServerToolUse            *BetaServerToolUse `json:"server_tool_use,omitempty"`
	ServiceTier              *string            `json:"service_tier,omitempty"`
	Speed                    *string            `json:"speed,omitempty"`
	Credits                  *float64           `json:"credits,omitempty"`
	OriginalCredits          *float64           `json:"original_credits,omitempty"`
	Billable                 *bool              `json:"billable,omitempty"`
}

// BetaStopReason is the stop reason on a BetaMessage.
type BetaStopReason string

// BetaMessage is the assistant message payload (Anthropic-style).
type BetaMessage struct {
	ID           string          `json:"id,omitempty"`
	Type         string          `json:"type,omitempty"` // "message"
	Role         string          `json:"role"`           // "assistant"
	Content      []ContentBlock  `json:"content"`
	Model        string          `json:"model,omitempty"`
	StopReason   *BetaStopReason `json:"stop_reason,omitempty"`
	StopSequence *string         `json:"stop_sequence,omitempty"`
	Usage        *BetaUsage      `json:"usage,omitempty"`
}

// BetaRawMessageStreamEvent is the incremental stream-event payload.
type BetaRawMessageStreamEvent struct {
	Type         string          `json:"type"`
	Index        *int            `json:"index,omitempty"`
	Delta        json.RawMessage `json:"delta,omitempty"`
	ContentBlock *ContentBlock   `json:"content_block,omitempty"`
	Message      *BetaMessage    `json:"message,omitempty"`
	Usage        *BetaUsage      `json:"usage,omitempty"`
}

// NonNullableCacheCreation is the non-null cache-creation breakdown.
type NonNullableCacheCreation struct {
	Ephemeral1HInputTokens int `json:"ephemeral_1h_input_tokens"`
	Ephemeral5MInputTokens int `json:"ephemeral_5m_input_tokens"`
}

// NonNullableServerToolUse is the non-null server-side tool-use breakdown.
type NonNullableServerToolUse struct {
	WebFetchRequests  int `json:"web_fetch_requests"`
	WebSearchRequests int `json:"web_search_requests"`
}

// NonNullableUsage is the SDK-consumed usage form (all fields non-null).
type NonNullableUsage struct {
	CacheCreation            NonNullableCacheCreation `json:"cache_creation"`
	CacheCreationInputTokens int                      `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int                      `json:"cache_read_input_tokens"`
	InferenceGeo             string                   `json:"inference_geo"`
	InputTokens              int                      `json:"input_tokens"`
	Iterations               json.RawMessage          `json:"iterations"`
	OutputTokens             int                      `json:"output_tokens"`
	ServerToolUse            NonNullableServerToolUse `json:"server_tool_use"`
	ServiceTier              string                   `json:"service_tier"`
	Speed                    string                   `json:"speed"`
	RequestID                string                   `json:"request_id,omitempty"`
	ContextUsageRatio        *float64                 `json:"context_usage_ratio,omitempty"`
	Credits                  *float64                 `json:"credits,omitempty"`
	OriginalCredits          *float64                 `json:"original_credits,omitempty"`
	Billable                 *bool                    `json:"billable,omitempty"`
}

// ApiKeySource identifies where the API key originated.
type ApiKeySource string

const (
	ApiKeySourceNone      ApiKeySource = "none"
	ApiKeySourceUser      ApiKeySource = "user"
	ApiKeySourceProject   ApiKeySource = "project"
	ApiKeySourceOrg       ApiKeySource = "org"
	ApiKeySourceTemporary ApiKeySource = "temporary"
	ApiKeySourceOAuth     ApiKeySource = "oauth"
)

// ExitReason is the session-end reason.
type ExitReason string

const (
	ExitReasonClear                     ExitReason = "clear"
	ExitReasonResume                    ExitReason = "resume"
	ExitReasonLogout                    ExitReason = "logout"
	ExitReasonPromptInputExit           ExitReason = "prompt_input_exit"
	ExitReasonOther                     ExitReason = "other"
	ExitReasonBypassPermissionsDisabled ExitReason = "bypass_permissions_disabled"
)

// FastModeState is the fast-mode lifecycle state.
type FastModeState string

const (
	FastModeOff      FastModeState = "off"
	FastModeCooldown FastModeState = "cooldown"
	FastModeOn       FastModeState = "on"
)

// SDKAssistantMessageError classifies an assistant-level error.
type SDKAssistantMessageError string

const (
	AssistantErrAuth            SDKAssistantMessageError = "authentication_failed"
	AssistantErrBilling         SDKAssistantMessageError = "billing_error"
	AssistantErrRateLimit       SDKAssistantMessageError = "rate_limit"
	AssistantErrInvalidRequest  SDKAssistantMessageError = "invalid_request"
	AssistantErrServer          SDKAssistantMessageError = "server_error"
	AssistantErrUnknown         SDKAssistantMessageError = "unknown"
	AssistantErrMaxOutputTokens SDKAssistantMessageError = "max_output_tokens"
)

// EffortLevel is a reasoning-effort selector.
type EffortLevel string

const (
	EffortLow    EffortLevel = "low"
	EffortMedium EffortLevel = "medium"
	EffortHigh   EffortLevel = "high"
	EffortMax    EffortLevel = "max"
)

// AccountInfo describes the authenticated account.
type AccountInfo struct {
	UserID           *string       `json:"userId,omitempty"`
	Name             *string       `json:"name,omitempty"`
	Email            *string       `json:"email,omitempty"`
	Organization     *string       `json:"organization,omitempty"`
	OrganizationName *string       `json:"organizationName,omitempty"`
	SubscriptionType *string       `json:"subscriptionType,omitempty"`
	TokenSource      *string       `json:"tokenSource,omitempty"`
	APIKeySource     *ApiKeySource `json:"apiKeySource,omitempty"`
	APIProvider      *string       `json:"apiProvider,omitempty"`
}

// SlashCommand is a registered slash command.
type SlashCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argumentHint"`
}

// PluginInfo is a discovered plugin entry.
type PluginInfo struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source,omitempty"`
}

// ModelUsage is per-model cumulative usage for a session.
type ModelUsage struct {
	InputTokens              int      `json:"inputTokens"`
	OutputTokens             int      `json:"outputTokens"`
	CacheReadInputTokens     int      `json:"cacheReadInputTokens"`
	CacheCreationInputTokens int      `json:"cacheCreationInputTokens"`
	WebSearchRequests        int      `json:"webSearchRequests"`
	CostUSD                  float64  `json:"costUSD"`
	Credits                  *float64 `json:"credits,omitempty"`
	ContextWindow            int      `json:"contextWindow"`
	MaxOutputTokens          int      `json:"maxOutputTokens"`
}

// ToolConfig carries optional tool-level configuration.
type ToolConfig struct {
	AskUserQuestion *struct {
		PreviewFormat string `json:"previewFormat,omitempty"`
	} `json:"askUserQuestion,omitempty"`
}

// SdkPluginConfig is a local plugin declaration.
type SdkPluginConfig struct {
	Type string `json:"type"` // "local"
	Path string `json:"path"`
}

// ModelContextWindowEntry is one context-window tier.
type ModelContextWindowEntry struct {
	TokenCount int   `json:"token_count"`
	IsDefault  *bool `json:"is_default,omitempty"`
}

// ModelContextConfig maps tier labels to token counts.
type ModelContextConfig map[string]ModelContextWindowEntry

// ModelEffortEntry describes one reasoning-effort option.
type ModelEffortEntry struct {
	Description *string `json:"description,omitempty"`
	IsDefault   *bool   `json:"is_default,omitempty"`
}

// ModelThinkingDisabled / Enabled are the server-side thinking config halves.
type ModelThinkingDisabled struct {
	Description *string `json:"description,omitempty"`
}
type ModelThinkingEnabled struct {
	Description *string                     `json:"description,omitempty"`
	Efforts     map[string]ModelEffortEntry `json:"efforts,omitempty"`
	IsDefault   *bool                       `json:"is_default,omitempty"`
}

// ModelThinkingConfig is the server-side thinking config block.
type ModelThinkingConfig struct {
	Disabled *ModelThinkingDisabled `json:"disabled,omitempty"`
	Enabled  *ModelThinkingEnabled  `json:"enabled,omitempty"`
}

// ModelStrategy is a codebase security-tag strategy entry.
type ModelStrategy struct {
	Tag                string `json:"tag"`
	Enabled            bool   `json:"enabled"`
	DisabledMessageKey string `json:"disabled_message_key"`
}

// ModelSource classifies the origin of a model entry.
type ModelSource string

const (
	ModelSourceSystem ModelSource = "system"
	ModelSourceUser   ModelSource = "user"
	ModelSourceOrg    ModelSource = "organization"
	ModelSourceCustom ModelSource = "custom"
)

// LocalizedModelText is a localized text bag.
type LocalizedModelText map[string]string

// ModelPromotion is per-model promotion/discount info.
type ModelPromotion struct {
	Active                     bool               `json:"active"`
	Badge                      LocalizedModelText `json:"badge,omitempty"`
	Description                LocalizedModelText `json:"description,omitempty"`
	DiscountFactor             *float64           `json:"discount_factor,omitempty"`
	BeforePromotionPriceFactor *float64           `json:"before_promotion_price_factor,omitempty"`
	Timezone                   *string            `json:"timezone,omitempty"`
	RuleID                     *string            `json:"rule_id,omitempty"`
	WindowStart                *string            `json:"window_start,omitempty"`
	WindowEnd                  *string            `json:"window_end,omitempty"`
}

// ModelInfo is a model catalog entry.
type ModelInfo struct {
	Value                   string               `json:"value"`
	ModelID                 *string              `json:"modelId,omitempty"`
	DisplayName             string               `json:"displayName"`
	Description             string               `json:"description"`
	Source                  *ModelSource         `json:"source,omitempty"`
	IsDefault               *bool                `json:"isDefault,omitempty"`
	IsEnabled               *bool                `json:"isEnabled,omitempty"`
	IsNew                   *bool                `json:"isNew,omitempty"`
	IsFree                  *bool                `json:"isFree,omitempty"`
	IsReasoning             *bool                `json:"isReasoning,omitempty"`
	IsVl                    *bool                `json:"isVl,omitempty"`
	PriceFactor             *float64             `json:"priceFactor,omitempty"`
	OriginalPriceFactor     *float64             `json:"originalPriceFactor,omitempty"`
	MaxInputTokens          *int                 `json:"maxInputTokens,omitempty"`
	MaxOutputTokens         *int                 `json:"maxOutputTokens,omitempty"`
	Efforts                 []string             `json:"efforts,omitempty"`
	DefaultEffort           *string              `json:"defaultEffort,omitempty"`
	SupportsDisabled        *bool                `json:"supportsDisabled,omitempty"`
	AvailableContextWindows []int                `json:"availableContextWindows,omitempty"`
	DefaultContextWindow    *int                 `json:"defaultContextWindow,omitempty"`
	Tags                    []string             `json:"tags,omitempty"`
	Strategies              []ModelStrategy      `json:"strategies,omitempty"`
	Format                  *string              `json:"format,omitempty"`
	Scene                   *string              `json:"scene,omitempty"`
	ServerScene             *string              `json:"serverScene,omitempty"`
	Icon                    *string              `json:"icon,omitempty"`
	URL                     *string              `json:"url,omitempty"`
	Model                   *string              `json:"model,omitempty"`
	Provider                *string              `json:"provider,omitempty"`
	ContextConfig           *ModelContextConfig  `json:"context_config,omitempty"`
	ThinkingConfig          *ModelThinkingConfig `json:"thinking_config,omitempty"`
	Promotion               *ModelPromotion      `json:"promotion,omitempty"`
	ServerModel             json.RawMessage      `json:"serverModel,omitempty"`
}
