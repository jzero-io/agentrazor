package agent

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func TestIsolatedCodexEnvironment(t *testing.T) {
	environment := isolatedCodexEnvironment([]string{
		"PATH=/usr/bin",
		"CODEX_HOME=/Users/test/.codex",
		"CODEX_SQLITE_HOME=/Users/test/.codex",
		"OPENAI_API_KEY=test",
	}, "/srv/police-ai/codex-home")

	if slices.Contains(environment, "CODEX_HOME=/Users/test/.codex") {
		t.Fatal("parent CODEX_HOME was retained")
	}
	if slices.Contains(environment, "CODEX_SQLITE_HOME=/Users/test/.codex") {
		t.Fatal("parent CODEX_SQLITE_HOME was retained")
	}
	if !slices.Contains(environment, "CODEX_HOME=/srv/police-ai/codex-home") {
		t.Fatal("isolated CODEX_HOME was not set")
	}
	if !slices.Contains(environment, "OPENAI_API_KEY=test") {
		t.Fatal("unrelated authentication environment was removed")
	}
}

func TestAppServerTurnCollectsEventsAndFinalOutput(t *testing.T) {
	var emitted []map[string]any
	execution := &appServerTurn{
		threadID:  "thread-1",
		maxEvents: 10,
		emit: func(event map[string]any) {
			emitted = append(emitted, event)
		},
		done: make(chan turnOutcome, 1),
	}

	execution.handleNotification("turn/started", map[string]any{
		"threadId": "thread-1",
		"turn":     map[string]any{"id": "turn-1"},
	})
	execution.handleNotification("item/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "item-1",
			"type": "agentMessage",
			"text": "first",
		},
	})
	execution.handleNotification("item/completed", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"item": map[string]any{
			"id":   "item-2",
			"type": "agentMessage",
			"text": "second",
		},
	})
	execution.handleNotification("turn/completed", map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "completed",
		},
	})

	outcome := <-execution.done
	if outcome.err != nil {
		t.Fatal(outcome.err)
	}
	result := execution.result("thread-1")
	if result.ExternalSessionID != "thread-1" {
		t.Fatalf("external session id = %q", result.ExternalSessionID)
	}
	if result.Output != "first\nsecond" {
		t.Fatalf("output = %q", result.Output)
	}
	if len(result.Events) != 4 || len(emitted) != 4 {
		t.Fatalf("events = %d, emitted = %d", len(result.Events), len(emitted))
	}
	if result.Events[1]["type"] != "item.completed" {
		t.Fatalf("event type = %#v", result.Events[1]["type"])
	}
}

func TestAppServerTurnStreamsAgentMessageDeltaWithoutPersistingIt(t *testing.T) {
	var emitted []map[string]any
	execution := &appServerTurn{
		threadID:  "thread-1",
		maxEvents: 10,
		emit: func(event map[string]any) {
			emitted = append(emitted, event)
		},
		done: make(chan turnOutcome, 1),
	}

	execution.handleNotification("item/agentMessage/delta", map[string]any{
		"threadId": "thread-1",
		"turnId":   "turn-1",
		"itemId":   "item-1",
		"delta":    "正在分析",
	})

	if len(emitted) != 1 {
		t.Fatalf("emitted events = %d, want 1", len(emitted))
	}
	if emitted[0]["type"] != "item.agentMessage.delta" {
		t.Fatalf("event type = %#v", emitted[0]["type"])
	}
	if len(execution.result("thread-1").Events) != 0 {
		t.Fatal("streaming delta should not be persisted in the run result")
	}
}

func TestAppServerTurnPropagatesFailure(t *testing.T) {
	execution := &appServerTurn{
		threadID:  "thread-1",
		maxEvents: 10,
		done:      make(chan turnOutcome, 1),
	}
	execution.handleNotification("turn/completed", map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "failed",
			"error":  map[string]any{"message": "model unavailable"},
		},
	})
	outcome := <-execution.done
	if outcome.err == nil || outcome.err.Error() != "model unavailable" {
		t.Fatalf("outcome error = %v", outcome.err)
	}
}

func TestAppServerTurnPropagatesInterruption(t *testing.T) {
	execution := &appServerTurn{
		threadID:  "thread-1",
		maxEvents: 10,
		done:      make(chan turnOutcome, 1),
	}
	execution.handleNotification("turn/completed", map[string]any{
		"threadId": "thread-1",
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "interrupted",
		},
	})
	outcome := <-execution.done
	if !errors.Is(outcome.err, context.Canceled) {
		t.Fatalf("outcome error = %v", outcome.err)
	}
}
