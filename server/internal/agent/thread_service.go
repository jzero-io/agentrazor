package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrServiceStopped  = errors.New("agent service is stopped")
	ErrThreadArchived  = errors.New("agent thread is archived")
	ErrInvalidThreadID = errors.New("invalid Codex thread id")
)

const turnIdleTimeout = 10 * time.Minute

func newRunID() string {
	return "run_" + uuid.NewString()
}

type StoredThread struct {
	ID        string
	Name      string
	Preview   string
	IsPinned  bool
	Archived  bool
	CreatedAt time.Time
	UpdatedAt time.Time
	Turns     []StoredTurn
}

type StoredTurn struct {
	ID          string
	Status      string
	Items       []map[string]any
	CreatedAt   time.Time
	CompletedAt *time.Time
	Error       string
}

type ThreadRun struct {
	ID        string
	ThreadID  string
	Prompt    string
	CreatedAt time.Time
}

type activeRun struct {
	id     string
	cancel context.CancelFunc
}

type ThreadService struct {
	runtime     ThreadRuntime
	idleTimeout time.Duration
	events      *EventHub

	mu     sync.Mutex
	runs   map[string]activeRun
	closed bool
}

func NewThreadService(runtime ThreadRuntime) *ThreadService {
	return &ThreadService{
		runtime:     runtime,
		idleTimeout: turnIdleTimeout,
		events:      NewEventHub(1_000, 256),
		runs:        make(map[string]activeRun),
	}
}

func (s *ThreadService) Create(ctx context.Context, title string) (StoredThread, error) {
	thread, err := s.runtime.CreateStoredThread(ctx)
	if err != nil {
		return StoredThread{}, err
	}
	title = strings.TrimSpace(title)
	if title != "" {
		if err := s.runtime.SetThreadName(ctx, thread.ID, title); err != nil {
			s.deleteCreatedThread(thread.ID)
			return StoredThread{}, err
		}
		thread.Name = title
	}
	return thread, nil
}

func (s *ThreadService) List(ctx context.Context) ([]StoredThread, error) {
	active, err := s.runtime.ListStoredThreads(ctx, false)
	if err != nil {
		return nil, err
	}
	archived, err := s.runtime.ListStoredThreads(ctx, true)
	if err != nil {
		return nil, err
	}
	return append(active, archived...), nil
}

func (s *ThreadService) Get(ctx context.Context, threadID string) (StoredThread, error) {
	if err := validateThreadID(threadID); err != nil {
		return StoredThread{}, err
	}
	return s.runtime.ReadStoredThread(ctx, threadID, true)
}

func (s *ThreadService) SetName(ctx context.Context, threadID, title string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("conversation title is required")
	}
	if err := s.runtime.SetThreadName(ctx, threadID, title); err != nil {
		// thread/name/set needs the rollout, which is moved away when the
		// thread is archived, so it fails with -32600. Surface that as a clear
		// "archived" error instead of leaking the raw "no rollout found".
		var rpcErr *RPCError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32600 {
			return ErrThreadArchived
		}
		return err
	}
	return nil
}

func (s *ThreadService) SetPinned(ctx context.Context, threadID string, pinned bool) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	return s.runtime.SetThreadPinned(ctx, threadID, pinned)
}

func (s *ThreadService) SetArchived(ctx context.Context, threadID string, archived bool) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	if archived {
		s.mu.Lock()
		_, running := s.runs[threadID]
		s.mu.Unlock()
		if running {
			return ErrThreadTurnRunning
		}
		return s.runtime.ArchiveStoredThread(ctx, threadID)
	}
	_, err := s.runtime.UnarchiveStoredThread(ctx, threadID)
	return err
}

func (s *ThreadService) Send(threadID, prompt string) (ThreadRun, error) {
	if err := validateThreadID(threadID); err != nil {
		return ThreadRun{}, err
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return ThreadRun{}, errors.New("message content is required")
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ThreadRun{}, ErrServiceStopped
	}
	if _, ok := s.runs[threadID]; ok {
		s.mu.Unlock()
		return ThreadRun{}, ErrThreadTurnRunning
	}
	run := ThreadRun{
		ID:        newRunID(),
		ThreadID:  threadID,
		Prompt:    prompt,
		CreatedAt: time.Now().UTC(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.runs[threadID] = activeRun{id: run.ID, cancel: cancel}
	s.mu.Unlock()

	go s.execute(ctx, cancel, run)
	return run, nil
}

func (s *ThreadService) execute(ctx context.Context, cancel context.CancelFunc, run ThreadRun) {
	defer cancel()
	idleTimer := time.AfterFunc(s.idleTimeout, cancel)
	defer idleTimer.Stop()
	defer func() {
		s.mu.Lock()
		if current, ok := s.runs[run.ThreadID]; ok && current.id == run.ID {
			delete(s.runs, run.ThreadID)
		}
		s.mu.Unlock()
	}()

	s.events.Publish(run.ThreadID, run.ID, "run.started", map[string]any{
		"startedAt": time.Now().UTC(),
	})
	emit := func(event map[string]any) {
		idleTimer.Reset(s.idleTimeout)
		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = "event"
		}
		publishedName := "codex." + eventType
		// item/agentMessage/delta is high-volume, transient streaming data: push
		// to live subscribers only, without caching it in history. The finalized
		// message arrives via run.completed and a detail refresh.
		if eventType == "item.agentMessage.delta" {
			s.events.Broadcast(run.ThreadID, run.ID, publishedName, event)
			return
		}
		s.events.Publish(run.ThreadID, run.ID, publishedName, event)
	}
	err := s.runtime.Resume(ctx, run.ThreadID, run.Prompt, emit)
	if err != nil {
		s.events.Publish(run.ThreadID, run.ID, "run.failed", map[string]any{"error": err.Error()})
		return
	}
	s.events.Publish(run.ThreadID, run.ID, "run.completed", nil)
}

func (s *ThreadService) Subscribe(threadID string, afterID int64) *Subscription {
	return s.events.Subscribe(threadID, afterID)
}

func (s *ThreadService) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	for _, run := range s.runs {
		run.cancel()
	}
	s.mu.Unlock()
	err := s.runtime.Close()
	s.events.Close()
	return err
}

func (s *ThreadService) Delete(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	s.mu.Lock()
	_, running := s.runs[threadID]
	s.mu.Unlock()
	if running {
		return ErrThreadTurnRunning
	}
	if err := s.runtime.DeleteThread(ctx, threadID); err != nil {
		return err
	}
	s.events.Release(threadID)
	return nil
}

func (s *ThreadService) ValidateThread(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	if _, err := s.runtime.ReadStoredThread(ctx, threadID, false); err != nil {
		return fmt.Errorf("read Codex thread: %w", err)
	}
	return nil
}

func validateThreadID(threadID string) error {
	value := strings.TrimSpace(threadID)
	switch strings.ToLower(value) {
	case "", "null", "undefined":
		return fmt.Errorf("%w: %q", ErrInvalidThreadID, threadID)
	default:
		return nil
	}
}

func (s *ThreadService) deleteCreatedThread(threadID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = s.runtime.DeleteThread(ctx, threadID)
}
