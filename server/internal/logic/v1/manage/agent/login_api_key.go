package agent

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type LoginApiKey struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewLoginApiKey(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *LoginApiKey {
	return &LoginApiKey{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *LoginApiKey) LoginApiKey(req *types.LoginApiKeyRequest) (resp *types.LoginApiKeyResponse, err error) {
	if err := l.svcCtx.AgentService.LoginAPIKey(l.ctx, strings.TrimSpace(req.ApiKey)); err != nil {
		return nil, err
	}
	account, err := l.svcCtx.AgentService.AccountStatus(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.LoginApiKeyResponse{Account: toAccountStatus(account)}, nil
}
