package agent

import (
	"context"
	"errors"
	"testing"
)

const cancelTestThreadID = "01a0ae32-4e9d-7580-826f-6f88266db685"

func TestCancelWaitsForTurnCompletion(t *testing.T) {
	cancelled := make(chan struct{})
	turn := &activeTurn{
		cancel: func() { close(cancelled) },
		done:   make(chan error, 1),
	}
	service := &Service{turns: map[string]*activeTurn{cancelTestThreadID: turn}}

	result := make(chan error, 1)
	go func() {
		result <- service.Cancel(context.Background(), cancelTestThreadID)
	}()

	<-cancelled
	select {
	case err := <-result:
		t.Fatalf("Cancel returned before the turn completed: %v", err)
	default:
	}

	turn.done <- context.Canceled
	close(turn.done)
	if err := <-result; err != nil {
		t.Fatalf("Cancel returned an error for an interrupted turn: %v", err)
	}
}

func TestCancelReturnsContextErrorWhileWaiting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	service := &Service{turns: map[string]*activeTurn{
		cancelTestThreadID: {
			cancel: func() {},
			done:   make(chan error, 1),
		},
	}}

	if err := service.Cancel(ctx, cancelTestThreadID); !errors.Is(err, context.Canceled) {
		t.Fatalf("Cancel error = %v, want context.Canceled", err)
	}
}

func TestCancelWithoutActiveTurnIsIdempotent(t *testing.T) {
	service := &Service{turns: make(map[string]*activeTurn)}
	if err := service.Cancel(context.Background(), cancelTestThreadID); err != nil {
		t.Fatalf("Cancel returned an error without an active turn: %v", err)
	}
}

func TestInterruptedNotificationIsEmittedBeforeCompletion(t *testing.T) {
	emitted := make(chan map[string]any, 1)
	turn := &appServerTurn{
		threadID: cancelTestThreadID,
		emit: func(event map[string]any, _ string) {
			emitted <- event
		},
		done: make(chan turnOutcome, 1),
	}

	turn.handleNotification("turn/completed", map[string]any{
		"threadId": cancelTestThreadID,
		"turn": map[string]any{
			"id":     "turn-1",
			"status": "interrupted",
		},
	}, "position-1")

	event := <-emitted
	if event["type"] != "turn.completed" {
		t.Fatalf("event type = %v, want turn.completed", event["type"])
	}
	outcome := <-turn.done
	if !errors.Is(outcome.err, context.Canceled) {
		t.Fatalf("turn outcome = %v, want context.Canceled", outcome.err)
	}
}
