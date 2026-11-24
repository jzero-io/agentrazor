package locales

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed *.json
var localeFiles embed.FS

func Bundles() map[string][]byte {
	entries, err := localeFiles.ReadDir(".")
	if err != nil {
		panic(fmt.Errorf("read embedded Admin locales: %w", err))
	}
	bundles := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		data, err := localeFiles.ReadFile(entry.Name())
		if err != nil {
			panic(fmt.Errorf("read embedded Admin locale %q: %w", entry.Name(), err))
		}
		bundles[strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))] = data
	}
	return bundles
}
