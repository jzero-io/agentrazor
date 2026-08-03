package conversation

import (
	"context"
	"errors"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type Create struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 创建会话
func NewCreate(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Create {
	return &Create{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Create) Create(req *types.CreateRequest) (resp *types.Conversation, err error) {
	if l.svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	created, err := l.svcCtx.AgentThreads.Create(l.ctx, req.Title)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.Model.Conversation.InsertV2(l.ctx, nil, &conversationmodel.Conversation{Id: created.ID, UserUuid: uuid}); err != nil {
		return nil, err
	}
	response := toConversation(created)
	return &response, nil
}
