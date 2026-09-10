package conversation

import (
	"context"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/service/quota"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type TokenQuotaStatus struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取当前用户 Token 额度状态
func NewTokenQuotaStatus(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *TokenQuotaStatus {
	return &TokenQuotaStatus{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *TokenQuotaStatus) TokenQuotaStatus() (resp *types.TokenQuotaStatusResponse, err error) {
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	value, err := l.svcCtx.TokenQuota.Status(l.ctx, userUUID)
	if err != nil {
		return nil, err
	}
	result := &types.TokenQuotaStatusResponse{
		Enabled:  value.Enabled,
		FiveHour: toTokenQuotaWindow(value.FiveHour),
		SevenDay: toTokenQuotaWindow(value.SevenDay),
	}
	if value.QuotaResetAt != nil {
		formatted := value.QuotaResetAt.UTC().Format(time.RFC3339)
		result.QuotaResetAt = &formatted
	}
	return result, nil
}

func toTokenQuotaWindow(value quota.Window) types.TokenQuotaWindow {
	result := types.TokenQuotaWindow{
		Limited:          value.Limited,
		UsedTokens:       value.UsedTokens,
		LimitTokens:      value.LimitTokens,
		RemainingTokens:  value.RemainingTokens,
		RemainingPercent: value.RemainingPercent,
		Source:           value.Source,
	}
	if value.ResetAt != nil {
		formatted := value.ResetAt.UTC().Format(time.RFC3339)
		result.ResetAt = &formatted
	}
	return result
}
