package group

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
)

type List struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewList(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *List {
	return &List{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *List) List() (resp *types.ListResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	rows, err := l.svcCtx.Model.ConversationGroup.FindByCondition(l.ctx, nil, condition.NewChain().Equal(conversationgroupmodel.UserUuid, user.Uuid).Build()...)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return rows[i].CreateTime.Before(rows[j].CreateTime)
	})
	resp = &types.ListResponse{Groups: make([]types.ConversationGroup, 0, len(rows))}
	for _, row := range rows {
		item := types.ConversationGroup{Id: row.Uuid, Name: row.Name, CreatedAt: row.CreateTime.UTC().Format(time.RFC3339Nano), UpdatedAt: row.UpdateTime.UTC().Format(time.RFC3339Nano)}
		resp.Groups = append(resp.Groups, item)
	}
	return resp, nil
}
