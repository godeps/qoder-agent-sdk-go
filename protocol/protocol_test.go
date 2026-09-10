package protocol

import (
	"encoding/json"
	"testing"
)

func TestParseMessage_Assistant(t *testing.T) {
	line := []byte(`{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"hello"}]},"parent_tool_use_id":null,"uuid":"u1","session_id":"s1"}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if m.MessageType() != "assistant" {
		t.Fatalf("type = %q", m.MessageType())
	}
	a, ok := m.(*AssistantMessage)
	if !ok {
		t.Fatalf("got %T", m)
	}
	if len(a.Message.Content) != 1 || a.Message.Content[0].Text != "hello" {
		t.Fatalf("content = %+v", a.Message.Content)
	}
}

func TestParseMessage_Result(t *testing.T) {
	line := []byte(`{"type":"result","subtype":"success","duration_ms":10,"duration_api_ms":5,"is_error":false,"num_turns":1,"stop_reason":"end_turn","total_cost_usd":0,"usage":{"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":0},"cache_creation_input_tokens":0,"cache_read_input_tokens":0,"inference_geo":"","input_tokens":10,"iterations":[],"output_tokens":5,"server_tool_use":{"web_fetch_requests":0,"web_search_requests":0},"service_tier":"","speed":""},"modelUsage":{},"permission_denials":[],"uuid":"u1","session_id":"s1"}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	r, ok := m.(*ResultMessage)
	if !ok {
		t.Fatalf("got %T", m)
	}
	if !r.IsSuccess() {
		t.Fatal("not success")
	}
	if r.Usage.InputTokens != 10 {
		t.Fatalf("input tokens = %d", r.Usage.InputTokens)
	}
}

func TestParseMessage_SystemInit(t *testing.T) {
	line := []byte(`{"type":"system","subtype":"init","protocol_version":"1.4.0","qodercli_version":"1.1.49","cwd":"/tmp","model":"m","permissionMode":"default","tools":[],"mcp_servers":[],"slash_commands":[],"output_style":"default","skills":[],"plugins":[],"capabilities":[],"uuid":"u1","session_id":"s1"}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	s, ok := m.(*SystemMessage)
	if !ok {
		t.Fatalf("got %T", m)
	}
	if s.Subtype != "init" {
		t.Fatalf("subtype = %q", s.Subtype)
	}
	if s.ProtocolVersion != "1.4.0" {
		t.Fatalf("protocol = %q", s.ProtocolVersion)
	}
}

func TestParseMessage_ControlRequest_CanUseTool(t *testing.T) {
	line := []byte(`{"type":"control_request","request_id":"r1","session_id":"s1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"cmd":"ls"},"tool_use_id":"tu1"}}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	cr, ok := m.(*ControlRequest)
	if !ok {
		t.Fatalf("got %T", m)
	}
	inner, err := cr.Inner()
	if err != nil {
		t.Fatalf("Inner: %v", err)
	}
	cut, ok := inner.(CanUseToolRequest)
	if !ok {
		t.Fatalf("inner type %T", inner)
	}
	if cut.ToolName != "Bash" {
		t.Fatalf("tool = %q", cut.ToolName)
	}
	if cut.ToolUseID != "tu1" {
		t.Fatalf("tool_use_id = %q", cut.ToolUseID)
	}
}

func TestParseMessage_ControlResponse(t *testing.T) {
	line := []byte(`{"type":"control_response","session_id":"s1","response":{"subtype":"success","request_id":"r1","response":{"model":"x"}}}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	cr, ok := m.(*ControlResponse)
	if !ok {
		t.Fatalf("got %T", m)
	}
	if !cr.IsSuccess() {
		t.Fatal("not success")
	}
	s, err := cr.Success()
	if err != nil || s == nil {
		t.Fatalf("Success: %v %v", s, err)
	}
	if s.RequestID != "r1" {
		t.Fatalf("request_id = %q", s.RequestID)
	}
}

func TestParseMessage_Unknown(t *testing.T) {
	line := []byte(`{"type":"future_kind","foo":"bar"}`)
	m, err := ParseMessage(line)
	if err != nil {
		t.Fatalf("ParseMessage: %v", err)
	}
	if m.MessageType() != "future_kind" {
		t.Fatalf("type = %q", m.MessageType())
	}
	u, ok := m.(*UnknownMessage)
	if !ok {
		t.Fatalf("got %T", m)
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(u.Raw, &probe); err != nil {
		t.Fatal(err)
	}
	if string(probe["foo"]) != `"bar"` {
		t.Fatalf("foo = %s", probe["foo"])
	}
}
