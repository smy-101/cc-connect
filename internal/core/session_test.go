package core

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestDeriveSessionID 测试从消息派生会话 ID
func TestDeriveSessionID(t *testing.T) {
	tests := []struct {
		name     string
		msg      *Message
		expected SessionID
	}{
		{
			name: "私聊消息派生用户会话 ID",
			msg: &Message{
				Platform: "feishu",
				UserID:   "ou_xxx",
			},
			expected: "feishu:user:ou_xxx",
		},
		{
			name: "群聊消息派生频道会话 ID",
			msg: &Message{
				Platform:  "feishu",
				UserID:    "ou_yyy",
				ChannelID: "oc_xxx",
			},
			expected: "feishu:channel:oc_xxx",
		},
		{
			name: "群聊优先使用频道 ID",
			msg: &Message{
				Platform:  "feishu",
				UserID:    "ou_zzz",
				ChannelID: "oc_yyy",
			},
			expected: "feishu:channel:oc_yyy",
		},
		{
			name: "不同平台的私聊",
			msg: &Message{
				Platform: "wechat",
				UserID:   "user_123",
			},
			expected: "wechat:user:user_123",
		},
		{
			name: "不同平台的群聊",
			msg: &Message{
				Platform:  "slack",
				UserID:    "U123",
				ChannelID: "C456",
			},
			expected: "slack:channel:C456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveSessionID(tt.msg)
			if result != tt.expected {
				t.Errorf("DeriveSessionID() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestDeriveSessionIDFormat 测试会话 ID 格式
func TestDeriveSessionIDFormat(t *testing.T) {
	t.Run("三段式格式", func(t *testing.T) {
		msg := &Message{
			Platform:  "feishu",
			UserID:    "ou_123",
			ChannelID: "oc_456",
		}
		id := DeriveSessionID(msg)

		// 验证三段式格式: platform:type:identifier
		if id != "feishu:channel:oc_456" {
			t.Errorf("Expected three-part format, got %q", id)
		}
	})

	t.Run("私聊格式正确", func(t *testing.T) {
		msg := &Message{
			Platform: "feishu",
			UserID:   "ou_abc",
		}
		id := DeriveSessionID(msg)

		if id != "feishu:user:ou_abc" {
			t.Errorf("Expected user format, got %q", id)
		}
	})
}

// TestSessionStatus 测试 SessionStatus 枚举
func TestSessionStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   SessionStatus
		expected string
	}{
		{"active status", SessionStatusActive, "active"},
		{"archived status", SessionStatusArchived, "archived"},
		{"destroyed status", SessionStatusDestroyed, "destroyed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.expected {
				t.Errorf("SessionStatus %s = %q, want %q", tt.name, tt.status, tt.expected)
			}
		})
	}
}

// TestSessionStruct 测试 Session 结构体字段
func TestSessionStruct(t *testing.T) {
	now := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	session := &Session{
		ID:             "feishu:user:ou_xxx",
		Status:         SessionStatusActive,
		AgentID:        "claudecode",
		PermissionMode: "default",
		Metadata:       map[string]string{"key": "value"},
		CreatedAt:      now,
		LastActiveAt:   now,
	}

	if session.ID != "feishu:user:ou_xxx" {
		t.Errorf("ID = %q, want %q", session.ID, "feishu:user:ou_xxx")
	}
	if session.Status != SessionStatusActive {
		t.Errorf("Status = %q, want %q", session.Status, SessionStatusActive)
	}
	if session.AgentID != "claudecode" {
		t.Errorf("AgentID = %q, want %q", session.AgentID, "claudecode")
	}
	if session.PermissionMode != "default" {
		t.Errorf("PermissionMode = %q, want %q", session.PermissionMode, "default")
	}
	if session.Metadata["key"] != "value" {
		t.Errorf("Metadata[key] = %q, want %q", session.Metadata["key"], "value")
	}
	if !session.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", session.CreatedAt, now)
	}
	if !session.LastActiveAt.Equal(now) {
		t.Errorf("LastActiveAt = %v, want %v", session.LastActiveAt, now)
	}
}

// TestNewSession 测试 NewSession 构造函数
func TestNewSession(t *testing.T) {
	id := SessionID("feishu:user:ou_xxx")
	session := NewSession(id)

	if session == nil {
		t.Fatal("NewSession returned nil")
	}
	if session.ID != id {
		t.Errorf("ID = %q, want %q", session.ID, id)
	}
	if session.Status != SessionStatusActive {
		t.Errorf("Status = %q, want %q", session.Status, SessionStatusActive)
	}
	if session.AgentID != "" {
		t.Errorf("AgentID should be empty, got %q", session.AgentID)
	}
	if session.PermissionMode != "" {
		t.Errorf("PermissionMode should be empty, got %q", session.PermissionMode)
	}
	if session.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if session.LastActiveAt.IsZero() {
		t.Error("LastActiveAt should be set")
	}
	if !session.LastActiveAt.Equal(session.CreatedAt) {
		t.Error("LastActiveAt should equal CreatedAt for new session")
	}
}

// TestSessionMethods 测试 Session 方法
func TestSessionMethods(t *testing.T) {
	t.Run("BindAgent", func(t *testing.T) {
		session := NewSession("feishu:user:ou_xxx")
		session.BindAgent("claudecode")
		if session.AgentID != "claudecode" {
			t.Errorf("AgentID = %q, want %q", session.AgentID, "claudecode")
		}
	})

	t.Run("SetPermissionMode", func(t *testing.T) {
		session := NewSession("feishu:user:ou_xxx")
		session.SetPermissionMode("yolo")
		if session.PermissionMode != "yolo" {
			t.Errorf("PermissionMode = %q, want %q", session.PermissionMode, "yolo")
		}
	})

	t.Run("SetMetadata", func(t *testing.T) {
		session := NewSession("feishu:user:ou_xxx")
		session.SetMetadata("key1", "value1")
		session.SetMetadata("key2", "value2")
		if session.Metadata["key1"] != "value1" {
			t.Errorf("Metadata[key1] = %q, want %q", session.Metadata["key1"], "value1")
		}
		if session.Metadata["key2"] != "value2" {
			t.Errorf("Metadata[key2] = %q, want %q", session.Metadata["key2"], "value2")
		}
	})

	t.Run("Touch", func(t *testing.T) {
		session := NewSession("feishu:user:ou_xxx")
		original := session.LastActiveAt
		time.Sleep(10 * time.Millisecond)
		session.Touch()
		if !session.LastActiveAt.After(original) {
			t.Errorf("LastActiveAt should be updated, before=%v, after=%v", original, session.LastActiveAt)
		}
	})

	t.Run("Clone", func(t *testing.T) {
		original := NewSession("feishu:user:ou_xxx")
		original.BindAgent("claudecode")
		original.SetPermissionMode("yolo")
		original.SetMetadata("key", "value")

		clone := original.Clone()

		// 验证副本内容相等
		if clone.ID != original.ID {
			t.Errorf("Clone ID = %q, want %q", clone.ID, original.ID)
		}
		if clone.AgentID != original.AgentID {
			t.Errorf("Clone AgentID = %q, want %q", clone.AgentID, original.AgentID)
		}
		if clone.PermissionMode != original.PermissionMode {
			t.Errorf("Clone PermissionMode = %q, want %q", clone.PermissionMode, original.PermissionMode)
		}

		// 验证修改副本不影响原对象
		clone.BindAgent("other")
		if original.AgentID != "claudecode" {
			t.Errorf("Modifying clone should not affect original, original.AgentID = %q", original.AgentID)
		}

		// 验证 Metadata 是深拷贝
		clone.SetMetadata("key", "modified")
		if original.Metadata["key"] != "value" {
			t.Errorf("Modifying clone metadata should not affect original, original.Metadata[key] = %q", original.Metadata["key"])
		}
	})
}

// TestNewSessionManager 测试创建 SessionManager
func TestNewSessionManager(t *testing.T) {
	t.Run("创建默认配置的 SessionManager", func(t *testing.T) {
		manager := NewSessionManager(DefaultSessionConfig())
		if manager == nil {
			t.Fatal("NewSessionManager returned nil")
		}
	})

	t.Run("默认配置值", func(t *testing.T) {
		config := DefaultSessionConfig()
		if config.ActiveTTL != 30*time.Minute {
			t.Errorf("ActiveTTL = %v, want %v", config.ActiveTTL, 30*time.Minute)
		}
		if config.ArchivedTTL != 24*time.Hour {
			t.Errorf("ArchivedTTL = %v, want %v", config.ArchivedTTL, 24*time.Hour)
		}
		if config.CleanupInterval != 5*time.Minute {
			t.Errorf("CleanupInterval = %v, want %v", config.CleanupInterval, 5*time.Minute)
		}
	})

	t.Run("初始状态为空", func(t *testing.T) {
		manager := NewSessionManager(DefaultSessionConfig())
		_, ok := manager.Get("non-existent")
		if ok {
			t.Error("Expected no session for non-existent ID")
		}
	})
}

// TestGetOrCreate 测试 GetOrCreate 方法
func TestGetOrCreate(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())

	t.Run("自动创建不存在的会话", func(t *testing.T) {
		session := manager.GetOrCreate("feishu:user:ou_xxx")
		if session == nil {
			t.Fatal("GetOrCreate returned nil")
		}
		if session.ID != "feishu:user:ou_xxx" {
			t.Errorf("ID = %q, want %q", session.ID, "feishu:user:ou_xxx")
		}
		if session.Status != SessionStatusActive {
			t.Errorf("Status = %q, want %q", session.Status, SessionStatusActive)
		}
	})

	t.Run("返回已存在的会话", func(t *testing.T) {
		id := SessionID("feishu:user:ou_yyy")
		manager.GetOrCreate(id)
		manager.Update(id, func(s *Session) {
			s.BindAgent("agent1")
		})

		// 再次获取应该返回修改后的会话
		session := manager.GetOrCreate(id)
		if session.AgentID != "agent1" {
			t.Errorf("AgentID = %q, want %q", session.AgentID, "agent1")
		}
	})

	t.Run("返回副本而非引用", func(t *testing.T) {
		id := SessionID("feishu:user:ou_zzz")
		manager.GetOrCreate(id)
		manager.Update(id, func(s *Session) {
			s.BindAgent("agent2")
		})

		session1 := manager.GetOrCreate(id)
		// 修改返回的副本不应影响内部状态
		session1.BindAgent("modified")

		session2 := manager.GetOrCreate(id)
		if session2.AgentID != "agent2" {
			t.Errorf("internal session should not be modified, AgentID = %q", session2.AgentID)
		}
	})
}

// TestSessionManagerGet 测试 Get 方法
func TestSessionManagerGet(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())

	t.Run("获取存在的会话", func(t *testing.T) {
		id := SessionID("feishu:user:ou_xxx")
		manager.GetOrCreate(id)

		session, ok := manager.Get(id)
		if !ok {
			t.Fatal("Get returned false for existing session")
		}
		if session.ID != id {
			t.Errorf("ID = %q, want %q", session.ID, id)
		}
	})

	t.Run("获取不存在的会话返回 nil false", func(t *testing.T) {
		session, ok := manager.Get("non-existent")
		if ok {
			t.Error("Get should return false for non-existent session")
		}
		if session != nil {
			t.Error("Get should return nil for non-existent session")
		}
	})

	t.Run("返回副本不影响内部状态", func(t *testing.T) {
		id := SessionID("feishu:user:ou_clone")
		manager.GetOrCreate(id)
		manager.Update(id, func(s *Session) {
			s.BindAgent("original-agent")
		})

		copy, _ := manager.Get(id)
		copy.BindAgent("copy-agent") // 修改副本

		// 再次获取，验证内部状态未被修改
		again, _ := manager.Get(id)
		if again.AgentID != "original-agent" {
			t.Errorf("internal session should not be modified, AgentID = %q", again.AgentID)
		}
	})
}

// TestSessionArchive 测试归档会话
func TestSessionArchive(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())

	t.Run("归档 active 会话", func(t *testing.T) {
		id := SessionID("feishu:user:archive_active")
		manager.GetOrCreate(id)

		err := manager.Archive(id)
		if err != nil {
			t.Fatalf("Archive failed: %v", err)
		}

		session, _ := manager.Get(id)
		if session.Status != SessionStatusArchived {
			t.Errorf("Status = %q, want %q", session.Status, SessionStatusArchived)
		}
		if session.ArchivedAt == nil {
			t.Error("ArchivedAt should be set")
		}
	})

	t.Run("归档不存在的会话返回错误", func(t *testing.T) {
		err := manager.Archive("non-existent")
		if err == nil {
			t.Error("Archive should return error for non-existent session")
		}
		if err != ErrSessionNotFound {
			t.Errorf("error should be ErrSessionNotFound, got: %v", err)
		}
	})
}

// TestSessionDestroy 测试销毁会话
func TestSessionDestroy(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())

	t.Run("销毁存在的会话", func(t *testing.T) {
		id := SessionID("feishu:user:destroy_test")
		manager.GetOrCreate(id)

		err := manager.Destroy(id)
		if err != nil {
			t.Fatalf("Destroy failed: %v", err)
		}

		_, ok := manager.Get(id)
		if ok {
			t.Error("session should be destroyed")
		}
	})

	t.Run("销毁不存在的会话返回错误", func(t *testing.T) {
		err := manager.Destroy("non-existent")
		if err == nil {
			t.Error("Destroy should return error for non-existent session")
		}
		if err != ErrSessionNotFound {
			t.Errorf("error should be ErrSessionNotFound, got: %v", err)
		}
	})
}

// TestSessionStateTransition 测试状态转换
func TestSessionStateTransition(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())

	t.Run("archived 状态无法转换回 active", func(t *testing.T) {
		id := SessionID("feishu:user:state_test")
		manager.GetOrCreate(id)
		manager.Archive(id)

		// 尝试再次归档已归档的会话
		err := manager.Archive(id)
		if err == nil {
			t.Error("Archive on archived session should return error")
		}
		if err != ErrInvalidStateTransition {
			t.Errorf("error should be ErrInvalidStateTransition, got: %v", err)
		}
	})

	t.Run("destroyed 会话无法归档", func(t *testing.T) {
		id := SessionID("feishu:user:state_test2")
		manager.GetOrCreate(id)
		manager.Destroy(id)

		err := manager.Archive(id)
		if err == nil {
			t.Error("Archive on destroyed session should return error")
		}
	})
}

// TestSessionCleanup 测试自动清理
func TestSessionCleanup(t *testing.T) {
	// 使用 mock 时间
	mockNow := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
	manager := &SessionManager{
		sessions: make(map[SessionID]*Session),
		config: SessionConfig{
			ActiveTTL:       30 * time.Minute,
			ArchivedTTL:     24 * time.Hour,
			CleanupInterval: 5 * time.Minute,
		},
		now: func() time.Time { return mockNow },
	}

	t.Run("active 超时自动归档", func(t *testing.T) {
		id := SessionID("feishu:user:cleanup_active")
		session := NewSession(id)
		session.LastActiveAt = mockNow.Add(-31 * time.Minute) // 超过 30 分钟
		manager.sessions[id] = session

		manager.cleanup()

		result, ok := manager.Get(id)
		if !ok {
			t.Fatal("session should still exist after archive")
		}
		if result.Status != SessionStatusArchived {
			t.Errorf("Status = %q, want %q", result.Status, SessionStatusArchived)
		}
	})

	t.Run("archived 超时自动销毁", func(t *testing.T) {
		id := SessionID("feishu:user:cleanup_archived")
		session := NewSession(id)
		session.Status = SessionStatusArchived
		archivedAt := mockNow.Add(-25 * time.Hour) // 超过 24 小时
		session.ArchivedAt = &archivedAt
		manager.sessions[id] = session

		manager.cleanup()

		_, ok := manager.Get(id)
		if ok {
			t.Error("session should be destroyed after archived TTL")
		}
	})

	t.Run("未超时会话不受影响", func(t *testing.T) {
		id := SessionID("feishu:user:cleanup_fresh")
		session := NewSession(id)
		session.LastActiveAt = mockNow.Add(-10 * time.Minute) // 未超时
		manager.sessions[id] = session

		manager.cleanup()

		_, ok := manager.Get(id)
		if !ok {
			t.Error("fresh session should not be cleaned up")
		}
	})
}

// TestStartCleanup 测试清理 goroutine
func TestStartCleanup(t *testing.T) {
	t.Run("定时执行清理", func(t *testing.T) {
		// 使用很短的清理间隔进行测试
		config := SessionConfig{
			ActiveTTL:       30 * time.Minute,
			ArchivedTTL:     24 * time.Hour,
			CleanupInterval: 50 * time.Millisecond,
		}
		manager := NewSessionManager(config)

		// 创建一个即将超时的会话
		id := SessionID("feishu:user:cleanup_timer")
		session := NewSession(id)
		session.LastActiveAt = time.Now().Add(-31 * time.Minute)
		manager.sessions[id] = session

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		manager.StartCleanup(ctx)

		// 等待清理执行
		time.Sleep(100 * time.Millisecond)

		// 验证会话被归档
		result, ok := manager.Get(id)
		if !ok {
			t.Fatal("session should exist after archive")
		}
		if result.Status != SessionStatusArchived {
			t.Errorf("Status = %q, want %q", result.Status, SessionStatusArchived)
		}
	})

	t.Run("context 取消退出", func(t *testing.T) {
		config := SessionConfig{
			CleanupInterval: 10 * time.Millisecond,
		}
		manager := NewSessionManager(config)

		ctx, cancel := context.WithCancel(context.Background())
		manager.StartCleanup(ctx)

		// 取消 context
		cancel()

		// 等待 goroutine 退出
		time.Sleep(50 * time.Millisecond)

		// goroutine 应该已退出，这里只是验证不会 panic
	})
}

// TestConcurrentGetOrCreate 测试并发 GetOrCreate
func TestConcurrentGetOrCreate(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())
	id := SessionID("feishu:user:concurrent")

	const goroutines = 100
	results := make(chan *Session, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			session := manager.GetOrCreate(id)
			results <- session
		}()
	}

	sessions := make([]*Session, 0, goroutines)
	for i := 0; i < goroutines; i++ {
		session := <-results
		sessions = append(sessions, session)
	}

	// 所有返回的会话应该有相同的 ID
	for _, session := range sessions {
		if session.ID != id {
			t.Errorf("session.ID = %q, want %q", session.ID, id)
		}
	}

	// 内部应该只创建了一个会话
	internal, ok := manager.Get(id)
	if !ok {
		t.Fatal("session should exist")
	}
	if internal.ID != id {
		t.Errorf("internal session.ID = %q, want %q", internal.ID, id)
	}
}

// TestConcurrentReadWrite 测试并发读写
func TestConcurrentReadWrite(t *testing.T) {
	manager := NewSessionManager(DefaultSessionConfig())
	id := SessionID("feishu:user:rw_test")
	manager.GetOrCreate(id)

	const goroutines = 50
	done := make(chan bool, goroutines*2)

	// 写入 goroutine
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			agentID := fmt.Sprintf("agent-%d", idx)
			manager.Update(id, func(s *Session) {
				s.BindAgent(agentID)
			})
			done <- true
		}(i)
	}

	// 读取 goroutine
	for i := 0; i < goroutines; i++ {
		go func() {
			manager.Get(id)
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < goroutines*2; i++ {
		<-done
	}
}

// TestConcurrentCleanup 测试并发清理和访问
func TestConcurrentCleanup(t *testing.T) {
	config := SessionConfig{
		ActiveTTL:       10 * time.Millisecond,
		ArchivedTTL:     20 * time.Millisecond,
		CleanupInterval: 5 * time.Millisecond,
	}
	manager := NewSessionManager(config)

	const sessions = 100
	for i := 0; i < sessions; i++ {
		id := SessionID(fmt.Sprintf("feishu:user:cleanup_%d", i))
		manager.GetOrCreate(id)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager.StartCleanup(ctx)

	const goroutines = 50
	done := make(chan bool, goroutines)

	// 访问 goroutine
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			id := SessionID(fmt.Sprintf("feishu:user:cleanup_%d", idx%sessions))
			manager.Get(id)
			manager.GetOrCreate(id)
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		<-done
	}
}

// TestMockTime 测试时间函数注入
func TestMockTime(t *testing.T) {
	t.Run("默认使用系统时间", func(t *testing.T) {
		manager := NewSessionManager(DefaultSessionConfig())
		before := time.Now()
		session := manager.GetOrCreate("feishu:user:time_test")
		after := time.Now()

		if session.CreatedAt.Before(before) || session.CreatedAt.After(after) {
			t.Errorf("CreatedAt = %v, should be between %v and %v", session.CreatedAt, before, after)
		}
	})

	t.Run("注入 mock 时间", func(t *testing.T) {
		mockNow := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
		manager := &SessionManager{
			sessions: make(map[SessionID]*Session),
			config:   DefaultSessionConfig(),
			now:      func() time.Time { return mockNow },
		}

		session := manager.GetOrCreate("feishu:user:mock_time")
		if !session.CreatedAt.Equal(mockNow) {
			t.Errorf("CreatedAt = %v, want %v", session.CreatedAt, mockNow)
		}
		if !session.LastActiveAt.Equal(mockNow) {
			t.Errorf("LastActiveAt = %v, want %v", session.LastActiveAt, mockNow)
		}
	})

	t.Run("mock 时间影响归档时间", func(t *testing.T) {
		mockNow := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
		manager := &SessionManager{
			sessions: make(map[SessionID]*Session),
			config:   DefaultSessionConfig(),
			now:      func() time.Time { return mockNow },
		}

		id := SessionID("feishu:user:archive_time")
		manager.GetOrCreate(id)
		manager.Archive(id)

		session, _ := manager.Get(id)
		if session.ArchivedAt == nil {
			t.Fatal("ArchivedAt should be set")
		}
		if !session.ArchivedAt.Equal(mockNow) {
			t.Errorf("ArchivedAt = %v, want %v", *session.ArchivedAt, mockNow)
		}
	})

	t.Run("mock 时间影响清理", func(t *testing.T) {
		mockNow := time.Date(2024, 3, 1, 12, 0, 0, 0, time.UTC)
		manager := &SessionManager{
			sessions: make(map[SessionID]*Session),
			config: SessionConfig{
				ActiveTTL:       30 * time.Minute,
				ArchivedTTL:     24 * time.Hour,
				CleanupInterval: 5 * time.Minute,
			},
			now: func() time.Time { return mockNow },
		}

		// 创建一个即将超时的会话
		id := SessionID("feishu:user:cleanup_time")
		session := NewSession(id)
		session.LastActiveAt = mockNow.Add(-31 * time.Minute)
		manager.sessions[id] = session

		// 执行清理
		manager.cleanup()

		// 验证会话被归档
		result, ok := manager.Get(id)
		if !ok {
			t.Fatal("session should still exist after archive")
		}
		if result.Status != SessionStatusArchived {
			t.Errorf("Status = %q, want %q", result.Status, SessionStatusArchived)
		}
	})
}
