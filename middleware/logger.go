package middleware

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/skys-mission/lmm-adapter-go/adapter"
)

type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

type LogEntry struct {
	Timestamp    time.Time              `json:"timestamp"`
	Level        LogLevel               `json:"level"`
	Message      string                 `json:"message"`
	Source       adapter.Protocol       `json:"source,omitempty"`
	Target       adapter.Protocol       `json:"target,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	LostFields   int                    `json:"lost_fields,omitempty"`
	Warnings     int                    `json:"warnings,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]any         `json:"metadata,omitempty"`
}

type Logger interface {
	Log(entry LogEntry)
}

type ConversionHook interface {
	BeforeConvert(source, target adapter.Protocol, data json.RawMessage)
	AfterConvert(source, target adapter.Protocol, data json.RawMessage, result json.RawMessage, report *adapter.Report, duration time.Duration, err error)
	OnError(source, target adapter.Protocol, data json.RawMessage, err error)
}

type LoggerAdapter struct {
	Logger Logger
	Level  LogLevel
}

func NewLoggerAdapter(logger Logger, level LogLevel) *LoggerAdapter {
	return &LoggerAdapter{
		Logger: logger,
		Level:  level,
	}
}

func (l *LoggerAdapter) BeforeConvert(source, target adapter.Protocol, data json.RawMessage) {
	if l.Level <= LogLevelDebug {
		l.Logger.Log(LogEntry{
			Timestamp: time.Now(),
			Level:     LogLevelDebug,
			Message:   "Starting conversion",
			Source:    source,
			Target:    target,
			Metadata: map[string]any{
				"request_size": len(data),
			},
		})
	}
}

func (l *LoggerAdapter) AfterConvert(source, target adapter.Protocol, data json.RawMessage, result json.RawMessage, report *adapter.Report, duration time.Duration, err error) {
	if err != nil {
		l.Logger.Log(LogEntry{
			Timestamp: time.Now(),
			Level:     LogLevelError,
			Message:   "Conversion failed",
			Source:    source,
			Target:    target,
			Duration:  duration,
			Error:     err.Error(),
		})
		return
	}

	level := LogLevelInfo
	if report != nil && len(report.LostFields) > 0 {
		level = LogLevelWarn
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   "Conversion completed",
		Source:    source,
		Target:    target,
		Duration:  duration,
		Metadata: map[string]any{
			"request_size":  len(data),
			"response_size": len(result),
		},
	}

	if report != nil {
		entry.LostFields = len(report.LostFields)
		entry.Warnings = len(report.Warnings)
	}

	l.Logger.Log(entry)
}

func (l *LoggerAdapter) OnError(source, target adapter.Protocol, data json.RawMessage, err error) {
	l.Logger.Log(LogEntry{
		Timestamp: time.Now(),
		Level:     LogLevelError,
		Message:   "Conversion error",
		Source:    source,
		Target:    target,
		Error:     err.Error(),
	})
}

type StdLogger struct{}

func (l *StdLogger) Log(entry LogEntry) {
	levelStr := "INFO"
	switch entry.Level {
	case LogLevelDebug:
		levelStr = "DEBUG"
	case LogLevelWarn:
		levelStr = "WARN"
	case LogLevelError:
		levelStr = "ERROR"
	}

	data, _ := json.Marshal(entry)
	fmt.Printf("[%s] %s\n", levelStr, string(data))
}
