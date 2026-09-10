package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
	"github.com/pelletier/go-toml/v2"
)

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

type Settings struct {
	ctx     context.Context
	service *Service
	config  map[string]any
}

func (s *Service) Settings(ctx context.Context) (*Settings, error) {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.loadSettings(ctx)
}

func (s *Service) SaveSelection(ctx context.Context, providerID, model, effort string) error {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return err
	}
	return settings.saveSelection(providerID, model, effort)
}

func (s *Service) SaveProviderAPIKey(ctx context.Context, providerID, apiKey string) error {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	settings, err := s.loadSettings(ctx)
	if err != nil {
		return err
	}
	return settings.saveProviderAPIKey(providerID, apiKey)
}

func (s *Service) loadSettings(ctx context.Context) (*Settings, error) {
	content, err := s.readConfigFile(ctx, "config.toml")
	if err != nil {
		return nil, err
	}
	config := map[string]any{}
	if strings.TrimSpace(content) != "" {
		if err := toml.Unmarshal([]byte(content), &config); err != nil {
			return nil, fmt.Errorf("decode Codex config: %w", err)
		}
	}
	return &Settings{ctx: ctx, service: s, config: config}, nil
}

func (s *Settings) ActiveProvider() string {
	return configString(s.config, "model_provider")
}

func (s *Settings) Model() string {
	return configString(s.config, "model")
}

func (s *Settings) ReasoningEffort() string {
	return configString(s.config, "model_reasoning_effort")
}

func (s *Settings) Providers() ([]managetypes.ModelProvider, error) {
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
			HasApiKey: configString(configured, "experimental_bearer_token") != "",
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
			result = append(result, item.model)
		}
	}
	return result
}

func (s *Settings) catalogModels() (providerCatalogModels, error) {
	catalogPath := configString(s.config, "model_catalog_json")
	if catalogPath == "" {
		return nil, errors.New("model_catalog_json is missing from config.toml")
	}
	content, err := s.service.readConfigFile(s.ctx, catalogPath)
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

func (s *Settings) saveSelection(providerID, model, effort string) error {
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
	values := map[string]any{
		"model_provider": providerID,
		"model":          model,
	}
	if effort == "" {
		values["model_reasoning_effort"] = nil
	} else {
		values["model_reasoning_effort"] = effort
	}
	if err := s.service.writeConfigValues(s.ctx, values); err != nil {
		return err
	}
	effective, err := s.service.readEffectiveConfig(s.ctx)
	if err != nil {
		return fmt.Errorf("verify Codex config: %w", err)
	}
	if configString(effective, "model_provider") != providerID || configString(effective, "model") != model ||
		configString(effective, "model_reasoning_effort") != effort {
		return errors.New("Codex config did not apply the selected model")
	}
	return nil
}

func (s *Settings) saveProviderAPIKey(providerID, apiKey string) error {
	providerID, apiKey = strings.TrimSpace(providerID), strings.TrimSpace(apiKey)
	if apiKey == "" {
		return errors.New("provider API key is required")
	}
	catalogModels, err := s.catalogModels()
	if err != nil {
		return err
	}
	if !catalogModels.hasProvider(providerID) {
		return fmt.Errorf("unsupported provider %q", providerID)
	}
	providers, ok := s.config["model_providers"].(map[string]any)
	if !ok {
		return errors.New("model_providers is missing from config.toml")
	}
	_, ok = providers[providerID].(map[string]any)
	if !ok {
		return fmt.Errorf("provider %q does not support a config API key", providerID)
	}
	return s.service.writeConfigValues(s.ctx, map[string]any{
		"model_providers." + providerID + ".experimental_bearer_token": apiKey,
	})
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
