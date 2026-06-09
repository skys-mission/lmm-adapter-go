package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeRequest(data json.RawMessage) (*uni.RequestParams, *adapter.Report, error) {
	report := adapter.NewReport()

	var req claudeMessageRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, report, fmt.Errorf("failed to unmarshal claude request: %w", err)
	}

	params := &uni.RequestParams{
		Model:         req.Model,
		Temperature:   req.Temperature,
		TopP:          req.TopP,
		TopK:          req.TopK,
		StopSequences: req.StopSequences,
		Stream:        req.Stream,
	}

	if req.MaxTokens != 0 {
		params.MaxTokens = &req.MaxTokens
	}

	systemBlocks, err := decodeSystemField(req.System)
	if err != nil {
		return nil, report, fmt.Errorf("decode system: %w", err)
	}
	for _, sb := range systemBlocks {
		params.Messages = append(params.Messages, uni.SystemMessage(uni.TextPart{Text: sb.Text}))
		if sb.CacheControl != nil {
			report.AddLostField("claude", "system.cache_control", "cache_control not supported in unified format")
		}
	}

	for _, msg := range req.Messages {
		converted, err := convertClaudeMessageToUni(msg, report)
		if err != nil {
			return nil, report, fmt.Errorf("failed to convert message: %w", err)
		}
		params.Messages = append(params.Messages, converted)
	}

	for _, td := range req.Tools {
		tool := uni.Tool{
			Name:        td.Name,
			Description: td.Description,
			InputSchema: td.InputSchema,
		}
		params.Tools = append(params.Tools, tool)
	}

	if req.ToolChoice != nil {
		tc := convertClaudeToolChoiceToUni(req.ToolChoice)
		params.ToolChoice = &tc
	}

	if req.Metadata != nil {
		if params.Ext == nil {
			params.Ext = make(uni.ExtData)
		}
		ext := map[string]json.RawMessage{}
		ext["metadata"] = req.Metadata
		raw, err := json.Marshal(ext)
		if err == nil {
			params.Ext["claude"] = raw
		}
	}

	return params, report, nil
}

func convertClaudeMessageToUni(msg claudeMessage, report *adapter.Report) (uni.Message, error) {
	var role uni.Role
	switch msg.Role {
	case "user":
		role = uni.RoleUser
	case "assistant":
		role = uni.RoleAssistant
	default:
		return uni.Message{}, fmt.Errorf("unknown claude message role: %s", msg.Role)
	}

	blocks, err := decodeContentField(msg.Content)
	if err != nil {
		return uni.Message{}, fmt.Errorf("decode content: %w", err)
	}

	var parts []uni.ContentPart
	for _, block := range blocks {
		part, err := convertClaudeBlockToUni(block, report)
		if err != nil {
			return uni.Message{}, err
		}
		if part != nil {
			parts = append(parts, part)
		}
	}

	return uni.Message{Role: role, Content: parts}, nil
}

func decodeContentField(raw json.RawMessage) ([]contentBlock, error) {
	if raw == nil || string(raw) == "null" {
		return nil, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil, nil
		}
		return []contentBlock{{Type: "text", Text: s}}, nil
	}

	var blocks []contentBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("unmarshal content blocks: %w", err)
	}
	return blocks, nil
}

func decodeSystemField(raw json.RawMessage) ([]systemBlock, error) {
	if raw == nil || string(raw) == "null" {
		return nil, nil
	}

	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return nil, nil
		}
		return []systemBlock{{Type: "text", Text: s}}, nil
	}

	var blocks []systemBlock
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return nil, fmt.Errorf("unmarshal system blocks: %w", err)
	}
	return blocks, nil
}

func convertClaudeBlockToUni(block contentBlock, report *adapter.Report) (uni.ContentPart, error) {
	switch block.Type {
	case "text":
		return uni.TextPart{Text: block.Text}, nil
	case "image":
		if block.Source == nil {
			return nil, fmt.Errorf("image block missing source")
		}
		switch block.Source.Type {
		case "base64":
			return uni.ImagePart{Data: block.Source.Data, MediaType: block.Source.MediaType}, nil
		case "url":
			return uni.ImagePart{URL: block.Source.URL}, nil
		default:
			return nil, fmt.Errorf("unknown image source type: %s", block.Source.Type)
		}
	case "tool_use":
		return uni.ToolUsePart{ToolCallID: block.ID, ToolName: block.Name, Arguments: block.Input}, nil
	case "tool_result":
		var content []uni.ContentPart
		if block.Text != "" {
			content = append(content, uni.TextPart{Text: block.Text})
		}
		if block.Content != nil {
			var subContent json.RawMessage
			if err := json.Unmarshal(block.Content, &subContent); err == nil {
				var textContent string
				if err := json.Unmarshal(subContent, &textContent); err == nil {
					if textContent != "" && block.Text == "" {
						content = append(content, uni.TextPart{Text: textContent})
					}
				} else {
					var parts []json.RawMessage
					if err := json.Unmarshal(subContent, &parts); err == nil {
						for _, p := range parts {
							cp, err := uni.UnmarshalContentPart(p)
							if err == nil && cp != nil {
								content = append(content, cp)
							}
						}
					}
				}
			}
		}
		return uni.ToolResultPart{ToolCallID: block.ToolUseID, Content: content, IsError: block.IsError}, nil
	case "thinking":
		return uni.ThinkingPart{Thinking: block.Thinking, Signature: block.Signature}, nil
	default:
		report.AddLostField("claude", "content_block.type", fmt.Sprintf("unknown block type: %s", block.Type))
		if block.Text != "" {
			return uni.TextPart{Text: block.Text}, nil
		}
		return nil, nil
	}
}

func convertClaudeToolChoiceToUni(tc *toolChoice) uni.ToolChoice {
	switch tc.Type {
	case "auto":
		return uni.ToolChoice{Type: uni.ToolChoiceAuto}
	case "any":
		return uni.ToolChoice{Type: uni.ToolChoiceRequired}
	case "none":
		return uni.ToolChoice{Type: uni.ToolChoiceNone}
	case "tool":
		return uni.ToolChoice{Type: uni.ToolChoiceSpecific, ToolName: tc.Name}
	default:
		return uni.ToolChoice{Type: uni.ToolChoiceAuto}
	}
}

func encodeRequest(params *uni.RequestParams) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	req := claudeMessageRequest{
		Model:         params.Model,
		Temperature:   params.Temperature,
		TopP:          params.TopP,
		TopK:          params.TopK,
		StopSequences: params.StopSequences,
		Stream:        params.Stream,
	}

	if params.MaxTokens != nil {
		req.MaxTokens = *params.MaxTokens
	} else {
		req.MaxTokens = 4096
	}

	var systemBlocks []systemBlock
	for _, msg := range params.Messages {
		if msg.Role == uni.RoleSystem || msg.Role == uni.RoleDeveloper {
			for _, part := range msg.Content {
				switch p := part.(type) {
				case uni.TextPart:
					systemBlocks = append(systemBlocks, systemBlock{Type: "text", Text: p.Text})
				default:
					report.AddLostField("unified", "system.content", fmt.Sprintf("unsupported system content type: %s", part.ContentType()))
				}
			}
			continue
		}

		claudeMsg, err := convertUniMessageToClaude(msg, report)
		if err != nil {
			return nil, report, fmt.Errorf("failed to convert message: %w", err)
		}
		req.Messages = append(req.Messages, claudeMsg)
	}

	if len(systemBlocks) > 0 {
		sysJSON, err := json.Marshal(systemBlocks)
		if err != nil {
			return nil, report, fmt.Errorf("marshal system: %w", err)
		}
		req.System = sysJSON
	}

	for _, tool := range params.Tools {
		td := toolDefinition{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
		req.Tools = append(req.Tools, td)
	}

	if params.ToolChoice != nil {
		tc := convertUniToolChoiceToClaude(params.ToolChoice)
		req.ToolChoice = &tc
	}

	if params.ParallelToolCalls != nil && !*params.ParallelToolCalls {
		if req.ToolChoice == nil {
			req.ToolChoice = &toolChoice{Type: "auto"}
		}
		req.ToolChoice.DisableParallelToolUse = true
	}

	if params.FrequencyPenalty != nil {
		report.AddLostField("unified", "frequency_penalty", "not supported by claude messages api")
	}
	if params.PresencePenalty != nil {
		report.AddLostField("unified", "presence_penalty", "not supported by claude messages api")
	}
	if params.Seed != nil {
		report.AddLostField("unified", "seed", "not supported by claude messages api")
	}

	if params.Ext != nil && params.Ext.Has("claude") {
		var ext map[string]json.RawMessage
		if err := json.Unmarshal(params.Ext["claude"], &ext); err == nil {
			if md, ok := ext["metadata"]; ok {
				req.Metadata = md
			}
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, report, fmt.Errorf("failed to marshal claude request: %w", err)
	}
	return data, report, nil
}

func convertUniMessageToClaude(msg uni.Message, report *adapter.Report) (claudeMessage, error) {
	var role string
	switch msg.Role {
	case uni.RoleUser:
		role = "user"
	case uni.RoleAssistant:
		role = "assistant"
	case uni.RoleTool:
		role = "user"
	default:
		return claudeMessage{}, fmt.Errorf("cannot convert role %s to claude message", msg.Role)
	}

	var blocks []contentBlock
	for _, part := range msg.Content {
		b, err := convertUniPartToClaude(part, report)
		if err != nil {
			return claudeMessage{}, err
		}
		blocks = append(blocks, b...)
	}

	contentJSON, err := json.Marshal(blocks)
	if err != nil {
		return claudeMessage{}, fmt.Errorf("marshal content: %w", err)
	}

	return claudeMessage{Role: role, Content: contentJSON}, nil
}

func convertUniPartToClaude(part uni.ContentPart, report *adapter.Report) ([]contentBlock, error) {
	switch p := part.(type) {
	case uni.TextPart:
		return []contentBlock{{Type: "text", Text: p.Text}}, nil
	case uni.ImagePart:
		if p.URL != "" {
			return []contentBlock{{
				Type: "image",
				Source: &contentSource{
					Type: "url",
					URL:  p.URL,
				},
			}}, nil
		}
		return []contentBlock{{
			Type: "image",
			Source: &contentSource{
				Type:      "base64",
				MediaType: p.MediaType,
				Data:      p.Data,
			},
		}}, nil
	case uni.ToolUsePart:
		return []contentBlock{{
			Type:  "tool_use",
			ID:    p.ToolCallID,
			Name:  p.ToolName,
			Input: p.Arguments,
		}}, nil
	case uni.ToolResultPart:
		var textParts []string
		for _, c := range p.Content {
			switch cp := c.(type) {
			case uni.TextPart:
				textParts = append(textParts, cp.Text)
			}
		}
		text := strings.Join(textParts, "")
		return []contentBlock{{
			Type:      "tool_result",
			ToolUseID: p.ToolCallID,
			IsError:   p.IsError,
			Content:   mustMarshalJSON(text),
		}}, nil
	case uni.ThinkingPart:
		return []contentBlock{{
			Type:      "thinking",
			Thinking:  p.Thinking,
			Signature: p.Signature,
		}}, nil
	case uni.RedactedThinkingPart:
		return []contentBlock{{
			Type: "redacted_thinking",
			Data: p.Data,
		}}, nil
	case uni.AudioPart:
		report.AddLostField("unified", "audio", "audio content not supported by claude messages api")
		return []contentBlock{{Type: "text", Text: "[audio content not supported]"}}, nil
	case uni.FilePart:
		report.AddLostField("unified", "file", "file content not supported by claude messages api")
		return []contentBlock{{Type: "text", Text: "[file content not supported]"}}, nil
	case uni.RefusalPart:
		return []contentBlock{{Type: "text", Text: p.Refusal}}, nil
	default:
		report.AddLostField("unified", "content", fmt.Sprintf("unsupported content part type: %T", part))
		return []contentBlock{{Type: "text", Text: ""}}, nil
	}
}

func convertUniToolChoiceToClaude(tc *uni.ToolChoice) toolChoice {
	switch tc.Type {
	case uni.ToolChoiceAuto:
		return toolChoice{Type: "auto"}
	case uni.ToolChoiceRequired:
		return toolChoice{Type: "any"}
	case uni.ToolChoiceNone:
		return toolChoice{Type: "none"}
	case uni.ToolChoiceSpecific:
		return toolChoice{Type: "tool", Name: tc.ToolName}
	default:
		return toolChoice{Type: "auto"}
	}
}

func mustMarshalJSON(v any) json.RawMessage {
	data, _ := json.Marshal(v)
	return data
}
