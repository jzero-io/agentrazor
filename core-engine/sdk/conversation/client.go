// Package conversation provides a small client for talking to an AgentRazor
// server through its conversation API.
package conversation

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	apiKeyHeader  = "X-API-Key"
	maxEventBytes = 8 << 20
)

// Config configures a conversation client.
type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// Client sends conversations to an AgentRazor server.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// Request is one conversational turn. Leave ConversationID empty to start a
// new conversation, then reuse the returned ID for subsequent turns.
type Request struct {
	ConversationID string
	GroupID        string
	Content        string
}

// Response contains the stable final answer for a conversational turn.
type Response struct {
	ConversationID string
	Answer         string
}

// APIError represents an error returned by the AgentRazor HTTP API.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf("conversation API error: %s (code %d)", e.Message, e.Code)
	}
	return fmt.Sprintf("conversation API error: %s (HTTP %d)", e.Message, e.StatusCode)
}

// RunError reports a failed AI run.
type RunError struct {
	Message string
}

func (e *RunError) Error() string {
	return "conversation run failed: " + e.Message
}

// NewClient creates a conversation client.
func NewClient(config Config) (*Client, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("conversation SDK: BaseURL must be an absolute URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("conversation SDK: BaseURL must use http or https")
	}

	apiKey := strings.TrimSpace(config.APIKey)
	if !strings.HasPrefix(apiKey, "ar-") {
		return nil, errors.New("conversation SDK: APIKey must start with ar-")
	}

	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: baseURL, apiKey: apiKey, httpClient: httpClient}, nil
}

// Chat sends one message, waits for its run to finish, and returns the final
// answer. The supplied context controls the entire request and wait lifecycle.
func (c *Client) Chat(ctx context.Context, request Request) (*Response, error) {
	content := strings.TrimSpace(request.Content)
	if content == "" {
		return nil, errors.New("conversation SDK: message content is required")
	}

	sent, err := c.SendMessage(ctx, SendMessageRequest{
		ConversationID: strings.TrimSpace(request.ConversationID),
		GroupID:        strings.TrimSpace(request.GroupID),
		Content:        content,
	})
	if err != nil {
		return nil, err
	}
	if err := c.waitRun(ctx, sent.ConversationID, sent.Run.ID); err != nil {
		return nil, err
	}

	detail, err := c.GetConversation(ctx, sent.ConversationID)
	if err != nil {
		return nil, err
	}
	answer := finalAnswer(detail.Turns)
	if answer == "" {
		return nil, errors.New("conversation SDK: run completed without a final answer")
	}
	return &Response{ConversationID: sent.ConversationID, Answer: answer}, nil
}

type envelope[T any] struct {
	Code int    `json:"code"`
	Data T      `json:"data"`
	Msg  string `json:"msg"`
}

func (c *Client) doJSON(ctx context.Context, method, path string, body any, output any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("conversation SDK: encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("conversation SDK: create request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set(apiKeyHeader, c.apiKey)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("conversation SDK: request: %w", err)
	}
	defer response.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("conversation SDK: read response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return &APIError{StatusCode: response.StatusCode, Message: responseMessage(payload, response.Status)}
	}

	var wrapped envelope[json.RawMessage]
	if err := json.Unmarshal(payload, &wrapped); err != nil {
		return fmt.Errorf("conversation SDK: decode response: %w", err)
	}
	if wrapped.Code != http.StatusOK {
		return &APIError{StatusCode: response.StatusCode, Code: wrapped.Code, Message: wrapped.Msg}
	}
	if output == nil {
		return nil
	}
	if err := json.Unmarshal(wrapped.Data, output); err != nil {
		return fmt.Errorf("conversation SDK: decode response data: %w", err)
	}
	return nil
}

func finalAnswer(turns []Turn) string {
	var fallback string
	for turnIndex := len(turns) - 1; turnIndex >= 0; turnIndex-- {
		items := turns[turnIndex].Items
		for itemIndex := len(items) - 1; itemIndex >= 0; itemIndex-- {
			current := items[itemIndex]
			itemType, _ := current["type"].(string)
			text, _ := current["text"].(string)
			if itemType != "agentMessage" || strings.TrimSpace(text) == "" {
				continue
			}
			if fallback == "" {
				fallback = text
			}
			phase, _ := current["phase"].(string)
			if phase == "final_answer" {
				return text
			}
		}
	}
	return fallback
}

func responseMessage(payload []byte, fallback string) string {
	var wrapped struct {
		Msg string `json:"msg"`
	}
	if json.Unmarshal(payload, &wrapped) == nil && strings.TrimSpace(wrapped.Msg) != "" {
		return wrapped.Msg
	}
	if message := strings.TrimSpace(string(payload)); message != "" {
		return message
	}
	return fallback
}
