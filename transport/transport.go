// Package transport implements the ProcessTransport that spawns qoderclicn as
// a child process and exchanges JSONL over stdin/stdout.
package transport

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
	"github.com/godeps/qoder-agent-sdk-go/runtime"
)

const (
	defaultCloseGraceMs = 2000
	defaultKillGraceMs  = 5000
	sdkVersion          = "1.0.39-go" // informational; reported via QODER_AGENT_SDK_VERSION
)

// TransportOptions mirrors the qoderclicn launch flags and env.
type TransportOptions struct {
	PathToCLI                       string
	CWD                             string
	Env                             map[string]string
	Proxy                           string
	VpcEndpoint                     string
	Auth                            auth.AuthOptions
	Debug                           bool
	Model                           string
	Agent                           string
	MaxTurns                        *int
	PermissionMode                  protocol.PermissionMode
	AllowDangerouslySkipPermissions bool
	IncludePartialMessages          bool
	AllowedTools                    []string
	DisallowedTools                 []string
	McpServers                      map[string]protocol.McpServerConfig
	AllowedMcpServerNames           []string
	StrictMcpConfig                 bool
	Settings                        map[string]json.RawMessage
	SettingSources                  []string
	AdditionalDirectories           []string
	Plugins                         []protocol.SdkPluginConfig
	ExtraArgs                       map[string]*string
	SessionID                       string
	Continue                        bool
	Resume                          string
	ForkSession                     bool
	PersistSession                  *bool
	SessionMirror                   bool
	CanUseTool                      bool // host has a permission callback → --permission-prompt-tool stdio
	CloseGraceMs                    int
	OnAuthExpired                   func()
	StderrHandler                   func(string)
}

// ProcessTransport spawns qoderclicn and manages stdin/stdout JSONL I/O.
type ProcessTransport struct {
	opts            TransportOptions
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	stdout          io.ReadCloser
	stderrBuf       bytes.Buffer
	stderrMu        sync.Mutex
	authPayloadPath string
	closed          atomic.Bool
	closeGraceMs    int
	killGraceMs     int
	writeMu         sync.Mutex
	startedOnce     sync.Once
	ready           atomic.Bool
	msgs            chan protocol.Message
	startErr        error
}

// NewProcessTransport creates a transport with the given options.
func NewProcessTransport(opts TransportOptions) *ProcessTransport {
	if opts.CloseGraceMs <= 0 {
		opts.CloseGraceMs = defaultCloseGraceMs
	}
	return &ProcessTransport{opts: opts, killGraceMs: defaultKillGraceMs}
}

// Initialize resolves the executable, builds argv/env (including the one-shot
// auth payload file), spawns the process, and starts the stdout/stderr pumps.
func (t *ProcessTransport) Initialize(ctx context.Context) error {
	path, err := runtime.ResolvePath(t.opts.PathToCLI)
	if err != nil {
		return err
	}
	if t.opts.Auth.Configured() {
		p, err := auth.WritePayloadFile(t.opts.Auth)
		if err != nil {
			return err
		}
		t.authPayloadPath = p
	}
	args := t.buildArgs()
	env := t.buildEnv()
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = t.opts.CWD
	cmd.Env = env
	cmd.Cancel = func() error { return nil } // graceful handled in Close

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.cleanup()
		return fmt.Errorf("qoder: stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.cleanup()
		return fmt.Errorf("qoder: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.cleanup()
		return fmt.Errorf("qoder: stderr pipe: %w", err)
	}
	t.cmd = cmd
	t.stdin = stdin
	t.stdout = stdout
	t.msgs = make(chan protocol.Message, 64)
	if err := cmd.Start(); err != nil {
		t.cleanup()
		return fmt.Errorf("qoder: spawn qoderclicn: %w", err)
	}
	go t.readStdout(stdout)
	go t.readStderr(stderr)
	t.ready.Store(true)
	return nil
}

// WriteLine writes one JSON line to the CLI stdin.
func (t *ProcessTransport) WriteLine(data []byte) error {
	if t.closed.Load() {
		return ErrClosed
	}
	t.writeMu.Lock()
	defer t.writeMu.Unlock()
	_, err := t.stdin.Write(append(data, '\n'))
	return err
}

// WriteJSON marshals v and writes it as one JSON line.
func (t *ProcessTransport) WriteJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return t.WriteLine(data)
}

// Messages returns the channel of decoded stdout messages. The channel closes
// when the CLI exits or the transport is closed.
func (t *ProcessTransport) Messages() <-chan protocol.Message {
	return t.msgs
}

// StderrTail returns the tail of accumulated stderr output.
func (t *ProcessTransport) StderrTail() string {
	t.stderrMu.Lock()
	defer t.stderrMu.Unlock()
	return t.stderrBuf.String()
}

// PID returns the child process PID, or 0 if not started.
func (t *ProcessTransport) PID() int {
	if t.cmd != nil && t.cmd.Process != nil {
		return t.cmd.Process.Pid
	}
	return 0
}

// Close performs graceful shutdown: close stdin → CloseGraceMs → SIGTERM →
// KillGraceMs → SIGKILL, then removes the auth payload file.
func (t *ProcessTransport) Close() error {
	if !t.closed.CompareAndSwap(false, true) {
		return nil
	}
	t.ready.Store(false)
	// 1. close stdin
	if t.stdin != nil {
		_ = t.stdin.Close()
	}
	// 2. wait for graceful exit
	if t.cmd != nil && t.cmd.Process != nil {
		done := make(chan struct{})
		go func() {
			_ = t.cmd.Wait()
			close(done)
		}()
		grace := time.Duration(t.opts.CloseGraceMs) * time.Millisecond
		select {
		case <-done:
		case <-time.After(grace):
			_ = t.cmd.Process.Signal(os.Interrupt)
			select {
			case <-done:
			case <-time.After(time.Duration(t.killGraceMs) * time.Millisecond):
				_ = t.cmd.Process.Kill()
				<-done
			}
		}
	}
	t.cleanup()
	return nil
}

func (t *ProcessTransport) cleanup() {
	if t.authPayloadPath != "" {
		auth.RemovePayloadFile(t.authPayloadPath)
		t.authPayloadPath = ""
	}
}

// readStdout pumps stdout lines, parses each as a Message, and sends to msgs.
// The channel is closed here (not in Close) when stdout ends.
func (t *ProcessTransport) readStdout(r io.Reader) {
	defer close(t.msgs)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		msg, err := protocol.ParseMessage(line)
		if err != nil {
			continue // skip unparseable lines (stderr leakage, keep-alive whitespace)
		}
		if !t.send(msg) {
			return
		}
	}
}

// readStderr pumps stderr into a buffer and forwards to the handler.
func (t *ProcessTransport) readStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 4*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		t.stderrMu.Lock()
		if t.stderrBuf.Len() > 64*1024 {
			t.stderrBuf.Reset()
		}
		t.stderrBuf.WriteString(line)
		t.stderrBuf.WriteByte('\n')
		t.stderrMu.Unlock()
		if t.opts.StderrHandler != nil {
			t.opts.StderrHandler(line)
		}
		if isAuthExpiredLine(line) && t.opts.OnAuthExpired != nil {
			t.opts.OnAuthExpired()
		}
	}
}

func (t *ProcessTransport) send(msg protocol.Message) bool {
	if t.closed.Load() {
		return false
	}
	select {
	case t.msgs <- msg:
		return true
	default:
		// channel full: block (back-pressure).
		t.msgs <- msg
		return true
	}
}

func isAuthExpiredLine(line string) bool {
	low := strings.ToLower(line)
	return strings.Contains(low, "authentication") || strings.Contains(low, "unauthorized") || strings.Contains(low, "auth expired")
}

// ErrClosed is returned when writing to a closed transport.
var ErrClosed = errors.New("qoder: transport closed")

// buildArgs constructs the qoderclicn argv, mirroring the TS ProcessTransport.buildArgs.
func (t *ProcessTransport) buildArgs() []string {
	o := t.opts
	args := []string{"--print", "--output-format", "stream-json", "--input-format", "stream-json"}
	if o.SessionMirror {
		args = append(args, "--session-mirror")
	}
	if o.PersistSession != nil && !*o.PersistSession {
		args = append(args, "--no-session-persistence")
	}
	if o.Model != "" {
		args = append(args, "--model", o.Model)
	}
	if o.Proxy != "" {
		args = append(args, "--proxy", o.Proxy)
	}
	if o.Agent != "" {
		args = append(args, "--agent", o.Agent)
	}
	if o.Debug {
		args = append(args, "--debug")
	}
	if o.Resume != "" {
		args = append(args, "--resume", o.Resume)
	} else if o.Continue {
		args = append(args, "--continue")
	}
	if o.SessionID != "" {
		args = append(args, "--session-id", o.SessionID)
	}
	if o.PermissionMode != "" {
		if o.PermissionMode == protocol.PermissionYolo {
			args = append(args, "--yolo")
		} else {
			args = append(args, "--permission-mode", string(o.PermissionMode))
		}
	}
	if o.AllowDangerouslySkipPermissions && o.PermissionMode != protocol.PermissionYolo {
		args = append(args, "--dangerously-skip-permissions")
	}
	if o.IncludePartialMessages {
		args = append(args, "--include-partial-messages")
	}
	for _, tool := range o.AllowedTools {
		args = append(args, "--allowed-tools", tool)
	}
	for _, tool := range o.DisallowedTools {
		args = append(args, "--disallowed-tools", tool)
	}
	args = append(args, "--tools", "default")
	if o.CanUseTool {
		args = append(args, "--permission-prompt-tool", "stdio")
	}
	if len(o.McpServers) > 0 {
		mcpJSON, _ := json.Marshal(map[string]any{"mcpServers": o.McpServers})
		args = append(args, "--mcp-config", string(mcpJSON))
	}
	for _, name := range o.AllowedMcpServerNames {
		args = append(args, "--allowed-mcp-server-names", name)
	}
	if o.StrictMcpConfig {
		args = append(args, "--strict-mcp-config")
	}
	if len(o.Settings) > 0 {
		s, _ := json.Marshal(o.Settings)
		args = append(args, "--settings", string(s))
	}
	if len(o.SettingSources) > 0 {
		args = append(args, "--setting-sources", strings.Join(o.SettingSources, ","))
	}
	args = append(args, "--disable-builtin-skills")
	for _, dir := range o.AdditionalDirectories {
		args = append(args, "--add-dir", dir)
	}
	if o.MaxTurns != nil {
		args = append(args, "--max-turns", strconvItoa(*o.MaxTurns))
	}
	if o.ForkSession {
		args = append(args, "--fork-session")
	}
	for _, p := range o.Plugins {
		if p.Type == "local" {
			args = append(args, "--plugin-dir", p.Path)
		}
	}
	for name, val := range o.ExtraArgs {
		args = append(args, "--"+name)
		if val != nil {
			args = append(args, *val)
		}
	}
	return args
}

func strconvItoa(n int) string {
	return fmt.Sprintf("%d", n)
}

// buildEnv builds the child environment: base env + SDK markers, with
// NODE_OPTIONS removed and the VPC endpoint injected.
func (t *ProcessTransport) buildEnv() []string {
	base := t.opts.Env
	if base == nil {
		base = map[string]string{}
		for _, kv := range os.Environ() {
			if i := strings.IndexByte(kv, '='); i >= 0 {
				base[kv[:i]] = kv[i+1:]
			}
		}
	}
	delete(base, "NODE_OPTIONS")
	base["QODER_AGENT_SDK_ENTRYPOINT"] = "sdk-go"
	base["QODER_AGENT_SDK_VERSION"] = sdkVersion
	if t.authPayloadPath != "" {
		base["QODER_SDK_AUTH_PAYLOAD_FILE"] = t.authPayloadPath
	}
	if t.opts.VpcEndpoint != "" {
		base["QODERCN_VPC_ENDPOINT"] = t.opts.VpcEndpoint
		base["QODER_VPC_ENDPOINT"] = t.opts.VpcEndpoint
	}
	env := make([]string, 0, len(base))
	for k, v := range base {
		env = append(env, k+"="+v)
	}
	return env
}
