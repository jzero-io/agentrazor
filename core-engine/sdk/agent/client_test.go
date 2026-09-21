package agent

import (
	"context"
	"errors"
	"testing"
)

func TestWaitForTurnReturnsTargetFinalAnswerFromSSE(t *testing.T) {
	events := make(chan Event, 6)
	streamErrors := make(chan error, 1)
	events <- eventWithData("item.completed", "other-turn", `{"params":{"item":{"type":"agentMessage","phase":"final_answer","text":"stale answer"}}}`)
	events <- eventWithData("turn.completed", "other-turn", `{"params":{"turn":{"status":"completed"}}}`)
	events <- eventWithData("item.completed", "target-turn", `{"params":{"item":{"type":"agentMessage","phase":"commentary","text":"internal note"}}}`)
	events <- eventWithData("item.completed", "target-turn", `{"params":{"item":{"type":"agentMessage","phase":"final_answer","text":"customer answer"}}}`)
	events <- eventWithData("turn.completed", "target-turn", `{"params":{"turn":{"status":"completed"}}}`)

	answer, err := waitForTurn(context.Background(), "target-turn", events, streamErrors)
	if err != nil {
		t.Fatalf("waitForTurn() error: %v", err)
	}
	if answer != "customer answer" {
		t.Fatalf("waitForTurn() = %q, want customer answer", answer)
	}
}

func TestWaitForTurnAcceptsFinalAnswerEmbeddedInCompletedTurn(t *testing.T) {
	events := make(chan Event, 1)
	streamErrors := make(chan error, 1)
	events <- eventWithData("turn.completed", "target-turn", `{"params":{"turn":{"status":"completed","items":[{"type":"agentMessage","phase":"final_answer","text":"embedded answer"}]}}}`)

	answer, err := waitForTurn(context.Background(), "target-turn", events, streamErrors)
	if err != nil {
		t.Fatalf("waitForTurn() error: %v", err)
	}
	if answer != "embedded answer" {
		t.Fatalf("waitForTurn() = %q, want embedded answer", answer)
	}
}

func TestWaitForTurnRejectsCompletedTurnWithoutFinalAnswer(t *testing.T) {
	events := make(chan Event, 2)
	streamErrors := make(chan error, 1)
	events <- eventWithData("item.completed", "target-turn", `{"params":{"item":{"type":"agentMessage","phase":"commentary","text":"internal note"}}}`)
	events <- eventWithData("turn.completed", "target-turn", `{"params":{"turn":{"status":"completed"}}}`)

	if _, err := waitForTurn(context.Background(), "target-turn", events, streamErrors); err == nil {
		t.Fatal("waitForTurn() accepted a completed turn without phase=final_answer")
	}
}

func TestWaitForTurnReturnsTurnFailure(t *testing.T) {
	events := make(chan Event, 1)
	streamErrors := make(chan error, 1)
	events <- eventWithData("turn.completed", "target-turn", `{"params":{"turn":{"status":"failed","error":{"message":"model failed"}}}}`)

	_, err := waitForTurn(context.Background(), "target-turn", events, streamErrors)
	var turnErr *TurnError
	if !errors.As(err, &turnErr) || turnErr.Message != "model failed" {
		t.Fatalf("waitForTurn() error = %#v", err)
	}
}

func eventWithData(eventType, turnID, data string) Event {
	return Event{Type: eventType, TurnID: turnID, Data: []byte(data)}
}
