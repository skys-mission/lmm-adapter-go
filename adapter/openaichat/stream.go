package openaichat

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *adapter.Report, error) {
	report := adapter.NewReport()

	var chunk chatCompletionChunk
	if err := json.Unmarshal(data, &chunk); err != nil {
		return nil, report, fmt.Errorf("unmarshal chunk: %w", err)
	}

	event := &uni.StreamEvent{
		ID:      chunk.ID,
		Created: chunk.Created,
		Model:   chunk.Model,
	}

	if chunk.Usage != nil {
		event.Usage = &uni.Usage{
			InputTokens:  chunk.Usage.PromptTokens,
			OutputTokens: chunk.Usage.CompletionTokens,
			TotalTokens:  chunk.Usage.TotalTokens,
		}
	}

	if len(chunk.Choices) == 0 {
		if event.Usage != nil {
			event.Type = uni.StreamEventDelta
		}
		return event, report, nil
	}

	choices := make([]uni.StreamChoice, 0, len(chunk.Choices))
	hasRole := false
	hasFinish := false
	isFirst := false

	for _, c := range chunk.Choices {
		sc := uni.StreamChoice{
			Index: c.Index,
		}

		if c.Delta.Role != "" {
			hasRole = true
			sc.Delta.Role = uni.Role(c.Delta.Role)
			isFirst = true
		}

		if c.Delta.Content != "" {
			sc.Delta.Content = []uni.ContentPart{uni.TextPart{Text: c.Delta.Content}}
		}

		if c.Delta.Refusal != "" {
			sc.Delta.Content = append(sc.Delta.Content, uni.RefusalPart{Refusal: c.Delta.Refusal})
		}

		if len(c.Delta.ToolCalls) > 0 {
			deltas := make([]uni.ToolCallDelta, 0, len(c.Delta.ToolCalls))
			for _, tc := range c.Delta.ToolCalls {
				deltas = append(deltas, uni.ToolCallDelta{
					Index:      tc.Index,
					ToolCallID: tc.ID,
					ToolName:   tc.Function.Name,
					Arguments:  tc.Function.Arguments,
				})
			}
			sc.Delta.ToolCalls = deltas
		}

		if c.FinishReason != "" {
			hasFinish = true
			reason := mapFinishReason(c.FinishReason)
			sc.FinishReason = &reason
		}

		choices = append(choices, sc)
	}

	event.Choices = choices

	if hasFinish {
		event.Type = uni.StreamEventStop
		if len(choices) > 0 && choices[0].FinishReason != nil {
			event.StopReason = choices[0].FinishReason
		}
	} else if isFirst && hasRole {
		event.Type = uni.StreamEventStart
	} else {
		event.Type = uni.StreamEventDelta
	}

	return event, report, nil
}

func encodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	chunk := chatCompletionChunk{
		ID:      event.ID,
		Object:  "chat.completion.chunk",
		Created: event.Created,
		Model:   event.Model,
	}

	if event.Usage != nil {
		chunk.Usage = &usage{
			PromptTokens:     event.Usage.InputTokens,
			CompletionTokens: event.Usage.OutputTokens,
			TotalTokens:      event.Usage.TotalTokens,
		}
	}

	if event.Ext != nil {
		if event.Ext.Has("system_fingerprint") {
			var fp string
			if err := event.Ext.Get("system_fingerprint", &fp); err == nil {
				chunk.SystemFingerprint = fp
			}
		}
	}

	choices := make([]chunkChoice, 0, len(event.Choices))
	for _, sc := range event.Choices {
		cc := chunkChoice{
			Index: sc.Index,
		}

		if sc.Delta.Role != "" {
			cc.Delta.Role = string(sc.Delta.Role)
		}

		for _, part := range sc.Delta.Content {
			switch p := part.(type) {
			case uni.TextPart:
				cc.Delta.Content += p.Text
			case uni.RefusalPart:
				cc.Delta.Refusal += p.Refusal
			case uni.ThinkingPart:
				report.AddLostField("uni", "thinking", "not supported in OpenAI Chat Completions stream")
			}
		}

		if len(sc.Delta.ToolCalls) > 0 {
			deltas := make([]toolCallDelta, 0, len(sc.Delta.ToolCalls))
			for _, tc := range sc.Delta.ToolCalls {
				deltas = append(deltas, toolCallDelta{
					Index: tc.Index,
					ID:    tc.ToolCallID,
					Type:  "function",
					Function: functionCallData{
						Name:      tc.ToolName,
						Arguments: tc.Arguments,
					},
				})
			}
			cc.Delta.ToolCalls = deltas
		}

		if sc.FinishReason != nil {
			cc.FinishReason = reverseMapFinishReason(*sc.FinishReason)
		}

		choices = append(choices, cc)
	}
	chunk.Choices = choices

	data, err := json.Marshal(chunk)
	if err != nil {
		return nil, report, fmt.Errorf("marshal chunk: %w", err)
	}
	return data, report, nil
}
