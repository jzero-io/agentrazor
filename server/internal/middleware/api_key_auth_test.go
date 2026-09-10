package middleware

import (
	"context"
	"net/http/httptest"
	"testing"
)

func TestAPIKeyFromRequest(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set(APIKeyHeader, " ar-example ")

	key, ok := APIKeyFromRequest(r)
	if !ok || key != "ar-example" {
		t.Fatalf("APIKeyFromRequest() = %q, %v", key, ok)
	}
}

func TestAPIKeyFromRequestDoesNotUseAuthorization(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer ar-example")

	if key, ok := APIKeyFromRequest(r); ok {
		t.Fatalf("APIKeyFromRequest() unexpectedly accepted Authorization value %q", key)
	}
}

func TestAgentIdentityUUIDRequiresUUID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want bool
	}{
		{name: "missing uuid", ctx: context.Background(), want: false},
		{name: "blank uuid", ctx: context.WithValue(context.Background(), "uuid", "  "), want: false},
		{name: "valid uuid", ctx: context.WithValue(context.Background(), "uuid", "user-uuid"), want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := agentIdentityUUID(tt.ctx)
			if got != tt.want {
				t.Fatalf("agentIdentityUUID() valid = %v, want %v", got, tt.want)
			}
		})
	}
}
