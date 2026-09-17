package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

func TestJwtSessionMiddleware(t *testing.T) {
	redisServer := miniredis.RunT(t)
	sessions := auth.NewSessionStore(redis.New(redisServer.Addr()))
	const (
		userUUID = "user-1"
		token    = "header.payload.signature"
	)
	if err := sessions.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save session: %v", err)
	}

	nextCalled := false
	handler := NewJwtSessionMiddleware(sessions).Handle(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req = req.WithContext(context.WithValue(req.Context(), "uuid", userUUID))
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
	if !nextCalled {
		t.Fatal("valid session should call next handler")
	}
}

func TestJwtSessionMiddlewareRejectsMissingSession(t *testing.T) {
	redisServer := miniredis.RunT(t)
	sessions := auth.NewSessionStore(redis.New(redisServer.Addr()))

	handler := NewJwtSessionMiddleware(sessions).Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("missing session must not call next handler")
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req = req.WithContext(context.WithValue(req.Context(), "uuid", "user-1"))
	req.Header.Set("Authorization", "Bearer unknown-token")
	res := httptest.NewRecorder()

	handler(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
		ok    bool
	}{
		{name: "bearer", value: "Bearer token", want: "token", ok: true},
		{name: "case insensitive", value: "bearer token", want: "token", ok: true},
		{name: "missing scheme", value: "token"},
		{name: "wrong scheme", value: "Basic token"},
		{name: "empty", value: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := bearerToken(test.value)
			if got != test.want || ok != test.ok {
				t.Fatalf("bearerToken(%q) = (%q, %v), want (%q, %v)", test.value, got, ok, test.want, test.ok)
			}
		})
	}
}
