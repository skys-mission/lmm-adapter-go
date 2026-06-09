package middleware

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/skys-mission/lmm-adapter-go/adapter"
)

type testLogger struct {
	entries []LogEntry
}

func (l *testLogger) Log(entry LogEntry) {
	l.entries = append(l.entries, entry)
}

func (l *testLogger) LastEntry() *LogEntry {
	if len(l.entries) == 0 {
		return nil
	}
	return &l.entries[len(l.entries)-1]
}

func TestStdLogger(t *testing.T) {
	// StdLogger writes to stdout — verify it doesn't panic
	logger := &StdLogger{}
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "test message",
		Source:    adapter.ProtocolOpenAIChat,
		Target:    adapter.ProtocolClaudeMessages,
	}
	// Should not panic
	logger.Log(entry)
}

func TestStdLogger_AllLevels(t *testing.T) {
	logger := &StdLogger{}
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError}
	for _, level := range levels {
		logger.Log(LogEntry{
			Timestamp: time.Now(),
			Level:     level,
			Message:   "test",
		})
		// Should not panic
	}
}

func TestLoggerAdapter_BeforeConvert(t *testing.T) {
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelDebug)

	la.BeforeConvert(adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, json.RawMessage(`{"test":true}`))

	entry := tl.LastEntry()
	if entry == nil {
		t.Fatal("expected log entry")
	}
	if entry.Level != LogLevelDebug {
		t.Fatalf("expected Debug level, got %d", entry.Level)
	}
	if entry.Source != adapter.ProtocolClaudeMessages {
		t.Fatalf("expected source claude, got %s", entry.Source)
	}
	if entry.Message != "Starting conversion" {
		t.Fatalf("expected 'Starting conversion', got %q", entry.Message)
	}
}

func TestLoggerAdapter_BeforeConvertFilteredByLevel(t *testing.T) {
	// With LogLevelInfo, debug messages should be filtered
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelInfo)

	la.BeforeConvert(adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, json.RawMessage(`{}`))

	if tl.LastEntry() != nil {
		t.Fatal("expected no entry for debug level filtered by info")
	}
}

func TestLoggerAdapter_AfterConvert_Success(t *testing.T) {
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelInfo)

	report := adapter.NewReport()
	la.AfterConvert(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		json.RawMessage(`{"req":1}`),
		json.RawMessage(`{"resp":2}`),
		report,
		time.Millisecond*100,
		nil,
	)

	entry := tl.LastEntry()
	if entry == nil {
		t.Fatal("expected log entry")
	}
	if entry.Level != LogLevelInfo {
		t.Fatalf("expected Info level, got %d", entry.Level)
	}
	if entry.Duration != time.Millisecond*100 {
		t.Fatalf("expected duration 100ms, got %v", entry.Duration)
	}
}

func TestLoggerAdapter_AfterConvert_Error(t *testing.T) {
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelInfo)

	la.AfterConvert(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		json.RawMessage(`{}`),
		nil,
		nil,
		time.Second,
		errors.New("upstream error"),
	)
}

func TestLoggerAdapter_AfterConvert_WithLostFields(t *testing.T) {
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelInfo)

	report := adapter.NewReport()
	report.AddLostField("test", "field1", "no mapping")
	report.AddWarning(adapter.SeverityWarning, "field2", "approximate mapping")

	la.AfterConvert(
		adapter.ProtocolOpenAIChat,
		adapter.ProtocolClaudeMessages,
		json.RawMessage(`{}`),
		json.RawMessage(`{}`),
		report,
		time.Millisecond*50,
		nil,
	)

	entry := tl.LastEntry()
	if entry == nil {
		t.Fatal("expected log entry")
	}
	if entry.Level != LogLevelWarn {
		t.Fatalf("expected Warn level (due to lost fields), got %d", entry.Level)
	}
	if entry.LostFields != 1 {
		t.Fatalf("expected 1 lost field, got %d", entry.LostFields)
	}
	if entry.Warnings != 1 {
		t.Fatalf("expected 1 warning, got %d", entry.Warnings)
	}
}

func TestLoggerAdapter_OnError(t *testing.T) {
	tl := &testLogger{}
	la := NewLoggerAdapter(tl, LogLevelError)

	la.OnError(
		adapter.ProtocolClaudeMessages,
		adapter.ProtocolOpenAIChat,
		json.RawMessage(`{}`),
		errors.New("conversion failure"),
	)
}

func TestLogEntry_JSONMarshal(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelInfo,
		Message:   "test",
		Source:    adapter.ProtocolOpenAIChat,
		Target:    adapter.ProtocolClaudeMessages,
		Duration:  time.Millisecond * 42,
		LostFields: 2,
		Warnings:   1,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if decoded["message"] != "test" {
		t.Fatalf("expected message 'test', got %v", decoded["message"])
	}
	if decoded["lost_fields"] != float64(2) {
		t.Fatalf("expected lost_fields 2, got %v", decoded["lost_fields"])
	}
}

func TestNewLoggerAdapter(t *testing.T) {
	logger := &StdLogger{}
	la := NewLoggerAdapter(logger, LogLevelWarn)

	if la == nil {
		t.Fatal("expected non-nil LoggerAdapter")
	}
	if la.Logger != logger {
		t.Fatal("expected logger to be stored")
	}
	if la.Level != LogLevelWarn {
		t.Fatalf("expected LogLevelWarn, got %d", la.Level)
	}
}
