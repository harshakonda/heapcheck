// Package jsonproc demonstrates escape analysis in JSON processing code.
// JSON marshaling/unmarshaling often causes escapes due to reflection.
package jsonproc

import (
	"bytes"
	"encoding/json"
	"strconv"
	"sync"
)

// Event represents a log event
type Event struct {
	Timestamp int64             `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
}

// =============================================================================
// Pattern: JSON Encoding Escapes
// =============================================================================

// EncodeBad creates new encoder each time - allocates
func EncodeBad(event Event) ([]byte, error) {
	return json.Marshal(event) // allocates buffer internally
}

// Buffer pool for JSON encoding
var bufferPool = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

// EncodeGood reuses buffers from pool
func EncodeGood(event Event) ([]byte, error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer bufferPool.Put(buf)

	enc := json.NewEncoder(buf)
	if err := enc.Encode(event); err != nil {
		return nil, err
	}

	// Make a copy since buffer will be reused
	result := make([]byte, buf.Len())
	copy(result, buf.Bytes())
	return result, nil
}

// =============================================================================
// Pattern: Avoiding Reflection with Manual JSON
// =============================================================================

// MarshalManual builds JSON without reflection - faster for hot paths
func MarshalManual(event Event) []byte {
	// Pre-calculate approximate size
	size := 100 + len(event.Message) + len(event.Level)
	for k, v := range event.Fields {
		size += len(k) + len(v) + 10
	}

	buf := make([]byte, 0, size)
	buf = append(buf, `{"timestamp":`...)
	buf = strconv.AppendInt(buf, event.Timestamp, 10)
	buf = append(buf, `,"level":"`...)
	buf = append(buf, event.Level...)
	buf = append(buf, `","message":"`...)
	buf = appendEscapedString(buf, event.Message)
	buf = append(buf, '"')

	if len(event.Fields) > 0 {
		buf = append(buf, `,"fields":{`...)
		first := true
		for k, v := range event.Fields {
			if !first {
				buf = append(buf, ',')
			}
			buf = append(buf, '"')
			buf = append(buf, k...)
			buf = append(buf, `":"`...)
			buf = appendEscapedString(buf, v)
			buf = append(buf, '"')
			first = false
		}
		buf = append(buf, '}')
	}

	buf = append(buf, '}')
	return buf
}

// appendEscapedString appends JSON-escaped string
func appendEscapedString(buf []byte, s string) []byte {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '"':
			buf = append(buf, '\\', '"')
		case '\\':
			buf = append(buf, '\\', '\\')
		case '\n':
			buf = append(buf, '\\', 'n')
		case '\r':
			buf = append(buf, '\\', 'r')
		case '\t':
			buf = append(buf, '\\', 't')
		default:
			buf = append(buf, c)
		}
	}
	return buf
}

// =============================================================================
// Pattern: Slice Growth in JSON Arrays
// =============================================================================

// ParseEventsBad doesn't pre-allocate
func ParseEventsBad(data []byte) ([]Event, error) {
	var events []Event // grows dynamically - may escape
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// ParseEventsGood pre-allocates if count is known
func ParseEventsGood(data []byte, expectedCount int) ([]Event, error) {
	events := make([]Event, 0, expectedCount) // pre-allocated
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// =============================================================================
// Pattern: Map Allocation in JSON
// =============================================================================

// NewEventBad allocates map every time
func NewEventBad(level, message string) Event {
	return Event{
		Level:   level,
		Message: message,
		Fields:  make(map[string]string), // always escapes
	}
}

// NewEventGood - nil map until needed
func NewEventGood(level, message string) Event {
	return Event{
		Level:   level,
		Message: message,
		// Fields is nil - only allocate when needed
	}
}

// AddField adds a field, allocating map only when needed
func (e *Event) AddField(key, value string) {
	if e.Fields == nil {
		e.Fields = make(map[string]string, 4) // small initial size
	}
	e.Fields[key] = value
}

// =============================================================================
// Pattern: interface{} in JSON Structures
// =============================================================================

// GenericMessageBad uses interface{} - causes boxing
type GenericMessageBad struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"` // ESCAPES anything assigned
}

// TypedMessage uses concrete type - no boxing
type TypedMessage[T any] struct {
	Type    string `json:"type"`
	Payload T      `json:"payload"` // No interface boxing
}

// EventMessage is a typed message for events
type EventMessage = TypedMessage[Event]

// =============================================================================
// Pattern: Streaming JSON
// =============================================================================

// ProcessStreamBad loads entire JSON into memory
func ProcessStreamBad(data []byte) (int, error) {
	var events []Event
	if err := json.Unmarshal(data, &events); err != nil {
		return 0, err
	}

	count := 0
	for _, e := range events {
		if e.Level == "error" {
			count++
		}
	}
	return count, nil
}

// ProcessStreamGood uses streaming decoder
func ProcessStreamGood(data []byte) (int, error) {
	dec := json.NewDecoder(bytes.NewReader(data))

	// Read opening bracket
	if _, err := dec.Token(); err != nil {
		return 0, err
	}

	count := 0
	for dec.More() {
		var event Event
		if err := dec.Decode(&event); err != nil {
			return 0, err
		}
		if event.Level == "error" {
			count++
		}
		// event goes out of scope - can be GC'd
	}

	return count, nil
}

// =============================================================================
// Benchmark Helper
// =============================================================================

// SampleEvent creates a sample event for testing
func SampleEvent() Event {
	return Event{
		Timestamp: 1699999999,
		Level:     "info",
		Message:   "Sample log message",
		Fields: map[string]string{
			"user_id": "12345",
			"action":  "login",
		},
	}
}
