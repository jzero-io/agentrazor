package agent

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ThreadRuntime is the storage and execution surface used by the conversation
// service. Codex app-server owns the persisted thread data.
type ThreadRuntime interface {
	CreateStoredThread(ctx context.Context) (StoredThread, error)
	ListStoredThreads(ctx context.Context, archived bool) ([]StoredThread, error)
	ReadStoredThread(ctx context.Context, threadID string, includeTurns bool) (StoredThread, error)
	SetThreadName(ctx context.Context, threadID, name string) error
	SetThreadPinned(ctx context.Context, threadID string, pinned bool) error
	ArchiveStoredThread(ctx context.Context, threadID string) error
	UnarchiveStoredThread(ctx context.Context, threadID string) (StoredThread, error)
	DeleteThread(ctx context.Context, threadID string) error
	DeleteConversationHome(threadID string) error
	StartTurn(ctx context.Context, threadID, prompt string, emit EventHandler) (StartedTurn, error)
	Close() error
}

type StartedTurn struct {
	ID        string
	StartedAt time.Time
	Done      <-chan error
}

func (r *CodexAppServerRuntime) CreateStoredThread(ctx context.Context) (StoredThread, error) {
	threadID, err := r.startThread(ctx)
	if err != nil {
		return StoredThread{}, err
	}
	if err := r.createConversationHome(threadID); err != nil {
		_ = r.DeleteThread(ctx, threadID)
		return StoredThread{}, err
	}
	thread, err := r.ReadStoredThread(ctx, threadID, false)
	if err == nil {
		return thread, nil
	}
	now := time.Now().UTC()
	return StoredThread{
		ID:        threadID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (r *CodexAppServerRuntime) ListStoredThreads(ctx context.Context, archived bool) ([]StoredThread, error) {
	var threads []StoredThread
	var cursor string
	for {
		params := map[string]any{
			"limit":         100,
			"archived":      archived,
			"sortKey":       "updated_at",
			"sortDirection": "desc",
			// Omitting sourceKinds only returns interactive CLI/VS Code
			// threads. Include every stable source kind because app-server
			// versions differ in how service-created threads are classified.
			"sourceKinds": []string{
				"cli", "vscode", "exec", "appServer",
				"subAgent", "subAgentReview", "subAgentCompact",
				"subAgentThreadSpawn", "subAgentOther", "unknown",
			},
		}
		if cursor != "" {
			params["cursor"] = cursor
		}
		result, err := r.request(ctx, "thread/list", params)
		if err != nil {
			return nil, fmt.Errorf("list Codex threads: %w", err)
		}
		if values, ok := result["data"].([]any); ok {
			for _, value := range values {
				raw, ok := value.(map[string]any)
				if !ok {
					continue
				}
				thread := decodeStoredThread(raw, archived)
				if thread.ID != "" {
					threads = append(threads, thread)
				}
			}
		}
		cursor = stringValue(result["nextCursor"])
		if cursor == "" {
			return threads, nil
		}
	}
}

func (r *CodexAppServerRuntime) ReadStoredThread(ctx context.Context, threadID string, includeTurns bool) (StoredThread, error) {
	thread, err := r.readThread(ctx, threadID, includeTurns)
	if err != nil {
		// thread/read reads stored threads (including archived ones) without
		// resuming. It only fails for threads not loaded in this process, so on
		// failure load (resume) and read again. Resume is intentionally NOT
		// called first: it fails for archived threads ("session is archived"),
		// which read fine directly. See openai/codex#27395.
		if _, resumeErr := r.ensureThread(ctx, threadID); resumeErr == nil {
			thread, err = r.readThread(ctx, threadID, includeTurns)
		} else if threadMissingError(resumeErr) {
			// 业务库有记录但 Codex thread 已不存在（resume 报 "no rollout
			// found"）：按"会话不存在"处理，避免把原始 RPC 错误抛给前端。
			return StoredThread{}, ErrThreadNotFound
		}
	}
	if err != nil {
		return StoredThread{}, fmt.Errorf("read Codex thread %s: %w", threadID, err)
	}
	return thread, nil
}

func (r *CodexAppServerRuntime) readThread(ctx context.Context, threadID string, includeTurns bool) (StoredThread, error) {
	result, streamPosition, err := r.requestWithPosition(ctx, "thread/read", map[string]any{
		"threadId":     threadID,
		"includeTurns": includeTurns,
	})
	if includeTurns && threadNotMaterializedError(err) {
		// app-server cannot include turns before the first user message. Read the
		// draft's metadata explicitly, but retain the rejected response position:
		// events after that boundary may contain the first turn and must be applied.
		var metadataPosition string
		result, metadataPosition, err = r.requestWithPosition(ctx, "thread/read", map[string]any{
			"threadId":     threadID,
			"includeTurns": false,
		})
		if streamPosition == "" {
			streamPosition = metadataPosition
		}
	}
	if err != nil {
		return StoredThread{}, err
	}
	raw, ok := result["thread"].(map[string]any)
	if !ok {
		return StoredThread{}, fmt.Errorf("Codex thread/read response did not contain thread %s", threadID)
	}
	thread := decodeStoredThread(raw, false)
	if archived, known := archiveValue(raw); known {
		thread.Archived = archived
	}
	if thread.ID == "" {
		return StoredThread{}, errors.New("Codex thread/read response did not contain a thread id")
	}
	thread.StreamPosition = streamPosition
	return thread, nil
}

// threadMissingError reports whether err is the app-server -32600 RPC error
// raised while loading a missing thread. At this point readThread has already
// failed (so the thread is not simply archived — archived threads read fine),
// and the remaining -32600 resume failure is "no rollout found".
func threadMissingError(err error) bool {
	var rpcErr *RPCError
	return errors.As(err, &rpcErr) && rpcErr.Code == -32600
}

func threadNotMaterializedError(err error) bool {
	var rpcErr *RPCError
	return errors.As(err, &rpcErr) &&
		rpcErr.Code == -32600 &&
		strings.Contains(rpcErr.Message, "is not materialized yet") &&
		strings.Contains(rpcErr.Message, "includeTurns is unavailable before first user message")
}

func (r *CodexAppServerRuntime) SetThreadName(ctx context.Context, threadID, name string) error {
	_, err := r.request(ctx, "thread/name/set", map[string]any{
		"threadId": threadID,
		"name":     name,
	})
	if err != nil {
		return fmt.Errorf("set Codex thread name: %w", err)
	}
	return nil
}

const pinnedThreadSectionName = "Pinned"

func (r *CodexAppServerRuntime) SetThreadPinned(ctx context.Context, threadID string, pinned bool) error {
	var sectionID any
	if pinned {
		id, err := r.pinnedThreadSectionID(ctx)
		if err != nil {
			return err
		}
		sectionID = id
	}
	_, err := r.request(ctx, "thread/section/move", map[string]any{
		"threadId":  threadID,
		"sectionId": sectionID,
	})
	if err != nil {
		return fmt.Errorf("update Codex thread pin state: %w", err)
	}
	return nil
}

func (r *CodexAppServerRuntime) pinnedThreadSectionID(ctx context.Context) (string, error) {
	var cursor string
	for {
		params := map[string]any{"limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		result, err := r.request(ctx, "threadSection/list", params)
		if err != nil {
			return "", fmt.Errorf("list Codex thread sections: %w", err)
		}
		if values, ok := result["data"].([]any); ok {
			for _, value := range values {
				raw, ok := value.(map[string]any)
				if !ok {
					continue
				}
				if stringValue(raw["name"]) == pinnedThreadSectionName {
					id := stringValue(raw["id"])
					if id == "" {
						return "", errors.New("Codex pinned thread section did not contain an id")
					}
					return id, nil
				}
			}
		}
		cursor = stringValue(result["nextCursor"])
		if cursor == "" {
			return "", errors.New("Codex pinned thread section not found")
		}
	}
}

func (r *CodexAppServerRuntime) ArchiveStoredThread(ctx context.Context, threadID string) error {
	if _, err := r.request(ctx, "thread/archive", map[string]any{"threadId": threadID}); err != nil {
		if threadMissingError(err) && r.threadListed(ctx, threadID, true) {
			return nil
		}
		return fmt.Errorf("archive Codex thread: %w", err)
	}
	r.stateMu.Lock()
	delete(r.loaded, threadID)
	r.stateMu.Unlock()
	return nil
}

func (r *CodexAppServerRuntime) threadListed(ctx context.Context, threadID string, archived bool) bool {
	threads, err := r.ListStoredThreads(ctx, archived)
	if err != nil {
		return false
	}
	for _, thread := range threads {
		if thread.ID == threadID {
			return true
		}
	}
	return false
}

func (r *CodexAppServerRuntime) DeleteThread(ctx context.Context, threadID string) error {
	if _, err := r.request(ctx, "thread/delete", map[string]any{"threadId": threadID}); err != nil {
		return fmt.Errorf("delete Codex thread: %w", err)
	}
	r.stateMu.Lock()
	delete(r.loaded, threadID)
	r.stateMu.Unlock()
	return nil
}

func (r *CodexAppServerRuntime) UnarchiveStoredThread(ctx context.Context, threadID string) (StoredThread, error) {
	result, err := r.request(ctx, "thread/unarchive", map[string]any{"threadId": threadID})
	if err != nil {
		return StoredThread{}, fmt.Errorf("unarchive Codex thread: %w", err)
	}
	raw, ok := result["thread"].(map[string]any)
	if !ok {
		return r.ReadStoredThread(ctx, threadID, false)
	}
	return decodeStoredThread(raw, false), nil
}

func decodeStoredThread(raw map[string]any, archived bool) StoredThread {
	preview := stringValue(raw["preview"])
	thread := StoredThread{
		ID:        stringValue(raw["id"]),
		Name:      threadName(raw, preview),
		Preview:   preview,
		IsPinned:  threadInPinnedSection(raw),
		Archived:  archived,
		CreatedAt: timeValue(raw["createdAt"]),
		UpdatedAt: timeValue(raw["updatedAt"]),
	}
	if thread.CreatedAt.IsZero() {
		thread.CreatedAt = thread.UpdatedAt
	}
	if thread.UpdatedAt.IsZero() {
		thread.UpdatedAt = thread.CreatedAt
	}
	if values, ok := raw["turns"].([]any); ok {
		thread.Turns = make([]StoredTurn, 0, len(values))
		for _, value := range values {
			turnRaw, ok := value.(map[string]any)
			if !ok {
				continue
			}
			turn := StoredTurn{
				ID:        stringValue(turnRaw["id"]),
				Status:    stringValue(turnRaw["status"]),
				CreatedAt: timeValue(turnRaw["startedAt"]),
			}
			if completed := timeValue(turnRaw["completedAt"]); !completed.IsZero() {
				turn.CompletedAt = &completed
			}
			if duration, ok := int64Value(turnRaw["durationMs"]); ok && duration >= 0 {
				turn.DurationMs = &duration
			}
			if detail, ok := turnRaw["error"].(map[string]any); ok {
				turn.Error = stringValue(detail["message"])
			} else {
				turn.Error = stringValue(turnRaw["error"])
			}
			if items, ok := turnRaw["items"].([]any); ok {
				turn.Items = make([]map[string]any, 0, len(items))
				for _, item := range items {
					if itemMap, ok := item.(map[string]any); ok {
						turn.Items = append(turn.Items, itemMap)
					}
				}
			}
			thread.Turns = append(thread.Turns, turn)
		}
	}
	return thread
}

func threadInPinnedSection(raw map[string]any) bool {
	section, ok := raw["section"].(map[string]any)
	return ok && stringValue(section["name"]) == pinnedThreadSectionName
}

func threadName(raw map[string]any, preview string) string {
	for _, key := range []string{"name", "threadName", "thread_name", "title"} {
		if value := strings.TrimSpace(stringValue(raw[key])); value != "" {
			return value
		}
	}
	return strings.TrimSpace(preview)
}

func int64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	default:
		return 0, false
	}
}

func archiveValue(raw map[string]any) (bool, bool) {
	for _, key := range []string{"archived", "isArchived"} {
		switch value := raw[key].(type) {
		case bool:
			return value, true
		case string:
			parsed, err := strconv.ParseBool(value)
			if err == nil {
				return parsed, true
			}
		}
	}
	return false, false
}

func timeValue(value any) time.Time {
	switch typed := value.(type) {
	case float64:
		return time.Unix(int64(typed), 0).UTC()
	case int64:
		return time.Unix(typed, 0).UTC()
	case int:
		return time.Unix(int64(typed), 0).UTC()
	case string:
		typed = strings.TrimSpace(typed)
		if typed == "" {
			return time.Time{}
		}
		if unix, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return time.Unix(unix, 0).UTC()
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.UTC()
		}
	}
	return time.Time{}
}
