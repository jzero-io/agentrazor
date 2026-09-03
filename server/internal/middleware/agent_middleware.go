// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/handler"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
)

type AgentMiddleware struct {
	svcCtx *svc.ServiceContext
}

func NewAgentMiddleware(svcCtx *svc.ServiceContext) *AgentMiddleware {
	return &AgentMiddleware{svcCtx: svcCtx}
}

func (m *AgentMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	jwtHandler := handler.Authorize(
		m.svcCtx.MustGetConfig().Jwt.AccessSecret,
		handler.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.ErrorCtx(r.Context(), w, err)
		}),
	)(http.HandlerFunc(next))

	return func(w http.ResponseWriter, r *http.Request) {
		if key, ok := APIKeyFromRequest(r); ok {
			ctx, err := ContextForAPIKey(r.Context(), key, m.svcCtx.Model)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
				return
			}
			next(w, r.WithContext(ctx))
			return
		}
		jwtHandler.ServeHTTP(w, r)
	}
}
