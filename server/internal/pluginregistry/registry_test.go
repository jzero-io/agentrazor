package pluginregistry

import (
	"strings"
	"testing"
)

type localeProvider map[string][]byte

func (p localeProvider) PluginLocales() map[string][]byte {
	return p
}

func TestRegistryAdminLocales(t *testing.T) {
	registry := New()

	if err := registry.RegisterProvider("legacy", struct{}{}); err != nil {
		t.Fatalf("register legacy plugin: %v", err)
	}
	if err := registry.RegisterProvider("plugin-b", localeProvider{
		"zh-CN": []byte(`{"route":{"plugin_b":"插件 B"}}`),
	}); err != nil {
		t.Fatalf("register plugin-b: %v", err)
	}
	if err := registry.RegisterProvider("plugin-a", localeProvider{
		"en-US": []byte(`{"route":{"plugin_a":"Plugin A"}}`),
		"zh-CN": []byte(`{"page":{"plugin":{"a":{"title":"插件 A"}}}}`),
	}); err != nil {
		t.Fatalf("register plugin-a: %v", err)
	}

	locales := registry.AdminLocales()
	if len(locales) != 2 {
		t.Fatalf("expected two locales, got %#v", locales)
	}

	zhCN := locales["zh-CN"].(map[string]interface{})
	route := zhCN["route"].(map[string]interface{})
	if route["plugin_b"] != "插件 B" {
		t.Fatalf("unexpected merged route messages: %#v", route)
	}
	page := zhCN["page"].(map[string]interface{})
	plugin := page["plugin"].(map[string]interface{})
	if _, ok := plugin["a"]; !ok {
		t.Fatalf("missing merged page messages: %#v", page)
	}

	route["plugin_b"] = "mutated"
	freshLocales := registry.AdminLocales()
	freshRoute := freshLocales["zh-CN"].(map[string]interface{})["route"].(map[string]interface{})
	if freshRoute["plugin_b"] != "插件 B" {
		t.Fatal("AdminLocales returned mutable registry state")
	}
}

func TestRegistryWithoutLocaleProvidersReturnsEmptyMap(t *testing.T) {
	registry := New()
	if err := registry.RegisterProvider("legacy", struct{}{}); err != nil {
		t.Fatalf("register legacy plugin: %v", err)
	}

	locales := registry.AdminLocales()
	if locales == nil || len(locales) != 0 {
		t.Fatalf("expected an empty non-nil locale map, got %#v", locales)
	}
}

func TestRegistryRejectsCollisionsAtomically(t *testing.T) {
	registry := New()
	if err := registry.RegisterProvider("first", localeProvider{
		"en-US": []byte(`{"route":{"shared":"First"}}`),
	}); err != nil {
		t.Fatalf("register first plugin: %v", err)
	}

	err := registry.RegisterProvider("second", localeProvider{
		"en-US": []byte(`{"route":{"shared":"Second"}}`),
		"zh-CN": []byte(`{"route":{"second":"第二个"}}`),
	})
	if err == nil || !strings.Contains(err.Error(), `route.shared`) {
		t.Fatalf("expected a collision error, got %v", err)
	}

	locales := registry.AdminLocales()
	if _, exists := locales["zh-CN"]; exists {
		t.Fatalf("failed registration partially added a locale: %#v", locales)
	}
}

func TestRegistryRejectsUnsafeAndMalformedBundles(t *testing.T) {
	tests := []struct {
		name   string
		bundle string
		want   string
	}{
		{
			name:   "unsafe key nested in array",
			bundle: `{"items":[{"__proto__":{"polluted":true}}]}`,
			want:   "unsafe key",
		},
		{
			name:   "trailing JSON",
			bundle: `{"route":{}} {"other":{}}`,
			want:   "trailing JSON data",
		},
		{
			name:   "non object",
			bundle: `[]`,
			want:   "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().RegisterProvider("bad", localeProvider{
				"en-US": []byte(tt.bundle),
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func TestRegistryRejectsDuplicatePlugin(t *testing.T) {
	registry := New()
	if err := registry.RegisterProvider("duplicate", struct{}{}); err != nil {
		t.Fatalf("register plugin: %v", err)
	}
	if err := registry.RegisterProvider("duplicate", struct{}{}); err == nil {
		t.Fatal("expected duplicate registration to fail")
	}
}
