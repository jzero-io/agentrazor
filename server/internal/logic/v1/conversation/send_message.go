package conversation

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type SendMessage struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 发送消息
func NewSendMessage(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *SendMessage {
	return &SendMessage{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *SendMessage) SendMessage(req *types.SendMessageRequest) (resp *types.SendMessageResponse, err error) {
	return sendMessage(l.ctx, l.svcCtx, req.ConversationId, req.GroupId, req.Content)
}

func sendMessage(ctx context.Context, svcCtx *svc.ServiceContext, conversationID string, groupID *string, content string) (*types.SendMessageResponse, error) {
	if svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content is required")
	}

	uuid, err := currentUserUUID(ctx)
	if err != nil {
		return nil, err
	}

	thread, created, err := ensureMessageThread(ctx, svcCtx, uuid, strings.TrimSpace(conversationID), groupID)
	if err != nil {
		return nil, err
	}
	if thread.Archived {
		return nil, agentdomain.ErrThreadArchived
	}

	run, err := svcCtx.AgentThreads.Send(thread.ID, content)
	if err != nil {
		if created {
			cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_ = svcCtx.AgentThreads.Delete(cleanupCtx, thread.ID)
		}
		return nil, err
	}

	conversation := toConversation(thread)
	conversation.Running = true
	startedAt := formatTime(run.CreatedAt)
	conversation.RunningStartedAt = &startedAt
	if groupID != nil && strings.TrimSpace(*groupID) != "" {
		value := strings.TrimSpace(*groupID)
		conversation.GroupId = &value
	}

	return &types.SendMessageResponse{
		ConversationId: thread.ID,
		Conversation:   conversation,
		Run:            toRun(run),
	}, nil
}

func ensureMessageThread(ctx context.Context, svcCtx *svc.ServiceContext, userUUID, conversationID string, groupID *string) (agentdomain.StoredThread, bool, error) {
	if conversationID != "" {
		if _, err := requireOwner(ctx, svcCtx, conversationID); err != nil {
			return agentdomain.StoredThread{}, false, err
		}
		thread, err := svcCtx.AgentThreads.Get(ctx, conversationID)
		return thread, false, err
	}

	groupUUID, err := normalizeOwnedGroupID(ctx, svcCtx, userUUID, groupID)
	if err != nil {
		return agentdomain.StoredThread{}, false, err
	}
	thread, err := svcCtx.AgentThreads.Create(ctx)
	if err != nil {
		return agentdomain.StoredThread{}, false, err
	}
	row := &conversationmodel.Conversation{
		Id:       thread.ID,
		UserUuid: userUUID,
	}
	if groupUUID != "" {
		row.GroupUuid = sql.NullString{String: groupUUID, Valid: true}
	}
	if err := svcCtx.Model.Conversation.InsertV2(ctx, nil, row); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = svcCtx.AgentThreads.Delete(cleanupCtx, thread.ID)
		return agentdomain.StoredThread{}, false, err
	}
	return thread, true, nil
}

func normalizeOwnedGroupID(ctx context.Context, svcCtx *svc.ServiceContext, userUUID string, groupID *string) (string, error) {
	if groupID == nil {
		return "", nil
	}
	value := strings.TrimSpace(*groupID)
	if value == "" {
		return "", nil
	}
	owner, exists, err := groupUser(ctx, svcCtx, value)
	if err != nil {
		return "", err
	}
	if !exists || owner != userUUID {
		return "", errors.New("conversation group not found")
	}
	return value, nil
}
