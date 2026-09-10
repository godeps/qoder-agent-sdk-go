package protocol

// McpStdioServerConfig configures a stdio-launched MCP server.
type McpStdioServerConfig struct {
	Type    string            `json:"type,omitempty"` // "stdio"
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Timeout *int              `json:"timeout,omitempty"` // ms; <1000 ignored
}

// McpSSEServerConfig configures an SSE MCP server.
type McpSSEServerConfig struct {
	Type    string            `json:"type"` // "sse"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout *int              `json:"timeout,omitempty"`
}

// McpHttpServerConfig configures a streamable-HTTP MCP server.
type McpHttpServerConfig struct {
	Type    string            `json:"type"` // "http"
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
	Timeout *int              `json:"timeout,omitempty"`
}

// McpSdkServerConfig references an in-process SDK MCP server by name.
type McpSdkServerConfig struct {
	Type string `json:"type"` // "sdk"
	Name string `json:"name"`
}

// McpToolRuntimeOverride adjusts how a server's tool is surfaced.
type McpToolRuntimeOverride struct {
	ExposedName string `json:"exposedName,omitempty"`
	AlwaysLoad  *bool  `json:"alwaysLoad,omitempty"`
}

// McpServerConfig is the process-transport union for an MCP server. The Type
// field selects which transport: "stdio" | "sse" | "http" | "sdk". Only the
// fields relevant to the selected type need be set.
type McpServerConfig struct {
	Type          string                            `json:"type"`
	Command       string                            `json:"command,omitempty"`
	Args          []string                          `json:"args,omitempty"`
	Env           map[string]string                 `json:"env,omitempty"`
	Timeout       *int                              `json:"timeout,omitempty"`
	URL           string                            `json:"url,omitempty"`
	Headers       map[string]string                 `json:"headers,omitempty"`
	Name          string                            `json:"name,omitempty"` // sdk
	ToolOverrides map[string]McpToolRuntimeOverride `json:"toolOverrides,omitempty"`
}

// NewMcpStdioConfig builds a stdio MCP server config.
func NewMcpStdioConfig(command string, args []string, env map[string]string) McpServerConfig {
	return McpServerConfig{Type: "stdio", Command: command, Args: args, Env: env}
}

// NewMcpHTTPConfig builds a streamable-HTTP MCP server config.
func NewMcpHTTPConfig(url string, headers map[string]string) McpServerConfig {
	return McpServerConfig{Type: "http", URL: url, Headers: headers}
}

// NewMcpSSEConfig builds an SSE MCP server config.
func NewMcpSSEConfig(url string, headers map[string]string) McpServerConfig {
	return McpServerConfig{Type: "sse", URL: url, Headers: headers}
}

// NewMcpSdkConfig references an in-process SDK MCP server by name.
func NewMcpSdkConfig(name string) McpServerConfig {
	return McpServerConfig{Type: "sdk", Name: name}
}

// McpServerStatusConfig is the config shape embedded in McpServerStatus.
type McpServerStatusConfig = McpServerConfig

// McpServerToolAnnotation carries tool-level annotations.
type McpServerToolAnnotation struct {
	ReadOnly    *bool `json:"readOnly,omitempty"`
	Destructive *bool `json:"destructive,omitempty"`
	OpenWorld   *bool `json:"openWorld,omitempty"`
}

// McpServerStatusTool describes one tool exposed by a server.
type McpServerStatusTool struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description,omitempty"`
	Annotations *McpServerToolAnnotation `json:"annotations,omitempty"`
}

// McpServerStatus reports a server's runtime state.
type McpServerStatus struct {
	Name       string `json:"name"`
	Status     string `json:"status"` // connected | failed | needs-auth | pending | disabled
	ServerInfo *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
	Error  string                 `json:"error,omitempty"`
	Config *McpServerStatusConfig `json:"config,omitempty"`
	Scope  string                 `json:"scope,omitempty"`
	Tools  []McpServerStatusTool  `json:"tools,omitempty"`
}

// McpSetServersResult is the response to mcp_set_servers.
type McpSetServersResult struct {
	Added   []string          `json:"added"`
	Removed []string          `json:"removed"`
	Errors  map[string]string `json:"errors"`
}

// OAuthToken is an MCP OAuth token.
type OAuthToken struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
	ExpiresAt    *int64 `json:"expiresAt,omitempty"`
	Scope        string `json:"scope,omitempty"`
}
