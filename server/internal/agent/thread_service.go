package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/zeromicro/go-zero/core/logx"
)

var (
	ErrServiceStopped  = errors.New("agent service is stopped")
	ErrThreadArchived  = errors.New("agent thread is archived")
	ErrInvalidThreadID = errors.New("invalid Codex thread id")
	// ErrThreadNotFound 表示会话在业务库有记录，但 Codex thread 已不存在
	// （读不到也无法 resume，通常是孤儿数据）。
	ErrThreadNotFound    = errors.New("agent thread not found")
	ErrRuntimeRestarting = errors.New("agent runtime is restarting")
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
	DurationMs  *int64
	Error       string
}

type ThreadRun struct {
	ID        string
	ThreadID  string
	Prompt    string
	CreatedAt time.Time
}

type ActiveRun struct {
	ID        string
	ThreadID  string
	CreatedAt time.Time
}

type TokenUsageBreakdown struct {
	InputTokens           int64
	CachedInputTokens     int64
	CacheWriteInputTokens int64
	OutputTokens          int64
	ReasoningOutputTokens int64
	TotalTokens           int64
}

type TokenUsageEvent struct {
	ConversationID     string
	TurnID             string
	Last               TokenUsageBreakdown
	Total              TokenUsageBreakdown
	ModelContextWindow *int64
}

type TokenUsageRecorder func(context.Context, TokenUsageEvent) error

type RuntimeFactory func() (ThreadRuntime, error)

type RuntimeStatus struct {
	Running         bool
	Restarting      bool
	ActiveRunCount  int
	LastRestartTime time.Time
}

type activeRun struct {
	id        string
	createdAt time.Time
	cancel    context.CancelFunc
}

type ThreadService struct {
	runtime        ThreadRuntime
	runtimeFactory RuntimeFactory
	idleTimeout    time.Duration
	events         *EventHub

	mu              sync.Mutex
	runs            map[string]activeRun
	closed          bool
	restarting      bool
	lastRestartTime time.Time

	tokenUsageRecorder TokenUsageRecorder
}

func NewThreadService(runtime ThreadRuntime, factory RuntimeFactory) *ThreadService {
	return &ThreadService{
		runtime:         runtime,
		runtimeFactory:  factory,
		idleTimeout:     turnIdleTimeout,
		events:          NewEventHub(32, 256),
		runs:            make(map[string]activeRun),
		lastRestartTime: time.Now().UTC(),
	}
}

func (s *ThreadService) currentRuntime() (ThreadRuntime, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, ErrServiceStopped
	}
	if s.restarting {
		return nil, ErrRuntimeRestarting
	}
	if s.runtime == nil {
		return nil, ErrRuntimeClosed
	}
	return s.runtime, nil
}

func (s *ThreadService) SetTokenUsageRecorder(recorder TokenUsageRecorder) {
	s.mu.Lock()
	s.tokenUsageRecorder = recorder
	s.mu.Unlock()
}

func (s *ThreadService) Create(ctx context.Context, title string) (StoredThread, error) {
	runtime, err := s.currentRuntime()
	if err != nil {
		return StoredThread{}, err
	}
	thread, err := runtime.CreateStoredThread(ctx)
	if err != nil {
		return StoredThread{}, err
	}
	title = strings.TrimSpace(title)
	if title != "" {
		if err := runtime.SetThreadName(ctx, thread.ID, title); err != nil {
			s.deleteCreatedThread(thread.ID)
			return StoredThread{}, err
		}
		thread.Name = title
	}
	return thread, nil
}

func (s *ThreadService) List(ctx context.Context) ([]StoredThread, error) {
	runtime, err := s.currentRuntime()
	if err != nil {
		return nil, err
	}
	active, err := runtime.ListStoredThreads(ctx, false)
	if err != nil {
		return nil, err
	}
	archived, err := runtime.ListStoredThreads(ctx, true)
	if err != nil {
		return nil, err
	}
	return append(active, archived...), nil
}

func (s *ThreadService) Metadata(ctx context.Context, threadID string) (StoredThread, error) {
	if err := validateThreadID(threadID); err != nil {
		return StoredThread{}, err
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return StoredThread{}, err
	}
	return runtime.ReadStoredThread(ctx, threadID, false)
}

func (s *ThreadService) Get(ctx context.Context, threadID string) (StoredThread, error) {
	if err := validateThreadID(threadID); err != nil {
		return StoredThread{}, err
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return StoredThread{}, err
	}
	return runtime.ReadStoredThread(ctx, threadID, true)
}

func (s *ThreadService) ActiveRun(threadID string) (ActiveRun, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	run, ok := s.runs[threadID]
	if !ok {
		return ActiveRun{}, false
	}
	return ActiveRun{ID: run.id, ThreadID: threadID, CreatedAt: run.createdAt}, true
}

func (s *ThreadService) EventCursor(threadID string) int64 {
	if err := validateThreadID(threadID); err != nil {
		return 0
	}
	return s.events.Cursor(threadID)
}

func (s *ThreadService) SetName(ctx context.Context, threadID, title string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("conversation title is required")
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	if err := runtime.SetThreadName(ctx, threadID, title); err != nil {
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
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	return runtime.SetThreadPinned(ctx, threadID, pinned)
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
		runtime, err := s.currentRuntime()
		if err != nil {
			return err
		}
		return runtime.ArchiveStoredThread(ctx, threadID)
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	_, err = runtime.UnarchiveStoredThread(ctx, threadID)
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
	if s.restarting {
		s.mu.Unlock()
		return ThreadRun{}, ErrRuntimeRestarting
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
	s.runs[threadID] = activeRun{id: run.ID, createdAt: run.CreatedAt, cancel: cancel}
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
		"startedAt": run.CreatedAt,
	})
	emit := func(event map[string]any) {
		idleTimer.Reset(s.idleTimeout)
		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = "event"
		}
		if eventType == "thread.tokenUsage.updated" {
			s.recordTokenUsage(event)
		}
		// Codex process events are UI progress only. Keep them live-only so the
		// server does not retain intermediate display state between subscribers.
		s.events.Broadcast(run.ThreadID, run.ID, "codex."+eventType, event)
	}
	runtime, runtimeErr := s.currentRuntime()
	if runtimeErr != nil {
		s.events.Publish(run.ThreadID, run.ID, "run.failed", map[string]any{"error": runtimeErr.Error()})
		return
	}
	err := runtime.Resume(ctx, run.ThreadID, run.Prompt, emit)
	if err != nil {
		s.events.Publish(run.ThreadID, run.ID, "run.failed", map[string]any{"error": err.Error()})
		return
	}
	s.events.Publish(run.ThreadID, run.ID, "run.completed", nil)
}

func (s *ThreadService) Subscribe(threadID string, afterID int64) *Subscription {
	return s.events.Subscribe(threadID, afterID)
}

func (s *ThreadService) RuntimeStatus() RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return RuntimeStatus{
		Running:         !s.closed && s.runtime != nil,
		Restarting:      s.restarting,
		ActiveRunCount:  len(s.runs),
		LastRestartTime: s.lastRestartTime,
	}
}

func (s *ThreadService) RestartRuntime() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrServiceStopped
	}
	if s.restarting {
		s.mu.Unlock()
		return ErrRuntimeRestarting
	}
	if len(s.runs) > 0 {
		s.mu.Unlock()
		return ErrThreadTurnRunning
	}
	factory := s.runtimeFactory
	old := s.runtime
	if factory == nil {
		s.mu.Unlock()
		return errors.New("agent runtime factory is not configured")
	}
	s.restarting = true
	s.mu.Unlock()

	next, err := factory()

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.restarting = false
		return err
	}
	s.runtime = next
	s.lastRestartTime = time.Now().UTC()
	s.restarting = false
	if old != nil {
		go func() { _ = old.Close() }()
	}
	return nil
}

func (s *ThreadService) Cancel(threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return ErrServiceStopped
	}
	run, ok := s.runs[threadID]
	if ok {
		delete(s.runs, threadID)
	}
	s.mu.Unlock()
	if ok {
		run.cancel()
	}
	return nil
}

func (s *ThreadService) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	cancels := make([]context.CancelFunc, 0, len(s.runs))
	for _, run := range s.runs {
		cancels = append(cancels, run.cancel)
	}
	s.runs = make(map[string]activeRun)
	runtime := s.runtime
	s.runtime = nil
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	err := error(nil)
	if runtime != nil {
		err = runtime.Close()
	}
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
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	if err := runtime.DeleteThread(ctx, threadID); err != nil {
		return err
	}
	s.events.Release(threadID)
	return nil
}

func (s *ThreadService) recordTokenUsage(event map[string]any) {
	usage, ok := tokenUsageEventFromCodex(event)
	if !ok {
		return
	}
	s.mu.Lock()
	recorder := s.tokenUsageRecorder
	s.mu.Unlock()
	if recorder == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := recorder(ctx, usage); err != nil {
		logx.Errorf("record Codex token usage failed: %v", err)
	}
}

func (s *ThreadService) ValidateThread(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	if _, err := runtime.ReadStoredThread(ctx, threadID, false); err != nil {
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
	runtime, err := s.currentRuntime()
	if err != nil {
		return
	}
	_ = runtime.DeleteThread(ctx, threadID)
}
