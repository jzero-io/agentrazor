// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/rest/handler"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/agentrazor/server/internal/model"
	agentapikeymodel "github.com/jzero-io/agentrazor/server/internal/model/agent_api_key"
	"github.com/jzero-io/agentrazor/server/internal/svc"
)

var ErrInvalidAPIKey = errors.New("invalid api key")

const APIKeyHeader = "X-API-Key"

func APIKeyFromRequest(r *http.Request) (string, bool) {
	key := strings.TrimSpace(r.Header.Get(APIKeyHeader))
	if !strings.HasPrefix(key, "ar-") {
		return "", false
	}
	return key, true
}

func ContextForAPIKey(ctx context.Context, key string, models model.Model) (context.Context, error) {
	digest := sha256.Sum256([]byte(key))
	row, err := models.AgentApiKey.FindOneByKeyHash(ctx, nil, hex.EncodeToString(digest[:]))
	if err != nil {
		if errors.Is(err, agentapikeymodel.ErrNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}
	user, err := models.ManageUser.FindOneByUuid(ctx, nil, row.UserUuid)
	if err != nil || user.Status != "1" {
		return nil, ErrInvalidAPIKey
	}
	ctx = context.WithValue(ctx, "uuid", user.Uuid)
	ctx = context.WithValue(ctx, "username", user.Username)
	ctx = context.WithValue(ctx, "role_uuids", []any{})
	return ctx, nil
}

type AgentMiddleware struct {
	svcCtx *svc.ServiceContext
}

func NewAgentMiddleware(svcCtx *svc.ServiceContext) *AgentMiddleware {
	return &AgentMiddleware{svcCtx: svcCtx}
}

func agentIdentityUUID(ctx context.Context) (string, bool) {
	info, err := auth.Info(ctx)
	userUUID := strings.TrimSpace(info.Uuid)
	return userUUID, err == nil && userUUID != ""
}

func (m *AgentMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	jwtNext := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userUUID, ok := agentIdentityUUID(r.Context())
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		user, err := m.svcCtx.Model.ManageUser.FindOneByUuid(r.Context(), nil, userUUID)
		if err != nil || user.Status != "1" {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		next(w, r)
	})
	jwtHandler := handler.Authorize(
		m.svcCtx.MustGetConfig().Jwt.AccessSecret,
		handler.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
			httpx.ErrorCtx(r.Context(), w, err)
		}),
	)(jwtNext)

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
