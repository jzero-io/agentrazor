package middleware

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	casbin "github.com/casbin/casbin/v2"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/rest/handler"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

const (
	accessTokenType  = "access"
	jwtAuthKeyPrefix = "JWT_AUTH"
)

var errInvalidJWTAuth = errors.New("invalid JWT auth entry")

type AuthxMiddleware struct {
	CasbinEnforcer *casbin.Enforcer
	Route2CodeFunc func(r *http.Request) string
	AccessSecret   string
	redis          *redis.Redis
}

func NewAuthxMiddleware(casbinEnforcer *casbin.Enforcer, route2codeFunc func(r *http.Request) string, accessSecret string, redisStore *redis.Redis) *AuthxMiddleware {
	return &AuthxMiddleware{
		CasbinEnforcer: casbinEnforcer,
		Route2CodeFunc: route2codeFunc,
		AccessSecret:   accessSecret,
		redis:          redisStore,
	}
}

// Save registers an access token in Redis for the same lifetime as the JWT.
func (m *AuthxMiddleware) Save(ctx context.Context, userUUID, token string, ttlSeconds int) error {
	if err := validateJWTAuth(userUUID, token, ttlSeconds); err != nil {
		return err
	}
	if m == nil || m.redis == nil {
		return errors.New("Authx Redis is unavailable")
	}

	return m.redis.SetexCtx(ctx, jwtAuthKey(userUUID, token), token, ttlSeconds)
}

// Delete revokes one exact access token.
func (m *AuthxMiddleware) Delete(ctx context.Context, userUUID, token string) error {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" {
		return errInvalidJWTAuth
	}
	if m == nil || m.redis == nil {
		return errors.New("Authx Redis is unavailable")
	}

	_, err := m.redis.DelCtx(ctx, jwtAuthKey(userUUID, token))
	return err
}

// Authenticate delegates JWT parsing, validation, and context injection to
// go-zero, then verifies that the exact access token is registered in Redis.
// Composite authentication middleware can reuse this without running Casbin.
func (m *AuthxMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	if m == nil || m.redis == nil || strings.TrimSpace(m.AccessSecret) == "" {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}

	verifyJWTAuth := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authInfo, err := auth.Info(r.Context())
		if err != nil || strings.TrimSpace(authInfo.Uuid) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if tokenType, ok := r.Context().Value("token_type").(string); !ok || tokenType != accessTokenType {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		rawToken, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		exists, err := m.exists(r.Context(), authInfo.Uuid, rawToken)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("validate JWT auth entry: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next(w, r)
	})

	return handler.Authorize(m.AccessSecret)(verifyJWTAuth).ServeHTTP
}

func (m *AuthxMiddleware) exists(ctx context.Context, userUUID, token string) (bool, error) {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" {
		return false, errInvalidJWTAuth
	}
	if m == nil || m.redis == nil {
		return false, errors.New("Authx Redis is unavailable")
	}

	storedToken, err := m.redis.GetCtx(ctx, jwtAuthKey(userUUID, token))
	if err != nil {
		return false, err
	}
	if storedToken == "" {
		return false, nil
	}

	return subtle.ConstantTimeCompare([]byte(storedToken), []byte(token)) == 1, nil
}

func jwtAuthKey(userUUID, token string) string {
	return fmt.Sprintf("%s:%s:%s", jwtAuthKeyPrefix, userUUID, token)
}

func validateJWTAuth(userUUID, token string, ttlSeconds int) error {
	if strings.TrimSpace(userUUID) == "" || strings.TrimSpace(token) == "" || ttlSeconds <= 0 {
		return errInvalidJWTAuth
	}
	return nil
}

func (m *AuthxMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	authorize := func(w http.ResponseWriter, r *http.Request) {
		authInfo, err := auth.Info(r.Context())
		if err != nil || authInfo.Uuid == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		subs := cast.ToStringSlice(authInfo.RoleUuids)
		obj := m.Route2CodeFunc(r)

		// verify casbin rule
		if result := batchCheck(m.CasbinEnforcer, subs, obj); !result {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next(w, r)
	}

	return m.Authenticate(authorize)
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func batchCheck(cbn *casbin.Enforcer, subs []string, obj string) bool {
	var checkReq [][]any
	for _, v := range subs {
		checkReq = append(checkReq, []any{v, obj})
	}

	result, err := cbn.BatchEnforce(checkReq)
	if err != nil {
		return false
	}

	for _, v := range result {
		if v {
			return true
		}
	}

	return false
}
