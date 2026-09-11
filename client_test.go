package qodersdk_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	qodersdk "github.com/godeps/qoder-agent-sdk-go"
	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

// buildFakeCLI compiles internal/fakecli into a temp binary and returns its path.
func buildFakeCLI(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "fake-qoderclicn")
	cmd := exec.Command("go", "build", "-o", bin, "./internal/fakecli")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fakecli: %v\n%s", err, out)
	}
	return bin
}

func TestQuery_FakeCLI(t *testing.T) {
	cli := buildFakeCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	opts := qodersdk.NewOptions().
		WithPathToCLI(cli).
		WithCWD(t.TempDir()).
		WithAuth(auth.AccessToken("test-token")).
		WithCanUseTool(func(ctx context.Context, req *protocol.CanUseToolRequest) (protocol.PermissionResult, error) {
			return protocol.PermissionResult{Behavior: protocol.PermissionAllow, ToolUseID: req.ToolUseID}, nil
		})

	msgs, err := qodersdk.Query(ctx, "hello", opts)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	var gotAssistant, gotResult bool
	for m := range msgs {
		switch v := m.(type) {
		case *protocol.AssistantMessage:
			gotAssistant = true
			if len(v.Message.Content) == 0 || v.Message.Content[0].Text == "" {
				t.Errorf("empty assistant text")
			}
		case *protocol.ResultMessage:
			gotResult = true
			if !v.IsSuccess() {
				t.Errorf("result not success: %s", v.Subtype)
			}
		}
	}
	if !gotAssistant {
		t.Error("did not receive an assistant message")
	}
	if !gotResult {
		t.Error("did not receive a result message")
	}
}

func TestQuery_AuthNotConfigured(t *testing.T) {
	_, err := qodersdk.Query(context.Background(), "hi", qodersdk.NewOptions())
	if err != qodersdk.ErrAuthNotConfigured {
		t.Fatalf("err = %v, want ErrAuthNotConfigured", err)
	}
}

func TestListModels_FakeCLI(t *testing.T) {
	cli := buildFakeCLI(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	opts := qodersdk.NewOptions().
		WithPathToCLI(cli).
		WithCWD(t.TempDir()).
		WithAuth(auth.AccessToken("test-token"))
	models, err := qodersdk.ListModels(ctx, opts)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}
	if models[0].Value != "fake-efficient" {
		t.Errorf("models[0].Value = %q", models[0].Value)
	}
	if models[1].Value != "fake-performance" {
		t.Errorf("models[1].Value = %q", models[1].Value)
	}
}

func TestListModels_RealQodercli(t *testing.T) {
	if testing.Short() {
		t.Skip("real qoderclicn")
	}
	if os.Getenv("QODERCLI_PATH") == "" {
		if _, err := exec.LookPath("qoderclicn"); err != nil {
			if _, err2 := exec.LookPath("qodercli"); err2 != nil {
				t.Skip("qoderclicn/qodercli not found (set QODERCLI_PATH)")
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	opts := qodersdk.NewOptions().WithAuth(auth.QodercliAuth())
	if p := os.Getenv("QODERCLI_PATH"); p != "" {
		opts = opts.WithPathToCLI(p)
	}
	models, err := qodersdk.ListModels(ctx, opts)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(models) == 0 {
		t.Fatal("no models returned")
	}
	for _, m := range models {
		t.Logf("model: %s (%s) default=%v reasoning=%v", m.Value, m.DisplayName, m.IsDefault, m.IsReasoning)
	}
}
