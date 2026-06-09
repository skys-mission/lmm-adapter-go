package openairesp

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeRequest(data json.RawMessage) (*uni.RequestParams, *adapter.Report, error) {
	report := adapter.NewReport()

	var req responseRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, report, fmt.Errorf("unmarshal responses request: %w", err)
	}

	params := &uni.RequestParams{
		Model:             req.Model,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		MaxTokens:         req.MaxOutputTokens,
		Stream:            req.Stream,
		ParallelToolCalls: req.ParallelToolCalls,
	}

	if req.PreviousResponseID != "" {
		report.AddLostField("request", "previous_response_id", "no direct mapping in uni")
	}
	if req.Store != nil {
		report.AddLostField("request", "store", "no direct mapping in uni")
	}
	if req.Reasoning != nil {
		report.AddLostField("request", "reasoning", "no direct mapping in uni")
	}
	if req.Metadata != nil {
		report.AddLostField("request", "metadata", "no direct mapping in uni")
	}
	if req.Truncation != "" {
		report.AddLostField("request", "truncation", "no direct mapping in uni")
	}
	if req.ServiceTier != "" {
		report.AddLostField("request", "service_tier", "no direct mapping in uni")
	}
	if req.Background != nil {
		report.AddLostField("request", "background", "no direct mapping in uni")
	}
	if req.MaxToolCalls != nil {
		report.AddLostField("request", "max_tool_calls", "no direct mapping in uni")
	}

	if req.Input != nil {
		msgs, err := decodeInput(req.Input)
		if err != nil {
			return nil, report, fmt.Errorf("decode input: %w", err)
		}
		params.Messages = msgs
	}

	if req.Instructions != "" {
		instrMsg := uni.SystemMessage(uni.TextPart{Text: req.Instructions})
		params.Messages = append([]uni.Message{instrMsg}, params.Messages...)
	}

	if len(req.Tools) > 0 {
		params.Tools = decodeTools(req.Tools, report)
	}

	if req.ToolChoice != nil {
		tc, err := decodeToolChoice(req.ToolChoice)
		if err != nil {
			return nil, report, fmt.Errorf("decode tool_choice: %w", err)
		}
		params.ToolChoice = tc
	}

	ext := make(uni.ExtData)
	extData := map[string]any{}
	if req.PreviousResponseID != "" {
		extData["previous_response_id"] = req.PreviousResponseID
	}
	if req.Store != nil {
		extData["store"] = req.Store
	}
	if req.Reasoning != nil {
		extData["reasoning"] = req.Reasoning
	}
	if req.Metadata != nil {
		extData["metadata"] = req.Metadata
	}
	if req.Truncation != "" {
		extData["truncation"] = req.Truncation
	}
	if req.ServiceTier != "" {
		extData["service_tier"] = req.ServiceTier
	}
	if req.Background != nil {
		extData["background"] = req.Background
	}
	if req.MaxToolCalls != nil {
		extData["max_tool_calls"] = req.MaxToolCalls
	}
	if len(extData) > 0 {
		ext.Set("openai_responses", extData)
		params.Ext = ext
	}

	return params, report, nil
}

func decodeInput(raw json.RawMessage) ([]uni.Message, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []uni.Message{uni.UserMessage(uni.TextPart{Text: s})}, nil
	}

	var items []inputItem
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("unmarshal input items: %w", err)
	}

	var messages []uni.Message
	var assistantParts []uni.ContentPart

	flushAssistant := func() {
		if len(assistantParts) > 0 {
			messages = append(messages, uni.Message{Role: uni.RoleAssistant, Content: assistantParts})
			assistantParts = nil
		}
	}

	for _, item := range items {
		switch item.Type {
		case "message":
			flushAssistant()
			role := uni.Role(item.Role)
			parts, err := decodeInputContent(item.Content)
			if err != nil {
				return nil, fmt.Errorf("decode message content: %w", err)
			}
			messages = append(messages, uni.Message{Role: role, Content: parts})

		case "function_call":
			assistantParts = append(assistantParts, uni.ToolUsePart{
				ToolCallID: item.CallID,
				ToolName:   item.Name,
				Arguments:  json.RawMessage(item.Arguments),
			})

		case "function_call_output":
			flushAssistant()
			toolMsg := uni.ToolMessage(
				item.CallID,
				[]uni.ContentPart{uni.TextPart{Text: item.Output}},
				false,
			)
			messages = append(messages, toolMsg)

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
			assistantParts = append(assistantParts, uni.ThinkingPart{Thinking: summary})
		}
	}
	flushAssistant()

	return messages, nil
}

func decodeInputContent(raw json.RawMessage) ([]uni.ContentPart, error) {
	if raw == nil {
		return nil, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return []uni.ContentPart{uni.TextPart{Text: s}}, nil
	}

	var contents []inputContent
	if err := json.Unmarshal(raw, &contents); err != nil {
		return nil, fmt.Errorf("unmarshal input content: %w", err)
	}

	var parts []uni.ContentPart
	for _, c := range contents {
		switch c.Type {
		case "input_text":
			parts = append(parts, uni.TextPart{Text: c.Text})
		case "input_image":
			if c.ImageURL != "" {
				parts = append(parts, uni.ImagePart{URL: c.ImageURL, Detail: c.Detail})
			} else if c.FileData != "" {
				parts = append(parts, uni.ImagePart{Data: c.FileData})
			}
		case "input_file":
			if c.FileURL != "" {
				parts = append(parts, uni.FilePart{URL: c.FileURL})
			} else if c.FileData != "" {
				parts = append(parts, uni.FilePart{Data: c.FileData})
			}
		}
	}
	return parts, nil
}

func decodeTools(tools []toolDef, report *adapter.Report) []uni.Tool {
	var result []uni.Tool
	for _, t := range tools {
		if t.Type != "function" {
			report.AddLostField("openai_responses", "tools.type",
				fmt.Sprintf("non-function tool type %q (%s) not supported in unified IR", t.Type, t.Name))
			continue
		}
		result = append(result, uni.Tool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
			Strict:      t.Strict,
		})
	}
	return result
}

func decodeToolChoice(raw json.RawMessage) (*uni.ToolChoice, error) {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		switch s {
		case "auto":
			return &uni.ToolChoice{Type: uni.ToolChoiceAuto}, nil
		case "required":
			return &uni.ToolChoice{Type: uni.ToolChoiceRequired}, nil
		case "none":
			return &uni.ToolChoice{Type: uni.ToolChoiceNone}, nil
		}
		return &uni.ToolChoice{Type: uni.ToolChoiceAuto}, nil
	}

	var obj struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal tool_choice: %w", err)
	}
	return &uni.ToolChoice{Type: uni.ToolChoiceSpecific, ToolName: obj.Name}, nil
}

func encodeRequest(params *uni.RequestParams) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	req := responseRequest{
		Model:             params.Model,
		Temperature:       params.Temperature,
		TopP:              params.TopP,
		MaxOutputTokens:   params.MaxTokens,
		Stream:            params.Stream,
		ParallelToolCalls: params.ParallelToolCalls,
	}

	if params.TopK != nil {
		report.AddLostField("request", "top_k", "not supported by responses API")
	}
	if params.FrequencyPenalty != nil {
		report.AddLostField("request", "frequency_penalty", "not supported by responses API")
	}
	if params.PresencePenalty != nil {
		report.AddLostField("request", "presence_penalty", "not supported by responses API")
	}
	if len(params.StopSequences) > 0 {
		report.AddLostField("request", "stop_sequences", "not supported by responses API")
	}
	if params.Seed != nil {
		report.AddLostField("request", "seed", "not supported by responses API")
	}

	input, instructions, err := encodeInput(params.Messages)
	if err != nil {
		return nil, report, fmt.Errorf("encode input: %w", err)
	}
	req.Input = input
	req.Instructions = instructions

	if len(params.Tools) > 0 {
		req.Tools = encodeTools(params.Tools)
	}

	if params.ToolChoice != nil {
		req.ToolChoice = encodeToolChoice(params.ToolChoice)
	}

	if params.Ext != nil && params.Ext.Has("openai_responses") {
		var ext map[string]json.RawMessage
		if err := params.Ext.Get("openai_responses", &ext); err == nil {
			if v, ok := ext["previous_response_id"]; ok {
				json.Unmarshal(v, &req.PreviousResponseID)
			}
			if v, ok := ext["store"]; ok {
				json.Unmarshal(v, &req.Store)
			}
			if v, ok := ext["reasoning"]; ok {
				req.Reasoning = v
			}
			if v, ok := ext["metadata"]; ok {
				req.Metadata = v
			}
			if v, ok := ext["truncation"]; ok {
				json.Unmarshal(v, &req.Truncation)
			}
			if v, ok := ext["service_tier"]; ok {
				json.Unmarshal(v, &req.ServiceTier)
			}
			if v, ok := ext["background"]; ok {
				json.Unmarshal(v, &req.Background)
			}
			if v, ok := ext["max_tool_calls"]; ok {
				json.Unmarshal(v, &req.MaxToolCalls)
			}
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, report, fmt.Errorf("marshal responses request: %w", err)
	}
	return data, report, nil
}

func encodeInput(messages []uni.Message) (json.RawMessage, string, error) {
	var items []inputItem
	var instructions string

	for _, msg := range messages {
		switch msg.Role {
		case uni.RoleSystem, uni.RoleDeveloper:
			text := extractText(msg.Content)
			if instructions != "" {
				instructions += "\n" + text
			} else {
				instructions = text
			}

		case uni.RoleUser:
			item := inputItem{Type: "message", Role: "user"}
			content := encodeInputContentParts(msg.Content)
			raw, err := json.Marshal(content)
			if err != nil {
				return nil, "", fmt.Errorf("marshal user content: %w", err)
			}
			item.Content = raw
			items = append(items, item)

		case uni.RoleAssistant:
			var textParts []uni.ContentPart
			for _, part := range msg.Content {
				switch p := part.(type) {
				case uni.TextPart:
					textParts = append(textParts, p)
				case uni.ToolUsePart:
					items = append(items, inputItem{
						Type:      "function_call",
						CallID:    p.ToolCallID,
						Name:      p.ToolName,
						Arguments: string(p.Arguments),
					})
				case uni.ThinkingPart:
					if p.Thinking == "" {
						continue
					}
					summaryJSON, _ := json.Marshal([]map[string]string{
						{"type": "summary_text", "text": p.Thinking},
					})
					items = append(items, inputItem{
						Type:    "reasoning",
						Summary: summaryJSON,
					})
				case uni.RefusalPart:
					textParts = append(textParts, p)
				default:
					textParts = append(textParts, part)
				}
			}
			if len(textParts) > 0 {
				item := inputItem{Type: "message", Role: "assistant"}
				content := encodeInputContentParts(textParts)
				raw, err := json.Marshal(content)
				if err != nil {
					return nil, "", fmt.Errorf("marshal assistant content: %w", err)
				}
				item.Content = raw
				items = append(items, item)
			}

		case uni.RoleTool:
			for _, part := range msg.Content {
				if tr, ok := part.(uni.ToolResultPart); ok {
					output := extractText(tr.Content)
					items = append(items, inputItem{
						Type:   "function_call_output",
						CallID: tr.ToolCallID,
						Output: output,
					})
				}
			}
		}
	}

	raw, err := json.Marshal(items)
	if err != nil {
		return nil, "", fmt.Errorf("marshal input items: %w", err)
	}
	return raw, instructions, nil
}

func encodeInputContentParts(parts []uni.ContentPart) []inputContent {
	var result []inputContent
	for _, part := range parts {
		switch p := part.(type) {
		case uni.TextPart:
			result = append(result, inputContent{Type: "input_text", Text: p.Text})
		case uni.ImagePart:
			if p.URL != "" {
				ic := inputContent{Type: "input_image", ImageURL: p.URL}
				if p.Detail != "" {
					ic.Detail = p.Detail
				}
				result = append(result, ic)
			} else if p.Data != "" {
				result = append(result, inputContent{Type: "input_image", FileData: p.Data})
			}
		case uni.AudioPart:
			result = append(result, inputContent{Type: "input_file", FileData: p.Data})
		case uni.FilePart:
			if p.URL != "" {
				result = append(result, inputContent{Type: "input_file", FileURL: p.URL})
			} else if p.Data != "" {
				result = append(result, inputContent{Type: "input_file", FileData: p.Data})
			}
		case uni.RefusalPart:
			result = append(result, inputContent{Type: "input_text", Text: p.Refusal})
		default:
			// Unsupported content part type — logged by caller if needed.
		}
	}
	return result
}

func encodeTools(tools []uni.Tool) []toolDef {
	var result []toolDef
	for _, t := range tools {
		result = append(result, toolDef{
			Type:        "function",
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.InputSchema,
			Strict:      t.Strict,
		})
	}
	return result
}

func encodeToolChoice(tc *uni.ToolChoice) json.RawMessage {
	switch tc.Type {
	case uni.ToolChoiceAuto:
		return json.RawMessage(`"auto"`)
	case uni.ToolChoiceRequired:
		return json.RawMessage(`"required"`)
	case uni.ToolChoiceNone:
		return json.RawMessage(`"none"`)
	case uni.ToolChoiceSpecific:
		data, _ := json.Marshal(map[string]string{
			"type": "function",
			"name": tc.ToolName,
		})
		return data
	}
	return json.RawMessage(`"auto"`)
}

func extractText(parts []uni.ContentPart) string {
	var text string
	for _, part := range parts {
		if p, ok := part.(uni.TextPart); ok {
			if text != "" {
				text += "\n" + p.Text
			} else {
				text = p.Text
			}
		}
	}
	return text
}
