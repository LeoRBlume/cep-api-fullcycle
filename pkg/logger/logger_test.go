package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// logEntry representa o JSON produzido pelo logger.
type logEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Service   string `json:"service"`
	Src       string `json:"src"`
	Message   string `json:"message"`
	TraceID   string `json:"trace_id"`
	UserID    string `json:"user_id"`
	Error     string `json:"error"`
}

func parseLog(t *testing.T, buf *bytes.Buffer) logEntry {
	t.Helper()
	var entry logEntry
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	return entry
}

func newTestLogger(level Level) (*Logger, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	l := newLogger(Config{ServiceName: "test-svc", Level: level}, buf)
	return l, buf
}

// --- TASK-01: Setup e Config ---

func TestSetup_AppliesServiceName(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	l.Info(context.Background(), "src", "msg")
	entry := parseLog(t, buf)
	assert.Equal(t, "test-svc", entry.Service)
}

func TestSetup_DefaultsWhenEmpty(t *testing.T) {
	var buf bytes.Buffer
	l := newLogger(Config{}, &buf)
	l.Info(context.Background(), "src", "msg")
	entry := parseLog(t, &buf)
	assert.Equal(t, "app", entry.Service)
}

func TestSetup_Global(t *testing.T) {
	prev := defaultLogger
	t.Cleanup(func() { defaultLogger = prev })

	Setup(Config{ServiceName: "global-svc", Level: LevelDebug})
	assert.Equal(t, "global-svc", defaultLogger.serviceName)
}

// --- TASK-02: Contexto ---

func TestWithTraceID_InjectsValue(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-abc")
	traceID, _ := extractFromContext(ctx)
	assert.Equal(t, "trace-abc", traceID)
}

func TestWithUserID_InjectsValue(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-xyz")
	_, userID := extractFromContext(ctx)
	assert.Equal(t, "user-xyz", userID)
}

func TestWithTraceID_DoesNotMutateOriginal(t *testing.T) {
	orig := context.Background()
	_ = WithTraceID(orig, "trace-abc")
	traceID, _ := extractFromContext(orig)
	assert.Empty(t, traceID)
}

func TestExtractFromContext_EmptyWhenMissing(t *testing.T) {
	traceID, userID := extractFromContext(context.Background())
	assert.Empty(t, traceID)
	assert.Empty(t, userID)
}

// --- TASK-03: Funções de log sem formatação ---

func TestInfo_OutputsRequiredFields(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	l.Info(context.Background(), "SomeService.Do", "hello world")
	entry := parseLog(t, buf)

	assert.Equal(t, "INFO", entry.Level)
	assert.Equal(t, "test-svc", entry.Service)
	assert.Equal(t, "SomeService.Do", entry.Src)
	assert.Equal(t, "hello world", entry.Message)
	assert.NotEmpty(t, entry.Timestamp)
}

func TestDebug_OutputsDebugLevel(t *testing.T) {
	l, buf := newTestLogger(LevelDebug)
	l.Debug(context.Background(), "src", "dbg msg")
	entry := parseLog(t, buf)
	assert.Equal(t, "DEBUG", entry.Level)
}

func TestWarn_OutputsWarnLevel(t *testing.T) {
	l, buf := newTestLogger(LevelWarn)
	l.Warn(context.Background(), "src", "warn msg")
	entry := parseLog(t, buf)
	assert.Equal(t, "WARN", entry.Level)
}

func TestError_IncludesErrorField(t *testing.T) {
	l, buf := newTestLogger(LevelError)
	l.Error(context.Background(), "src", "something failed", errors.New("oops"))
	entry := parseLog(t, buf)
	assert.Equal(t, "ERROR", entry.Level)
	assert.Equal(t, "oops", entry.Error)
}

func TestError_NilErr_NoErrorField(t *testing.T) {
	l, buf := newTestLogger(LevelError)
	l.Error(context.Background(), "src", "msg", nil)
	var raw map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	_, hasError := raw["error"]
	assert.False(t, hasError)
}

func TestLevelFiltering_DebugSilencedAtInfo(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	l.Debug(context.Background(), "src", "should be silent")
	assert.Empty(t, buf.String())
}

func TestLevelFiltering_WarnSilencedAtError(t *testing.T) {
	l, buf := newTestLogger(LevelError)
	l.Warn(context.Background(), "src", "should be silent")
	assert.Empty(t, buf.String())
}

func TestContextFieldsAppearedInLog(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	ctx := WithTraceID(context.Background(), "tid-111")
	ctx = WithUserID(ctx, "uid-222")
	l.Info(ctx, "src", "msg")
	entry := parseLog(t, buf)
	assert.Equal(t, "tid-111", entry.TraceID)
	assert.Equal(t, "uid-222", entry.UserID)
}

func TestContextFields_OmittedWhenAbsent(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	l.Info(context.Background(), "src", "msg")
	var raw map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &raw))
	_, hasTrace := raw["trace_id"]
	_, hasUser := raw["user_id"]
	assert.False(t, hasTrace)
	assert.False(t, hasUser)
}

func TestFatal_CallsExitAndLogs(t *testing.T) {
	l, buf := newTestLogger(LevelDebug)
	var exitCode int
	l.exitFunc = func(code int) { exitCode = code }

	l.Fatal(context.Background(), "src", "fatal msg", errors.New("boom"))

	assert.Equal(t, 1, exitCode)
	entry := parseLog(t, buf)
	assert.Equal(t, "FATAL", entry.Level)
	assert.Equal(t, "boom", entry.Error)
}

// --- TASK-04: Funções com formatação ---

func TestInfof_FormatsMessage(t *testing.T) {
	l, buf := newTestLogger(LevelInfo)
	l.Infof(context.Background(), "src", "user %s created with id %d", "alice", 42)
	entry := parseLog(t, buf)
	assert.Equal(t, "user alice created with id 42", entry.Message)
}

func TestDebugf_FormatsMessage(t *testing.T) {
	l, buf := newTestLogger(LevelDebug)
	l.Debugf(context.Background(), "src", "debug %s", "value")
	entry := parseLog(t, buf)
	assert.Equal(t, "debug value", entry.Message)
}

func TestWarnf_FormatsMessage(t *testing.T) {
	l, buf := newTestLogger(LevelWarn)
	l.Warnf(context.Background(), "src", "warn %d", 99)
	entry := parseLog(t, buf)
	assert.Equal(t, "warn 99", entry.Message)
}

func TestErrorf_FormatsMessageAndIncludesError(t *testing.T) {
	l, buf := newTestLogger(LevelError)
	l.Errorf(context.Background(), "src", "failed to save user %s", errors.New("db error"), "bob")
	entry := parseLog(t, buf)
	assert.Equal(t, "failed to save user bob", entry.Message)
	assert.Equal(t, "db error", entry.Error)
}

func TestFatalf_FormatsMessageCallsExit(t *testing.T) {
	l, buf := newTestLogger(LevelDebug)
	var exitCode int
	l.exitFunc = func(code int) { exitCode = code }

	l.Fatalf(context.Background(), "src", "port %d unavailable", errors.New("bind error"), 8080)

	assert.Equal(t, 1, exitCode)
	entry := parseLog(t, buf)
	assert.Equal(t, "port 8080 unavailable", entry.Message)
	assert.Equal(t, "bind error", entry.Error)
}

// --- TASK-05: TraceMiddleware ---

func TestTraceMiddleware_UsesExistingHeader(t *testing.T) {
	var capturedTraceID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID, _ = extractFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	mux := http.NewServeMux()
	mux.Handle("/", TraceMiddleware(handler))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Correlation-ID", "existing-id")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, "existing-id", capturedTraceID)
	assert.Equal(t, "existing-id", rec.Header().Get("X-Correlation-ID"))
}

func TestTraceMiddleware_GeneratesUUIDWhenAbsent(t *testing.T) {
	var capturedTraceID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedTraceID, _ = extractFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	TraceMiddleware(handler).ServeHTTP(rec, req)

	assert.NotEmpty(t, capturedTraceID)
	assert.Equal(t, capturedTraceID, rec.Header().Get("X-Correlation-ID"))
	// UUID v4 tem 36 chars
	assert.Len(t, capturedTraceID, 36)
}

func TestTraceMiddleware_AlwaysReturnsHeader(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	TraceMiddleware(handler).ServeHTTP(rec, req)

	assert.NotEmpty(t, rec.Header().Get("X-Correlation-ID"))
}

// --- TASK-06: NewNop ---

func TestNewNop_ProducesNoOutput(t *testing.T) {
	// Redireciona stdout para verificar que nada é escrito
	// Usa io.Discard internamente — basta verificar que não há panics
	// e que o handler realmente descarta
	nop := NewNop()
	ctx := WithTraceID(context.Background(), "trace")
	ctx = WithUserID(ctx, "user")

	// nenhuma dessas chamadas deve panicar
	nop.Debug(ctx, "src", "msg")
	nop.Info(ctx, "src", "msg")
	nop.Warn(ctx, "src", "msg")
	nop.Error(ctx, "src", "msg", errors.New("err"))
	nop.Debugf(ctx, "src", "msg %s", "arg")
	nop.Infof(ctx, "src", "msg %s", "arg")
	nop.Warnf(ctx, "src", "msg %s", "arg")
	nop.Errorf(ctx, "src", "msg %s", errors.New("err"), "arg")
}

func TestNewNop_FatalCallsExit(t *testing.T) {
	nop := NewNop()
	var exitCode int
	nop.exitFunc = func(code int) { exitCode = code }

	nop.Fatal(context.Background(), "src", "fatal", nil)
	assert.Equal(t, 1, exitCode)
}

func TestNewNop_FatalfCallsExit(t *testing.T) {
	nop := NewNop()
	var exitCode int
	nop.exitFunc = func(code int) { exitCode = code }

	nop.Fatalf(context.Background(), "src", "fatal %d", nil, 1)
	assert.Equal(t, 1, exitCode)
}

// --- Funções globais ---

func TestGlobalFunctions_DelegateToDefaultLogger(t *testing.T) {
	prev := defaultLogger
	t.Cleanup(func() { defaultLogger = prev })

	var buf bytes.Buffer
	defaultLogger = newLogger(Config{ServiceName: "global", Level: LevelDebug}, &buf)
	// override exitFunc so Fatal doesn't os.Exit
	defaultLogger.exitFunc = func(int) {}

	ctx := context.Background()
	Info(ctx, "src", "info msg")

	entry := parseLog(t, &buf)
	assert.Equal(t, "global", entry.Service)
	assert.Equal(t, "info msg", entry.Message)
}

func TestGlobalFunctions_AllLevels(t *testing.T) {
	prev := defaultLogger
	t.Cleanup(func() { defaultLogger = prev })

	defaultLogger = newLogger(Config{ServiceName: "g", Level: LevelDebug}, &bytes.Buffer{})
	defaultLogger.exitFunc = func(int) {}

	ctx := context.Background()
	err := errors.New("err")

	Debug(ctx, "src", "msg")
	Warn(ctx, "src", "msg")
	Error(ctx, "src", "msg", err)
	Fatal(ctx, "src", "msg", err)
	Debugf(ctx, "src", "msg %s", "a")
	Infof(ctx, "src", "msg %s", "a")
	Warnf(ctx, "src", "msg %s", "a")
	Errorf(ctx, "src", "msg %s", err, "a")
	Fatalf(ctx, "src", "msg %s", err, "a")
}
