package conversation

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"time"

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
	userUUID, err := currentUserUUID(l.ctx)
	if err != nil {
		return nil, err
	}
	groupUUID, err := normalizeOwnedGroupID(l.ctx, l.svcCtx, userUUID, req.GroupId)
	if err != nil {
		return nil, err
	}
	thread, err := l.svcCtx.AgentThreads.Create(l.ctx)
	if err != nil {
		return nil, err
	}
	row := &conversationmodel.Conversation{Id: thread.ID, UserUuid: userUUID}
	if groupUUID != "" {
		row.GroupUuid = sql.NullString{String: groupUUID, Valid: true}
	}
	if err := l.svcCtx.Model.Conversation.InsertV2(l.ctx, nil, row); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = l.svcCtx.AgentThreads.Delete(cleanupCtx, thread.ID)
		return nil, err
	}

	conversation := toConversation(thread)
	if groupUUID != "" {
		conversation.GroupId = &groupUUID
	}
	return &conversation, nil
}
