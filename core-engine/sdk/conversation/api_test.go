package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConversationAPIs(t *testing.T) {
	const apiKey = "ar-api-test"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, apiKey, r.Header.Get(apiKeyHeader))
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/conversation":
			writeJSON(t, w, success(map[string]any{"conversations": []any{map[string]any{"id": "c1", "title": "one"}}}))
		case "GET /api/v1/conversation/stats":
			writeJSON(t, w, success(map[string]any{"totalConversations": 2, "totalTokens": 42, "tokenUsageAvailable": true}))
		case "POST /api/v1/conversation":
			var request CreateConversationRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, "g1", request.GroupID)
			writeJSON(t, w, success(map[string]any{"id": "c1", "groupId": "g1"}))
		case "POST /api/v1/conversation/c1/messages":
			var request SendMessageRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.Equal(t, "hello", request.Content)
			writeJSON(t, w, success(map[string]any{"id": "r1", "createdAt": "2026-09-03T09:02:21Z"}))
		case "GET /api/v1/conversation/c1":
			writeJSON(t, w, success(map[string]any{"conversation": map[string]any{"id": "c1"}, "eventCursor": 9, "turns": []any{}}))
		case "GET /api/v1/conversation/c1/metadata":
			writeJSON(t, w, success(map[string]any{"id": "c1", "title": "metadata"}))
		case "PATCH /api/v1/conversation/c1":
			var request UpdateConversationRequest
			require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
			require.NotNil(t, request.Title)
			require.Equal(t, "renamed", *request.Title)
			writeJSON(t, w, success(map[string]any{"id": "c1", "title": "renamed"}))
		case "DELETE /api/v1/conversation/c1", "POST /api/v1/conversation/c1/turn/cancel":
			writeJSON(t, w, success(map[string]any{}))
		case "GET /api/v1/conversation/c1/workspace/files":
			writeJSON(t, w, success(map[string]any{"entries": []any{map[string]any{"name": "main.go", "path": "main.go", "type": "file", "size": 12}}}))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: apiKey})
	require.NoError(t, err)
	ctx := context.Background()

	conversations, err := client.ListConversations(ctx)
	require.NoError(t, err)
	require.Equal(t, "c1", conversations[0].ID)
	stats, err := client.ConversationStats(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 42, stats.TotalTokens)
	created, err := client.CreateConversation(ctx, CreateConversationRequest{GroupID: " g1 "})
	require.NoError(t, err)
	require.Equal(t, "c1", created.ID)
	sent, err := client.SendMessage(ctx, "c1", SendMessageRequest{Content: " hello "})
	require.NoError(t, err)
	require.Equal(t, "r1", sent.ID)
	detail, err := client.GetConversation(ctx, "c1")
	require.NoError(t, err)
	require.EqualValues(t, 9, detail.EventCursor)
	metadata, err := client.GetConversationMetadata(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, "metadata", metadata.Title)
	title := "renamed"
	updated, err := client.UpdateConversation(ctx, "c1", UpdateConversationRequest{Title: &title})
	require.NoError(t, err)
	require.Equal(t, title, updated.Title)
	files, err := client.ListWorkspaceFiles(ctx, "c1")
	require.NoError(t, err)
	require.Equal(t, "main.go", files[0].Name)
	require.NoError(t, client.CancelTurn(ctx, "c1"))
	require.NoError(t, client.DeleteConversation(ctx, "c1"))
}

func TestGroupAPIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/conversation-groups":
			writeJSON(t, w, success(map[string]any{"groups": []any{map[string]any{"id": "g1", "name": "group"}}}))
		case "POST /api/v1/conversation-groups":
			assertNameBody(t, r, "new group")
			writeJSON(t, w, success(map[string]any{"id": "g2", "name": "new group"}))
		case "PATCH /api/v1/conversation-groups/g1":
			assertNameBody(t, r, "renamed")
			writeJSON(t, w, success(map[string]any{"id": "g1", "name": "renamed"}))
		case "DELETE /api/v1/conversation-groups/g1",
			"POST /api/v1/conversation-groups/g1/archive-conversations",
			"POST /api/v1/conversation-groups/g1/delete-archived-conversations":
			writeJSON(t, w, success(map[string]any{}))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "ar-test"})
	require.NoError(t, err)
	ctx := context.Background()
	groups, err := client.ListGroups(ctx)
	require.NoError(t, err)
	require.Equal(t, "g1", groups[0].ID)
	created, err := client.CreateGroup(ctx, " new group ")
	require.NoError(t, err)
	require.Equal(t, "g2", created.ID)
	updated, err := client.UpdateGroup(ctx, "g1", "renamed")
	require.NoError(t, err)
	require.Equal(t, "renamed", updated.Name)
	require.NoError(t, client.ArchiveGroupConversations(ctx, "g1"))
	require.NoError(t, client.DeleteGroupArchivedConversations(ctx, "g1"))
	require.NoError(t, client.DeleteGroup(ctx, "g1"))
}

func TestStreamEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "7", r.Header.Get("Last-Event-ID"))
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		heartbeat, err := json.Marshal(eventsResponse{Event: "stream.heartbeat", Data: "{}"})
		require.NoError(t, err)
		eventData, err := json.Marshal(Event{ID: 8, Type: "item.completed", ConversationID: "c1", TurnID: "t1"})
		require.NoError(t, err)
		event, err := json.Marshal(eventsResponse{Event: "item.completed", Data: string(eventData)})
		require.NoError(t, err)
		_, err = fmt.Fprintf(w, "data: %s\n\ndata: %s\n\n", heartbeat, event)
		require.NoError(t, err)
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "ar-test"})
	require.NoError(t, err)
	errStop := errors.New("stop")
	var received []Event
	err = client.StreamEvents(context.Background(), "c1", 7, func(event Event) error {
		received = append(received, event)
		return errStop
	})
	require.ErrorIs(t, err, errStop)
	require.Len(t, received, 1)
	require.EqualValues(t, 8, received[0].ID)
}

func TestAPIInputValidation(t *testing.T) {
	client, err := NewClient(Config{BaseURL: "http://localhost", APIKey: "ar-test"})
	require.NoError(t, err)
	_, err = client.GetConversation(context.Background(), " ")
	require.Error(t, err)
	_, err = client.CreateGroup(context.Background(), " ")
	require.Error(t, err)
	_, err = client.CreateGroup(context.Background(), "12345678901234567890123456789012345678901")
	require.Error(t, err)
	err = client.StreamEvents(context.Background(), "c1", -1, func(Event) error { return nil })
	require.Error(t, err)
}

func success(data any) envelope[any] {
	return envelope[any]{Code: http.StatusOK, Msg: "success", Data: data}
}

func assertNameBody(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	var request map[string]string
	require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
	require.Equal(t, expected, request["name"])
}
