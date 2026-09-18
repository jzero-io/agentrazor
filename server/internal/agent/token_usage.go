package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"

	conversationmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation"
	conversationtokenusageeventmodel "github.com/jzero-io/agentrazor/server/internal/model/conversation_token_usage_event"
)

type tokenUsageStore struct {
	conversation conversationmodel.ConversationModel
	usageEvent   conversationtokenusageeventmodel.ConversationTokenUsageEventModel
}

func (s tokenUsageStore) record(ctx context.Context, event TokenUsageEvent) error {
	conversation, err := s.conversation.FindOne(ctx, nil, event.ConversationID)
	if errors.Is(err, conversationmodel.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}

	var modelContextWindow sql.NullInt64
	if event.ModelContextWindow != nil {
		modelContextWindow = sql.NullInt64{Int64: *event.ModelContextWindow, Valid: true}
	}
	return s.usageEvent.InsertV2(ctx, nil, &conversationtokenusageeventmodel.ConversationTokenUsageEvent{
		ConversationId:             event.ConversationID,
		UserUuid:                   conversation.UserUuid,
		TurnId:                     event.TurnID,
		LastInputTokens:            event.Last.InputTokens,
		LastCachedInputTokens:      event.Last.CachedInputTokens,
		LastCacheWriteInputTokens:  event.Last.CacheWriteInputTokens,
		LastOutputTokens:           event.Last.OutputTokens,
		LastReasoningOutputTokens:  event.Last.ReasoningOutputTokens,
		LastTotalTokens:            event.Last.TotalTokens,
		TotalInputTokens:           event.Total.InputTokens,
		TotalCachedInputTokens:     event.Total.CachedInputTokens,
		TotalCacheWriteInputTokens: event.Total.CacheWriteInputTokens,
		TotalOutputTokens:          event.Total.OutputTokens,
		TotalReasoningOutputTokens: event.Total.ReasoningOutputTokens,
		TotalTokens:                event.Total.TotalTokens,
		ModelContextWindow:         modelContextWindow,
	})
}

// tokenUsageWriter keeps database work off the app-server reader while
// preserving event order and draining accepted events during shutdown.
type tokenUsageWriter struct {
	mu     sync.Mutex
	ready  *sync.Cond
	queue  []TokenUsageEvent
	write  func(TokenUsageEvent)
	closed bool
	done   chan struct{}
}

func newTokenUsageWriter(write func(TokenUsageEvent)) *tokenUsageWriter {
	w := &tokenUsageWriter{write: write, done: make(chan struct{})}
	w.ready = sync.NewCond(&w.mu)
	go w.run()
	return w
}

func (w *tokenUsageWriter) enqueue(event TokenUsageEvent) {
	w.mu.Lock()
	if !w.closed {
		w.queue = append(w.queue, event)
		w.ready.Signal()
	}
	w.mu.Unlock()
}

func (w *tokenUsageWriter) close() {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		w.ready.Broadcast()
	}
	w.mu.Unlock()
	<-w.done
}

func (w *tokenUsageWriter) run() {
	defer close(w.done)
	for {
		w.mu.Lock()
		for len(w.queue) == 0 && !w.closed {
			w.ready.Wait()
		}
		if len(w.queue) == 0 {
			w.mu.Unlock()
			return
		}
		event := w.queue[0]
		w.queue[0] = TokenUsageEvent{}
		w.queue = w.queue[1:]
		w.mu.Unlock()
		w.write(event)
	}
}

func (s *Service) persistTokenUsage(usage TokenUsageEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := s.tokenUsageStore.record(ctx, usage); err != nil {
		logx.Errorf("record Codex token usage failed: %v", err)
	}
}

func tokenUsageEventFromCodex(event map[string]any) (TokenUsageEvent, bool) {
	params, ok := event["params"].(map[string]any)
	if !ok {
		return TokenUsageEvent{}, false
	}
	threadID := stringValue(params["threadId"])
	turnID := stringValue(params["turnId"])
	if threadID == "" || turnID == "" {
		return TokenUsageEvent{}, false
	}
	tokenUsage, ok := params["tokenUsage"].(map[string]any)
	if !ok {
		return TokenUsageEvent{}, false
	}
	last, ok := tokenUsageBreakdownFromCodex(tokenUsage["last"])
	if !ok {
		return TokenUsageEvent{}, false
	}
	total, ok := tokenUsageBreakdownFromCodex(tokenUsage["total"])
	if !ok {
		return TokenUsageEvent{}, false
	}
	usage := TokenUsageEvent{
		ConversationID: threadID,
		TurnID:         turnID,
		Last:           last,
		Total:          total,
	}
	if value, ok := codexTokenInt64Value(tokenUsage["modelContextWindow"]); ok {
		usage.ModelContextWindow = &value
	}
	return usage, true
}

func tokenUsageBreakdownFromCodex(value any) (TokenUsageBreakdown, bool) {
	raw, ok := value.(map[string]any)
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	inputTokens, ok := requiredInt64(raw, "inputTokens")
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	cachedInputTokens, ok := requiredInt64(raw, "cachedInputTokens")
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	outputTokens, ok := requiredInt64(raw, "outputTokens")
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	reasoningOutputTokens, ok := requiredInt64(raw, "reasoningOutputTokens")
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	totalTokens, ok := requiredInt64(raw, "totalTokens")
	if !ok {
		return TokenUsageBreakdown{}, false
	}
	cacheWriteInputTokens, _ := codexTokenInt64Value(raw["cacheWriteInputTokens"])
	return TokenUsageBreakdown{
		InputTokens:           inputTokens,
		CachedInputTokens:     cachedInputTokens,
		CacheWriteInputTokens: cacheWriteInputTokens,
		OutputTokens:          outputTokens,
		ReasoningOutputTokens: reasoningOutputTokens,
		TotalTokens:           totalTokens,
	}, true
}

func requiredInt64(raw map[string]any, key string) (int64, bool) {
	return codexTokenInt64Value(raw[key])
}

func codexTokenInt64Value(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case float32:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case int32:
		return int64(typed), true
	case json.Number:
		parsed, err := typed.Int64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
