package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/golang-jwt/jwt/v4"
	"github.com/zeromicro/go-zero/core/stores/redis"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

const testCasbinModel = `[request_definition]
r = sub, obj

[policy_definition]
p = sub, obj

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = r.sub == p.sub && r.obj == p.obj`

func TestAuthxAuthenticatesJWTAndRedisBeforeCasbin(t *testing.T) {
	redisServer := miniredis.RunT(t)
	sessions := auth.NewSessionStore(redis.New(redisServer.Addr()))
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("role-1", "route-1"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedToken(t, secret, userUUID, []string{"role-1"})
	if err := sessions.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save session: %v", err)
	}

	nextCalled := false
	authx := NewAuthxMiddleware(enforcer, func(*http.Request) string { return "route-1" }, secret, sessions)
	protected := authx.Handle(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		info, err := auth.Info(r.Context())
		if err != nil || info.Uuid != userUUID || info.Username != "alice" {
			t.Fatalf("claims were not injected into context: %#v, %v", info, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	protected.ServeHTTP(res, req)
	if res.Code != http.StatusNoContent || !nextCalled {
		t.Fatalf("registered session status=%d next=%v, want 204/true", res.Code, nextCalled)
	}

	if err := sessions.Delete(context.Background(), userUUID, token); err != nil {
		t.Fatalf("delete session: %v", err)
	}
	nextCalled = false
	res = httptest.NewRecorder()
	protected.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized || nextCalled {
		t.Fatalf("revoked session status=%d next=%v, want 401/false", res.Code, nextCalled)
	}
}

func TestAuthxReturnsForbiddenWhenSessionExistsWithoutPolicy(t *testing.T) {
	redisServer := miniredis.RunT(t)
	sessions := auth.NewSessionStore(redis.New(redisServer.Addr()))
	enforcer := newTestEnforcer(t)
	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedToken(t, secret, userUUID, []string{"role-1"})
	if err := sessions.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save session: %v", err)
	}

	authx := NewAuthxMiddleware(enforcer, func(*http.Request) string { return "route-1" }, secret, sessions)
	protected := authx.Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("unauthorized role must not reach next handler")
	})
	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()

	protected.ServeHTTP(res, req)
	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestAuthxRejectsInvalidJWTBeforeSessionAndCasbin(t *testing.T) {
	redisServer := miniredis.RunT(t)
	sessions := auth.NewSessionStore(redis.New(redisServer.Addr()))
	authx := NewAuthxMiddleware(newTestEnforcer(t), func(*http.Request) string {
		t.Fatal("invalid JWT must not reach route authorization")
		return ""
	}, "test-secret", sessions)
	protected := authx.Handle(func(http.ResponseWriter, *http.Request) {
		t.Fatal("invalid JWT must not reach next handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	req.Header.Set("Authorization", "Bearer invalid-token")
	res := httptest.NewRecorder()
	protected.ServeHTTP(res, req)

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

func newTestEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	model, err := casbinmodel.NewModelFromString(testCasbinModel)
	if err != nil {
		t.Fatalf("create model: %v", err)
	}
	enforcer, err := casbin.NewEnforcer(model)
	if err != nil {
		t.Fatalf("create enforcer: %v", err)
	}
	return enforcer
}

func signedToken(t *testing.T, secret, userUUID string, roles []string) string {
	t.Helper()
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uuid":       userUUID,
		"username":   "alice",
		"role_uuids": roles,
		"exp":        time.Now().Add(time.Minute).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return value
}
