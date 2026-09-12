// fake-qoderclicn is a scripted stand-in for the qoderclicn/qodercli CLI used
// by tests. It emits a system/init handshake, answers control requests
// (initialize, set_model, interrupt, get_context_usage, get_usage_info,
// mcp_status, mcp_message, byok, reload_*, end_session), and drives a
// multi-turn conversation: each user message yields an assistant message plus
// a success result. Optional hook and permission round-trips are driven by env
// knobs so the test suite can exercise those paths against a real subprocess.
//
// Env knobs:
//   - FAKE_TEXT: assistant reply text (default "Hello from fake qoderclicn")
//   - FAKE_TOOL_PROMPT=1: before the first result, emit a tool_use and a
//     can_use_tool control request; record the SDK decision on stderr.
//   - FAKE_HOOK=1: before the first result, emit a hook_callback control
//     request for PreToolUse and record the SDK's continue/decision on stderr.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

func writeLine(v any) {
	data, _ := json.Marshal(v)
	_, _ = os.Stdout.Write(append(data, '\n'))
}

func main() {
	text := os.Getenv("FAKE_TEXT")
	if text == "" {
		text = "Hello from fake qoderclicn"
	}
	toolPrompt := os.Getenv("FAKE_TOOL_PROMPT") == "1"
	hookPrompt := os.Getenv("FAKE_HOOK") == "1"

	// 1. system/init handshake
	writeLine(map[string]any{
		"type": "system", "subtype": "init",
		"protocol_version": "1.4.0",
		"qodercli_version": "fakecli-1.0",
		"cwd":              ".",
		"model":            "fake-model",
		"apiKeySource":     "temporary",
		"permissionMode":   "default",
		"tools":            []string{"Bash"},
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
	turn := 0
	currentModel := "fake-efficient"
	for {
		msg := nextMessage(scanner)
		if msg == nil {
			return
		}
		switch msg["type"] {
		case "control_request":
			reqID, _ := msg["request_id"].(string)
			req, _ := msg["request"].(map[string]any)
			reqType, _ := req["type"].(string)
			if reqType == "" {
				reqType, _ = req["subtype"].(string)
			}
			resp := controlResponse(reqType, req, &currentModel)
			writeLine(map[string]any{
				"type":       "control_response",
				"session_id": "fake-session",
				"response": map[string]any{
					"subtype":    "success",
					"request_id": reqID,
					"response":   resp,
				},
			})
			if reqType == "initialize" && !initialized {
				initialized = true
				writeLine(availableModelsUpdate())
			}
			if reqType == "end_session" {
				return
			}
		case "user":
			turn++
			if toolPrompt && turn == 1 {
				writeLine(assistantToolUse())
				writeLine(map[string]any{
					"type": "control_request", "request_id": "perm_1",
					"request": map[string]any{
						"subtype": "can_use_tool", "tool_name": "Bash",
						"input":       map[string]any{"command": "echo hi"},
						"tool_use_id": "call_1",
					},
				})
				// Block until the SDK answers, like the real CLI does.
				drainUntilResponse(scanner, "perm_1")
			}
			if hookPrompt && turn == 1 {
				writeLine(map[string]any{
					"type": "control_request", "request_id": "hook_1",
					"request": map[string]any{
						"subtype": "hook_callback", "callback_id": "hook_0",
						"tool_use_id": "call_h",
						"input": map[string]any{
							"hook_event_name": "PreToolUse", "tool_name": "Bash",
							"session_id": "fake-session", "cwd": ".",
						},
					},
				})
				drainUntilResponse(scanner, "hook_1")
			}
			writeLine(assistantMessage(fmt.Sprintf("asst-%d", turn), text, currentModel))
			writeLine(resultSuccess(fmt.Sprintf("res-%d", turn)))
		}
	}
}

// pending holds messages read while blocking on a control response.
var pending []map[string]any

// nextMessage returns the next stdin message, preferring the pending queue.
// Returns nil at EOF.
func nextMessage(scanner *bufio.Scanner) map[string]any {
	if len(pending) > 0 {
		m := pending[0]
		pending = pending[1:]
		return m
	}
	for scanner.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		return msg
	}
	return nil
}

// drainUntilResponse blocks until a control_response for reqID arrives.
// Unrelated stdin lines are queued into pending for the main loop.
func drainUntilResponse(scanner *bufio.Scanner, reqID string) {
	for scanner.Scan() {
		var msg map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg["type"] == "control_response" {
			resp, _ := msg["response"].(map[string]any)
			if resp != nil && resp["request_id"] == reqID {
				return
			}
		}
		pending = append(pending, msg)
	}
}

// controlResponse builds the typed success payload for a control request.
func controlResponse(reqType string, req map[string]any, currentModel *string) map[string]any {
	switch reqType {
	case "initialize":
		return map[string]any{
			"commands":       []any{},
			"models":         []map[string]any{{"id": "fake-efficient", "name": "Efficient"}, {"id": "fake-performance", "name": "Performance"}},
			"currentModelId": *currentModel,
		}
	case "set_model":
		if m, ok := req["model"].(string); ok {
			prev := *currentModel
			*currentModel = m
			return map[string]any{"model": m, "previous_model": prev}
		}
		return map[string]any{}
	case "interrupt":
		return map[string]any{"still_queued": []string{}}
	case "get_context_usage":
		return map[string]any{
			"model":              *currentModel,
			"contextWindow":      map[string]any{"usedPercentage": 12.5},
			"categories":         []map[string]any{{"type": "messages", "percentage": 10.0}},
			"autoCompact":        map[string]any{"enabled": true, "thresholdPercentage": 92.0},
			"skills":             map[string]any{"count": 0, "percentageOfContext": 0.0, "items": []any{}},
			"duplicateFileReads": []any{},
			"session": map[string]any{
				"messageCount": 2, "promptCount": 1,
				"toolCalls":    map[string]any{"total": 0, "succeeded": 0, "failed": 0},
				"linesChanged": map[string]any{"added": 0, "removed": 0},
			},
		}
	case "get_usage_info":
		return map[string]any{
			"usage": map[string]any{
				"userId": "fake-user", "userType": "personal_standard",
				"totalUsagePercentage": 5.0, "isHighestTier": false,
				"userQuota": map[string]any{"total": 1000.0, "used": 50.0, "remaining": 950.0, "percentage": 5.0, "unit": "credits"},
			},
		}
	case "mcp_status":
		return map[string]any{"servers": []any{}}
	case "mcp_message":
		// Echo a minimal JSON-RPC result; the SDK handler is the real server.
		return map[string]any{"mcp_response": map[string]any{"jsonrpc": "2.0", "result": map[string]any{}}}
	case "get_byok_config":
		return map[string]any{"providers": []map[string]any{{
			"key": "openai", "display_name": "OpenAI", "api_key_url": "https://x", "url": "https://api.openai.com",
			"fields": []any{}, "types": []any{},
		}}}
	case "validate_byok_model":
		return map[string]any{"success": true}
	case "list_byok_configs":
		return map[string]any{"configs": []any{}}
	case "create_byok_config":
		return map[string]any{"config": map[string]any{"key": "fake-key"}}
	case "check_byok_config":
		return map[string]any{"success": true}
	case "reload_plugins":
		return map[string]any{"commands": []any{}, "agents": []any{}, "plugins": []any{}, "mcpServers": []any{}, "error_count": 0}
	case "reload_skills":
		return map[string]any{"skills": []any{}}
	case "rewind_files":
		return map[string]any{"canRewind": true, "filesChanged": []string{"a.txt"}}
	case "rewind":
		return map[string]any{"status": "success", "targetUserMessageId": "m1", "scope": "both", "filesChanged": []string{"a.txt"}}
	default:
		return map[string]any{}
	}
}

func availableModelsUpdate() map[string]any {
	return map[string]any{
		"type":    "system",
		"subtype": "available_models_update",
		"models": []map[string]any{
			{"value": "fake-efficient", "displayName": "Efficient", "isDefault": true},
			{"value": "fake-performance", "displayName": "Performance", "isReasoning": true, "maxInputTokens": 200000},
		},
		"currentModel": "fake-efficient",
		"uuid":         "models-1",
		"session_id":   "fake-session",
	}
}

func assistantToolUse() map[string]any {
	return map[string]any{
		"type": "assistant", "uuid": "asst-tool", "session_id": "fake-session",
		"message": map[string]any{
			"role": "assistant",
			"content": []map[string]any{{
				"type": "tool_use", "id": "call_1", "name": "Bash",
				"input": map[string]any{"command": "echo hi"},
			}},
		},
		"parent_tool_use_id": nil,
	}
}

func assistantMessage(uuid, text, model string) map[string]any {
	return map[string]any{
		"type":               "assistant",
		"message":            map[string]any{"role": "assistant", "model": model, "content": []map[string]any{{"type": "text", "text": text}}},
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
