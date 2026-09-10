# qoder-agent-sdk-go

A Go SDK for driving the [qoderclicn](https://docs.qoder.cn/cli) agent runtime
over its bidirectional stdin/stdout JSONL wire protocol. It is a Go port of
`@qodercn-ai/qodercn-agent-sdk`, intended for embedding the Qoder agent engine
as an in-process kernel in Go applications.

## Overview

The SDK spawns a `qoderclicn` child process and speaks the Qoder wire protocol
(protocol version `1.4.0`): one JSON object per line over stdin/stdout. It
handles the `initialize` handshake, streams typed `SDKMessage` events
(assistant text, tool use, streaming deltas, results), and brokers
bidirectional control requests (permissions, hooks, in-process MCP, model
policy, elicitation).

## Quick start

```go
opts := qodersdk.NewOptions().
    WithModel("performance").
    WithCWD(".").
    WithAccessToken(os.Getenv("QODERCN_PERSONAL_ACCESS_TOKEN"))

msgs, err := qodersdk.Query(ctx, "List files in the current directory", opts)
if err != nil {
    log.Fatal(err)
}
for msg := range msgs {
    fmt.Println(msg.Type())
}
```

## Packages

- `protocol` — JSONL wire types (messages, control, common, permissions, mcp, hooks, agents).
- `auth` — Authentication options and one-shot payload file handling.
- `transport` — ProcessTransport: spawn qoderclicn, JSONL I/O, graceful shutdown.
- `runtime` — qoderclicn executable resolution.
- `mcp` — External and in-process MCP server configuration.

## License

Apache-2.0. The wire protocol is derived from the Qoder Agent SDK
(`@qodercn-ai/qodercn-agent-sdk`), whose protocol types are licensed under
Apache-2.0 (Copyright 2026 Google LLC).
