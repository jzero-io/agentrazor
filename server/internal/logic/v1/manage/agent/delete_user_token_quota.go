package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type DeleteUserTokenQuota struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDeleteUserTokenQuota(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *DeleteUserTokenQuota {
	return &DeleteUserTokenQuota{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *DeleteUserTokenQuota) DeleteUserTokenQuota(req *types.DeleteUserTokenQuotaRequest) (resp *types.DeleteUserTokenQuotaResponse, err error) {
	if err := ensureTokenQuotaUser(l.ctx, l.svcCtx, req.UserUuid); err != nil {
		return nil, err
	}
	if err := l.svcCtx.TokenQuota.DeleteUser(l.ctx, req.UserUuid); err != nil {
		return nil, err
	}
	return &types.DeleteUserTokenQuotaResponse{}, nil
}
