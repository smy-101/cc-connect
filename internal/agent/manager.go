package agent

import (
	"fmt"
	"sync"
)

// Manager manages multiple Agent instances keyed by session ID
type Manager struct {
	agents map[string]Agent
	mu     sync.RWMutex
}

// NewManager creates a new Agent Manager
func NewManager() *Manager {
	return &Manager{
		agents: make(map[string]Agent),
	}
}

// AgentFactory is a function that creates a new Agent
type AgentFactory func() (Agent, error)

// GetOrCreate returns an existing agent for the session, or creates a new one using the factory
func (m *Manager) GetOrCreate(sessionID string, factory AgentFactory) (Agent, error) {
	m.mu.RLock()
	if ag, exists := m.agents[sessionID]; exists {
		m.mu.RUnlock()
		return ag, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double check after acquiring write lock
	if ag, exists := m.agents[sessionID]; exists {
		return ag, nil
	}

	ag, err := factory()
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}

	m.agents[sessionID] = ag
	return ag, nil
}

// Get returns the agent for the given session ID, or nil if not found
func (m *Manager) Get(sessionID string) Agent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.agents[sessionID]
}

// Remove stops and removes the agent for the given session ID
func (m *Manager) Remove(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ag, exists := m.agents[sessionID]
	if !exists {
		return nil
	}

	// Stop the agent
	if err := ag.Stop(); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	delete(m.agents, sessionID)
	return nil
}

// List returns all session IDs with active agents
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]string, 0, len(m.agents))
	for sessionID := range m.agents {
		sessions = append(sessions, sessionID)
	}
	return sessions
}

// StopAll stops all agents
func (m *Manager) StopAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for sessionID, ag := range m.agents {
		if err := ag.Stop(); err != nil {
			lastErr = fmt.Errorf("failed to stop agent %s: %w", sessionID, err)
		}
	}

	// Clear the map
	m.agents = make(map[string]Agent)
	return lastErr
}

// Count returns the number of active agents
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.agents)
}
