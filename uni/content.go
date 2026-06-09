package uni

import (
	"encoding/json"
	"fmt"
)

func MarshalContentPart(part ContentPart) ([]byte, error) {
	switch p := part.(type) {
	case TextPart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			TextPart
		}{ContentPartText, p})
	case ImagePart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			ImagePart
		}{ContentPartImage, p})
	case AudioPart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			AudioPart
		}{ContentPartAudio, p})
	case FilePart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			FilePart
		}{ContentPartFile, p})
	case ThinkingPart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			ThinkingPart
		}{ContentPartThinking, p})
	case RedactedThinkingPart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			RedactedThinkingPart
		}{ContentPartRedactedThinking, p})
	case ToolUsePart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			ToolUsePart
		}{ContentPartToolUse, p})
	case ToolResultPart:
		nested, err := MarshalContentParts(p.Content)
		if err != nil {
			return nil, err
		}
		return json.Marshal(struct {
			Type       ContentPartType `json:"type"`
			ToolCallID string          `json:"tool_call_id"`
			Content    json.RawMessage `json:"content,omitempty"`
			IsError    bool            `json:"is_error,omitempty"`
		}{
			Type:       ContentPartToolResult,
			ToolCallID: p.ToolCallID,
			Content:    json.RawMessage(nested),
			IsError:    p.IsError,
		})
	case RefusalPart:
		return json.Marshal(struct {
			Type ContentPartType `json:"type"`
			RefusalPart
		}{ContentPartRefusal, p})
	default:
		return nil, fmt.Errorf("unknown ContentPart type: %T", part)
	}
}

func UnmarshalContentPart(data []byte) (ContentPart, error) {
	var env struct {
		Type ContentPartType `json:"type"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal content part type: %w", err)
	}
	switch env.Type {
	case ContentPartText:
		var p TextPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartImage:
		var p ImagePart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartAudio:
		var p AudioPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartFile:
		var p FilePart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartThinking:
		var p ThinkingPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartRedactedThinking:
		var p RedactedThinkingPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartToolUse:
		var p ToolUsePart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	case ContentPartToolResult:
		var p toolResultPartRaw
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return ToolResultPart{
			ToolCallID: p.ToolCallID,
			IsError:    p.IsError,
			Content:    p.Content,
		}, nil
	case ContentPartRefusal:
		var p RefusalPart
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, err
		}
		return p, nil
	default:
		return nil, fmt.Errorf("unknown content part type: %s", env.Type)
	}
}

type toolResultPartRaw struct {
	ToolCallID string          `json:"tool_call_id"`
	Content    []ContentPart   `json:"content,omitempty"`
	IsError    bool            `json:"is_error,omitempty"`
}

func (r *toolResultPartRaw) UnmarshalJSON(data []byte) error {
	var raw struct {
		ToolCallID string            `json:"tool_call_id"`
		Content    []json.RawMessage `json:"content,omitempty"`
		IsError    bool              `json:"is_error,omitempty"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	r.ToolCallID = raw.ToolCallID
	r.IsError = raw.IsError
	for _, c := range raw.Content {
		part, err := UnmarshalContentPart(c)
		if err != nil {
			return err
		}
		r.Content = append(r.Content, part)
	}
	return nil
}

func MarshalContentParts(parts []ContentPart) ([]byte, error) {
	if len(parts) == 0 {
		return []byte("[]"), nil
	}
	var items []json.RawMessage
	for _, p := range parts {
		data, err := MarshalContentPart(p)
		if err != nil {
			return nil, err
		}
		items = append(items, data)
	}
	return json.Marshal(items)
}

func UnmarshalContentParts(data []byte) ([]ContentPart, error) {
	if data == nil || string(data) == "null" {
		return nil, nil
	}
	var rawItems []json.RawMessage
	if err := json.Unmarshal(data, &rawItems); err != nil {
		return nil, err
	}
	parts := make([]ContentPart, 0, len(rawItems))
	for _, raw := range rawItems {
		p, err := UnmarshalContentPart(raw)
		if err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, nil
}
