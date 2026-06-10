package token

import (
	"encoding/json"
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestEstimateTokens_English(t *testing.T) {
	tokens := EstimateTokens("Hello, world!")
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
	// "Hello, world!" = 13 chars, roughly 13/4 ≈ 3 tokens
	expected := int64(3)
	if tokens != expected && tokens != expected+1 {
		t.Fatalf("expected ~%d tokens, got %d", expected, tokens)
	}
}

func TestEstimateTokens_Chinese(t *testing.T) {
	tokens := EstimateTokens("你好世界")
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
	// 4 CJK chars / 1.5 ≈ 2-3 tokens
	expected := int64(2)
	if tokens != expected && tokens != expected+1 {
		t.Fatalf("expected ~%d tokens, got %d", expected, tokens)
	}
}

func TestEstimateTokens_Japanese(t *testing.T) {
	tokens := EstimateTokens("こんにちは世界")
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
}

func TestEstimateTokens_Korean(t *testing.T) {
	tokens := EstimateTokens("안녕하세요")
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
}

func TestEstimateTokens_Mixed(t *testing.T) {
	// English + Chinese mixed text
	tokens := EstimateTokens("Hello 世界")
	if tokens <= 0 {
		t.Fatalf("expected positive token count, got %d", tokens)
	}
}

func TestEstimateTokens_Empty(t *testing.T) {
	tokens := EstimateTokens("")
	if tokens != 0 {
		t.Fatalf("expected 0 tokens for empty string, got %d", tokens)
	}
}

func TestEstimateTokens_LongEnglish(t *testing.T) {
	text := "This is a longer piece of text that should have more tokens than a short piece of text."
	tokens := EstimateTokens(text)
	if tokens <= 5 {
		t.Fatalf("expected >5 tokens for long text, got %d", tokens)
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	msg := uni.UserMessage(uni.TextPart{Text: "Hello, world!"})
	tokens := EstimateMessageTokens(msg)
	if tokens <= MessageOverhead {
		t.Fatalf("expected >%d tokens (overhead), got %d", MessageOverhead, tokens)
	}
}

func TestEstimateMessageTokens_MultipleParts(t *testing.T) {
	msg := uni.UserMessage(
		uni.TextPart{Text: "What is in this image?"},
		uni.ImagePart{URL: "https://example.com/img.png", Detail: "high"},
	)
	tokens := EstimateMessageTokens(msg)
	if tokens < 85 {
		t.Fatalf("expected >=85 tokens (image high detail), got %d", tokens)
	}
}

func TestEstimateContentPartTokens_TextPart(t *testing.T) {
	tokens := EstimateContentPartTokens(uni.TextPart{Text: "Hello"})
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_ImagePart(t *testing.T) {
	// Default detail
	tokens := EstimateContentPartTokens(uni.ImagePart{URL: "https://example.com/img.png"})
	if tokens != 85 {
		t.Fatalf("expected 85 tokens for default detail, got %d", tokens)
	}

	// High detail
	tokens = EstimateContentPartTokens(uni.ImagePart{URL: "https://example.com/img.png", Detail: "high"})
	if tokens != 170 {
		t.Fatalf("expected 170 tokens for high detail, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_AudioPart(t *testing.T) {
	tokens := EstimateContentPartTokens(uni.AudioPart{Data: "base64data", Format: "wav"})
	if tokens != 50 {
		t.Fatalf("expected 50 tokens for audio, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_FilePart(t *testing.T) {
	// File with URL only
	tokens := EstimateContentPartTokens(uni.FilePart{URL: "https://example.com/doc.pdf", Name: "doc.pdf"})
	if tokens != 50 {
		t.Fatalf("expected 50 tokens for URL-based file, got %d", tokens)
	}

	// File with data
	tokens = EstimateContentPartTokens(uni.FilePart{Data: "file content here", Name: "doc.txt"})
	if tokens <= 0 {
		t.Fatalf("expected positive tokens for file with data, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_ToolUsePart(t *testing.T) {
	toolUse := uni.ToolUsePart{
		ToolCallID: "call_1",
		ToolName:   "get_weather",
		Arguments:  json.RawMessage(`{"city":"Tokyo"}`),
	}
	tokens := EstimateContentPartTokens(toolUse)
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_ToolResultPart(t *testing.T) {
	toolResult := uni.ToolResultPart{
		ToolCallID: "call_1",
		Content:    []uni.ContentPart{uni.TextPart{Text: "Sunny, 25°C"}},
	}
	tokens := EstimateContentPartTokens(toolResult)
	if tokens <= 2 {
		t.Fatalf("expected >2 tokens, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_ThinkingPart(t *testing.T) {
	tokens := EstimateContentPartTokens(uni.ThinkingPart{Thinking: "Let me think about this carefully."})
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateContentPartTokens_RefusalPart(t *testing.T) {
	tokens := EstimateContentPartTokens(uni.RefusalPart{Refusal: "I cannot help with that."})
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateRequestTokens(t *testing.T) {
	req := &uni.RequestParams{
		Model: "gpt-4o",
		Messages: []uni.Message{
			uni.SystemMessage(uni.TextPart{Text: "You are helpful."}),
			uni.UserMessage(uni.TextPart{Text: "Hello"}),
		},
		Tools: []uni.Tool{
			{
				Name:        "search",
				Description: "Search the web",
				InputSchema: json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`),
			},
		},
	}

	tokens := EstimateRequestTokens(req)
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateToolTokens(t *testing.T) {
	tool := uni.Tool{
		Name:        "get_weather",
		Description: "Get weather for a city",
		InputSchema: json.RawMessage(`{"type":"object"}`),
	}

	tokens := EstimateToolTokens(tool)
	// overhead(7) + name tokens + description tokens + schema tokens
	if tokens <= 7 {
		t.Fatalf("expected >7 tokens, got %d", tokens)
	}
}

func TestEstimateResponseTokens(t *testing.T) {
	resp := &uni.Response{
		Messages: []uni.Message{
			uni.AssistantMessage(uni.TextPart{Text: "The answer is 42."}),
		},
	}
	tokens := EstimateResponseTokens(resp)
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestEstimateStreamTokens(t *testing.T) {
	event := &uni.StreamEvent{
		Type: uni.StreamEventDelta,
		Choices: []uni.StreamChoice{
			{
				Index: 0,
				Delta: uni.StreamDelta{
					Content: []uni.ContentPart{uni.TextPart{Text: "Hello"}},
				},
			},
		},
	}
	tokens := EstimateStreamTokens(event)
	if tokens <= 0 {
		t.Fatalf("expected positive tokens, got %d", tokens)
	}
}

func TestTruncateToTokenLimit_WithinLimit(t *testing.T) {
	text := "Hello world"
	result := TruncateToTokenLimit(text, 100)
	if result != text {
		t.Fatalf("expected unchanged text, got %q", result)
	}
}

func TestTruncateToTokenLimit_ExceedsLimit(t *testing.T) {
	text := "This is a very long piece of text that should definitely exceed the token limit we set"
	result := TruncateToTokenLimit(text, 3) // Very small limit
	if len(result) >= len(text) {
		t.Fatalf("expected truncated text, got same length")
	}
	if len(result) == 0 {
		t.Fatal("expected non-empty truncated text")
	}
}

func TestTruncateToTokenLimit_ZeroLimit(t *testing.T) {
	result := TruncateToTokenLimit("Hello", 0)
	if result != "" {
		t.Fatalf("expected empty string for zero limit, got %q", result)
	}
}

func TestTruncateToTokenLimit_NegativeLimit(t *testing.T) {
	result := TruncateToTokenLimit("Hello", -1)
	if result != "" {
		t.Fatalf("expected empty string for negative limit, got %q", result)
	}
}

func TestTruncateToTokenLimit_EmptyText(t *testing.T) {
	result := TruncateToTokenLimit("", 100)
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestTruncateToTokenLimit_WordBoundary(t *testing.T) {
	// Text where token limit falls mid-word
	text := "hello world foo bar baz qux"
	result := TruncateToTokenLimit(text, 2)
	if result == text {
		t.Fatalf("expected truncation, but got full text")
	}
	// Should end with "..."
	if len(result) < 3 || result[len(result)-3:] != "..." {
		t.Fatalf("expected '...' suffix, got %q", result)
	}
}

func TestCharsPerTokenConstants(t *testing.T) {
	if CharsPerTokenEnglish <= 0 {
		t.Fatal("CharsPerTokenEnglish must be positive")
	}
	if CharsPerTokenCJK <= 0 {
		t.Fatal("CharsPerTokenCJK must be positive")
	}
	if CharsPerTokenCJK >= CharsPerTokenEnglish {
		t.Fatal("CJK chars per token should be less than English (more tokens per char for CJK)")
	}
	if MessageOverhead <= 0 {
		t.Fatal("MessageOverhead must be positive")
	}
}

func TestIsCJK(t *testing.T) {
	tests := []struct {
		char rune
		cjk  bool
	}{
		{'a', false},
		{'1', false},
		{' ', false},
		{'\n', false},
		{'你', true},  // Chinese
		{'あ', true}, // Japanese Hiragana
		{'カ', true}, // Japanese Katakana
		{'한', true}, // Korean Hangul
	}

	for _, tt := range tests {
		got := isCJK(tt.char)
		if got != tt.cjk {
			t.Errorf("isCJK(%q) = %v, expected %v", tt.char, got, tt.cjk)
		}
	}
}
