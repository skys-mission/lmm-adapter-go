package claude

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeResponse(data json.RawMessage) (*uni.Response, *adapter.Report, error) {
	report := adapter.NewReport()

	var resp claudeMessageResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, report, fmt.Errorf("failed to unmarshal claude response: %w", err)
	}

	uniResp := &uni.Response{
		ID:           resp.ID,
		Model:        resp.Model,
		StopReason:   convertClaudeStopReason(resp.StopReason),
		StopSequence: resp.StopSequence,
		Usage: uni.Usage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			TotalTokens:              resp.Usage.InputTokens + resp.Usage.OutputTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		},
	}

	var assistantParts []uni.ContentPart
	var toolMessages []uni.Message

	for _, block := range resp.Content {
		switch block.Type {
		case "text":
			assistantParts = append(assistantParts, uni.TextPart{Text: block.Text})
		case "tool_use":
			assistantParts = append(assistantParts, uni.ToolUsePart{
				ToolCallID: block.ID,
				ToolName:   block.Name,
				Arguments:  block.Input,
			})
		case "thinking":
			assistantParts = append(assistantParts, uni.ThinkingPart{
				Thinking:  block.Thinking,
				Signature: block.Signature,
			})
		case "tool_result":
			var content []uni.ContentPart
			if block.Text != "" {
				content = append(content, uni.TextPart{Text: block.Text})
			}
			toolMessages = append(toolMessages, uni.ToolMessage(block.ToolUseID, content, block.IsError))
		}
	}

	if len(assistantParts) > 0 {
		uniResp.Messages = append(uniResp.Messages, uni.AssistantMessage(assistantParts...))
	}
	uniResp.Messages = append(uniResp.Messages, toolMessages...)

	return uniResp, report, nil
}

func encodeResponse(resp *uni.Response) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	claudeResp := claudeMessageResponse{
		ID:           resp.ID,
		Type:         "message",
		Role:         "assistant",
		Model:        resp.Model,
		StopReason:   convertUniStopReasonToClaude(resp.StopReason),
		StopSequence: resp.StopSequence,
		Usage: usage{
			InputTokens:              resp.Usage.InputTokens,
			OutputTokens:             resp.Usage.OutputTokens,
			CacheCreationInputTokens: resp.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     resp.Usage.CacheReadInputTokens,
		},
	}

	for _, msg := range resp.Messages {
		// Claude responses only contain assistant content blocks;
		// tool result messages belong in requests, not responses.
		if msg.Role != uni.RoleAssistant {
			continue
		}
		for _, part := range msg.Content {
			blocks, err := convertUniPartToClaude(part, report)
			if err != nil {
				return nil, report, fmt.Errorf("failed to convert part to claude block: %w", err)
			}
			claudeResp.Content = append(claudeResp.Content, blocks...)
		}
	}

	data, err := json.Marshal(claudeResp)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal claude response: %w", err)
	}
	return data, report, nil
}

func convertClaudeStopReason(reason string) uni.StopReason {
	switch reason {
	case "end_turn":
		return uni.StopReasonEndTurn
	case "max_tokens":
		return uni.StopReasonMaxTokens
	case "tool_use":
		return uni.StopReasonToolCalls
	case "stop_sequence":
		return uni.StopReasonStopSequence
	case "refusal":
		return uni.StopReasonRefusal
	default:
		return uni.StopReason(reason)
	}
}

func convertUniStopReasonToClaude(reason uni.StopReason) string {
	switch reason {
	case uni.StopReasonEndTurn:
		return "end_turn"
	case uni.StopReasonMaxTokens:
		return "max_tokens"
	case uni.StopReasonToolCalls:
		return "tool_use"
	case uni.StopReasonStopSequence:
		return "stop_sequence"
	case uni.StopReasonRefusal:
		return "refusal"
	case uni.StopReasonContentFilter:
		return "refusal"
	default:
		return string(reason)
	}
}
