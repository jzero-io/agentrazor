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
