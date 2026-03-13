package feishu

import (
	"context"
	"sync"
)

// MockClient is a mock implementation of FeishuClient for testing purposes.
type MockClient struct {
	mu sync.RWMutex

	// Connection state
	connected bool

	// Event handler
	eventHandler EventHandler

	// Track method calls
	ConnectCalled    int
	DisconnectCalled int
	SendTextCalled   int
	OnEventCalled    int

	// Additional tracking for integration tests
	SendTextCalledCount int

	// Configurable behavior
	ConnectError    error
	SendTextError   error
	SendTextHandler func(chatID, content string) error

	// Recorded calls
	LastSendTextChatID  string
	LastSendTextContent string
}

// NewMockClient creates a new MockClient instance.
func NewMockClient() *MockClient {
	return &MockClient{}
}

// Connect implements FeishuClient.Connect.
func (m *MockClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ConnectCalled++

	if m.ConnectError != nil {
		return m.ConnectError
	}

	m.connected = true
	return nil
}

// Disconnect implements FeishuClient.Disconnect.
func (m *MockClient) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.DisconnectCalled++
	m.connected = false

	return nil
}

// IsConnected implements FeishuClient.IsConnected.
func (m *MockClient) IsConnected() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.connected
}

// SendText implements FeishuClient.SendText.
func (m *MockClient) SendText(ctx context.Context, chatID, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SendTextCalled++
	m.SendTextCalledCount++
	m.LastSendTextChatID = chatID
	m.LastSendTextContent = content

	if m.SendTextHandler != nil {
		return m.SendTextHandler(chatID, content)
	}

	return m.SendTextError
}

// OnEvent implements FeishuClient.OnEvent.
func (m *MockClient) OnEvent(handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OnEventCalled++
	m.eventHandler = handler
}

// SimulateMessageEvent simulates receiving a message event.
// This is useful for testing event handling logic.
func (m *MockClient) SimulateMessageEvent(ctx context.Context, event *MessageReceiveEvent) error {
	m.mu.RLock()
	handler := m.eventHandler
	m.mu.RUnlock()

	if handler == nil {
		return nil
	}

	return handler(ctx, event)
}

// Reset resets all recorded state.
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.connected = false
	m.ConnectCalled = 0
	m.DisconnectCalled = 0
	m.SendTextCalled = 0
	m.SendTextCalledCount = 0
	m.OnEventCalled = 0
	m.ConnectError = nil
	m.SendTextError = nil
	m.SendTextHandler = nil
	m.LastSendTextChatID = ""
	m.LastSendTextContent = ""
	m.eventHandler = nil
}
