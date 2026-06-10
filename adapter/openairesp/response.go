package openairesp

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeResponse(data json.RawMessage) (*uni.Response, *adapter.Report, error) {
	report := adapter.NewReport()

	var resp responseResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, report, fmt.Errorf("unmarshal responses response: %w", err)
	}

	result := &uni.Response{
		ID:      resp.ID,
		Model:   resp.Model,
		Created: resp.CreatedAt,
	}

	if resp.Usage != nil {
		result.Usage = uni.Usage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	result.StopReason = mapStatus(resp.Status)

	var assistantParts []uni.ContentPart
	var toolMessages []uni.Message

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, c := range item.Content {
				switch c.Type {
				case "output_text":
					assistantParts = append(assistantParts, uni.TextPart{Text: c.Text})
				case "refusal":
					assistantParts = append(assistantParts, uni.RefusalPart{Refusal: c.Refusal})
				}
			}

		case "function_call":
			assistantParts = append(assistantParts, uni.ToolUsePart{
				ToolCallID: item.CallID,
				ToolName:   item.Name,
				Arguments:  json.RawMessage(item.Arguments),
			})
			if result.StopReason == "" {
				result.StopReason = uni.StopReasonToolCalls
			}

		case "function_call_output":
			toolMessages = append(toolMessages, uni.ToolMessage(
				item.CallID,
				[]uni.ContentPart{uni.TextPart{Text: item.Output}},
				false,
			))

		case "reasoning":
			summary := ""
			if item.Summary != nil {
				var summaries []struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(item.Summary, &summaries); err == nil && len(summaries) > 0 {
					summary = summaries[0].Text
				}
			}
			// Skip empty reasoning (e.g., summary: [])
			if summary != "" {
				assistantParts = append(assistantParts, uni.ThinkingPart{Thinking: summary})
			}
		}
	}

	if len(assistantParts) > 0 {
		result.Messages = append(result.Messages, uni.Message{
			Role:    uni.RoleAssistant,
			Content: assistantParts,
		})
	}
	result.Messages = append(result.Messages, toolMessages...)

	if resp.Error != nil {
		result.StopReason = uni.StopReasonRefusal
		ext := make(uni.ExtData)
		ext.Set("openai_responses", map[string]any{
			"error": map[string]any{
				"type":    resp.Error.Type,
				"code":    resp.Error.Code,
				"message": resp.Error.Message,
			},
		})
		result.Ext = ext
	}

	if resp.Metadata != nil {
		report.AddLostField("response", "metadata", "no direct mapping in uni")
		if result.Ext == nil {
			result.Ext = make(uni.ExtData)
		}
		if !result.Ext.Has("openai_responses") {
			result.Ext.Set("openai_responses", map[string]any{
				"metadata": resp.Metadata,
			})
		}
	}

	if resp.IncompleteDetails != nil {
		report.AddLostField("response", "incomplete_details", "no direct mapping in uni")
	}

	return result, report, nil
}

func encodeResponse(resp *uni.Response) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	out := responseResponse{
		ID:        resp.ID,
		Object:    "response",
		CreatedAt: resp.Created,
		Model:     resp.Model,
		Status:    mapStopReasonToStatus(resp.StopReason),
	}

	if resp.Usage != (uni.Usage{}) {
		out.Usage = &respUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	for _, msg := range resp.Messages {
		switch msg.Role {
		case uni.RoleAssistant:
			for _, part := range msg.Content {
				switch p := part.(type) {
				case uni.TextPart:
					out.Output = append(out.Output, outputItem{
						Type:   "message",
						Role:   "assistant",
						Status: "completed",
						Content: []outputContent{
							{Type: "output_text", Text: p.Text},
						},
					})
				case uni.ToolUsePart:
					out.Output = append(out.Output, outputItem{
						Type:      "function_call",
						Status:    "completed",
						CallID:    p.ToolCallID,
						Name:      p.ToolName,
						Arguments: string(p.Arguments),
					})
				case uni.ThinkingPart:
					summaryData, _ := json.Marshal([]map[string]string{
						{"type": "summary_text", "text": p.Thinking},
					})
					out.Output = append(out.Output, outputItem{
						Type:    "reasoning",
						Status:  "completed",
						Summary: summaryData,
					})
				case uni.RefusalPart:
					out.Output = append(out.Output, outputItem{
						Type:   "message",
						Role:   "assistant",
						Status: "completed",
						Content: []outputContent{
							{Type: "refusal", Refusal: p.Refusal},
						},
					})
				}
			}

		case uni.RoleTool:
			for _, part := range msg.Content {
				if tr, ok := part.(uni.ToolResultPart); ok {
					text := extractText(tr.Content)
					out.Output = append(out.Output, outputItem{
						Type:   "function_call_output",
						Status: "completed",
						CallID: tr.ToolCallID,
						Output: text,
					})
				}
			}
		}
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil, report, fmt.Errorf("marshal responses response: %w", err)
	}
	return data, report, nil
}

func mapStatus(status string) uni.StopReason {
	switch status {
	case "completed":
		return uni.StopReasonEndTurn
	case "failed":
		return uni.StopReasonRefusal
	case "incomplete":
		return uni.StopReasonMaxTokens
	case "cancelled":
		return uni.StopReasonEndTurn
	default:
		return ""
	}
}

func mapStopReasonToStatus(reason uni.StopReason) string {
	switch reason {
	case uni.StopReasonEndTurn:
		return "completed"
	case uni.StopReasonRefusal:
		return "failed"
	case uni.StopReasonMaxTokens:
		return "incomplete"
	case uni.StopReasonToolCalls:
		return "completed"
	default:
		return "completed"
	}
}
