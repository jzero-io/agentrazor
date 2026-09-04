package agent

import (
	"sync"
	"time"
)

type StreamEvent struct {
	Type           string    `json:"type"`
	ConversationID string    `json:"conversationId"`
	TurnID         string    `json:"turnId,omitempty"`
	StreamPosition string    `json:"streamPosition,omitempty"`
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

type eventSubscriber struct {
	mu        sync.Mutex
	queue     []StreamEvent
	events    chan StreamEvent
	wake      chan struct{}
	done      chan struct{}
	closeOnce sync.Once
	closed    bool
}

func newEventSubscriber() *eventSubscriber {
	subscriber := &eventSubscriber{
		events: make(chan StreamEvent),
		wake:   make(chan struct{}, 1),
		done:   make(chan struct{}),
	}
	go subscriber.run()
	return subscriber
}

func (s *eventSubscriber) enqueue(event StreamEvent) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.queue = append(s.queue, event)
	s.mu.Unlock()
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *eventSubscriber) run() {
	defer close(s.events)
	for {
		s.mu.Lock()
		var event StreamEvent
		queued := len(s.queue) > 0
		if queued {
			event = s.queue[0]
			s.queue[0] = StreamEvent{}
			s.queue = s.queue[1:]
			if len(s.queue) == 0 {
				s.queue = nil
			}
		}
		s.mu.Unlock()

		if queued {
			select {
			case s.events <- event:
			case <-s.done:
				return
			}
			continue
		}

		select {
		case <-s.wake:
		case <-s.done:
			return
		}
	}
}

func (s *eventSubscriber) close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.queue = nil
		s.mu.Unlock()
		close(s.done)
	})
}

type EventHub struct {
	mu      sync.Mutex
	streams map[string]map[*eventSubscriber]struct{}
	closed  bool
}

func NewEventHub() *EventHub {
	return &EventHub{streams: make(map[string]map[*eventSubscriber]struct{})}
}

func (h *EventHub) Publish(conversationID, turnID, eventType, streamPosition string, data any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	subscribers := h.streams[conversationID]
	if subscribers == nil {
		return
	}

	event := StreamEvent{
		Type:           eventType,
		ConversationID: conversationID,
		TurnID:         turnID,
		StreamPosition: streamPosition,
		Data:           data,
		CreatedAt:      time.Now().UTC(),
	}
	for subscriber := range subscribers {
		subscriber.enqueue(event)
	}
}

func (h *EventHub) Subscribe(conversationID string) *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return closedSubscription()
	}

	subscribers := h.streams[conversationID]
	if subscribers == nil {
		subscribers = make(map[*eventSubscriber]struct{})
		h.streams[conversationID] = subscribers
	}
	subscriber := newEventSubscriber()
	subscribers[subscriber] = struct{}{}

	return &Subscription{
		Events: subscriber.events,
		close: func() {
			h.mu.Lock()
			defer h.mu.Unlock()
			if current := h.streams[conversationID]; current != nil {
				if _, exists := current[subscriber]; exists {
					delete(current, subscriber)
					subscriber.close()
				}
				if len(current) == 0 {
					delete(h.streams, conversationID)
				}
			}
		},
	}
}

func closedSubscription() *Subscription {
	channel := make(chan StreamEvent)
	close(channel)
	return &Subscription{Events: channel}
}

func closeEventStreamSubscribers(subscribers map[*eventSubscriber]struct{}) {
	for subscriber := range subscribers {
		delete(subscribers, subscriber)
		subscriber.close()
	}
}

// Release closes live subscribers for a conversation that has been deleted.
func (h *EventHub) Release(conversationID string) {
	h.mu.Lock()
	if subscribers, ok := h.streams[conversationID]; ok {
		closeEventStreamSubscribers(subscribers)
		delete(h.streams, conversationID)
	}
	h.mu.Unlock()
}

func (h *EventHub) Close() {
	h.mu.Lock()
	h.closed = true
	for _, subscribers := range h.streams {
		closeEventStreamSubscribers(subscribers)
	}
	h.streams = make(map[string]map[*eventSubscriber]struct{})
	h.mu.Unlock()
}
