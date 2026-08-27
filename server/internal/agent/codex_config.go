package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultCodexConfigDir = "etc/codex"

var defaultCodexConfigFiles = []string{
	"config.toml",
	"models.json",
	"auth.json",
}

func ensureDefaultCodexConfig(codexHome string) error {
	info, err := os.Stat(defaultCodexConfigDir)
	if err != nil {
		return fmt.Errorf("stat default Codex config dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("default Codex config path is not a directory: %s", defaultCodexConfigDir)
	}

	if err := os.MkdirAll(codexHome, 0o700); err != nil {
		return fmt.Errorf("create Codex home: %w", err)
	}
	for _, name := range defaultCodexConfigFiles {
		target := filepath.Join(codexHome, name)
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat Codex config %s: %w", name, err)
		}

		source := filepath.Join(defaultCodexConfigDir, name)
		data, err := os.ReadFile(source)
		if err != nil {
			return fmt.Errorf("read default Codex config %s: %w", name, err)
		}
		if err := os.WriteFile(target, data, 0o600); err != nil {
			return fmt.Errorf("write Codex config %s: %w", name, err)
		}
	}
	return nil
}
