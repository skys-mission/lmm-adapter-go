package claude

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *adapter.Report, error) {
	report := adapter.NewReport()

	var event claudeStreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, report, fmt.Errorf("failed to unmarshal claude stream event: %w", err)
	}

	switch event.Type {
	case "message_start":
		return decodeMessageStart(event, report)
	case "content_block_start":
		return decodeContentBlockStart(event, report)
	case "content_block_delta":
		return decodeContentBlockDelta(event, report)
	case "content_block_stop":
		return decodeContentBlockStop(event)
	case "message_delta":
		return decodeMessageDelta(event, report)
	case "message_stop":
		return &uni.StreamEvent{Type: uni.StreamEventStop}, report, nil
	case "error":
		se := &uni.StreamEvent{Type: uni.StreamEventError}
		if event.Error != nil {
			se.Error = &uni.StreamError{
				Type:    event.Error.Type,
				Message: event.Error.Message,
			}
		} else {
			se.Error = &uni.StreamError{
				Type:    "error",
				Message: string(data),
			}
		}
		return se, report, nil
	default:
		return &uni.StreamEvent{
			Type: uni.StreamEventType(event.Type),
		}, report, nil
	}
}

func decodeMessageStart(event claudeStreamEvent, report *adapter.Report) (*uni.StreamEvent, *adapter.Report, error) {
	se := &uni.StreamEvent{
		Type: uni.StreamEventStart,
	}
	if event.Message != nil {
		se.ID = event.Message.ID
		se.Model = event.Message.Model
	}
	return se, report, nil
}

func decodeContentBlockStart(event claudeStreamEvent, report *adapter.Report) (*uni.StreamEvent, *adapter.Report, error) {
	se := &uni.StreamEvent{
		Type: uni.StreamEventContentStart,
	}

	if event.ContentBlock != nil {
		var parts []uni.ContentPart
		switch event.ContentBlock.Type {
		case "text":
			parts = append(parts, uni.TextPart{Text: event.ContentBlock.Text})
		case "tool_use":
			parts = append(parts, uni.ToolUsePart{
				ToolCallID: event.ContentBlock.ID,
				ToolName:   event.ContentBlock.Name,
				Arguments:  event.ContentBlock.Input,
			})
		case "thinking":
			parts = append(parts, uni.ThinkingPart{
				Thinking:  event.ContentBlock.Thinking,
				Signature: event.ContentBlock.Signature,
			})
		}
		se.Choices = []uni.StreamChoice{
			{
				Index: event.Index,
				Delta: uni.StreamDelta{Content: parts},
			},
		}
	}

	return se, report, nil
}

func decodeContentBlockDelta(event claudeStreamEvent, report *adapter.Report) (*uni.StreamEvent, *adapter.Report, error) {
	se := &uni.StreamEvent{
		Type: uni.StreamEventDelta,
	}

	if event.Delta != nil {
		var parts []uni.ContentPart
		var toolDeltas []uni.ToolCallDelta

		switch event.Delta.Type {
		case "text_delta":
			parts = append(parts, uni.TextPart{Text: event.Delta.Text})
		case "input_json_delta":
			toolDeltas = append(toolDeltas, uni.ToolCallDelta{
				Index:     event.Index,
				Arguments: event.Delta.PartialJSON,
			})
		case "thinking_delta":
			parts = append(parts, uni.ThinkingPart{Thinking: event.Delta.Thinking})
		case "signature_delta":
			parts = append(parts, uni.ThinkingPart{Signature: event.Delta.Signature})
		}

		se.Choices = []uni.StreamChoice{
			{
				Index: event.Index,
				Delta: uni.StreamDelta{
					Content:   parts,
					ToolCalls: toolDeltas,
				},
			},
		}
	}

	return se, report, nil
}

func decodeContentBlockStop(event claudeStreamEvent) (*uni.StreamEvent, *adapter.Report, error) {
	report := adapter.NewReport()
	return &uni.StreamEvent{
		Type: uni.StreamEventContentStop,
		Choices: []uni.StreamChoice{
			{Index: event.Index},
		},
	}, report, nil
}

func decodeMessageDelta(event claudeStreamEvent, report *adapter.Report) (*uni.StreamEvent, *adapter.Report, error) {
	se := &uni.StreamEvent{
		Type: uni.StreamEventStop,
	}

	// stop_reason and stop_sequence may be at the top level or nested inside delta.
	stopReason := event.StopReason
	stopSequence := event.StopSequence
	if event.Delta != nil {
		if stopReason == "" {
			stopReason = event.Delta.StopReason
		}
		if stopSequence == "" {
			stopSequence = event.Delta.StopSequence
		}
	}
	se.StopSequence = stopSequence

	if stopReason != "" {
		reason := convertClaudeStopReason(stopReason)
		se.StopReason = &reason
	}

	if event.Usage != nil {
		se.Usage = &uni.Usage{
			InputTokens:              event.Usage.InputTokens,
			OutputTokens:             event.Usage.OutputTokens,
			TotalTokens:              event.Usage.InputTokens + event.Usage.OutputTokens,
			CacheCreationInputTokens: event.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     event.Usage.CacheReadInputTokens,
		}
	}

	return se, report, nil
}

func encodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	switch event.Type {
	case uni.StreamEventStart:
		return encodeStreamStart(event, report)
	case uni.StreamEventContentStart:
		return encodeStreamContentStart(event, report)
	case uni.StreamEventDelta:
		return encodeStreamDelta(event, report)
	case uni.StreamEventContentStop:
		return encodeStreamContentStop(event, report)
	case uni.StreamEventStop:
		return encodeStreamStop(event, report)
	case uni.StreamEventError:
		return encodeStreamError(event, report)
	default:
		data, err := json.Marshal(claudeStreamEvent{Type: string(event.Type)})
		if err != nil {
			return nil, report, fmt.Errorf("failed to marshal stream event: %w", err)
		}
		return data, report, nil
	}
}

func encodeStreamStart(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	ce := claudeStreamEvent{
		Type: "message_start",
		Message: &claudeMessageResponse{
			ID:    event.ID,
			Type:  "message",
			Role:  "assistant",
			Model: event.Model,
		},
	}
	data, err := json.Marshal(ce)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal message_start: %w", err)
	}
	return data, report, nil
}

func encodeStreamContentStart(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	ce := claudeStreamEvent{
		Type: "content_block_start",
	}
	if len(event.Choices) > 0 {
		ce.Index = event.Choices[0].Index
		for _, part := range event.Choices[0].Delta.Content {
			switch p := part.(type) {
			case uni.TextPart:
				ce.ContentBlock = &contentBlock{Type: "text", Text: p.Text}
			case uni.ToolUsePart:
				ce.ContentBlock = &contentBlock{Type: "tool_use", ID: p.ToolCallID, Name: p.ToolName, Input: p.Arguments}
			case uni.ThinkingPart:
				ce.ContentBlock = &contentBlock{Type: "thinking", Thinking: p.Thinking, Signature: p.Signature}
			}
			break
		}
	}
	data, err := json.Marshal(ce)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal content_block_start: %w", err)
	}
	return data, report, nil
}

func encodeStreamDelta(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	ce := claudeStreamEvent{
		Type: "content_block_delta",
	}
	if len(event.Choices) > 0 {
		ce.Index = event.Choices[0].Index
		choice := event.Choices[0]

		if len(choice.Delta.Content) > 0 {
			part := choice.Delta.Content[0]
			switch p := part.(type) {
			case uni.TextPart:
				ce.Delta = &contentBlockDelta{Type: "text_delta", Text: p.Text}
			case uni.ThinkingPart:
				if p.Thinking != "" {
					ce.Delta = &contentBlockDelta{Type: "thinking_delta", Thinking: p.Thinking}
				} else if p.Signature != "" {
					ce.Delta = &contentBlockDelta{Type: "signature_delta", Signature: p.Signature}
				}
			}
		}

		if len(choice.Delta.ToolCalls) > 0 {
			td := choice.Delta.ToolCalls[0]
			ce.Delta = &contentBlockDelta{Type: "input_json_delta", PartialJSON: td.Arguments}
		}
	}
	data, err := json.Marshal(ce)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal content_block_delta: %w", err)
	}
	return data, report, nil
}

func encodeStreamContentStop(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	ce := claudeStreamEvent{
		Type: "content_block_stop",
	}
	if len(event.Choices) > 0 {
		ce.Index = event.Choices[0].Index
	}
	data, err := json.Marshal(ce)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal content_block_stop: %w", err)
	}
	return data, report, nil
}

func encodeStreamStop(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	ce := claudeStreamEvent{
		Type: "message_delta",
	}
	if event.StopReason != nil {
		ce.StopReason = convertUniStopReasonToClaude(*event.StopReason)
	}
	ce.StopSequence = event.StopSequence
	if event.Usage != nil {
		ce.Usage = &usage{
			InputTokens:              event.Usage.InputTokens,
			OutputTokens:             event.Usage.OutputTokens,
			CacheCreationInputTokens: event.Usage.CacheCreationInputTokens,
			CacheReadInputTokens:     event.Usage.CacheReadInputTokens,
		}
	}
	data, err := json.Marshal(ce)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal message_delta: %w", err)
	}
	return data, report, nil
}

func encodeStreamError(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	errObj := map[string]any{
		"type": "error",
	}
	if event.Error != nil {
		errObj["error"] = map[string]string{
			"type":    event.Error.Type,
			"message": event.Error.Message,
		}
	}
	data, err := json.Marshal(errObj)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal error event: %w", err)
	}
	return data, report, nil
}
