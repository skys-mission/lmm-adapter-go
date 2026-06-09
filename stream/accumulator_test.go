package stream

import (
	"testing"

	"github.com/skys-mission/lmm-adapter-go/uni"
)

func TestAccumulatorTextStream(t *testing.T) {
	acc := NewAccumulator()

	events := []*uni.StreamEvent{
		{
			Type:  uni.StreamEventStart,
			ID:    "msg_123",
			Model: "gpt-4o",
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Role: uni.RoleAssistant,
					},
				},
			},
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.TextPart{Text: "Hello"}},
					},
				},
			},
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.TextPart{Text: " World"}},
					},
				},
			},
		},
		{
			Type:       uni.StreamEventStop,
			StopReason: ptrStopReason(uni.StopReasonEndTurn),
			Usage: &uni.Usage{
				InputTokens:  10,
				OutputTokens: 20,
				TotalTokens:  30,
			},
		},
	}

	for _, event := range events {
		if err := acc.Accumulate(event); err != nil {
			t.Fatalf("accumulate error: %v", err)
		}
	}

	resp := acc.Response()

	if resp.ID != "msg_123" {
		t.Errorf("expected ID msg_123, got %q", resp.ID)
	}
	if resp.Model != "gpt-4o" {
		t.Errorf("expected model gpt-4o, got %q", resp.Model)
	}
	if resp.StopReason != uni.StopReasonEndTurn {
		t.Errorf("expected stop reason end_turn, got %q", resp.StopReason)
	}
	if resp.Usage.InputTokens != 10 {
		t.Errorf("expected 10 input tokens, got %d", resp.Usage.InputTokens)
	}
	if resp.Usage.OutputTokens != 20 {
		t.Errorf("expected 20 output tokens, got %d", resp.Usage.OutputTokens)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}

	msg := resp.Messages[0]
	if msg.Role != uni.RoleAssistant {
		t.Errorf("expected assistant role, got %q", msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(msg.Content))
	}

	textPart, ok := msg.Content[0].(uni.TextPart)
	if !ok {
		t.Fatalf("expected TextPart, got %T", msg.Content[0])
	}
	if textPart.Text != "Hello World" {
		t.Errorf("expected 'Hello World', got %q", textPart.Text)
	}
}

func TestAccumulatorToolCalls(t *testing.T) {
	acc := NewAccumulator()

	events := []*uni.StreamEvent{
		{
			Type:  uni.StreamEventStart,
			ID:    "msg_456",
			Model: "gpt-4o",
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						ToolCalls: []uni.ToolCallDelta{
							{
								Index:      0,
								ToolCallID: "call_1",
								ToolName:   "get_weather",
								Arguments:  `{"location":"`,
							},
						},
					},
				},
			},
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						ToolCalls: []uni.ToolCallDelta{
							{
								Index:     0,
								Arguments: `San Francisco"}`,
							},
						},
					},
				},
			},
		},
		{
			Type:       uni.StreamEventStop,
			StopReason: ptrStopReason(uni.StopReasonToolCalls),
		},
	}

	for _, event := range events {
		if err := acc.Accumulate(event); err != nil {
			t.Fatalf("accumulate error: %v", err)
		}
	}

	resp := acc.Response()

	if resp.StopReason != uni.StopReasonToolCalls {
		t.Errorf("expected stop reason tool_calls, got %q", resp.StopReason)
	}

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}

	msg := resp.Messages[0]
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content part, got %d", len(msg.Content))
	}

	toolPart, ok := msg.Content[0].(uni.ToolUsePart)
	if !ok {
		t.Fatalf("expected ToolUsePart, got %T", msg.Content[0])
	}
	if toolPart.ToolCallID != "call_1" {
		t.Errorf("expected call_1, got %q", toolPart.ToolCallID)
	}
	if toolPart.ToolName != "get_weather" {
		t.Errorf("expected get_weather, got %q", toolPart.ToolName)
	}
	if string(toolPart.Arguments) != `{"location":"San Francisco"}` {
		t.Errorf("expected full arguments, got %q", string(toolPart.Arguments))
	}
}

func TestAccumulatorThinking(t *testing.T) {
	acc := NewAccumulator()

	events := []*uni.StreamEvent{
		{
			Type:  uni.StreamEventStart,
			ID:    "msg_789",
			Model: "claude-3-opus",
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.ThinkingPart{Thinking: "Let me think"}},
					},
				},
			},
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.ThinkingPart{Thinking: " about this"}},
					},
				},
			},
		},
		{
			Type: uni.StreamEventDelta,
			Choices: []uni.StreamChoice{
				{
					Index: 0,
					Delta: uni.StreamDelta{
						Content: []uni.ContentPart{uni.TextPart{Text: "The answer is 42"}},
					},
				},
			},
		},
		{
			Type:       uni.StreamEventStop,
			StopReason: ptrStopReason(uni.StopReasonEndTurn),
		},
	}

	for _, event := range events {
		if err := acc.Accumulate(event); err != nil {
			t.Fatalf("accumulate error: %v", err)
		}
	}

	resp := acc.Response()

	if len(resp.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.Messages))
	}

	msg := resp.Messages[0]
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(msg.Content))
	}

	thinkingPart, ok := msg.Content[0].(uni.ThinkingPart)
	if !ok {
		t.Fatalf("expected ThinkingPart, got %T", msg.Content[0])
	}
	if thinkingPart.Thinking != "Let me think about this" {
		t.Errorf("expected full thinking, got %q", thinkingPart.Thinking)
	}

	textPart, ok := msg.Content[1].(uni.TextPart)
	if !ok {
		t.Fatalf("expected TextPart, got %T", msg.Content[1])
	}
	if textPart.Text != "The answer is 42" {
		t.Errorf("expected 'The answer is 42', got %q", textPart.Text)
	}
}

func TestAccumulatorReset(t *testing.T) {
	acc := NewAccumulator()

	acc.Accumulate(&uni.StreamEvent{
		Type:  uni.StreamEventStart,
		ID:    "msg_123",
		Model: "gpt-4o",
	})

	acc.Reset()

	resp := acc.Response()
	if resp.ID != "" {
		t.Errorf("expected empty ID after reset, got %q", resp.ID)
	}
	if resp.Model != "" {
		t.Errorf("expected empty model after reset, got %q", resp.Model)
	}
}

func ptrStopReason(reason uni.StopReason) *uni.StopReason {
	return &reason
}
