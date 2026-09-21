package pluginregistry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

// LocaleProvider is implemented by serverless plugins that ship Admin locale
// bundles. Plugins that do not implement this interface remain compatible and
// are simply registered without locales.
type LocaleProvider interface {
	PluginLocales() map[string][]byte
}

// Registry contains the runtime capabilities contributed by serverless
// plugins. Locale bundles are validated and merged when a plugin is loaded so
// requests only read an immutable snapshot.
type Registry struct {
	mu sync.RWMutex

	plugins        map[string]struct{}
	localeMessages map[string]map[string]interface{}
}

func New() *Registry {
	return &Registry{
		plugins:        make(map[string]struct{}),
		localeMessages: make(map[string]map[string]interface{}),
	}
}

// RegisterProvider registers a serverless plugin and, when supported, its
// embedded locale bundles. Registration is atomic: an invalid bundle or a
// collision leaves the registry unchanged.
func (r *Registry) RegisterProvider(name string, provider interface{}) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("plugin name is required")
	}

	var bundles map[string][]byte
	if localeProvider, ok := provider.(LocaleProvider); ok {
		bundles = localeProvider.PluginLocales()
	}

	parsed, err := parseBundles(name, bundles)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}

	nextMessages := cloneLocales(r.localeMessages)
	locales := make([]string, 0, len(parsed))
	for locale := range parsed {
		locales = append(locales, locale)
	}
	sort.Strings(locales)

	for _, locale := range locales {
		messages := nextMessages[locale]
		if messages == nil {
			messages = make(map[string]interface{})
			nextMessages[locale] = messages
		}
		if err := mergeMessages(messages, parsed[locale], ""); err != nil {
			return fmt.Errorf("register plugin %q locale %q: %w", name, locale, err)
		}
	}

	r.plugins[name] = struct{}{}
	r.localeMessages = nextMessages
	return nil
}

// AdminLocales returns a detached aggregate of all plugin Admin locales. An
// empty registry returns a non-nil map so the public API serializes it as {}.
func (r *Registry) AdminLocales() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	locales := make(map[string]interface{}, len(r.localeMessages))
	for locale, messages := range r.localeMessages {
		locales[locale] = cloneMessages(messages)
	}
	return locales
}

func parseBundles(plugin string, bundles map[string][]byte) (map[string]map[string]interface{}, error) {
	parsed := make(map[string]map[string]interface{}, len(bundles))
	for rawLocale, bundle := range bundles {
		locale := strings.TrimSpace(rawLocale)
		if locale == "" {
			return nil, fmt.Errorf("plugin %q has an empty locale name", plugin)
		}
		if _, exists := parsed[locale]; exists {
			return nil, fmt.Errorf("plugin %q contains duplicate locale %q", plugin, locale)
		}

		decoder := json.NewDecoder(bytes.NewReader(bundle))
		decoder.UseNumber()
		var messages map[string]interface{}
		if err := decoder.Decode(&messages); err != nil {
			return nil, fmt.Errorf("plugin %q locale %q is invalid JSON: %w", plugin, locale, err)
		}
		if messages == nil {
			return nil, fmt.Errorf("plugin %q locale %q must contain a JSON object", plugin, locale)
		}
		var trailing interface{}
		if err := decoder.Decode(&trailing); err != io.EOF {
			return nil, fmt.Errorf("plugin %q locale %q contains trailing JSON data", plugin, locale)
		}
		if err := validateMessages(messages, ""); err != nil {
			return nil, fmt.Errorf("plugin %q locale %q: %w", plugin, locale, err)
		}
		parsed[locale] = messages
	}
	return parsed, nil
}

func validateMessages(messages map[string]interface{}, parent string) error {
	for key, value := range messages {
		path := key
		if parent != "" {
			path = parent + "." + key
		}
		switch key {
		case "__proto__", "constructor", "prototype":
			return fmt.Errorf("unsafe key %q", path)
		}
		if err := validateValue(value, path); err != nil {
			return err
		}
	}
	return nil
}

func validateValue(value interface{}, path string) error {
	switch typed := value.(type) {
	case map[string]interface{}:
		return validateMessages(typed, path)
	case []interface{}:
		for i, item := range typed {
			if err := validateValue(item, fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func mergeMessages(target, source map[string]interface{}, parent string) error {
	keys := make([]string, 0, len(source))
	for key := range source {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		value := source[key]
		path := key
		if parent != "" {
			path = parent + "." + key
		}
		current, exists := target[key]
		if !exists {
			target[key] = cloneValue(value)
			continue
		}

		currentMessages, currentOK := current.(map[string]interface{})
		valueMessages, valueOK := value.(map[string]interface{})
		if currentOK && valueOK {
			if err := mergeMessages(currentMessages, valueMessages, path); err != nil {
				return err
			}
			continue
		}
		return fmt.Errorf("locale key %q is already owned by another plugin", path)
	}
	return nil
}

func cloneLocales(locales map[string]map[string]interface{}) map[string]map[string]interface{} {
	cloned := make(map[string]map[string]interface{}, len(locales))
	for locale, messages := range locales {
		cloned[locale] = cloneMessages(messages)
	}
	return cloned
}

func cloneMessages(messages map[string]interface{}) map[string]interface{} {
	if messages == nil {
		return nil
	}
	cloned := make(map[string]interface{}, len(messages))
	for key, value := range messages {
		cloned[key] = cloneValue(value)
	}
	return cloned
}

func cloneValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return cloneMessages(typed)
	case []interface{}:
		cloned := make([]interface{}, len(typed))
		for i := range typed {
			cloned[i] = cloneValue(typed[i])
		}
		return cloned
	default:
		return typed
	}
}
