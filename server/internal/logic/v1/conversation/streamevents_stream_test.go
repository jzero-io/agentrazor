package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	_ "modernc.org/sqlite"

	agentdomain "github.com/jzero-io/agentrazor/server/internal/agent"
	"github.com/jzero-io/agentrazor/server/internal/model"
	"github.com/jzero-io/agentrazor/server/internal/svc"
	types "github.com/jzero-io/agentrazor/server/internal/types/v1/conversation"
)

type streamRuntimeStub struct {
	thread agentdomain.StoredThread
}

func (s *streamRuntimeStub) CreateStoredThread(context.Context) (agentdomain.StoredThread, error) {
	return s.thread, nil
}

func (s *streamRuntimeStub) ListStoredThreads(context.Context, bool) ([]agentdomain.StoredThread, error) {
	return []agentdomain.StoredThread{s.thread}, nil
}

func (s *streamRuntimeStub) ReadStoredThread(context.Context, string, bool) (agentdomain.StoredThread, error) {
	return s.thread, nil
}

func (s *streamRuntimeStub) SetThreadName(context.Context, string, string) error { return nil }

func (s *streamRuntimeStub) SetThreadPinned(context.Context, string, bool) error { return nil }

func (s *streamRuntimeStub) ArchiveStoredThread(context.Context, string) error { return nil }

func (s *streamRuntimeStub) DeleteThread(context.Context, string) error { return nil }

func (s *streamRuntimeStub) UnarchiveStoredThread(context.Context, string) (agentdomain.StoredThread, error) {
	return s.thread, nil
}

func (s *streamRuntimeStub) Resume(_ context.Context, id, _ string, _ agentdomain.EventHandler) (agentdomain.RuntimeResult, error) {
	return agentdomain.RuntimeResult{ExternalSessionID: id, Output: "done"}, nil
}

func (s *streamRuntimeStub) Close() error { return nil }

func TestStreamEventsReplaysFromCursor(t *testing.T) {
	runtime := &streamRuntimeStub{thread: agentdomain.StoredThread{
		ID: "conversation-1", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}}
	service := agentdomain.NewThreadService(runtime, time.Second)
	defer service.Close()
	run, err := service.Send("conversation-1", "trace")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "uuid", "test-user"))
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err = db.Exec(`CREATE TABLE conversation (id TEXT PRIMARY KEY, user_uuid TEXT NOT NULL, group_uuid TEXT NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO conversation(id, user_uuid) VALUES ('conversation-1', 'test-user')`); err != nil {
		t.Fatal(err)
	}
	logic := NewStreamEvents(ctx, &svc.ServiceContext{AgentThreads: service, Model: model.NewModel(sqlx.NewSqlConnFromDB(db))})
	client := make(chan *types.EventsResponse, 8)
	errs := make(chan error, 1)
	go func() {
		errs <- logic.StreamEvents(&types.EventsRequest{
			ConversationId:  "conversation-1",
			LastEventIdForm: "0",
		}, client)
	}()

	select {
	case response := <-client:
		if response.Id != 1 || response.Event != "run.queued" {
			t.Fatalf("unexpected first event: %#v", response)
		}
		var event agentdomain.StreamEvent
		if err := json.Unmarshal([]byte(response.Data), &event); err != nil {
			t.Fatal(err)
		}
		if event.SessionID != run.ThreadID || event.ID != response.Id {
			t.Fatalf("unexpected event envelope: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replayed event")
	}

	cancel()
	select {
	case err := <-errs:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not stop after cancellation")
	}
}

func TestParseLastEventID(t *testing.T) {
	id, err := parseLastEventID(&types.EventsRequest{LastEventId: "12", LastEventIdForm: "7"})
	if err != nil || id != 12 {
		t.Fatalf("header cursor = %d, err = %v", id, err)
	}
	if _, err := parseLastEventID(&types.EventsRequest{LastEventIdForm: "-1"}); err == nil {
		t.Fatal("negative cursor should be rejected")
	}
}
