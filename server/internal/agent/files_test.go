package agent

import (
	"path/filepath"
	"testing"
)

func TestWorkspaceFileContentType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		data     []byte
		expected string
	}{
		{
			name:     "svg uses image MIME type",
			path:     "images/cat-icon.svg",
			data:     []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
			expected: "image/svg+xml",
		},
		{
			name:     "uppercase SVG extension",
			path:     "CAT-ICON.SVG",
			data:     []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`),
			expected: "image/svg+xml",
		},
		{
			name:     "gif uses registered image MIME type",
			path:     "animation.gif",
			data:     []byte("not needed for registered extensions"),
			expected: "image/gif",
		},
		{
			name:     "webp uses registered image MIME type",
			path:     "photo.webp",
			data:     []byte("not needed for registered extensions"),
			expected: "image/webp",
		},
		{
			name:     "avif uses registered image MIME type",
			path:     "photo.avif",
			data:     []byte("not needed for registered extensions"),
			expected: "image/avif",
		},
		{
			name:     "registered text extension uses MIME database",
			path:     "notes.txt",
			data:     []byte{0},
			expected: "text/plain; charset=utf-8",
		},
		{
			name:     "unknown extension falls back to content sniffing",
			path:     "notes.agentrazor-unknown",
			data:     []byte("hello"),
			expected: "text/plain; charset=utf-8",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if actual := workspaceFileContentType(test.path, test.data); actual != test.expected {
				t.Fatalf("workspaceFileContentType() = %q, want %q", actual, test.expected)
			}
		})
	}
}

func TestGeneratedImagePathStaysInConversationRoot(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "codex", "generated_images", "conversation-a")
	target, err := safePathWithin(root, "batch/image.png")
	if err != nil {
		t.Fatalf("safePathWithin() returned error for valid image: %v", err)
	}
	if expected := filepath.Join(root, "batch", "image.png"); target != expected {
		t.Fatalf("safePathWithin() = %q, want %q", target, expected)
	}

	for _, candidate := range []string{
		"../conversation-b/image.png",
		filepath.Join(string(filepath.Separator), "codex", "generated_images", "conversation-b", "image.png"),
	} {
		if _, err := safePathWithin(root, candidate); err == nil {
			t.Fatalf("safePathWithin(%q) succeeded outside conversation root", candidate)
		}
	}
}

func TestGeneratedImageTargetRejectsCrossConversationAndNestedPaths(t *testing.T) {
	service := &Service{codexHome: filepath.Join(string(filepath.Separator), "codex")}
	root, target, err := service.generatedImageTarget("conversation-a", "image.png")
	if err != nil {
		t.Fatalf("generatedImageTarget() returned error for valid image: %v", err)
	}
	expectedRoot := filepath.Join(service.codexHome, "generated_images", "conversation-a")
	if root != expectedRoot || target != filepath.Join(expectedRoot, "image.png") {
		t.Fatalf("generatedImageTarget() = (%q, %q), want (%q, %q)", root, target, expectedRoot, filepath.Join(expectedRoot, "image.png"))
	}

	validAbsolute := filepath.Join(expectedRoot, "absolute.png")
	if _, actual, err := service.generatedImageTarget("conversation-a", validAbsolute); err != nil || actual != validAbsolute {
		t.Fatalf("generatedImageTarget() rejected valid absolute savedPath: target=%q err=%v", actual, err)
	}

	for _, test := range []struct {
		conversationID string
		path           string
	}{
		{conversationID: "../conversation-b", path: "image.png"},
		{conversationID: "conversation-a/child", path: "image.png"},
		{conversationID: "conversation-a", path: "batch/image.png"},
		{conversationID: "conversation-a", path: "../conversation-b/image.png"},
		{conversationID: "conversation-a", path: filepath.Join(service.codexHome, "generated_images", "conversation-b", "image.png")},
	} {
		if _, _, err := service.generatedImageTarget(test.conversationID, test.path); err == nil {
			t.Fatalf("generatedImageTarget(%q, %q) unexpectedly succeeded", test.conversationID, test.path)
		}
	}
}

func TestGeneratedImageAssetNameMustBeDirectChild(t *testing.T) {
	for _, test := range []struct {
		name  string
		valid bool
	}{
		{name: "image.png", valid: true},
		{name: "image-01.webp", valid: true},
		{name: "", valid: false},
		{name: ".", valid: false},
		{name: "..", valid: false},
		{name: "nested/image.png", valid: false},
		{name: "nested\\image.png", valid: false},
		{name: "../other.png", valid: false},
	} {
		if actual := isGeneratedImageAssetName(test.name); actual != test.valid {
			t.Fatalf("isGeneratedImageAssetName(%q) = %t, want %t", test.name, actual, test.valid)
		}
	}
}
