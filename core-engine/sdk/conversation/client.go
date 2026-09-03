// Package conversation provides a small client for talking to an AgentRazor
// server through its conversation API.
package conversation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
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

	sent, err := c.send(ctx, sendRequest{
		ConversationID: strings.TrimSpace(request.ConversationID),
		Content:        content,
	})
	if err != nil {
		return nil, err
	}
	if err := c.waitRun(ctx, sent.ConversationID, sent.Run.ID); err != nil {
		return nil, err
	}

	detail, err := c.detail(ctx, sent.ConversationID)
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

type sendRequest struct {
	ConversationID string `json:"conversationId,omitempty"`
	Content        string `json:"content"`
}

type sendResponse struct {
	ConversationID string `json:"conversationId"`
	Run            struct {
		ID string `json:"id"`
	} `json:"run"`
}

type detailResponse struct {
	Turns []turn `json:"turns"`
}

type turn struct {
	Items []item `json:"items"`
}

type item struct {
	Type  string  `json:"type"`
	Text  string  `json:"text"`
	Phase *string `json:"phase"`
}

type eventsResponse struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

type streamEvent struct {
	Type  string          `json:"type"`
	RunID string          `json:"runId"`
	Data  json.RawMessage `json:"data"`
}

func (c *Client) send(ctx context.Context, request sendRequest) (*sendResponse, error) {
	var response sendResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/conversations", request, &response); err != nil {
		return nil, err
	}
	if response.ConversationID == "" || response.Run.ID == "" {
		return nil, errors.New("conversation SDK: server returned an incomplete run")
	}
	return &response, nil
}

func (c *Client) detail(ctx context.Context, conversationID string) (*detailResponse, error) {
	var response detailResponse
	path := "/api/v1/conversations/" + url.PathEscape(conversationID)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
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
	if err := json.Unmarshal(wrapped.Data, output); err != nil {
		return fmt.Errorf("conversation SDK: decode response data: %w", err)
	}
	return nil
}

func (c *Client) waitRun(ctx context.Context, conversationID, runID string) error {
	path := "/api/v1/conversations/" + url.PathEscape(conversationID) + "/events"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("conversation SDK: create event request: %w", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set(apiKeyHeader, c.apiKey)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("conversation SDK: subscribe to events: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		payload, _ := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		return &APIError{StatusCode: response.StatusCode, Message: responseMessage(payload, response.Status)}
	}
	contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	if contentType != "text/event-stream" {
		return fmt.Errorf("conversation SDK: expected text/event-stream, got %q", contentType)
	}

	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 64<<10), maxEventBytes)
	var dataLines []string
	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line != "" {
			if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
			continue
		}
		finished, err := consumeEvent(dataLines, runID)
		dataLines = dataLines[:0]
		if err != nil {
			return err
		}
		if finished {
			return nil
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("conversation SDK: read event stream: %w", err)
	}
	return io.ErrUnexpectedEOF
}

func consumeEvent(dataLines []string, runID string) (bool, error) {
	if len(dataLines) == 0 {
		return false, nil
	}
	var outer eventsResponse
	if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &outer); err != nil {
		return false, fmt.Errorf("conversation SDK: decode event: %w", err)
	}
	if outer.Event == "stream.heartbeat" {
		return false, nil
	}
	var event streamEvent
	if err := json.Unmarshal([]byte(outer.Data), &event); err != nil {
		return false, fmt.Errorf("conversation SDK: decode event data: %w", err)
	}
	if event.RunID != runID {
		return false, nil
	}
	switch event.Type {
	case "run.completed":
		return true, nil
	case "run.failed":
		var failure struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(event.Data, &failure)
		if failure.Error == "" {
			failure.Error = "unknown error"
		}
		return false, &RunError{Message: failure.Error}
	default:
		return false, nil
	}
}

func finalAnswer(turns []turn) string {
	var fallback string
	for turnIndex := len(turns) - 1; turnIndex >= 0; turnIndex-- {
		items := turns[turnIndex].Items
		for itemIndex := len(items) - 1; itemIndex >= 0; itemIndex-- {
			current := items[itemIndex]
			if current.Type != "agentMessage" || strings.TrimSpace(current.Text) == "" {
				continue
			}
			if fallback == "" {
				fallback = current.Text
			}
			if current.Phase != nil && *current.Phase == "final_answer" {
				return current.Text
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
