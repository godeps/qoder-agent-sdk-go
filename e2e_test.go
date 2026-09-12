package qodersdk_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	qodersdk "github.com/godeps/qoder-agent-sdk-go"
	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

// requireRealCLI skips unless a usable qodercli is available. Enable with
// QODER_SDK_E2E=1.
func requireRealCLI(t *testing.T) string {
	t.Helper()
	if os.Getenv("QODER_SDK_E2E") != "1" {
		t.Skip("set QODER_SDK_E2E=1 to run real-CLI integration tests")
	}
	path := os.Getenv("QODERCLI_PATH")
	if path == "" {
		p, err := exec.LookPath("qodercli")
		if err != nil {
			p, err = exec.LookPath("qoderclicn")
			if err != nil {
				t.Skip("qodercli/qoderclicn not on PATH")
			}
		}
		path = p
	}
	return path
}

func realOpts(cli string, t *testing.T) *qodersdk.Options {
	return qodersdk.NewOptions().
		WithPathToCLI(cli).
		WithCWD(t.TempDir()).
		WithAuth(auth.QodercliAuth())
}

func TestE2E_QueryOnce(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	msgs, err := qodersdk.Query(ctx, "Reply with exactly: QODER-E2E-OK", realOpts(cli, t))
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var result string
	var initLike int
	for m := range msgs {
		switch v := m.(type) {
		case *protocol.SystemMessage:
			initLike++
			_ = v
		case *protocol.ResultMessage:
			result = v.Result
		}
	}
	// NOTE: long-lived qodercli does not emit system/init; it leads with
	// artifacts_update / available_models_update. Only require that *some*
	// system messages and a result arrived.
	if initLike == 0 {
		t.Log("warning: no system messages observed")
	}
	if !strings.Contains(result, "QODER-E2E-OK") {
		t.Errorf("result = %q", result)
	}
	t.Logf("result=%q system_msgs=%d", result, initLike)
}

func TestE2E_Session_MultiTurn(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, realOpts(cli, t))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.Send("Remember the number 91. Reply exactly: STORED"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, _, err := sess.ReceiveResponse(120 * time.Second); err != nil {
		t.Fatalf("ReceiveResponse 1: %v", err)
	}
	if err := sess.Send("What number did I ask you to remember? Reply with only the number."); err != nil {
		t.Fatalf("Send 2: %v", err)
	}
	res, _, err := sess.ReceiveResponse(120 * time.Second)
	if err != nil {
		t.Fatalf("ReceiveResponse 2: %v", err)
	}
	if !strings.Contains(res.Result, "91") {
		t.Errorf("session lost context: %q", res.Result)
	}
	t.Logf("turn2=%q session=%s", res.Result, sess.SessionID())
}

func TestE2E_GetContextUsage_And_UsageInfo(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, realOpts(cli, t))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.Send("reply with one word: OK"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, _, err := sess.ReceiveResponse(120 * time.Second); err != nil {
		t.Fatalf("ReceiveResponse: %v", err)
	}

	cu, err := sess.GetContextUsage()
	if err != nil {
		t.Fatalf("GetContextUsage: %v", err)
	}
	if cu.Model == "" || cu.ContextWindow.UsedPercentage < 0 {
		t.Errorf("context usage looks empty: %+v", cu)
	}
	t.Logf("context: model=%s used=%.1f%% categories=%d", cu.Model, cu.ContextWindow.UsedPercentage, len(cu.Categories))

	ui, err := sess.GetUsageInfo()
	if err != nil {
		t.Fatalf("GetUsageInfo: %v", err)
	}
	if ui.Usage == nil {
		t.Skipf("usage info unavailable on this account: %s", ui.UsageError)
	}
	t.Logf("usage: type=%v pct=%v", ui.Usage.UserType, ui.Usage.TotalUsagePercentage)
}

func TestE2E_SetModel(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, realOpts(cli, t))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	models, err := qodersdk.ListModels(ctx, realOpts(cli, t))
	if err != nil || len(models) == 0 {
		t.Skipf("ListModels unavailable: %v", err)
	}
	// Establish the session with a first turn.
	if err := sess.Send("reply with one word: OK"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, _, err := sess.ReceiveResponse(120 * time.Second); err != nil {
		t.Fatalf("ReceiveResponse 1: %v", err)
	}

	cu, err := sess.GetContextUsage()
	if err != nil {
		t.Fatalf("GetContextUsage: %v", err)
	}
	current := cu.Model
	target := ""
	for _, m := range models {
		if m.Value != current {
			target = m.Value
			break
		}
	}
	if target == "" {
		t.Skip("no alternate model")
	}
	if err := sess.SetModel(target); err != nil {
		t.Fatalf("SetModel(%s): %v", target, err)
	}
	cu2, err := sess.GetContextUsage()
	if err != nil {
		t.Fatalf("GetContextUsage 2: %v", err)
	}
	if cu2.Model != target {
		t.Errorf("model after switch = %q, want %q", cu2.Model, target)
	}
	t.Logf("switched %s -> %s", current, cu2.Model)
}

func TestE2E_Interrupt(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	opts := realOpts(cli, t).WithPartialMessages(true)
	sess, err := qodersdk.NewSession(ctx, opts)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.Send("Count from 1 to 400, one number per line. Do not stop early."); err != nil {
		t.Fatalf("Send: %v", err)
	}

	// Wait for first assistant/partial output, then interrupt.
	interrupted := false
	deadline := time.After(90 * time.Second)
	for !interrupted {
		select {
		case m, ok := <-sess.Stream():
			if !ok {
				t.Fatal("stream closed early")
			}
			switch m.(type) {
			case *protocol.PartialAssistantMessage, *protocol.AssistantMessage:
				if _, err := sess.Interrupt(); err != nil {
					t.Logf("Interrupt error (may still abort): %v", err)
				}
				interrupted = true
			}
		case <-deadline:
			t.Fatal("no output within 90s")
		}
	}

	res, _, err := sess.ReceiveResponse(120 * time.Second)
	if err != nil {
		t.Fatalf("no result after interrupt: %v", err)
	}
	t.Logf("post-interrupt: subtype=%s turns=%d", res.Subtype, res.NumTurns)
	if res.Subtype == "success" && !res.IsError && strings.Count(res.Result, "\n") > 300 {
		t.Errorf("interrupt ineffective: %d lines", strings.Count(res.Result, "\n")+1)
	}
}

func TestE2E_Hooks_Block(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	dir := t.TempDir()
	canary := dir + "/hook_canary.txt"

	var mu sync.Mutex
	fired := 0
	opts := realOpts(cli, t).
		WithHooks(map[protocol.HookEvent][]protocol.HookSpec{
			protocol.HookPreToolUse: {{Matcher: ""}}, // all tools
		}).
		WithHookCallback(func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error) {
			mu.Lock()
			fired++
			mu.Unlock()
			cont := false
			return protocol.HookJSONOutput{
				Continue: &cont,
				Decision: "block",
				Reason:   "all tools disabled by E2E policy; reply exactly BLOCKED-BY-HOOK and stop",
			}, nil
		})

	msgs, err := qodersdk.Query(ctx,
		"Create a file hook_canary.txt containing CANARY. If every attempt is blocked, reply exactly BLOCKED-BY-HOOK and stop.", opts)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	for range msgs {
	}

	mu.Lock()
	n := fired
	mu.Unlock()
	if n == 0 {
		t.Fatal("hook never fired")
	}
	if _, err := os.Stat(canary); err == nil {
		t.Errorf("hook block did NOT prevent file creation")
	}
	t.Logf("hook fired %d time(s); canary correctly absent", n)
}

func TestE2E_CanUseTool_Deny(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	var mu sync.Mutex
	var prompted []string
	opts := realOpts(cli, t).
		WithPermissionMode(protocol.PermissionDefault).
		WithCanUseTool(func(ctx context.Context, req *protocol.CanUseToolRequest) (protocol.PermissionResult, error) {
			mu.Lock()
			prompted = append(prompted, req.ToolName)
			mu.Unlock()
			return protocol.PermissionResult{Behavior: protocol.PermissionDeny, Message: "denied by SDK test"}, nil
		})

	msgs, err := qodersdk.Query(ctx,
		"Use bash to create a file named denied_canary.txt in the current directory containing X. If the tool is denied, reply exactly BLOCKED and stop.", opts)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var result string
	for m := range msgs {
		if r, ok := m.(*protocol.ResultMessage); ok {
			result = r.Result
		}
	}
	mu.Lock()
	n := len(prompted)
	names := strings.Join(prompted, ",")
	mu.Unlock()
	if n == 0 {
		t.Skip("CLI did not route a permission prompt")
	}
	t.Logf("prompted=%s result=%q", names, result)
}

func TestE2E_SdkMCP(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	const magic = "QODER-GO-MCP-42"
	var mu sync.Mutex
	methods := map[string]int{}

	handler := func(ctx context.Context, server string, msg json.RawMessage) (json.RawMessage, error) {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.Unmarshal(msg, &req); err != nil {
			return nil, err
		}
		mu.Lock()
		methods[req.Method]++
		mu.Unlock()
		result := func(res any) (json.RawMessage, error) {
			return json.Marshal(map[string]any{"jsonrpc": "2.0", "id": json.RawMessage(req.ID), "result": res})
		}
		switch req.Method {
		case "initialize":
			return result(map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": server, "version": "0.1"},
			})
		case "tools/list":
			return result(map[string]any{"tools": []map[string]any{{
				"name":        "get_magic",
				"description": "Returns the magic string. Call when asked for magic.",
				"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
			}}})
		case "tools/call":
			return result(map[string]any{
				"content": []map[string]any{{"type": "text", "text": magic}},
				"isError": false,
			})
		}
		return nil, nil
	}

	opts := realOpts(cli, t).
		WithPermissionMode(protocol.PermissionBypassPermissions).
		WithSdkMcpServers("go-e2e-mcp").
		WithMcpServers(map[string]protocol.McpServerConfig{
			"go-e2e-mcp": protocol.NewMcpSdkConfig("go-e2e-mcp"),
		}).
		WithMcpMessageHandler(handler)

	msgs, err := qodersdk.Query(ctx,
		"Call the get_magic tool from the go-e2e-mcp MCP server exactly once, then reply with ONLY its output.", opts)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	var result string
	for m := range msgs {
		if r, ok := m.(*protocol.ResultMessage); ok {
			result = r.Result
		}
	}
	mu.Lock()
	calls := methods["tools/call"]
	seen := map[string]int{}
	for k, v := range methods {
		seen[k] = v
	}
	mu.Unlock()
	t.Logf("mcp methods=%v result=%q", seen, result)
	if calls == 0 {
		t.Fatalf("get_magic never called via mcp_message")
	}
	if !strings.Contains(result, magic) {
		t.Errorf("result missing tool output %q", magic)
	}
}

func TestE2E_PluginManagement(t *testing.T) {
	cli := requireRealCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	plugins, err := qodersdk.ListPlugins(ctx, &qodersdk.PluginOptions{PathToCLI: cli})
	if err != nil {
		t.Fatalf("ListPlugins: %v", err)
	}
	t.Logf("installed plugins: %d", len(plugins))

	// Validate a synthetic minimal plugin directory (static; executes nothing).
	dir := t.TempDir()
	pdir := dir + "/test-plugin"
	if err := os.MkdirAll(pdir+"/.qoder-plugin", 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"e2e-test-plugin","version":"0.0.1","description":"validate probe"}`
	if err := os.WriteFile(pdir+"/.qoder-plugin/plugin.json", []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := qodersdk.ValidatePlugin(ctx, pdir, &qodersdk.PluginOptions{PathToCLI: cli})
	if err != nil {
		t.Skipf("validate unsupported by this CLI: %v", err)
	}
	t.Logf("validate: valid=%v errors=%d warnings=%d diag=%d", report.Valid, report.Summary.ErrorCount, report.Summary.WarningCount, len(report.Diagnostics))
	// A bare manifest with no content is expected to be reported invalid —
	// the point of the test is that the CLI ran the static validation and the
	// SDK parsed the structured report.
	if report.Validator.Name == "" {
		t.Error("report missing validator info")
	}
}
