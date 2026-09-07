package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	managetypes "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

func writeTestModelCatalog(t *testing.T, home string) {
	t.Helper()
	catalog := `{"models":[
{"provider":"openai","slug":"gpt-5.6-sol","display_name":"GPT-5.6-Sol","visibility":"list","default_reasoning_level":"low","supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"},{"effort":"ultra"}]},
{"provider":"openai","slug":"gpt-hidden","display_name":"GPT Hidden","visibility":"hide","default_reasoning_level":"medium","supported_reasoning_levels":[]},
{"provider":"deepseek","slug":"deepseek-v4-flash","display_name":"DeepSeek-V4-Flash","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]},
{"provider":"deepseek","slug":"deepseek-v4-pro","display_name":"DeepSeek-V4-Pro","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]},
{"provider":"deepseek","slug":"deepseek-v4-flash-vision-exp","display_name":"DeepSeek-V4-Flash-Vision","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]},
{"provider":"ZAI","slug":"glm-5.3","display_name":"GLM-5.3","default_reasoning_level":"max","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]},
{"provider":"ZAI","slug":"glm-5-turbo","display_name":"GLM-5-Turbo","default_reasoning_level":"max","supported_reasoning_levels":[]},
{"provider":"kimi","slug":"k3-256k","display_name":"Kimi K3 256K","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]},
{"provider":"kimi","slug":"k3","display_name":"Kimi K3","default_reasoning_level":"high","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"},{"effort":"max"}]}
]}`
	if err := os.WriteFile(filepath.Join(home, "models.json"), []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
}

func providerByID(t *testing.T, providers []managetypes.ModelProvider, id string) managetypes.ModelProvider {
	t.Helper()
	for _, provider := range providers {
		if provider.Id == id {
			return provider
		}
	}
	t.Fatalf("provider %q was not returned: %#v", id, providers)
	return managetypes.ModelProvider{}
}

func TestProviderAPIKeyIsWrittenToConfig(t *testing.T) {
	home := t.TempDir()
	config := `model = "deepseek-v4-flash"
model_provider = "deepseek"

[model_providers.deepseek]
name = "deepseek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"

[model_providers.ZAI]
name = "ZAI"
base_url = "https://open.bigmodel.cn/api/v1"
wire_api = "responses"

[model_providers.kimi]
name = "kimi"
base_url = "https://api.kimi.com/coding/v1"
wire_api = "responses"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, providerID := range []string{"deepseek", "ZAI", "kimi"} {
		if err := store.saveProviderAPIKey(providerID, providerID+"-secret"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.saveProviderAPIKey("unsupported", "secret-value"); err == nil {
		t.Fatal("expected an unsupported provider to be rejected")
	}
	data, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	configText := string(data)
	for _, expected := range []string{
		"[model_providers.deepseek]", "deepseek-secret", "https://api.deepseek.com/",
		"[model_providers.ZAI]", "name = 'ZAI'", "ZAI-secret", "https://open.bigmodel.cn/api/v1",
		"[model_providers.kimi]", "name = 'kimi'", "kimi-secret", "https://api.kimi.com/coding/v1",
	} {
		if !strings.Contains(configText, expected) {
			t.Fatalf("generated config.toml does not contain %q: %s", expected, data)
		}
	}
	if strings.Count(configText, "experimental_bearer_token") != 3 {
		t.Fatalf("expected one token field per external provider: %s", data)
	}
	reloaded, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := reloaded.providers()
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 4 {
		t.Fatalf("provider API key status was not read from config.toml: %#v", providers)
	}
	for _, providerID := range []string{"deepseek", "ZAI", "kimi"} {
		if !providerByID(t, providers, providerID).HasApiKey {
			t.Fatalf("provider %q API key status was not read from config.toml: %#v", providerID, providers)
		}
	}
}

func TestAgentSettingsStoreSelectionUsesCatalogProviders(t *testing.T) {
	home := t.TempDir()
	config := `[model_providers.deepseek]
name = "DeepSeek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	for _, selection := range []struct {
		provider string
		model    string
		effort   string
	}{
		{provider: "deepseek", model: "deepseek-v4-flash", effort: "high"},
		{provider: "ZAI", model: "glm-5.3", effort: "max"},
		{provider: "kimi", model: "k3-256k", effort: "high"},
	} {
		if err := store.saveSelection(selection.provider, selection.model, selection.effort); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.saveSelection("custom_one", "model-a", "high"); err == nil {
		t.Fatal("expected an unsupported provider to be rejected")
	}
	reloaded, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.activeProvider() != "kimi" || reloaded.model() != "k3-256k" || reloaded.reasoningEffort() != "high" {
		t.Fatalf("selection was not persisted: provider=%q model=%q effort=%q", reloaded.activeProvider(), reloaded.model(), reloaded.reasoningEffort())
	}
}

func TestCatalogModelsBelongToTheirProvider(t *testing.T) {
	home := t.TempDir()
	config := `model = "deepseek-v4-flash"
model_provider = "deepseek"

[model_providers.deepseek]
name = "DeepSeek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := store.providers()
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 4 {
		t.Fatalf("unexpected providers: %#v", providers)
	}
	if !containsModel(providers[0].Models, "gpt-5.6-sol") {
		t.Fatalf("OpenAI catalog models are missing: %#v", providers[0].Models)
	}
	if containsModel(providers[0].Models, "deepseek-v4-pro") {
		t.Fatalf("active provider models leaked into OpenAI: %#v", providers[0].Models)
	}
	if !containsModel(providers[1].Models, "deepseek-v4-flash") || !containsModel(providers[1].Models, "deepseek-v4-pro") {
		t.Fatalf("DeepSeek catalog models are missing: %#v", providers[1].Models)
	}
	if containsModel(providers[1].Models, "glm-5.3") {
		t.Fatalf("another provider.s catalog models leaked into DeepSeek: %#v", providers[1].Models)
	}
}

func TestInactiveDeepSeekProviderUsesCatalogModels(t *testing.T) {
	home := t.TempDir()
	config := `model = "gpt-5.6-sol"
model_provider = "openai"

[model_providers.deepseek]
name = "DeepSeek"
base_url = "https://api.deepseek.com/"
wire_api = "responses"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := store.providers()
	if err != nil {
		t.Fatal(err)
	}
	if len(providers) != 4 {
		t.Fatalf("unexpected providers: %#v", providers)
	}
	if !containsModel(providers[1].Models, "deepseek-v4-flash") ||
		!containsModel(providers[1].Models, "deepseek-v4-pro") ||
		!containsModel(providers[1].Models, "deepseek-v4-flash-vision-exp") {
		t.Fatalf("inactive DeepSeek catalog models are missing: %#v", providers[1].Models)
	}
	if providers[1].Models[0].DefaultReasoningEffort != "high" {
		t.Fatalf("DeepSeek catalog details were not loaded: %#v", providers[1].Models)
	}
}

func TestZAIAndKimiProviderCatalogs(t *testing.T) {
	home := t.TempDir()
	config := `[model_providers.ZAI]
name = "ignored display name"
base_url = "https://open.bigmodel.cn/api/v1"
wire_api = "responses"

[model_providers.kimi]
name = "also ignored"
base_url = "https://api.kimi.com/coding/v1"
wire_api = "responses"
`
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	providers, err := store.providers()
	if err != nil {
		t.Fatal(err)
	}

	zai := providerByID(t, providers, "ZAI")
	if zai.Name != "ZAI" || zai.BaseUrl != "https://open.bigmodel.cn/api/v1" || zai.WireApi != "responses" {
		t.Fatalf("unexpected ZAI provider: %#v", zai)
	}
	if len(zai.Models) != 2 || zai.Models[0].Id != "glm-5.3" || zai.Models[0].DefaultReasoningEffort != "max" {
		t.Fatalf("unexpected GLM catalog: %#v", zai.Models)
	}

	kimi := providerByID(t, providers, "kimi")
	if kimi.Name != "kimi" || kimi.BaseUrl != "https://api.kimi.com/coding/v1" || kimi.WireApi != "responses" {
		t.Fatalf("unexpected Kimi provider: %#v", kimi)
	}
	if len(kimi.Models) != 2 || kimi.Models[0].Id != "k3-256k" || kimi.Models[0].DefaultReasoningEffort != "high" {
		t.Fatalf("unexpected Kimi catalog: %#v", kimi.Models)
	}
}

func TestCatalogRejectsModelWithoutProvider(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	catalog := `{"models":[{"slug":"model-without-provider","display_name":"Missing Provider","visibility":"list"}]}`
	if err := os.WriteFile(filepath.Join(home, "models.json"), []byte(catalog), 0o600); err != nil {
		t.Fatal(err)
	}
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.providers(); err == nil || !strings.Contains(err.Error(), "has no provider") {
		t.Fatalf("expected missing provider to be rejected, got %v", err)
	}
}

func TestOpenAIUsesOnlyVisibleCatalogModels(t *testing.T) {
	home := t.TempDir()
	config := "model = \"gpt-5.6-sol\"\nmodel_provider = \"openai\"\n"
	if err := os.WriteFile(filepath.Join(home, "config.toml"), []byte(config), 0o600); err != nil {
		t.Fatal(err)
	}
	writeTestModelCatalog(t, home)
	store, err := loadAgentSettingsStore(home)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.saveSelection("openai", "gpt-5.6-sol", "medium"); err != nil {
		t.Fatal(err)
	}
	configData, err := os.ReadFile(filepath.Join(home, "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(configData), "model_catalog_json") || !strings.Contains(string(configData), "models.json") {
		t.Fatalf("OpenAI selection did not retain models.json as its catalog: %s", configData)
	}
	providers, err := store.providers()
	if err != nil {
		t.Fatal(err)
	}
	if !containsModel(providers[0].Models, "gpt-5.6-sol") {
		t.Fatalf("visible OpenAI catalog model is missing: %#v", providers[0].Models)
	}
	if containsModel(providers[0].Models, "gpt-hidden") {
		t.Fatalf("hidden OpenAI catalog model must not be returned: %#v", providers[0].Models)
	}
	if _, err := os.Stat(filepath.Join(home, "providers.json")); !os.IsNotExist(err) {
		t.Fatalf("providers.json must not be created, stat error: %v", err)
	}
}
