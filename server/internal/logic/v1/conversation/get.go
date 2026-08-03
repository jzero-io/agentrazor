package conversation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jzero-io/jzero-admin/core-engine/helper/auth"
	"github.com/zeromicro/go-zero/core/logx"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type Get struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
	r      *http.Request
}

// 获取会话详情
func NewGet(ctx context.Context, svcCtx *svc.ServiceContext, r *http.Request) *Get {
	return &Get{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
		r:      r,
	}
}

func (l *Get) Get(req *types.PathRequest) (resp *types.DetailResponse, err error) {
	return buildDetail(l.ctx, l.svcCtx, req.ConversationId)
}

// errConversationNotOwned is returned when a conversation does not exist or is
// not owned by the requesting user. The message is intentionally generic so we
// do not leak which conversations exist for other users.
var errConversationNotOwned = errors.New("conversation not found")

// currentUserUUID extracts the authenticated user's UUID from the request
// context (populated by the JWT middleware).
func currentUserUUID(ctx context.Context) (string, error) {
	info, err := auth.Info(ctx)
	if err != nil {
		return "", err
	}
	return info.Uuid, nil
}

// requireOwner returns the requesting user's UUID after verifying they own the
// given conversation. If the conversation does not exist or belongs to another
// user, it returns errConversationNotOwned.
func requireOwner(ctx context.Context, svcCtx *svc.ServiceContext, threadID string) (string, error) {
	uuid, err := currentUserUUID(ctx)
	if err != nil {
		return "", err
	}
	owner, ok, err := conversationUser(ctx, svcCtx, threadID)
	if err != nil {
		return "", err
	}
	if !ok || owner != uuid {
		return "", errConversationNotOwned
	}
	return uuid, nil
}

func toConversation(value agentdomain.StoredThread) types.Conversation {
	title := strings.TrimSpace(value.Name)
	if title == "" {
		// codex does not auto-generate a name; fall back to its preview (a
		// short summary of the conversation, usually the first user message).
		title = strings.TrimSpace(value.Preview)
	}
	if title == "" {
		title = "新对话"
	}
	status := "active"
	if value.Archived {
		status = "archived"
	}
	result := types.Conversation{
		Id:        value.ID,
		Title:     title,
		Status:    status,
		CreatedAt: formatTime(value.CreatedAt),
		UpdatedAt: formatTime(value.UpdatedAt),
	}
	if value.IsPinned {
		result.PinnedAt = timePointer(value.UpdatedAt)
	}
	if value.Archived {
		result.ArchivedAt = timePointer(value.UpdatedAt)
	}
	return result
}

func toRun(value agentdomain.ThreadRun) types.Run {
	return types.Run{
		Id:        value.ID,
		Sequence:  0,
		Prompt:    value.Prompt,
		Status:    value.Status,
		CreatedAt: formatTime(value.CreatedAt),
	}
}

func buildDetail(ctx context.Context, svcCtx *svc.ServiceContext, conversationID string) (*types.DetailResponse, error) {
	if svcCtx.AgentThreads == nil {
		return nil, errors.New("agent runtime is disabled")
	}
	uuid, err := requireOwner(ctx, svcCtx, conversationID)
	if err != nil {
		return nil, err
	}
	thread, err := svcCtx.AgentThreads.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	detail := &types.DetailResponse{
		Conversation: toConversation(thread),
		SessionId:    thread.ID,
		Messages:     make([]types.Message, 0, len(thread.Turns)*2),
	}
	assignments, err := groupAssignments(ctx, svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	if groupID := assignments[conversationID]; groupID != "" {
		detail.Conversation.GroupId = &groupID
	}
	for _, turn := range thread.Turns {
		createdAt := turn.CreatedAt
		if createdAt.IsZero() {
			createdAt = thread.CreatedAt
		}
		var userParts []string
		var assistantParts []string
		var userID, assistantID string
		for _, item := range turn.Items {
			switch stringValue(item["type"]) {
			case "userMessage":
				if userID == "" {
					userID = stringValue(item["id"])
				}
				if text := itemText(item); text != "" {
					userParts = append(userParts, text)
				}
			case "agentMessage":
				if assistantID == "" {
					assistantID = stringValue(item["id"])
				}
				if text := itemText(item); text != "" {
					assistantParts = append(assistantParts, text)
				}
			}
		}
		if text := strings.Join(userParts, "\n"); text != "" {
			if userID == "" {
				userID = turn.ID + ":user"
			}
			detail.Messages = append(detail.Messages, types.Message{
				Id: userID, RunId: turn.ID, Role: "user", Content: text,
				Status: turn.Status, CreatedAt: formatTime(createdAt),
			})
		}
		assistantText := strings.Join(assistantParts, "\n\n")
		if assistantText == "" && turn.Error != "" {
			assistantText = turn.Error
		}
		if assistantText != "" {
			if assistantID == "" {
				assistantID = turn.ID + ":assistant"
			}
			completedAt := createdAt
			if turn.CompletedAt != nil {
				completedAt = *turn.CompletedAt
			}
			detail.Messages = append(detail.Messages, types.Message{
				Id: assistantID, RunId: turn.ID, Role: "assistant", Content: assistantText,
				Status: turn.Status, CreatedAt: formatTime(completedAt),
			})
		}
	}
	return detail, nil
}

func itemText(item map[string]any) string {
	if text := strings.TrimSpace(stringValue(item["text"])); text != "" {
		return text
	}
	content, ok := item["content"].([]any)
	if !ok {
		return strings.TrimSpace(stringValue(item["content"]))
	}
	parts := make([]string, 0, len(content))
	for _, value := range content {
		switch typed := value.(type) {
		case string:
			parts = append(parts, typed)
		case map[string]any:
			if text := stringValue(typed["text"]); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return ""
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func timePointer(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := formatTime(value)
	return &formatted
}
