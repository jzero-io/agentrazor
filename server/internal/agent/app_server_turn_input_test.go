package agent

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAppServerTurnInputKeepsSkillSeparateFromPrompt(t *testing.T) {
	server := &appServer{codexHome: "/codex-home"}
	prompt := "answer the ticket"

	got := server.turnInput(prompt)
	want := []map[string]any{
		{
			"type": "text",
			"text": prompt,
		},
		{
			"type": "skill",
			"name": requestGuardSkillName,
			"path": filepath.Join("/codex-home", "skills", ".system", requestGuardSkillName, "SKILL.md"),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}
