package agent

import (
	"sync"
	"time"
)

type StreamEvent struct {
	ID             int64     `json:"id"`
	Type           string    `json:"type"`
	ConversationID string    `json:"conversationId"`
	RunID          string    `json:"runId,omitempty"`
	Data           any       `json:"data,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
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
	lastActive  time.Time
}

type EventHub struct {
	mu           sync.Mutex
	streams      map[string]*eventStream
	historyLimit int
	bufferSize   int
	idleTTL      time.Duration
	stop         chan struct{}
	stopOnce     sync.Once
	closed       bool
}

const (
	eventStreamIdleTTL       = 30 * time.Minute
	eventStreamSweepInterval = 5 * time.Minute
)

func NewEventHub(historyLimit, bufferSize int) *EventHub {
	if historyLimit <= 0 {
		historyLimit = 1_000
	}
	if bufferSize <= 0 {
		bufferSize = 256
	}
	hub := &EventHub{
		streams:      make(map[string]*eventStream),
		historyLimit: historyLimit,
		bufferSize:   bufferSize,
		idleTTL:      eventStreamIdleTTL,
		stop:         make(chan struct{}),
	}
	go hub.sweepIdleStreams()
	return hub
}

func (h *EventHub) Publish(conversationID, runID, eventType string, data any) StreamEvent {
	return h.publish(conversationID, runID, eventType, data, true)
}

// Broadcast pushes an event to live subscribers without caching it in history.
// Use for high-volume, transient events (e.g. streaming token deltas) that are
// not worth replaying on reconnect: the finalized message is delivered via a
// cached lifecycle event (run.completed) and a detail refresh.
func (h *EventHub) Broadcast(conversationID, runID, eventType string, data any) StreamEvent {
	return h.publish(conversationID, runID, eventType, data, false)
}

func (h *EventHub) publish(conversationID, runID, eventType string, data any, cache bool) StreamEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return StreamEvent{}
	}

	stream := h.streamLocked(conversationID)
	stream.lastActive = time.Now().UTC()
	stream.nextID++
	event := StreamEvent{
		ID:             stream.nextID,
		Type:           eventType,
		ConversationID: conversationID,
		RunID:          runID,
		Data:           data,
		CreatedAt:      time.Now().UTC(),
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

func (h *EventHub) Cursor(conversationID string) int64 {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return 0
	}
	stream, ok := h.streams[conversationID]
	if !ok {
		return 0
	}
	return stream.nextID
}

func (h *EventHub) Subscribe(conversationID string, afterID int64) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return closedSubscription()
	}

	stream := h.streamLocked(conversationID)
	stream.lastActive = time.Now().UTC()
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
			if current, ok := h.streams[conversationID]; ok && current == stream {
				if subscriber, ok := current.subscribers[subID]; ok {
					delete(current.subscribers, subID)
					close(subscriber)
				}
				current.lastActive = time.Now().UTC()
			}
		},
	}
}

func closedSubscription() *Subscription {
	channel := make(chan StreamEvent)
	close(channel)
	return &Subscription{Events: channel}
}

func closeEventStreamSubscribers(stream *eventStream) {
	for id, subscriber := range stream.subscribers {
		delete(stream.subscribers, id)
		close(subscriber)
	}
}

// Release drops replay history for a conversation that has been permanently
// deleted and closes live subscribers so SSE handlers can exit promptly.
func (h *EventHub) Release(conversationID string) {
	h.mu.Lock()
	if stream, ok := h.streams[conversationID]; ok {
		closeEventStreamSubscribers(stream)
		delete(h.streams, conversationID)
	}
	h.mu.Unlock()
}

func (h *EventHub) Close() {
	h.stopOnce.Do(func() { close(h.stop) })
	h.mu.Lock()
	h.closed = true
	for _, stream := range h.streams {
		closeEventStreamSubscribers(stream)
	}
	h.streams = make(map[string]*eventStream)
	h.mu.Unlock()
}

func (h *EventHub) sweepIdleStreams() {
	ticker := time.NewTicker(eventStreamSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case now := <-ticker.C:
			h.mu.Lock()
			for conversationID, stream := range h.streams {
				if len(stream.subscribers) == 0 && now.Sub(stream.lastActive) >= h.idleTTL {
					delete(h.streams, conversationID)
				}
			}
			h.mu.Unlock()
		case <-h.stop:
			return
		}
	}
}

func (h *EventHub) streamLocked(conversationID string) *eventStream {
	stream, ok := h.streams[conversationID]
	if !ok {
		stream = &eventStream{
			subscribers: make(map[uint64]chan StreamEvent),
			lastActive:  time.Now().UTC(),
		}
		h.streams[conversationID] = stream
	}
	return stream
}
