package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
)

type SessionChecker interface {
	Exists(ctx context.Context, userUUID, token string) (bool, error)
}

type JwtSessionMiddleware struct {
	sessions SessionChecker
}

func NewJwtSessionMiddleware(sessions SessionChecker) *JwtSessionMiddleware {
	return &JwtSessionMiddleware{sessions: sessions}
}

func (m *JwtSessionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authInfo, err := auth.Info(r.Context())
		if err != nil || strings.TrimSpace(authInfo.Uuid) == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok || m.sessions == nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		exists, err := m.sessions.Exists(r.Context(), authInfo.Uuid, token)
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
	}
}

func bearerToken(value string) (string, bool) {
	parts := strings.Fields(value)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
		return "", false
	}
	return parts[1], true
}
