package uni

import (
	"encoding/json"
	"testing"
)

func TestMarshalUnmarshalContentPart_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		part ContentPart
	}{
		{
			name: "TextPart",
			part: TextPart{Text: "Hello, world!"},
		},
		{
			name: "ImagePart with URL",
			part: ImagePart{URL: "https://example.com/img.png", Detail: "high"},
		},
		{
			name: "ImagePart with data",
			part: ImagePart{Data: "iVBORw0KGgo=", MediaType: "image/png"},
		},
		{
			name: "AudioPart",
			part: AudioPart{Data: "base64audio", Format: "wav"},
		},
		{
			name: "FilePart with URL",
			part: FilePart{URL: "https://example.com/doc.pdf", Name: "doc.pdf", MediaType: "application/pdf"},
		},
		{
			name: "FilePart with data",
			part: FilePart{Data: "base64file", Name: "file.txt"},
		},
		{
			name: "ThinkingPart",
			part: ThinkingPart{Thinking: "Let me think...", Signature: "sig123"},
		},
		{
			name: "RedactedThinkingPart",
			part: RedactedThinkingPart{Data: "redacted data"},
		},
		{
			name: "ToolUsePart",
			part: ToolUsePart{ToolCallID: "call_001", ToolName: "get_weather", Arguments: json.RawMessage(`{"city":"Tokyo"}`)},
		},
		{
			name: "ToolResultPart with text",
			part: ToolResultPart{ToolCallID: "call_001", Content: []ContentPart{TextPart{Text: "Sunny, 25°C"}}, IsError: false},
		},
		{
			name: "ToolResultPart with error",
			part: ToolResultPart{ToolCallID: "call_002", Content: []ContentPart{TextPart{Text: "Not found"}}, IsError: true},
		},
		{
			name: "RefusalPart",
			part: RefusalPart{Refusal: "I cannot answer that."},
		},
		{
			name: "ThinkingPart without signature",
			part: ThinkingPart{Thinking: "Just thinking..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := MarshalContentPart(tt.part)
			if err != nil {
				t.Fatalf("MarshalContentPart failed: %v", err)
			}

			// Verify type discriminator is in the JSON
			var env struct {
				Type ContentPartType `json:"type"`
			}
			if err := json.Unmarshal(data, &env); err != nil {
				t.Fatalf("unmarshal type wrapper: %v", err)
			}
			if env.Type != tt.part.ContentType() {
				t.Fatalf("expected discriminator %q, got %q", tt.part.ContentType(), env.Type)
			}

			got, err := UnmarshalContentPart(data)
			if err != nil {
				t.Fatalf("UnmarshalContentPart failed: %v", err)
			}

			if got.ContentType() != tt.part.ContentType() {
				t.Fatalf("content type mismatch: got %q, expected %q", got.ContentType(), tt.part.ContentType())
			}

			// Re-marshal and compare JSON
			reData, err := MarshalContentPart(got)
			if err != nil {
				t.Fatalf("re-marshal failed: %v", err)
			}
			if string(data) != string(reData) {
				t.Fatalf("round-trip mismatch:\n  original: %s\n  re-marshaled: %s", string(data), string(reData))
			}
		})
	}
}

func TestMarshalContentParts(t *testing.T) {
	t.Run("nil parts", func(t *testing.T) {
		data, err := MarshalContentParts(nil)
		if err != nil {
			t.Fatalf("MarshalContentParts(nil) failed: %v", err)
		}
		if string(data) != "[]" {
			t.Fatalf("expected [], got %s", string(data))
		}
	})

	t.Run("empty parts", func(t *testing.T) {
		parts := []ContentPart{}
		data, err := MarshalContentParts(parts)
		if err != nil {
			t.Fatalf("MarshalContentParts([]) failed: %v", err)
		}
		if string(data) != "[]" {
			t.Fatalf("expected [], got %s", string(data))
		}
	})

	t.Run("multiple parts", func(t *testing.T) {
		parts := []ContentPart{
			TextPart{Text: "Hello"},
			ImagePart{URL: "https://example.com/img.png"},
			ToolUsePart{ToolCallID: "t1", ToolName: "search", Arguments: json.RawMessage(`{"q":"test"}`)},
		}
		data, err := MarshalContentParts(parts)
		if err != nil {
			t.Fatalf("MarshalContentParts failed: %v", err)
		}
		var raw []json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			t.Fatalf("unmarshal parts array: %v", err)
		}
		if len(raw) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(raw))
		}
	})
}

func TestUnmarshalContentParts(t *testing.T) {
	t.Run("null data", func(t *testing.T) {
		parts, err := UnmarshalContentParts(nil)
		if err != nil {
			t.Fatalf("UnmarshalContentParts(nil) failed: %v", err)
		}
		if parts != nil {
			t.Fatalf("expected nil, got %v", parts)
		}
	})

	t.Run("\"null\" string", func(t *testing.T) {
		parts, err := UnmarshalContentParts([]byte("null"))
		if err != nil {
			t.Fatalf("UnmarshalContentParts(null) failed: %v", err)
		}
		if parts != nil {
			t.Fatalf("expected nil, got %v", parts)
		}
	})

	t.Run("valid parts array", func(t *testing.T) {
		data := json.RawMessage(`[{"type":"text","text":"Hello"},{"type":"image","url":"https://example.com/img.png"}]`)
		parts, err := UnmarshalContentParts(data)
		if err != nil {
			t.Fatalf("UnmarshalContentParts failed: %v", err)
		}
		if len(parts) != 2 {
			t.Fatalf("expected 2 parts, got %d", len(parts))
		}
		tp, ok := parts[0].(TextPart)
		if !ok || tp.Text != "Hello" {
			t.Fatalf("expected TextPart{Hello}, got %+v", parts[0])
		}
		ip, ok := parts[1].(ImagePart)
		if !ok || ip.URL != "https://example.com/img.png" {
			t.Fatalf("expected ImagePart with URL, got %+v", parts[1])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := UnmarshalContentParts([]byte("not-json"))
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})

	t.Run("unknown type", func(t *testing.T) {
		_, err := UnmarshalContentParts([]byte(`[{"type":"unknown_type","value":123}]`))
		if err == nil {
			t.Fatal("expected error for unknown type")
		}
	})
}

func TestUnmarshalContentPart_UnknownType(t *testing.T) {
	_, err := UnmarshalContentPart([]byte(`{"type":"nonexistent","text":"hi"}`))
	if err == nil {
		t.Fatal("expected error for unknown content part type")
	}
}

func TestExtData(t *testing.T) {
	t.Run("Set and Get string", func(t *testing.T) {
		var ext ExtData
		if err := ext.Set("key", "value"); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		if !ext.Has("key") {
			t.Fatal("expected Has(key) true")
		}
		var s string
		if err := ext.Get("key", &s); err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if s != "value" {
			t.Fatalf("expected 'value', got %q", s)
		}
	})

	t.Run("Set and Get int", func(t *testing.T) {
		var ext ExtData
		if err := ext.Set("count", int64(42)); err != nil {
			t.Fatalf("Set failed: %v", err)
		}
		var n int64
		if err := ext.Get("count", &n); err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if n != 42 {
			t.Fatalf("expected 42, got %d", n)
		}
	})

	t.Run("Get missing key", func(t *testing.T) {
		var ext ExtData
		var s string
		if err := ext.Get("nonexistent", &s); err != nil {
			t.Fatalf("Get missing key should not error: %v", err)
		}
		if s != "" {
			t.Fatalf("expected empty string, got %q", s)
		}
	})

	t.Run("Has missing key", func(t *testing.T) {
		var ext ExtData
		if ext.Has("nonexistent") {
			t.Fatal("expected Has(nonexistent) false")
		}
	})

	t.Run("Set multiple and Has", func(t *testing.T) {
		var ext ExtData
		ext.Set("a", "1")
		ext.Set("b", "2")
		if !ext.Has("a") {
			t.Fatal("expected Has(a) true")
		}
		if !ext.Has("b") {
			t.Fatal("expected Has(b) true")
		}
	})

	t.Run("Set nil map", func(t *testing.T) {
		var ext ExtData
		if err := ext.Set("x", "y"); err != nil {
			t.Fatalf("Set on nil ExtData failed: %v", err)
		}
		if !ext.Has("x") {
			t.Fatal("expected Has(x) true after Set on nil")
		}
	})

	t.Run("Set raw JSON", func(t *testing.T) {
		ext := make(ExtData)
		ext["raw"] = json.RawMessage(`{"nested":"data"}`)
		var m map[string]string
		if err := ext.Get("raw", &m); err != nil {
			t.Fatalf("Get raw failed: %v", err)
		}
		if m["nested"] != "data" {
			t.Fatalf("expected nested=data, got %v", m)
		}
	})

	t.Run("Get type mismatch", func(t *testing.T) {
		var ext ExtData
		ext.Set("key", "string_value")
		var n int
		err := ext.Get("key", &n)
		if err == nil {
			t.Fatal("expected error for type mismatch")
		}
	})
}

func TestMessageConstructors(t *testing.T) {
	content := []ContentPart{TextPart{Text: "Hello"}}

	t.Run("UserMessage", func(t *testing.T) {
		msg := UserMessage(content...)
		if msg.Role != RoleUser {
			t.Fatalf("expected user role, got %s", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(msg.Content))
		}
	})

	t.Run("AssistantMessage", func(t *testing.T) {
		msg := AssistantMessage(content...)
		if msg.Role != RoleAssistant {
			t.Fatalf("expected assistant role, got %s", msg.Role)
		}
	})

	t.Run("SystemMessage", func(t *testing.T) {
		msg := SystemMessage(content...)
		if msg.Role != RoleSystem {
			t.Fatalf("expected system role, got %s", msg.Role)
		}
	})

	t.Run("DeveloperMessage", func(t *testing.T) {
		msg := DeveloperMessage(content...)
		if msg.Role != RoleDeveloper {
			t.Fatalf("expected developer role, got %s", msg.Role)
		}
	})

	t.Run("ToolMessage", func(t *testing.T) {
		msg := ToolMessage("call_123", content, true)
		if msg.Role != RoleTool {
			t.Fatalf("expected tool role, got %s", msg.Role)
		}
		if len(msg.Content) != 1 {
			t.Fatalf("expected 1 content part, got %d", len(msg.Content))
		}
		tr, ok := msg.Content[0].(ToolResultPart)
		if !ok {
			t.Fatal("expected ToolResultPart")
		}
		if tr.ToolCallID != "call_123" {
			t.Fatalf("expected call_123, got %s", tr.ToolCallID)
		}
		if !tr.IsError {
			t.Fatal("expected IsError true")
		}
	})
}

func TestContentType(t *testing.T) {
	tests := []struct {
		part     ContentPart
		expected ContentPartType
	}{
		{TextPart{}, ContentPartText},
		{ImagePart{}, ContentPartImage},
		{AudioPart{}, ContentPartAudio},
		{FilePart{}, ContentPartFile},
		{ThinkingPart{}, ContentPartThinking},
		{RedactedThinkingPart{}, ContentPartRedactedThinking},
		{ToolUsePart{}, ContentPartToolUse},
		{ToolResultPart{}, ContentPartToolResult},
		{RefusalPart{}, ContentPartRefusal},
	}

	for _, tt := range tests {
		if tt.part.ContentType() != tt.expected {
			t.Errorf("%T: expected %q, got %q", tt.part, tt.expected, tt.part.ContentType())
		}
	}
}

func TestToolResultPart_NestedContent(t *testing.T) {
	// Test round-trip for ToolResultPart with nested ImagePart
	orig := ToolResultPart{
		ToolCallID: "call_nested",
		Content: []ContentPart{
			TextPart{Text: "Here is an image:"},
			ImagePart{URL: "https://example.com/img.png", Detail: "low"},
		},
		IsError: false,
	}

	data, err := MarshalContentPart(orig)
	if err != nil {
		t.Fatalf("MarshalContentPart failed: %v", err)
	}

	got, err := UnmarshalContentPart(data)
	if err != nil {
		t.Fatalf("UnmarshalContentPart failed: %v", err)
	}

	tr, ok := got.(ToolResultPart)
	if !ok {
		t.Fatalf("expected ToolResultPart, got %T", got)
	}
	if tr.ToolCallID != "call_nested" {
		t.Fatalf("expected call_nested, got %s", tr.ToolCallID)
	}
	if len(tr.Content) != 2 {
		t.Fatalf("expected 2 nested content parts, got %d", len(tr.Content))
	}
	if tp, ok := tr.Content[0].(TextPart); !ok || tp.Text != "Here is an image:" {
		t.Fatalf("nested TextPart mismatch: %+v", tr.Content[0])
	}
	if ip, ok := tr.Content[1].(ImagePart); !ok || ip.URL != "https://example.com/img.png" {
		t.Fatalf("nested ImagePart mismatch: %+v", tr.Content[1])
	}
}

func TestErrorResponse(t *testing.T) {
	e := &ErrorResponse{
		Type:    "invalid_request_error",
		Message: "Invalid model",
		Status:  400,
	}

	if e.Error() != "Invalid model" {
		t.Fatalf("expected Error() = 'Invalid model', got %q", e.Error())
	}
}
