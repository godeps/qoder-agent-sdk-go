package qodersdk

import (
	"errors"
	"fmt"
)

// Sentinel errors.
var (
	ErrAuthNotConfigured = errors.New("qoder: auth not configured")
	ErrCLINotFound       = errors.New("qoder: qoderclicn executable not found")
	ErrInitializeTimeout = errors.New("qoder: qoderclicn initialization timed out")
	ErrSessionClosed     = errors.New("qoder: session closed")
	// ErrSessionNotEstablished is returned by control operations (e.g.
	// SetModel, Interrupt) that the CLI only accepts once system/init has
	// arrived — in practice after the first user turn has been processed.
	ErrSessionNotEstablished = errors.New("qoder: session not established yet (send a turn first)")
)

// ProtocolVersionMismatchError is returned when the CLI announces a wire
// protocol version incompatible with this SDK (cross-major mismatch).
type ProtocolVersionMismatchError struct {
	Got  string
	Want string
}

func (e *ProtocolVersionMismatchError) Error() string {
	return fmt.Sprintf("qoder: protocol version mismatch: CLI announced %s, SDK built for %s", e.Got, e.Want)
}

// ControlRequestError is returned when a control request fails with an error
// response from the CLI.
type ControlRequestError struct {
	RequestID string
	Code      string
	Message   string
	Retryable bool
}

func (e *ControlRequestError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("qoder: control request %s failed: %s (%s)", e.RequestID, e.Message, e.Code)
	}
	return fmt.Sprintf("qoder: control request %s failed: %s", e.RequestID, e.Message)
}

// AuthAccessTokenEnvVarError indicates an access-token env var was not set.
type AuthAccessTokenEnvVarError struct {
	EnvVar string
}

func (e *AuthAccessTokenEnvVarError) Error() string {
	return fmt.Sprintf("qoder: access token env var %q not configured", e.EnvVar)
}

// AuthServiceAccountEnvVarError indicates a service-account env var was not set.
type AuthServiceAccountEnvVarError struct {
	EnvVar string
}

func (e *AuthServiceAccountEnvVarError) Error() string {
	return fmt.Sprintf("qoder: service account env var %q not configured", e.EnvVar)
}
