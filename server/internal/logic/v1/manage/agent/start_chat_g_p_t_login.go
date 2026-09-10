package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type StartChatGPTLogin struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewStartChatGPTLogin(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *StartChatGPTLogin {
	return &StartChatGPTLogin{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *StartChatGPTLogin) StartChatGPTLogin(req *types.StartChatGPTLoginRequest) (resp *types.StartChatGPTLoginResponse, err error) {
	if account, statusErr := l.svcCtx.AgentService.AccountStatus(l.ctx); statusErr == nil && account.LoggedIn {
		if err := l.svcCtx.AgentService.Logout(l.ctx); err != nil {
			return nil, err
		}
	}
	login, err := l.svcCtx.AgentService.StartChatGPTLogin(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.StartChatGPTLoginResponse{
		LoginId: login.LoginID, VerificationUrl: login.VerificationURL, UserCode: login.UserCode, ExpiresIn: login.ExpiresIn,
	}, nil
}
