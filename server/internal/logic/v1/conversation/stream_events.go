// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type StreamEvents struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 订阅会话与 Agent 流式事件
func NewStreamEvents(ctx context.Context, svcCtx *svc.ServiceContext) *StreamEvents {
	return &StreamEvents{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *StreamEvents) StreamEvents(req *types.EventsRequest, client chan<- *types.EventsResponse) error {
	return l.stream(req, client)
}

const streamHeartbeatInterval = 15 * time.Second

func (l *StreamEvents) stream(req *types.EventsRequest, client chan<- *types.EventsResponse) error {
	if l.svcCtx.AgentThreads == nil {
		return errors.New("agent runtime is disabled")
	}
	if _, err := requireOwner(l.ctx, l.svcCtx, req.ConversationId); err != nil {
		return err
	}
	if err := l.svcCtx.AgentThreads.ValidateThread(l.ctx, req.ConversationId); err != nil {
		return err
	}
	subscription := l.svcCtx.AgentThreads.Subscribe(req.ConversationId)
	defer subscription.Close()
	if !sendStreamResponse(l.ctx.Done(), client, &types.EventsResponse{
		Event: "stream.ready",
		Data:  "{}",
	}) {
		return nil
	}

	heartbeat := time.NewTicker(streamHeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-l.ctx.Done():
			return nil
		case event, ok := <-subscription.Events:
			if !ok {
				return nil
			}
			response, err := toEventsResponse(event)
			if err != nil {
				return err
			}
			if !sendStreamResponse(l.ctx.Done(), client, response) {
				return nil
			}
		case <-heartbeat.C:
			response := &types.EventsResponse{
				Event: "stream.heartbeat",
				Data:  "{}",
			}
			if !sendStreamResponse(l.ctx.Done(), client, response) {
				return nil
			}
		}
	}
}

func toEventsResponse(event agentdomain.StreamEvent) (*types.EventsResponse, error) {
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("marshal stream event: %w", err)
	}
	return &types.EventsResponse{
		Event: event.Type,
		Data:  string(payload),
	}, nil
}

func sendStreamResponse(done <-chan struct{}, client chan<- *types.EventsResponse, response *types.EventsResponse) bool {
	select {
	case client <- response:
		return true
	case <-done:
		return false
	}
}
