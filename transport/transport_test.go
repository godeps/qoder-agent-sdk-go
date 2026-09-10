package transport

import (
	"os"
	"strings"
	"testing"

	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

func TestBuildArgs_StreamJSONPreamble(t *testing.T) {
	tr := NewProcessTransport(TransportOptions{Model: "m1", IncludePartialMessages: true})
	args := tr.buildArgs()
	want := []string{"--print", "--output-format", "stream-json", "--input-format", "stream-json"}
	for i, w := range want {
		if args[i] != w {
			t.Fatalf("arg[%d] = %q, want %q (args=%v)", i, args[i], w, args)
		}
	}
	if !contains(args, "--include-partial-messages") {
		t.Error("missing --include-partial-messages")
	}
	if !contains(args, "--disable-builtin-skills") {
		t.Error("missing --disable-builtin-skills")
	}
	if i := indexOf(args, "--model"); i < 0 || args[i+1] != "m1" {
		t.Fatalf("model args: %v", args)
	}
	if i := indexOf(args, "--tools"); i < 0 || args[i+1] != "default" {
		t.Fatalf("tools args: %v", args)
	}
}

func TestBuildArgs_PermissionPromptToolStdio(t *testing.T) {
	tr := NewProcessTransport(TransportOptions{CanUseTool: true})
	args := tr.buildArgs()
	i := indexOf(args, "--permission-prompt-tool")
	if i < 0 || args[i+1] != "stdio" {
		t.Fatalf("permission-prompt-tool: %v", args)
	}
}

func TestBuildArgs_McpConfigJSONString(t *testing.T) {
	mcp := map[string]protocol.McpServerConfig{"srv": protocol.NewMcpHTTPConfig("http://localhost:7777", nil)}
	tr := NewProcessTransport(TransportOptions{McpServers: mcp})
	args := tr.buildArgs()
	i := indexOf(args, "--mcp-config")
	if i < 0 {
		t.Fatalf("no --mcp-config: %v", args)
	}
	jsonStr := args[i+1]
	if !strings.Contains(jsonStr, "mcpServers") || !strings.Contains(jsonStr, "localhost:7777") {
		t.Fatalf("mcp-config json: %s", jsonStr)
	}
}

func TestBuildArgs_ResumeAndYolo(t *testing.T) {
	tr := NewProcessTransport(TransportOptions{
		Resume:         "session-abc",
		PermissionMode: protocol.PermissionYolo,
	})
	args := tr.buildArgs()
	if i := indexOf(args, "--resume"); i < 0 || args[i+1] != "session-abc" {
		t.Fatalf("resume args: %v", args)
	}
	if !contains(args, "--yolo") {
		t.Fatalf("missing --yolo: %v", args)
	}
	if contains(args, "--permission-mode") {
		t.Fatalf("should not have --permission-mode with yolo: %v", args)
	}
}

func TestBuildEnv_MarkersAndAuthPayload(t *testing.T) {
	dir, err := os.MkdirTemp("", "qoder-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	payloadPath := dir + "/payload.json"
	if err := os.WriteFile(payloadPath, []byte(`{"type":"accessToken","accessToken":"t"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tr := NewProcessTransport(TransportOptions{
		Auth:        auth.AccessToken("t"),
		VpcEndpoint: "https://vpc.example",
	})
	tr.authPayloadPath = payloadPath
	env := envMap(tr.buildEnv())
	if env["QODER_AGENT_SDK_ENTRYPOINT"] != "sdk-go" {
		t.Errorf("entrypoint=%q", env["QODER_AGENT_SDK_ENTRYPOINT"])
	}
	if env["QODER_AGENT_SDK_VERSION"] == "" {
		t.Error("missing QODER_AGENT_SDK_VERSION")
	}
	if env["QODER_SDK_AUTH_PAYLOAD_FILE"] != payloadPath {
		t.Errorf("payload file=%q", env["QODER_SDK_AUTH_PAYLOAD_FILE"])
	}
	if env["QODERCN_VPC_ENDPOINT"] != "https://vpc.example" {
		t.Errorf("vpc=%q", env["QODERCN_VPC_ENDPOINT"])
	}
	if _, ok := env["NODE_OPTIONS"]; ok {
		t.Error("NODE_OPTIONS not removed")
	}
}

func TestAuthPayloadFileRoundTrip(t *testing.T) {
	path, err := auth.WritePayloadFile(auth.AccessToken("tok123"))
	if err != nil {
		t.Fatal(err)
	}
	defer auth.RemovePayloadFile(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "tok123") {
		t.Fatalf("payload missing token: %s", data)
	}
}

func contains(s []string, v string) bool {
	return indexOf(s, v) >= 0
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	return m
}
