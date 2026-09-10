// Package protocol contains the JSONL wire types spoken between qoderclicn
// (the Qoder CLI runtime) and the SDK. It is the Go translation of the
// @qodercn-ai/qodercn-agent-sdk /protocol subpath export, whose types are
// licensed under Apache-2.0 (Copyright 2026 Google LLC).
//
// The CLI writes one JSON object per line to stdout; the SDK writes one JSON
// object per line to stdin. stderr carries diagnostics only and is not part
// of the protocol. Each line is either an agent Message (assistant/user/result/
// system/stream_event/...), a control request/response/cancel, a keep_alive,
// or a transcript_mirror frame.
package protocol

// WireProtocolVersion is the wire protocol version this SDK is built against.
//
// The CLI announces its compiled-in version in the system/init JSONL message
// (the protocol_version field). The SDK validates compatibility using semver:
// a cross-major mismatch is refused at handshake.
const WireProtocolVersion = "1.4.0"
