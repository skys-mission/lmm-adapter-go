package openaichat

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestDecodeRequest(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"messages": [
			{"role": "system", "content": "You are helpful."},
			{"role": "user", "content": [
				{"type": "text", "text": "What is this?"},
				{"type": "image_url", "image_url": {"url": "https://example.com/img.png", "detail": "high"}}
			]},
			{"role": "assistant", "tool_calls": [
				{"id": "call_abc", "type": "function", "function": {"name": "search", "arguments": "{\"q\":\"test\"}"}}
			], "content": ""},
			{"role": "tool", "tool_call_id": "call_abc", "content": "search result"}
		],
		"tools": [
			{"type": "function", "function": {"name": "search", "description": "Search the web", "parameters": {"type": "object", "properties": {"q": {"type": "string"}}}}}
		],
		"tool_choice": "auto",
		"temperature": 0.7,
		"top_p": 0.95,
		"max_completion_tokens": 2048,
		"frequency_penalty": 0.1,
		"presence_penalty": 0.2,
		"stream": true,
		"seed": 42,
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
		t.Fatalf("expected max_tokens 2048, got %v", params.MaxTokens)
	}
	if params.FrequencyPenalty == nil || *params.FrequencyPenalty != 0.1 {
		t.Fatalf("expected frequency_penalty 0.1, got %v", params.FrequencyPenalty)
	}
	if params.PresencePenalty == nil || *params.PresencePenalty != 0.2 {
		t.Fatalf("expected presence_penalty 0.2, got %v", params.PresencePenalty)
	}
	if params.Seed == nil || *params.Seed != 42 {
		t.Fatalf("expected seed 42, got %v", params.Seed)
	}
	if !params.Stream {
		t.Fatal("expected stream true")
	}

	if len(params.Messages) != 4 {
		t.Fatalf("expected 4 messages, got %d", len(params.Messages))
	}

	if params.Messages[0].Role != uni.RoleSystem {
		t.Fatalf("expected system, got %s", params.Messages[0].Role)
	}
	stp, ok := params.Messages[0].Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart")
	}
	if stp.Text != "You are helpful." {
		t.Fatalf("expected system text, got %s", stp.Text)
	}

	if params.Messages[1].Role != uni.RoleUser {
		t.Fatalf("expected user, got %s", params.Messages[1].Role)
	}
	if len(params.Messages[1].Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(params.Messages[1].Content))
	}
	imgP, ok := params.Messages[1].Content[1].(uni.ImagePart)
	if !ok {
		t.Fatal("expected ImagePart")
	}
	if imgP.URL != "https://example.com/img.png" {
		t.Fatalf("expected image url, got %s", imgP.URL)
	}
	if imgP.Detail != "high" {
		t.Fatalf("expected detail high, got %s", imgP.Detail)
	}

	if params.Messages[2].Role != uni.RoleAssistant {
		t.Fatalf("expected assistant, got %s", params.Messages[2].Role)
	}
	tuP, ok := params.Messages[2].Content[0].(uni.ToolUsePart)
	if !ok {
		t.Fatal("expected ToolUsePart")
	}
	if tuP.ToolCallID != "call_abc" {
		t.Fatalf("expected call_abc, got %s", tuP.ToolCallID)
	}
	if tuP.ToolName != "search" {
		t.Fatalf("expected search, got %s", tuP.ToolName)
	}

	if params.Messages[3].Role != uni.RoleTool {
		t.Fatalf("expected tool, got %s", params.Messages[3].Role)
	}
	trP, ok := params.Messages[3].Content[0].(uni.ToolResultPart)
	if !ok {
		t.Fatal("expected ToolResultPart")
	}
	if trP.ToolCallID != "call_abc" {
		t.Fatalf("expected call_abc, got %s", trP.ToolCallID)
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
	fp := 0.1
	params := &uni.RequestParams{
		Model:            "gpt-4o",
		MaxTokens:        &maxTokens,
		Temperature:      &temp,
		FrequencyPenalty: &fp,
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

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if req.Model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", req.Model)
	}
	if req.MaxCompletionTokens == nil || *req.MaxCompletionTokens != 1024 {
		t.Fatalf("expected max_completion_tokens 1024, got %v", req.MaxCompletionTokens)
	}
	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "system" {
		t.Fatalf("expected system, got %s", req.Messages[0].Role)
	}
	if req.Messages[1].Role != "user" {
		t.Fatalf("expected user, got %s", req.Messages[1].Role)
	}
	if len(req.Tools) != 1 || req.Tools[0].Function.Name != "calc" {
		t.Fatalf("expected 1 tool, got %v", req.Tools)
	}
	if string(req.ToolChoice) != `"required"` {
		t.Fatalf("expected tool_choice required, got %s", string(req.ToolChoice))
	}
}

func TestDecodeResponse(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"id": "chatcmpl-abc",
		"object": "chat.completion",
		"created": 1700000000,
		"model": "gpt-4o",
		"choices": [
			{
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "The answer is 42.",
					"tool_calls": [
						{"id": "call_1", "type": "function", "function": {"name": "calc", "arguments": "{\"expr\":\"6*7\"}"}}
					]
				},
				"finish_reason": "tool_calls"
			}
		],
		"usage": {"prompt_tokens": 50, "completion_tokens": 30, "total_tokens": 80}
	}`)

	resp, _, err := a.DecodeResponse(data)
	if err != nil {
		t.Fatalf("DecodeResponse failed: %v", err)
	}

	if resp.ID != "chatcmpl-abc" {
		t.Fatalf("expected id chatcmpl-abc, got %s", resp.ID)
	}
	if resp.Model != "gpt-4o" {
		t.Fatalf("expected model gpt-4o, got %s", resp.Model)
	}
	if resp.StopReason != uni.StopReasonToolCalls {
		t.Fatalf("expected stop_reason tool_calls, got %s", resp.StopReason)
	}
	if resp.Usage.InputTokens != 50 {
		t.Fatalf("expected 50 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 30 {
		t.Fatalf("expected 30 output tokens, got %d", resp.Usage.OutputTokens)
	}
	if resp.Created != 1700000000 {
		t.Fatalf("expected created 1700000000, got %d", resp.Created)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}
	msg := resp.Messages[0]
	if msg.Role != uni.RoleAssistant {
		t.Fatalf("expected assistant, got %s", msg.Role)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
	}

	tp, ok := msg.Content[0].(uni.TextPart)
	if !ok {
		t.Fatal("expected TextPart")
	}
	if tp.Text != "The answer is 42." {
		t.Fatalf("expected text, got %s", tp.Text)
	}

	tup, ok := msg.Content[1].(uni.ToolUsePart)
	if !ok {
		t.Fatal("expected ToolUsePart")
	}
	if tup.ToolCallID != "call_1" {
		t.Fatalf("expected call_1, got %s", tup.ToolCallID)
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

	var out chatCompletionResponse
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.ID != "resp_test" {
		t.Fatalf("expected id resp_test, got %s", out.ID)
	}
	if out.Object != "chat.completion" {
		t.Fatalf("expected object chat.completion, got %s", out.Object)
	}
	if len(out.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(out.Choices))
	}
	if out.Choices[0].FinishReason != "stop" {
		t.Fatalf("expected finish_reason stop, got %s", out.Choices[0].FinishReason)
	}
	if out.Usage == nil || out.Usage.TotalTokens != 150 {
		t.Fatalf("expected total_tokens 150, got %v", out.Usage)
	}
}

func TestRequestRoundTrip(t *testing.T) {
	a := New()
	original := json.RawMessage(`{
		"model": "gpt-4o",
		"messages": [
			{"role": "user", "content": "Hello world"}
		],
		"temperature": 0.8,
		"max_completion_tokens": 512,
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

	var result chatCompletionRequest
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if result.Model != "gpt-4o" {
		t.Fatalf("model not preserved: %s", result.Model)
	}
	if result.Temperature == nil || *result.Temperature != 0.8 {
		t.Fatalf("temperature not preserved: %v", result.Temperature)
	}
	if result.MaxCompletionTokens == nil || *result.MaxCompletionTokens != 512 {
		t.Fatalf("max_completion_tokens not preserved: %v", result.MaxCompletionTokens)
	}
	if len(result.Messages) != 1 {
		t.Fatalf("messages not preserved: %d", len(result.Messages))
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
			name:  "role chunk",
			input: `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStart {
					return errCheck("type", "start", string(e.Type))
				}
				if e.ID != "chatcmpl-1" {
					return errCheck("id", "chatcmpl-1", e.ID)
				}
				if len(e.Choices) != 1 || e.Choices[0].Delta.Role != uni.RoleAssistant {
					return errCheck("role", "assistant", "missing")
				}
				return nil
			},
		},
		{
			name:  "text delta",
			input: `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
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
			name:  "tool_call delta",
			input: `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"search","arguments":"{\"q\":"}}]},"finish_reason":null}]}`,
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
				tc := e.Choices[0].Delta.ToolCalls[0]
				if tc.ToolCallID != "call_1" {
					return errCheck("tool_call_id", "call_1", tc.ToolCallID)
				}
				if tc.ToolName != "search" {
					return errCheck("tool_name", "search", tc.ToolName)
				}
				return nil
			},
		},
		{
			name:  "finish chunk",
			input: `{"id":"chatcmpl-1","object":"chat.completion.chunk","created":1700000000,"model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`,
			checkFn: func(e *uni.StreamEvent) error {
				if e.Type != uni.StreamEventStop {
					return errCheck("type", "stop", string(e.Type))
				}
				if e.StopReason == nil || *e.StopReason != uni.StopReasonEndTurn {
					return errCheck("stop_reason", "end_turn", "nil")
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
	params := &uni.RequestParams{
		Model:     "gpt-4o",
		MaxTokens: int64Ptr(1024),
		Messages: []uni.Message{
			uni.AssistantMessage(
				uni.ThinkingPart{Thinking: "Let me think about this..."},
				uni.TextPart{Text: "The answer."},
			),
		},
	}

	_, report, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	found := false
	for _, lf := range report.LostFields {
		if lf.Field == "thinking" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected thinking in lost fields")
	}
}

func TestEncodeStreamEvent(t *testing.T) {
	a := New()

	event := &uni.StreamEvent{
		Type:  uni.StreamEventDelta,
		ID:    "chatcmpl-1",
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

	var chunk chatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if chunk.Object != "chat.completion.chunk" {
		t.Fatalf("expected object chat.completion.chunk, got %s", chunk.Object)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content != "Hello" {
		t.Fatalf("expected content Hello, got %s", chunk.Choices[0].Delta.Content)
	}
}

func TestDecodeRequestToolChoiceSpecific(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "hi"}],
		"tool_choice": {"type": "function", "function": {"name": "my_tool"}}
	}`)

	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}

	if params.ToolChoice == nil {
		t.Fatal("expected tool_choice")
	}
	if params.ToolChoice.Type != uni.ToolChoiceSpecific {
		t.Fatalf("expected specific, got %s", params.ToolChoice.Type)
	}
	if params.ToolChoice.ToolName != "my_tool" {
		t.Fatalf("expected my_tool, got %s", params.ToolChoice.ToolName)
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

func TestEncodeRequest_ExtRestoration(t *testing.T) {
	a := New()
	ext := make(uni.ExtData)
	ext.Set("n", int64(2))
	ext.Set("user", "alice")
	ext["stream_options"] = json.RawMessage(`{"include_usage":true}`)
	ext["response_format"] = json.RawMessage(`{"type":"json_object"}`)

	params := &uni.RequestParams{
		Model:    "gpt-4o",
		Messages: []uni.Message{uni.UserMessage(uni.TextPart{Text: "hi"})},
		Ext:      ext,
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.N == nil || *req.N != 2 {
		t.Fatalf("expected n=2, got %v", req.N)
	}
	if req.User != "alice" {
		t.Fatalf("expected user=alice, got %s", req.User)
	}
	if req.StreamOptions == nil {
		t.Fatal("expected stream_options restored")
	}
	if req.ResponseFormat == nil {
		t.Fatal("expected response_format restored")
	}
}

func TestEncodeMessage_MultimediaUser(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model: "gpt-4o",
		Messages: []uni.Message{
			uni.UserMessage(
				uni.TextPart{Text: "What is this?"},
				uni.ImagePart{URL: "https://example.com/img.png", Detail: "high"},
				uni.AudioPart{Data: "base64audio", Format: "mp3"},
				uni.FilePart{Data: "base64file", Name: "notes.txt"},
			),
		},
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}

	var parts []contentPart
	if err := json.Unmarshal(req.Messages[0].Content, &parts); err != nil {
		t.Fatalf("unmarshal content parts: %v", err)
	}
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts, got %d", len(parts))
	}
	if parts[1].Type != "image_url" || parts[1].ImageURL.URL != "https://example.com/img.png" {
		t.Fatalf("unexpected image part: %+v", parts[1])
	}
	if parts[2].Type != "input_audio" {
		t.Fatalf("expected input_audio, got %s", parts[2].Type)
	}
	if parts[3].Type != "file" {
		t.Fatalf("expected file, got %s", parts[3].Type)
	}
}

func TestEncodeMessage_ImageWithMediaType(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model: "gpt-4o",
		Messages: []uni.Message{
			uni.UserMessage(uni.ImagePart{Data: "abc123", MediaType: "image/png"}),
		},
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	var parts []contentPart
	if err := json.Unmarshal(req.Messages[0].Content, &parts); err != nil {
		t.Fatalf("unmarshal content parts: %v", err)
	}
	if parts[0].ImageURL.URL != "data:image/png;base64,abc123" {
		t.Fatalf("expected data URI, got %s", parts[0].ImageURL.URL)
	}
}

func TestEncodeMessage_AssistantWithToolAndRefusal(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model: "gpt-4o",
		Messages: []uni.Message{
			uni.AssistantMessage(
				uni.TextPart{Text: "ok"},
				uni.ToolUsePart{ToolCallID: "call_1", ToolName: "search", Arguments: json.RawMessage(`{"q":"x"}`)},
				uni.RefusalPart{Refusal: "I can't"},
			),
		},
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "assistant" {
		t.Fatalf("expected assistant, got %s", req.Messages[0].Role)
	}
	if len(req.Messages[0].ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(req.Messages[0].ToolCalls))
	}
	if req.Messages[0].Refusal != "I can't" {
		t.Fatalf("expected refusal, got %s", req.Messages[0].Refusal)
	}
}

func TestEncodeMessage_ToolRoleNonTextLost(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model: "gpt-4o",
		Messages: []uni.Message{
			uni.Message{
				Role: uni.RoleTool,
				Content: []uni.ContentPart{
					uni.ToolResultPart{ToolCallID: "call_1", Content: []uni.ContentPart{
						uni.TextPart{Text: "ok"},
						uni.ImagePart{URL: "https://example.com/img.png"},
					}},
				},
			},
		},
	}

	_, report, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	found := false
	for _, lf := range report.LostFields {
		if lf.Field == "tool_result.content" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected tool_result.content in lost fields")
	}
}

func TestEncodeToolChoice_Branches(t *testing.T) {
	a := New()

	cases := []struct {
		name     string
		choice   uni.ToolChoice
		expected string
	}{
		{"auto", uni.ToolChoice{Type: uni.ToolChoiceAuto}, `"auto"`},
		{"required", uni.ToolChoice{Type: uni.ToolChoiceRequired}, `"required"`},
		{"none", uni.ToolChoice{Type: uni.ToolChoiceNone}, `"none"`},
		{"specific", uni.ToolChoice{Type: uni.ToolChoiceSpecific, ToolName: "foo"}, `{"function":{"name":"foo"},"type":"function"}`},
		{"default", uni.ToolChoice{Type: uni.ToolChoiceType("weird")}, `"auto"`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			params := &uni.RequestParams{
				Model:      "gpt-4o",
				ToolChoice: &c.choice,
				Messages:   []uni.Message{uni.UserMessage(uni.TextPart{Text: "hi"})},
			}
			data, _, err := a.EncodeRequest(params)
			if err != nil {
				t.Fatalf("EncodeRequest failed: %v", err)
			}
			var req chatCompletionRequest
			if err := json.Unmarshal(data, &req); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			// Normalize by round-tripping through json.Unmarshal to compare ignoring key order.
			var got map[string]any
			var want map[string]any
			if err := json.Unmarshal(req.ToolChoice, &got); err == nil {
				_ = json.Unmarshal([]byte(c.expected), &want)
				// Compare as re-marshaled strings.
				gotBytes, _ := json.Marshal(got)
				wantBytes, _ := json.Marshal(want)
				if string(gotBytes) != string(wantBytes) {
					t.Fatalf("expected %s, got %s", wantBytes, gotBytes)
				}
				return
			}
			var gotStr string
			if err := json.Unmarshal(req.ToolChoice, &gotStr); err != nil {
				t.Fatalf("tool_choice not string or object: %s", string(req.ToolChoice))
			}
			var wantStr string
			json.Unmarshal([]byte(c.expected), &wantStr)
			if gotStr != wantStr {
				t.Fatalf("expected %s, got %s", wantStr, gotStr)
			}
		})
	}
}

func TestDecodeContentPart_UnknownType(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": [{"type": "weird", "text": "fallback"}]}]
	}`)

	params, report, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}
	tp, ok := params.Messages[0].Content[0].(uni.TextPart)
	if !ok || tp.Text != "fallback" {
		t.Fatalf("expected fallback TextPart, got %+v", params.Messages[0].Content[0])
	}

	found := false
	for _, lf := range report.LostFields {
		if lf.Field == "content_part.type" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected content_part.type in lost fields")
	}
}

func TestDecodeRequest_StopAsString(t *testing.T) {
	a := New()
	data := json.RawMessage(`{
		"model": "gpt-4o",
		"messages": [{"role": "user", "content": "hi"}],
		"stop": "halt"
	}`)

	params, _, err := a.DecodeRequest(data)
	if err != nil {
		t.Fatalf("DecodeRequest failed: %v", err)
	}
	if len(params.StopSequences) != 1 || params.StopSequences[0] != "halt" {
		t.Fatalf("expected stop=[halt], got %v", params.StopSequences)
	}
}

func TestEncodeRequest_StopSequences(t *testing.T) {
	a := New()
	params := &uni.RequestParams{
		Model:         "gpt-4o",
		StopSequences: []string{"halt", "stop"},
		Messages:      []uni.Message{uni.UserMessage(uni.TextPart{Text: "hi"})},
	}

	data, _, err := a.EncodeRequest(params)
	if err != nil {
		t.Fatalf("EncodeRequest failed: %v", err)
	}

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var stops []string
	if err := json.Unmarshal(req.Stop, &stops); err != nil {
		t.Fatalf("unmarshal stop: %v", err)
	}
	if len(stops) != 2 || stops[0] != "halt" || stops[1] != "stop" {
		t.Fatalf("unexpected stop: %v", stops)
	}
}
