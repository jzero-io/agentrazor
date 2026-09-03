package agent

import "sync"

// tokenUsageWriter keeps database work off the app-server stdout reader while
// preserving notification order and draining accepted events during shutdown.
type tokenUsageWriter struct {
	mu     sync.Mutex
	ready  *sync.Cond
	queue  []TokenUsageEvent
	write  func(TokenUsageEvent)
	closed bool
	done   chan struct{}
}

func newTokenUsageWriter(write func(TokenUsageEvent)) *tokenUsageWriter {
	w := &tokenUsageWriter{
		write: write,
		done:  make(chan struct{}),
	}
	w.ready = sync.NewCond(&w.mu)
	go w.run()
	return w
}

func (w *tokenUsageWriter) Enqueue(event TokenUsageEvent) {
	w.mu.Lock()
	if !w.closed {
		w.queue = append(w.queue, event)
		w.ready.Signal()
	}
	w.mu.Unlock()
}

func (w *tokenUsageWriter) Close() {
	w.mu.Lock()
	if !w.closed {
		w.closed = true
		w.ready.Broadcast()
	}
	w.mu.Unlock()
	<-w.done
}

func (w *tokenUsageWriter) run() {
	defer close(w.done)
	for {
		w.mu.Lock()
		for len(w.queue) == 0 && !w.closed {
			w.ready.Wait()
		}
		if len(w.queue) == 0 {
			w.mu.Unlock()
			return
		}
		event := w.queue[0]
		w.queue[0] = TokenUsageEvent{}
		w.queue = w.queue[1:]
		w.mu.Unlock()

		w.write(event)
	}
}
