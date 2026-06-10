package openaichat

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeResponse(data json.RawMessage) (*uni.Response, *adapter.Report, error) {
	report := adapter.NewReport()

	var resp chatCompletionResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, report, fmt.Errorf("unmarshal response: %w", err)
	}

	result := &uni.Response{
		ID:      resp.ID,
		Model:   resp.Model,
		Created: resp.Created,
	}

	if resp.Usage != nil {
		result.Usage = uni.Usage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	msgs := make([]uni.Message, 0, len(resp.Choices))
	for i, c := range resp.Choices {
		msg, msgReport, err := decodeChoiceMessage(c)
		if err != nil {
			return nil, report, fmt.Errorf("choice[%d]: %w", i, err)
		}
		report.Merge(msgReport)
		msgs = append(msgs, msg)

		if i == 0 {
			result.StopReason = mapFinishReason(c.FinishReason)
		}
	}
	result.Messages = msgs

	ext := make(uni.ExtData)
	if resp.SystemFingerprint != "" {
		if err := ext.Set("system_fingerprint", resp.SystemFingerprint); err != nil {
			return nil, report, fmt.Errorf("set ext system_fingerprint: %w", err)
		}
	}
	if resp.ServiceTier != "" {
		if err := ext.Set("service_tier", resp.ServiceTier); err != nil {
			return nil, report, fmt.Errorf("set ext service_tier: %w", err)
		}
	}
	if resp.Usage != nil && resp.Usage.CompletionDetails != nil && resp.Usage.CompletionDetails.ReasoningTokens > 0 {
		if err := ext.Set("reasoning_tokens", resp.Usage.CompletionDetails.ReasoningTokens); err != nil {
			return nil, report, fmt.Errorf("set ext reasoning_tokens: %w", err)
		}
	}
	if len(ext) > 0 {
		result.Ext = ext
	}

	return result, report, nil
}

func decodeChoiceMessage(c choice) (uni.Message, *adapter.Report, error) {
	report := adapter.NewReport()
	msg := uni.Message{Role: uni.RoleAssistant}

	content, contentReport, err := decodeContent(c.Message.Content)
	if err != nil {
		return msg, report, fmt.Errorf("decode content: %w", err)
	}
	report.Merge(contentReport)

	var parts []uni.ContentPart
	parts = append(parts, content...)

	for _, tc := range c.Message.ToolCalls {
		parts = append(parts, uni.ToolUsePart{
			ToolCallID: tc.ID,
			ToolName:   tc.Function.Name,
			Arguments:  json.RawMessage(tc.Function.Arguments),
		})
	}

	if c.Message.Refusal != "" {
		parts = append(parts, uni.RefusalPart{Refusal: c.Message.Refusal})
	}

	msg.Content = parts
	return msg, report, nil
}

func mapFinishReason(reason string) uni.StopReason {
	switch reason {
	case "stop":
		return uni.StopReasonEndTurn
	case "length":
		return uni.StopReasonMaxTokens
	case "tool_calls":
		return uni.StopReasonToolCalls
	case "content_filter":
		return uni.StopReasonContentFilter
	default:
		return uni.StopReason(reason)
	}
}

func encodeResponse(resp *uni.Response) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	oaiResp := chatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created,
		Model:   resp.Model,
	}

	if resp.Usage != (uni.Usage{}) {
		oaiResp.Usage = &usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	if resp.Ext != nil {
		if resp.Ext.Has("system_fingerprint") {
			var fp string
			if err := resp.Ext.Get("system_fingerprint", &fp); err == nil {
				oaiResp.SystemFingerprint = fp
			}
		}
		if resp.Ext.Has("service_tier") {
			var st string
			if err := resp.Ext.Get("service_tier", &st); err == nil {
				oaiResp.ServiceTier = st
			}
		}
		if resp.Ext.Has("reasoning_tokens") {
			var rt int64
			if err := resp.Ext.Get("reasoning_tokens", &rt); err == nil {
				if oaiResp.Usage == nil {
					oaiResp.Usage = &usage{}
				}
				oaiResp.Usage.CompletionDetails = &completionTokenDetails{
					ReasoningTokens: rt,
				}
			}
		}
	}

	choices := make([]choice, 0, len(resp.Messages))
	for i, msg := range resp.Messages {
		cm, msgReport, err := encodeMessage(msg)
		if err != nil {
			return nil, report, fmt.Errorf("encode message[%d]: %w", i, err)
		}
		report.Merge(msgReport)
		c := choice{
			Index:        i,
			Message:      cm,
			FinishReason: reverseMapFinishReason(resp.StopReason),
		}
		choices = append(choices, c)
	}
	oaiResp.Choices = choices

	data, err := json.Marshal(oaiResp)
	if err != nil {
		return nil, report, fmt.Errorf("marshal response: %w", err)
	}
	return data, report, nil
}

func reverseMapFinishReason(reason uni.StopReason) string {
	switch reason {
	case uni.StopReasonEndTurn:
		return "stop"
	case uni.StopReasonMaxTokens:
		return "length"
	case uni.StopReasonToolCalls:
		return "tool_calls"
	case uni.StopReasonContentFilter:
		return "content_filter"
	default:
		return string(reason)
	}
}
