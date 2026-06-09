package stream

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/skys-mission/lmm-adapter-go/adapter"
)

type Pipeline struct {
	converter *StreamConverter
}

func NewPipeline(src, dst adapter.Adapter) *Pipeline {
	return &Pipeline{
		converter: NewStreamConverter(src, dst),
	}
}

func (p *Pipeline) Converter() *StreamConverter {
	return p.converter
}

func (p *Pipeline) Reset() {
	p.converter.Reset()
}

func (p *Pipeline) Pipe(ctx context.Context, r io.Reader, w io.Writer) error {
	sseReader := NewSSEReader(r)
	sseWriter := NewSSEWriter(w)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		event, err := sseReader.Read()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read SSE event: %w", err)
		}

		if event.Data == "" {
			continue
		}

		results, err := p.converter.Convert(json.RawMessage(event.Data))
		if err != nil {
			// Log warning but continue processing
			if p.converter.report != nil {
				p.converter.report.AddWarning(
					adapter.SeverityWarning,
					"stream.convert",
					fmt.Sprintf("failed to convert event: %v", err),
				)
			}
			continue
		}

		for _, result := range results {
			outEvent := &SSEEvent{
				Data: string(result),
			}
			if event.Event != "" {
				outEvent.Event = event.Event
			}
			if event.ID != "" {
				outEvent.ID = event.ID
			}

			if err := sseWriter.Write(outEvent); err != nil {
				return fmt.Errorf("write SSE event: %w", err)
			}
		}

		if err := sseWriter.Flush(); err != nil {
			return fmt.Errorf("flush SSE writer: %w", err)
		}
	}
}

func (p *Pipeline) PipeEvents(ctx context.Context, events <-chan json.RawMessage, output chan<- json.RawMessage) error {
	defer close(output)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case data, ok := <-events:
			if !ok {
				return nil
			}

			results, err := p.converter.Convert(data)
			if err != nil {
				// Log warning but continue processing
				if p.converter.report != nil {
					p.converter.report.AddWarning(
						adapter.SeverityWarning,
						"stream.convert",
						fmt.Sprintf("failed to convert event: %v", err),
					)
				}
				continue
			}

			for _, result := range results {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case output <- result:
				}
			}
		}
	}
}

func (p *Pipeline) ConvertSingle(data json.RawMessage) ([]json.RawMessage, error) {
	return p.converter.Convert(data)
}
