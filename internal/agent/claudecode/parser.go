package claudecode

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
)

// Parse errors
var (
	ErrEmptyInput = errors.New("empty input")
	ErrInvalidJSON = errors.New("invalid JSON")
)

// ParseEvent parses a single JSONL line into a StreamEvent
func ParseEvent(data []byte) (*StreamEvent, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	var event StreamEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, ErrInvalidJSON
	}

	return &event, nil
}

// StreamParser handles streaming JSONL parsing with buffering for incomplete lines
type StreamParser struct {
	buffer bytes.Buffer
}

// NewStreamParser creates a new StreamParser
func NewStreamParser() *StreamParser {
	return &StreamParser{}
}

// Parse processes input data and returns complete events.
// It handles incomplete lines by buffering them until a newline is received.
func (p *StreamParser) Parse(data []byte) ([]*StreamEvent, error) {
	p.buffer.Write(data)

	var events []*StreamEvent

	// Process complete lines from the buffer
	for {
		line, err := p.buffer.ReadBytes('\n')
		if err == io.EOF {
			// No more complete lines, put the partial line back
			if len(line) > 0 {
				p.buffer.Write(line)
			}
			break
		}
		if err != nil {
			return nil, err
		}

		// Remove the newline character
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}

		event, err := ParseEvent(line)
		if err != nil {
			// Skip invalid lines but continue parsing
			continue
		}
		events = append(events, event)
	}

	return events, nil
}

// Reset clears the internal buffer
func (p *StreamParser) Reset() {
	p.buffer.Reset()
}

// ParseFromReader reads JSONL events from an io.Reader and sends them to a callback.
// This is useful for processing stdout from the Claude Code CLI.
func ParseFromReader(reader io.Reader, callback func(*StreamEvent) error) error {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		event, err := ParseEvent(line)
		if err != nil {
			// Skip invalid lines
			continue
		}

		if err := callback(event); err != nil {
			return err
		}
	}
	return scanner.Err()
}
