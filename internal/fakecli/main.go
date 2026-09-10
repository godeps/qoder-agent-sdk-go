// fake-qoderclicn is a scripted stand-in for the qoderclicn CLI used by tests.
// It emits a system/init handshake line, then drives a minimal conversation:
// on the initialize control request it replies with a success response, and
// on the first user message it emits an assistant message followed by a
// success result, then exits when stdin closes.
//
// It is a separate `package main` so the SDK test suite can build and spawn it
// as a real subprocess.
package main

import (
	"bufio"
	"encoding/json"
	"os"
)

func writeLine(v any) {
	data, _ := json.Marshal(v)
	_, _ = os.Stdout.Write(append(data, '\n'))
}

func main() {
	// 1. system/init handshake
	writeLine(map[string]any{
		"type": "system", "subtype": "init",
		"protocol_version": "1.4.0",
		"qodercli_version": "fakecli-1.0",
		"cwd":              ".",
		"model":            "fake-model",
		"apiKeySource":     "temporary",
		"permissionMode":   "default",
		"tools":            []string{},
		"mcp_servers":      []any{},
		"slash_commands":   []any{},
		"output_style":     "default",
		"skills":           []string{},
		"plugins":          []any{},
		"capabilities":     []string{},
		"uuid":             "init-uuid",
		"session_id":       "fake-session",
	})

	// 2. read stdin and drive the conversation
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	initialized := false
	for scanner.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		switch msg["type"] {
		case "control_request":
			reqID, _ := msg["request_id"].(string)
			req, _ := msg["request"].(map[string]any)
			reqType, _ := req["type"].(string)
			// success response
			writeLine(map[string]any{
				"type":       "control_response",
				"session_id": "fake-session",
				"response": map[string]any{
					"subtype":    "success",
					"request_id": reqID,
				},
			})
			if reqType == "initialize" && !initialized {
				initialized = true
			}
		case "user":
			// reply with an assistant message + success result
			writeLine(assistantMessage("asst-1", "Hello from fake qoderclicn"))
			writeLine(resultSuccess("res-1"))
		}
	}
}

func assistantMessage(uuid, text string) map[string]any {
	return map[string]any{
		"type":               "assistant",
		"message":            map[string]any{"role": "assistant", "content": []map[string]any{{"type": "text", "text": text}}},
		"parent_tool_use_id": nil,
		"uuid":               uuid,
		"session_id":         "fake-session",
	}
}

func resultSuccess(uuid string) map[string]any {
	return map[string]any{
		"type": "result", "subtype": "success",
		"duration_ms": 10, "duration_api_ms": 5, "is_error": false, "num_turns": 1,
		"stop_reason": "end_turn", "total_cost_usd": 0,
		"usage": map[string]any{
			"cache_creation":              map[string]any{"ephemeral_1h_input_tokens": 0, "ephemeral_5m_input_tokens": 0},
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens":     0,
			"inference_geo":               "",
			"input_tokens":                10,
			"iterations":                  []any{},
			"output_tokens":               5,
			"server_tool_use":             map[string]any{"web_fetch_requests": 0, "web_search_requests": 0},
			"service_tier":                "",
			"speed":                       "",
		},
		"modelUsage":         map[string]any{},
		"permission_denials": []any{},
		"uuid":               uuid,
		"session_id":         "fake-session",
	}
}
