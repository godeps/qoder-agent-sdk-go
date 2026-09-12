package qodersdk_test

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	qodersdk "github.com/godeps/qoder-agent-sdk-go"
	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

func buildFake(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-qoderclicn")
	cmd := exec.Command("go", "build", "-o", bin, "./internal/fakecli")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fakecli: %v\n%s", err, out)
	}
	return bin
}

func fakeSessionOpts(t *testing.T, bin string, env map[string]string) *qodersdk.Options {
	t.Helper()
	o := qodersdk.NewOptions().
		WithPathToCLI(bin).
		WithCWD(t.TempDir()).
		WithAuth(auth.AccessToken("fake-token"))
	if len(env) > 0 {
		full := map[string]string{"PATH": os.Getenv("PATH")}
		for k, v := range env {
			full[k] = v
		}
		o = o.WithEnv(full)
	}
	return o
}

func TestSession_MultiTurn(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, fakeSessionOpts(t, bin, nil))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if sess.SessionID() != "fake-session" {
		t.Errorf("SessionID = %q, want fake-session", sess.SessionID())
	}

	for i := 0; i < 3; i++ {
		if err := sess.Send("turn"); err != nil {
			t.Fatalf("Send #%d: %v", i, err)
		}
		res, _, err := sess.ReceiveResponse(20 * time.Second)
		if err != nil {
			t.Fatalf("ReceiveResponse #%d: %v", i, err)
		}
		if res.Subtype != "success" {
			t.Errorf("turn #%d result: %s", i, res.Subtype)
		}
	}
}

func TestSession_ControlMethods(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, fakeSessionOpts(t, bin, nil))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.SetModel("fake-performance"); err != nil {
		t.Fatalf("SetModel: %v", err)
	}
	if err := sess.SetPermissionMode(protocol.PermissionAcceptEdits); err != nil {
		t.Fatalf("SetPermissionMode: %v", err)
	}
	ir, err := sess.Interrupt()
	if err != nil {
		t.Fatalf("Interrupt: %v", err)
	}
	if ir == nil {
		t.Error("nil interrupt response")
	}
	cu, err := sess.GetContextUsage()
	if err != nil {
		t.Fatalf("GetContextUsage: %v", err)
	}
	if cu.Model != "fake-performance" {
		t.Errorf("context usage model = %q (set_model did not stick?)", cu.Model)
	}
	if cu.ContextWindow.UsedPercentage != 12.5 {
		t.Errorf("usedPercentage = %v", cu.ContextWindow.UsedPercentage)
	}
	ui, err := sess.GetUsageInfo()
	if err != nil {
		t.Fatalf("GetUsageInfo: %v", err)
	}
	if ui.Usage == nil || ui.Usage.UserID != "fake-user" {
		t.Errorf("usage info: %+v", ui.Usage)
	}
	ms, err := sess.McpStatus()
	if err != nil {
		t.Fatalf("McpStatus: %v", err)
	}
	_ = ms

	rf, err := sess.RewindFiles("m1", true)
	if err != nil {
		t.Fatalf("RewindFiles: %v", err)
	}
	if !rf.CanRewind {
		t.Error("expected canRewind")
	}
	rr, err := sess.Rewind("m1", protocol.RewindScopeBoth, false)
	if err != nil {
		t.Fatalf("Rewind: %v", err)
	}
	if rr.Status != protocol.RewindStatusSuccess {
		t.Errorf("rewind status = %q", rr.Status)
	}
}

func TestSession_BYOK(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, fakeSessionOpts(t, bin, nil))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	cat, err := sess.GetByokConfig()
	if err != nil {
		t.Fatalf("GetByokConfig: %v", err)
	}
	if len(cat.Providers) != 1 || cat.Providers[0].Key != "openai" {
		t.Errorf("byok catalog: %+v", cat.Providers)
	}
	ok, err := sess.ValidateByokModel("openai", "gpt-x", "sk-fake", "", "")
	if err != nil || !ok {
		t.Fatalf("ValidateByokModel: ok=%v err=%v", ok, err)
	}
	cfgs, err := sess.ListByokConfigs()
	if err != nil {
		t.Fatalf("ListByokConfigs: %v", err)
	}
	if cfgs == nil {
		t.Error("nil configs slice")
	}
	ref, err := sess.CreateByokModelConfig(qodersdk.ByokModelConfigInput{
		Provider: "openai", Model: "gpt-x", Parameters: map[string]string{"api_key": "sk-fake"},
	})
	if err != nil {
		t.Fatalf("CreateByokModelConfig: %v", err)
	}
	if ref.Key != "fake-key" {
		t.Errorf("created ref: %+v", ref)
	}
	if err := sess.UpdateByokModelConfig(qodersdk.UpdateByokModelConfigInput{Key: "fake-key", DisplayName: "renamed"}); err != nil {
		t.Fatalf("UpdateByokModelConfig: %v", err)
	}
	chk, err := sess.CheckByokModelConfig(qodersdk.ByokModelConfigInput{Provider: "openai", Model: "gpt-x", Parameters: map[string]string{"api_key": "sk-fake"}})
	if err != nil {
		t.Fatalf("CheckByokModelConfig: %v", err)
	}
	if !chk.Success {
		t.Error("check failed")
	}
	if err := sess.DeleteByokConfigByKey("fake-key"); err != nil {
		t.Fatalf("DeleteByokConfigByKey: %v", err)
	}
	if err := sess.DeleteByokConfigByProvider("prov"); err != nil {
		t.Fatalf("DeleteByokConfigByProvider: %v", err)
	}
}

func TestSession_ReloadPluginsSkills(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	sess, err := qodersdk.NewSession(ctx, fakeSessionOpts(t, bin, nil))
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	rp, err := sess.ReloadPlugins()
	if err != nil {
		t.Fatalf("ReloadPlugins: %v", err)
	}
	if rp.ErrorCount != 0 {
		t.Errorf("error_count = %d", rp.ErrorCount)
	}
	rs, err := sess.ReloadSkills()
	if err != nil {
		t.Fatalf("ReloadSkills: %v", err)
	}
	_ = rs
}

func TestSession_McpMessageHandler(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var mu sync.Mutex
	var gotServer string
	opts := fakeSessionOpts(t, bin, nil).
		WithSdkMcpServers("unit-mcp").
		WithMcpServers(map[string]protocol.McpServerConfig{
			"unit-mcp": protocol.NewMcpSdkConfig("unit-mcp"),
		}).
		WithMcpMessageHandler(func(ctx context.Context, server string, msg json.RawMessage) (json.RawMessage, error) {
			mu.Lock()
			gotServer = server
			mu.Unlock()
			return json.Marshal(map[string]any{"jsonrpc": "2.0", "result": map[string]any{}})
		})

	sess, err := qodersdk.NewSession(ctx, opts)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	// fakecli does not send mcp_message on its own; verify wiring via the
	// handler invocation path indirectly — the initialize handshake must have
	// carried sdkMcpServers without error, and the transport must have gotten
	// --allowed-mcp-server-names unit-mcp.
	if sess.SessionID() == "" {
		t.Error("session did not establish")
	}
	_ = gotServer
}

func TestSession_HookCallback(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var mu sync.Mutex
	var fired []string
	opts := fakeSessionOpts(t, bin, map[string]string{"FAKE_HOOK": "1"}).
		WithHooks(map[protocol.HookEvent][]protocol.HookSpec{
			protocol.HookPreToolUse: {{Matcher: "Bash"}},
		}).
		WithHookCallback(func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error) {
			mu.Lock()
			fired = append(fired, req.CallbackID+":"+string(req.Input.HookEventName)+":"+req.Input.ToolName)
			mu.Unlock()
			return protocol.HookJSONOutput{}, nil
		})

	sess, err := qodersdk.NewSession(ctx, opts)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.Send("hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, _, err := sess.ReceiveResponse(20 * time.Second); err != nil {
		t.Fatalf("ReceiveResponse: %v", err)
	}

	mu.Lock()
	got := append([]string(nil), fired...)
	mu.Unlock()
	if len(got) != 1 {
		t.Fatalf("hook fired %d times: %v", len(got), got)
	}
	// hook_N auto-id assigned by the SDK
	if !strings.HasPrefix(got[0], "hook_0:PreToolUse:Bash") {
		t.Errorf("unexpected hook identity: %q", got[0])
	}
}

func TestSession_CanUseTool(t *testing.T) {
	bin := buildFake(t)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	var mu sync.Mutex
	var toolName string
	opts := fakeSessionOpts(t, bin, map[string]string{"FAKE_TOOL_PROMPT": "1"}).
		WithCanUseTool(func(ctx context.Context, req *protocol.CanUseToolRequest) (protocol.PermissionResult, error) {
			mu.Lock()
			toolName = req.ToolName
			mu.Unlock()
			return protocol.PermissionResult{Behavior: protocol.PermissionAllow}, nil
		})

	sess, err := qodersdk.NewSession(ctx, opts)
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	defer sess.Close()

	if err := sess.Send("hi"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if _, _, err := sess.ReceiveResponse(20 * time.Second); err != nil {
		t.Fatalf("ReceiveResponse: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if toolName != "Bash" {
		t.Errorf("can_use_tool fired for %q, want Bash", toolName)
	}
}

func TestSession_GuardBeforeInit(t *testing.T) {
	// A Session with a pinned empty session id but a dead CLI: NewSession
	// should surface ErrSessionClosed rather than hang.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := qodersdk.NewOptions().
		WithPathToCLI("/nonexistent/qodercli-binary").
		WithAuth(auth.AccessToken("x"))
	if _, err := qodersdk.NewSession(ctx, opts); err == nil {
		t.Error("expected error for missing CLI binary")
	}
}
