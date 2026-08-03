package agent

import (
	"sync"
	"time"
)

type StreamEvent struct {
	ID        int64     `json:"id"`
	Type      string    `json:"type"`
	SessionID string    `json:"sessionId"`
	RunID     string    `json:"runId,omitempty"`
	Data      any       `json:"data,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type Subscription struct {
	Events <-chan StreamEvent
	close  func()
	once   sync.Once
}

func (s *Subscription) Close() {
	if s == nil || s.close == nil {
		return
	}
	s.once.Do(s.close)
}

type eventStream struct {
	nextID      int64
	history     []StreamEvent
	subscribers map[uint64]chan StreamEvent
	nextSubID   uint64
}

type EventHub struct {
	mu           sync.Mutex
	streams      map[string]*eventStream
	historyLimit int
	bufferSize   int
}

func NewEventHub(historyLimit, bufferSize int) *EventHub {
	if historyLimit <= 0 {
		historyLimit = 1_000
	}
	if bufferSize <= 0 {
		bufferSize = 256
	}
	return &EventHub{
		streams:      make(map[string]*eventStream),
		historyLimit: historyLimit,
		bufferSize:   bufferSize,
	}
}

func (h *EventHub) Publish(sessionID, runID, eventType string, data any) StreamEvent {
	return h.publish(sessionID, runID, eventType, data, true)
}

// Broadcast pushes an event to live subscribers without caching it in history.
// Use for high-volume, transient events (e.g. streaming token deltas) that are
// not worth replaying on reconnect: the finalized message is delivered via a
// cached lifecycle event (run.completed) and a detail refresh.
func (h *EventHub) Broadcast(sessionID, runID, eventType string, data any) StreamEvent {
	return h.publish(sessionID, runID, eventType, data, false)
}

func (h *EventHub) publish(sessionID, runID, eventType string, data any, cache bool) StreamEvent {
	h.mu.Lock()
	defer h.mu.Unlock()

	stream := h.streamLocked(sessionID)
	stream.nextID++
	event := StreamEvent{
		ID:        stream.nextID,
		Type:      eventType,
		SessionID: sessionID,
		RunID:     runID,
		Data:      data,
		CreatedAt: time.Now().UTC(),
	}
	if cache {
		stream.history = append(stream.history, event)
		if len(stream.history) > h.historyLimit {
			stream.history = append([]StreamEvent(nil), stream.history[len(stream.history)-h.historyLimit:]...)
		}
	}
	for _, subscriber := range stream.subscribers {
		select {
		case subscriber <- event:
		default:
			// Cached events remain in history and can be recovered with Last-Event-ID.
		}
	}
	return event
}

func (h *EventHub) Subscribe(sessionID string, afterID int64) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()

	stream := h.streamLocked(sessionID)
	stream.nextSubID++
	subID := stream.nextSubID
	replay := make([]StreamEvent, 0, len(stream.history))
	for _, event := range stream.history {
		if event.ID > afterID {
			replay = append(replay, event)
		}
	}
	capacity := h.bufferSize
	if len(replay) > capacity {
		capacity = len(replay)
	}
	channel := make(chan StreamEvent, capacity)
	for _, event := range replay {
		channel <- event
	}
	stream.subscribers[subID] = channel

	return &Subscription{
		Events: channel,
		close: func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if current, ok := h.streams[sessionID]; ok {
				delete(current.subscribers, subID)
			}
		},
	}
}

func (h *EventHub) streamLocked(sessionID string) *eventStream {
	stream, ok := h.streams[sessionID]
	if !ok {
		stream = &eventStream{subscribers: make(map[uint64]chan StreamEvent)}
		h.streams[sessionID] = stream
	}
	return stream
}
