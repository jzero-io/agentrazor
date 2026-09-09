package svc

import (
	"context"
	"embed"
	"testing"

	corei18n "github.com/jzero-io/agentrazor/core-engine/i18n"
)

func TestAgentLocaleTranslations(t *testing.T) {
	trans := corei18n.NewTranslator(corei18n.I18nConf{Dir: "../../etc/locale"}, embed.FS{})
	tests := []struct {
		lang string
		want string
	}{
		{lang: "zh-CN", want: "Skill 压缩包中必须包含 SKILL.md 文件"},
		{lang: "en-US", want: "The Skill archive must contain a SKILL.md file"},
	}
	for _, tt := range tests {
		ctx := context.WithValue(context.Background(), "lang", tt.lang)
		if got := trans.Trans(ctx, "manage.agent.skill.manifestMissing"); got != tt.want {
			t.Fatalf("Trans(%q) = %q, want %q", tt.lang, got, tt.want)
		}
	}
}
