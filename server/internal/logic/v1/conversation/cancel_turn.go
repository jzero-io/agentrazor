package conversation

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type CancelTurn struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 停止当前 Turn
func NewCancelTurn(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *CancelTurn {
	return &CancelTurn{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *CancelTurn) CancelTurn(req *types.PathRequest) (resp *types.CancelTurnResponse, err error) {
	if _, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return nil, err
	}
	if err := l.svcCtx.AgentThreads.Cancel(req.ConversationId); err != nil {
		return nil, err
	}
	return &types.CancelTurnResponse{}, nil
}
