package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type GetAccountStatus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetAccountStatus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetAccountStatus {
	return &GetAccountStatus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetAccountStatus) GetAccountStatus(req *types.GetAccountStatusRequest) (resp *types.GetAccountStatusResponse, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, agentdomain.ErrServiceStopped
	}
	account, err := l.svcCtx.AgentThreads.AccountStatus(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetAccountStatusResponse{Account: toAccountStatus(account)}, nil
}
