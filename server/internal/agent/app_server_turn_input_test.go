package agent

import (
	"reflect"
	"testing"
)

func TestAppServerTurnInputMapsText(t *testing.T) {
	server := &appServer{}
	got := server.turnInput(TurnInput{Text: "answer the ticket"})
	want := []map[string]any{{"type": "text", "text": "answer the ticket"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}

func TestAppServerTurnInputMapsImagesAndReferencesFiles(t *testing.T) {
	server := &appServer{}
	got := server.turnInput(TurnInput{
		Text: "review these",
		Attachments: []MessageAttachment{
			{Name: "notes.pdf", Path: "attachments/a/notes.pdf", Kind: "file", LocalPath: "/workspace/thread/attachments/a/notes.pdf"},
			{Name: "screen.png", Path: "attachments/b/screen.png", Kind: "image", LocalPath: "/workspace/thread/attachments/b/screen.png"},
		},
	})
	want := []map[string]any{
		{
			"type": "text",
			"text": "review these\n\n# Files mentioned by the user:\n- `attachments/a/notes.pdf` (notes.pdf)",
		},
		{
			"type": "localImage",
			"path": "/workspace/thread/attachments/b/screen.png",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}

func TestAppServerTurnInputAllowsImageOnly(t *testing.T) {
	server := &appServer{}
	got := server.turnInput(TurnInput{Attachments: []MessageAttachment{{
		Name: "screen.png", Kind: "image", LocalPath: "/workspace/thread/screen.png",
	}}})
	want := []map[string]any{{"type": "localImage", "path": "/workspace/thread/screen.png"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("turnInput() = %#v, want %#v", got, want)
	}
}
