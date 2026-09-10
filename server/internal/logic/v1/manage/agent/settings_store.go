package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
	"github.com/pelletier/go-toml/v2"
)

var (
	agentSettingsMu     = sync.RWMutex{}
	allReasoningEfforts = []string{"low", "medium", "high", "xhigh", "max", "ultra"}
)

const openAIProviderID = "openai"
const providerAPIKeyField = "experimental_bearer_token"

type modelCatalog struct {
	Models []catalogModel `json:"models"`
}

type catalogModel struct {
	Slug                     string                  `json:"slug"`
	DisplayName              string                  `json:"display_name"`
	Provider                 string                  `json:"provider"`
	DefaultReasoningLevel    string                  `json:"default_reasoning_level"`
	SupportedReasoningLevels []catalogReasoningLevel `json:"supported_reasoning_levels"`
	Visibility               string                  `json:"visibility"`
}

type catalogReasoningLevel struct {
	Effort string `json:"effort"`
}

type agentSettingsFiles interface {
	ReadConfigFile(context.Context, string) (string, error)
	WriteConfigFile(context.Context, string, string) error
}

type agentSettingsStore struct {
	ctx     context.Context
	runtime agentSettingsFiles
	config  map[string]any
}

func readAgentSettingsStore(ctx context.Context, runtime *agentdomain.CodexAppServerClient) (*agentSettingsStore, error) {
	agentSettingsMu.RLock()
	defer agentSettingsMu.RUnlock()
	return loadAgentSettingsStoreFromRuntime(ctx, runtime)
}

func updateAgentSettingsStore(ctx context.Context, runtime *agentdomain.CodexAppServerClient, update func(*agentSettingsStore) error) error {
	agentSettingsMu.Lock()
	defer agentSettingsMu.Unlock()
	store, err := loadAgentSettingsStoreFromRuntime(ctx, runtime)
	if err != nil {
		return err
	}
	return update(store)
}

func loadAgentSettingsStoreFromRuntime(ctx context.Context, runtime agentSettingsFiles) (*agentSettingsStore, error) {
	content, err := runtime.ReadConfigFile(ctx, "config.toml")
	if err != nil {
		return nil, err
	}
	config := map[string]any{}
	if strings.TrimSpace(content) != "" {
		if err := toml.Unmarshal([]byte(content), &config); err != nil {
			return nil, fmt.Errorf("decode Codex config: %w", err)
		}
	}
	return &agentSettingsStore{ctx: ctx, runtime: runtime, config: config}, nil
}

type localAgentSettingsFiles struct {
	codexHome string
}

func (f localAgentSettingsFiles) ReadConfigFile(_ context.Context, name string) (string, error) {
	return readAgentConfigFile(f.codexHome, name)
}

func (f localAgentSettingsFiles) WriteConfigFile(_ context.Context, name, content string) error {
	return writeAgentConfigFile(f.codexHome, name, content)
}

// loadAgentSettingsStore keeps unit tests focused on configuration semantics;
// production handlers use loadAgentSettingsStoreFromRuntime through the
// app-server socket.
func loadAgentSettingsStore(codexHome string) (*agentSettingsStore, error) {
	return loadAgentSettingsStoreFromRuntime(context.Background(), localAgentSettingsFiles{codexHome: codexHome})
}

func (s *agentSettingsStore) activeProvider() string {
	return configString(s.config, "model_provider")
}

func (s *agentSettingsStore) model() string {
	return configString(s.config, "model")
}

func (s *agentSettingsStore) reasoningEffort() string {
	return configString(s.config, "model_reasoning_effort")
}

func (s *agentSettingsStore) providers() ([]managetypes.ModelProvider, error) {
	catalogModels, err := s.catalogModels()
	if err != nil {
		return nil, err
	}
	providerConfig, _ := s.config["model_providers"].(map[string]any)
	providerIDs := catalogModels.providerIDs()
	providers := make([]managetypes.ModelProvider, 0, len(providerIDs))
	for _, providerID := range providerIDs {
		configured, _ := providerConfig[providerID].(map[string]any)
		providers = append(providers, managetypes.ModelProvider{
			Id:        providerID,
			Name:      providerID,
			BaseUrl:   configString(configured, "base_url"),
			WireApi:   configString(configured, "wire_api"),
			HasApiKey: providerID != openAIProviderID && configString(configured, providerAPIKeyField) != "",
			Models:    catalogModels.forProvider(providerID),
		})
	}
	return providers, nil
}

type providerCatalogModel struct {
	provider string
	model    managetypes.ProviderModel
}

type providerCatalogModels []providerCatalogModel

func (models providerCatalogModels) providerIDs() []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, item := range models {
		if !seen[item.provider] {
			seen[item.provider] = true
			result = append(result, item.provider)
		}
	}
	return result
}

func (models providerCatalogModels) hasProvider(providerID string) bool {
	for _, item := range models {
		if item.provider == providerID {
			return true
		}
	}
	return false
}

func (models providerCatalogModels) forProvider(providerID string) []managetypes.ProviderModel {
	result := make([]managetypes.ProviderModel, 0)
	for _, item := range models {
		if item.provider == providerID {
			result = append(result, cloneProviderModel(item.model))
		}
	}
	return result
}

func (s *agentSettingsStore) catalogModels() (providerCatalogModels, error) {
	content, err := s.runtime.ReadConfigFile(s.ctx, "models.json")
	if err != nil {
		return nil, err
	}
	models := make(providerCatalogModels, 0)
	if strings.TrimSpace(content) == "" {
		return models, nil
	}
	var catalog modelCatalog
	if err := json.Unmarshal([]byte(content), &catalog); err != nil {
		return nil, fmt.Errorf("decode Codex model catalog: %w", err)
	}
	for _, raw := range catalog.Models {
		if strings.EqualFold(strings.TrimSpace(raw.Visibility), "hide") {
			continue
		}
		id := strings.TrimSpace(raw.Slug)
		if id == "" {
			continue
		}
		providerID := strings.TrimSpace(raw.Provider)
		if providerID == "" {
			return nil, fmt.Errorf("model %q has no provider in Codex model catalog", id)
		}
		efforts := make([]string, 0, len(raw.SupportedReasoningLevels))
		for _, level := range raw.SupportedReasoningLevels {
			if effort := strings.TrimSpace(level.Effort); effort != "" {
				efforts = append(efforts, effort)
			}
		}
		if len(efforts) == 0 {
			efforts = cloneStrings(allReasoningEfforts)
		}
		models = append(models, providerCatalogModel{
			provider: providerID,
			model: managetypes.ProviderModel{
				Id: id, Name: firstNonEmpty(strings.TrimSpace(raw.DisplayName), id),
				DefaultReasoningEffort: strings.TrimSpace(raw.DefaultReasoningLevel), ReasoningEfforts: uniqueStrings(efforts),
			},
		})
	}
	return models, nil
}

func cloneProviderModel(model managetypes.ProviderModel) managetypes.ProviderModel {
	model.ReasoningEfforts = cloneStrings(model.ReasoningEfforts)
	return model
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}

func (s *agentSettingsStore) saveSelection(providerID, model, effort string) error {
	providerID, model, effort = strings.TrimSpace(providerID), strings.TrimSpace(model), strings.TrimSpace(effort)
	catalogModels, err := s.catalogModels()
	if err != nil {
		return err
	}
	if !catalogModels.hasProvider(providerID) {
		return fmt.Errorf("unsupported provider %q", providerID)
	}
	if !containsModel(catalogModels.forProvider(providerID), model) {
		return fmt.Errorf("model %q does not belong to provider %q", model, providerID)
	}
	s.config["model_provider"] = providerID
	s.config["model"] = model
	s.config["model_catalog_json"] = "models.json"
	if effort == "" {
		delete(s.config, "model_reasoning_effort")
	} else {
		s.config["model_reasoning_effort"] = effort
	}
	return s.writeConfig()
}

func (s *agentSettingsStore) saveProviderAPIKey(providerID, apiKey string) error {
	providerID, apiKey = strings.TrimSpace(providerID), strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("provider API key is required")
	}
	catalogModels, err := s.catalogModels()
	if err != nil {
		return err
	}
	if providerID == openAIProviderID || !catalogModels.hasProvider(providerID) {
		return fmt.Errorf("provider %q does not support a config API key", providerID)
	}
	providers, ok := s.config["model_providers"].(map[string]any)
	if !ok {
		return errors.New("model_providers is missing from config.toml")
	}
	configured, ok := providers[providerID].(map[string]any)
	if !ok {
		return fmt.Errorf("provider %q is missing from config.toml", providerID)
	}
	configured[providerAPIKeyField] = apiKey
	return s.writeConfig()
}

func (s *agentSettingsStore) writeConfig() error {
	data, err := toml.Marshal(s.config)
	if err != nil {
		return fmt.Errorf("encode Codex config: %w", err)
	}
	return s.runtime.WriteConfigFile(s.ctx, "config.toml", string(data))
}

func configString(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func containsModel(models []managetypes.ProviderModel, id string) bool {
	for _, model := range models {
		if model.Id == id {
			return true
		}
	}
	return false
}
