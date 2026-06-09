package adapter

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

type Protocol string

const (
	ProtocolClaudeMessages  Protocol = "claude_messages"
	ProtocolOpenAIChat      Protocol = "openai_chat"
	ProtocolOpenAIResponses Protocol = "openai_responses"
)

type RequestAdapter interface {
	DecodeRequest(data json.RawMessage) (*uni.RequestParams, *Report, error)
	EncodeRequest(params *uni.RequestParams) (json.RawMessage, *Report, error)
}

type ResponseAdapter interface {
	DecodeResponse(data json.RawMessage) (*uni.Response, *Report, error)
	EncodeResponse(resp *uni.Response) (json.RawMessage, *Report, error)
}

type StreamEventAdapter interface {
	DecodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *Report, error)
	EncodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *Report, error)
}

type ErrorAdapter interface {
	DecodeError(data json.RawMessage, statusCode int) (*uni.ErrorResponse, *Report, error)
	EncodeError(err *uni.ErrorResponse) (json.RawMessage, *Report, error)
}

type Adapter interface {
	RequestAdapter
	ResponseAdapter
	StreamEventAdapter
	ErrorAdapter
	Protocol() Protocol
}

type WarningSeverity int

const (
	SeverityInfo WarningSeverity = iota
	SeverityWarning
	SeverityError
)

type Warning struct {
	Severity WarningSeverity `json:"severity"`
	Field    string          `json:"field,omitempty"`
	Message  string          `json:"message"`
}

type LostField struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	Reason string `json:"reason,omitempty"`
}

type Report struct {
	Warnings   []Warning   `json:"warnings,omitempty"`
	LostFields []LostField `json:"lost_fields,omitempty"`
}

func NewReport() *Report {
	return &Report{}
}

func (r *Report) AddWarning(severity WarningSeverity, field, message string) {
	r.Warnings = append(r.Warnings, Warning{
		Severity: severity,
		Field:    field,
		Message:  message,
	})
}

func (r *Report) AddLostField(source, field, reason string) {
	r.LostFields = append(r.LostFields, LostField{
		Source: source,
		Field:  field,
		Reason: reason,
	})
}

func (r *Report) Merge(other *Report) {
	if other == nil {
		return
	}
	r.Warnings = append(r.Warnings, other.Warnings...)
	r.LostFields = append(r.LostFields, other.LostFields...)
}

type Converter struct {
	adapters map[Protocol]Adapter
	mu       sync.RWMutex
}

type ConverterOption func(*Converter)

func NewConverter(opts ...ConverterOption) *Converter {
	c := &Converter{
		adapters: make(map[Protocol]Adapter),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func WithAdapter(a Adapter) ConverterOption {
	return func(c *Converter) {
		c.Register(a)
	}
}

func (c *Converter) Register(a Adapter) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.adapters[a.Protocol()] = a
}

func (c *Converter) Get(protocol Protocol) (Adapter, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	a, ok := c.adapters[protocol]
	if !ok {
		return nil, fmt.Errorf("adapter not registered: %s", protocol)
	}
	return a, nil
}

func (c *Converter) List() []Protocol {
	c.mu.RLock()
	defer c.mu.RUnlock()
	protocols := make([]Protocol, 0, len(c.adapters))
	for p := range c.adapters {
		protocols = append(protocols, p)
	}
	return protocols
}

func (c *Converter) ConvertRequest(from, to Protocol, data json.RawMessage) (json.RawMessage, *Report, error) {
	report := NewReport()

	src, err := c.Get(from)
	if err != nil {
		return nil, report, fmt.Errorf("source protocol: %w", err)
	}

	dst, err := c.Get(to)
	if err != nil {
		return nil, report, fmt.Errorf("target protocol: %w", err)
	}

	unified, decodeReport, err := src.DecodeRequest(data)
	if err != nil {
		return nil, report, fmt.Errorf("decode from %s: %w", from, err)
	}
	report.Merge(decodeReport)

	result, encodeReport, err := dst.EncodeRequest(unified)
	if err != nil {
		return nil, report, fmt.Errorf("encode to %s: %w", to, err)
	}
	report.Merge(encodeReport)

	return result, report, nil
}

func (c *Converter) ConvertResponse(from, to Protocol, data json.RawMessage) (json.RawMessage, *Report, error) {
	report := NewReport()

	src, err := c.Get(from)
	if err != nil {
		return nil, report, fmt.Errorf("source protocol: %w", err)
	}

	dst, err := c.Get(to)
	if err != nil {
		return nil, report, fmt.Errorf("target protocol: %w", err)
	}

	unified, decodeReport, err := src.DecodeResponse(data)
	if err != nil {
		return nil, report, fmt.Errorf("decode from %s: %w", from, err)
	}
	report.Merge(decodeReport)

	result, encodeReport, err := dst.EncodeResponse(unified)
	if err != nil {
		return nil, report, fmt.Errorf("encode to %s: %w", to, err)
	}
	report.Merge(encodeReport)

	return result, report, nil
}

func (c *Converter) ConvertStreamEvent(from, to Protocol, data json.RawMessage) (json.RawMessage, *Report, error) {
	report := NewReport()

	src, err := c.Get(from)
	if err != nil {
		return nil, report, fmt.Errorf("source protocol: %w", err)
	}

	dst, err := c.Get(to)
	if err != nil {
		return nil, report, fmt.Errorf("target protocol: %w", err)
	}

	unified, decodeReport, err := src.DecodeStreamEvent(data)
	if err != nil {
		return nil, report, fmt.Errorf("decode from %s: %w", from, err)
	}
	report.Merge(decodeReport)

	result, encodeReport, err := dst.EncodeStreamEvent(unified)
	if err != nil {
		return nil, report, fmt.Errorf("encode to %s: %w", to, err)
	}
	report.Merge(encodeReport)

	return result, report, nil
}

func (c *Converter) ConvertError(from, to Protocol, data json.RawMessage, statusCode int) (json.RawMessage, *Report, error) {
	report := NewReport()

	src, err := c.Get(from)
	if err != nil {
		return nil, report, fmt.Errorf("source protocol: %w", err)
	}

	dst, err := c.Get(to)
	if err != nil {
		return nil, report, fmt.Errorf("target protocol: %w", err)
	}

	unified, decodeReport, err := src.DecodeError(data, statusCode)
	if err != nil {
		return nil, report, fmt.Errorf("decode error from %s: %w", from, err)
	}
	report.Merge(decodeReport)

	result, encodeReport, err := dst.EncodeError(unified)
	if err != nil {
		return nil, report, fmt.Errorf("encode error to %s: %w", to, err)
	}
	report.Merge(encodeReport)

	return result, report, nil
}
