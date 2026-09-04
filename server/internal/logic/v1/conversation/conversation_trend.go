package conversation

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type ConversationTrend struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取对话数量趋势
func NewConversationTrend(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *ConversationTrend {
	return &ConversationTrend{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *ConversationTrend) ConversationTrend(req *types.ConversationTrendRequest) (*types.ConversationTrendResponse, error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
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

	stats := &Stats{ctx: l.ctx, svcCtx: l.svcCtx}
	owned, err := stats.conversationRows(userUUID, superAdmin)
	if err != nil {
		return nil, err
	}
	threads, err := stats.listOwnedThreads(owned)
	if err != nil {
		return nil, err
	}

	return &types.ConversationTrendResponse{
		Dimension: dimension,
		Points:    buildConversationTrend(threads, dimension, time.Now()),
	}, nil
}

func buildConversationTrend(threads []agentdomain.StoredThread, dimension string, now time.Time) []types.ConversationTrendPoint {
	location := now.Location()
	var start time.Time
	count := 30
	periodLayout := "2006-01-02"
	if dimension == "month" {
		start = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location).AddDate(0, -11, 0)
		count = 12
		periodLayout = "2006-01"
	} else {
		start = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).AddDate(0, 0, -29)
	}

	points := make([]types.ConversationTrendPoint, count)
	indexes := make(map[string]int, count)
	for i := 0; i < count; i++ {
		bucket := start.AddDate(0, i, 0)
		if dimension == "day" {
			bucket = start.AddDate(0, 0, i)
		}
		period := bucket.Format(periodLayout)
		points[i].Period = period
		indexes[period] = i
	}

	end := start.AddDate(0, count, 0)
	if dimension == "day" {
		end = start.AddDate(0, 0, count)
	}
	add := func(value time.Time, archived bool) {
		if value.IsZero() {
			return
		}
		value = value.In(location)
		if value.Before(start) || !value.Before(end) {
			return
		}
		index, ok := indexes[value.Format(periodLayout)]
		if !ok {
			return
		}
		if archived {
			points[index].ArchivedConversations++
		} else {
			points[index].TotalConversations++
		}
	}

	for _, thread := range threads {
		add(thread.CreatedAt, false)
		if thread.Archived {
			add(thread.UpdatedAt, true)
		}
	}
	return points
}
