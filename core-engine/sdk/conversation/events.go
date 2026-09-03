package conversation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

type eventsResponse struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// StreamEvents subscribes to live conversation events. Events with IDs up to
// afterID are skipped by the server. The method blocks until the context is
// canceled, the connection closes, or the handler returns an error.
func (c *Client) StreamEvents(ctx context.Context, conversationID string, afterID int64, handler EventHandler) error {
	if handler == nil {
		return errors.New("conversation SDK: event handler is required")
	}
	if afterID < 0 {
		return errors.New("conversation SDK: after event ID must not be negative")
	}
	return c.streamEvents(ctx, conversationID, afterID, nil, func(event Event) (bool, error) {
		return false, handler(event)
	})
}

func (c *Client) streamEvents(ctx context.Context, conversationID string, afterID int64, onReady func(), handler func(Event) (bool, error)) error {
	path, err := conversationPath(conversationID, "/events")
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("conversation SDK: create event request: %w", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	request.Header.Set(apiKeyHeader, c.apiKey)
	if afterID > 0 {
		request.Header.Set("Last-Event-ID", strconv.FormatInt(afterID, 10))
	}

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
		finished, err := handleEventFrame(dataLines, onReady, handler)
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

func handleEventFrame(dataLines []string, onReady func(), handler func(Event) (bool, error)) (bool, error) {
	if len(dataLines) == 0 {
		return false, nil
	}
	var outer eventsResponse
	if err := json.Unmarshal([]byte(strings.Join(dataLines, "\n")), &outer); err != nil {
		return false, fmt.Errorf("conversation SDK: decode event: %w", err)
	}
	if outer.Event == "stream.ready" {
		if onReady != nil {
			onReady()
		}
		return false, nil
	}
	if outer.Event == "stream.heartbeat" {
		return false, nil
	}
	var event Event
	if err := json.Unmarshal([]byte(outer.Data), &event); err != nil {
		return false, fmt.Errorf("conversation SDK: decode event data: %w", err)
	}
	return handler(event)
}
