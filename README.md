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
| `WithResolveModel` | callback | Model-policy callback (BYOK, `ModelPolicyResult`). |
| — | `OnAuthExpired` | Fires (at most once) when auth expires. |

Other fields: `Hooks`, `Agents`, `Skills`, `Plugins`, `Continue`, `Resume`, `ForkSession`, `PersistSession`, `AdditionalDirectories`, `SettingSources`, `Settings`, `ExtraArgs`, `CloseGraceMs`, `InitializeTimeoutMs`.

## Wire protocol

Protocol version `1.4.0` (`protocol.WireProtocolVersion`). The CLI announces its version in the `system/init` message; cross-major mismatch is refused.

**Agent messages** (`protocol.Message`, discriminated by `type`/`subtype`): `assistant`, `user`, `result` (`success`/`error_*`), `system` (init/status/hook_*/task_*/session_state_changed/permission_denied/artifacts_update/…), `stream_event`, `command_lifecycle`, `prompt_suggestion`, `cloud_agent_event`.

**Control protocol** (bidirectional, `request_id`-correlated): `control_request` / `control_response` / `control_cancel`. CLI→SDK requests handled via callbacks: `can_use_tool`, `hook_callback`, `mcp_message`, `get_model_policy`, `elicitation`. SDK→CLI: `initialize`, `interrupt`, `set_model`, `set_permission_mode`, `side_question`, etc.

Decode any line with `protocol.ParseMessage(line)`, then type-switch.

## Packages

| Package | Contents |
|---|---|
| `qodersdk` (root) | `Query`, `Options`, `ModelPolicyResult`, errors. |
| `protocol` | Wire types: `Message`/`ParseMessage`, `AssistantMessage`, `ResultMessage`, `SystemMessage`, `ControlRequest`/`Response`, `InitializeRequest`, `McpServerConfig`, `PermissionMode`, `HookEvent`, `AgentDefinition`, `WireProtocolVersion`. |
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

- `internal/fakecli`: a scripted fake `qoderclicn` used by the SDK's integration tests (handshake → initialize → assistant → result → control round-trips).
- Unit tests cover protocol decoding, transport argv/env, auth payload, and brand env resolution.
- Real end-to-end: the SDK is verified against real `qodercli 1.1.49` via `qodercliAuth` (local login).

```bash
go test ./...
# Real CLI (set QODERCLI_PATH or have qoderclicn/qodercli in PATH):
QODERCLI_PATH=/path/to/qodercli go test ./...
```

## License

Apache-2.0. The wire protocol is derived from the Qoder Agent SDK (`@qoder-ai/qoder-agent-sdk` / `@qodercn-ai/qodercn-agent-sdk`), whose protocol types are licensed under Apache-2.0 (Copyright 2026 Google LLC).
