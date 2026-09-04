package conversation

import (
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type TokenUsageTrend struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取 Token 消耗趋势
func NewTokenUsageTrend(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *TokenUsageTrend {
	return &TokenUsageTrend{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *TokenUsageTrend) TokenUsageTrend(req *types.TokenUsageTrendRequest) (resp *types.TokenUsageTrendResponse, err error) {
	dimension := req.Dimension
	if dimension == "" {
		dimension = "day"
	}
	if dimension != "day" && dimension != "month" {
		return nil, errors.New("dimension must be day or month")
	}

	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	superAdmin, err := isSuperAdmin(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if superAdmin {
		userUUID = ""
	}

	rows, err := l.svcCtx.Model.ConversationTokenUsageEvent.TokenUsageTrend(l.ctx, userUUID, dimension)
	if err != nil {
		return nil, err
	}
	points := make([]types.TokenUsageTrendPoint, 0, len(rows))
	for _, row := range rows {
		points = append(points, types.TokenUsageTrendPoint{Period: row.Period, Tokens: row.Tokens})
	}
	return &types.TokenUsageTrendResponse{Dimension: dimension, Points: points}, nil
}
