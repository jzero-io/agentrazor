package agent

import "testing"

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
