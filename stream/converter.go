package stream

import (
	"encoding/json"
	"fmt"

	"github.com/skys-mission/lmm-adapter-go/adapter"
	"github.com/skys-mission/lmm-adapter-go/uni"
)

type StreamState struct {
	Started           bool
	ContentBlockOpen  bool
	RoleSent          bool
	CurrentBlockIndex int
	CurrentToolIndex  int
	Model             string
	ID                string
	Usage             *uni.Usage
	StopReason        *uni.StopReason
	StopSequence      string
	Metadata          map[string]any
}

func NewStreamState() *StreamState {
	return &StreamState{
		Metadata: make(map[string]any),
	}
}

func (s *StreamState) Reset() {
	s.Started = false
	s.ContentBlockOpen = false
	s.RoleSent = false
	s.CurrentBlockIndex = 0
	s.CurrentToolIndex = 0
	s.Model = ""
	s.ID = ""
	s.Usage = nil
	s.StopReason = nil
	s.StopSequence = ""
	for k := range s.Metadata {
		delete(s.Metadata, k)
	}
}

type StreamConverter struct {
	src    adapter.Adapter
	dst    adapter.Adapter
	state  *StreamState
	report *adapter.Report
}

func NewStreamConverter(src, dst adapter.Adapter) *StreamConverter {
	return &StreamConverter{
		src:    src,
		dst:    dst,
		state:  NewStreamState(),
		report: adapter.NewReport(),
	}
}

func (c *StreamConverter) State() *StreamState {
	return c.state
}

func (c *StreamConverter) Report() *adapter.Report {
	return c.report
}

func (c *StreamConverter) Reset() {
	c.state.Reset()
	c.report = adapter.NewReport()
}

func (c *StreamConverter) Convert(data json.RawMessage) ([]json.RawMessage, error) {
	unified, decodeReport, err := c.src.DecodeStreamEvent(data)
	if err != nil {
		return nil, fmt.Errorf("decode stream event: %w", err)
	}
	c.report.Merge(decodeReport)

	events, err := c.transform(unified)
	if err != nil {
		return nil, fmt.Errorf("transform stream event: %w", err)
	}

	var results []json.RawMessage
	for _, evt := range events {
		result, encodeReport, err := c.dst.EncodeStreamEvent(evt)
		if err != nil {
			return nil, fmt.Errorf("encode stream event: %w", err)
		}
		c.report.Merge(encodeReport)
		results = append(results, result)
	}

	return results, nil
}

func (c *StreamConverter) transform(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	switch event.Type {
	case uni.StreamEventStart:
		return c.handleStart(event)
	case uni.StreamEventDelta:
		return c.handleDelta(event)
	case uni.StreamEventContentStart:
		return c.handleContentStart(event)
	case uni.StreamEventContentStop:
		return c.handleContentStop(event)
	case uni.StreamEventStop:
		return c.handleStop(event)
	case uni.StreamEventError:
		return []*uni.StreamEvent{event}, nil
	default:
		return []*uni.StreamEvent{event}, nil
	}
}

func (c *StreamConverter) handleStart(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	if c.state.Started {
		return nil, nil
	}
	c.state.Started = true
	c.state.Model = event.Model
	c.state.ID = event.ID

	return []*uni.StreamEvent{event}, nil
}

func (c *StreamConverter) handleDelta(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	if !c.state.Started {
		startEvt := &uni.StreamEvent{
			Type:  uni.StreamEventStart,
			Model: event.Model,
			ID:    event.ID,
		}
		if startEvt.Model == "" {
			startEvt.Model = c.state.Model
		}
		if startEvt.ID == "" {
			startEvt.ID = c.state.ID
		}
		c.state.Started = true
		c.state.Model = startEvt.Model
		c.state.ID = startEvt.ID

		events := []*uni.StreamEvent{startEvt}

		if !c.state.RoleSent && len(event.Choices) > 0 && event.Choices[0].Delta.Role != "" {
			c.state.RoleSent = true
		} else if !c.state.RoleSent {
			roleEvt := &uni.StreamEvent{
				Type: uni.StreamEventDelta,
				Choices: []uni.StreamChoice{
					{
						Index: 0,
						Delta: uni.StreamDelta{Role: uni.RoleAssistant},
					},
				},
			}
			events = append(events, roleEvt)
			c.state.RoleSent = true
		}

		events = append(events, event)
		return events, nil
	}

	if !c.state.RoleSent && len(event.Choices) > 0 && event.Choices[0].Delta.Role != "" {
		c.state.RoleSent = true
	}

	return []*uni.StreamEvent{event}, nil
}

func (c *StreamConverter) handleContentStart(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	c.state.ContentBlockOpen = true
	if len(event.Choices) > 0 {
		c.state.CurrentBlockIndex = event.Choices[0].Index
	}
	return []*uni.StreamEvent{event}, nil
}

func (c *StreamConverter) handleContentStop(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	c.state.ContentBlockOpen = false
	return []*uni.StreamEvent{event}, nil
}

func (c *StreamConverter) handleStop(event *uni.StreamEvent) ([]*uni.StreamEvent, error) {
	var events []*uni.StreamEvent

	if c.state.ContentBlockOpen {
		contentStop := &uni.StreamEvent{
			Type: uni.StreamEventContentStop,
			Choices: []uni.StreamChoice{
				{Index: c.state.CurrentBlockIndex},
			},
		}
		events = append(events, contentStop)
		c.state.ContentBlockOpen = false
	}

	if event.Usage != nil {
		c.state.Usage = event.Usage
	}
	if event.StopReason != nil {
		c.state.StopReason = event.StopReason
	}
	c.state.StopSequence = event.StopSequence

	events = append(events, event)
	return events, nil
}
