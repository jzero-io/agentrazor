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
	enforcer := newTestEnforcer(t)
	if _, err := enforcer.AddPolicy("role-1", "route-1"); err != nil {
		t.Fatalf("add policy: %v", err)
	}

	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedToken(t, secret, userUUID, []string{"role-1"})
	authx := NewAuthxMiddleware(enforcer, func(*http.Request) string { return "route-1" }, secret, redis.New(redisServer.Addr()))
	if err := authx.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save JWT auth entry: %v", err)
	}

	nextCalled := false
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
		t.Fatalf("registered JWT status=%d next=%v, want 204/true", res.Code, nextCalled)
	}

	if err := authx.Delete(context.Background(), userUUID, token); err != nil {
		t.Fatalf("delete JWT auth entry: %v", err)
	}
	nextCalled = false
	res = httptest.NewRecorder()
	protected.ServeHTTP(res, req)
	if res.Code != http.StatusUnauthorized || nextCalled {
		t.Fatalf("revoked JWT status=%d next=%v, want 401/false", res.Code, nextCalled)
	}
}

func TestAuthxReturnsForbiddenWhenJWTExistsWithoutPolicy(t *testing.T) {
	redisServer := miniredis.RunT(t)
	enforcer := newTestEnforcer(t)
	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedToken(t, secret, userUUID, []string{"role-1"})
	authx := NewAuthxMiddleware(enforcer, func(*http.Request) string { return "route-1" }, secret, redis.New(redisServer.Addr()))
	if err := authx.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save JWT auth entry: %v", err)
	}

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

func TestAuthxAuthenticateUsesGoZeroClaimsWithoutCasbin(t *testing.T) {
	redisServer := miniredis.RunT(t)
	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedToken(t, secret, userUUID, []string{"role-1"})
	authx := NewAuthxMiddleware(newTestEnforcer(t), func(*http.Request) string {
		t.Fatal("authentication-only flow must not resolve a Casbin route")
		return ""
	}, secret, redis.New(redisServer.Addr()))
	if err := authx.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save JWT auth entry: %v", err)
	}

	authenticated := authx.Authenticate(func(w http.ResponseWriter, r *http.Request) {
		info, err := auth.Info(r.Context())
		if err != nil || info.Uuid != userUUID || info.Username != "alice" {
			t.Fatalf("go-zero claims were not injected into context: %#v, %v", info, err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/agent", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	authenticated(res, req)

	if res.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusNoContent)
	}
}

func TestAuthxAuthenticateRejectsRefreshTokenEvenWhenRegistered(t *testing.T) {
	redisServer := miniredis.RunT(t)
	const (
		secret   = "test-secret"
		userUUID = "user-1"
	)
	token := signedTokenWithType(t, secret, userUUID, []string{"role-1"}, "refresh")
	authx := NewAuthxMiddleware(newTestEnforcer(t), func(*http.Request) string { return "route-1" }, secret, redis.New(redisServer.Addr()))
	if err := authx.Save(context.Background(), userUUID, token, 30); err != nil {
		t.Fatalf("save JWT auth entry: %v", err)
	}

	authenticated := authx.Authenticate(func(http.ResponseWriter, *http.Request) {
		t.Fatal("refresh token must not reach next handler")
	})
	req := httptest.NewRequest(http.MethodGet, "/agent", http.NoBody)
	req.Header.Set("Authorization", "Bearer "+token)
	res := httptest.NewRecorder()
	authenticated(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestAuthxRejectsInvalidJWTBeforeRedisAndCasbin(t *testing.T) {
	redisServer := miniredis.RunT(t)
	authx := NewAuthxMiddleware(newTestEnforcer(t), func(*http.Request) string {
		t.Fatal("invalid JWT must not reach route authorization")
		return ""
	}, "test-secret", redis.New(redisServer.Addr()))
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

func TestAuthxJWTAuthLifecycle(t *testing.T) {
	redisServer := miniredis.RunT(t)
	authx := NewAuthxMiddleware(nil, nil, "test-secret", redis.New(redisServer.Addr()))
	ctx := context.Background()

	const (
		userUUID = "user-1"
		token    = "header.payload.signature"
		ttl      = 30
	)
	if err := authx.Save(ctx, userUUID, token, ttl); err != nil {
		t.Fatalf("save JWT auth entry: %v", err)
	}

	key := jwtAuthKey(userUUID, token)
	stored, err := redisServer.Get(key)
	if err != nil {
		t.Fatalf("read JWT auth entry: %v", err)
	}
	if stored != token {
		t.Fatalf("stored token = %q, want %q", stored, token)
	}
	if got := redisServer.TTL(key); got != ttl*time.Second {
		t.Fatalf("ttl = %s, want %s", got, ttl*time.Second)
	}

	exists, err := authx.exists(ctx, userUUID, token)
	if err != nil || !exists {
		t.Fatalf("saved token exists = %v, err = %v, want true/nil", exists, err)
	}
	if err := redisServer.Set(key, "tampered"); err != nil {
		t.Fatalf("tamper JWT auth value: %v", err)
	}
	exists, err = authx.exists(ctx, userUUID, token)
	if err != nil || exists {
		t.Fatalf("tampered token exists = %v, err = %v, want false/nil", exists, err)
	}

	if err := authx.Save(ctx, userUUID, token, ttl); err != nil {
		t.Fatalf("restore JWT auth entry: %v", err)
	}
	if err := authx.Delete(ctx, userUUID, token); err != nil {
		t.Fatalf("delete JWT auth entry: %v", err)
	}
	exists, err = authx.exists(ctx, userUUID, token)
	if err != nil || exists {
		t.Fatalf("deleted token exists = %v, err = %v, want false/nil", exists, err)
	}
}

func TestAuthxJWTAuthRejectsInvalidInputAndUnavailableRedis(t *testing.T) {
	ctx := context.Background()
	authx := NewAuthxMiddleware(nil, nil, "test-secret", nil)

	if err := authx.Save(ctx, "user-1", "token", 30); err == nil {
		t.Fatal("save should fail without Redis")
	}
	if _, err := authx.exists(ctx, "user-1", "token"); err == nil {
		t.Fatal("exists should fail without Redis")
	}
	if err := authx.Delete(ctx, "user-1", "token"); err == nil {
		t.Fatal("delete should fail without Redis")
	}

	redisServer := miniredis.RunT(t)
	authx = NewAuthxMiddleware(nil, nil, "test-secret", redis.New(redisServer.Addr()))
	if err := authx.Save(ctx, "", "token", 30); err == nil {
		t.Fatal("save should reject an empty user UUID")
	}
	if err := authx.Save(ctx, "user-1", "", 30); err == nil {
		t.Fatal("save should reject an empty token")
	}
	if err := authx.Save(ctx, "user-1", "token", 0); err == nil {
		t.Fatal("save should reject a non-positive TTL")
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
	return signedTokenWithType(t, secret, userUUID, roles, accessTokenType)
}

func signedTokenWithType(t *testing.T, secret, userUUID string, roles []string, tokenType string) string {
	t.Helper()
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uuid":       userUUID,
		"username":   "alice",
		"role_uuids": roles,
		"token_type": tokenType,
		"exp":        time.Now().Add(time.Minute).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return value
}
