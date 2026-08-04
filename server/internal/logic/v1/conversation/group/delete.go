package group

import (
	"context"
	"errors"
	"net/http"

	"github.com/jzero-io/jzero-admin/core-engine/helper/auth"
	"github.com/jzero-io/jzero/core/stores/condition"
	"github.com/zeromicro/go-zero/core/logx"

	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	conversationgroupmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_group"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation/group"
)

type Delete struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

func NewDelete(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Delete {
	return &Delete{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx, r: r}
}

func (l *Delete) Delete(req *types.PathRequest) (resp *types.DeleteResponse, err error) {
	user, err := auth.Info(l.ctx)
	if err != nil {
		return nil, err
	}
	row, err := l.svcCtx.Model.ConversationGroup.FindOne(l.ctx, nil, req.GroupId)
	if err != nil || row.UserUuid != user.Uuid {
		return nil, errors.New("conversation group not found")
	}
	if err := l.svcCtx.Model.Conversation.UpdateFieldsByCondition(l.ctx, nil, map[string]any{string(conversationmodel.GroupUuid): nil}, condition.NewChain().Equal(conversationmodel.GroupUuid, req.GroupId).Equal(conversationmodel.UserUuid, user.Uuid).Build()...); err != nil {
		return nil, err
	}
	if err := l.svcCtx.Model.ConversationGroup.DeleteByCondition(l.ctx, nil, condition.NewChain().Equal(conversationgroupmodel.Uuid, req.GroupId).Equal(conversationgroupmodel.UserUuid, user.Uuid).Build()...); err != nil {
		return nil, err
	}
	return nil, nil
}
