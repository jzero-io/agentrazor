package group

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
)

type Update struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewUpdate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Update {
	return &Update{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Update) Update(req *types.UpdateRequest) (resp *types.ConversationGroup, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, req.GroupId)
	if err != nil || row.UserUuid != user.Uuid {
		return nil, errors.New("conversation group not found")
	}
	values := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.New("group name is required")
		}
		values[string(conversationgroupmodel.Name)] = name
	}
	if len(values) > 0 {
		if err := l.svcCtx.Model.ConversationGroup.UpdateFieldsByCondition(l.ctx, nil, values, condition.NewChain().Equal(conversationgroupmodel.Uuid, req.GroupId).Equal(conversationgroupmodel.UserUuid, user.Uuid).Build()...); err != nil {
			return nil, err
		}
	}
	row, err = l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, req.GroupId)
	if err != nil {
		return nil, err
	}
	resp = &types.ConversationGroup{Id: row.Uuid, Name: row.Name, CreatedAt: row.CreateTime.UTC().Format(time.RFC3339Nano), UpdatedAt: row.UpdateTime.UTC().Format(time.RFC3339Nano)}
	return resp, nil
}
