package agent

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/manage/agent"
)

type GetTokenQuotaGlobal struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewGetTokenQuotaGlobal(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *GetTokenQuotaGlobal {
	return &GetTokenQuotaGlobal{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *GetTokenQuotaGlobal) GetTokenQuotaGlobal(req *types.GetTokenQuotaGlobalRequest) (resp *types.GetTokenQuotaGlobalResponse, err error) {
	value, err := l.svcCtx.TokenQuota.Global(l.ctx)
	if err != nil {
		return nil, err
	}
	return &types.GetTokenQuotaGlobalResponse{
		Quota: types.TokenQuotaGlobal{
			FiveHourLimitTokens: value.FiveHourLimitTokens,
			SevenDayLimitTokens: value.SevenDayLimitTokens,
		},
	}, nil
}
