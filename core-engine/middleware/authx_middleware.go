package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	casbin "github.com/casbin/casbin/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/token"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

var (
	errInvalidAuthToken = errors.New("invalid auth token")
	errInvalidClaims    = errors.New("invalid auth claims")
	errInvalidSession   = errors.New("invalid auth session")
)

type AuthxMiddleware struct {
	CasbinEnforcer *casbin.Enforcer
	Route2CodeFunc func(r *http.Request) string
	AccessSecret   string
	AuthSessions   *auth.SessionStore
	tokenParser    *token.TokenParser
}

func NewAuthxMiddleware(casbinEnforcer *casbin.Enforcer, route2codeFunc func(r *http.Request) string, accessSecret string, authSessions *auth.SessionStore) *AuthxMiddleware {
	return &AuthxMiddleware{
		CasbinEnforcer: casbinEnforcer,
		Route2CodeFunc: route2codeFunc,
		AccessSecret:   accessSecret,
		AuthSessions:   authSessions,
		tokenParser:    token.NewTokenParser(),
	}
}

// Authenticate verifies the JWT and its Redis session, then injects the JWT
// claims into the request context. It is also used by composite authentication
// middleware that needs to support non-JWT credentials.
func (m *AuthxMiddleware) Authenticate(r *http.Request) (*http.Request, error) {
	if m == nil || m.tokenParser == nil || m.AuthSessions == nil || strings.TrimSpace(m.AccessSecret) == "" {
		return nil, errInvalidAuthToken
	}

	rawToken, ok := bearerToken(r.Header.Get("Authorization"))
	if !ok {
		return nil, errInvalidAuthToken
	}

	parsedToken, err := m.tokenParser.ParseToken(r, m.AccessSecret, "")
	if err != nil || parsedToken == nil || !parsedToken.Valid {
		return nil, errInvalidAuthToken
	}
	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errInvalidClaims
	}

	userUUID, ok := claims["uuid"].(string)
	if !ok || strings.TrimSpace(userUUID) == "" {
		return nil, errInvalidClaims
	}
	exists, err := m.AuthSessions.Exists(r.Context(), userUUID, rawToken)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errInvalidSession
	}

	ctx := r.Context()
	for key, value := range claims {
		if isStandardClaim(key) {
			continue
		}
		ctx = context.WithValue(ctx, key, value)
	}

	return r.WithContext(ctx), nil
}

func (m *AuthxMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authenticatedRequest, err := m.Authenticate(r)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("authenticate request: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		authInfo, err := auth.Info(authenticatedRequest.Context())
		if err != nil || authInfo.Uuid == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		subs := cast.ToStringSlice(authInfo.RoleUuids)
		obj := m.Route2CodeFunc(authenticatedRequest)

		// verify casbin rule
		if result := batchCheck(m.CasbinEnforcer, subs, obj); !result {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		next(w, authenticatedRequest)
	}
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}

func isStandardClaim(claim string) bool {
	switch claim {
	case "aud", "exp", "jti", "iat", "iss", "nbf", "sub":
		return true
	default:
		return false
	}
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
