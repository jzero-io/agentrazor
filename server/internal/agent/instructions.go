package agent

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

const (
	agentInstructionsFile    = "AGENTS.md"
	maxAgentInstructionsSize = 16000
)

// WriteConversationInstructions writes optional, conversation-scoped Codex
// instructions after the thread and its workspace have been created.
func (s *Service) WriteConversationInstructions(ctx context.Context, conversationID, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil
	}
	if len([]byte(prompt)) > maxAgentInstructionsSize {
		return errors.New("agent character prompt is too large")
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	dir, err := server.conversationDir(conversationID)
	if err != nil {
		return err
	}
	data := []byte(prompt + "\n")
	if _, err := server.request(ctx, "fs/writeFile", map[string]any{
		"path":       filepath.Join(dir, agentInstructionsFile),
		"dataBase64": base64.StdEncoding.EncodeToString(data),
	}); err != nil {
		return fmt.Errorf("write conversation AGENTS.md: %w", err)
	}
	return nil
}
