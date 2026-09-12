# qoder-agent-sdk-go

A Go SDK for driving the [Qoder](https://docs.qoder.com/cli) / [QoderCN](https://docs.qoder.cn/cli) agent runtime (`qodercli` / `qoderclicn`) over its bidirectional **stdin/stdout JSONL wire protocol**. It is a Go port of `@qoder-ai/qoder-agent-sdk` (global) and `@qodercn-ai/qodercn-agent-sdk` (CN), intended for embedding the Qoder agent engine as an in-process kernel in Go applications.

- **Wire protocol**: `1.4.0` (see `protocol.WireProtocolVersion`)
- **Brands**: `global` (`@qoder-ai`, `qodercli`) and `cn` (`@qodercn-ai`, `qoderclicn`) — both supported, auto-detectable
- **License**: Apache-2.0 (the wire protocol is derived from the Qoder Agent SDK, whose protocol types are Apache-2.0, Copyright 2026 Google LLC)

## Installation

```bash
go get github.com/godeps/qoder-agent-sdk-go@v0.1.0
```

The SDK does **not** bundle the `qoderclicn`/`qodercli` binary. Provide it via:
- `QODERCLI_PATH` env var, or
- `PATH` (the SDK looks for `qoderclicn` then `qodercli`), or
- `Options.PathToCLI` / `auth.QodercliAuth()` options.

Install the CLI from the official packages (`npm install -g @qoder-ai/qoder-agent-sdk` or `@qodercn-ai/qodercn-agent-sdk`, which download the runtime), or from https://qoder.com.cn/download.

## Overview

The SDK spawns a `qoderclicn`/`qodercli` child process and speaks the Qoder wire protocol: **one JSON object per line** over the child's stdin/stdout (stderr carries diagnostics only). It performs the `initialize` handshake, writes the initial user message, streams typed `protocol.Message` events, and brokers bidirectional control requests (permissions, hooks, in-process MCP, model policy, elicitation).

```
Go app  →  qodersdk.Query()  →  ProcessTransport  →  qoderclicn/qodercli
                                                     ├─ Qoder model service
                                                     ├─ file/command/MCP tools
                                                     └─ sub-agents
```

## Brands

Two SDK brands are supported, mirroring the two npm packages. `auth.BrandAuto` (the zero value) detects either.

| | `BrandGlobal` | `BrandCN` | `BrandAuto` |
|---|---|---|---|
| npm package | `@qoder-ai/qoder-agent-sdk` | `@qodercn-ai/qodercn-agent-sdk` | either |
| binary | `qodercli` | `qoderclicn` | both (cn first) |
| access-token env | `QODER_PERSONAL_ACCESS_TOKEN` | `QODERCN_PERSONAL_ACCESS_TOKEN` | cn then global |
| service-account env | `QODER_SERVICE_ACCOUNT_KEY` | `QODERCN_SERVICE_ACCOUNT_KEY` | cn then global |
| VPC env | `QODER_VPC_ENDPOINT` | `QODERCN_VPC_ENDPOINT` | both |
| site | qoder.com | qoder.cn | — |

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	qodersdk "github.com/godeps/qoder-agent-sdk-go"
	"github.com/godeps/qoder-agent-sdk-go/auth"
	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	opts := qodersdk.NewOptions().
		WithCWD(".").
		WithBrand(auth.BrandCN).                              // or BrandGlobal / BrandAuto
		WithAuth(auth.AccessTokenFromEnvBrand(auth.BrandCN)) // reads QODERCN_PERSONAL_ACCESS_TOKEN
		// .WithAuth(auth.QodercliAuth())                   // …or reuse local CLI login

	msgs, err := qodersdk.Query(ctx, "List the files in the current directory", opts)
	if err != nil {
		log.Fatal(err)
	}
	for m := range msgs {
		switch v := m.(type) {
		case *protocol.AssistantMessage:
			for _, b := range v.Message.Content {
				if b.Text != "" {
					fmt.Print(b.Text)
				}
			}
		case *protocol.ResultMessage:
			fmt.Printf("\n[done] turns=%d in=%d out=%d\n",
				v.NumTurns, v.Usage.InputTokens, v.Usage.OutputTokens)
		}
	}
}
```

## Authentication

All methods write a one-shot payload JSON to a temp file whose path is passed to the CLI via `QODER_SDK_AUTH_PAYLOAD_FILE`; the CLI reads it, initializes auth, then deletes it.

| Method | Description |
|---|---|
| `auth.AccessToken(token)` | Personal access token, provided directly. |
| `auth.AccessTokenFromEnv(envVar...)` | Read PAT from an env var (default `QODERCN_PERSONAL_ACCESS_TOKEN`). |
| `auth.AccessTokenFromEnvBrand(brand)` | Read the brand-specific PAT env var; `BrandAuto` checks cn then global. |
| `auth.QodercliAuth()` | Reuse the local `qoderclicn`/`qodercli` login state. |
| `auth.ServiceAccount(key)` | Service account key, provided directly. |
| `auth.ServiceAccountFromEnv(envVar...)` | Read SA key from env var. |
| `auth.ServiceAccountFromEnvBrand(brand)` | Brand-specific SA-key env var. |
| `auth.ServiceAccountWithTokenFetcher(fn)` | Host supplies short-lived SATs. |
| `auth.JobToken(fn)` | Host-supplied job token fetcher. |

Env var helpers: `auth.AccessTokenEnvVar(brand)`, `auth.ServiceAccountEnvVar(brand)`, `auth.EnvAccessToken(brand)`.

## Options

Build with `qodersdk.NewOptions()` and the `With*` chain, or populate the struct directly.

| Option | Field | Notes |
|---|---|---|
| `WithPathToCLI` | `PathToCLI` | Override the CLI executable path. |
| `WithCWD` | `CWD` | Working directory for the CLI process. |
| `WithEnv` | `Env` | Extra env vars (defaults to `os.Environ`). |
| `WithProxy` | `Proxy` | Outbound proxy. |
| `WithVpcEndpoint` | `VpcEndpoint` | VPC private-deployment endpoint. |
| `WithAuth` | `Auth` | See [Authentication](#authentication). |
| `WithBrand` | `Brand` | `BrandGlobal` / `BrandCN` / `BrandAuto`. |
| `WithModel` | `Model` | Model identifier. |
| `WithMaxTurns` | `MaxTurns` | Agentic-loop turn cap. |
| `WithSystemPrompt` / `WithAppendSystemPrompt` | — | System prompt override / append. |
| `WithPermissionMode` | `PermissionMode` | `default`/`acceptEdits`/`bypassPermissions`/`yolo`/`plan`/`dontAsk`/`auto`. |
| `WithDangerouslySkipPermissions` | — | Skip permission prompts. |
| `WithAllowedTools` / `WithDisallowedTools` | — | Tool allow/deny lists. |
| `WithPartialMessages` | — | Stream incremental `stream_event` deltas. |
| `WithMcpServers` | `McpServers` | External MCP servers (`protocol.NewMcpHTTPConfig`/`NewMcpStdioConfig`/`NewMcpSSEConfig`). |
| `WithAgent` | `Agent` | Sub-agent name. |
| `WithSessionID` | `SessionID` | Resume/continue a session. |
| `WithCanUseTool` | callback | Tool-permission callback (CLI `can_use_tool` → `protocol.PermissionResult`). |
| `WithHookCallback` | callback | Hook execution callback (`protocol.HookJSONOutput`). |
| `WithHooks` | `Hooks` | Register hook matchers per event; callback ids (`hook_N`) are auto-assigned. |
| `WithResolveModel` | callback | Model-policy callback (BYOK, `ModelPolicyResult`). |
| `WithMcpMessageHandler` | callback | In-process SDK MCP server proxy (JSON-RPC frames). |
| `WithSdkMcpServers` | `SdkMcpServers` | Declare in-process MCP server names. |
| `WithEnableFileCheckpointing` | — | Enable workspace snapshots (required for file rewind). |
| `WithResume` / `WithContinue` / `WithForkSession` / `WithPersistSession` | — | Session lifecycle. |
| — | `OnAuthExpired` | Fires (at most once) when auth expires. |

Other fields: `Agents`, `Skills`, `Plugins`, `AdditionalDirectories`, `SettingSources`, `Settings`, `ExtraArgs`, `CloseGraceMs`, `InitializeTimeoutMs`.

## Multi-turn sessions & control methods

`Session` keeps one CLI process alive across turns (context persists
in-process; no `--resume` needed):

```go
sess, _ := qodersdk.NewSession(ctx, opts)
defer sess.Close()

sess.Send("remember the number 91")
res1, _, _ := sess.ReceiveResponse(60 * time.Second)

sess.Send("what number did I say?")
res2, _, _ := sess.ReceiveResponse(60 * time.Second) // "91"
```

Mid-session control (all verified against the real CLI):

```go
sess.SetModel("performance")                       // applies from next call
sess.SetPermissionMode(protocol.PermissionAcceptEdits)
sess.Interrupt()                                   // abort current turn
cu, _ := sess.GetContextUsage()                    // /context-style usage
ui, _ := sess.GetUsageInfo()                       // account quota + session credits
ms, _ := sess.McpStatus()                          // MCP server states
sess.RewindFiles(msgID, true)                      // dry-run file rollback
sess.Rewind(msgID, protocol.RewindScopeBoth, false) // files + conversation
```

> Control methods that mutate the session (`SetModel`, `Interrupt`) require an
> established session — call them after the first `Send`. The SDK guards this
> with `ErrSessionNotEstablished`.

## Hooks

```go
opts := qodersdk.NewOptions().
    WithHooks(map[protocol.HookEvent][]protocol.HookSpec{
        protocol.HookPreToolUse: {{Matcher: "Bash"}},
    }).
    WithHookCallback(func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error) {
        if strings.Contains(string(req.Input.ToolInput), "rm -rf") {
            cont := false
            return protocol.HookJSONOutput{Continue: &cont, Decision: "block", Reason: "destructive"}, nil
        }
        return protocol.HookJSONOutput{}, nil // proceed
    })
```

Callback ids (`hook_N`) are generated deterministically at initialize, so
registration needs no manual ids. A PreToolUse block requires
`Continue: false` (verified against qodercli 1.1.49).

## In-process SDK MCP servers

Serve tools in-process — the CLI proxies JSON-RPC frames through the
`mcp_message` control channel (declared via `--mcp-config` +
`--allowed-mcp-server-names`, wired automatically):

```go
opts := qodersdk.NewOptions().
    WithSdkMcpServers("my-tools").
    WithMcpServers(map[string]protocol.McpServerConfig{
        "my-tools": protocol.NewMcpSdkConfig("my-tools"),
    }).
    WithMcpMessageHandler(func(ctx context.Context, server string, msg json.RawMessage) (json.RawMessage, error) {
        // answer initialize / tools/list / tools/call JSON-RPC frames
        return handleJSONRPC(msg), nil
    })
```

## BYOK (Bring Your Own Key)

Manage third-party model credentials through a live session:

```go
cat, _ := sess.GetByokConfig()          // server-side provider catalog
ok, _ := sess.ValidateByokModel("openai", "gpt-x", "sk-...", "", "")
ref, _ := sess.CreateByokModelConfig(qodersdk.ByokModelConfigInput{...})
cfgs, _ := sess.ListByokConfigs()
sess.UpdateByokModelConfig(qodersdk.UpdateByokModelConfigInput{Key: ref.Key, ...})
res, _ := sess.CheckByokModelConfig(qodersdk.ByokModelConfigInput{...})
sess.DeleteByokConfigByKey(ref.Key)
// custom OpenAI-compatible / Alibaba providers:
sess.CreateByokCustomProvider(qodersdk.CustomByokProviderConfigInput{...})
```

Per-call BYOK credentials can also flow through `WithResolveModel`
(`ModelPolicyResult.CustomModel`).

## Plugin management

Session-less wrappers over `qodercli plugins <subcommand> --json` (a plugin
bundles skills, agents, MCP servers, commands, and hooks):

```go
plugins, _ := qodersdk.ListPlugins(ctx, &qodersdk.PluginOptions{})
res, _ := qodersdk.InstallPlugin(ctx, "my-plugin@marketplace", &qodersdk.PluginOptions{Scope: qodersdk.PluginScopeUser})
qodersdk.EnablePlugin(ctx, res.PluginID, nil)
qodersdk.DisablePlugin(ctx, res.PluginID, nil)
qodersdk.UninstallPlugin(ctx, res.PluginID, &qodersdk.PluginOptions{KeepData: true})
report, _ := qodersdk.ValidatePlugin(ctx, "./my-plugin", nil) // static; executes nothing
// live-session reload:
sess.ReloadPlugins()
sess.ReloadSkills()
```

## Wire protocol

Protocol version `1.4.0` (`protocol.WireProtocolVersion`). The CLI announces its version in the `system/init` message; cross-major mismatch is refused.

**Agent messages** (`protocol.Message`, discriminated by `type`/`subtype`): `assistant`, `user`, `result` (`success`/`error_*`), `system` (init/status/hook_*/task_*/session_state_changed/permission_denied/artifacts_update/…), `stream_event`, `command_lifecycle`, `prompt_suggestion`, `cloud_agent_event`.

**Control protocol** (bidirectional, `request_id`-correlated): `control_request` / `control_response` / `control_cancel`. CLI→SDK requests handled via callbacks: `can_use_tool`, `hook_callback`, `mcp_message`, `get_model_policy`, `elicitation`. SDK→CLI: `initialize`, `interrupt`, `set_model`, `set_permission_mode`, `side_question`, etc.

Decode any line with `protocol.ParseMessage(line)`, then type-switch.

## Packages

| Package | Contents |
|---|---|
| `qodersdk` (root) | `Query`, `ListModels`, `Session` (multi-turn + control), BYOK methods, plugin management, `Options`, `ModelPolicyResult`, errors. |
| `protocol` | Wire types: `Message`/`ParseMessage`, `AssistantMessage`, `ResultMessage`, `SystemMessage`, `ControlRequest`/`Response`, `InitializeRequest`/`InitializeResponse`, control response payloads (`InterruptResponse`, `GetContextUsageResponse`, `GetUsageInfoResponse`, `RewindResult`, BYOK types…), `McpServerConfig`, `PermissionMode`, `HookEvent`, `AgentDefinition`, `WireProtocolVersion`. |
| `auth` | `AuthOptions`, `Brand`, `AccessToken*`/`QodercliAuth`/`ServiceAccount*`/`JobToken`, `WritePayloadFile`. |
| `transport` | `ProcessTransport` (spawn + JSONL I/O + graceful shutdown). |
| `runtime` | `ResolvePath(brand, override)`, `BinaryNames(brand)`. |

## Runtime resolution

`runtime.ResolvePath(brand, override)` precedence: explicit override → `QODERCLI_PATH` env → `PATH` lookup over the brand's binary names (`qodercli` for global, `qoderclicn` for cn; both for auto, cn first).

## MCP

External MCP servers are passed via `Options.McpServers` and serialized to the CLI's `--mcp-config`:

```go
opts.WithMcpServers(map[string]protocol.McpServerConfig{
	"my-tools": protocol.NewMcpHTTPConfig("http://localhost:7777", map[string]string{"Authorization": "Bearer xxx"}),
})
```

In-process MCP (CLI→SDK `mcp_message` control) is brokered through callbacks.

## Testing

- `internal/fakecli`: a scripted fake `qoderclicn` used by the SDK's unit tests (handshake → initialize → multi-turn assistant/result → control round-trips, incl. blocking `can_use_tool` / `hook_callback` and typed responses for set_model/interrupt/context_usage/byok/rewind/reload).
- Unit tests cover protocol decoding, transport argv/env, auth payload, brand env resolution, Session multi-turn + all control methods, BYOK CRUD, hooks wiring, and MCP handler plumbing.
- Real end-to-end suite (gated on `QODER_SDK_E2E=1`), verified against `qodercli 1.1.49` with `QodercliAuth()`:

| E2E test | Verifies |
|---|---|
| `TestE2E_QueryOnce` | one-shot query round-trip |
| `TestE2E_Session_MultiTurn` | in-process context retention across turns |
| `TestE2E_GetContextUsage_And_UsageInfo` | context window + account usage |
| `TestE2E_SetModel` | mid-session model switch takes effect |
| `TestE2E_Interrupt` | turn abort mid-generation |
| `TestE2E_Hooks_Block` | PreToolUse hook blocks the tool (canary file NOT created) |
| `TestE2E_CanUseTool_Deny` | permission prompt routed to the SDK callback |
| `TestE2E_SdkMCP` | in-process MCP tool called via `mcp_message` |
| `TestE2E_PluginManagement` | `plugins list` + static `plugins validate` |

```bash
go test ./...
# Real CLI suite:
QODER_SDK_E2E=1 go test -run TestE2E -v
```

## License

Apache-2.0. The wire protocol is derived from the Qoder Agent SDK (`@qoder-ai/qoder-agent-sdk` / `@qodercn-ai/qodercn-agent-sdk`), whose protocol types are licensed under Apache-2.0 (Copyright 2026 Google LLC).
