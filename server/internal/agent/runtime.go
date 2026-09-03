package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrRuntimeClosed       = errors.New("agent runtime is closed")
	ErrThreadTurnRunning   = errors.New("agent thread already has an active turn")
	ErrAppServerTerminated = errors.New("codex app-server terminated")
)

type EventHandler func(map[string]any)

type CodexAppServerOptions struct {
	Binary             string
	CodexHome          string
	PluginsRoot        string
	AgentrazorHome     string
	DisableApps        bool
	DisabledMCPServers []string
	StartTimeout       time.Duration
	ModelProvider      string
	Model              string
	ReasoningEffort    string
}

// CodexAppServerRuntime owns one long-running Codex app-server process. Business
// conversations are mapped to Codex threads and messages are mapped to turns.
type CodexAppServerRuntime struct {
	options CodexAppServerOptions

	cmd      *exec.Cmd
	stdin    io.WriteCloser
	writeMu  sync.Mutex
	stderr   limitedBuffer
	waitDone chan struct{}
	waitOnce sync.Once

	nextRequestID atomic.Int64

	stateMu     sync.Mutex
	pending     map[int64]chan rpcEnvelope
	executions  map[string]*appServerTurn
	loaded      map[string]bool
	loads       map[string]*threadLoad
	closed      bool
	terminalErr error
}

type rpcEnvelope struct {
	ID     *int64          `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// RPCError is the typed error returned when the app-server rejects a request.
// Callers inspect it with errors.As (on Code) rather than matching err.Error()
// text, which is not a stable API. rpcError above is the on-the-wire JSON shape;
// RPCError is the value returned to callers.
type RPCError struct {
	Method  string
	Code    int
	Message string
}

func (e *RPCError) Error() string {
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
	emit     EventHandler
	done     chan turnOutcome
	once     sync.Once
}

func NewCodexAppServerRuntime(options CodexAppServerOptions) (*CodexAppServerRuntime, error) {
	if options.Binary == "" {
		options.Binary = "codex"
	}
	if options.StartTimeout <= 0 {
		options.StartTimeout = 15 * time.Second
	}
	options.ModelProvider = strings.TrimSpace(options.ModelProvider)
	options.Model = strings.TrimSpace(options.Model)
	options.ReasoningEffort = strings.TrimSpace(options.ReasoningEffort)
	if options.CodexHome != "" {
		codexHome, err := filepath.Abs(options.CodexHome)
		if err != nil {
			return nil, fmt.Errorf("resolve Codex home: %w", err)
		}
		if err := ensureDefaultCodexConfig(codexHome); err != nil {
			return nil, err
		}
		options.CodexHome = codexHome
		if err := syncPluginSkills(options.PluginsRoot, options.CodexHome); err != nil {
			return nil, fmt.Errorf("sync plugin skills: %w", err)
		}
	}
	if options.AgentrazorHome == "" {
		options.AgentrazorHome = "data/agentrazor-home"
	}
	agentrazorHome, err := filepath.Abs(options.AgentrazorHome)
	if err != nil {
		return nil, fmt.Errorf("resolve conversation home: %w", err)
	}
	if err := os.MkdirAll(agentrazorHome, 0o700); err != nil {
		return nil, fmt.Errorf("create conversation home: %w", err)
	}
	options.AgentrazorHome = agentrazorHome

	args := []string{"app-server", "--listen", "stdio://"}
	if options.DisableApps {
		args = append(args, "--disable", "apps")
	}
	for _, server := range options.DisabledMCPServers {
		server = strings.TrimSpace(server)
		if server == "" {
			continue
		}
		args = append(args, "-c", fmt.Sprintf("mcp_servers.%s.enabled=false", server))
	}

	cmd := exec.Command(options.Binary, args...)
	if options.CodexHome != "" {
		cmd.Env = isolatedCodexEnvironment(os.Environ(), options.CodexHome)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("create Codex app-server stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("create Codex app-server stdout: %w", err)
	}

	runtime := &CodexAppServerRuntime{
		options:    options,
		cmd:        cmd,
		stdin:      stdin,
		waitDone:   make(chan struct{}),
		pending:    make(map[int64]chan rpcEnvelope),
		executions: make(map[string]*appServerTurn),
		loaded:     make(map[string]bool),
		loads:      make(map[string]*threadLoad),
	}
	runtime.stderr.limit = 64 << 10
	cmd.Stderr = &runtime.stderr

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("start Codex app-server: %w", err)
	}
	go runtime.readLoop(stdout)
	go runtime.waitProcess()

	startCtx, cancel := context.WithTimeout(context.Background(), options.StartTimeout)
	defer cancel()
	if _, err := runtime.request(startCtx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    "agentrazor",
			"title":   "AgentRazor",
			"version": "0.1.0",
		},
		"capabilities": map[string]any{
			"experimentalApi": false,
		},
	}); err != nil {
		_ = runtime.Close()
		return nil, fmt.Errorf("initialize Codex app-server: %w", err)
	}
	if err := runtime.notify("initialized", map[string]any{}); err != nil {
		_ = runtime.Close()
		return nil, fmt.Errorf("acknowledge Codex app-server initialization: %w", err)
	}
	return runtime, nil
}

func isolatedCodexEnvironment(environment []string, codexHome string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, value := range environment {
		if strings.HasPrefix(value, "CODEX_HOME=") || strings.HasPrefix(value, "CODEX_SQLITE_HOME=") {
			continue
		}
		result = append(result, value)
	}
	return append(result, "CODEX_HOME="+codexHome)
}

func (r *CodexAppServerRuntime) StartTurn(ctx context.Context, threadID, prompt string, emit EventHandler) (StartedTurn, error) {
	if threadID == "" {
		return StartedTurn{}, errors.New("thread id is required")
	}
	resumed, err := r.ensureThread(ctx, threadID)
	if err != nil {
		return StartedTurn{}, err
	}
	if resumed {
		emitRuntimeEvent(emit, "thread.resumed", map[string]any{
			"threadId": threadID,
		})
	}
	return r.startTurn(ctx, threadID, prompt, emit)
}

func (r *CodexAppServerRuntime) startThread(ctx context.Context) (string, error) {
	result, err := r.request(ctx, "thread/start", map[string]any{
		"approvalPolicy": "never",
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
func (r *CodexAppServerRuntime) ensureThread(ctx context.Context, threadID string) (bool, error) {
	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return false, ErrRuntimeClosed
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

	result, err := r.request(ctx, "thread/resume", map[string]any{
		"threadId":       threadID,
		"approvalPolicy": "never",
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

func (r *CodexAppServerRuntime) startTurn(ctx context.Context, threadID, prompt string, emit EventHandler) (StartedTurn, error) {
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
	if r.options.ModelProvider != "" {
		params["modelProvider"] = r.options.ModelProvider
	}
	if r.options.Model != "" {
		params["model"] = r.options.Model
	}
	if r.options.ReasoningEffort != "" {
		params["reasoningEffort"] = r.options.ReasoningEffort
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

func (r *CodexAppServerRuntime) interruptTurn(threadID, turnID string) {
	interruptCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = r.request(interruptCtx, "turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	})
}

func (r *CodexAppServerRuntime) registerExecution(execution *appServerTurn) error {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.closed {
		return ErrRuntimeClosed
	}
	if r.terminalErr != nil {
		return r.terminalErr
	}
	if r.executions[execution.threadID] != nil {
		return ErrThreadTurnRunning
	}
	r.executions[execution.threadID] = execution
	return nil
}

func (r *CodexAppServerRuntime) unregisterExecution(threadID string, execution *appServerTurn) {
	r.stateMu.Lock()
	defer r.stateMu.Unlock()
	if r.executions[threadID] == execution {
		delete(r.executions, threadID)
	}
}

func (r *CodexAppServerRuntime) request(ctx context.Context, method string, params any) (map[string]any, error) {
	requestID := r.nextRequestID.Add(1)
	responseCh := make(chan rpcEnvelope, 1)

	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return nil, ErrRuntimeClosed
	}
	if r.terminalErr != nil {
		err := r.terminalErr
		r.stateMu.Unlock()
		return nil, err
	}
	r.pending[requestID] = responseCh
	r.stateMu.Unlock()

	if err := r.writeJSON(map[string]any{
		"id":     requestID,
		"method": method,
		"params": params,
	}); err != nil {
		r.removePending(requestID)
		return nil, err
	}

	select {
	case response := <-responseCh:
		if response.Error != nil {
			return nil, &RPCError{
				Method:  method,
				Code:    response.Error.Code,
				Message: response.Error.Message,
			}
		}
		if len(response.Result) == 0 || string(response.Result) == "null" {
			return map[string]any{}, nil
		}
		var result map[string]any
		if err := json.Unmarshal(response.Result, &result); err != nil {
			return nil, fmt.Errorf("decode Codex app-server RPC %s result: %w", method, err)
		}
		return result, nil
	case <-ctx.Done():
		r.removePending(requestID)
		return nil, ctx.Err()
	}
}

func (r *CodexAppServerRuntime) notify(method string, params any) error {
	return r.writeJSON(map[string]any{
		"method": method,
		"params": params,
	})
}

func (r *CodexAppServerRuntime) writeJSON(value any) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := json.NewEncoder(r.stdin).Encode(value); err != nil {
		return fmt.Errorf("write Codex app-server message: %w", err)
	}
	return nil
}

func (r *CodexAppServerRuntime) removePending(requestID int64) {
	r.stateMu.Lock()
	delete(r.pending, requestID)
	r.stateMu.Unlock()
}

func (r *CodexAppServerRuntime) readLoop(stdout io.Reader) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 16<<20)
	for scanner.Scan() {
		var envelope rpcEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &envelope); err != nil {
			r.failAll(fmt.Errorf("decode Codex app-server message: %w", err))
			return
		}
		switch {
		case envelope.Method != "" && envelope.ID != nil:
			// This integration runs with approvalPolicy=never. Reject any server
			// request instead of leaving app-server waiting indefinitely.
			_ = r.writeJSON(map[string]any{
				"id": *envelope.ID,
				"error": map[string]any{
					"code":    -32601,
					"message": "server request is not supported by this client",
				},
			})
		case envelope.Method != "":
			r.handleNotification(envelope.Method, envelope.Params)
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
	if err := scanner.Err(); err != nil {
		r.failAll(fmt.Errorf("read Codex app-server output: %w", err))
		return
	}
	// stdout normally closes just before cmd.Wait returns. Let waitProcess
	// report the exit status together with stderr instead of racing it with a
	// generic "terminated" error.
	<-r.waitDone
}

func (r *CodexAppServerRuntime) handleNotification(method string, rawParams json.RawMessage) {
	var params map[string]any
	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &params); err != nil {
			r.failAll(fmt.Errorf("decode Codex app-server notification %s: %w", method, err))
			return
		}
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
	execution.handleNotification(method, params)
}

func (r *CodexAppServerRuntime) waitProcess() {
	err := r.cmd.Wait()
	message := strings.TrimSpace(r.stderr.String())
	if err != nil {
		if message != "" {
			err = fmt.Errorf("%w: %v: %s", ErrAppServerTerminated, err, message)
		} else {
			err = fmt.Errorf("%w: %v", ErrAppServerTerminated, err)
		}
	} else {
		err = ErrAppServerTerminated
	}
	r.failAll(err)
	r.waitOnce.Do(func() { close(r.waitDone) })
}

func (r *CodexAppServerRuntime) failAll(runtimeErr error) {
	if runtimeErr == nil {
		runtimeErr = ErrAppServerTerminated
	}
	r.stateMu.Lock()
	if r.terminalErr == nil {
		r.terminalErr = runtimeErr
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
			Message: runtimeErr.Error(),
		}}
	}
	for _, execution := range executions {
		execution.finish(turnOutcome{err: runtimeErr})
	}
}

func (r *CodexAppServerRuntime) Close() error {
	r.stateMu.Lock()
	if r.closed {
		r.stateMu.Unlock()
		return nil
	}
	r.closed = true
	r.stateMu.Unlock()

	r.failAll(ErrRuntimeClosed)
	_ = r.stdin.Close()
	select {
	case <-r.waitDone:
	case <-time.After(2 * time.Second):
		if r.cmd.Process != nil {
			_ = r.cmd.Process.Kill()
		}
		<-r.waitDone
	}
	return nil
}

func (t *appServerTurn) handleNotification(method string, params map[string]any) {
	eventType := strings.ReplaceAll(method, "/", ".")
	event := map[string]any{
		"type":   eventType,
		"method": method,
		"params": params,
	}
	if t.emit != nil {
		t.emit(event)
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

func emitRuntimeEvent(emit EventHandler, eventType string, params map[string]any) {
	if emit == nil {
		return
	}
	emit(map[string]any{
		"type":   eventType,
		"method": strings.ReplaceAll(eventType, ".", "/"),
		"params": params,
	})
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

type limitedBuffer struct {
	mu sync.Mutex
	bytes.Buffer
	limit int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(p)
	if b.Len() < b.limit {
		remaining := b.limit - b.Len()
		if len(p) > remaining {
			p = p[:remaining]
		}
		_, _ = b.Buffer.Write(p)
	}
	return original, nil
}

func (b *limitedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.Buffer.String()
}
