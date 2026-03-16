package core

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// 会话管理错误定义
var (
	// ErrSessionNotFound 会话不存在
	ErrSessionNotFound = errors.New("session not found")
	// ErrInvalidStateTransition 无效的状态转换
	ErrInvalidStateTransition = errors.New("invalid session state transition")
)

// SessionID 会话标识符
// 格式: platform:type:identifier
// 私聊: "feishu:user:ou_xxx"
// 群聊: "feishu:channel:oc_xxx"
type SessionID string

// SessionStatus 会话状态枚举
type SessionStatus string

const (
	// SessionStatusActive 活跃状态
	SessionStatusActive SessionStatus = "active"
	// SessionStatusArchived 已归档状态
	SessionStatusArchived SessionStatus = "archived"
	// SessionStatusDestroyed 已销毁状态
	SessionStatusDestroyed SessionStatus = "destroyed"
)

// SessionConfig 会话管理配置
type SessionConfig struct {
	ActiveTTL       time.Duration // 活跃会话超时，默认 30 分钟
	ArchivedTTL     time.Duration // 归档会话超时，默认 24 小时
	CleanupInterval time.Duration // 清理间隔，默认 5 分钟
}

// DefaultSessionConfig 返回默认配置
func DefaultSessionConfig() SessionConfig {
	return SessionConfig{
		ActiveTTL:       30 * time.Minute,
		ArchivedTTL:     24 * time.Hour,
		CleanupInterval: 5 * time.Minute,
	}
}

// Session 会话状态
type Session struct {
	ID             SessionID
	Status         SessionStatus
	AgentID        string
	PermissionMode string
	Metadata       map[string]string
	CreatedAt      time.Time
	LastActiveAt   time.Time
	ArchivedAt     *time.Time
}

// NewSession 创建新会话
func NewSession(id SessionID) *Session {
	now := time.Now()
	return &Session{
		ID:           id,
		Status:       SessionStatusActive,
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		LastActiveAt: now,
	}
}

// BindAgent 绑定代理
func (s *Session) BindAgent(agentID string) {
	s.AgentID = agentID
}

// SetPermissionMode 设置权限模式
func (s *Session) SetPermissionMode(mode string) {
	s.PermissionMode = mode
}

// SetMetadata 设置元数据
func (s *Session) SetMetadata(key, value string) {
	if s.Metadata == nil {
		s.Metadata = make(map[string]string)
	}
	s.Metadata[key] = value
}

// Touch 更新最后活跃时间
func (s *Session) Touch() {
	s.LastActiveAt = time.Now()
}

// Clone 返回会话的深拷贝
func (s *Session) Clone() *Session {
	clone := &Session{
		ID:             s.ID,
		Status:         s.Status,
		AgentID:        s.AgentID,
		PermissionMode: s.PermissionMode,
		CreatedAt:      s.CreatedAt,
		LastActiveAt:   s.LastActiveAt,
	}
	if s.Metadata != nil {
		clone.Metadata = make(map[string]string)
		for k, v := range s.Metadata {
			clone.Metadata[k] = v
		}
	}
	if s.ArchivedAt != nil {
		archivedAt := *s.ArchivedAt
		clone.ArchivedAt = &archivedAt
	}
	return clone
}

// SessionManager 会话管理器
type SessionManager struct {
	sessions map[SessionID]*Session
	config   SessionConfig
	mu       sync.RWMutex
	now      func() time.Time // 可注入的时间函数
}

// NewSessionManager 创建会话管理器
func NewSessionManager(config SessionConfig) *SessionManager {
	return &SessionManager{
		sessions: make(map[SessionID]*Session),
		config:   config,
		now:      time.Now,
	}
}

// GetOrCreate 获取或创建会话
func (m *SessionManager) GetOrCreate(id SessionID) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		return session.Clone()
	}

	// 使用管理器的 now 函数创建会话
	now := m.now()
	session := &Session{
		ID:           id,
		Status:       SessionStatusActive,
		Metadata:     make(map[string]string),
		CreatedAt:    now,
		LastActiveAt: now,
	}
	m.sessions[id] = session
	return session.Clone()
}

// Get 获取会话（返回副本）
func (m *SessionManager) Get(id SessionID) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if session, ok := m.sessions[id]; ok {
		return session.Clone(), true
	}
	return nil, false
}

// Update 更新会话（传入修改函数）
func (m *SessionManager) Update(id SessionID, fn func(*Session)) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if session, ok := m.sessions[id]; ok {
		fn(session)
		return session.Clone(), true
	}
	return nil, false
}

// Archive 归档会话
func (m *SessionManager) Archive(id SessionID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	// 只有 active 状态可以归档
	if session.Status != SessionStatusActive {
		return ErrInvalidStateTransition
	}

	now := m.now()
	session.Status = SessionStatusArchived
	session.ArchivedAt = &now
	return nil
}

// Destroy 销毁会话
func (m *SessionManager) Destroy(id SessionID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, ok := m.sessions[id]
	if !ok {
		return ErrSessionNotFound
	}

	delete(m.sessions, id)
	return nil
}

// cleanup 清理过期会话（私有方法）
func (m *SessionManager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	for id, session := range m.sessions {
		if session.Status == SessionStatusActive {
			// 检查是否超时未活跃
			if now.Sub(session.LastActiveAt) > m.config.ActiveTTL {
				session.Status = SessionStatusArchived
				session.ArchivedAt = &now
			}
		} else if session.Status == SessionStatusArchived {
			// 检查归档是否超时
			if session.ArchivedAt != nil && now.Sub(*session.ArchivedAt) > m.config.ArchivedTTL {
				delete(m.sessions, id)
			}
		}
	}
}

// StartCleanup 启动后台清理 goroutine
func (m *SessionManager) StartCleanup(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.config.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.cleanup()
			}
		}
	}()
}

// DeriveSessionID 从消息派生会话 ID
// 规则：优先使用 channel（群聊），否则使用 userID（私聊）
func DeriveSessionID(msg *Message) SessionID {
	if msg.ChannelID != "" {
		return SessionID(fmt.Sprintf("%s:channel:%s", msg.Platform, msg.ChannelID))
	}
	return SessionID(fmt.Sprintf("%s:user:%s", msg.Platform, msg.UserID))
}

// List returns a list of all active sessions (clones).
func (m *SessionManager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		result = append(result, session.Clone())
	}
	return result
}
