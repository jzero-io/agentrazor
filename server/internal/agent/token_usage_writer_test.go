package agent

import (
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestTokenUsageWriterDoesNotBlockAndPreservesOrder(t *testing.T) {
	release := make(chan struct{})
	var mu sync.Mutex
	var written []string
	w := newTokenUsageWriter(func(event TokenUsageEvent) {
		<-release
		mu.Lock()
		written = append(written, event.TurnID)
		mu.Unlock()
	})

	enqueued := make(chan struct{})
	go func() {
		w.Enqueue(TokenUsageEvent{TurnID: "first"})
		w.Enqueue(TokenUsageEvent{TurnID: "second"})
		close(enqueued)
	}()

	select {
	case <-enqueued:
	case <-time.After(time.Second):
		t.Fatal("enqueue blocked on token usage persistence")
	}

	close(release)
	w.Close()

	mu.Lock()
	defer mu.Unlock()
	if want := []string{"first", "second"}; !reflect.DeepEqual(written, want) {
		t.Fatalf("written events = %v, want %v", written, want)
	}
}
