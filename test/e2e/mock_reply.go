package e2e

import (
	"context"
	"sync"

	"github.com/smy-101/cc-connect/internal/app"
)

// MockReplySender is a mock implementation of app.ReplySender for testing.
type MockReplySender struct {
	mu        sync.Mutex
	responses []string
	err       error
}

// NewMockReplySender creates a new MockReplySender.
func NewMockReplySender() *MockReplySender {
	return &MockReplySender{
		responses: make([]string, 0),
	}
}

// SendReply implements app.ReplySender.
func (m *MockReplySender) SendReply(ctx context.Context, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.err != nil {
		return m.err
	}

	m.responses = append(m.responses, content)
	return nil
}

// SetError sets an error to be returned by SendReply.
func (m *MockReplySender) SetError(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.err = err
}

// GetResponses returns all sent responses.
func (m *MockReplySender) GetResponses() []string {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]string, len(m.responses))
	copy(result, m.responses)
	return result
}

// LastResponse returns the last sent response.
func (m *MockReplySender) LastResponse() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.responses) == 0 {
		return ""
	}
	return m.responses[len(m.responses)-1]
}

// Clear clears all stored responses.
func (m *MockReplySender) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = make([]string, 0)
}

// Verify MockReplySender implements app.ReplySender
var _ app.ReplySender = (*MockReplySender)(nil)
