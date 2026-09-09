package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type ResetUserTokenQuota struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewResetUserTokenQuota(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ResetUserTokenQuota {
	return &ResetUserTokenQuota{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *ResetUserTokenQuota) ResetUserTokenQuota(req *types.ResetUserTokenQuotaRequest) (resp *types.ResetUserTokenQuotaResponse, err error) {
	if err := ensureTokenQuotaUser(l.ctx, l.svcCtx, req.UserUuid); err != nil {
		return nil, err
	}
	if err := l.svcCtx.TokenQuota.ResetUser(l.ctx, req.UserUuid); err != nil {
		return nil, err
	}
	return &types.ResetUserTokenQuotaResponse{}, nil
}
