package qodersdk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
	"github.com/godeps/qoder-agent-sdk-go/runtime"
)

// Plugin management. Mirrors the TypeScript SDK's global/plugins.js: a thin,
// session-less wrapper around `qodercli plugins <subcommand> --json`.
// These calls do not spawn an agent session and never execute plugin code
// (validate is static).

// PluginScope is an installation scope.
type PluginScope string

const (
	PluginScopeUser    PluginScope = "user"    // global (default)
	PluginScopeProject PluginScope = "project" // shared, committed to VCS
	PluginScopeLocal   PluginScope = "local"   // personal, not committed
)

// PluginResourceSkill is one skill bundled by a plugin.
type PluginResourceSkill struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ArgumentHint string `json:"argumentHint,omitempty"`
}

// PluginResourceAgent is one agent bundled by a plugin.
type PluginResourceAgent struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Color       string `json:"color,omitempty"`
}

// PluginResourceMcpServer is one MCP server bundled by a plugin.
type PluginResourceMcpServer struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

// PluginResourceCommand is one command bundled by a plugin.
type PluginResourceCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description,omitempty"`
	ArgumentHint string `json:"argumentHint,omitempty"`
}

// PluginResourceHook is one hook bundled by a plugin.
type PluginResourceHook struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher,omitempty"`
	Type    string `json:"type"`
}

// PluginResources is the bundle of resources a plugin contributes.
type PluginResources struct {
	Skills     []PluginResourceSkill     `json:"skills"`
	Agents     []PluginResourceAgent     `json:"agents"`
	McpServers []PluginResourceMcpServer `json:"mcpServers"`
	Commands   []PluginResourceCommand   `json:"commands"`
	Hooks      []PluginResourceHook      `json:"hooks"`
}

// PluginDetails describes one installed plugin.
type PluginDetails struct {
	ID          string          `json:"id"`
	Name        string          `json:"name,omitempty"`
	Source      string          `json:"source"`
	Version     string          `json:"version"`
	Scope       string          `json:"scope"`
	Enabled     bool            `json:"enabled"`
	CanDisable  bool            `json:"canDisable"`
	DisplayName string          `json:"displayName,omitempty"`
	Description string          `json:"description,omitempty"`
	InstalledAt string          `json:"installedAt,omitempty"`
	LastUpdated string          `json:"lastUpdated,omitempty"`
	ProjectPath string          `json:"projectPath,omitempty"`
	Resources   PluginResources `json:"resources"`
}

// InstallPluginResult is the outcome of an install.
type InstallPluginResult struct {
	PluginID string `json:"pluginId"`
	Version  string `json:"version"`
	Path     string `json:"path"`
}

// UninstallPluginResult is the outcome of an uninstall.
type UninstallPluginResult struct {
	PluginID          string   `json:"pluginId"`
	FullyRemoved      bool     `json:"fullyRemoved"`
	ReverseDependents []string `json:"reverseDependents"`
}

// PluginValidationPosition is a text position in a diagnostic.
type PluginValidationPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// PluginValidationLocation pinpoints a diagnostic.
type PluginValidationLocation struct {
	Path        string `json:"path"`
	JSONPointer string `json:"jsonPointer,omitempty"`
	Range       *struct {
		Start PluginValidationPosition `json:"start"`
		End   PluginValidationPosition `json:"end"`
	} `json:"range,omitempty"`
}

// PluginValidationDiagnostic is one validation finding.
type PluginValidationDiagnostic struct {
	Severity   string                    `json:"severity"` // error | warning
	Code       string                    `json:"code"`
	Message    string                    `json:"message"`
	Phase      string                    `json:"phase"` // target | manifest | component | load-simulation | installability
	Location   *PluginValidationLocation `json:"location,omitempty"`
	Suggestion string                    `json:"suggestion,omitempty"`
}

// PluginValidationReport is the structured result of ValidatePlugin.
type PluginValidationReport struct {
	SchemaVersion int `json:"schemaVersion"`
	Validator     struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"validator"`
	Valid  bool `json:"valid"`
	Target struct {
		InputPath    string `json:"inputPath"`
		ResolvedPath string `json:"resolvedPath"`
		ManifestPath string `json:"manifestPath,omitempty"`
	} `json:"target"`
	Plugin *struct {
		Name        string `json:"name"`
		Version     string `json:"version,omitempty"`
		DisplayName string `json:"displayName,omitempty"`
	} `json:"plugin,omitempty"`
	Diagnostics []PluginValidationDiagnostic `json:"diagnostics"`
	Summary     struct {
		ErrorCount     int `json:"errorCount"`
		WarningCount   int `json:"warningCount"`
		ComponentCount int `json:"componentCount"`
	} `json:"summary"`
	Resources *PluginResources `json:"resources,omitempty"`
}

// PluginOptions configures plugin management calls.
type PluginOptions struct {
	// CWD resolves project/local plugin state and enablement.
	CWD string
	// Scope overrides the installation scope (install/uninstall/enable/disable).
	Scope PluginScope
	// PathToCLI overrides CLI resolution.
	PathToCLI string
	// Brand selects the CLI binary family for resolution.
	Brand auth.Brand
	// Env is extra environment for the child process.
	Env map[string]string
	// KeepData preserves the plugin data directory on uninstall.
	KeepData bool
	// Strict makes ValidatePlugin fail on warnings.
	Strict bool
}

// pluginCLI resolves the CLI path for plugin management.
func pluginCLI(opts *PluginOptions) (string, error) {
	if opts == nil {
		opts = &PluginOptions{}
	}
	return runtime.ResolvePath(opts.Brand, opts.PathToCLI)
}

// runPluginCLI runs one `qodercli plugins ...` invocation and returns stdout.
func runPluginCLI(ctx context.Context, opts *PluginOptions, args ...string) ([]byte, error) {
	path, err := pluginCLI(opts)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, path, args...)
	if opts != nil && opts.CWD != "" {
		cmd.Dir = opts.CWD
	}
	cmd.Env = os.Environ()
	if opts != nil {
		for k, v := range opts.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Some subcommands (plugins validate) exit non-zero while still
		// writing a valid JSON report to stdout — return both.
		return stdout.Bytes(), fmt.Errorf("qoder plugins %v: %w: %s", args, err, truncate(stderrFirst(stderr.String(), stdout.String()), 500))
	}
	return stdout.Bytes(), nil
}

func stderrFirst(stderr, stdout string) string {
	if stderr != "" {
		return stderr
	}
	return stdout
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func scopeArgs(opts *PluginOptions) []string {
	if opts != nil && opts.Scope != "" {
		return []string{"--scope", string(opts.Scope)}
	}
	return nil
}

// ListPlugins lists installed plugins (and, with the CLI's --available flag,
// marketplace offerings). Session-less.
func ListPlugins(ctx context.Context, opts *PluginOptions) ([]PluginDetails, error) {
	args := append([]string{"plugins", "list", "--json"}, scopeArgs(opts)...)
	out, err := runPluginCLI(ctx, opts, args...)
	if err != nil {
		return nil, err
	}
	var details []PluginDetails
	if err := json.Unmarshal(out, &details); err != nil {
		return nil, fmt.Errorf("qoder plugins list: decode: %w (raw: %s)", err, truncate(string(out), 200))
	}
	return details, nil
}

// InstallPlugin installs from a marketplace name (name@marketplace), a
// remote HTTPS .zip URL, or a local directory path. Session-less.
func InstallPlugin(ctx context.Context, plugin string, opts *PluginOptions) (*InstallPluginResult, error) {
	args := append([]string{"plugins", "install", plugin, "--json"}, scopeArgs(opts)...)
	out, err := runPluginCLI(ctx, opts, args...)
	if err != nil {
		return nil, err
	}
	var res InstallPluginResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("qoder plugins install: decode: %w (raw: %s)", err, truncate(string(out), 200))
	}
	return &res, nil
}

// UninstallPlugin removes a plugin. KeepData preserves its data directory.
// Session-less.
func UninstallPlugin(ctx context.Context, id string, opts *PluginOptions) (*UninstallPluginResult, error) {
	args := append([]string{"plugins", "uninstall", id, "--json"}, scopeArgs(opts)...)
	if opts != nil && opts.KeepData {
		args = append(args, "--keep-data")
	}
	out, err := runPluginCLI(ctx, opts, args...)
	if err != nil {
		return nil, err
	}
	var res UninstallPluginResult
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("qoder plugins uninstall: decode: %w (raw: %s)", err, truncate(string(out), 200))
	}
	return &res, nil
}

// EnablePlugin enables a disabled plugin. Session-less.
func EnablePlugin(ctx context.Context, id string, opts *PluginOptions) error {
	args := append([]string{"plugins", "enable", id}, scopeArgs(opts)...)
	_, err := runPluginCLI(ctx, opts, args...)
	return err
}

// DisablePlugin disables an enabled plugin. Session-less.
func DisablePlugin(ctx context.Context, id string, opts *PluginOptions) error {
	args := append([]string{"plugins", "disable", id}, scopeArgs(opts)...)
	_, err := runPluginCLI(ctx, opts, args...)
	return err
}

// ValidatePlugin statically validates a local plugin directory or manifest
// without creating a session or executing plugin hooks, MCP servers, or
// binaries. Invalid content is returned as a report (Valid=false with
// diagnostics); only transport and CLI capability failures return an error.
func ValidatePlugin(ctx context.Context, path string, opts *PluginOptions) (*PluginValidationReport, error) {
	args := []string{"plugins", "validate", path, "--json"}
	if opts != nil && opts.Strict {
		args = append(args, "--strict")
	}
	out, err := runPluginCLI(ctx, opts, args...)
	// validate exits non-zero on an invalid plugin; the JSON report is still
	// on stdout, so decode before propagating the run error.
	if report, perr := parseValidationReport(out); perr == nil {
		return report, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("qoder plugins validate: decode: invalid report (raw: %s)", truncate(string(out), 200))
}

// parseValidationReport decodes the validate output, accepting both the bare
// report and the CLI's {schemaVersion, ok, report} envelope (qodercli 1.1.49).
func parseValidationReport(out []byte) (*PluginValidationReport, error) {
	var envelope struct {
		SchemaVersion int                     `json:"schemaVersion"`
		OK            bool                    `json:"ok"`
		Report        *PluginValidationReport `json:"report"`
	}
	if err := json.Unmarshal(out, &envelope); err == nil && envelope.Report != nil && envelope.Report.SchemaVersion > 0 {
		return envelope.Report, nil
	}
	var report PluginValidationReport
	if err := json.Unmarshal(out, &report); err == nil && report.SchemaVersion > 0 {
		return &report, nil
	}
	return nil, fmt.Errorf("unrecognized validation report shape")
}

// ReloadPlugins asks a live session to reload plugins and returns the new
// command/agent/plugin/MCP surface. (Control-request counterpart of the
// session-less helpers above.)
func (s *Session) ReloadPlugins() (*protocol.ReloadPluginsResult, error) {
	var out protocol.ReloadPluginsResult
	if err := s.sendControl(protocol.ReloadPluginsRequest{Type: "reload_plugins"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ReloadSkills asks a live session to reload skills.
func (s *Session) ReloadSkills() (*protocol.ReloadSkillsResult, error) {
	var out protocol.ReloadSkillsResult
	if err := s.sendControl(protocol.ReloadSkillsRequest{Type: "reload_skills"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
