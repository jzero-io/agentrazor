package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

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

type StoredThread struct {
	ID             string
	Name           string
	Preview        string
	IsPinned       bool
	Archived       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StreamPosition string
	Turns          []StoredTurn
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

type ActiveTurn struct {
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
	ActiveTurnCount int
	LastRestartTime time.Time
}

type activeTurn struct {
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
	turns           map[string]*activeTurn
	closed          bool
	restarting      bool
	lastRestartTime time.Time

	tokenUsageRecorder TokenUsageRecorder
	tokenUsageWriter   *tokenUsageWriter
}

func NewThreadService(runtime ThreadRuntime, factory RuntimeFactory) *ThreadService {
	service := &ThreadService{
		runtime:         runtime,
		runtimeFactory:  factory,
		idleTimeout:     turnIdleTimeout,
		events:          NewEventHub(),
		turns:           make(map[string]*activeTurn),
		lastRestartTime: time.Now().UTC(),
	}
	service.tokenUsageWriter = newTokenUsageWriter(service.persistTokenUsage)
	return service
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

func (s *ThreadService) Create(ctx context.Context) (StoredThread, error) {
	runtime, err := s.currentRuntime()
	if err != nil {
		return StoredThread{}, err
	}
	return runtime.CreateStoredThread(ctx)
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

func (s *ThreadService) ActiveTurn(threadID string) (ActiveTurn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	turn, ok := s.turns[threadID]
	if !ok {
		return ActiveTurn{}, false
	}
	return ActiveTurn{ID: turn.id, ThreadID: threadID, CreatedAt: turn.createdAt}, true
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
		_, running := s.turns[threadID]
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

func (s *ThreadService) Send(threadID, prompt string) (StartedTurn, error) {
	if err := validateThreadID(threadID); err != nil {
		return StartedTurn{}, err
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return StartedTurn{}, errors.New("message content is required")
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return StartedTurn{}, ErrServiceStopped
	}
	if s.restarting {
		s.mu.Unlock()
		return StartedTurn{}, ErrRuntimeRestarting
	}
	if s.runtime == nil {
		s.mu.Unlock()
		return StartedTurn{}, ErrRuntimeClosed
	}
	if _, ok := s.turns[threadID]; ok {
		s.mu.Unlock()
		return StartedTurn{}, ErrThreadTurnRunning
	}
	runtime := s.runtime
	ctx, cancel := context.WithCancel(context.Background())
	active := &activeTurn{createdAt: time.Now().UTC(), cancel: cancel}
	s.turns[threadID] = active
	s.mu.Unlock()

	idleTimer := time.AfterFunc(s.idleTimeout, cancel)
	emit := func(event map[string]any, streamPosition string) {
		idleTimer.Reset(s.idleTimeout)
		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = "event"
		}
		if eventType == "thread.tokenUsage.updated" {
			s.recordTokenUsage(event)
		}
		turnID := s.resolveActiveTurnID(threadID, active, codexEventTurnID(event))
		s.events.Publish(threadID, turnID, eventType, streamPosition, event)
	}

	started, err := runtime.StartTurn(ctx, threadID, prompt, emit)
	if err != nil {
		idleTimer.Stop()
		cancel()
		s.removeActiveTurn(threadID, active)
		return StartedTurn{}, err
	}
	s.mu.Lock()
	active.id = started.ID
	active.createdAt = started.StartedAt
	s.mu.Unlock()

	go func() {
		defer cancel()
		defer idleTimer.Stop()
		defer s.removeActiveTurn(threadID, active)
		<-started.Done
	}()
	return started, nil
}

func (s *ThreadService) removeActiveTurn(threadID string, active *activeTurn) {
	s.mu.Lock()
	if s.turns[threadID] == active {
		delete(s.turns, threadID)
	}
	s.mu.Unlock()
}

func (s *ThreadService) resolveActiveTurnID(threadID string, active *activeTurn, eventTurnID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.turns[threadID] != active {
		return eventTurnID
	}
	if eventTurnID != "" {
		active.id = eventTurnID
	}
	return active.id
}

func codexEventTurnID(event map[string]any) string {
	params, _ := event["params"].(map[string]any)
	if params == nil {
		return ""
	}
	if turnID := stringValue(params["turnId"]); turnID != "" {
		return turnID
	}
	turn, _ := params["turn"].(map[string]any)
	return stringValue(turn["id"])
}

func (s *ThreadService) Subscribe(threadID string) *Subscription {
	return s.events.Subscribe(threadID)
}

func (s *ThreadService) RuntimeStatus() RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return RuntimeStatus{
		Running:         !s.closed && s.runtime != nil,
		Restarting:      s.restarting,
		ActiveTurnCount: len(s.turns),
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
	if len(s.turns) > 0 {
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
	turn, ok := s.turns[threadID]
	if ok {
		delete(s.turns, threadID)
	}
	s.mu.Unlock()
	if ok {
		turn.cancel()
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
	cancels := make([]context.CancelFunc, 0, len(s.turns))
	for _, run := range s.turns {
		cancels = append(cancels, run.cancel)
	}
	s.turns = make(map[string]*activeTurn)
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
	s.tokenUsageWriter.Close()
	s.events.Close()
	return err
}

func (s *ThreadService) Delete(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	s.mu.Lock()
	_, running := s.turns[threadID]
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
	if err := runtime.DeleteConversationHome(threadID); err != nil {
		return err
	}
	s.events.Release(threadID)
	return nil
}

func (s *ThreadService) DeleteConversationHome(threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	runtime, err := s.currentRuntime()
	if err != nil {
		return err
	}
	return runtime.DeleteConversationHome(threadID)
}

func (s *ThreadService) recordTokenUsage(event map[string]any) {
	usage, ok := tokenUsageEventFromCodex(event)
	if !ok {
		return
	}
	s.tokenUsageWriter.Enqueue(usage)
}

func (s *ThreadService) persistTokenUsage(usage TokenUsageEvent) {
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
