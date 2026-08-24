package agent

import "encoding/json"

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
