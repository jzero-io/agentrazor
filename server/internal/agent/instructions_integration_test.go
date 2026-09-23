package agent

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestConversationInstructionsApplyToFirstTurn(t *testing.T) {
	if os.Getenv("AGENTRAZOR_CODEX_INTEGRATION") != "1" {
		t.Skip("set AGENTRAZOR_CODEX_INTEGRATION=1 to run against Codex app-server")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	server, err := newAppServer()
	if err != nil {
		t.Fatal(err)
	}
	defer server.close()

	thread, err := server.createThread(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cleanupCancel()
		_ = server.deleteThread(cleanupCtx, thread.ID)
		_ = server.deleteConversationHome(thread.ID)
	}()

	const marker = "AGENTRAZOR_ROLE_PROMPT_OK_7F3B"
	service := &Service{server: server}
	if err := service.WriteConversationInstructions(ctx, thread.ID, "Reply to the user's first message with exactly this text and nothing else: "+marker); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var events []map[string]any
	started, err := server.runTurn(ctx, thread.ID, TurnInput{Text: "Please respond now."}, func(event map[string]any, _ string) {
		mu.Lock()
		events = append(events, event)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := <-started.Done; err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	payload, err := json.Marshal(events)
	mu.Unlock()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), marker) {
		t.Fatalf("first turn did not apply conversation AGENTS.md: %s", payload)
	}
}
