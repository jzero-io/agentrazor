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
	errServiceStopped  = errors.New("agent service is stopped")
	ErrThreadArchived  = errors.New("agent thread is archived")
	errInvalidThreadID = errors.New("invalid Codex thread id")
	// ErrThreadNotFound 表示会话在业务库有记录，但 Codex thread 已不存在
	// （读不到也无法 resume，通常是孤儿数据）。
	ErrThreadNotFound = errors.New("agent thread not found")
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

type activeTurn struct {
	id        string
	createdAt time.Time
	cancel    context.CancelFunc
}

type Service struct {
	server    *appServer
	codexHome string
	workspace string
	events    *eventHub

	mu         sync.Mutex
	settingsMu sync.RWMutex
	turns      map[string]*activeTurn
	closed     bool

	tokenUsageRecorder func(context.Context, TokenUsageEvent) error
	tokenUsageWriter   *tokenUsageWriter
}

func NewService() (*Service, error) {
	server, err := newAppServer()
	if err != nil {
		return nil, err
	}
	service := &Service{
		server:    server,
		codexHome: server.codexHome,
		workspace: server.workspaceHome,
		events:    newEventHub(),
		turns:     make(map[string]*activeTurn),
	}
	service.tokenUsageWriter = newTokenUsageWriter(service.persistTokenUsage)
	return service, nil
}

func (s *Service) currentServer() (*appServer, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, errServiceStopped
	}
	server := s.server
	if server != nil && server.available() {
		s.mu.Unlock()
		return server, nil
	}

	reconnected, err := newAppServer()
	if err != nil {
		s.mu.Unlock()
		return nil, fmt.Errorf("reconnect Codex app-server: %w", err)
	}
	s.server = reconnected
	s.codexHome = reconnected.codexHome
	s.workspace = reconnected.workspaceHome
	s.mu.Unlock()

	if server != nil {
		_ = server.close()
	}
	return reconnected, nil
}

func (s *Service) call(ctx context.Context, method string, params any) (map[string]any, error) {
	server, err := s.currentServer()
	if err != nil {
		return nil, err
	}
	return server.request(ctx, method, params)
}

func (s *Service) SetTokenUsageRecorder(recorder func(context.Context, TokenUsageEvent) error) {
	s.mu.Lock()
	s.tokenUsageRecorder = recorder
	s.mu.Unlock()
}

func (s *Service) Create(ctx context.Context) (StoredThread, error) {
	server, err := s.currentServer()
	if err != nil {
		return StoredThread{}, err
	}
	return server.createThread(ctx)
}

func (s *Service) List(ctx context.Context) ([]StoredThread, error) {
	server, err := s.currentServer()
	if err != nil {
		return nil, err
	}
	active, err := server.listThreads(ctx, false)
	if err != nil {
		return nil, err
	}
	archived, err := server.listThreads(ctx, true)
	if err != nil {
		return nil, err
	}
	return append(active, archived...), nil
}

func (s *Service) Metadata(ctx context.Context, threadID string) (StoredThread, error) {
	if err := validateThreadID(threadID); err != nil {
		return StoredThread{}, err
	}
	server, err := s.currentServer()
	if err != nil {
		return StoredThread{}, err
	}
	return server.readThread(ctx, threadID, false)
}

func (s *Service) Get(ctx context.Context, threadID string) (StoredThread, error) {
	if err := validateThreadID(threadID); err != nil {
		return StoredThread{}, err
	}
	server, err := s.currentServer()
	if err != nil {
		return StoredThread{}, err
	}
	stored, err := server.readThread(ctx, threadID, true)
	if err != nil {
		return StoredThread{}, err
	}
	if len(stored.Turns) == 0 {
		return StoredThread{}, ErrThreadNotFound
	}
	archived, err := server.listThreads(ctx, true)
	if err != nil {
		return StoredThread{}, err
	}
	for _, thread := range archived {
		if thread.ID == threadID {
			stored.Archived = true
			break
		}
	}
	return stored, nil
}

func (s *Service) ActiveTurn(threadID string) (ActiveTurn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	turn, ok := s.turns[threadID]
	if !ok {
		return ActiveTurn{}, false
	}
	return ActiveTurn{ID: turn.id, ThreadID: threadID, CreatedAt: turn.createdAt}, true
}

func (s *Service) SetName(ctx context.Context, threadID, title string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("conversation title is required")
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	if err := server.setThreadName(ctx, threadID, title); err != nil {
		// thread/name/set needs the rollout, which is moved away when the
		// thread is archived, so it fails with -32600. Surface that as a clear
		// "archived" error instead of leaking the raw "no rollout found".
		var rpcErr *rpcCallError
		if errors.As(err, &rpcErr) && rpcErr.Code == -32600 {
			return ErrThreadArchived
		}
		return err
	}
	return nil
}

func (s *Service) SetPinned(ctx context.Context, threadID string, pinned bool) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	return server.setThreadPinned(ctx, threadID, pinned)
}

func (s *Service) SetArchived(ctx context.Context, threadID string, archived bool) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	if archived {
		s.mu.Lock()
		_, running := s.turns[threadID]
		s.mu.Unlock()
		if running {
			return errTurnRunning
		}
		server, err := s.currentServer()
		if err != nil {
			return err
		}
		return server.archiveThread(ctx, threadID)
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	_, err = server.unarchiveThread(ctx, threadID)
	return err
}

func (s *Service) Send(threadID, prompt string) (StartedTurn, error) {
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
		return StartedTurn{}, errServiceStopped
	}
	if s.server == nil {
		s.mu.Unlock()
		return StartedTurn{}, errAppServerClosed
	}
	if _, ok := s.turns[threadID]; ok {
		s.mu.Unlock()
		return StartedTurn{}, errTurnRunning
	}
	server := s.server
	ctx, cancel := context.WithCancel(context.Background())
	active := &activeTurn{createdAt: time.Now().UTC(), cancel: cancel}
	s.turns[threadID] = active
	s.mu.Unlock()

	idleTimer := time.AfterFunc(turnIdleTimeout, cancel)
	emit := func(event map[string]any, streamPosition string) {
		idleTimer.Reset(turnIdleTimeout)
		eventType, _ := event["type"].(string)
		if eventType == "" {
			eventType = "event"
		}
		if eventType == "thread.tokenUsage.updated" {
			s.recordTokenUsage(event)
		}
		turnID := s.resolveActiveTurnID(threadID, active, codexEventTurnID(event))
		s.events.publish(threadID, turnID, eventType, streamPosition, event)
	}

	started, err := server.runTurn(ctx, threadID, prompt, emit)
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

func (s *Service) removeActiveTurn(threadID string, active *activeTurn) {
	s.mu.Lock()
	if s.turns[threadID] == active {
		delete(s.turns, threadID)
	}
	s.mu.Unlock()
}

func (s *Service) resolveActiveTurnID(threadID string, active *activeTurn, eventTurnID string) string {
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

func (s *Service) Subscribe(threadID string) *Subscription {
	return s.events.subscribe(threadID)
}

func (s *Service) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.closed && s.server != nil
}

func (s *Service) AccountStatus(ctx context.Context) (AccountStatus, error) {
	server, err := s.currentServer()
	if err != nil {
		return AccountStatus{}, err
	}
	return server.accountStatus(ctx)
}

func (s *Service) LoginAPIKey(ctx context.Context, apiKey string) error {
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	return server.loginAPIKey(ctx, apiKey)
}

func (s *Service) StartChatGPTLogin(ctx context.Context) (DeviceLogin, error) {
	server, err := s.currentServer()
	if err != nil {
		return DeviceLogin{}, err
	}
	return server.startChatGPTLogin(ctx)
}

func (s *Service) Logout(ctx context.Context) error {
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	return server.logout(ctx)
}

func (s *Service) Cancel(threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return errServiceStopped
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

func (s *Service) Close() error {
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
	server := s.server
	s.server = nil
	s.mu.Unlock()

	for _, cancel := range cancels {
		cancel()
	}
	err := error(nil)
	if server != nil {
		err = server.close()
	}
	s.tokenUsageWriter.close()
	s.events.close()
	return err
}

func (s *Service) Delete(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	s.mu.Lock()
	_, running := s.turns[threadID]
	s.mu.Unlock()
	if running {
		return errTurnRunning
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	if err := server.deleteThread(ctx, threadID); err != nil {
		return err
	}
	if err := server.deleteConversationHome(threadID); err != nil {
		return err
	}
	s.events.release(threadID)
	return nil
}

func (s *Service) DeleteConversationHome(threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	return server.deleteConversationHome(threadID)
}

func (s *Service) recordTokenUsage(event map[string]any) {
	usage, ok := tokenUsageEventFromCodex(event)
	if !ok {
		return
	}
	s.tokenUsageWriter.enqueue(usage)
}

func (s *Service) persistTokenUsage(usage TokenUsageEvent) {
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

func (s *Service) ValidateThread(ctx context.Context, threadID string) error {
	if err := validateThreadID(threadID); err != nil {
		return err
	}
	server, err := s.currentServer()
	if err != nil {
		return err
	}
	if _, err := server.readThread(ctx, threadID, false); err != nil {
		return fmt.Errorf("read Codex thread: %w", err)
	}
	return nil
}

func validateThreadID(threadID string) error {
	value := strings.TrimSpace(threadID)
	switch strings.ToLower(value) {
	case "", "null", "undefined":
		return fmt.Errorf("%w: %q", errInvalidThreadID, threadID)
	default:
		return nil
	}
}
