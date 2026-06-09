package openairesp

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeStreamEvent(data json.RawMessage) (*uni.StreamEvent, *adapter.Report, error) {
	report := adapter.NewReport()

	var evt responseStreamEvent
	if err := json.Unmarshal(data, &evt); err != nil {
		return nil, report, fmt.Errorf("unmarshal responses stream event: %w", err)
	}

	switch evt.Type {
	case "response.created":
		return decodeCreated(&evt), report, nil

	case "response.in_progress":
		return &uni.StreamEvent{
			Type:  uni.StreamEventDelta,
			Model: extractModel(evt.Response),
		}, report, nil

	case "response.completed":
		return decodeCompleted(&evt), report, nil

	case "response.failed":
		return decodeFailed(&evt), report, nil

	case "response.output_item.added":
		return decodeOutputItemAdded(&evt), report, nil

	case "response.output_item.done":
		return decodeOutputItemDone(&evt), report, nil

	case "response.content_part.added":
		return decodeContentPartAdded(&evt), report, nil

	case "response.content_part.done":
		return decodeContentPartDone(&evt), report, nil

	case "response.output_text.delta":
		return &uni.StreamEvent{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.TextPart{Text: evt.Delta}},
					},
				},
			},
		}, report, nil

	case "response.output_text.done":
		return &uni.StreamEvent{
			Type: uni.StreamEventContentStop,
		}, report, nil

	case "response.function_call_arguments.delta":
		return &uni.StreamEvent{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						ToolCalls: []uni.ToolCallDelta{
							{
								Index:     int(evt.OutputIndex),
								Arguments: evt.Delta,
							},
						},
					},
				},
			},
		}, report, nil

	case "response.function_call_arguments.done":
		return &uni.StreamEvent{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						ToolCalls: []uni.ToolCallDelta{
							{
								Index:      int(evt.OutputIndex),
								Arguments:  evt.Arguments,
								ToolCallID: evt.ItemID,
							},
						},
					},
				},
			},
		}, report, nil

	case "response.refusal.delta":
		return &uni.StreamEvent{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.RefusalPart{Refusal: evt.Delta}},
					},
				},
			},
		}, report, nil

	case "response.refusal.done":
		return &uni.StreamEvent{
			Type: uni.StreamEventContentStop,
		}, report, nil

	case "error":
		return decodeError(&evt), report, nil

	default:
		return &uni.StreamEvent{
			Type: uni.StreamEventDelta,
		}, report, nil
	}
}

func decodeCreated(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventStart,
	}
	if evt.Response != nil {
		se.ID = evt.Response.ID
		se.Model = evt.Response.Model
		se.Created = evt.Response.CreatedAt
	}
	return se
}

func decodeCompleted(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventStop,
	}
	if evt.Response != nil {
		se.Model = evt.Response.Model
		if evt.Response.Usage != nil {
			se.Usage = &uni.Usage{
				InputTokens:  evt.Response.Usage.InputTokens,
				OutputTokens: evt.Response.Usage.OutputTokens,
				TotalTokens:  evt.Response.Usage.TotalTokens,
			}
		}
		reason := mapStatus(evt.Response.Status)
		if reason != "" {
			se.StopReason = &reason
		}
	}
	return se
}

func decodeFailed(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventError,
	}
	if evt.Response != nil && evt.Response.Error != nil {
		se.Error = &uni.StreamError{
			Type:    evt.Response.Error.Type,
			Message: evt.Response.Error.Message,
			Code:    evt.Response.Error.Code,
		}
	}
	return se
}

func decodeOutputItemAdded(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventContentStart,
	}
	if evt.Item != nil {
		switch evt.Item.Type {
		case "message":
			se.Choices = []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						Role:    uni.RoleAssistant,
						Content: []uni.ContentPart{},
					},
				},
			}
		case "function_call":
			se.Choices = []uni.StreamChoice{
				{
					Index: int(evt.OutputIndex),
					Delta: uni.StreamDelta{
						ToolCalls: []uni.ToolCallDelta{
							{
								Index:      int(evt.OutputIndex),
								ToolCallID: evt.Item.CallID,
								ToolName:   evt.Item.Name,
							},
						},
					},
				},
			}
		}
	}
	return se
}

func decodeOutputItemDone(evt *responseStreamEvent) *uni.StreamEvent {
	return &uni.StreamEvent{
		Type: uni.StreamEventDelta,
	}
}

func decodeContentPartAdded(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventContentStart,
	}
	if evt.Part != nil {
		switch evt.Part.Type {
		case "output_text":
			se.Choices = []uni.StreamChoice{
				{
					Index: int(evt.ContentIndex),
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{},
					},
				},
			}
		case "refusal":
			se.Choices = []uni.StreamChoice{
				{
					Index: int(evt.ContentIndex),
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{},
					},
				},
			}
		}
	}
	return se
}

func decodeContentPartDone(evt *responseStreamEvent) *uni.StreamEvent {
	return &uni.StreamEvent{
		Type: uni.StreamEventContentStop,
	}
}

func decodeError(evt *responseStreamEvent) *uni.StreamEvent {
	se := &uni.StreamEvent{
		Type: uni.StreamEventError,
	}
	if evt.Error != nil {
		se.Error = &uni.StreamError{
			Type:    evt.Error.Type,
			Message: evt.Error.Message,
			Code:    evt.Error.Code,
		}
	}
	return se
}

func extractModel(resp *responseResponse) string {
	if resp != nil {
		return resp.Model
	}
	return ""
}

func encodeStreamEvent(event *uni.StreamEvent) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	switch event.Type {
	case uni.StreamEventStart:
		return encodeStartEvent(event, report)

	case uni.StreamEventDelta:
		return encodeDeltaEvent(event, report)

	case uni.StreamEventContentStart:
		return encodeContentStartEvent(event, report)

	case uni.StreamEventContentStop:
		return encodeContentStopEvent(report)

	case uni.StreamEventStop:
		return encodeStopEvent(event, report)

	case uni.StreamEventError:
		return encodeErrorEvent(event, report)

	default:
		return nil, report, fmt.Errorf("unsupported stream event type: %s", event.Type)
	}
}

func encodeStartEvent(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	evt := responseStreamEvent{
		Type: "response.created",
		Response: &responseResponse{
			ID:        event.ID,
			Object:    "response",
			CreatedAt: event.Created,
			Model:     event.Model,
			Status:    "in_progress",
			Output:    []outputItem{},
		},
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal response.created: %w", err)
	}
	return data, report, nil
}

func encodeDeltaEvent(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	if len(event.Choices) == 0 {
		evt := responseStreamEvent{Type: "response.in_progress"}
		data, err := json.Marshal(evt)
		if err != nil {
			return nil, report, fmt.Errorf("marshal response.in_progress: %w", err)
		}
		return data, report, nil
	}

	choice := event.Choices[0]

	if len(choice.Delta.ToolCalls) > 0 {
		tc := choice.Delta.ToolCalls[0]
		if tc.Arguments != "" {
			evt := responseStreamEvent{
				Type:        "response.function_call_arguments.delta",
				OutputIndex: int64(choice.Index),
				Delta:       tc.Arguments,
				ItemID:      tc.ToolCallID,
			}
			data, err := json.Marshal(evt)
			if err != nil {
				return nil, report, fmt.Errorf("marshal function_call_arguments.delta: %w", err)
			}
			return data, report, nil
		}
	}

	if len(choice.Delta.Content) > 0 {
		switch p := choice.Delta.Content[0].(type) {
		case uni.TextPart:
			if p.Text != "" {
				evt := responseStreamEvent{
					Type:        "response.output_text.delta",
					OutputIndex: int64(choice.Index),
					Delta:       p.Text,
				}
				data, err := json.Marshal(evt)
				if err != nil {
					return nil, report, fmt.Errorf("marshal output_text.delta: %w", err)
				}
				return data, report, nil
			}
		case uni.RefusalPart:
			if p.Refusal != "" {
				evt := responseStreamEvent{
					Type:        "response.refusal.delta",
					OutputIndex: int64(choice.Index),
					Delta:       p.Refusal,
				}
				data, err := json.Marshal(evt)
				if err != nil {
					return nil, report, fmt.Errorf("marshal refusal.delta: %w", err)
				}
				return data, report, nil
			}
		}
	}

	evt := responseStreamEvent{Type: "response.in_progress"}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal response.in_progress: %w", err)
	}
	return data, report, nil
}

func encodeContentStartEvent(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	outputIndex := int64(0)
	if len(event.Choices) > 0 {
		outputIndex = int64(event.Choices[0].Index)
	}

	item := &outputItem{
		Type:   "message",
		Role:   "assistant",
		Status: "in_progress",
	}

	if len(event.Choices) > 0 && len(event.Choices[0].Delta.ToolCalls) > 0 {
		tc := event.Choices[0].Delta.ToolCalls[0]
		item = &outputItem{
			Type:   "function_call",
			Status: "in_progress",
			CallID: tc.ToolCallID,
			Name:   tc.ToolName,
		}
	}

	evt := responseStreamEvent{
		Type:        "response.output_item.added",
		OutputIndex: outputIndex,
		Item:        item,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal output_item.added: %w", err)
	}
	return data, report, nil
}

func encodeContentStopEvent(report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	evt := responseStreamEvent{
		Type: "response.content_part.done",
		Part: &outputContent{
			Type: "output_text",
		},
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal content_part.done: %w", err)
	}
	return data, report, nil
}

func encodeStopEvent(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	resp := &responseResponse{
		ID:     event.ID,
		Object: "response",
		Model:  event.Model,
		Status: "completed",
		Output: []outputItem{},
	}

	if event.Usage != nil {
		resp.Usage = &respUsage{
			InputTokens:  event.Usage.InputTokens,
			OutputTokens: event.Usage.OutputTokens,
			TotalTokens:  event.Usage.TotalTokens,
		}
	}

	if event.StopReason != nil {
		resp.Status = mapStopReasonToStatus(*event.StopReason)
	}

	evt := responseStreamEvent{
		Type:     "response.completed",
		Response: resp,
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal response.completed: %w", err)
	}
	return data, report, nil
}

func encodeErrorEvent(event *uni.StreamEvent, report *adapter.Report) (json.RawMessage, *adapter.Report, error) {
	evt := responseStreamEvent{
		Type: "error",
	}
	if event.Error != nil {
		evt.Error = &respError{
			Type:    event.Error.Type,
			Message: event.Error.Message,
			Code:    event.Error.Code,
		}
	}
	data, err := json.Marshal(evt)
	if err != nil {
		return nil, report, fmt.Errorf("marshal error event: %w", err)
	}
	return data, report, nil
}
