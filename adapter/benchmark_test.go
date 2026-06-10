package adapter_test

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/adapter/claude"
	"github.com/skys-mission/lmm-adapter-go/adapter/openaichat"
	"github.com/skys-mission/lmm-adapter-go/adapter/openairesp"
)

var (
	claudeRequest = json.RawMessage(`{
		"model": "claude-3-opus",
		"max_tokens": 1024,
		"messages": [
			{"role": "user", "content": "Hello, how are you?"},
			{"role": "assistant", "content": "I'm doing well, thank you!"},
			{"role": "user", "content": "What's the weather like?"}
		],
		"temperature": 0.7,
		"top_p": 0.9
	}`)

	openaiChatRequest = json.RawMessage(`{
		"model": "gpt-4",
		"messages": [
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "Hello, how are you?"},
			{"role": "assistant", "content": "I'm doing well, thank you!"},
			{"role": "user", "content": "What's the weather like?"}
		],
		"temperature": 0.7,
		"max_tokens": 1024
	}`)

	openaiRespRequest = json.RawMessage(`{
		"model": "gpt-4",
		"input": [
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "Hello"}]},
			{"type": "message", "role": "assistant", "content": [{"type": "output_text", "text": "Hi there!"}]},
			{"type": "message", "role": "user", "content": [{"type": "input_text", "text": "How are you?"}]}
		],
		"instructions": "You are a helpful assistant.",
		"temperature": 0.7,
		"max_output_tokens": 1024
	}`)

	claudeResponse = json.RawMessage(`{
		"id": "msg_123",
		"type": "message",
		"role": "assistant",
		"content": [
			{"type": "text", "text": "I'm doing well, thank you for asking!"}
		],
		"model": "claude-3-opus",
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 20}
	}`)

	openaiChatResponse = json.RawMessage(`{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1234567890,
		"model": "gpt-4",
		"choices": [{
			"index": 0,
			"message": {
				"role": "assistant",
				"content": "I'm doing well, thank you for asking!"
			},
			"finish_reason": "stop"
		}],
		"usage": {"prompt_tokens": 10, "completion_tokens": 20, "total_tokens": 30}
	}`)

	openaiRespResponse = json.RawMessage(`{
		"id": "resp_123",
		"object": "response",
		"created_at": 1234567890,
		"model": "gpt-4",
		"output": [{
			"type": "message",
			"role": "assistant",
			"content": [{"type": "output_text", "text": "I'm doing well!"}]
		}],
		"status": "completed",
		"usage": {"input_tokens": 10, "output_tokens": 20, "total_tokens": 30}
	}`)
)

func BenchmarkClaudeDecodeRequest(b *testing.B) {
	a := claude.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeRequest(claudeRequest)
	}
}

func BenchmarkClaudeEncodeRequest(b *testing.B) {
	a := claude.New()
	req, _, _ := a.DecodeRequest(claudeRequest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeRequest(req)
	}
}

func BenchmarkClaudeDecodeResponse(b *testing.B) {
	a := claude.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeResponse(claudeResponse)
	}
}

func BenchmarkClaudeEncodeResponse(b *testing.B) {
	a := claude.New()
	resp, _, _ := a.DecodeResponse(claudeResponse)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeResponse(resp)
	}
}

func BenchmarkOpenAIChatDecodeRequest(b *testing.B) {
	a := openaichat.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeRequest(openaiChatRequest)
	}
}

func BenchmarkOpenAIChatEncodeRequest(b *testing.B) {
	a := openaichat.New()
	req, _, _ := a.DecodeRequest(openaiChatRequest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeRequest(req)
	}
}

func BenchmarkOpenAIChatDecodeResponse(b *testing.B) {
	a := openaichat.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeResponse(openaiChatResponse)
	}
}

func BenchmarkOpenAIChatEncodeResponse(b *testing.B) {
	a := openaichat.New()
	resp, _, _ := a.DecodeResponse(openaiChatResponse)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeResponse(resp)
	}
}

func BenchmarkOpenAIRespDecodeRequest(b *testing.B) {
	a := openairesp.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeRequest(openaiRespRequest)
	}
}

func BenchmarkOpenAIRespEncodeRequest(b *testing.B) {
	a := openairesp.New()
	req, _, _ := a.DecodeRequest(openaiRespRequest)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeRequest(req)
	}
}

func BenchmarkOpenAIRespDecodeResponse(b *testing.B) {
	a := openairesp.New()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.DecodeResponse(openaiRespResponse)
	}
}

func BenchmarkOpenAIRespEncodeResponse(b *testing.B) {
	a := openairesp.New()
	resp, _, _ := a.DecodeResponse(openaiRespResponse)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = a.EncodeResponse(resp)
	}
}

func BenchmarkCrossProtocolClaudeToOpenAIChat(b *testing.B) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.ConvertRequest(adapter.ProtocolClaudeMessages, adapter.ProtocolOpenAIChat, claudeRequest)
	}
}

func BenchmarkCrossProtocolOpenAIChatToClaude(b *testing.B) {
	c := adapter.NewConverter(
		adapter.WithAdapter(claude.New()),
		adapter.WithAdapter(openaichat.New()),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.ConvertRequest(adapter.ProtocolOpenAIChat, adapter.ProtocolClaudeMessages, openaiChatRequest)
	}
}

func BenchmarkCrossProtocolOpenAIChatToResponses(b *testing.B) {
	c := adapter.NewConverter(
		adapter.WithAdapter(openaichat.New()),
		adapter.WithAdapter(openairesp.New()),
	)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = c.ConvertRequest(adapter.ProtocolOpenAIChat, adapter.ProtocolOpenAIResponses, openaiChatRequest)
	}
}
