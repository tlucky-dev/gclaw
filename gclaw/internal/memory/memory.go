package memory

import (
	"container/list"
	"sync"
	"time"

	"gclaw/pkg/types"
)

// Memory 记忆接口
type Memory interface {
	Add(sessionID string, message types.Message) error
	Get(sessionID string, limit int) ([]types.Message, error)
	Clear(sessionID string) error
	Delete(sessionID string, messageIndex int) error
}

// InMemoryMemory 基于内存的记忆实现
type InMemoryMemory struct {
	mu       sync.RWMutex
	sessions map[string]*list.List
	maxSize  int
}

// NewInMemoryMemory 创建新的内存存储
func NewInMemoryMemory(maxSize int) *InMemoryMemory {
	return &InMemoryMemory{
		sessions: make(map[string]*list.List),
		maxSize:  maxSize,
	}
}

// Add 添加消息
func (m *InMemoryMemory) Add(sessionID string, message types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.sessions[sessionID]; !exists {
		m.sessions[sessionID] = list.New()
	}

	session := m.sessions[sessionID]
	session.PushBack(message)

	// 如果超出最大大小，移除最早的消息
	for session.Len() > m.maxSize {
		session.Remove(session.Front())
	}

	return nil
}

// Get 获取消息
func (m *InMemoryMemory) Get(sessionID string, limit int) ([]types.Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return []types.Message{}, nil
	}

	result := make([]types.Message, 0)
	count := 0
	
	// 从后向前遍历，获取最新的消息
	for e := session.Back(); e != nil && count < limit; e = e.Prev() {
		if msg, ok := e.Value.(types.Message); ok {
			result = append(result, msg)
			count++
		}
	}

	// 反转结果，使其按时间顺序排列
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// Clear 清空会话
func (m *InMemoryMemory) Clear(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// Delete 删除指定消息
func (m *InMemoryMemory) Delete(sessionID string, messageIndex int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil
	}

	if messageIndex < 0 || messageIndex >= session.Len() {
		return nil
	}

	var element *list.Element
	i := 0
	for e := session.Front(); e != nil; e = e.Next() {
		if i == messageIndex {
			element = e
			break
		}
		i++
	}

	if element != nil {
		session.Remove(element)
	}

	return nil
}

// ExpiringMessage 带过期时间的消息
type ExpiringMessage struct {
	Message   types.Message
	ExpiresAt time.Time
}

// ExpiringMemory 带过期时间的内存存储
type ExpiringMemory struct {
	*InMemoryMemory
	expiration time.Duration
}

// NewExpiringMemory 创建带过期时间的内存存储
func NewExpiringMemory(maxSize int, expiration time.Duration) *ExpiringMemory {
	return &ExpiringMemory{
		InMemoryMemory: NewInMemoryMemory(maxSize),
		expiration:     expiration,
	}
}
