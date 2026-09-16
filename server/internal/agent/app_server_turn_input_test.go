package agent

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAppServerTurnInputKeepsSkillSeparateFromPrompt(t *testing.T) {
	server := &appServer{codexHome: "/codex-home"}
	prompt := "answer the ticket"
	requestGuard := &Skill{
		Name:    requestGuardSkillName,
		Path:    filepath.Join("/codex-home", "skills", requestGuardSkillName, "SKILL.md"),
		Enabled: true,
	}

	got := server.turnInput(prompt, requestGuard)
	want := []map[string]any{
		{
			"type": "text",
			"text": prompt,
		},
		{
			"type": "skill",
			"name": requestGuardSkillName,
			"path": filepath.Join("/codex-home", "skills", requestGuardSkillName, "SKILL.md"),
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}

func TestAppServerTurnInputOmitsDisabledRequestGuard(t *testing.T) {
	server := &appServer{codexHome: "/codex-home"}
	requestGuard := &Skill{
		Name:    requestGuardSkillName,
		Path:    filepath.Join("/codex-home", "skills", requestGuardSkillName, "SKILL.md"),
		Enabled: false,
	}

	got := server.turnInput("answer the ticket", requestGuard)
	want := []map[string]any{{"type": "text", "text": "answer the ticket"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}
