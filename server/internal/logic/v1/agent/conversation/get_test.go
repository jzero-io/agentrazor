package conversation

import (
	"context"
	"testing"
)

func TestCurrentUserUUID(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		want    string
		wantErr bool
	}{
		{
			name: "authenticated user",
			ctx:  context.WithValue(context.Background(), "uuid", " user-1 "),
			want: "user-1",
		},
		{
			name:    "missing user",
			ctx:     context.Background(),
			wantErr: true,
		},
		{
			name:    "blank user",
			ctx:     context.WithValue(context.Background(), "uuid", "  "),
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := currentUserUUID(test.ctx)
			if test.wantErr {
				if err == nil {
					t.Fatal("currentUserUUID() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("currentUserUUID() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("currentUserUUID() = %q, want %q", got, test.want)
			}
		})
	}
}
