package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const conversationContextFile = "context.json"

type conversationContext struct {
	ConversationID string `json:"conversationId"`
}

func (r *CodexAppServerRuntime) createConversationHome(conversationID string) error {
	dir, err := r.conversationDir(conversationID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create conversation home: %w", err)
	}
	data, err := json.MarshalIndent(conversationContext{ConversationID: conversationID}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conversation context: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(dir, conversationContextFile)
	temporaryPath := path + ".tmp"
	if err := os.WriteFile(temporaryPath, data, 0o600); err != nil {
		return fmt.Errorf("write conversation context: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)
		return fmt.Errorf("publish conversation context: %w", err)
	}
	return nil
}

func (r *CodexAppServerRuntime) conversationDir(conversationID string) (string, error) {
	if err := validateThreadID(conversationID); err != nil {
		return "", err
	}
	if filepath.Base(conversationID) != conversationID || strings.ContainsAny(conversationID, `/\`) {
		return "", fmt.Errorf("%w: %q", ErrInvalidThreadID, conversationID)
	}
	return filepath.Join(r.options.AgentrazorHome, conversationID), nil
}

func (r *CodexAppServerRuntime) DeleteConversationHome(conversationID string) error {
	dir, err := r.conversationDir(conversationID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("delete conversation home: %w", err)
	}
	return nil
}
