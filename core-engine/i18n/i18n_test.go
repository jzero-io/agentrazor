package i18n

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslateFromDirectory(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"zh-CN.json": `{"test":{"message":"中文消息"}}`,
		"en-US.json": `{"test":{"message":"English message"}}`,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write locale %s: %v", name, err)
		}
	}

	trans := NewTranslator(I18nConf{Dir: dir})
	assert.Equal(t, "中文消息", trans.Trans(context.WithValue(context.Background(), "lang", "zh-CN"), "test.message"))
	assert.Equal(t, "English message", trans.Trans(context.WithValue(context.Background(), "lang", "en-US"), "test.message"))
}
