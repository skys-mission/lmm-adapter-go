package token

import (
	"strings"
	"unicode"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

const (
	// Average characters per token for English text
	CharsPerTokenEnglish = 4.0
	// Average characters per token for CJK text
	CharsPerTokenCJK = 1.5
	// Overhead tokens per message for role/formatting
	MessageOverhead = 4
)

func EstimateTokens(text string) int64 {
	if text == "" {
		return 0
	}

	var cjkChars, otherChars int64
	for _, r := range text {
		if isCJK(r) {
			cjkChars++
		} else if !unicode.IsSpace(r) {
			otherChars++
		}
	}

	cjkTokens := int64(float64(cjkChars) / CharsPerTokenCJK)
	otherTokens := int64(float64(otherChars) / CharsPerTokenEnglish)

	return cjkTokens + otherTokens
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}

func EstimateMessageTokens(msg uni.Message) int64 {
	tokens := int64(MessageOverhead) // Role overhead

	for _, part := range msg.Content {
		tokens += EstimateContentPartTokens(part)
	}

	return tokens
}

func EstimateContentPartTokens(part uni.ContentPart) int64 {
	switch p := part.(type) {
	case uni.TextPart:
		return EstimateTokens(p.Text)
	case uni.ThinkingPart:
		return EstimateTokens(p.Thinking)
	case uni.RefusalPart:
		return EstimateTokens(p.Refusal)
	case uni.ToolUsePart:
		tokens := EstimateTokens(p.ToolName)
		tokens += EstimateTokens(string(p.Arguments))
		return tokens
	case uni.ToolResultPart:
		tokens := int64(2) // Tool result overhead
		for _, c := range p.Content {
			tokens += EstimateContentPartTokens(c)
		}
		return tokens
	case uni.ImagePart:
		// Rough estimate: 85 tokens for low detail, 170 for high detail
		if p.Detail == "high" {
			return 170
		}
		return 85
	case uni.AudioPart:
		// Rough estimate: ~50 tokens per audio segment
		return 50
	case uni.FilePart:
		// File tokens depend on content
		if p.Data != "" {
			return EstimateTokens(p.Data)
		}
		return 50 // Default estimate for URL-based files
	default:
		return 10
	}
}

func EstimateRequestTokens(req *uni.RequestParams) int64 {
	tokens := int64(0)

	for _, msg := range req.Messages {
		tokens += EstimateMessageTokens(msg)
	}

	for _, tool := range req.Tools {
		tokens += EstimateToolTokens(tool)
	}

	return tokens
}

func EstimateToolTokens(tool uni.Tool) int64 {
	tokens := EstimateTokens(tool.Name)
	tokens += EstimateTokens(tool.Description)
	tokens += EstimateTokens(string(tool.InputSchema))
	return tokens + 7 // Tool definition overhead
}

func EstimateResponseTokens(resp *uni.Response) int64 {
	tokens := int64(0)

	for _, msg := range resp.Messages {
		tokens += EstimateMessageTokens(msg)
	}

	return tokens
}

func EstimateStreamTokens(event *uni.StreamEvent) int64 {
	tokens := int64(0)

	for _, choice := range event.Choices {
		for _, part := range choice.Delta.Content {
			tokens += EstimateContentPartTokens(part)
		}
		for _, tc := range choice.Delta.ToolCalls {
			tokens += EstimateTokens(tc.ToolName)
			tokens += EstimateTokens(tc.Arguments)
		}
	}

	return tokens
}

func TruncateToTokenLimit(text string, maxTokens int64) string {
	if maxTokens <= 0 {
		return ""
	}

	estimatedTokens := EstimateTokens(text)
	if estimatedTokens <= maxTokens {
		return text
	}

	// Rough truncation based on character estimate.
	// Detect the dominant character type to pick the right ratio.
	charsPerToken := CharsPerTokenEnglish
	var cjkCount int
	for _, r := range text {
		if isCJK(r) {
			cjkCount++
		}
	}
	if cjkCount > len(text)/2 {
		charsPerToken = CharsPerTokenCJK
	}

	maxChars := int(float64(maxTokens) * charsPerToken)
	if maxChars >= len(text) {
		return text
	}

	// Try to truncate at word boundary
	truncated := text[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > maxChars*8/10 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
