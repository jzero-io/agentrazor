package role

import (
	"reflect"
	"testing"
)

func TestNormalizeMenuDependencies(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "chatgpt login requires save settings", in: []string{agentChatGPTLoginMenuUUID}, want: []string{agentChatGPTLoginMenuUUID, agentSaveSettingsMenuUUID}},
		{name: "api key login requires save settings", in: []string{agentAPIKeyLoginMenuUUID}, want: []string{agentAPIKeyLoginMenuUUID, agentSaveSettingsMenuUUID}},
		{name: "existing save settings is not duplicated", in: []string{agentSaveSettingsMenuUUID, agentChatGPTLoginMenuUUID}, want: []string{agentSaveSettingsMenuUUID, agentChatGPTLoginMenuUUID}},
		{name: "unrelated menus are unchanged and duplicates are removed", in: []string{"menu-a", "menu-a", "menu-b"}, want: []string{"menu-a", "menu-b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeMenuDependencies(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeMenuDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}
