package agent

import "testing"

func TestDecodeStoredThreadArchiveState(t *testing.T) {
	tests := []struct {
		name            string
		raw             map[string]any
		defaultArchived bool
		want            bool
	}{
		{
			name: "uses archived list state",
			raw: map[string]any{
				"id":   "archived-thread",
				"path": "/dist/data/codex-home/archived_sessions/rollout.jsonl",
			},
			defaultArchived: true,
			want:            true,
		},
		{
			name: "does not infer state from rollout path",
			raw: map[string]any{
				"id":   "archived-path-thread",
				"path": "/dist/data/codex-home/archived_sessions/rollout.jsonl",
			},
			want: false,
		},
		{
			name: "explicit state takes precedence",
			raw: map[string]any{
				"id":       "explicit-active-thread",
				"archived": false,
				"path":     "/dist/data/codex-home/archived_sessions/rollout.jsonl",
			},
			defaultArchived: true,
			want:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			thread := decodeStoredThread(tt.raw, tt.defaultArchived)
			if thread.Archived != tt.want {
				t.Fatalf("Archived = %v, want %v", thread.Archived, tt.want)
			}
		})
	}
}
