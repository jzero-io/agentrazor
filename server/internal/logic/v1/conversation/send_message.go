package conversation

import (
	"context"
	"net/http"
	"strings"

	"github.com/jzero-io/jzero/core/status"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/errcodes"
	"github.com/jzero-io/agentrazor/server/internal/quota"
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

func (l *SendMessage) SendMessage(req *types.SendMessageRequest) (resp *types.StartedTurn, err error) {
	return sendMessage(l.ctx, l.svcCtx, req.ConversationId, req.Content)
}

func sendMessage(ctx context.Context, svcCtx *svc.ServiceContext, conversationID, content string) (*types.StartedTurn, error) {
	if svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, errors.New("message content is required")
	}

	conversationID = strings.TrimSpace(conversationID)
	userUUID, err := requireOwner(ctx, svcCtx, conversationID)
	if err != nil {
		return nil, err
	}
	thread, err := svcCtx.AgentThreads.Metadata(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if thread.Archived {
		return nil, agentdomain.ErrThreadArchived
	}
	if err := svcCtx.TokenQuota.Check(ctx, userUUID); err != nil {
		switch {
		case errors.Is(err, quota.ErrAgentDisabled):
			return nil, status.Wrap(errcodes.AgentDisabledCode, err)
		case errors.Is(err, quota.ErrFiveHourQuotaExceeded):
			return nil, status.Wrap(errcodes.FiveHourQuotaExceededCode, err)
		case errors.Is(err, quota.ErrSevenDayQuotaExceeded):
			return nil, status.Wrap(errcodes.SevenDayQuotaExceededCode, err)
		default:
			return nil, err
		}
	}

	turn, err := svcCtx.AgentThreads.Send(thread.ID, content)
	if err != nil {
		return nil, err
	}

	result := toStartedTurn(turn)
	return &result, nil
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
