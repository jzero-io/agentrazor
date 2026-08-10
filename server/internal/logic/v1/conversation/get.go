package conversation

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jzero-io/agentrazor/core-engine/helper/auth"
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

func setConversationActiveRun(conversation *types.Conversation, threads *agentdomain.ThreadService, conversationID string) {
	run, running := threads.ActiveRun(conversationID)
	conversation.Running = running
	if running && !run.CreatedAt.IsZero() {
		value := formatTime(run.CreatedAt)
		conversation.RunningStartedAt = &value
	}
}

func toRun(value agentdomain.ThreadRun) types.Run {
	return types.Run{
		Id:        value.ID,
		Prompt:    value.Prompt,
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
		// 业务库有记录但 Codex thread 已不存在（孤儿数据）：按"会话不存在"处理
		if errors.Is(err, agentdomain.ErrThreadNotFound) {
			return nil, errConversationNotOwned
		}
		return nil, err
	}
	conversation := toConversation(thread)
	setConversationActiveRun(&conversation, svcCtx.AgentThreads, conversationID)
	detail := &types.DetailResponse{
		Conversation: conversation,
		SessionId:    thread.ID,
		EventCursor:  svcCtx.AgentThreads.EventCursor(conversationID),
		Turns:        make([]types.Turn, 0, len(thread.Turns)),
	}
	assignments, err := groupAssignments(ctx, svcCtx, uuid)
	if err != nil {
		return nil, err
	}
	if groupID := assignments[conversationID]; groupID != "" {
		detail.Conversation.GroupId = &groupID
	}
	for _, turn := range thread.Turns {
		items := make([]map[string]any, 0, len(turn.Items))
		for _, item := range turn.Items {
			copyItem := make(map[string]any, len(item)+2)
			for key, value := range item {
				copyItem[key] = value
			}
			if stringValue(item["type"]) == "imageGeneration" {
				if image, ok := generatedImage(item, svcCtx.MustGetConfig().Agent.CodexHome, conversationID); ok {
					copyItem["dataUrl"] = image.DataUrl
					copyItem["alt"] = image.Alt
				}
			}
			items = append(items, copyItem)
		}
		var startedAt, completedAt *string
		if !turn.CreatedAt.IsZero() {
			value := formatTime(turn.CreatedAt)
			startedAt = &value
		}
		if turn.CompletedAt != nil {
			value := formatTime(*turn.CompletedAt)
			completedAt = &value
		}
		detail.Turns = append(detail.Turns, types.Turn{
			Id: turn.ID, Status: turn.Status, StartedAt: startedAt, CompletedAt: completedAt,
			DurationMs: turn.DurationMs, Error: turn.Error, Items: items,
		})
	}
	return detail, nil
}

const maxGeneratedImageSize = 10 << 20

func generatedImage(item map[string]any, codexHome, conversationID string) (types.GeneratedImage, bool) {
	path := strings.TrimSpace(stringValue(item["savedPath"]))
	if path == "" {
		return types.GeneratedImage{}, false
	}
	root, err := filepath.Abs(filepath.Join(codexHome, "generated_images", conversationID))
	if err != nil {
		return types.GeneratedImage{}, false
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return types.GeneratedImage{}, false
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return types.GeneratedImage{}, false
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedPath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return types.GeneratedImage{}, false
	}
	info, err := os.Stat(resolvedPath)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxGeneratedImageSize {
		return types.GeneratedImage{}, false
	}
	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return types.GeneratedImage{}, false
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return types.GeneratedImage{}, false
	}
	alt := strings.TrimSpace(stringValue(item["revisedPrompt"]))
	if alt == "" {
		alt = "生成的图片"
	}
	return types.GeneratedImage{
		Id:      stringValue(item["id"]),
		DataUrl: "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data),
		Alt:     alt,
	}, true
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
