package qodersdk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godeps/qoder-agent-sdk-go/protocol"
	"github.com/godeps/qoder-agent-sdk-go/transport"
)

// Query starts a one-shot session: spawns qoderclicn, performs the initialize
// handshake, writes the prompt as the first user message, and returns a
// channel of protocol.Message events. The channel closes after the terminal
// result message (or when the CLI exits). The session is canceled when ctx
// is done.
func Query(ctx context.Context, prompt string, opts *Options) (<-chan protocol.Message, error) {
	if opts == nil {
		opts = NewOptions()
	}
	if !opts.Auth.Configured() {
		return nil, ErrAuthNotConfigured
	}
	r := newQueryRunner(ctx, opts)
	if err := r.start(); err != nil {
		r.shutdown()
		return nil, err
	}
	if err := r.handshake(); err != nil {
		r.shutdown()
		return nil, err
	}
	if err := r.sendUserPrompt(prompt); err != nil {
		r.shutdown()
		return nil, err
	}
	return r.out, nil
}

// ListModels spawns qoderclicn, performs the initialize handshake, and returns
// the model catalog the CLI pushes via system/available_models_update. It does
// not send a user message; the session is torn down before returning.
func ListModels(ctx context.Context, opts *Options) ([]protocol.ModelInfo, error) {
	if opts == nil {
		opts = NewOptions()
	}
	if !opts.Auth.Configured() {
		return nil, ErrAuthNotConfigured
	}
	r := newQueryRunner(ctx, opts)
	if err := r.start(); err != nil {
		r.shutdown()
		return nil, err
	}
	defer r.shutdown()
	if err := r.handshake(); err != nil {
		return nil, err
	}
	deadline := 15 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if d := time.Until(dl); d > 0 && d < deadline {
			deadline = d
		}
	}
	timer := time.NewTimer(deadline)
	defer timer.Stop()
	for {
		select {
		case m, ok := <-r.out:
			if !ok {
				return nil, ErrSessionClosed
			}
			if s, ok := m.(*protocol.SystemMessage); ok && s.Subtype == "available_models_update" {
				return s.Models, nil
			}
		case <-timer.C:
			return nil, fmt.Errorf("qoder: timed out waiting for available_models_update")
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// queryRunner drives a single session: transport + message dispatch loop.
type queryRunner struct {
	ctx       context.Context
	opts      *Options
	tr        *transport.ProcessTransport
	out       chan protocol.Message
	runDone   chan struct{}
	initCh    chan *protocol.SystemMessage
	pendingMu sync.Mutex
	pending   map[string]chan protocol.ControlResponse
	closeOnce sync.Once
	closed    atomic.Bool
	oneShot   bool

	sessionMu sync.RWMutex
	sessionID string
}

func newQueryRunner(ctx context.Context, opts *Options) *queryRunner {
	return &queryRunner{
		ctx:     ctx,
		opts:    opts,
		out:     make(chan protocol.Message, 64),
		runDone: make(chan struct{}),
		initCh:  make(chan *protocol.SystemMessage, 1),
		pending: make(map[string]chan protocol.ControlResponse),
		oneShot: true,
	}
}

// newSessionRunner creates a runner that keeps reading past result messages
// (multi-turn sessions and background events).
func newSessionRunner(ctx context.Context, opts *Options) *queryRunner {
	r := newQueryRunner(ctx, opts)
	r.oneShot = false
	return r
}

// lastSessionID returns the session id captured from system/init.
func (r *queryRunner) lastSessionID() string {
	r.sessionMu.RLock()
	defer r.sessionMu.RUnlock()
	return r.sessionID
}

func (r *queryRunner) start() error {
	o := r.opts
	topts := transport.TransportOptions{
		PathToCLI:                       o.PathToCLI,
		CWD:                             o.CWD,
		Env:                             o.Env,
		Proxy:                           o.Proxy,
		VpcEndpoint:                     o.VpcEndpoint,
		Auth:                            o.Auth,
		Brand:                           o.Brand,
		Debug:                           o.Debug,
		Model:                           o.Model,
		Agent:                           o.Agent,
		MaxTurns:                        o.MaxTurns,
		PermissionMode:                  o.PermissionMode,
		AllowDangerouslySkipPermissions: o.AllowDangerouslySkipPermissions,
		IncludePartialMessages:          o.IncludePartialMessages,
		AllowedTools:                    o.AllowedTools,
		DisallowedTools:                 o.DisallowedTools,
		McpServers:                      o.McpServers,
		AllowedMcpServerNames:           allowedMcpNames(o),
		SettingSources:                  o.SettingSources,
		AdditionalDirectories:           o.AdditionalDirectories,
		Plugins:                         o.Plugins,
		ExtraArgs:                       o.ExtraArgs,
		SessionID:                       o.SessionID,
		Continue:                        o.Continue,
		Resume:                          o.Resume,
		ForkSession:                     o.ForkSession,
		PersistSession:                  o.PersistSession,
		CloseGraceMs:                    o.CloseGraceMs,
		CanUseTool:                      o.CanUseTool != nil,
		OnAuthExpired:                   o.OnAuthExpired,
	}
	r.tr = transport.NewProcessTransport(topts)
	if err := r.tr.Initialize(r.ctx); err != nil {
		return err
	}
	go r.runLoop()
	return nil
}

// runLoop dispatches messages from the transport until the CLI exits or a
// terminal result message arrives (one-shot mode only).
func (r *queryRunner) runLoop() {
	defer r.shutdown()
	defer close(r.runDone)
	defer close(r.out)
	for msg := range r.tr.Messages() {
		if r.closed.Load() {
			break
		}
		r.dispatch(msg)
		if r.oneShot {
			if _, ok := msg.(*protocol.ResultMessage); ok {
				break
			}
		}
	}
}

func (r *queryRunner) dispatch(msg protocol.Message) {
	// Capture the session id from the first message that carries one. The
	// real CLI does not always emit system/init first (in long-lived mode it
	// may lead with artifacts_update / available_models_update), so we accept
	// a session id from any message rather than only from init.
	r.captureSessionID(msg)

	switch m := msg.(type) {
	case *protocol.SystemMessage:
		// system/init (with protocol_version) is forwarded to the caller.
		r.forward(m)
	case *protocol.ControlResponse:
		r.routeResponse(m)
	case *protocol.ControlRequest:
		go r.handleControlRequest(m)
	case *protocol.ControlCancel:
		// drop
	default:
		r.forward(m)
	}
}

// sessionIDOf extracts the session id from any message that carries one.
func sessionIDOf(msg protocol.Message) string {
	switch m := msg.(type) {
	case *protocol.SystemMessage:
		return m.SessionID
	case *protocol.AssistantMessage:
		return m.SessionID
	case *protocol.UserMessage:
		return m.SessionID
	case *protocol.ResultMessage:
		return m.SessionID
	}
	return ""
}

func (r *queryRunner) captureSessionID(msg protocol.Message) {
	sid := sessionIDOf(msg)
	if sid == "" {
		return
	}
	r.sessionMu.Lock()
	if r.sessionID == "" {
		r.sessionID = sid
	}
	r.sessionMu.Unlock()
}

func (r *queryRunner) forward(m protocol.Message) {
	if r.closed.Load() {
		return
	}
	r.out <- m
}

func (r *queryRunner) routeResponse(m *protocol.ControlResponse) {
	// The control_response envelope carries session_id too — capture it so
	// control methods work even before the first agent message arrives.
	if m.SessionID != "" {
		r.sessionMu.Lock()
		if r.sessionID == "" {
			r.sessionID = m.SessionID
		}
		r.sessionMu.Unlock()
	}
	var probe struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(m.Response, &probe)
	r.pendingMu.Lock()
	ch, ok := r.pending[probe.RequestID]
	if ok {
		delete(r.pending, probe.RequestID)
	}
	r.pendingMu.Unlock()
	if ok {
		select {
		case ch <- *m:
		default:
		}
	}
}

// handshake waits for system/init, validates the protocol version, and sends
// the initialize control request.
func (r *queryRunner) handshake() error {
	// Auto-fill empty hook callback ids with the TS SDK's deterministic
	// hook_N scheme so hosts can register hooks without inventing ids.
	r.opts.Hooks = fillHookCallbackIDs(r.opts.Hooks)

	// qoderclicn expects the SDK to send initialize first; it replies with a
	// system/init agent message (forwarded to the caller) and a control_response.
	initReq := protocol.InitializeRequest{
		Type:                           "initialize",
		Model:                          r.opts.Model,
		CWD:                            r.opts.CWD,
		AllowedTools:                   r.opts.AllowedTools,
		DisallowedTools:                r.opts.DisallowedTools,
		McpServers:                     r.opts.McpServers,
		Agents:                         r.opts.Agents,
		Skills:                         r.opts.Skills,
		Plugins:                        r.opts.Plugins,
		SystemPrompt:                   r.opts.SystemPrompt,
		AppendSystemPrompt:             r.opts.AppendSystemPrompt,
		PermissionMode:                 r.opts.PermissionMode,
		Hooks:                          r.opts.Hooks,
		SdkMcpServers:                  r.opts.SdkMcpServers,
		EnableFileCheckpointing:        r.opts.EnableFileCheckpointing,
		SupportsCatalogReadyInitialize: boolPtr(true),
		SupportsAvailableModelsUpdate:  boolPtr(true),
		SupportsCommandsChanged:        boolPtr(true),
	}
	resp, err := r.sendControlRequest(initReq)
	if err != nil {
		return fmt.Errorf("qoder: initialize failed: %w (stderr: %s)", err, r.tr.StderrTail())
	}
	if !resp.IsSuccess() {
		if e, _ := resp.Error(); e != nil {
			return &ControlRequestError{RequestID: e.RequestID, Code: e.Code, Message: e.Error, Retryable: e.Retryable != nil && *e.Retryable}
		}
		return fmt.Errorf("qoder: initialize failed")
	}
	return nil
}

// sendUserPrompt writes the initial user message to the CLI stdin.
func (r *queryRunner) sendUserPrompt(prompt string) error {
	user := map[string]any{
		"type":               "user",
		"message":            map[string]any{"role": "user", "content": prompt},
		"parent_tool_use_id": nil,
	}
	return r.tr.WriteJSON(user)
}

// sendControlRequest sends a control_request envelope and awaits the matching
// control_response by request_id.
func (r *queryRunner) sendControlRequest(inner any) (protocol.ControlResponse, error) {
	id := newRequestID()
	innerJSON, err := json.Marshal(inner)
	if err != nil {
		return protocol.ControlResponse{}, err
	}
	req := protocol.ControlRequest{
		Type:      "control_request",
		RequestID: id,
		Request:   innerJSON,
	}
	ch := make(chan protocol.ControlResponse, 1)
	r.pendingMu.Lock()
	r.pending[id] = ch
	r.pendingMu.Unlock()
	defer func() {
		r.pendingMu.Lock()
		delete(r.pending, id)
		r.pendingMu.Unlock()
	}()
	if err := r.tr.WriteJSON(req); err != nil {
		return protocol.ControlResponse{}, err
	}
	select {
	case resp := <-ch:
		return resp, nil
	case <-r.ctx.Done():
		return protocol.ControlResponse{}, r.ctx.Err()
	case <-r.runDone:
		return protocol.ControlResponse{}, ErrSessionClosed
	}
}

// handleControlRequest dispatches a CLI→SDK control request to callbacks.
func (r *queryRunner) handleControlRequest(req *protocol.ControlRequest) {
	inner, err := req.Inner()
	if err != nil {
		r.respondError(req.RequestID, "decode error: "+err.Error())
		return
	}
	ctx := r.ctx
	switch ir := inner.(type) {
	case protocol.CanUseToolRequest:
		r.handleCanUseTool(ctx, req.RequestID, &ir)
	case protocol.HookCallbackRequest:
		r.handleHook(ctx, req.RequestID, &ir)
	case protocol.GetModelPolicyRequest:
		r.handleModelPolicy(ctx, req.RequestID, &ir)
	case protocol.ElicitationRequest:
		r.respondSuccess(req.RequestID, nil)
	case protocol.McpMessageRequest:
		r.handleMcpMessage(ctx, req.RequestID, &ir)
	default:
		r.respondSuccess(req.RequestID, nil)
	}
}

func (r *queryRunner) handleCanUseTool(ctx context.Context, id string, req *protocol.CanUseToolRequest) {
	if r.opts.CanUseTool == nil {
		r.respondError(id, "no permission callback configured")
		return
	}
	res, err := r.opts.CanUseTool(ctx, req)
	if err != nil {
		r.respondError(id, err.Error())
		return
	}
	r.respondSuccess(id, structToMap(res))
}

func (r *queryRunner) handleHook(ctx context.Context, id string, req *protocol.HookCallbackRequest) {
	if r.opts.HookCallback == nil {
		r.respondSuccess(id, nil)
		return
	}
	out, err := r.opts.HookCallback(ctx, req)
	if err != nil {
		r.respondError(id, err.Error())
		return
	}
	r.respondSuccess(id, structToMap(out))
}

func (r *queryRunner) handleModelPolicy(ctx context.Context, id string, req *protocol.GetModelPolicyRequest) {
	if r.opts.ResolveModel == nil {
		r.respondSuccess(id, nil)
		return
	}
	res, err := r.opts.ResolveModel(ctx, req)
	if err != nil {
		r.respondError(id, err.Error())
		return
	}
	r.respondSuccess(id, structToMap(res))
}

// handleMcpMessage proxies one JSON-RPC frame to the host's in-process MCP
// server and replies with the server's response inside the control_response
// envelope (verified wire shape: {mcp_response: <JSONRPCMessage>}).
func (r *queryRunner) handleMcpMessage(ctx context.Context, id string, req *protocol.McpMessageRequest) {
	if r.opts.McpMessageHandler == nil {
		r.respondError(id, "no MCP message handler configured for server "+req.ServerName)
		return
	}
	resp, err := r.opts.McpMessageHandler(ctx, req.ServerName, req.Message)
	if err != nil {
		r.respondError(id, err.Error())
		return
	}
	body := map[string]any{}
	if len(resp) > 0 {
		body["mcp_response"] = json.RawMessage(resp)
	} else {
		// Notification: the CLI still expects a non-null mcp_response. Send an
		// id-less ack so it never collides with a real JSON-RPC request id.
		body["mcp_response"] = map[string]any{"jsonrpc": "2.0", "result": map[string]any{}}
	}
	raw, _ := json.Marshal(body)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(raw, &m)
	r.respondSuccess(id, m)
}

func (r *queryRunner) respondSuccess(id string, response map[string]json.RawMessage) {
	_ = r.tr.WriteJSON(protocol.NewSuccessResponse(id, response))
}

func (r *queryRunner) respondError(id, message string) {
	_ = r.tr.WriteJSON(protocol.NewErrorResponse(id, message))
}

func (r *queryRunner) shutdown() {
	r.closeOnce.Do(func() {
		r.closed.Store(true)
		if r.tr != nil {
			_ = r.tr.Close()
		}
	})
}

// --- helpers ---

func newRequestID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func boolPtr(b bool) *bool { return &b }

// allowedMcpNames collects every declared MCP server name (in-process sdk
// servers plus external ones). The CLI's --allowed-mcp-server-names flag
// gates which of them may run; without it, servers declared in initialize
// can stay "disconnected" (verified against qodercli 1.1.49).
func allowedMcpNames(o *Options) []string {
	seen := map[string]bool{}
	var names []string
	for _, n := range o.SdkMcpServers {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for n := range o.McpServers {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	return names
}

// fillHookCallbackIDs assigns deterministic hook_N ids to specs that do not
// carry explicit HookCallbackIDs, mirroring the TS SDK's scheme
// (global counter across events, in registration order).
func fillHookCallbackIDs(hooks map[protocol.HookEvent][]protocol.HookSpec) map[protocol.HookEvent][]protocol.HookSpec {
	if len(hooks) == 0 {
		return hooks
	}
	out := make(map[protocol.HookEvent][]protocol.HookSpec, len(hooks))
	next := 0
	// Deterministic iteration: sort events for stable id assignment.
	events := make([]protocol.HookEvent, 0, len(hooks))
	for e := range hooks {
		events = append(events, e)
	}
	sort.Slice(events, func(i, j int) bool { return events[i] < events[j] })
	for _, e := range events {
		specs := hooks[e]
		filled := make([]protocol.HookSpec, len(specs))
		for i, s := range specs {
			if len(s.HookCallbackIDs) == 0 {
				s.HookCallbackIDs = []string{fmt.Sprintf("hook_%d", next)}
				next++
			}
			filled[i] = s
		}
		out[e] = filled
	}
	return out
}

func structToMap(v any) map[string]json.RawMessage {
	data, _ := json.Marshal(v)
	var m map[string]json.RawMessage
	_ = json.Unmarshal(data, &m)
	return m
}

// checkProtocolVersion refuses cross-major mismatches.
func checkProtocolVersion(got string) error {
	wantMajor := strings.Split(protocol.WireProtocolVersion, ".")[0]
	gotParts := strings.SplitN(got, ".", 2)
	if len(gotParts) == 0 || gotParts[0] != wantMajor {
		return &ProtocolVersionMismatchError{Got: got, Want: protocol.WireProtocolVersion}
	}
	return nil
}
