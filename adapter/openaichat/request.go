package openaichat

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

func decodeRequest(data json.RawMessage) (*uni.RequestParams, *adapter.Report, error) {
	report := adapter.NewReport()

	var req chatCompletionRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, report, fmt.Errorf("unmarshal request: %w", err)
	}

	params := &uni.RequestParams{
		Model:             req.Model,
		Temperature:       req.Temperature,
		TopP:              req.TopP,
		FrequencyPenalty:  req.FrequencyPenalty,
		PresencePenalty:   req.PresencePenalty,
		Stream:            req.Stream,
		Seed:              req.Seed,
		ParallelToolCalls: req.ParallelToolCalls,
	}

	if req.MaxCompletionTokens != nil {
		params.MaxTokens = req.MaxCompletionTokens
	} else if req.MaxTokens != nil {
		params.MaxTokens = req.MaxTokens
	}

	if req.Stop != nil {
		var stopSeqs []string
		if err := json.Unmarshal(req.Stop, &stopSeqs); err != nil {
			var single string
			if err2 := json.Unmarshal(req.Stop, &single); err2 != nil {
				return nil, report, fmt.Errorf("unmarshal stop: %w", err)
			}
			stopSeqs = []string{single}
		}
		params.StopSequences = stopSeqs
	}

	msgs, msgReport, err := decodeMessages(req.Messages)
	if err != nil {
		return nil, report, fmt.Errorf("decode messages: %w", err)
	}
	report.Merge(msgReport)
	params.Messages = msgs

	if len(req.Tools) > 0 {
		tools, err := decodeTools(req.Tools)
		if err != nil {
			return nil, report, fmt.Errorf("decode tools: %w", err)
		}
		params.Tools = tools
	}

	if req.ToolChoice != nil {
		tc, err := decodeToolChoice(req.ToolChoice)
		if err != nil {
			return nil, report, fmt.Errorf("decode tool_choice: %w", err)
		}
		params.ToolChoice = tc
	}

	if req.N != nil {
		report.AddLostField("openai_chat", "n", "no direct uni equivalent, stored in ext")
	}
	if req.User != "" {
		report.AddLostField("openai_chat", "user", "no direct uni equivalent, stored in ext")
	}
	if req.StreamOptions != nil {
		report.AddLostField("openai_chat", "stream_options", "no direct uni equivalent, stored in ext")
	}
	if req.ResponseFormat != nil {
		report.AddLostField("openai_chat", "response_format", "no direct uni equivalent, stored in ext")
	}

	ext := make(uni.ExtData)
	if req.N != nil {
		if err := ext.Set("n", req.N); err != nil {
			return nil, report, fmt.Errorf("set ext n: %w", err)
		}
	}
	if req.StreamOptions != nil {
		ext["stream_options"] = req.StreamOptions
	}
	if req.User != "" {
		if err := ext.Set("user", req.User); err != nil {
			return nil, report, fmt.Errorf("set ext user: %w", err)
		}
	}
	if req.ResponseFormat != nil {
		ext["response_format"] = req.ResponseFormat
	}
	if len(ext) > 0 {
		params.Ext = ext
	}

	return params, report, nil
}

func decodeMessages(msgs []chatMessage) ([]uni.Message, *adapter.Report, error) {
	report := adapter.NewReport()
	result := make([]uni.Message, 0, len(msgs))
	for i, m := range msgs {
		msg, msgReport, err := decodeMessage(m)
		if err != nil {
			return nil, report, fmt.Errorf("message[%d]: %w", i, err)
		}
		report.Merge(msgReport)
		result = append(result, msg)
	}
	return result, report, nil
}

func decodeMessage(m chatMessage) (uni.Message, *adapter.Report, error) {
	report := adapter.NewReport()

	var role uni.Role
	switch m.Role {
	case "system":
		role = uni.RoleSystem
	case "developer":
		role = uni.RoleDeveloper
	case "user":
		role = uni.RoleUser
	case "assistant":
		role = uni.RoleAssistant
	case "tool":
		role = uni.RoleTool
	default:
		role = uni.Role(m.Role)
	}

	msg := uni.Message{Role: role}

	if m.ToolCallID != "" {
		content, contentReport, err := decodeContent(m.Content)
		if err != nil {
			return msg, report, fmt.Errorf("decode tool content: %w", err)
		}
		report.Merge(contentReport)
		msg.Content = []uni.ContentPart{uni.ToolResultPart{ToolCallID: m.ToolCallID, Content: content, IsError: false}}
		return msg, report, nil
	}

	if len(m.ToolCalls) > 0 {
		var parts []uni.ContentPart
		if m.Content != nil {
			content, contentReport, err := decodeContent(m.Content)
			if err != nil {
				return msg, report, fmt.Errorf("decode assistant content: %w", err)
			}
			report.Merge(contentReport)
			parts = append(parts, content...)
		}
		for _, tc := range m.ToolCalls {
			parts = append(parts, uni.ToolUsePart{
				ToolCallID: tc.ID,
				ToolName:   tc.Function.Name,
				Arguments:  json.RawMessage(tc.Function.Arguments),
			})
		}
		if m.Refusal != "" {
			parts = append(parts, uni.RefusalPart{Refusal: m.Refusal})
		}
		msg.Content = parts
		return msg, report, nil
	}

	content, contentReport, err := decodeContent(m.Content)
	if err != nil {
		return msg, report, fmt.Errorf("decode content: %w", err)
	}
	report.Merge(contentReport)
	msg.Content = content

	if m.Refusal != "" {
		msg.Content = append(msg.Content, uni.RefusalPart{Refusal: m.Refusal})
	}

	return msg, report, nil
}

func decodeContent(raw json.RawMessage) ([]uni.ContentPart, *adapter.Report, error) {
	report := adapter.NewReport()

	if raw == nil || string(raw) == "null" {
		return nil, report, nil
	}

	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		if str == "" {
			return nil, report, nil
		}
		return []uni.ContentPart{uni.TextPart{Text: str}}, report, nil
	}

	var parts []contentPart
	if err := json.Unmarshal(raw, &parts); err != nil {
		return nil, report, fmt.Errorf("unmarshal content parts: %w", err)
	}

	result := make([]uni.ContentPart, 0, len(parts))
	for i, p := range parts {
		cp, err := decodeContentPart(p, report)
		if err != nil {
			return nil, report, fmt.Errorf("content part[%d]: %w", i, err)
		}
		result = append(result, cp)
	}
	return result, report, nil
}

func decodeContentPart(p contentPart, report *adapter.Report) (uni.ContentPart, error) {
	switch p.Type {
	case "text":
		return uni.TextPart{Text: p.Text}, nil
	case "image_url":
		if p.ImageURL == nil {
			return nil, fmt.Errorf("image_url is nil")
		}
		return uni.ImagePart{URL: p.ImageURL.URL, Detail: p.ImageURL.Detail}, nil
	case "input_audio":
		if p.InputAudio == nil {
			return nil, fmt.Errorf("input_audio is nil")
		}
		return uni.AudioPart{Data: p.InputAudio.Data, Format: p.InputAudio.Format}, nil
	case "file":
		if p.File == nil {
			return nil, fmt.Errorf("file is nil")
		}
		return uni.FilePart{Data: p.File.FileData, Name: p.File.FileName}, nil
	default:
		report.AddLostField("openai_chat", "content_part.type", fmt.Sprintf("unknown content part type %q, defaulting to text", p.Type))
		return uni.TextPart{Text: p.Text}, nil
	}
}

func decodeTools(defs []toolDef) ([]uni.Tool, error) {
	result := make([]uni.Tool, 0, len(defs))
	for i, d := range defs {
		t := uni.Tool{
			Name:        d.Function.Name,
			Description: d.Function.Description,
			InputSchema: d.Function.Parameters,
			Strict:      d.Function.Strict,
		}
		ext := make(uni.ExtData)
		if d.Type != "" && d.Type != "function" {
			if err := ext.Set("type", d.Type); err != nil {
				return nil, fmt.Errorf("tool[%d] set ext type: %w", i, err)
			}
		}
		if len(ext) > 0 {
			t.Ext = ext
		}
		result = append(result, t)
	}
	return result, nil
}

func decodeToolChoice(raw json.RawMessage) (*uni.ToolChoice, error) {
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		var tcType uni.ToolChoiceType
		switch str {
		case "auto":
			tcType = uni.ToolChoiceAuto
		case "required":
			tcType = uni.ToolChoiceRequired
		case "none":
			tcType = uni.ToolChoiceNone
		default:
			tcType = uni.ToolChoiceType(str)
		}
		return &uni.ToolChoice{Type: tcType}, nil
	}

	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("unmarshal tool_choice object: %w", err)
	}
	return &uni.ToolChoice{Type: uni.ToolChoiceSpecific, ToolName: obj.Function.Name}, nil
}

func encodeRequest(params *uni.RequestParams) (json.RawMessage, *adapter.Report, error) {
	report := adapter.NewReport()

	req := chatCompletionRequest{
		Model:             params.Model,
		Temperature:       params.Temperature,
		TopP:              params.TopP,
		FrequencyPenalty:  params.FrequencyPenalty,
		PresencePenalty:   params.PresencePenalty,
		Stream:            params.Stream,
		Seed:              params.Seed,
		ParallelToolCalls: params.ParallelToolCalls,
	}

	if params.MaxTokens != nil {
		req.MaxCompletionTokens = params.MaxTokens
	}

	if len(params.StopSequences) > 0 {
		stop, err := json.Marshal(params.StopSequences)
		if err != nil {
			return nil, report, fmt.Errorf("marshal stop: %w", err)
		}
		req.Stop = stop
	}

	msgs, msgReport, err := encodeMessages(params.Messages)
	if err != nil {
		return nil, report, fmt.Errorf("encode messages: %w", err)
	}
	report.Merge(msgReport)
	req.Messages = msgs

	if len(params.Tools) > 0 {
		req.Tools = encodeTools(params.Tools)
	}

	if params.ToolChoice != nil {
		tc, err := encodeToolChoice(params.ToolChoice)
		if err != nil {
			return nil, report, fmt.Errorf("encode tool_choice: %w", err)
		}
		req.ToolChoice = tc
	}

	if params.Ext != nil {
		if params.Ext.Has("n") {
			var n int64
			if err := params.Ext.Get("n", &n); err == nil {
				req.N = &n
			}
		}
		if params.Ext.Has("stream_options") {
			req.StreamOptions = params.Ext["stream_options"]
		}
		if params.Ext.Has("user") {
			var user string
			if err := params.Ext.Get("user", &user); err == nil {
				req.User = user
			}
		}
		if params.Ext.Has("response_format") {
			req.ResponseFormat = params.Ext["response_format"]
		}
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, report, fmt.Errorf("marshal request: %w", err)
	}
	return data, report, nil
}

func encodeMessages(msgs []uni.Message) ([]chatMessage, *adapter.Report, error) {
	report := adapter.NewReport()
	result := make([]chatMessage, 0, len(msgs))
	for i, m := range msgs {
		cm, msgReport, err := encodeMessage(m)
		if err != nil {
			return nil, report, fmt.Errorf("message[%d]: %w", i, err)
		}
		report.Merge(msgReport)
		result = append(result, cm)
	}
	return result, report, nil
}

func encodeMessage(m uni.Message) (chatMessage, *adapter.Report, error) {
	report := adapter.NewReport()
	cm := chatMessage{
		Role: string(m.Role),
	}

	if m.Role == uni.RoleTool {
		for _, part := range m.Content {
			if tr, ok := part.(uni.ToolResultPart); ok {
				cm.ToolCallID = tr.ToolCallID
				text := ""
				for _, c := range tr.Content {
					if tp, ok := c.(uni.TextPart); ok {
						if text != "" {
							text += "\n"
						}
						text += tp.Text
					} else {
						report.AddLostField("unified", "tool_result.content",
							fmt.Sprintf("non-text content type %s not supported in OpenAI Chat tool messages", c.ContentType()))
					}
				}
				content, err := json.Marshal(text)
				if err != nil {
					return cm, report, fmt.Errorf("marshal tool content: %w", err)
				}
				cm.Content = content
				return cm, report, nil
			}
		}
		content, err := json.Marshal("")
		if err != nil {
			return cm, report, fmt.Errorf("marshal empty tool content: %w", err)
		}
		cm.Content = content
		return cm, report, nil
	}

	var textParts []string
	var contentParts []contentPart
	var toolCalls []toolCall
	var refusal string

	for _, part := range m.Content {
		switch p := part.(type) {
		case uni.TextPart:
			if m.Role == uni.RoleUser && len(m.Content) > 1 {
				contentParts = append(contentParts, contentPart{Type: "text", Text: p.Text})
			} else {
				textParts = append(textParts, p.Text)
			}
		case uni.ImagePart:
			cp := contentPart{Type: "image_url"}
			if p.URL != "" {
				cp.ImageURL = &imageURLPart{URL: p.URL, Detail: p.Detail}
			} else if p.Data != "" {
				url := p.Data
				if p.MediaType != "" {
					url = "data:" + p.MediaType + ";base64," + p.Data
				}
				cp.ImageURL = &imageURLPart{URL: url, Detail: p.Detail}
			}
			contentParts = append(contentParts, cp)
		case uni.AudioPart:
			contentParts = append(contentParts, contentPart{
				Type:       "input_audio",
				InputAudio: &inputAudioPart{Data: p.Data, Format: p.Format},
			})
		case uni.FilePart:
			fp := &filePart{FileData: p.Data, FileName: p.Name}
			contentParts = append(contentParts, contentPart{Type: "file", File: fp})
		case uni.ToolUsePart:
			tc := toolCall{
				ID:   p.ToolCallID,
				Type: "function",
				Function: functionCallData{
					Name:      p.ToolName,
					Arguments: string(p.Arguments),
				},
			}
			toolCalls = append(toolCalls, tc)
		case uni.RefusalPart:
			refusal = p.Refusal
		case uni.ThinkingPart:
			report.AddLostField("uni", "thinking", "not supported in OpenAI Chat Completions")
		case uni.RedactedThinkingPart:
			report.AddLostField("uni", "redacted_thinking", "not supported in OpenAI Chat Completions")
		}
	}

	if len(contentParts) > 0 {
		if len(textParts) > 0 {
			allParts := make([]contentPart, 0, len(textParts)+len(contentParts))
			for _, t := range textParts {
				allParts = append(allParts, contentPart{Type: "text", Text: t})
			}
			allParts = append(allParts, contentParts...)
			data, err := json.Marshal(allParts)
			if err != nil {
				return cm, report, fmt.Errorf("marshal content parts: %w", err)
			}
			cm.Content = data
		} else {
			data, err := json.Marshal(contentParts)
			if err != nil {
				return cm, report, fmt.Errorf("marshal content parts: %w", err)
			}
			cm.Content = data
		}
	} else if len(textParts) > 0 {
		text := ""
		for _, t := range textParts {
			if text != "" {
				text += "\n"
			}
			text += t
		}
		data, err := json.Marshal(text)
		if err != nil {
			return cm, report, fmt.Errorf("marshal text content: %w", err)
		}
		cm.Content = data
	}

	if len(toolCalls) > 0 {
		cm.ToolCalls = toolCalls
	}
	if refusal != "" {
		cm.Refusal = refusal
	}

	return cm, report, nil
}

func encodeTools(tools []uni.Tool) []toolDef {
	result := make([]toolDef, 0, len(tools))
	for _, t := range tools {
		result = append(result, toolDef{
			Type: "function",
			Function: functionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
				Strict:      t.Strict,
			},
		})
	}
	return result
}

func encodeToolChoice(tc *uni.ToolChoice) (json.RawMessage, error) {
	switch tc.Type {
	case uni.ToolChoiceAuto:
		return json.RawMessage(`"auto"`), nil
	case uni.ToolChoiceRequired:
		return json.RawMessage(`"required"`), nil
	case uni.ToolChoiceNone:
		return json.RawMessage(`"none"`), nil
	case uni.ToolChoiceSpecific:
		obj := struct {
			Type     string `json:"type"`
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}{
			Type: "function",
		}
		obj.Function.Name = tc.ToolName
		data, err := json.Marshal(obj)
		if err != nil {
			return nil, fmt.Errorf("marshal tool_choice: %w", err)
		}
		return data, nil
	default:
		return json.RawMessage(`"auto"`), nil
	}
}
