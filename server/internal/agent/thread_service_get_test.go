package agent

import (
	"context"
	"errors"
	"testing"
)

type getThreadRuntime struct {
	ThreadRuntime
	archived  []StoredThread
	stored    StoredThread
	readCalls int
}

func (r *getThreadRuntime) ListStoredThreads(_ context.Context, archived bool) ([]StoredThread, error) {
	if archived {
		return r.archived, nil
	}
	return nil, nil
}

func (r *getThreadRuntime) ReadStoredThread(context.Context, string, bool) (StoredThread, error) {
	r.readCalls++
	return r.stored, nil
}
func (*getThreadRuntime) Close() error { return nil }

func TestGetRejectsThreadWithoutTurns(t *testing.T) {
	id := "01a07ef7-df8a-7072-9cd5-c4e4af546be4"
	runtime := &getThreadRuntime{stored: StoredThread{ID: id}}
	service := NewThreadService(runtime, nil)
	defer func() { _ = service.Close() }()

	_, err := service.Get(context.Background(), id)
	if !errors.Is(err, ErrThreadNotFound) {
		t.Fatalf("Get() error = %v, want %v", err, ErrThreadNotFound)
	}
	if runtime.readCalls != 1 {
		t.Fatalf("ReadStoredThread() calls = %d, want 1", runtime.readCalls)
	}
}

func TestGetReadsThreadWithTurns(t *testing.T) {
	id := "01a07ef7-df8a-7072-9cd5-c4e4af546be4"
	runtime := &getThreadRuntime{stored: StoredThread{ID: id, Turns: []StoredTurn{{ID: "turn"}}}}
	service := NewThreadService(runtime, nil)
	defer func() { _ = service.Close() }()

	got, err := service.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Fatalf("Get() id = %q, want %q", got.ID, id)
	}
	if runtime.readCalls != 1 {
		t.Fatalf("ReadStoredThread() calls = %d, want 1", runtime.readCalls)
	}
}

func TestGetReadsThreadListedAsArchived(t *testing.T) {
	id := "01a07ef7-df8a-7072-9cd5-c4e4af546be4"
	runtime := &getThreadRuntime{
		archived: []StoredThread{{ID: id, Archived: true}},
		stored:   StoredThread{ID: id, Turns: []StoredTurn{{ID: "turn"}}},
	}
	service := NewThreadService(runtime, nil)
	defer func() { _ = service.Close() }()

	got, err := service.Get(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Archived {
		t.Fatal("Get() returned an active thread, want archived")
	}
	if runtime.readCalls != 1 {
		t.Fatalf("ReadStoredThread() calls = %d, want 1", runtime.readCalls)
	}
}
