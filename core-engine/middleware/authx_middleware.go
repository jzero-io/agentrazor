package middleware

import (
	"net/http"
	"strings"

	casbin "github.com/casbin/casbin/v2"
	"github.com/spf13/cast"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/handler"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

const accessTokenType = "access"

type AuthxMiddleware struct {
	CasbinEnforcer *casbin.Enforcer
	Route2CodeFunc func(r *http.Request) string
	AccessSecret   string
	AuthSessions   *auth.SessionStore
}

func NewAuthxMiddleware(casbinEnforcer *casbin.Enforcer, route2codeFunc func(r *http.Request) string, accessSecret string, authSessions *auth.SessionStore) *AuthxMiddleware {
	return &AuthxMiddleware{
		CasbinEnforcer: casbinEnforcer,
		Route2CodeFunc: route2codeFunc,
		AccessSecret:   accessSecret,
		AuthSessions:   authSessions,
	}
}

// Authenticate delegates JWT parsing, validation, and context injection to
// go-zero, then verifies that the exact access token is registered in Redis.
// Composite authentication middleware can reuse this without running Casbin.
func (m *AuthxMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	if m == nil || m.AuthSessions == nil || strings.TrimSpace(m.AccessSecret) == "" {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}
	}

	verifySession := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		exists, err := m.AuthSessions.Exists(r.Context(), authInfo.Uuid, rawToken)
		if err != nil {
			logx.WithContext(r.Context()).Errorf("validate JWT session: %v", err)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !exists {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		next(w, r)
	})

	return handler.Authorize(m.AccessSecret)(verifySession).ServeHTTP
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
