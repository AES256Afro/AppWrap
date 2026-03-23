package service

import "time"

// EventKind categorizes service events for frontend consumption.
type EventKind int

const (
	EventInfo     EventKind = iota // General informational message
	EventProgress                  // Progress update with percentage
	EventLogLine                   // Streaming log output (build logs, etc.)
	EventWarning                   // Non-fatal warning
	EventError                     // Error message
	EventComplete                  // Operation completed
)

// Event is the universal message type sent from service methods to frontends.
type Event struct {
	Kind      EventKind   `json:"kind"`
	Message   string      `json:"message"`
	Percent   int         `json:"percent,omitempty"`   // 0-100, for EventProgress
	Data      interface{} `json:"data,omitempty"`       // Optional structured payload
	Timestamp time.Time   `json:"timestamp"`
}

// Emit sends an event to the channel if it's not nil.
func Emit(ch chan<- Event, kind EventKind, msg string) {
	if ch == nil {
		return
	}
	select {
	case ch <- Event{Kind: kind, Message: msg, Timestamp: time.Now()}:
	default:
		// Drop event if channel is full (non-blocking)
	}
}

// EmitProgress sends a progress event.
func EmitProgress(ch chan<- Event, percent int, msg string) {
	if ch == nil {
		return
	}
	select {
	case ch <- Event{Kind: EventProgress, Message: msg, Percent: percent, Timestamp: time.Now()}:
	default:
	}
}

// EmitLog sends a log line event.
func EmitLog(ch chan<- Event, line string) {
	if ch == nil {
		return
	}
	select {
	case ch <- Event{Kind: EventLogLine, Message: line, Timestamp: time.Now()}:
	default:
	}
}

// EmitComplete sends a completion event.
func EmitComplete(ch chan<- Event, msg string) {
	if ch == nil {
		return
	}
	select {
	case ch <- Event{Kind: EventComplete, Message: msg, Timestamp: time.Now()}:
	default:
	}
}

// EventWriter wraps a channel and implements io.Writer, splitting lines and sending each as EventLogLine.
// Used to capture Docker build output.
type EventWriter struct {
	ch  chan<- Event
	buf []byte
}

func NewEventWriter(ch chan<- Event) *EventWriter {
	return &EventWriter{ch: ch}
}

func (w *EventWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	w.buf = append(w.buf, p...)
	for {
		idx := -1
		for i, b := range w.buf {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}
		line := string(w.buf[:idx])
		w.buf = w.buf[idx+1:]
		EmitLog(w.ch, line)
	}
	return n, nil
}

// Flush sends any remaining buffered content.
func (w *EventWriter) Flush() {
	if len(w.buf) > 0 {
		EmitLog(w.ch, string(w.buf))
		w.buf = nil
	}
}
