package agent

import "encoding/json"

// Conversation is the metadata for one conversation.
type Conversation struct {
	ID               string  `json:"id"`
	Title            string  `json:"title"`
	Status           string  `json:"status"`
	PinnedAt         *string `json:"pinnedAt,omitempty"`
	ArchivedAt       *string `json:"archivedAt,omitempty"`
	GroupID          *string `json:"groupId,omitempty"`
	Running          bool    `json:"running"`
	RunningStartedAt *string `json:"runningStartedAt,omitempty"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

// StartedTurn identifies the Codex turn accepted by turn/start.
type StartedTurn struct {
	ID        string `json:"id"`
	StartedAt string `json:"startedAt"`
}

// Item is an item emitted during a turn. Its fields vary by item type.
type Item map[string]any

// Turn contains the persisted items and status of one conversation turn.
type Turn struct {
	ID          string  `json:"id"`
	Status      string  `json:"status"`
	StartedAt   *string `json:"startedAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
	DurationMS  *int64  `json:"durationMs,omitempty"`
	Error       string  `json:"error,omitempty"`
	Items       []Item  `json:"items"`
}

// Detail is the full persisted state of a conversation.
type Detail struct {
	Conversation   Conversation `json:"conversation"`
	StreamPosition string       `json:"streamPosition"`
	Turns          []Turn       `json:"turns"`
}

// Metadata is the lightweight metadata returned for one conversation.
type Metadata struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	WorkspaceDir string `json:"workspaceDir"`
	UpdatedAt    string `json:"updatedAt"`
}

// Stats summarizes the authenticated account's conversations.
type Stats struct {
	TotalConversations    int64 `json:"totalConversations"`
	ActiveConversations   int64 `json:"activeConversations"`
	RunningConversations  int64 `json:"runningConversations"`
	ArchivedConversations int64 `json:"archivedConversations"`
	TotalTokens           int64 `json:"totalTokens"`
	TokenUsageAvailable   bool  `json:"tokenUsageAvailable"`
}

// CreateConversationRequest contains optional metadata for a new conversation.
type CreateConversationRequest struct {
	GroupID string `json:"groupId,omitempty"`
}

// SendMessageRequest starts a turn in an existing conversation.
type SendMessageRequest struct {
	Content     string                 `json:"content,omitempty"`
	Attachments []MessageAttachmentRef `json:"attachments,omitempty"`
}

// MessageAttachmentRef identifies an uploaded attachment by workspace path.
type MessageAttachmentRef struct {
	Path string `json:"path"`
}

// UpdateConversationRequest contains the conversation fields to change.
// A non-nil empty GroupID removes the conversation from its group.
type UpdateConversationRequest struct {
	Title    *string `json:"title,omitempty"`
	Pinned   *bool   `json:"pinned,omitempty"`
	Archived *bool   `json:"archived,omitempty"`
	GroupID  *string `json:"groupId,omitempty"`
}

// Event is one server-sent conversation event.
type Event struct {
	Type           string          `json:"type"`
	ConversationID string          `json:"conversationId"`
	TurnID         string          `json:"turnId,omitempty"`
	StreamPosition string          `json:"streamPosition,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	CreatedAt      string          `json:"createdAt"`
}

// EventHandler receives events until the context is canceled, the stream
// closes, or the handler returns an error.
type EventHandler func(Event) error

// Group is a user-defined conversation group.
type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}
