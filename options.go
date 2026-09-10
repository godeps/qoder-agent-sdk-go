package qodersdk

import (
	"context"
	"encoding/json"

	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

// Options configures a query session. Build one with NewOptions and the With*
// builders, or populate the struct directly.
type Options struct {
	// --- Runtime / launch ---
	PathToCLI   string
	CWD         string
	Env         map[string]string
	Proxy       string
	VpcEndpoint string
	Auth        auth.AuthOptions
	Debug       bool

	// --- Model / behavior ---
	Model              string
	Agent              string
	MaxTurns           *int
	SystemPrompt       string
	AppendSystemPrompt string

	// --- Permissions ---
	PermissionMode                  protocol.PermissionMode
	AllowDangerouslySkipPermissions bool
	AllowedTools                    []string
	DisallowedTools                 []string

	// --- Streaming ---
	IncludePartialMessages bool
	IncludeHookEvents      bool

	// --- MCP / tools ---
	McpServers map[string]protocol.McpServerConfig
	Hooks      map[protocol.HookEvent][]protocol.HookSpec
	Agents     map[string]protocol.AgentDefinition
	Skills     []string
	Plugins    []protocol.SdkPluginConfig

	// --- Session ---
	SessionID      string
	Continue       bool
	Resume         string
	ForkSession    bool
	PersistSession *bool

	// --- Settings / directories ---
	AdditionalDirectories []string
	SettingSources        []string
	Settings              map[string]json.RawMessage
	ExtraArgs             map[string]*string

	// --- Transport tuning ---
	CloseGraceMs        int
	InitializeTimeoutMs int

	// --- Callbacks (CLI→SDK control requests) ---
	CanUseTool    func(ctx context.Context, req *protocol.CanUseToolRequest) (protocol.PermissionResult, error)
	HookCallback  func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error)
	ResolveModel  func(ctx context.Context, req *protocol.GetModelPolicyRequest) (ModelPolicyResult, error)
	OnAuthExpired func()
}

// ModelPolicyResult is the host's response to a get_model_policy request.
type ModelPolicyResult struct {
	Model       string                     `json:"model,omitempty"`
	Parameters  map[string]json.RawMessage `json:"parameters,omitempty"`
	CustomModel *CustomModel               `json:"custom_model,omitempty"`
	TaskID      string                     `json:"task_id,omitempty"`
	SubTask     string                     `json:"sub_task,omitempty"`
}

// CustomModel is a per-call BYOK credential payload.
type CustomModel struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model,omitempty"`
	URL      string `json:"url,omitempty"`
	Style    string `json:"style,omitempty"`
	IsVl     *bool  `json:"isVl,omitempty"`
}

// NewOptions returns a new Options with sensible defaults.
func NewOptions() *Options {
	return &Options{
		CloseGraceMs:        2000,
		InitializeTimeoutMs: 60000,
	}
}

// --- Builders ---

func (o *Options) WithModel(m string) *Options              { o.Model = m; return o }
func (o *Options) WithCWD(cwd string) *Options              { o.CWD = cwd; return o }
func (o *Options) WithEnv(env map[string]string) *Options   { o.Env = env; return o }
func (o *Options) WithProxy(p string) *Options              { o.Proxy = p; return o }
func (o *Options) WithVpcEndpoint(v string) *Options        { o.VpcEndpoint = v; return o }
func (o *Options) WithAuth(a auth.AuthOptions) *Options     { o.Auth = a; return o }
func (o *Options) WithPathToCLI(p string) *Options          { o.PathToCLI = p; return o }
func (o *Options) WithMaxTurns(n int) *Options              { o.MaxTurns = &n; return o }
func (o *Options) WithSystemPrompt(s string) *Options       { o.SystemPrompt = s; return o }
func (o *Options) WithAppendSystemPrompt(s string) *Options { o.AppendSystemPrompt = s; return o }
func (o *Options) WithPermissionMode(m protocol.PermissionMode) *Options {
	o.PermissionMode = m
	return o
}
func (o *Options) WithAllowedTools(t []string) *Options    { o.AllowedTools = t; return o }
func (o *Options) WithDisallowedTools(t []string) *Options { o.DisallowedTools = t; return o }
func (o *Options) WithMcpServers(m map[string]protocol.McpServerConfig) *Options {
	o.McpServers = m
	return o
}
func (o *Options) WithPartialMessages(b bool) *Options { o.IncludePartialMessages = b; return o }
func (o *Options) WithHookEvents(b bool) *Options      { o.IncludeHookEvents = b; return o }
func (o *Options) WithAgent(a string) *Options         { o.Agent = a; return o }
func (o *Options) WithSessionID(s string) *Options     { o.SessionID = s; return o }
func (o *Options) WithDangerouslySkipPermissions(b bool) *Options {
	o.AllowDangerouslySkipPermissions = b
	return o
}

// WithCanUseTool registers the tool-permission callback.
func (o *Options) WithCanUseTool(f func(ctx context.Context, req *protocol.CanUseToolRequest) (protocol.PermissionResult, error)) *Options {
	o.CanUseTool = f
	return o
}

// WithHookCallback registers the hook callback.
func (o *Options) WithHookCallback(f func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error)) *Options {
	o.HookCallback = f
	return o
}

// WithResolveModel registers the model-policy callback.
func (o *Options) WithResolveModel(f func(ctx context.Context, req *protocol.GetModelPolicyRequest) (ModelPolicyResult, error)) *Options {
	o.ResolveModel = f
	return o
}
