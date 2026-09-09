package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type SaveTokenQuotaGlobal struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewSaveTokenQuotaGlobal(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SaveTokenQuotaGlobal {
	return &SaveTokenQuotaGlobal{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SaveTokenQuotaGlobal) SaveTokenQuotaGlobal(req *types.SaveTokenQuotaGlobalRequest) (resp *types.SaveTokenQuotaGlobalResponse, err error) {
	if err := l.svcCtx.TokenQuota.SaveGlobal(
		l.ctx,
		req.FiveHourLimitTokens,
		req.SevenDayLimitTokens,
	); err != nil {
		return nil, err
	}
	return &types.SaveTokenQuotaGlobalResponse{}, nil
}
