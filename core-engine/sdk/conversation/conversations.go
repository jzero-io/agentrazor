package conversation

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// ListConversations lists conversations owned by the authenticated account.
func (c *Client) ListConversations(ctx context.Context) ([]Conversation, error) {
	var response struct {
		Conversations []Conversation `json:"conversations"`
	}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/conversation", nil, &response); err != nil {
		return nil, err
	}
	return response.Conversations, nil
}

// ConversationStats returns aggregate conversation and token statistics.
func (c *Client) ConversationStats(ctx context.Context) (*Stats, error) {
	var response Stats
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/conversation/stats", nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// CreateConversation creates an empty conversation without starting a turn.
func (c *Client) CreateConversation(ctx context.Context, request CreateConversationRequest) (*Conversation, error) {
	request.GroupID = strings.TrimSpace(request.GroupID)
	var response Conversation
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/conversation", request, &response); err != nil {
		return nil, err
	}
	if response.ID == "" {
		return nil, errors.New("conversation SDK: server returned an incomplete conversation")
	}
	return &response, nil
}

// SendMessage starts a Codex turn in an existing conversation.
func (c *Client) SendMessage(ctx context.Context, conversationID string, request SendMessageRequest) (*StartedTurn, error) {
	request.Content = strings.TrimSpace(request.Content)
	if request.Content == "" {
		return nil, errors.New("conversation SDK: message content is required")
	}
	path, err := conversationPath(conversationID, "/messages")
	if err != nil {
		return nil, err
	}
	var response StartedTurn
	if err := c.doJSON(ctx, http.MethodPost, path, request, &response); err != nil {
		return nil, err
	}
	if response.ID == "" {
		return nil, errors.New("conversation SDK: server returned an incomplete turn")
	}
	return &response, nil
}

// GetConversation returns the full persisted conversation detail.
func (c *Client) GetConversation(ctx context.Context, conversationID string) (*Detail, error) {
	path, err := conversationPath(conversationID, "")
	if err != nil {
		return nil, err
	}
	var response Detail
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetConversationMetadata returns conversation metadata without its turns.
func (c *Client) GetConversationMetadata(ctx context.Context, conversationID string) (*Conversation, error) {
	path, err := conversationPath(conversationID, "/metadata")
	if err != nil {
		return nil, err
	}
	var response Conversation
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateConversation updates conversation metadata.
func (c *Client) UpdateConversation(ctx context.Context, conversationID string, request UpdateConversationRequest) (*Conversation, error) {
	path, err := conversationPath(conversationID, "")
	if err != nil {
		return nil, err
	}
	var response Conversation
	if err := c.doJSON(ctx, http.MethodPatch, path, request, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// DeleteConversation permanently deletes a conversation.
func (c *Client) DeleteConversation(ctx context.Context, conversationID string) error {
	path, err := conversationPath(conversationID, "")
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil)
}

// CancelTurn requests cancellation of the active turn.
func (c *Client) CancelTurn(ctx context.Context, conversationID string) error {
	path, err := conversationPath(conversationID, "/turn/cancel")
	if err != nil {
		return err
	}
	return c.doJSON(ctx, http.MethodPost, path, nil, nil)
}

// ListWorkspaceFiles lists files in a conversation workspace.
func (c *Client) ListWorkspaceFiles(ctx context.Context, conversationID string) ([]WorkspaceEntry, error) {
	path, err := conversationPath(conversationID, "/workspace/files")
	if err != nil {
		return nil, err
	}
	var response struct {
		Entries []WorkspaceEntry `json:"entries"`
	}
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return response.Entries, nil
}

func conversationPath(conversationID, suffix string) (string, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" {
		return "", errors.New("conversation SDK: conversation ID is required")
	}
	return "/api/v1/conversation/" + url.PathEscape(conversationID) + suffix, nil
}
