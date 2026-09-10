// Package auth configures SDK→CLI authentication.
//
// The SDK writes a one-shot auth payload JSON to a temp file and passes its
// path to qoderclicn via the QODER_SDK_AUTH_PAYLOAD_FILE environment
// variable. The CLI reads the payload, initializes auth, then deletes the
// file; the SDK removes the temp directory on shutdown as a safety net.
package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultAccessTokenEnvVar is the default env var for an access token.
	DefaultAccessTokenEnvVar = "QODERCN_PERSONAL_ACCESS_TOKEN"
	// DefaultServiceAccountEnvVar is the default env var for a service account key.
	DefaultServiceAccountEnvVar = "QODERCN_SERVICE_ACCOUNT_KEY"
)

// ErrNotConfigured is returned when no auth method is configured.
var ErrNotConfigured = errors.New("qoder: auth not configured")

// ServiceAccountToken is a short-lived service account token.
type ServiceAccountToken struct {
	Token     string
	ExpiresAt int64 // unix ms; 0 if unknown
}

// JobTokenRequest describes why a job token is needed.
type JobTokenRequest struct {
	RequestID string
	Reason    string // initial | expiring | unauthorized
	SessionID string
}

// JobTokenResult is a fetched job token.
type JobTokenResult struct {
	Token     string
	ExpiresAt int64 // unix ms; 0 if unknown
}

// FetchServiceAccountToken fetches a short-lived SAT (host-provided).
type FetchServiceAccountToken func(ctx context.Context) (*ServiceAccountToken, error)

// FetchJobToken fetches a job token (host-provided).
type FetchJobToken func(ctx context.Context, req JobTokenRequest) (*JobTokenResult, error)

// AuthOptions configures one of the authentication methods. Construct one with
// AccessToken / AccessTokenFromEnv / QodercliAuth / ServiceAccount /
// ServiceAccountWithTokenFetcher / JobToken.
type AuthOptions struct {
	kind                     string
	accessToken              string
	envVar                   string
	serviceAccountKey        string
	serviceAccountEnvVar     string
	fetchServiceAccountToken FetchServiceAccountToken
	fetchJobToken            FetchJobToken
}

// AccessToken uses a personal access token provided directly.
func AccessToken(token string) AuthOptions {
	return AuthOptions{kind: "accessToken", accessToken: token}
}

// AccessTokenFromEnv reads a personal access token from an env var at payload
// time. If envVar is empty, DefaultAccessTokenEnvVar is used.
func AccessTokenFromEnv(envVar ...string) AuthOptions {
	v := DefaultAccessTokenEnvVar
	if len(envVar) > 0 && envVar[0] != "" {
		v = envVar[0]
	}
	return AuthOptions{kind: "accessToken", envVar: v}
}

// QodercliAuth reuses the local qoderclicn login state.
func QodercliAuth() AuthOptions { return AuthOptions{kind: "qodercli"} }

// ServiceAccount uses a service account key provided directly.
func ServiceAccount(key string) AuthOptions {
	return AuthOptions{kind: "serviceAccount", serviceAccountKey: key}
}

// ServiceAccountFromEnv reads a service account key from an env var.
func ServiceAccountFromEnv(envVar ...string) AuthOptions {
	v := DefaultServiceAccountEnvVar
	if len(envVar) > 0 && envVar[0] != "" {
		v = envVar[0]
	}
	return AuthOptions{kind: "serviceAccount", serviceAccountEnvVar: v}
}

// ServiceAccountWithTokenFetcher lets the host supply short-lived SATs.
func ServiceAccountWithTokenFetcher(f FetchServiceAccountToken) AuthOptions {
	return AuthOptions{kind: "serviceAccount", fetchServiceAccountToken: f}
}

// JobToken uses a host-provided job token fetcher.
func JobToken(f FetchJobToken) AuthOptions {
	return AuthOptions{kind: "jobToken", fetchJobToken: f}
}

// Configured reports whether an auth method was set.
func (o AuthOptions) Configured() bool { return o.kind != "" }

// SdkAuthPayload is the JSON structure written to the one-shot payload file.
type SdkAuthPayload struct {
	Type                        string `json:"type"`
	AccessToken                 string `json:"accessToken,omitempty"`
	ServiceAccountKey           string `json:"serviceAccountKey,omitempty"`
	JobTokenProvider            string `json:"jobTokenProvider,omitempty"`
	ServiceAccountTokenProvider string `json:"serviceAccountTokenProvider,omitempty"`
}

// Payload builds the wire payload, resolving env vars as needed.
func (o AuthOptions) Payload() (SdkAuthPayload, error) {
	switch o.kind {
	case "accessToken":
		tok := o.accessToken
		if tok == "" && o.envVar != "" {
			tok = os.Getenv(o.envVar)
		}
		if tok == "" {
			return SdkAuthPayload{}, fmt.Errorf("qoder: access token not available (env var %q)", o.envVar)
		}
		return SdkAuthPayload{Type: "accessToken", AccessToken: tok}, nil
	case "qodercli":
		return SdkAuthPayload{Type: "qodercli"}, nil
	case "serviceAccount":
		if o.fetchServiceAccountToken != nil {
			return SdkAuthPayload{Type: "serviceAccount", ServiceAccountTokenProvider: "host"}, nil
		}
		key := o.serviceAccountKey
		if key == "" && o.serviceAccountEnvVar != "" {
			key = os.Getenv(o.serviceAccountEnvVar)
		}
		if key == "" {
			return SdkAuthPayload{}, fmt.Errorf("qoder: service account key not available (env var %q)", o.serviceAccountEnvVar)
		}
		return SdkAuthPayload{Type: "serviceAccount", ServiceAccountKey: key}, nil
	case "jobToken":
		if o.fetchJobToken == nil {
			return SdkAuthPayload{}, errors.New("qoder: job token fetcher not set")
		}
		return SdkAuthPayload{Type: "jobToken", JobTokenProvider: "host"}, nil
	}
	return SdkAuthPayload{}, ErrNotConfigured
}

// WritePayloadFile writes the auth payload to a temp file (mode 0o600) inside
// a fresh temp directory and returns the file path. Call RemovePayloadFile to
// clean up (the CLI also deletes the file after reading it).
func WritePayloadFile(o AuthOptions) (string, error) {
	if !o.Configured() {
		return "", ErrNotConfigured
	}
	payload, err := o.Payload()
	if err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "qoder-sdk-auth-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "payload.json")
	data, err := json.Marshal(payload)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return path, nil
}

// RemovePayloadFile removes the payload file and its parent temp directory.
func RemovePayloadFile(path string) {
	if path == "" {
		return
	}
	_ = os.RemoveAll(filepath.Dir(path))
}
