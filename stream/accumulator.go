package stream

import (
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type Accumulator struct {
	id           string
	model        string
	created      int64
	messages     []uni.Message
	usage        uni.Usage
	stopReason   uni.StopReason
	stopSequence string

	currentAssistantParts []uni.ContentPart
	currentToolCalls      map[int]*toolCallAccumulator
	role                  uni.Role
	started               bool
}

type toolCallAccumulator struct {
	id        string
	name      string
	arguments string
}

func NewAccumulator() *Accumulator {
	return &Accumulator{
		currentToolCalls: make(map[int]*toolCallAccumulator),
	}
}

func (a *Accumulator) Accumulate(event *uni.StreamEvent) error {
	switch event.Type {
	case uni.StreamEventStart:
		a.started = true
		if event.ID != "" {
			a.id = event.ID
		}
		if event.Model != "" {
			a.model = event.Model
		}
		if event.Created > 0 {
			a.created = event.Created
		}

	case uni.StreamEventDelta:
		for _, choice := range event.Choices {
			a.accumulateChoice(choice)
		}

	case uni.StreamEventContentStart:
		for _, choice := range event.Choices {
			for _, part := range choice.Delta.Content {
				a.currentAssistantParts = append(a.currentAssistantParts, part)
			}
		}

	case uni.StreamEventContentStop:
		// Content block completed

	case uni.StreamEventStop:
		if event.Usage != nil {
			a.usage = *event.Usage
		}
		if event.StopReason != nil {
			a.stopReason = *event.StopReason
		}
		if event.StopSequence != "" {
			a.stopSequence = event.StopSequence
		}
		a.finalizeToolCalls()

	case uni.StreamEventError:
		// Error during streaming - finalize what we have
		a.finalizeToolCalls()
	}

	return nil
}

func (a *Accumulator) accumulateChoice(choice uni.StreamChoice) {
	if choice.Delta.Role != "" {
		a.role = choice.Delta.Role
	}

	for _, part := range choice.Delta.Content {
		a.accumulateContentPart(part)
	}

	for _, tc := range choice.Delta.ToolCalls {
		a.accumulateToolCall(tc)
	}

	if choice.FinishReason != nil {
		a.stopReason = *choice.FinishReason
	}
}

func (a *Accumulator) accumulateContentPart(part uni.ContentPart) {
	if len(a.currentAssistantParts) == 0 {
		a.currentAssistantParts = append(a.currentAssistantParts, part)
		return
	}

	lastIdx := len(a.currentAssistantParts) - 1
	last := a.currentAssistantParts[lastIdx]

	switch p := part.(type) {
	case uni.TextPart:
		if existing, ok := last.(uni.TextPart); ok {
			a.currentAssistantParts[lastIdx] = uni.TextPart{Text: existing.Text + p.Text}
		} else {
			a.currentAssistantParts = append(a.currentAssistantParts, p)
		}
	case uni.ThinkingPart:
		if existing, ok := last.(uni.ThinkingPart); ok {
			sig := existing.Signature
			if p.Signature != "" {
				sig = p.Signature
			}
			a.currentAssistantParts[lastIdx] = uni.ThinkingPart{
				Thinking:  existing.Thinking + p.Thinking,
				Signature: sig,
			}
		} else {
			a.currentAssistantParts = append(a.currentAssistantParts, p)
		}
	case uni.RefusalPart:
		if existing, ok := last.(uni.RefusalPart); ok {
			a.currentAssistantParts[lastIdx] = uni.RefusalPart{Refusal: existing.Refusal + p.Refusal}
		} else {
			a.currentAssistantParts = append(a.currentAssistantParts, p)
		}
	default:
		a.currentAssistantParts = append(a.currentAssistantParts, part)
	}
}

func (a *Accumulator) accumulateToolCall(tc uni.ToolCallDelta) {
	existing, ok := a.currentToolCalls[tc.Index]
	if !ok {
		existing = &toolCallAccumulator{}
		a.currentToolCalls[tc.Index] = existing
	}

	if tc.ToolCallID != "" {
		existing.id = tc.ToolCallID
	}
	if tc.ToolName != "" {
		existing.name = tc.ToolName
	}
	existing.arguments += tc.Arguments
}

func (a *Accumulator) finalizeToolCalls() {
	// Build lookup of accumulated tool calls by ID for precise matching.
	accByID := make(map[string]*toolCallAccumulator)
	unmatched := make(map[int]*toolCallAccumulator)
	for idx, tc := range a.currentToolCalls {
		unmatched[idx] = tc
		if tc.id != "" {
			accByID[tc.id] = tc
		}
	}

	var textParts []uni.ContentPart
	for _, part := range a.currentAssistantParts {
		tp, ok := part.(uni.ToolUsePart)
		if !ok {
			textParts = append(textParts, part)
			continue
		}

		if tc, found := accByID[tp.ToolCallID]; found {
			// Prefer accumulated arguments; keep shell name if accumulator lacks one.
			if tc.name == "" {
				tc.name = tp.ToolName
			}
			if tc.id == "" {
				tc.id = tp.ToolCallID
			}
			continue
		}

		// No matching accumulator: preserve shell only if it already has arguments.
		if len(tp.Arguments) > 0 {
			textParts = append(textParts, tp)
		}
	}

	var toolParts []uni.ContentPart
	for _, tc := range unmatched {
		toolParts = append(toolParts, uni.ToolUsePart{
			ToolCallID: tc.id,
			ToolName:   tc.name,
			Arguments:  []byte(tc.arguments),
		})
	}

	allParts := append(textParts, toolParts...)
	if len(allParts) > 0 {
		a.messages = append(a.messages, uni.AssistantMessage(allParts...))
	}

	a.currentAssistantParts = nil
	a.currentToolCalls = make(map[int]*toolCallAccumulator)
}

func (a *Accumulator) Response() *uni.Response {
	return &uni.Response{
		ID:           a.id,
		Model:        a.model,
		Messages:     a.messages,
		Usage:        a.usage,
		StopReason:   a.stopReason,
		StopSequence: a.stopSequence,
		Created:      a.created,
	}
}

func (a *Accumulator) Reset() {
	a.id = ""
	a.model = ""
	a.created = 0
	a.messages = nil
	a.usage = uni.Usage{}
	a.stopReason = ""
	a.stopSequence = ""
	a.currentAssistantParts = nil
	a.currentToolCalls = make(map[int]*toolCallAccumulator)
	a.role = ""
	a.started = false
}
