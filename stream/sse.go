package stream

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

func (e *SSEEvent) IsEmpty() bool {
	return e.ID == "" && e.Event == "" && e.Data == "" && e.Retry == 0
}

func (e *SSEEvent) String() string {
	var b strings.Builder
	if e.ID != "" {
		fmt.Fprintf(&b, "id: %s\n", e.ID)
	}
	if e.Event != "" {
		fmt.Fprintf(&b, "event: %s\n", e.Event)
	}
	if e.Retry > 0 {
		fmt.Fprintf(&b, "retry: %d\n", e.Retry)
	}
	if e.Data != "" {
		lines := strings.Split(e.Data, "\n")
		for _, line := range lines {
			fmt.Fprintf(&b, "data: %s\n", line)
		}
	}
	b.WriteString("\n")
	return b.String()
}

type SSEReader struct {
	scanner *bufio.Scanner
	lineBuf []string
}

func NewSSEReader(r io.Reader) *SSEReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	return &SSEReader{scanner: scanner}
}

func (r *SSEReader) Read() (*SSEEvent, error) {
	event := &SSEEvent{}
	var dataLines []string

	for {
		var line string
		if len(r.lineBuf) > 0 {
			line = r.lineBuf[0]
			r.lineBuf = r.lineBuf[1:]
		} else {
			if !r.scanner.Scan() {
				if err := r.scanner.Err(); err != nil {
					return nil, err
				}
				if len(dataLines) > 0 {
					event.Data = strings.Join(dataLines, "\n")
					return event, nil
				}
				return nil, io.EOF
			}
			line = r.scanner.Text()
		}

		if line == "" {
			if !event.IsEmpty() || len(dataLines) > 0 {
				event.Data = strings.Join(dataLines, "\n")
				return event, nil
			}
			continue
		}

		if strings.HasPrefix(line, ":") {
			continue
		}

		colonIdx := strings.Index(line, ":")
		var field, value string
		if colonIdx == -1 {
			field = line
			value = ""
		} else {
			field = line[:colonIdx]
			value = line[colonIdx+1:]
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
		}

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "retry":
			if n, err := strconv.Atoi(value); err == nil {
				event.Retry = n
			}
		}
	}
}

func (r *SSEReader) Peek() (*SSEEvent, error) {
	event, err := r.Read()
	if err != nil {
		return nil, err
	}
	r.Unread(event)
	return event, nil
}

func (r *SSEReader) Unread(event *SSEEvent) {
	s := event.String()
	// Remove trailing \n added by String(), split into lines
	s = strings.TrimSuffix(s, "\n")
	lines := strings.Split(s, "\n")
	r.lineBuf = append(lines, r.lineBuf...)
}

type SSEWriter struct {
	w io.Writer
}

func NewSSEWriter(w io.Writer) *SSEWriter {
	return &SSEWriter{w: w}
}

func (w *SSEWriter) Write(event *SSEEvent) error {
	_, err := w.w.Write([]byte(event.String()))
	return err
}

func (w *SSEWriter) WriteRaw(data string) error {
	_, err := w.w.Write([]byte(data))
	return err
}

func (w *SSEWriter) Flush() error {
	if f, ok := w.w.(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}
