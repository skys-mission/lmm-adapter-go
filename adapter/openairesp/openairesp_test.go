package openairesp

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeRequest(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"instructions": "You are helpful.",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "Hello"}]},
			{"type": "function_call", "call_id": "call_1", "name": "search", "arguments": "{\"q\":\"test\"}"},
			{"type": "function_call_output", "call_id": "call_1", "output": "result data"}
		],
		"tools": [
			{"type": "function", "name": "search", "description": "Search the web", "parameters": {"type": "object", "properties": {"q": {"type": "string"}}}}
		],
		"tool_choice": "auto",
		"temperature": 0.7,
		"top_p": 0.95,
		"max_output_tokens": 2048,
		"stream": true,
		"parallel_tool_calls": true
	}`)

	params, report, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}
	_ = report

	if params.Model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", params.Model)
	}
	if params.Temperature == nil || *params.Temperature != 0.7 {
		t.Fatalf("expected temperature 0.7, got %v", params.Temperature)
	}
	if params.TopP == nil || *params.TopP != 0.95 {
		t.Fatalf("expected top_p 0.95, got %v", params.TopP)
	}
	if params.MaxTokens == nil || *params.MaxTokens != 2048 {
		t.Fatalf("expected max_output_tokens 2048, got %v", params.MaxTokens)
	}
	if !params.Stream {
		t.Fatal("expected stream true")
	}

	if len(params.Messages) < 3 {
		t.Fatalf("expected at least 3 messages (system + user + tool), got %d", len(params.Messages))
	}

	if params.Messages[0].Role != uni.RoleSystem {
		t.Fatalf("expected first message system, got %s", params.Messages[0].Role)
	}
	stp, ok := params.Messages[0].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart in system message")
	}
	if stp.Text != "You are helpful." {
		t.Fatalf("expected instructions text, got %s", stp.Text)
	}

	hasUser := false
	hasTool := false
	hasAssistant := false
	for _, msg := range params.Messages {
		if msg.Role == uni.RoleUser {
			hasUser = true
		}
		if msg.Role == uni.RoleTool {
			hasTool = true
			trp, ok := msg.Content[0].(uni.ToolResultPart)
			if !ok {
				t.Fatal("expected ToolResultPart")
			}
			if trp.ToolCallID != "call_1" {
				t.Fatalf("expected call_1, got %s", trp.ToolCallID)
			}
		}
		if msg.Role == uni.RoleAssistant {
			hasAssistant = true
			tup, ok := msg.Content[0].(uni.ToolUsePart)
			if !ok {
				t.Fatal("expected ToolUsePart")
			}
			if tup.ToolCallID != "call_1" {
				t.Fatalf("expected call_1, got %s", tup.ToolCallID)
			}
		}
	}
	if !hasUser {
		t.Fatal("expected user message")
	}
	if !hasTool {
		t.Fatal("expected tool message")
	}
	if !hasAssistant {
		t.Fatal("expected assistant message with function_call")
	}

	if len(params.Tools) != 1 || params.Tools[0].Name != "search" {
		t.Fatalf("expected 1 tool named search, got %v", params.Tools)
	}
	if params.ToolChoice == nil || params.ToolChoice.Type != uni.ToolChoiceAuto {
		t.Fatalf("expected tool_choice auto, got %v", params.ToolChoice)
	}
}

func TestEncodeRequest(t *testing.T) {
	a := New()
	maxTokens := int64(1024)
	temp := 0.5
	params := &uni.RequestParams{
		Model:       "gpt-4o",
		MaxTokens:   &maxTokens,
		Temperature: &temp,
		Messages: []uni.Message{
			uni.SystemMessage(uni.TextPart{Text: "Be brief."}),
			uni.UserMessage(uni.TextPart{Text: "Hi"}),
		},
		Tools: []uni.Tool{
			{Name: "calc", Description: "Calculator", InputSchema: json.RawMessage(`{"type":"object"}`)},
		},
		ToolChoice: &uni.ToolChoice{Type: uni.ToolChoiceRequired},
		Stream:     true,
	}

	data, report, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}
	_ = report

	var req responseRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if req.Model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", req.Model)
	}
	if req.MaxOutputTokens == nil || *req.MaxOutputTokens != 1024 {
		t.Fatalf("expected max_output_tokens 1024, got %v", req.MaxOutputTokens)
	}
	if req.Instructions != "Be brief." {
		t.Fatalf("expected instructions 'Be brief.', got %s", req.Instructions)
	}
	if len(req.Tools) != 1 || req.Tools[0].Name != "calc" {
		t.Fatalf("expected 1 tool, got %v", req.Tools)
	}
	if string(req.ToolChoice) != `"required"` {
		t.Fatalf("expected tool_choice required, got %s", string(req.ToolChoice))
	}

	var items []inputItem
	if err := json.Unmarshal(req.Input, &items); err != nil {
		t.Fatalf("unmarshal input: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 input item, got %d", len(items))
	}
	if items[0].Type != "message" || items[0].Role != "user" {
		t.Fatalf("expected user message item, got %+v", items[0])
	}
}

func TestDecodeResponse(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"id": "resp_abc123",
		"object": "response",
		"created_at": 1700000000,
		"model": "gpt-4o",
		"status": "completed",
		"output": [
			{
				"type": "message",
				"role": "assistant",
				"status": "completed",
				"content": [{"type": "output_text", "text": "The answer is 42."}]
			},
			{
				"type": "function_call",
				"status": "completed",
				"call_id": "call_xyz",
				"name": "calculator",
				"arguments": "{\"expr\":\"6*7\"}"
			},
			{
				"type": "reasoning",
				"status": "completed",
				"summary": [{"type": "summary_text", "text": "Let me calculate..."}]
			}
		],
		"usage": {"input_tokens": 100, "output_tokens": 50, "total_tokens": 150}
	}`)

	resp, _, err := a.DecodeResponse(data)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}

	if resp.ID != "resp_abc123" {
		t.Fatalf("expected id resp_abc123, got %s", resp.ID)
	}
	if resp.Model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", resp.Model)
	}
	if resp.StopReason != uni.StopReasonEndTurn {
		t.Fatalf("expected stop_reason end_turn, got %s", resp.StopReason)
	}
	if resp.Usage.InputTokens != 100 {
		t.Fatalf("expected 100 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 50 {
		t.Fatalf("expected 50 output tokens, got %d", resp.Usage.OutputTokens)
	}
	if resp.Created != 1700000000 {
		t.Fatalf("expected created 1700000000, got %d", resp.Created)
	}

	if len(resp.Messages) < 1 {
		t.Fatalf("expected at least 1 message, got %d", len(resp.Messages))
	}

	assistantMsg := resp.Messages[0]
	if assistantMsg.Role != uni.RoleAssistant {
		t.Fatalf("expected assistant, got %s", assistantMsg.Role)
	}

	hasText := false
	hasToolUse := false
	hasThinking := false
	for _, part := range assistantMsg.Content {
		switch p := part.(type) {
		case uni.TextPart:
			hasText = true
			if p.Text != "The answer is 42." {
				t.Fatalf("expected text, got %s", p.Text)
			}
		case uni.ToolUsePart:
			hasToolUse = true
			if p.ToolCallID != "call_xyz" {
				t.Fatalf("expected call_xyz, got %s", p.ToolCallID)
			}
			if p.ToolName != "calculator" {
				t.Fatalf("expected calculator, got %s", p.ToolName)
			}
		case uni.ThinkingPart:
			hasThinking = true
			if p.Thinking != "Let me calculate..." {
				t.Fatalf("expected thinking text, got %s", p.Thinking)
			}
		}
	}
	if !hasText {
		t.Fatal("expected text content")
	}
	if !hasToolUse {
		t.Fatal("expected tool_use content")
	}
	if !hasThinking {
		t.Fatal("expected thinking content")
	}
}

func TestEncodeResponse(t *testing.T) {
	a := New()
	resp := &uni.Response{
		ID:      "resp_test",
		Model:   "gpt-4o",
		Created: 1700000000,
		Messages: []uni.Message{
			uni.AssistantMessage(
				uni.TextPart{Text: "Hello!"},
				uni.ToolUsePart{ToolCallID: "tc_1", ToolName: "search", Arguments: json.RawMessage(`{"q":"test"}`)},
			),
		},
		Usage: uni.Usage{
			InputTokens:  100,
			OutputTokens: 50,
			TotalTokens:  150,
		},
		StopReason: uni.StopReasonEndTurn,
	}

	data, _, err := a.EncodeResponse(resp)
	if err != nil {
		t.Fatalf("EncodeResponse failed: %v", err)
	}

	var out responseResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != "resp_test" {
		t.Fatalf("expected id resp_test, got %s", out.ID)
	}
	if out.Object != "response" {
		t.Fatalf("expected object response, got %s", out.Object)
	}
	if out.Status != "completed" {
		t.Fatalf("expected status completed, got %s", out.Status)
	}
	if out.Usage == nil || out.Usage.TotalTokens != 150 {
		t.Fatalf("expected total_tokens 150, got %v", out.Usage)
	}

	hasMessage := false
	hasFuncCall := false
	for _, item := range out.Output {
		if item.Type == "message" && len(item.Content) > 0 && item.Content[0].Text == "Hello!" {
			hasMessage = true
		}
		if item.Type == "function_call" && item.CallID == "tc_1" && item.Name == "search" {
			hasFuncCall = true
		}
	}
	if !hasMessage {
		t.Fatal("expected message output item")
	}
	if !hasFuncCall {
		t.Fatal("expected function_call output item")
	}
}

func TestRequestRoundTrip(t *testing.T) {
	a := New()
	original := json.RawMessage(`{
		"model": "gpt-4o",
		"input": [{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "Hello world"}]}],
		"instructions": "Be helpful.",
		"temperature": 0.8,
		"max_output_tokens": 512,
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

	var result responseRequest
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Model != "gpt-4o" {
		t.Fatalf("model not preserved: %s", result.Model)
	}
	if result.Temperature == nil || *result.Temperature != 0.8 {
		t.Fatalf("temperature not preserved: %v", result.Temperature)
	}
	if result.MaxOutputTokens == nil || *result.MaxOutputTokens != 512 {
		t.Fatalf("max_output_tokens not preserved: %v", result.MaxOutputTokens)
	}
	if result.Instructions != "Be helpful." {
		t.Fatalf("instructions not preserved: %s", result.Instructions)
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
			name:  "response.created",
			input: `{"type":"response.created","response":{"id":"resp_1","object":"response","created_at":1700000000,"model":"gpt-4o","status":"in_progress","output":[]}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStart {
					return errCheck("type", "start", string(e.Type))
				}
				if e.ID != "resp_1" {
					return errCheck("id", "resp_1", e.ID)
				}
				if e.Model != "gpt-4o" {
					return errCheck("model", "gpt-4o", e.Model)
				}
				return nil
			},
		},
		{
			name:  "response.output_text.delta",
			input: `{"type":"response.output_text.delta","output_index":0,"delta":"Hello"}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventDelta {
					return errCheck("type", "delta", string(e.Type))
				}
				if len(e.Choices) != 1 {
					return errCheck("choices", "1", "0")
				}
				if len(e.Choices[0].Delta.Content) != 1 {
					return errCheck("content", "1", "0")
				}
				tp, ok := e.Choices[0].Delta.Content[0].(uni.TextPart)
				if !ok {
					return errCheck("type", "TextPart", "other")
				}
				if tp.Text != "Hello" {
					return errCheck("text", "Hello", tp.Text)
				}
				return nil
			},
		},
		{
			name:  "response.function_call_arguments.delta",
			input: `{"type":"response.function_call_arguments.delta","output_index":0,"delta":"{\"q\":"}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventDelta {
					return errCheck("type", "delta", string(e.Type))
				}
				if len(e.Choices) != 1 {
					return errCheck("choices", "1", "0")
				}
				if len(e.Choices[0].Delta.ToolCalls) != 1 {
					return errCheck("tool_calls", "1", "0")
				}
				if e.Choices[0].Delta.ToolCalls[0].Arguments != "{\"q\":" {
					return errCheck("arguments", "{\"q\":", e.Choices[0].Delta.ToolCalls[0].Arguments)
				}
				return nil
			},
		},
		{
			name:  "response.completed",
			input: `{"type":"response.completed","response":{"id":"resp_1","object":"response","model":"gpt-4o","status":"completed","output":[],"usage":{"input_tokens":100,"output_tokens":50,"total_tokens":150}}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStop {
					return errCheck("type", "stop", string(e.Type))
				}
				if e.StopReason == nil || *e.StopReason != uni.StopReasonEndTurn {
					return errCheck("stop_reason", "end_turn", "nil")
				}
				if e.Usage == nil || e.Usage.TotalTokens != 150 {
					return errCheck("usage", "150", "nil")
				}
				return nil
			},
		},
		{
			name:  "error",
			input: `{"type":"error","error":{"type":"server_error","code":"internal_error","message":"Something went wrong"}}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventError {
					return errCheck("type", "error", string(e.Type))
				}
				if e.Error == nil {
					return errCheck("error", "non-nil", "nil")
				}
				if e.Error.Type != "server_error" {
					return errCheck("error.type", "server_error", e.Error.Type)
				}
				if e.Error.Message != "Something went wrong" {
					return errCheck("error.message", "Something went wrong", e.Error.Message)
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
	topK := int64(40)
	fp := 0.5
	params := &uni.RequestParams{
		Model:            "gpt-4o",
		MaxTokens:        int64Ptr(1024),
		TopK:             &topK,
		FrequencyPenalty: &fp,
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
	if !found["top_k"] {
		t.Fatal("expected top_k in lost fields")
	}
	if !found["frequency_penalty"] {
		t.Fatal("expected frequency_penalty in lost fields")
	}
}

func TestEncodeStreamEvent(t *testing.T) {
	a := New()

	event := &uni.StreamEvent{
		Type:  uni.StreamEventDelta,
		ID:    "resp_1",
		Model: "gpt-4o",
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

	var evt responseStreamEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if evt.Type != "response.output_text.delta" {
		t.Fatalf("expected type response.output_text.delta, got %s", evt.Type)
	}
	if evt.Delta != "Hello" {
		t.Fatalf("expected delta Hello, got %s", evt.Delta)
	}
}

func TestDecodeRequestStringInput(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"input": "Hello world"
	}`)

	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	if len(params.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(params.Messages))
	}
	if params.Messages[0].Role != uni.RoleUser {
		t.Fatalf("expected user, got %s", params.Messages[0].Role)
	}
	tp, ok := params.Messages[0].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart")
	}
	if tp.Text != "Hello world" {
		t.Fatalf("expected 'Hello world', got %s", tp.Text)
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
