package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SaveUserTokenQuota struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveUserTokenQuota(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveUserTokenQuota {
	return &SaveUserTokenQuota{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveUserTokenQuota) SaveUserTokenQuota(req *types.SaveUserTokenQuotaRequest) (resp *types.SaveUserTokenQuotaResponse, err error) {
	if err := ensureTokenQuotaUser(l.ctx, l.svcCtx.Model.ManageUser, req.UserUuid); err != nil {
		return nil, err
	}
	if err := l.svcCtx.TokenQuota.SaveUser(
		l.ctx,
		req.UserUuid,
		req.Enabled,
		req.FiveHourDisabled,
		req.FiveHourLimitTokens,
		req.SevenDayLimitTokens,
	); err != nil {
		return nil, err
	}
	return &types.SaveUserTokenQuotaResponse{}, nil
}
