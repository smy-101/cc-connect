package claudecode

import (
	"errors"
	"strings"
	"testing"
)

// TestParseFromReaderBasic tests ParseFromReader function
func TestParseFromReaderBasic(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{
			name:  "single event",
			input: `{"type":"system","subtype":"init"}`,
			want:  1,
		},
		{
			name: "multiple events",
			input: `{"type":"system","subtype":"init"}
{"type":"result","subtype":"success"}`,
			want: 2,
		},
		{
			name:  "empty input",
			input: ``,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			var count int
			err := ParseFromReader(reader, func(event *StreamEvent) error {
				count++
				return nil
			})

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if count != tt.want {
				t.Errorf("expected %d events, got %d", tt.want, count)
			}
		})
	}
}

// TestParseFromReaderWithCallbackError tests that callback errors are propagated
func TestParseFromReaderWithCallbackError(t *testing.T) {
	input := `{"type":"system","subtype":"init"}
{"type":"result","subtype":"success"}`
	reader := strings.NewReader(input)

	testErr := errors.New("callback error")
	callCount := 0
	err := ParseFromReader(reader, func(event *StreamEvent) error {
		callCount++
		if callCount == 1 {
			return testErr
		}
		return nil
	})

	if err != testErr {
		t.Errorf("expected callback error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected 1 callback, got %d", callCount)
	}
}

// TestParseFromReaderWithInvalidJSON tests handling of invalid JSON lines
func TestParseFromReaderWithInvalidJSON(t *testing.T) {
	input := `{"type":"system","subtype":"init"}
invalid json line
{"type":"result","subtype":"success"}`
	reader := strings.NewReader(input)

	var count int
	err := ParseFromReader(reader, func(event *StreamEvent) error {
		count++
		return nil
	})

	// Should not error, just skip invalid lines
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Should only get 2 valid events
	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}
}

// TestParseFromReaderWithEmptyLines tests handling of empty lines
func TestParseFromReaderWithEmptyLines(t *testing.T) {
	input := `{"type":"system","subtype":"init"}

{"type":"result","subtype":"success"}`
	reader := strings.NewReader(input)

	var count int
	err := ParseFromReader(reader, func(event *StreamEvent) error {
		count++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 events, got %d", count)
	}
}

// TestStreamParserResetMethod tests the Reset method
func TestStreamParserResetMethod(t *testing.T) {
	parser := NewStreamParser()

	// Parse some data with newline to complete the event
	input := "{\"type\":\"system\",\"subtype\":\"init\"}\n"
	events, err := parser.Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	// Reset
	parser.Reset()

	// Parse new data - should start fresh
	input2 := "{\"type\":\"result\",\"subtype\":\"success\"}\n"
	events, err = parser.Parse([]byte(input2))
	if err != nil {
		t.Fatalf("Parse() after Reset() error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event after reset, got %d", len(events))
	}
	if events[0].Type != "result" {
		t.Errorf("expected result event, got %s", events[0].Type)
	}
}

// TestStreamParserWithMultipleChunks tests parsing data received in chunks
func TestStreamParserWithMultipleChunks(t *testing.T) {
	parser := NewStreamParser()

	// Send first chunk (incomplete)
	events, err := parser.Parse([]byte("{\"type\":\"system\""))
	if err != nil {
		t.Errorf("Parse() error: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events for incomplete data, got %d", len(events))
	}

	// Send second chunk (complete the line)
	events, err = parser.Parse([]byte(",\"subtype\":\"init\"}\n"))
	if err != nil {
		t.Errorf("Parse() error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event after completing line, got %d", len(events))
	}
}

// TestStreamParserWithInvalidJSON tests handling of invalid JSON in stream
func TestStreamParserWithInvalidJSON(t *testing.T) {
	parser := NewStreamParser()

	// Send invalid JSON
	events, err := parser.Parse([]byte("not valid json\n"))
	if err != nil {
		t.Errorf("Parse() error: %v", err)
	}
	// Invalid JSON should be skipped
	if len(events) != 0 {
		t.Errorf("expected 0 events for invalid JSON, got %d", len(events))
	}

	// Send valid JSON after
	events, err = parser.Parse([]byte("{\"type\":\"result\",\"subtype\":\"success\"}\n"))
	if err != nil {
		t.Errorf("Parse() error: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event for valid JSON, got %d", len(events))
	}
}
