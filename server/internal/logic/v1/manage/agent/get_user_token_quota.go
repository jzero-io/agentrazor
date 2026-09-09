package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type GetUserTokenQuota struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetUserTokenQuota(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetUserTokenQuota {
	return &GetUserTokenQuota{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetUserTokenQuota) GetUserTokenQuota(req *types.GetUserTokenQuotaRequest) (resp *types.GetUserTokenQuotaResponse, err error) {
	if err := ensureTokenQuotaUser(l.ctx, l.svcCtx, req.UserUuid); err != nil {
		return nil, err
	}
	value, err := l.svcCtx.TokenQuota.Status(l.ctx, req.UserUuid)
	if err != nil {
		return nil, err
	}
	return &types.GetUserTokenQuotaResponse{Quota: toTokenQuotaUser(value)}, nil
}
