package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultWorkspaceHome   = "workspace"
	defaultAppServerSocket = "/var/run/codex-app-server.sock"

	// Bounds socket connection, initialization and short workspace operations.
	appServerTimeout = 15 * time.Second

	// Codex device-code authentication expires after 15 minutes. The app-server
	// response does not currently expose this timeout, so keep it centralized here.
	chatGPTDeviceLoginExpiresIn = 15 * time.Minute
)

var (
	errAppServerClosed     = errors.New("Codex app-server is closed")
	errTurnRunning         = errors.New("agent thread already has an active turn")
	errAppServerTerminated = errors.New("codex app-server terminated")
)

type eventHandler func(map[string]any, string)

// appServer is a Unix socket client for the Codex app-server owned
// by the separate codex service.
type appServer struct {
	codexHome     string
	workspaceHome string
	threadCWD     string

	connection *websocket.Conn
	writeMu    sync.Mutex
	waitDone   chan struct{}
	waitOnce   sync.Once

	nextRequestID atomic.Int64
	nextInboundID atomic.Int64
	streamEpoch   string

	stateMu      sync.Mutex
	pending      map[int64]chan rpcEnvelope
	executions   map[string]*appServerTurn
	loaded       map[string]bool
	loads        map[string]*threadLoad
	account      AccountStatus
	accountKnown bool
	closed       bool
	terminalErr  error
}

type rpcEnvelope struct {
	ID             *int64          `json:"id,omitempty"`
	Method         string          `json:"method,omitempty"`
	Params         json.RawMessage `json:"params,omitempty"`
	Result         json.RawMessage `json:"result,omitempty"`
	Error          *rpcError       `json:"error,omitempty"`
	StreamPosition string          `json:"-"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// rpcCallError is the typed error returned when the app-server rejects a request.
// Callers inspect it with errors.As (on Code) rather than matching err.Error()
// text, which is not a stable API. rpcError above is the on-the-wire JSON shape;
// rpcCallError is the value returned to callers.
type rpcCallError struct {
	Method  string
	Code    int
	Message string
}

func (e *rpcCallError) Error() string {
	return fmt.Sprintf("Codex app-server RPC %s failed (%d): %s", e.Method, e.Code, e.Message)
}

type turnOutcome struct {
	err error
}

type threadLoad struct {
	done chan struct{}
	err  error
}

type appServerTurn struct {
	threadID string
	emit     eventHandler
	done     chan turnOutcome
	once     sync.Once
}

func (r *appServer) available() bool {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return !r.closed && r.terminalErr == nil
}

func newAppServer() (*appServer, error) {
	socketPath := strings.TrimSpace(os.Getenv("CODEX_SOCKET"))
	if socketPath == "" {
		socketPath = defaultAppServerSocket
	}
	connection, err := dialAppServer(socketPath, appServerTimeout)
	if err != nil {
		return nil, err
	}
	server := &appServer{
		threadCWD:  defaultWorkspaceHome,
		connection: connection,
		waitDone:   make(chan struct{}),
		pending:    make(map[int64]chan rpcEnvelope),
		executions: make(map[string]*appServerTurn),
		loaded:     make(map[string]bool),
		loads:      make(map[string]*threadLoad),
	}
	server.streamEpoch = fmt.Sprintf("%d", time.Now().UnixNano())
	go server.readLoop()

	startCtx, cancel := context.WithTimeout(context.Background(), appServerTimeout)
	defer cancel()
	initializeResult, err := server.request(startCtx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "agentrazor",
			"title":   "AgentRazor",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"experimentalApi": false,
		},
	})
	if err != nil {
		_ = server.close()
		return nil, fmt.Errorf("initialize Codex app-server: %w", err)
	}
	codexHome := filepath.Clean(strings.TrimSpace(stringValue(initializeResult["codexHome"])))
	if !filepath.IsAbs(codexHome) {
		_ = server.close()
		return nil, fmt.Errorf("initialize Codex app-server: invalid codexHome %q", codexHome)
	}
	server.codexHome = codexHome
	if filepath.IsAbs(server.threadCWD) {
		server.workspaceHome = filepath.Clean(server.threadCWD)
	} else {
		server.workspaceHome = filepath.Join(codexHome, server.threadCWD)
	}
	if err := server.notify("initialized", map[string]any{}); err != nil {
		_ = server.close()
		return nil, fmt.Errorf("acknowledge Codex app-server initialization: %w", err)
	}
	return server, nil
}

func dialAppServer(socketPath string, timeout time.Duration) (*websocket.Conn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
		NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", socketPath)
		},
	}
	var lastErr error
	for {
		connection, response, err := dialer.DialContext(ctx, "ws://localhost/rpc", nil)
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		if err == nil {
			connection.SetReadLimit(16 << 20)
			return connection, nil
		}
		lastErr = err
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect Codex app-server socket %s: %w", socketPath, lastErr)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

type AccountStatus struct {
	AuthMode string
	Email    string
	PlanType string
	LoggedIn bool
}

type DeviceLogin struct {
	LoginID         string
	VerificationURL string
	UserCode        string
	ExpiresIn       int64
}

func (r *appServer) accountStatus(ctx context.Context) (AccountStatus, error) {
	result, err := r.request(ctx, "account/read", map[string]any{"refreshToken": false})
	if err != nil {
		return AccountStatus{}, err
	}
	account, _ := result["account"].(map[string]any)
	status := accountStatusFromMap(account)
	if status.LoggedIn {
		r.setAccountStatus(status)
		return status, nil
	}
	if cached, ok := r.cachedAccountStatus(); ok {
		return cached, nil
	}
	return status, nil
}

func (r *appServer) loginAPIKey(ctx context.Context, apiKey string) error {
	_, err := r.request(ctx, "account/login/start", map[string]any{"type": "apiKey", "apiKey": apiKey})
	if err == nil {
		r.setAccountStatus(AccountStatus{AuthMode: "apikey", LoggedIn: true})
	}
	return err
}

func (r *appServer) startChatGPTLogin(ctx context.Context) (DeviceLogin, error) {
	result, err := r.request(ctx, "account/login/start", map[string]any{"type": "chatgptDeviceCode"})
	if err != nil {
		return DeviceLogin{}, err
	}
	return DeviceLogin{
		LoginID:         stringValue(result["loginId"]),
		VerificationURL: stringValue(result["verificationUrl"]),
		UserCode:        stringValue(result["userCode"]),
		ExpiresIn:       int64(chatGPTDeviceLoginExpiresIn / time.Second),
	}, nil
}

func (r *appServer) logout(ctx context.Context) error {
	_, err := r.request(ctx, "account/logout", map[string]any{})
	if err == nil {
		r.setAccountStatus(AccountStatus{})
	}
	return err
}

func accountStatusFromMap(account map[string]any) AccountStatus {
	mode := normalizeAuthMode(stringValue(account["type"]))
	return AccountStatus{
		AuthMode: mode,
		Email:    stringValue(account["email"]),
		PlanType: stringValue(account["planType"]),
		LoggedIn: mode != "",
	}
}

func normalizeAuthMode(mode string) string {
	if mode == "apiKey" {
		return "apikey"
	}
	return mode
}

func (r *appServer) setAccountStatus(status AccountStatus) {
	r.stateMu.Lock()
	r.account = status
	r.accountKnown = true
	r.stateMu.Unlock()
}

func (r *appServer) cachedAccountStatus() (AccountStatus, bool) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	return r.account, r.accountKnown
}

func (r *appServer) runTurn(ctx context.Context, threadID, prompt string, emit eventHandler) (StartedTurn, error) {
	if threadID == "" {
		return StartedTurn{}, errors.New("thread id is required")
	}
	resumed, err := r.ensureThread(ctx, threadID)
	if err != nil {
		return StartedTurn{}, err
	}
	if resumed {
		emitEvent(emit, "thread.resumed", map[string]any{
			"threadId": threadID,
		}, r.currentStreamPosition())
	}
	return r.startTurn(ctx, threadID, prompt, emit)
}

func (r *appServer) startThread(ctx context.Context) (string, error) {
	if _, err := r.request(ctx, "fs/createDirectory", map[string]any{
		"path": r.workspaceHome, "recursive": true,
	}); err != nil {
		return "", fmt.Errorf("create conversation root: %w", err)
	}
	result, err := r.request(ctx, "thread/start", map[string]any{
		"cwd": r.threadCWD,
	})
	if err != nil {
		return "", fmt.Errorf("start Codex thread: %w", err)
	}
	threadID := ""
	if thread, ok := result["thread"].(map[string]any); ok {
		threadID = stringValue(thread["id"])
	}
	if threadID == "" {
		threadID = stringValue(result["id"])
	}
	if threadID == "" {
		return "", errors.New("Codex thread/start response did not contain a thread id")
	}
	r.stateMu.Lock()
	r.loaded[threadID] = true
	r.stateMu.Unlock()
	return threadID, nil
}

// ensureThread only resumes after this app-server process starts. Threads
// created in the current process stay loaded and can receive turns directly.
func (r *appServer) ensureThread(ctx context.Context, threadID string) (bool, error) {
	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return false, errAppServerClosed
	}
	if r.loaded[threadID] {
		r.stateMu.Unlock()
		return false, nil
	}
	if load := r.loads[threadID]; load != nil {
		r.stateMu.Unlock()
		select {
		case <-load.done:
			return false, load.err
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	load := &threadLoad{done: make(chan struct{})}
	r.loads[threadID] = load
	r.stateMu.Unlock()

	conversationDir, err := r.conversationDir(threadID)
	if err != nil {
		r.stateMu.Lock()
		load.err = err
		delete(r.loads, threadID)
		close(load.done)
		r.stateMu.Unlock()
		return false, err
	}
	result, err := r.request(ctx, "thread/resume", map[string]any{
		"threadId": threadID,
		"cwd":      conversationDir,
	})
	if err != nil {
		err = fmt.Errorf("resume Codex thread %s: %w", threadID, err)
	}
	if err == nil {
		if thread, ok := result["thread"].(map[string]any); ok {
			resumedID := stringValue(thread["id"])
			if resumedID != "" && resumedID != threadID {
				err = fmt.Errorf("Codex resumed thread %s as unexpected thread %s", threadID, resumedID)
			}
		}
	}
	r.stateMu.Lock()
	if err == nil {
		r.loaded[threadID] = true
	}
	load.err = err
	delete(r.loads, threadID)
	close(load.done)
	r.stateMu.Unlock()
	return err == nil, err
}

func (r *appServer) startTurn(ctx context.Context, threadID, prompt string, emit eventHandler) (StartedTurn, error) {
	conversationDir, err := r.conversationDir(threadID)
	if err != nil {
		return StartedTurn{}, err
	}
	execution := &appServerTurn{
		threadID: threadID,
		emit:     emit,
		done:     make(chan turnOutcome, 1),
	}
	if err := r.registerExecution(execution); err != nil {
		return StartedTurn{}, err
	}

	params := map[string]any{
		"threadId": threadID,
		"cwd":      conversationDir,
		"input": []map[string]any{{
			"type": "text",
			"text": prompt,
		}},
	}
	result, err := r.request(ctx, "turn/start", params)
	if err != nil {
		r.unregisterExecution(threadID, execution)
		return StartedTurn{}, fmt.Errorf("start Codex turn: %w", err)
	}
	turnID := ""
	if turn, ok := result["turn"].(map[string]any); ok {
		turnID = stringValue(turn["id"])
	}
	if turnID == "" {
		turnID = stringValue(result["id"])
	}
	if turnID == "" {
		r.unregisterExecution(threadID, execution)
		return StartedTurn{}, errors.New("Codex turn/start response did not contain a turn id")
	}
	done := make(chan error, 1)
	go func() {
		defer close(done)
		defer r.unregisterExecution(threadID, execution)
		select {
		case outcome := <-execution.done:
			done <- outcome.err
		case <-ctx.Done():
			r.interruptTurn(threadID, turnID)
			done <- ctx.Err()
		}
	}()
	return StartedTurn{ID: turnID, StartedAt: time.Now().UTC(), Done: done}, nil
}

func (r *appServer) interruptTurn(threadID, turnID string) {
	interruptCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = r.request(interruptCtx, "turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	})
}

func (r *appServer) registerExecution(execution *appServerTurn) error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.closed {
		return errAppServerClosed
	}
	if r.terminalErr != nil {
		return r.terminalErr
	}
	if r.executions[execution.threadID] != nil {
		return errTurnRunning
	}
	r.executions[execution.threadID] = execution
	return nil
}

func (r *appServer) unregisterExecution(threadID string, execution *appServerTurn) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.executions[threadID] == execution {
		delete(r.executions, threadID)
	}
}

func (r *appServer) request(ctx context.Context, method string, params any) (map[string]any, error) {
	result, _, err := r.requestWithPosition(ctx, method, params)
	return result, err
}

func (r *appServer) requestWithPosition(ctx context.Context, method string, params any) (map[string]any, string, error) {
	requestID := r.nextRequestID.Add(1)
	responseCh := make(chan rpcEnvelope, 1)

	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return nil, "", errAppServerClosed
	}
	if r.terminalErr != nil {
		err := r.terminalErr
		r.stateMu.Unlock()
		return nil, "", err
	}
	r.pending[requestID] = responseCh
	r.stateMu.Unlock()

	if err := r.writeJSON(map[string]any{
		"id":     requestID,
		"method": method,
		"params": params,
	}); err != nil {
		r.removePending(requestID)
		return nil, "", err
	}

	select {
	case response := <-responseCh:
		if response.Error != nil {
			return nil, response.StreamPosition, &rpcCallError{
				Method:  method,
				Code:    response.Error.Code,
				Message: response.Error.Message,
			}
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return map[string]any{}, response.StreamPosition, nil
		}
		var result map[string]any
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return nil, response.StreamPosition, fmt.Errorf("decode Codex app-server RPC %s result: %w", method, err)
		}
		return result, response.StreamPosition, nil
	case <-ctx.Done():
		r.removePending(requestID)
		return nil, "", ctx.Err()
	}
}

func (r *appServer) notify(method string, params any) error {
	return r.writeJSON(map[string]any{
		"method": method,
		"params": params,
	})
}

func (r *appServer) writeJSON(value any) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.connection.WriteJSON(value); err != nil {
		return fmt.Errorf("write Codex app-server message: %w", err)
	}
	return nil
}

func (r *appServer) removePending(requestID int64) {
	r.stateMu.Lock()
	delete(r.pending, requestID)
	r.stateMu.Unlock()
}

func (r *appServer) readLoop() {
	defer r.waitOnce.Do(func() { close(r.waitDone) })
	for {
		var envelope rpcEnvelope
		if err := r.connection.ReadJSON(&envelope); err != nil {
			r.failAll(fmt.Errorf("read Codex app-server message: %w", err))
			return
		}
		envelope.StreamPosition = r.nextStreamPosition()
		switch {
		case envelope.Method != "" && envelope.ID != nil:
			// This integration runs with approval_policy="never" from config.toml. Reject any server
			// request instead of leaving app-server waiting indefinitely.
			_ = r.writeJSON(map[string]any{
				"id": *envelope.ID,
				"error": map[string]any{
					"code":    -32601,
					"message": "server request is not supported by this client",
				},
			})
		case envelope.Method != "":
			r.handleNotification(envelope.Method, envelope.Params, envelope.StreamPosition)
		case envelope.ID != nil:
			r.stateMu.Lock()
			responseCh := r.pending[*envelope.ID]
			delete(r.pending, *envelope.ID)
			r.stateMu.Unlock()
			if responseCh != nil {
				responseCh <- envelope
			}
		}
	}
}

func (r *appServer) handleNotification(method string, rawParams json.RawMessage, streamPosition string) {
	var params map[string]any
	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &params); err != nil {
			r.failAll(fmt.Errorf("decode Codex app-server notification %s: %w", method, err))
			return
		}
	}
	switch method {
	case "account/updated":
		status := AccountStatus{
			AuthMode: normalizeAuthMode(stringValue(params["authMode"])),
			PlanType: stringValue(params["planType"]),
		}
		status.LoggedIn = status.AuthMode != ""
		r.setAccountStatus(status)
		return
	case "account/login/completed":
		success, _ := params["success"].(bool)
		if success {
			status, _ := r.cachedAccountStatus()
			if status.AuthMode == "" {
				status.AuthMode = "chatgpt"
			}
			status.LoggedIn = true
			r.setAccountStatus(status)
		}
		return
	}
	threadID := stringValue(params["threadId"])
	if threadID == "" {
		if thread, ok := params["thread"].(map[string]any); ok {
			threadID = stringValue(thread["id"])
		}
	}
	if threadID == "" {
		if turn, ok := params["turn"].(map[string]any); ok {
			threadID = stringValue(turn["threadId"])
		}
	}
	if threadID == "" {
		return
	}

	r.stateMu.Lock()
	execution := r.executions[threadID]
	r.stateMu.Unlock()
	if execution == nil {
		return
	}
	execution.handleNotification(method, params, streamPosition)
}

func (r *appServer) failAll(err error) {
	if err == nil {
		err = errAppServerTerminated
	}
	r.stateMu.Lock()
	if r.terminalErr == nil {
		r.terminalErr = err
	}
	pending := r.pending
	r.pending = make(map[int64]chan rpcEnvelope)
	executions := make([]*appServerTurn, 0, len(r.executions))
	for _, execution := range r.executions {
		executions = append(executions, execution)
	}
	r.stateMu.Unlock()

	for _, responseCh := range pending {
		responseCh <- rpcEnvelope{Error: &rpcError{
			Code:    -32000,
			Message: err.Error(),
		}}
	}
	for _, execution := range executions {
		execution.finish(turnOutcome{err: err})
	}
}

func (r *appServer) close() error {
	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return nil
	}
	r.closed = true
	r.stateMu.Unlock()

	r.failAll(errAppServerClosed)
	_ = r.connection.Close()
	select {
	case <-r.waitDone:
	case <-time.After(2 * time.Second):
	}
	return nil
}

func (t *appServerTurn) handleNotification(method string, params map[string]any, streamPosition string) {
	eventType := strings.ReplaceAll(method, "/", ".")
	event := map[string]any{
		"type":   eventType,
		"method": method,
		"params": params,
	}
	if t.emit != nil {
		t.emit(event, streamPosition)
	}
	if method != "turn/completed" {
		return
	}

	status := ""
	var turnErr error
	if turn, ok := params["turn"].(map[string]any); ok {
		status = stringValue(turn["status"])
		if status == "failed" {
			if detail, ok := turn["error"].(map[string]any); ok {
				turnErr = errors.New(stringValue(detail["message"]))
			}
			if turnErr == nil || turnErr.Error() == "" {
				turnErr = errors.New("Codex turn failed")
			}
		} else if status == "interrupted" {
			turnErr = context.Canceled
		}
	}
	t.finish(turnOutcome{err: turnErr})
}

func (t *appServerTurn) finish(outcome turnOutcome) {
	t.once.Do(func() {
		t.done <- outcome
	})
}

func emitEvent(emit eventHandler, eventType string, params map[string]any, streamPosition string) {
	if emit == nil {
		return
	}
	emit(map[string]any{
		"type":   eventType,
		"method": strings.ReplaceAll(eventType, ".", "/"),
		"params": params,
	}, streamPosition)
}

func (r *appServer) nextStreamPosition() string {
	return fmt.Sprintf("%s:%d", r.streamEpoch, r.nextInboundID.Add(1))
}

func (r *appServer) currentStreamPosition() string {
	return fmt.Sprintf("%s:%d", r.streamEpoch, r.nextInboundID.Load())
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
