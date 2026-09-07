package agent

import (
	"errors"
	"testing"
)

func TestThreadNotMaterializedError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "draft thread",
			err: &RPCError{
				Method:  "thread/read",
				Code:    -32600,
				Message: "thread draft-id is not materialized yet; includeTurns is unavailable before first user message",
			},
			want: true,
		},
		{
			name: "missing thread",
			err: &RPCError{
				Method:  "thread/read",
				Code:    -32600,
				Message: "thread missing-id not found",
			},
		},
		{
			name: "wrapped draft error",
			err: errors.Join(errors.New("read failed"), &RPCError{
				Method:  "thread/read",
				Code:    -32600,
				Message: "thread draft-id is not materialized yet; includeTurns is unavailable before first user message",
			}),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := threadNotMaterializedError(tt.err); got != tt.want {
				t.Fatalf("threadNotMaterializedError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeStoredThreadArchiveState(t *testing.T) {
	tests := []struct {
		name            string
		raw             map[string]any
		defaultArchived bool
		want            bool
	}{
		{
			name: "archived rollout path",
			raw: map[string]any{
				"id":   "archived-thread",
				"path": "/dist/data/codex-home/archived_sessions/rollout.jsonl",
			},
			want: true,
		},
		{
			name: "active rollout path",
			raw: map[string]any{
				"id":   "active-thread",
				"path": "/dist/data/codex-home/sessions/2026/09/07/rollout.jsonl",
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
