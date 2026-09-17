package auth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/golang-jwt/jwt/v4"
	coreconfig "github.com/jzero-io/agentrazor/core-engine/config"
	coremiddleware "github.com/jzero-io/agentrazor/core-engine/middleware"
	coresvc "github.com/jzero-io/agentrazor/core-engine/svc"
	"github.com/zeromicro/go-zero/core/stores/redis"

	serverconfig "github.com/jzero-io/agentrazor/server/internal/config"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

type staticConfigCenter struct {
	config serverconfig.Config
}

func (s staticConfigCenter) GetConfig() (serverconfig.Config, error) {
	return s.config, nil
}

func (s staticConfigCenter) MustGetConfig() serverconfig.Config {
	return s.config
}

func (staticConfigCenter) AddListener(func()) {}

func TestIssueTokenPairStoresOnlyAccessToken(t *testing.T) {
	redisServer := miniredis.RunT(t)
	authx := coremiddleware.NewAuthxMiddleware(nil, nil, "test-secret", redis.New(redisServer.Addr()))
	config := serverconfig.Config{Config: coreconfig.Config{}}
	config.Jwt.AccessSecret = "test-secret"
	config.Jwt.AccessExpire = 30
	config.Jwt.RefreshExpire = 300
	svcCtx := &svc.ServiceContext{
		ConfigCenter: staticConfigCenter{config: config},
		ServiceContext: &coresvc.ServiceContext{
			AuthxMiddleware: authx,
		},
	}
	claims := map[string]any{
		"uuid":       "user-1",
		"username":   "alice",
		"role_uuids": []string{"role-1"},
	}

	accessToken, refreshToken, err := issueTokenPair(context.Background(), svcCtx, "user-1", claims)
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}
	if accessToken == refreshToken {
		t.Fatal("access and refresh tokens must differ")
	}

	accessClaims := parseClaims(t, accessToken, config.Jwt.AccessSecret)
	if accessClaims[claimTokenType] != tokenTypeAccess {
		t.Fatalf("access token type = %v, want %q", accessClaims[claimTokenType], tokenTypeAccess)
	}
	refreshClaims := parseClaims(t, refreshToken, config.Jwt.AccessSecret)
	if refreshClaims[claimTokenType] != tokenTypeRefresh {
		t.Fatalf("refresh token type = %v, want %q", refreshClaims[claimTokenType], tokenTypeRefresh)
	}
	if accessClaims["jti"] == "" || refreshClaims["jti"] == "" {
		t.Fatal("each token must have a unique identifier")
	}
	if accessClaims["jti"] == refreshClaims["jti"] {
		t.Fatal("access and refresh token identifiers must differ")
	}

	accessKey := "JWT_AUTH:user-1:" + accessToken
	stored, err := redisServer.Get(accessKey)
	if err != nil || stored != accessToken {
		t.Fatalf("stored access token = %q, err = %v", stored, err)
	}
	if redisServer.Exists("JWT_AUTH:user-1:" + refreshToken) {
		t.Fatal("refresh token must not be stored in Authx Redis")
	}
	if got := redisServer.TTL(accessKey); got != 30*time.Second {
		t.Fatalf("access JWT ttl = %s, want %s", got, 30*time.Second)
	}
}

func TestIssueTokenPairIsUniqueWithinTheSameSecond(t *testing.T) {
	redisServer := miniredis.RunT(t)
	authx := coremiddleware.NewAuthxMiddleware(nil, nil, "test-secret", redis.New(redisServer.Addr()))
	config := serverconfig.Config{Config: coreconfig.Config{}}
	config.Jwt.AccessSecret = "test-secret"
	config.Jwt.AccessExpire = 30
	config.Jwt.RefreshExpire = 300
	svcCtx := &svc.ServiceContext{
		ConfigCenter: staticConfigCenter{config: config},
		ServiceContext: &coresvc.ServiceContext{
			AuthxMiddleware: authx,
		},
	}
	claims := map[string]any{"uuid": "user-1"}

	firstAccess, firstRefresh, err := issueTokenPair(context.Background(), svcCtx, "user-1", claims)
	if err != nil {
		t.Fatalf("issue first token pair: %v", err)
	}
	secondAccess, secondRefresh, err := issueTokenPair(context.Background(), svcCtx, "user-1", claims)
	if err != nil {
		t.Fatalf("issue second token pair: %v", err)
	}
	if firstAccess == secondAccess || firstRefresh == secondRefresh {
		t.Fatal("separate token pairs issued in one second must remain distinct")
	}
}

func parseClaims(t *testing.T, value, secret string) jwt.MapClaims {
	t.Helper()
	parsed, err := jwt.Parse(value, func(*jwt.Token) (any, error) {
		return []byte(secret), nil
	})
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("unexpected claims type")
	}
	return claims
}
