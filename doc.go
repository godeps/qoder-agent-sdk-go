// Package qodersdk provides a Go SDK for driving the qoderclicn agent runtime
// over a bidirectional stdin/stdout JSONL wire protocol.
//
// It is a Go port of @qodercn-ai/qodercn-agent-sdk (the Qoder Agent SDK),
// mirroring the wire protocol types and the ProcessTransport semantics. The
// qoderclicn CLI and the SDK exchange one JSON object per line over the child
// process stdin/stdout; stderr carries diagnostics only.
//
// The primary entry point is [Query], which resolves the qoderclicn runtime,
// spawns it as a child process, performs the initialize handshake, writes the
// initial user message, and returns a channel of [protocol.Message] events.
// Control requests issued by the CLI (permission prompts, hook callbacks,
// in-process MCP calls, model-policy queries, elicitation) are delivered to
// caller-supplied callbacks on [Options].
//
// Wire protocol version: 1.4.0 (see [protocol.WireProtocolVersion]).
package qodersdk
