package agent

import "testing"

func TestEventHubReplaysEventsAfterLastEventID(t *testing.T) {
	hub := NewEventHub(10, 2)
	for i := 0; i < 5; i++ {
		hub.Publish("session-1", "run-1", "test.event", i)
	}

	subscription := hub.Subscribe("session-1", 2)
	defer subscription.Close()
	for wantID := int64(3); wantID <= 5; wantID++ {
		event := <-subscription.Events
		if event.ID != wantID {
			t.Fatalf("event id = %d, want %d", event.ID, wantID)
		}
	}
	select {
	case event := <-subscription.Events:
		t.Fatalf("unexpected replay event: %#v", event)
	default:
	}
}

func TestEventHubKeepsBoundedHistory(t *testing.T) {
	hub := NewEventHub(3, 1)
	for i := 0; i < 5; i++ {
		hub.Publish("session-1", "", "test.event", i)
	}

	subscription := hub.Subscribe("session-1", 0)
	defer subscription.Close()
	for wantID := int64(3); wantID <= 5; wantID++ {
		event := <-subscription.Events
		if event.ID != wantID {
			t.Fatalf("event id = %d, want %d", event.ID, wantID)
		}
	}
}
