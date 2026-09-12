# qoder-agent-sdk-go(中文文档)

一个 Go SDK,通过**双向 stdin/stdout JSONL 线协议**驱动 [Qoder](https://docs.qoder.com/cli) / [QoderCN](https://docs.qoder.cn/cli) agent 运行时(`qodercli` / `qoderclicn`)。它是 `@qoder-ai/qoder-agent-sdk`(global)与 `@qodercn-ai/qodercn-agent-sdk`(cn)的 Go 移植,用于将 Qoder agent 引擎作为进程内内核嵌入 Go 应用。

- **线协议版本**:`1.4.0`(见 `protocol.WireProtocolVersion`)
- **支持 brand**:`global`(`@qoder-ai`,`qodercli`)与 `cn`(`@qodercn-ai`,`qoderclicn`)——两者均支持,可自动检测
- **许可证**:Apache-2.0(线协议派生自 Qoder Agent SDK,其协议类型为 Apache-2.0,Copyright 2026 Google LLC)

[English](./README.md)

## 安装

```bash
go get github.com/godeps/qoder-agent-sdk-go@v0.1.0
```

SDK **不自带** `qoderclicn`/`qodercli` 二进制。请通过以下方式提供:
- `QODERCLI_PATH` 环境变量,或
- `PATH`(SDK 会依次查找 `qoderclicn`、`qodercli`),或
- `Options.PathToCLI` / `auth.QodercliAuth()` 选项。

可从官方 npm 包安装 CLI(`npm install -g @qoder-ai/qoder-agent-sdk` 或 `@qodercn-ai/qodercn-agent-sdk`,安装时会自动下载运行时),或从 https://qoder.com.cn/download 下载。

## 概述

SDK spawn 一个 `qoderclicn`/`qodercli` 子进程,并通过 Qoder 线协议与之通信:子进程的 stdin/stdout 上**每行一个 JSON 对象**(stderr 仅输出诊断信息)。SDK 完成 `initialize` 握手、写入首条用户消息、流式产出类型化的 `protocol.Message` 事件,并代理双向控制请求(权限、hooks、进程内 MCP、模型策略、elicitation)。

```
Go 应用  →  qodersdk.Query()  →  ProcessTransport  →  qoderclicn/qodercli
                                                          ├─ Qoder 模型服务
                                                          ├─ 文件/命令/MCP 工具
                                                          └─ 子代理
```

## Brand(品牌)

支持两个 SDK brand,对应两个 npm 包。`auth.BrandAuto`(零值)可自动检测任一。

| | `BrandGlobal` | `BrandCN` | `BrandAuto` |
|---|---|---|---|
| npm 包 | `@qoder-ai/qoder-agent-sdk` | `@qodercn-ai/qodercn-agent-sdk` | 两者 |
| 二进制 | `qodercli` | `qoderclicn` | 两者(cn 优先) |
| PAT 环境变量 | `QODER_PERSONAL_ACCESS_TOKEN` | `QODERCN_PERSONAL_ACCESS_TOKEN` | cn 再 global |
| Service Account 环境变量 | `QODER_SERVICE_ACCOUNT_KEY` | `QODERCN_SERVICE_ACCOUNT_KEY` | cn 再 global |
| VPC 环境变量 | `QODER_VPC_ENDPOINT` | `QODERCN_VPC_ENDPOINT` | 两者 |
| 站点 | qoder.com | qoder.cn | — |

## 快速开始

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
		WithBrand(auth.BrandCN).                              // 或 BrandGlobal / BrandAuto
		WithAuth(auth.AccessTokenFromEnvBrand(auth.BrandCN)) // 读取 QODERCN_PERSONAL_ACCESS_TOKEN
		// .WithAuth(auth.QodercliAuth())                   // …或复用本地 CLI 登录态

	msgs, err := qodersdk.Query(ctx, "列出当前目录下的文件", opts)
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
			fmt.Printf("\n[完成] 轮数=%d 输入=%d 输出=%d\n",
				v.NumTurns, v.Usage.InputTokens, v.Usage.OutputTokens)
		}
	}
}
```

## 认证

所有方式都将一次性认证 payload JSON 写入临时文件,路径经 `QODER_SDK_AUTH_PAYLOAD_FILE` 环境变量传给 CLI;CLI 读取后初始化认证并删除该文件。

| 方法 | 说明 |
|---|---|
| `auth.AccessToken(token)` | 直接提供个人访问令牌(PAT)。 |
| `auth.AccessTokenFromEnv(envVar...)` | 从环境变量读取 PAT(默认 `QODERCN_PERSONAL_ACCESS_TOKEN`)。 |
| `auth.AccessTokenFromEnvBrand(brand)` | 读取 brand 对应的 PAT 环境变量;`BrandAuto` 先查 cn 再查 global。 |
| `auth.QodercliAuth()` | 复用本地 `qoderclicn`/`qodercli` 登录态。 |
| `auth.ServiceAccount(key)` | 直接提供 Service Account Key。 |
| `auth.ServiceAccountFromEnv(envVar...)` | 从环境变量读取 SA Key。 |
| `auth.ServiceAccountFromEnvBrand(brand)` | brand 对应的 SA-Key 环境变量。 |
| `auth.ServiceAccountWithTokenFetcher(fn)` | 由宿主提供短时 SAT。 |
| `auth.JobToken(fn)` | 宿主提供的 Job Token 获取函数。 |

环境变量辅助函数:`auth.AccessTokenEnvVar(brand)`、`auth.ServiceAccountEnvVar(brand)`、`auth.EnvAccessToken(brand)`。

## 选项(Options)

用 `qodersdk.NewOptions()` 配合 `With*` 链式构造,或直接填充结构体。

| 选项 | 字段 | 说明 |
|---|---|---|
| `WithPathToCLI` | `PathToCLI` | 覆盖 CLI 可执行文件路径。 |
| `WithCWD` | `CWD` | CLI 子进程工作目录。 |
| `WithEnv` | `Env` | 额外环境变量(默认 `os.Environ`)。 |
| `WithProxy` | `Proxy` | 出站代理。 |
| `WithVpcEndpoint` | `VpcEndpoint` | VPC 私有部署端点。 |
| `WithAuth` | `Auth` | 见[认证](#认证)。 |
| `WithBrand` | `Brand` | `BrandGlobal` / `BrandCN` / `BrandAuto`。 |
| `WithModel` | `Model` | 模型标识。 |
| `WithMaxTurns` | `MaxTurns` | agentic 循环轮数上限。 |
| `WithSystemPrompt` / `WithAppendSystemPrompt` | — | 覆盖 / 追加系统提示词。 |
| `WithPermissionMode` | `PermissionMode` | `default`/`acceptEdits`/`bypassPermissions`/`yolo`/`plan`/`dontAsk`/`auto`。 |
| `WithDangerouslySkipPermissions` | — | 跳过权限提示。 |
| `WithAllowedTools` / `WithDisallowedTools` | — | 工具允许/禁止列表。 |
| `WithPartialMessages` | — | 流式输出增量 `stream_event`。 |
| `WithMcpServers` | `McpServers` | 外部 MCP 服务器(`protocol.NewMcpHTTPConfig`/`NewMcpStdioConfig`/`NewMcpSSEConfig`)。 |
| `WithAgent` | `Agent` | 子代理名称。 |
| `WithSessionID` | `SessionID` | 恢复/继续会话。 |
| `WithCanUseTool` | 回调 | 工具权限回调(CLI `can_use_tool` → `protocol.PermissionResult`)。 |
| `WithHookCallback` | 回调 | Hook 执行回调(`protocol.HookJSONOutput`)。 |
| `WithHooks` | `Hooks` | 按事件注册 hook matcher;回调 id(`hook_N`)自动分配。 |
| `WithResolveModel` | 回调 | 模型策略回调(BYOK,`ModelPolicyResult`)。 |
| `WithMcpMessageHandler` | 回调 | 进程内 SDK MCP 服务器代理(JSON-RPC 帧)。 |
| `WithSdkMcpServers` | `SdkMcpServers` | 声明进程内 MCP 服务器名。 |
| `WithEnableFileCheckpointing` | — | 启用工作区快照(文件回退 rewind 必需)。 |
| `WithResume` / `WithContinue` / `WithForkSession` / `WithPersistSession` | — | 会话生命周期。 |
| — | `OnAuthExpired` | 认证过期时触发(每会话最多一次)。 |

其他字段:`Agents`、`Skills`、`Plugins`、`AdditionalDirectories`、`SettingSources`、`Settings`、`ExtraArgs`、`CloseGraceMs`、`InitializeTimeoutMs`。

## 多轮会话与控制方法

`Session` 让一个 CLI 进程跨轮存活(上下文在进程内保持,无需 `--resume`):

```go
sess, _ := qodersdk.NewSession(ctx, opts)
defer sess.Close()

sess.Send("记住数字 91")
res1, _, _ := sess.ReceiveResponse(60 * time.Second)

sess.Send("我让你记的数字是几?")
res2, _, _ := sess.ReceiveResponse(60 * time.Second) // "91"
```

会话中控制(均经真实 CLI 验证):

```go
sess.SetModel("performance")                        // 下一次调用生效
sess.SetPermissionMode(protocol.PermissionAcceptEdits)
sess.Interrupt()                                    // 中止当前轮
cu, _ := sess.GetContextUsage()                     // /context 风格用量
ui, _ := sess.GetUsageInfo()                        // 账户配额 + 会话积分
ms, _ := sess.McpStatus()                           // MCP 服务器状态
sess.RewindFiles(msgID, true)                       // 文件回退 dry-run
sess.Rewind(msgID, protocol.RewindScopeBoth, false) // 文件 + 对话
```

> 会话变更类控制(`SetModel`、`Interrupt`)要求会话已建立——首轮 `Send` 之后调用;之前调用返回 `ErrSessionNotEstablished`。

## Hooks

```go
opts := qodersdk.NewOptions().
    WithHooks(map[protocol.HookEvent][]protocol.HookSpec{
        protocol.HookPreToolUse: {{Matcher: "Bash"}},
    }).
    WithHookCallback(func(ctx context.Context, req *protocol.HookCallbackRequest) (protocol.HookJSONOutput, error) {
        if strings.Contains(string(req.Input.ToolInput), "rm -rf") {
            cont := false
            return protocol.HookJSONOutput{Continue: &cont, Decision: "block", Reason: "危险命令"}, nil
        }
        return protocol.HookJSONOutput{}, nil // 放行
    })
```

回调 id(`hook_N`)在 initialize 时确定性生成,注册无需手工指定 id。PreToolUse 拦截需要 `Continue: false`(qodercli 1.1.49 实测)。

## 进程内 SDK MCP 服务器

工具在本进程内服务——CLI 通过 `mcp_message` 控制通道代理 JSON-RPC 帧(`--mcp-config` + `--allowed-mcp-server-names` 自动接线):

```go
opts := qodersdk.NewOptions().
    WithSdkMcpServers("my-tools").
    WithMcpServers(map[string]protocol.McpServerConfig{
        "my-tools": protocol.NewMcpSdkConfig("my-tools"),
    }).
    WithMcpMessageHandler(func(ctx context.Context, server string, msg json.RawMessage) (json.RawMessage, error) {
        // 应答 initialize / tools/list / tools/call JSON-RPC 帧
        return handleJSONRPC(msg), nil
    })
```

## BYOK(自带密钥)

通过活动会话管理第三方模型凭据:

```go
cat, _ := sess.GetByokConfig()          // 服务端 provider 目录
ok, _ := sess.ValidateByokModel("openai", "gpt-x", "sk-...", "", "")
ref, _ := sess.CreateByokModelConfig(qodersdk.ByokModelConfigInput{...})
cfgs, _ := sess.ListByokConfigs()
sess.UpdateByokModelConfig(qodersdk.UpdateByokModelConfigInput{Key: ref.Key, ...})
res, _ := sess.CheckByokModelConfig(qodersdk.ByokModelConfigInput{...})
sess.DeleteByokConfigByKey(ref.Key)
// 自定义 OpenAI 兼容 / Alibaba provider:
sess.CreateByokCustomProvider(qodersdk.CustomByokProviderConfigInput{...})
```

单次调用的 BYOK 凭据也可通过 `WithResolveModel`(`ModelPolicyResult.CustomModel`)注入。

## Plugin 管理

`qodercli plugins <子命令> --json` 的无会话封装(一个 plugin 可捆绑 skills、agents、MCP servers、commands、hooks):

```go
plugins, _ := qodersdk.ListPlugins(ctx, &qodersdk.PluginOptions{})
res, _ := qodersdk.InstallPlugin(ctx, "my-plugin@marketplace", &qodersdk.PluginOptions{Scope: qodersdk.PluginScopeUser})
qodersdk.EnablePlugin(ctx, res.PluginID, nil)
qodersdk.DisablePlugin(ctx, res.PluginID, nil)
qodersdk.UninstallPlugin(ctx, res.PluginID, &qodersdk.PluginOptions{KeepData: true})
report, _ := qodersdk.ValidatePlugin(ctx, "./my-plugin", nil) // 静态校验,不执行任何代码
// 活动会话内热重载:
sess.ReloadPlugins()
sess.ReloadSkills()
```

## 线协议(Wire protocol)

协议版本 `1.4.0`(`protocol.WireProtocolVersion`)。CLI 在 `system/init` 消息中宣告其版本;跨主版本不匹配将被拒绝。

**Agent 消息**(`protocol.Message`,按 `type`/`subtype` 区分):`assistant`、`user`、`result`(`success`/`error_*`)、`system`(init/status/hook_*/task_*/session_state_changed/permission_denied/artifacts_update/…)、`stream_event`、`command_lifecycle`、`prompt_suggestion`、`cloud_agent_event`。

**控制协议**(双向,按 `request_id` 关联):`control_request` / `control_response` / `control_cancel`。CLI→SDK 的请求通过回调处理:`can_use_tool`、`hook_callback`、`mcp_message`、`get_model_policy`、`elicitation`。SDK→CLI:`initialize`、`interrupt`、`set_model`、`set_permission_mode`、`side_question` 等。

用 `protocol.ParseMessage(line)` 解码任意一行,再 type-switch。

## 包结构

| 包 | 内容 |
|---|---|
| `qodersdk`(根) | `Query`、`ListModels`、`Session`(多轮 + 控制)、BYOK 方法、Plugin 管理、`Options`、`ModelPolicyResult`、错误类型。 |
| `protocol` | 线协议类型:`Message`/`ParseMessage`、`AssistantMessage`、`ResultMessage`、`SystemMessage`、`ControlRequest`/`Response`、`InitializeRequest`、控制响应载荷(`InterruptResponse`、`GetContextUsageResponse`、`GetUsageInfoResponse`、`RewindResult`、BYOK 类型…)、`McpServerConfig`、`PermissionMode`、`HookEvent`、`AgentDefinition`、`WireProtocolVersion`。 |
| `auth` | `AuthOptions`、`Brand`、`AccessToken*`/`QodercliAuth`/`ServiceAccount*`/`JobToken`、`WritePayloadFile`。 |
| `transport` | `ProcessTransport`(spawn + JSONL I/O + 优雅关闭)。 |
| `runtime` | `ResolvePath(brand, override)`、`BinaryNames(brand)`。 |

## 运行时解析

`runtime.ResolvePath(brand, override)` 优先级:显式 override → `QODERCLI_PATH` 环境变量 → 在 `PATH` 中按 brand 二进制名查找(global 查 `qodercli`,cn 查 `qoderclicn`,auto 两者皆查、cn 优先)。

## MCP

外部 MCP 服务器通过 `Options.McpServers` 传入,序列化为 CLI 的 `--mcp-config`:

```go
opts.WithMcpServers(map[string]protocol.McpServerConfig{
	"my-tools": protocol.NewMcpHTTPConfig("http://localhost:7777", map[string]string{"Authorization": "Bearer xxx"}),
})
```

进程内 MCP(CLI→SDK 的 `mcp_message` 控制)通过回调代理。

## 测试

- `internal/fakecli`:脚本化的假 `qoderclicn`,用于 SDK 单元测试(握手 → initialize → 多轮 assistant/result → 控制往返,含阻塞式 `can_use_tool`/`hook_callback` 与 set_model/interrupt/context_usage/byok/rewind/reload 的类型化响应)。
- 单元测试覆盖:协议解码、transport argv/env、auth payload、brand env 解析、Session 多轮 + 全部控制方法、BYOK CRUD、hooks 接线、MCP handler 管线。
- 真实 CLI E2E 套件(`QODER_SDK_E2E=1` 门控),针对 `qodercli 1.1.49` + `QodercliAuth()` 全部验证通过:

| E2E 测试 | 验证内容 |
|---|---|
| `TestListModels_RealQodercli` | 模型目录(12 个模型) |
| `TestE2E_QueryOnce` | 单发查询往返 |
| `TestE2E_Session_MultiTurn` | 跨轮上下文保持("记住91"→"91") |
| `TestE2E_GetContextUsage_And_UsageInfo` | 上下文窗口 + 账户用量 |
| `TestE2E_SetModel` | 会话中模型切换实际生效 |
| `TestE2E_Interrupt` | 生成中途打断 |
| `TestE2E_Hooks_Block` | PreToolUse hook 拦截工具(canary 文件未创建) |
| `TestE2E_CanUseTool_Deny` | 权限提示路由到 SDK 回调并遵守拒绝 |
| `TestE2E_SdkMCP` | 进程内 MCP 工具经 `mcp_message` 被真实调用 |
| `TestE2E_PluginManagement` | `plugins list` + 静态 `plugins validate` 报告解析 |

```bash
go test ./...
# 真实 CLI 套件:
QODER_SDK_E2E=1 go test -run TestE2E -v
```

## 许可证

Apache-2.0。线协议派生自 Qoder Agent SDK(`@qoder-ai/qoder-agent-sdk` / `@qodercn-ai/qodercn-agent-sdk`),其协议类型为 Apache-2.0(Copyright 2026 Google LLC)。
