package stream

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/adapter/claude"
	"github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestSSEReader(t *testing.T) {
	input := "event: message\ndata: {\"type\":\"hello\"}\nid: 1\n\n" +
		"data: {\"type\":\"world\"}\n\n" +
		": comment\n\n" +
		"data: line1\ndata: line2\n\n"

	reader := NewSSEReader(strings.NewReader(input))

	evt1, err := reader.Read()
	if err != nil {
		t.Fatalf("Read 1 failed: %v", err)
	}
	if evt1.Event != "message" {
		t.Fatalf("expected event message, got %s", evt1.Event)
	}
	if evt1.Data != `{"type":"hello"}` {
		t.Fatalf("expected data, got %s", evt1.Data)
	}
	if evt1.ID != "1" {
		t.Fatalf("expected id 1, got %s", evt1.ID)
	}

	evt2, err := reader.Read()
	if err != nil {
		t.Fatalf("Read 2 failed: %v", err)
	}
	if evt2.Data != `{"type":"world"}` {
		t.Fatalf("expected data, got %s", evt2.Data)
	}
	if evt2.Event != "" {
		t.Fatalf("expected empty event, got %s", evt2.Event)
	}

	evt3, err := reader.Read()
	if err != nil {
		t.Fatalf("Read 3 failed: %v", err)
	}
	if evt3.Data != "line1\nline2" {
		t.Fatalf("expected multi-line data, got %s", evt3.Data)
	}

	_, err = reader.Read()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestSSEWriter(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	evt := &SSEEvent{
		Event: "message",
		Data:  `{"type":"hello"}`,
		ID:    "42",
	}
	if err := writer.Write(evt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "event: message") {
		t.Fatalf("expected 'event: message' in output, got %s", output)
	}
	if !strings.Contains(output, `data: {"type":"hello"}`) {
		t.Fatalf("expected data line in output, got %s", output)
	}
	if !strings.Contains(output, "id: 42") {
		t.Fatalf("expected id line in output, got %s", output)
	}
}

func TestSSEWriterMultiLineData(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	evt := &SSEEvent{
		Data: "line1\nline2\nline3",
	}
	if err := writer.Write(evt); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "data: line1\ndata: line2\ndata: line3\n") {
		t.Fatalf("expected multi-line data, got %s", output)
	}
}

func TestSSEWriterRaw(t *testing.T) {
	var buf bytes.Buffer
	writer := NewSSEWriter(&buf)

	if err := writer.WriteRaw("custom: data\n\n"); err != nil {
		t.Fatalf("WriteRaw failed: %v", err)
	}

	if buf.String() != "custom: data\n\n" {
		t.Fatalf("expected raw output, got %s", buf.String())
	}
}

func TestStreamConverter(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	converter := NewStreamConverter(src, dst)

	events := []string{
		`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`,
		`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
		`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
		`{"type":"content_block_stop","index":0}`,
		`{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":10}}`,
		`{"type":"message_stop"}`,
	}

	var allResults []json.RawMessage
	for _, evt := range events {
		results, err := converter.Convert(json.RawMessage(evt))
		if err != nil {
			t.Fatalf("Convert failed for %s: %v", evt, err)
		}
		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		t.Fatal("expected some output events")
	}

	hasDelta := false
	for _, r := range allResults {
		var chunk map[string]any
		if err := json.Unmarshal(r, &chunk); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		obj, _ := chunk["object"].(string)
		if obj == "chat.completion.chunk" {
			choices, _ := chunk["choices"].([]any)
			if len(choices) > 0 {
				c := choices[0].(map[string]any)
				if delta, ok := c["delta"].(map[string]any); ok {
					if content, ok := delta["content"].(string); ok && content != "" {
						hasDelta = true
					}
				}
			}
		}
	}

	if !hasDelta {
		t.Fatal("expected at least one delta event with content")
	}
}

func TestStreamPipeline(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	pipeline := NewPipeline(src, dst)

	sseInput := "data: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_001\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-3-opus\",\"content\":[],\"usage\":{\"input_tokens\":10,\"output_tokens\":0}}}\n\n" +
		"data: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n" +
		"data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":5}}\n\n" +
		"data: {\"type\":\"message_stop\"}\n\n"

	var outBuf bytes.Buffer
	ctx := context.Background()
	if err := pipeline.Pipe(ctx, strings.NewReader(sseInput), &outBuf); err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}

	output := outBuf.String()
	if output == "" {
		t.Fatal("expected non-empty output")
	}

	reader := NewSSEReader(strings.NewReader(output))
	eventCount := 0
	for {
		evt, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read failed: %v", err)
		}
		if evt.Data == "" {
			continue
		}
		eventCount++

		var chunk map[string]any
		if err := json.Unmarshal([]byte(evt.Data), &chunk); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
	}

	if eventCount == 0 {
		t.Fatal("expected at least 1 SSE event in output")
	}
}

func TestStreamStateTracking(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	converter := NewStreamConverter(src, dst)
	state := converter.State()

	if state.Started {
		t.Fatal("expected Started false initially")
	}
	if state.RoleSent {
		t.Fatal("expected RoleSent false initially")
	}
	if state.ContentBlockOpen {
		t.Fatal("expected ContentBlockOpen false initially")
	}

	converter.Convert(json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`))
	if !state.Started {
		t.Fatal("expected Started true after message_start")
	}
	if state.Model != "claude-3-opus" {
		t.Fatalf("expected model claude-3-opus, got %s", state.Model)
	}
	if state.ID != "msg_001" {
		t.Fatalf("expected id msg_001, got %s", state.ID)
	}

	converter.Convert(json.RawMessage(`{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`))
	if !state.ContentBlockOpen {
		t.Fatal("expected ContentBlockOpen true after content_block_start")
	}

	converter.Convert(json.RawMessage(`{"type":"content_block_stop","index":0}`))
	if state.ContentBlockOpen {
		t.Fatal("expected ContentBlockOpen false after content_block_stop")
	}

	converter.Convert(json.RawMessage(`{"type":"message_delta","stop_reason":"end_turn","usage":{"output_tokens":10}}`))
	if state.StopReason == nil || *state.StopReason != uni.StopReasonEndTurn {
		t.Fatal("expected StopReason end_turn after message_delta")
	}
	if state.Usage == nil || state.Usage.OutputTokens != 10 {
		t.Fatal("expected Usage with 10 output tokens")
	}
}

func TestStreamConverterReset(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	converter := NewStreamConverter(src, dst)

	converter.Convert(json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`))

	if !converter.State().Started {
		t.Fatal("expected Started true")
	}

	converter.Reset()

	if converter.State().Started {
		t.Fatal("expected Started false after reset")
	}
	if converter.State().Model != "" {
		t.Fatal("expected empty model after reset")
	}
}

func TestSSEReaderPeek(t *testing.T) {
	input := "data: hello\n\n"
	reader := NewSSEReader(strings.NewReader(input))

	peeked, err := reader.Peek()
	if err != nil {
		t.Fatalf("Peek failed: %v", err)
	}
	if peeked.Data != "hello" {
		t.Fatalf("expected peeked data 'hello', got %s", peeked.Data)
	}
}

func TestSSEEventString(t *testing.T) {
	evt := &SSEEvent{
		ID:    "1",
		Event: "test",
		Data:  "hello",
		Retry: 3000,
	}

	s := evt.String()
	if !strings.Contains(s, "id: 1") {
		t.Fatalf("expected id line, got %s", s)
	}
	if !strings.Contains(s, "event: test") {
		t.Fatalf("expected event line, got %s", s)
	}
	if !strings.Contains(s, "retry: 3000") {
		t.Fatalf("expected retry line, got %s", s)
	}
	if !strings.Contains(s, "data: hello") {
		t.Fatalf("expected data line, got %s", s)
	}
}

func TestSSEEventIsEmpty(t *testing.T) {
	evt := &SSEEvent{}
	if !evt.IsEmpty() {
		t.Fatal("expected empty event")
	}

	evt.Data = "hello"
	if evt.IsEmpty() {
		t.Fatal("expected non-empty event")
	}
}

func TestStreamPipelinePipeEvents(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	pipeline := NewPipeline(src, dst)

	input := make(chan json.RawMessage, 3)
	output := make(chan json.RawMessage, 10)

	input <- json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`)
	input <- json.RawMessage(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi"}}`)
	input <- json.RawMessage(`{"type":"message_stop"}`)
	close(input)

	ctx := context.Background()
	if err := pipeline.PipeEvents(ctx, input, output); err != nil {
		t.Fatalf("PipeEvents failed: %v", err)
	}

	count := 0
	for range output {
		count++
	}
	if count == 0 {
		t.Fatal("expected at least 1 output event")
	}
}

func TestPipelineConvertSingle(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	pipeline := NewPipeline(src, dst)

	results, err := pipeline.ConvertSingle(json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`))
	if err != nil {
		t.Fatalf("ConvertSingle failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected at least 1 result")
	}

	var chunk map[string]any
	if err := json.Unmarshal(results[0], &chunk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if chunk["object"] != "chat.completion.chunk" {
		t.Fatalf("expected chat.completion.chunk, got %v", chunk["object"])
	}
}

func TestStreamStateReset(t *testing.T) {
	state := NewStreamState()
	state.Started = true
	state.RoleSent = true
	state.ContentBlockOpen = true
	state.CurrentBlockIndex = 5
	state.CurrentToolIndex = 3
	state.Model = "test-model"
	state.ID = "test-id"
	stopReason := uni.StopReasonEndTurn
	state.StopReason = &stopReason
	state.StopSequence = "END"
	state.Usage = &uni.Usage{InputTokens: 100}
	state.Metadata["key"] = "value"

	state.Reset()

	if state.Started {
		t.Fatal("expected Started false")
	}
	if state.RoleSent {
		t.Fatal("expected RoleSent false")
	}
	if state.ContentBlockOpen {
		t.Fatal("expected ContentBlockOpen false")
	}
	if state.CurrentBlockIndex != 0 {
		t.Fatal("expected CurrentBlockIndex 0")
	}
	if state.CurrentToolIndex != 0 {
		t.Fatal("expected CurrentToolIndex 0")
	}
	if state.Model != "" {
		t.Fatal("expected empty model")
	}
	if state.ID != "" {
		t.Fatal("expected empty id")
	}
	if state.StopReason != nil {
		t.Fatal("expected nil StopReason")
	}
	if state.StopSequence != "" {
		t.Fatal("expected empty StopSequence")
	}
	if state.Usage != nil {
		t.Fatal("expected nil Usage")
	}
	if len(state.Metadata) != 0 {
		t.Fatal("expected empty Metadata")
	}
}

func TestStreamConverterReport(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	converter := NewStreamConverter(src, dst)

	converter.Convert(json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`))

	report := converter.Report()
	if report == nil {
		t.Fatal("expected non-nil report")
	}
}

func TestNewStreamState(t *testing.T) {
	state := NewStreamState()
	if state == nil {
		t.Fatal("expected non-nil state")
	}
	if state.Metadata == nil {
		t.Fatal("expected non-nil Metadata")
	}
}

func TestNewPipeline(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	pipeline := NewPipeline(src, dst)

	if pipeline == nil {
		t.Fatal("expected non-nil pipeline")
	}
	if pipeline.Converter() == nil {
		t.Fatal("expected non-nil converter")
	}
}

func TestPipelineReset(t *testing.T) {
	src := claude.New()
	dst := openaichat.New()
	pipeline := NewPipeline(src, dst)

	pipeline.ConvertSingle(json.RawMessage(`{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"usage":{"input_tokens":10,"output_tokens":0}}}`))

	if !pipeline.Converter().State().Started {
		t.Fatal("expected Started true")
	}

	pipeline.Reset()

	if pipeline.Converter().State().Started {
		t.Fatal("expected Started false after reset")
	}
}
