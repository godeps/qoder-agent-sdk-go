package qodersdk

import (
	"encoding/json"
	"time"

	"github.com/godeps/qoder-agent-sdk-go/protocol"
)

// BYOK (Bring Your Own Key) management. Mirrors the TypeScript SDK's
// byok control surface: query the server-side provider catalog, validate
// credentials, and CRUD persisted configs through the live CLI session.

// GetByokConfig returns the server-side BYOK provider catalog (providers,
// required fields, model groups).
func (s *Session) GetByokConfig() (*protocol.GetByokConfigResponse, error) {
	var out protocol.GetByokConfigResponse
	if err := s.sendControl(protocol.GetByokConfigRequest{Type: "get_byok_config"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ValidateByokModel validates a provider/model/API-key combination through
// the CLI's server-side BYOK check endpoint. The key is sent once and never
// persisted by this call.
func (s *Session) ValidateByokModel(provider, model, apiKey, url, style string) (bool, error) {
	var out protocol.ValidateByokModelResponse
	req := protocol.ValidateByokModelRequest{
		Type:     "validate_byok_model",
		Provider: provider,
		Model:    model,
		APIKey:   apiKey,
		URL:      url,
		Style:    style,
	}
	if err := s.sendControl(req, 60*time.Second, &out); err != nil {
		return false, err
	}
	return out.Success, nil
}

// ListByokConfigs lists persisted BYOK configs (secret-free).
func (s *Session) ListByokConfigs() ([]protocol.ByokModelConfigInfo, error) {
	var out protocol.ListByokConfigsResponse
	if err := s.sendControl(protocol.ListByokConfigsRequest{Type: "list_byok_configs"}, 30*time.Second, &out); err != nil {
		return nil, err
	}
	return out.Configs, nil
}

// ByokModelConfigInput creates/updates a per-model BYOK config.
type ByokModelConfigInput struct {
	Provider        string            `json:"provider"`
	Model           string            `json:"model"`
	Parameters      map[string]string `json:"parameters"`
	Key             string            `json:"key,omitempty"`
	DisplayName     string            `json:"displayName,omitempty"`
	Type            string            `json:"type,omitempty"`
	BaseURL         string            `json:"baseUrl,omitempty"`
	Style           string            `json:"style,omitempty"`
	ReasoningEffort string            `json:"reasoningEffort,omitempty"`
	ContextWindow   int               `json:"contextWindow,omitempty"`
}

// CustomByokProviderModelInput declares one model of a custom provider.
type CustomByokProviderModelInput struct {
	Model            string   `json:"model"`
	DisplayName      string   `json:"displayName,omitempty"`
	ContextWindow    int      `json:"contextWindow,omitempty"`
	MaxOutputTokens  int      `json:"maxOutputTokens,omitempty"`
	ReasoningEfforts []string `json:"reasoningEfforts,omitempty"`
}

// CustomByokProviderConfigInput creates/updates a whole custom provider
// (openai-compatible / alibaba / aliyun-bailian endpoints).
type CustomByokProviderConfigInput struct {
	ProviderID  string                         `json:"providerId"`
	BaseURL     string                         `json:"baseUrl"`
	APIKey      string                         `json:"apiKey"`
	Type        string                         `json:"type,omitempty"`     // openai-compatible | alibaba | aliyun-bailian
	Protocol    string                         `json:"protocol,omitempty"` // openai | openai-responses | anthropic
	AuthType    string                         `json:"authType,omitempty"` // bearer | api-key
	DisplayName string                         `json:"displayName,omitempty"`
	Model       string                         `json:"model,omitempty"`
	Models      []CustomByokProviderModelInput `json:"models,omitempty"`
}

// CreateByokModelConfig persists a per-model BYOK config. Returns the
// reference (key) of the created config.
func (s *Session) CreateByokModelConfig(cfg ByokModelConfigInput) (*protocol.ByokConfigReference, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return s.createByok(raw)
}

// CreateByokCustomProvider persists a custom-provider BYOK config.
func (s *Session) CreateByokCustomProvider(cfg CustomByokProviderConfigInput) (*protocol.ByokConfigReference, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return s.createByok(raw)
}

func (s *Session) createByok(raw json.RawMessage) (*protocol.ByokConfigReference, error) {
	var out protocol.CreateByokConfigResponse
	req := protocol.CreateByokConfigRequest{Type: "create_byok_config", Config: raw}
	if err := s.sendControl(req, 60*time.Second, &out); err != nil {
		return nil, err
	}
	return &out.Config, nil
}

// UpdateByokModelConfigInput updates fields of a persisted per-model config.
type UpdateByokModelConfigInput struct {
	Key             string            `json:"key"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	DisplayName     string            `json:"displayName,omitempty"`
	BaseURL         string            `json:"baseUrl,omitempty"`
	Style           string            `json:"style,omitempty"`
	ReasoningEffort string            `json:"reasoningEffort,omitempty"`
	ContextWindow   int               `json:"contextWindow,omitempty"`
}

// UpdateByokModelConfig updates a persisted per-model BYOK config.
func (s *Session) UpdateByokModelConfig(cfg UpdateByokModelConfigInput) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.sendControl(protocol.UpdateByokConfigRequest{Type: "update_byok_config", Config: raw}, 60*time.Second, nil)
}

// UpdateByokCustomProvider updates a persisted custom-provider config.
func (s *Session) UpdateByokCustomProvider(cfg CustomByokProviderConfigInput) error {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return s.sendControl(protocol.UpdateByokConfigRequest{Type: "update_byok_config", Config: raw}, 60*time.Second, nil)
}

// DeleteByokConfigByKey deletes a per-model BYOK config by key.
func (s *Session) DeleteByokConfigByKey(key string) error {
	raw, _ := json.Marshal(map[string]string{"key": key})
	return s.sendControl(protocol.DeleteByokConfigRequest{Type: "delete_byok_config", Target: raw}, 30*time.Second, nil)
}

// DeleteByokConfigByProvider deletes a custom-provider BYOK config.
func (s *Session) DeleteByokConfigByProvider(providerID string) error {
	raw, _ := json.Marshal(map[string]string{"providerId": providerID})
	return s.sendControl(protocol.DeleteByokConfigRequest{Type: "delete_byok_config", Target: raw}, 30*time.Second, nil)
}

// CheckByokModelConfig verifies a per-model config would work, without
// persisting it. Failure reasons are credential-redacted by the CLI.
func (s *Session) CheckByokModelConfig(cfg ByokModelConfigInput) (*protocol.ByokConfigCheckResult, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return s.checkByok(raw)
}

// CheckByokCustomProvider verifies a custom-provider config without
// persisting it.
func (s *Session) CheckByokCustomProvider(cfg CustomByokProviderConfigInput) (*protocol.ByokConfigCheckResult, error) {
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	return s.checkByok(raw)
}

func (s *Session) checkByok(raw json.RawMessage) (*protocol.ByokConfigCheckResult, error) {
	var out protocol.ByokConfigCheckResult
	req := protocol.CheckByokConfigRequest{Type: "check_byok_config", Config: raw}
	if err := s.sendControl(req, 60*time.Second, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
