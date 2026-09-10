package qodersdk

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
}

func newQueryRunner(ctx context.Context, opts *Options) *queryRunner {
	return &queryRunner{
		ctx:     ctx,
		opts:    opts,
		out:     make(chan protocol.Message, 64),
		runDone: make(chan struct{}),
		initCh:  make(chan *protocol.SystemMessage, 1),
		pending: make(map[string]chan protocol.ControlResponse),
	}
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
// terminal result message arrives.
func (r *queryRunner) runLoop() {
	defer r.shutdown()
	defer close(r.runDone)
	defer close(r.out)
	for msg := range r.tr.Messages() {
		if r.closed.Load() {
			break
		}
		r.dispatch(msg)
		if _, ok := msg.(*protocol.ResultMessage); ok {
			break
		}
	}
}

func (r *queryRunner) dispatch(msg protocol.Message) {
	switch m := msg.(type) {
	case *protocol.SystemMessage:
		if m.Subtype == "init" {
			select {
			case r.initCh <- m:
			default:
			}
		}
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

func (r *queryRunner) forward(m protocol.Message) {
	if r.closed.Load() {
		return
	}
	r.out <- m
}

func (r *queryRunner) routeResponse(m *protocol.ControlResponse) {
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
	var init *protocol.SystemMessage
	timeout := time.Duration(r.opts.InitializeTimeoutMs) * time.Millisecond
	select {
	case init = <-r.initCh:
	case <-r.runDone:
		return fmt.Errorf("qoder: qoderclicn exited before system/init\n%s", r.tr.StderrTail())
	case <-time.After(timeout):
		return ErrInitializeTimeout
	case <-r.ctx.Done():
		return r.ctx.Err()
	}
	if init.ProtocolVersion != "" {
		if err := checkProtocolVersion(init.ProtocolVersion); err != nil {
			return err
		}
	}
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
		SupportsCatalogReadyInitialize: boolPtr(true),
		SupportsAvailableModelsUpdate:  boolPtr(true),
		SupportsCommandsChanged:        boolPtr(true),
	}
	resp, err := r.sendControlRequest(initReq)
	if err != nil {
		return err
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
		r.respondError(req.RequestID, "in-process mcp not supported in this host")
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
