package qodersdk

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

// Session is a long-lived Qoder session: one CLI child process serving
// multiple turns. It mirrors the TypeScript SDK's Query control surface
// (query() with an async-iterable prompt + control methods).
//
// A Session keeps reading the agent stream in the background; use Stream to
// receive events, Send to post new user turns, and the control methods
// (Interrupt, SetModel, SetPermissionMode, Rewind, GetContextUsage,
// GetUsageInfo, McpStatus) to steer and inspect the session. Close tears the
// process down.
type Session struct {
	r        *queryRunner
	events   chan protocol.Message
	sendMu   sync.Mutex
	closedCh chan struct{}
	closeMu  sync.Once
}

// NewSession starts a session WITHOUT sending a prompt. The CLI process
// spawns and the initialize handshake completes before returning; the first
// turn starts on the first Send. Unlike Query, the stream keeps delivering
// events past result messages (multi-turn + background task events).
func NewSession(ctx context.Context, opts *Options) (*Session, error) {
	if opts == nil {
		opts = NewOptions()
	}
	if !opts.Auth.Configured() {
		return nil, ErrAuthNotConfigured
	}
	r := newSessionRunner(ctx, opts)
	if err := r.start(); err != nil {
		r.shutdown()
		return nil, err
	}
	if err := r.handshake(); err != nil {
		r.shutdown()
		return nil, err
	}
	s := &Session{
		r:        r,
		events:   make(chan protocol.Message, 256),
		closedCh: make(chan struct{}),
	}
	// Tee the runner output into the session event channel for the session's
	// lifetime (the runner's out channel is consumed here, not by the caller).
	go func() {
		defer close(s.events)
		for m := range r.out {
			select {
			case s.events <- m:
			case <-s.closedCh:
				return
			}
		}
	}()
	// Briefly wait for the CLI to surface a session id (system/init on
	// one-shot CLIs). Long-lived CLIs only attach session_id to agent
	// messages, which arrive on the first turn — so we do not block long
	// here; control methods capture the id from the first turn's messages
	// and guard with ErrSessionNotEstablished until then.
	deadline := time.NewTimer(3 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for r.lastSessionID() == "" {
		select {
		case <-deadline.C:
			return s, nil
		case <-r.runDone:
			return nil, ErrSessionClosed
		case <-ctx.Done():
			r.shutdown()
			return nil, ctx.Err()
		case <-tick.C:
		}
	}
	return s, nil
}

// Send posts a user turn (text). Events for the turn (and any background
// tasks) arrive on Stream.
func (s *Session) Send(text string) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	select {
	case <-s.closedCh:
		return ErrSessionClosed
	default:
	}
	return s.r.sendUserPrompt(text)
}

// Stream returns the channel of agent messages. It stays open across turns
// and closes when the session ends.
func (s *Session) Stream() <-chan protocol.Message {
	return s.events
}

// ReceiveResponse reads Stream until the next terminal ResultMessage and
// returns it together with every message seen along the way.
func (s *Session) ReceiveResponse(timeout time.Duration) (*protocol.ResultMessage, []protocol.Message, error) {
	var seen []protocol.Message
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case m, ok := <-s.events:
			if !ok {
				return nil, seen, ErrSessionClosed
			}
			seen = append(seen, m)
			if r, ok := m.(*protocol.ResultMessage); ok {
				return r, seen, nil
			}
		case <-deadline.C:
			return nil, seen, context.DeadlineExceeded
		case <-s.closedCh:
			return nil, seen, ErrSessionClosed
		}
	}
}

// SessionID returns the CLI-assigned session id captured from system/init.
func (s *Session) SessionID() string {
	if s.r.opts.SessionID != "" {
		return s.r.opts.SessionID
	}
	return s.r.lastSessionID()
}

// sendControl is the shared plumbing for Session control methods: send one
// request, await the response, decode the success payload into out.
func (s *Session) sendControl(inner any, timeout time.Duration, out any) error {
	ctx, cancel := context.WithTimeout(s.r.ctx, timeout)
	defer cancel()
	done := make(chan struct {
		resp protocol.ControlResponse
		err  error
	}, 1)
	go func() {
		resp, err := s.r.sendControlRequest(inner)
		done <- struct {
			resp protocol.ControlResponse
			err  error
		}{resp, err}
	}()
	select {
	case res := <-done:
		if res.err != nil {
			return res.err
		}
		if !res.resp.IsSuccess() {
			if e, _ := res.resp.Error(); e != nil {
				return &ControlRequestError{RequestID: e.RequestID, Code: e.Code, Message: e.Error, Retryable: e.Retryable != nil && *e.Retryable}
			}
			return fmt.Errorf("qoder: control request failed")
		}
		if out == nil {
			return nil
		}
		succ, err := res.resp.Success()
		if err != nil {
			return err
		}
		if succ == nil || succ.Response == nil {
			return nil
		}
		// The success body's "response" field holds the typed payload.
		raw, _ := json.Marshal(succ.Response)
		return json.Unmarshal(raw, out)
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closedCh:
		return ErrSessionClosed
	}
}

// Interrupt aborts the current turn. The CLI requires an established session.
// Returns the queued-message uuids that survive the interrupt.
func (s *Session) Interrupt() (*protocol.InterruptResponse, error) {
	if s.r.lastSessionID() == "" {
		return nil, ErrSessionNotEstablished
	}
	var out protocol.InterruptResponse
	err := s.sendControl(protocol.InterruptRequest{Type: "interrupt"}, 30*time.Second, &out)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// SetModel switches the active model mid-session. Verified against qodercli
// 1.1.49: requires an established session; the switch applies to subsequent
// model calls.
func (s *Session) SetModel(model string) error {
	if s.r.lastSessionID() == "" {
		return ErrSessionNotEstablished
	}
	return s.sendControl(protocol.SetModelRequest{Type: "set_model", Model: model}, 30*time.Second, nil)
}

// SetPermissionMode switches the permission mode mid-session.
func (s *Session) SetPermissionMode(mode protocol.PermissionMode) error {
	if s.r.lastSessionID() == "" {
		return ErrSessionNotEstablished
	}
	return s.sendControl(protocol.SetPermissionModeRequest{Type: "set_permission_mode", Mode: mode}, 30*time.Second, nil)
}

// GetContextUsage returns /context-style context-window usage.
func (s *Session) GetContextUsage() (*protocol.GetContextUsageResponse, error) {
	var out protocol.GetContextUsageResponse
	if err := s.sendControl(protocol.GetContextUsageRequest{Type: "get_context_usage"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetUsageInfo returns account/session usage and quota information.
func (s *Session) GetUsageInfo() (*protocol.GetUsageInfoResponse, error) {
	var out protocol.GetUsageInfoResponse
	if err := s.sendControl(protocol.GetUsageInfoRequest{Type: "get_usage_info"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// McpStatus returns the runtime status of configured MCP servers.
func (s *Session) McpStatus() (*protocol.McpStatusResponse, error) {
	var out protocol.McpStatusResponse
	if err := s.sendControl(protocol.McpStatusRequest{Type: "mcp_status"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RewindFiles rolls workspace files back to the state before the given user
// message. Requires Options.WithEnableFileCheckpointing(true). Anchors come
// from user message uuids on the stream.
func (s *Session) RewindFiles(userMessageID string, dryRun bool) (*protocol.RewindFilesResult, error) {
	var out protocol.RewindFilesResult
	req := protocol.RewindFilesRequest{Type: "rewind_files", UserMessageID: userMessageID}
	if dryRun {
		req.DryRun = boolPtr(true)
	}
	if err := s.sendControl(req, 60*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Rewind rolls back conversation and/or files to the state before the given
// user message. Scope: conversation | files | both (default both).
// Requires Options.WithEnableFileCheckpointing(true) for file scopes.
func (s *Session) Rewind(userMessageID string, scope protocol.RewindScope, dryRun bool) (*protocol.RewindResult, error) {
	var out protocol.RewindResult
	req := protocol.RewindRequest{Type: "rewind", UserMessageID: userMessageID}
	if scope != "" {
		sc := string(scope)
		req.Scope = &sc
	}
	if dryRun {
		req.DryRun = boolPtr(true)
	}
	if err := s.sendControl(req, 60*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Close ends the session and reaps the CLI child process.
func (s *Session) Close() error {
	s.closeMu.Do(func() {
		close(s.closedCh)
		// Best-effort graceful end before transport teardown.
		_ = s.sendControl(protocol.EndSessionRequest{Type: "end_session"}, 3*time.Second, nil)
		s.r.shutdown()
	})
	return nil
}
