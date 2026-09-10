package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

const conversationContextFile = "context.json"

type conversationContext struct {
	ConversationID string `json:"conversationId"`
}

func (r *appServer) createConversationHome(conversationID string) error {
	dir, err := r.conversationDir(conversationID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), appServerTimeout)
	defer cancel()
	if _, err := r.request(ctx, "fs/createDirectory", map[string]any{
		"path": dir, "recursive": true,
	}); err != nil {
		return fmt.Errorf("create conversation home: %w", err)
	}
	data, err := json.MarshalIndent(conversationContext{ConversationID: conversationID}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conversation context: %w", err)
	}
	data = append(data, '\n')
	if _, err := r.request(ctx, "fs/writeFile", map[string]any{
		"path":       filepath.Join(dir, conversationContextFile),
		"dataBase64": base64.StdEncoding.EncodeToString(data),
	}); err != nil {
		return fmt.Errorf("write conversation context: %w", err)
	}
	return nil
}

func (r *appServer) conversationDir(conversationID string) (string, error) {
	if err := validateThreadID(conversationID); err != nil {
		return "", err
	}
	if filepath.Base(conversationID) != conversationID || strings.ContainsAny(conversationID, `/\`) {
		return "", fmt.Errorf("%w: %q", errInvalidThreadID, conversationID)
	}
	return filepath.Join(r.workspaceHome, conversationID), nil
}

func (r *appServer) deleteConversationHome(conversationID string) error {
	dir, err := r.conversationDir(conversationID)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.request(ctx, "fs/remove", map[string]any{
		"path": dir, "recursive": true, "force": true,
	}); err != nil {
		return fmt.Errorf("delete conversation home: %w", err)
	}
	return nil
}
