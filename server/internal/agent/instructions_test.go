package agent

import (
	"context"
	"strings"
	"testing"
)

func TestWriteConversationInstructionsSkipsEmptyPrompt(t *testing.T) {
	var service Service
	if err := service.WriteConversationInstructions(context.Background(), "conversation-id", "  \n\t"); err != nil {
		t.Fatalf("empty prompt should not require a server or write a file: %v", err)
	}
}

func TestWriteConversationInstructionsRejectsOversizedPrompt(t *testing.T) {
	var service Service
	err := service.WriteConversationInstructions(context.Background(), "conversation-id", strings.Repeat("x", maxAgentInstructionsSize+1))
	if err == nil || err.Error() != "agent character prompt is too large" {
		t.Fatalf("unexpected error: %v", err)
	}
}
