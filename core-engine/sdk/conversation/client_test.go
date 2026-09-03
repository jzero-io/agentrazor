package conversation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChat(t *testing.T) {
	const (
		apiKey         = "ar-test-key"
		conversationID = "conversation-1"
		runID          = "run-1"
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, apiKey, r.Header.Get(apiKeyHeader))
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/conversations":
			var request SendMessageRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, "你好", request.Content)
			writeJSON(t, w, envelope[any]{Code: 200, Msg: "success", Data: map[string]any{
				"conversationId": conversationID,
				"run":            map[string]any{"id": runID},
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/conversations/"+conversationID+"/events":
			require.Equal(t, "text/event-stream", r.Header.Get("Accept"))
			w.Header().Set("Content-Type", "text/event-stream")
			eventData, err := json.Marshal(map[string]any{
				"type": "run.completed", "conversationId": conversationID, "runId": runID,
			})
			require.NoError(t, err)
			outer, err := json.Marshal(eventsResponse{Event: "run.completed", Data: string(eventData)})
			require.NoError(t, err)
			_, err = fmt.Fprintf(w, "data: %s\n\n", outer)
			require.NoError(t, err)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/conversations/"+conversationID:
			writeJSON(t, w, envelope[any]{Code: 200, Msg: "success", Data: map[string]any{
				"turns": []any{map[string]any{"items": []any{
					map[string]any{"type": "agentMessage", "phase": "final_answer", "text": "你好，有什么可以帮你？"},
				}}},
			}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: apiKey})
	require.NoError(t, err)
	response, err := client.Chat(context.Background(), Request{Content: " 你好 "})
	require.NoError(t, err)
	require.Equal(t, conversationID, response.ConversationID)
	require.Equal(t, "你好，有什么可以帮你？", response.Answer)
}

func TestChatContinuesConversation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "stop after request validation", http.StatusBadRequest)
			return
		}
		var request SendMessageRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		require.Equal(t, "existing-conversation", request.ConversationID)
		writeJSON(t, w, envelope[any]{Code: 500, Msg: "expected stop", Data: nil})
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "ar-test"})
	require.NoError(t, err)
	_, err = client.Chat(context.Background(), Request{
		ConversationID: "existing-conversation",
		Content:        "继续",
	})
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr)
	require.Equal(t, 500, apiErr.Code)
}

func TestNewClientValidation(t *testing.T) {
	_, err := NewClient(Config{BaseURL: "localhost:8080", APIKey: "ar-test"})
	require.Error(t, err)
	_, err = NewClient(Config{BaseURL: "http://localhost:8080", APIKey: "invalid"})
	require.Error(t, err)
}

func TestConsumeRunFailure(t *testing.T) {
	payload, err := json.Marshal(Event{
		Type:  "run.failed",
		RunID: "run-1",
		Data:  json.RawMessage(`{"error":"model unavailable"}`),
	})
	require.NoError(t, err)
	outer, err := json.Marshal(eventsResponse{Event: "run.failed", Data: string(payload)})
	require.NoError(t, err)

	finished, err := handleEventFrame([]string{string(outer)}, func(event Event) (bool, error) {
		if event.RunID != "run-1" {
			return false, nil
		}
		var failure struct {
			Error string `json:"error"`
		}
		require.NoError(t, json.Unmarshal(event.Data, &failure))
		return false, &RunError{Message: failure.Error}
	})
	require.False(t, finished)
	var runErr *RunError
	require.ErrorAs(t, err, &runErr)
	require.Equal(t, "model unavailable", runErr.Message)
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
