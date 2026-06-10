package claude

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeRequest(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "claude-3-opus-20240229",
		"max_tokens": 1024,
		"system": [{"type": "text", "text": "You are a helpful assistant."}],
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "Hello"}]},
			{"role": "assistant", "content": [{"type": "text", "text": "Hi there!"}]},
			{"role": "user", "content": [
				{"type": "tool_result", "tool_use_id": "call_123", "content": "result data"}
			]}
		],
		"tools": [
			{
				"name": "get_weather",
				"description": "Get the weather",
				"input_schema": {"type": "object", "properties": {"city": {"type": "string"}}}
			}
		],
		"tool_choice": {"type": "auto"},
		"temperature": 0.7,
		"top_p": 0.9,
		"top_k": 40,
		"stop_sequences": ["END"],
		"stream": true
	}`)

	params, report, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}
	_ = report

	if params.Model != "claude-3-opus-20240229" {
		t.Fatalf("expected model claude-3-opus-20240229, got %s", params.Model)
	}
	if params.MaxTokens == nil || *params.MaxTokens != 1024 {
		t.Fatalf("expected max_tokens 1024, got %v", params.MaxTokens)
	}
	if params.Temperature == nil || *params.Temperature != 0.7 {
		t.Fatalf("expected temperature 0.7, got %v", params.Temperature)
	}
	if params.TopP == nil || *params.TopP != 0.9 {
		t.Fatalf("expected top_p 0.9, got %v", params.TopP)
	}
	if params.TopK == nil || *params.TopK != 40 {
		t.Fatalf("expected top_k 40, got %v", params.TopK)
	}
	if !params.Stream {
		t.Fatal("expected stream true")
	}
	if len(params.StopSequences) != 1 || params.StopSequences[0] != "END" {
		t.Fatalf("expected stop_sequences [END], got %v", params.StopSequences)
	}

	if len(params.Messages) != 4 {
		t.Fatalf("expected 4 messages (1 system + 3), got %d", len(params.Messages))
	}

	if params.Messages[0].Role != uni.RoleSystem {
		t.Fatalf("expected first message role system, got %s", params.Messages[0].Role)
	}
	tp, ok := params.Messages[0].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart in system message")
	}
	if tp.Text != "You are a helpful assistant." {
		t.Fatalf("expected system text, got %s", tp.Text)
	}

	if params.Messages[1].Role != uni.RoleUser {
		t.Fatalf("expected user role, got %s", params.Messages[1].Role)
	}
	utp, ok := params.Messages[1].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart in user message")
	}
	if utp.Text != "Hello" {
		t.Fatalf("expected 'Hello', got %s", utp.Text)
	}

	if params.Messages[3].Role != uni.RoleUser {
		t.Fatalf("expected user role for tool result message (claude maps to user), got %s", params.Messages[3].Role)
	}
	trp, ok := params.Messages[3].Content[0].(uni.ToolResultPart)
	if !ok {
		t.Fatal("expected ToolResultPart")
	}
	if trp.ToolCallID != "call_123" {
		t.Fatalf("expected tool_call_id call_123, got %s", trp.ToolCallID)
	}

	if len(params.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(params.Tools))
	}
	if params.Tools[0].Name != "get_weather" {
		t.Fatalf("expected tool name get_weather, got %s", params.Tools[0].Name)
	}

	if params.ToolChoice == nil || params.ToolChoice.Type != uni.ToolChoiceAuto {
		t.Fatalf("expected tool_choice auto, got %v", params.ToolChoice)
	}
}

func TestEncodeRequest(t *testing.T) {
	a := New()
	maxTokens := int64(2048)
	temp := 0.5
	params := &uni.RequestParams{
		Model:       "claude-3-sonnet",
		MaxTokens:   &maxTokens,
		Temperature: &temp,
		Messages: []uni.Message{
			uni.SystemMessage(uni.TextPart{Text: "Be concise."}),
			uni.UserMessage(uni.TextPart{Text: "What is 2+2?"}),
		},
		Tools: []uni.Tool{
			{
				Name:        "calculator",
				Description: "A calculator",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"expr":{"type":"string"}}}`),
			},
		},
		ToolChoice: &uni.ToolChoice{Type: uni.ToolChoiceRequired},
		Stream:     true,
	}

	data, report, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}
	_ = report

	var req claudeMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if req.Model != "claude-3-sonnet" {
		t.Fatalf("expected model claude-3-sonnet, got %s", req.Model)
	}
	if req.MaxTokens != 2048 {
		t.Fatalf("expected max_tokens 2048, got %d", req.MaxTokens)
	}
	if req.Temperature == nil || *req.Temperature != 0.5 {
		t.Fatalf("expected temperature 0.5, got %v", req.Temperature)
	}
	if !req.Stream {
		t.Fatal("expected stream true")
	}
	var sysBlocks []systemBlock
	if err := json.Unmarshal(req.System, &sysBlocks); err != nil {
		t.Fatalf("unmarshal system: %v", err)
	}
	if len(sysBlocks) != 1 || sysBlocks[0].Text != "Be concise." {
		t.Fatalf("expected system block, got %v", string(req.System))
	}
	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("expected user role, got %s", req.Messages[0].Role)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "calculator" {
		t.Fatalf("expected 1 tool named calculator, got %v", req.Tools)
	}
	if req.ToolChoice == nil || req.ToolChoice.Type != "any" {
		t.Fatalf("expected tool_choice any, got %v", req.ToolChoice)
	}
}

func TestDecodeResponse(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"id": "msg_abc123",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-opus-20240229",
		"content": [
			{"type": "text", "text": "The answer is 42."},
			{"type": "tool_use", "id": "call_xyz", "name": "calculator", "input": {"expr": "6*7"}}
		],
		"stop_reason": "tool_use",
		"stop_sequence": null,
		"usage": {"input_tokens": 100, "output_tokens": 50}
	}`)

	resp, report, err := a.DecodeResponse(data)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	_ = report

	if resp.ID != "msg_abc123" {
		t.Fatalf("expected id msg_abc123, got %s", resp.ID)
	}
	if resp.Model != "claude-3-opus-20240229" {
		t.Fatalf("expected model, got %s", resp.Model)
	}
	if resp.StopReason != uni.StopReasonToolCalls {
		t.Fatalf("expected stop_reason tool_calls, got %s", resp.StopReason)
	}
	if resp.Usage.InputTokens != 100 {
		t.Fatalf("expected 100 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 50 {
		t.Fatalf("expected 50 output tokens, got %d", resp.Usage.OutputTokens)
	}
	if resp.Usage.TotalTokens != 150 {
		t.Fatalf("expected 150 total tokens, got %d", resp.Usage.TotalTokens)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Role != uni.RoleAssistant {
		t.Fatalf("expected assistant role, got %s", resp.Messages[0].Role)
	}
	if len(resp.Messages[0].Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(resp.Messages[0].Content))
	}

	tp, ok := resp.Messages[0].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart")
	}
	if tp.Text != "The answer is 42." {
		t.Fatalf("expected text, got %s", tp.Text)
	}

	tup, ok := resp.Messages[0].Content[1].(uni.ToolUsePart)
	if !ok {
		t.Fatal("expected ToolUsePart")
	}
	if tup.ToolCallID != "call_xyz" {
		t.Fatalf("expected call_xyz, got %s", tup.ToolCallID)
	}
	if tup.ToolName != "calculator" {
		t.Fatalf("expected calculator, got %s", tup.ToolName)
	}
}

func TestEncodeResponse(t *testing.T) {
	a := New()
	resp := &uni.Response{
		ID:    "msg_test",
		Model: "claude-3-opus",
		Messages: []uni.Message{
			uni.AssistantMessage(
				uni.TextPart{Text: "Hello!"},
				uni.ToolUsePart{ToolCallID: "tc_1", ToolName: "search", Arguments: json.RawMessage(`{"q":"test"}`)},
			),
		},
		Usage: uni.Usage{
			InputTokens:  200,
			OutputTokens: 100,
			TotalTokens:  300,
		},
		StopReason: uni.StopReasonToolCalls,
	}

	data, report, err := a.EncodeResponse(resp)
	if err != nil {
		t.Fatalf("EncodeResponse failed: %v", err)
	}
	_ = report

	var out claudeMessageResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != "msg_test" {
		t.Fatalf("expected id msg_test, got %s", out.ID)
	}
	if out.Model != "claude-3-opus" {
		t.Fatalf("expected model, got %s", out.Model)
	}
	if out.StopReason != "tool_use" {
		t.Fatalf("expected stop_reason tool_use, got %s", out.StopReason)
	}
	if len(out.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(out.Content))
	}
	if out.Content[0].Type != "text" || out.Content[0].Text != "Hello!" {
		t.Fatalf("expected text block, got %+v", out.Content[0])
	}
	if out.Content[1].Type != "tool_use" || out.Content[1].ID != "tc_1" {
		t.Fatalf("expected tool_use block, got %+v", out.Content[1])
	}
	if out.Usage.InputTokens != 200 {
		t.Fatalf("expected 200 input tokens, got %d", out.Usage.InputTokens)
	}
}

func TestRequestRoundTrip(t *testing.T) {
	a := New()
	original := json.RawMessage(`{
		"model": "claude-3-opus-20240229",
		"max_tokens": 4096,
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "Hello world"}]}
		],
		"temperature": 0.8,
		"stream": false
	}`)

	params, _, err := a.DecodeRequest(original)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	encoded, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var result claudeMessageRequest
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Model != "claude-3-opus-20240229" {
		t.Fatalf("model not preserved: %s", result.Model)
	}
	if result.MaxTokens != 4096 {
		t.Fatalf("max_tokens not preserved: %d", result.MaxTokens)
	}
	if result.Temperature == nil || *result.Temperature != 0.8 {
		t.Fatalf("temperature not preserved: %v", result.Temperature)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages not preserved: %d", len(result.Messages))
	}
	if result.Messages[0].Role != "user" {
		t.Fatalf("role not preserved: %s", result.Messages[0].Role)
	}
	var contentBlocks []contentBlock
	if err := json.Unmarshal(result.Messages[0].Content, &contentBlocks); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if len(contentBlocks) != 1 || contentBlocks[0].Text != "Hello world" {
		t.Fatalf("content not preserved: %v", string(result.Messages[0].Content))
	}
}

func TestStreamEventDecode(t *testing.T) {
	a := New()

	tests := []struct {
		name    string
		input   string
		checkFn func(*uni.StreamEvent) error
	}{
		{
			name:  "message_start",
			input: `{"type":"message_start","message":{"id":"msg_001","type":"message","role":"assistant","model":"claude-3-opus","content":[],"stop_reason":null,"usage":{"input_tokens":10,"output_tokens":0}}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStart {
					return errCheck("type", "start", string(e.Type))
				}
				if e.ID != "msg_001" {
					return errCheck("id", "msg_001", e.ID)
				}
				if e.Model != "claude-3-opus" {
					return errCheck("model", "claude-3-opus", e.Model)
				}
				return nil
			},
		},
		{
			name:  "content_block_start",
			input: `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventContentStart {
					return errCheck("type", "content_start", string(e.Type))
				}
				if len(e.Choices) != 1 || e.Choices[0].Index != 0 {
					return errCheck("choices", "index 0", "missing")
				}
				return nil
			},
		},
		{
			name:  "content_block_delta text",
			input: `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventDelta {
					return errCheck("type", "delta", string(e.Type))
				}
				if len(e.Choices) != 1 {
					return errCheck("choices len", "1", "0")
				}
				if len(e.Choices[0].Delta.Content) != 1 {
					return errCheck("content len", "1", "0")
				}
				tp, ok := e.Choices[0].Delta.Content[0].(uni.TextPart)
				if !ok {
					return errCheck("content type", "TextPart", "other")
				}
				if tp.Text != "Hello" {
					return errCheck("text", "Hello", tp.Text)
				}
				return nil
			},
		},
		{
			name:  "message_delta",
			input: `{"type":"message_delta","stop_reason":"end_turn","usage":{"output_tokens":50}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStop {
					return errCheck("type", "stop", string(e.Type))
				}
				if e.StopReason == nil || *e.StopReason != uni.StopReasonEndTurn {
					return errCheck("stop_reason", "end_turn", "nil")
				}
				if e.Usage == nil || e.Usage.OutputTokens != 50 {
					return errCheck("usage", "50", "nil")
				}
				return nil
			},
		},
		{
			name:  "message_stop",
			input: `{"type":"message_stop"}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStop {
					return errCheck("type", "stop", string(e.Type))
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, _, err := a.DecodeStreamEvent(json.RawMessage(tt.input))
			if err != nil {
				t.Fatalf("DecodeStreamEvent failed: %v", err)
			}
			if err := tt.checkFn(event); err != nil {
				t.Fatalf("check failed: %v", err)
			}
		})
	}
}

func TestReportLostFields(t *testing.T) {
	a := New()
	fp := 0.5
	pp := 0.3
	params := &uni.RequestParams{
		Model:            "claude-3-opus",
		MaxTokens:        int64Ptr(1024),
		FrequencyPenalty: &fp,
		PresencePenalty:  &pp,
		Messages: []uni.Message{
			uni.UserMessage(uni.TextPart{Text: "hi"}),
		},
	}

	_, report, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if len(report.LostFields) == 0 {
		t.Fatal("expected lost fields in report")
	}

	found := map[string]bool{}
	for _, lf := range report.LostFields {
		found[lf.Field] = true
	}
	if !found["frequency_penalty"] {
		t.Fatal("expected frequency_penalty in lost fields")
	}
	if !found["presence_penalty"] {
		t.Fatal("expected presence_penalty in lost fields")
	}
}

func int64Ptr(v int64) *int64 { return &v }

type checkError struct{ field, expected, got string }

func (e *checkError) Error() string {
	return e.field + ": expected " + e.expected + ", got " + e.got
}

func errCheck(field, expected, got string) error {
	return &checkError{field: field, expected: expected, got: got}
}

func TestStreamEventEncode(t *testing.T) {
	a := New()

	stopReason := uni.StopReasonEndTurn
	event := &uni.StreamEvent{
		Type:       uni.StreamEventStop,
		StopReason: &stopReason,
		Usage: &uni.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "message_delta" {
		t.Fatalf("expected type message_delta, got %v", raw["type"])
	}
	if raw["stop_reason"] != "end_turn" {
		t.Fatalf("expected stop_reason end_turn, got %v", raw["stop_reason"])
	}
}

func TestDecodeRequestWithImage(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "text", "text": "What is in this image?"},
				{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "iVBOR"}}
			]}
		]
	}`)

	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	if len(params.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(params.Messages))
	}
	if len(params.Messages[0].Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(params.Messages[0].Content))
	}

	ip, ok := params.Messages[0].Content[1].(uni.ImagePart)
	if !ok {
		t.Fatal("expected ImagePart")
	}
	if ip.Data != "iVBOR" {
		t.Fatalf("expected image data iVBOR, got %s", ip.Data)
	}
	if ip.MediaType != "image/png" {
		t.Fatalf("expected media type image/png, got %s", ip.MediaType)
	}
}

func TestEncodeRequestToolChoiceSpecific(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model:     "claude-3-opus",
		MaxTokens: int64Ptr(1024),
		Messages:  []uni.Message{uni.UserMessage(uni.TextPart{Text: "hi"})},
		ToolChoice: &uni.ToolChoice{
			Type:     uni.ToolChoiceSpecific,
			ToolName: "my_tool",
		},
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req claudeMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.ToolChoice == nil {
		t.Fatal("expected tool_choice")
	}
	if req.ToolChoice.Type != "tool" {
		t.Fatalf("expected type tool, got %s", req.ToolChoice.Type)
	}
	if req.ToolChoice.Name != "my_tool" {
		t.Fatalf("expected name my_tool, got %s", req.ToolChoice.Name)
	}
}

func TestDecodeResponseWithThinking(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"id": "msg_think",
		"type": "message",
		"role": "assistant",
		"model": "claude-3-opus",
		"content": [
			{"type": "thinking", "thinking": "Let me think...", "signature": "sig123"},
			{"type": "text", "text": "The answer."}
		],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`)

	resp, _, err := a.DecodeResponse(data)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}
	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	if len(resp.Messages[0].Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(resp.Messages[0].Content))
	}

	thp, ok := resp.Messages[0].Content[0].(uni.ThinkingPart)
	if !ok {
		t.Fatal("expected ThinkingPart")
	}
	if thp.Thinking != "Let me think..." {
		t.Fatalf("expected thinking text, got %s", thp.Thinking)
	}
	if thp.Signature != "sig123" {
		t.Fatalf("expected signature sig123, got %s", thp.Signature)
	}
}

func TestReportLostFieldsDecodeRequest(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"system": [{"type": "text", "text": "hi", "cache_control": {"type": "ephemeral"}}],
		"messages": [
			{"role": "user", "content": [{"type": "text", "text": "hello"}]}
		]
	}`)

	_, report, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	found := false
	for _, lf := range report.LostFields {
		if lf.Field == "system.cache_control" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected system.cache_control in lost fields")
	}
}

func TestEncodeClaudeToolResultContent(t *testing.T) {
	a := New()
	report := adapter.NewReport()

	// Text only content
	p := uni.ToolResultPart{
		ToolCallID: "call_1",
		Content:    []uni.ContentPart{uni.TextPart{Text: "result"}},
	}
	data, _, err := a.EncodeRequest(&uni.RequestParams{
		Model:    "claude-3-opus",
		Messages: []uni.Message{uni.UserMessage(p)},
	})
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}
	var req claudeMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Verify the encoded message structure.
	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("expected role user, got %s", req.Messages[0].Role)
	}

	var blocks []contentBlock
	if err := json.Unmarshal(req.Messages[0].Content, &blocks); err != nil {
		t.Fatalf("unmarshal content blocks: %v", err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(blocks))
	}
	block := blocks[0]
	if block.Type != "tool_result" {
		t.Fatalf("expected tool_result block, got %s", block.Type)
	}
	if block.ToolUseID != "call_1" {
		t.Fatalf("expected tool_use_id call_1, got %s", block.ToolUseID)
	}
	if block.IsError {
		t.Fatal("expected is_error to be false")
	}

	// The content should be a JSON string "result".
	var resultText string
	if err := json.Unmarshal(block.Content, &resultText); err != nil {
		t.Fatalf("unmarshal content text: %v", err)
	}
	if resultText != "result" {
		t.Fatalf("expected content text 'result', got %q", resultText)
	}

	// Report should not have any errors.
	if len(report.Warnings) > 0 || len(report.LostFields) > 0 {
		t.Fatalf("expected empty report, got warnings: %d, lost fields: %d", len(report.Warnings), len(report.LostFields))
	}
}

func TestDecodeToolResultContent_MergesTextAndContent(t *testing.T) {
	data := json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "tool_result", "tool_use_id": "call_1", "text": "from text", "content": "from content"}
			]}
		]
	}`)

	a := New()
	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	if len(params.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(params.Messages))
	}
	trp, ok := params.Messages[0].Content[0].(uni.ToolResultPart)
	if !ok {
		t.Fatal("expected ToolResultPart")
	}
	if trp.ToolCallID != "call_1" {
		t.Fatalf("expected call_1, got %s", trp.ToolCallID)
	}
	if len(trp.Content) != 2 {
		t.Fatalf("expected both text and content, got %d parts", len(trp.Content))
	}
}

func TestDecodeToolResultContent_WithArrayContent(t *testing.T) {
	data := json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": [
				{"type": "tool_result", "tool_use_id": "call_1", "content": [{"type": "text", "text": "nested"}]}
			]}
		]
	}`)

	a := New()
	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	trp := params.Messages[0].Content[0].(uni.ToolResultPart)
	if len(trp.Content) != 1 {
		t.Fatalf("expected 1 nested part, got %d", len(trp.Content))
	}
	tp, ok := trp.Content[0].(uni.TextPart)
	if !ok || tp.Text != "nested" {
		t.Fatalf("expected nested text part, got %+v", trp.Content[0])
	}
}

func TestConvertClaudeToolChoiceToUni_Unknown(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"messages": [{"role": "user", "content": "hi"}],
		"tool_choice": {"type": "unknown"}
	}`)

	params, report, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}
	if params.ToolChoice == nil || params.ToolChoice.Type != uni.ToolChoiceAuto {
		t.Fatalf("expected default auto, got %v", params.ToolChoice)
	}
	found := false
	for _, lf := range report.LostFields {
		if lf.Field == "tool_choice.type" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected tool_choice.type in lost fields")
	}
}

func TestStreamEventEncode_Start(t *testing.T) {
	a := New()
	event := &uni.StreamEvent{
		Type:    uni.StreamEventStart,
		ID:      "msg_001",
		Model:   "claude-3-opus",
		Created: 1234567890,
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "message_start" {
		t.Fatalf("expected message_start, got %v", raw["type"])
	}
}

func TestStreamEventEncode_ContentStart(t *testing.T) {
	a := New()
	event := &uni.StreamEvent{
		Type: uni.StreamEventContentStart,
		Choices: []uni.StreamChoice{
			{
				Index: 0,
				Delta: uni.StreamDelta{
					Content: []uni.ContentPart{uni.TextPart{Text: "Hello"}},
				},
			},
		},
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "content_block_start" {
		t.Fatalf("expected content_block_start, got %v", raw["type"])
	}
}

func TestStreamEventEncode_Delta(t *testing.T) {
	a := New()
	event := &uni.StreamEvent{
		Type: uni.StreamEventDelta,
		Choices: []uni.StreamChoice{
			{
				Index: 0,
				Delta: uni.StreamDelta{
					Content: []uni.ContentPart{uni.TextPart{Text: "delta text"}},
				},
			},
		},
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "content_block_delta" {
		t.Fatalf("expected content_block_delta, got %v", raw["type"])
	}
}

func TestStreamEventEncode_ContentStop(t *testing.T) {
	a := New()
	event := &uni.StreamEvent{
		Type: uni.StreamEventContentStop,
		Choices: []uni.StreamChoice{
			{Index: 0},
		},
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "content_block_stop" {
		t.Fatalf("expected content_block_stop, got %v", raw["type"])
	}
}

func TestStreamEventEncode_Error(t *testing.T) {
	a := New()
	event := &uni.StreamEvent{
		Type: uni.StreamEventError,
		Error: &uni.StreamError{
			Type:    "api_error",
			Message: "something went wrong",
		},
	}

	data, _, err := a.EncodeStreamEvent(event)
	if err != nil {
		t.Fatalf("EncodeStreamEvent failed: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if raw["type"] != "error" {
		t.Fatalf("expected error, got %v", raw["type"])
	}
}

func TestDecodeContentBlockStop(t *testing.T) {
	a := New()
	data := json.RawMessage(`{"type":"content_block_stop","index":2}`)

	event, _, err := a.DecodeStreamEvent(data)
	if err != nil {
		t.Fatalf("DecodeStreamEvent failed: %v", err)
	}
	if event.Type != uni.StreamEventContentStop {
		t.Fatalf("expected content_stop, got %s", event.Type)
	}
	if len(event.Choices) != 1 || event.Choices[0].Index != 2 {
		t.Fatalf("expected index 2, got %+v", event.Choices)
	}
}
