package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gclaw/pkg/types"
)

// MemoryLevel 记忆级别
type MemoryLevel string

const (
	MemoryLevelShortTerm MemoryLevel = "short_term" // 短期记忆：最近几轮对话
	MemoryLevelNearTerm  MemoryLevel = "near_term"  // 近端记忆：当前会话
	MemoryLevelLongTerm  MemoryLevel = "long_term"  // 长期记忆：持久化存储
)

// MemoryEntry 记忆条目
type MemoryEntry struct {
	ID        string                 `json:"id"`
	Level     MemoryLevel            `json:"level"`
	Message   types.Message          `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Version   int                    `json:"version"`
	Checksum  string                 `json:"checksum"`
}

// MemorySummary 记忆摘要
type MemorySummary struct {
	SessionID   string    `json:"session_id"`
	Summary     string    `json:"summary"`
	KeyPoints   []string  `json:"key_points,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	MessageCount int      `json:"message_count"`
	Version     int       `json:"version"`
}

// KnowledgeNode 知识图谱节点
type KnowledgeNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"` // entity, concept, event
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// KnowledgeEdge 知识图谱边
type KnowledgeEdge struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"` // 源节点 ID
	Target     string                 `json:"target"` // 目标节点 ID
	Relation   string                 `json:"relation"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// KnowledgeGraph 知识图谱
type KnowledgeGraph struct {
	mu     sync.RWMutex
	nodes  map[string]*KnowledgeNode
	edges  map[string]*KnowledgeEdge
	index  map[string][]string // 按类型索引
}

// VersionedMemory 带版本控制的记忆
type VersionedMemory struct {
	*InMemoryMemory
	mu             sync.RWMutex
	shortTerm      map[string][]*MemoryEntry    // session -> entries
	nearTerm       map[string][]*MemoryEntry    // session -> entries
	longTerm       map[string][]*MemoryEntry    // session -> entries
	summaries      map[string]*MemorySummary    // session -> summary
	knowledgeGraph *KnowledgeGraph
	versionHistory map[string][]*MemorySnapshot // session -> snapshots
	maxShortTerm   int
	maxNearTerm    int
}

// MemorySnapshot 记忆快照
type MemorySnapshot struct {
	SessionID string               `json:"session_id"`
	Version   int                  `json:"version"`
	Timestamp time.Time            `json:"timestamp"`
	Entries   []*MemoryEntry       `json:"entries"`
	Summary   *MemorySummary       `json:"summary,omitempty"`
}

// NewVersionedMemory 创建带版本控制的三级记忆系统
func NewVersionedMemory(maxSize, maxShortTerm, maxNearTerm int) *VersionedMemory {
	return &VersionedMemory{
		InMemoryMemory: NewInMemoryMemory(maxSize),
		shortTerm:      make(map[string][]*MemoryEntry),
		nearTerm:       make(map[string][]*MemoryEntry),
		longTerm:       make(map[string][]*MemoryEntry),
		summaries:      make(map[string]*MemorySummary),
		knowledgeGraph: NewKnowledgeGraph(),
		versionHistory: make(map[string][]*MemorySnapshot),
		maxShortTerm:   maxShortTerm,
		maxNearTerm:    maxNearTerm,
	}
}

// NewKnowledgeGraph 创建知识图谱
func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{
		nodes: make(map[string]*KnowledgeNode),
		edges: make(map[string]*KnowledgeEdge),
		index: make(map[string][]string),
	}
}

// Add 添加消息到三级记忆系统
func (m *VersionedMemory) Add(sessionID string, message types.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := &MemoryEntry{
		ID:        generateID(),
		Level:     MemoryLevelShortTerm,
		Message:   message,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Version:   1,
		Checksum:  calculateChecksum(message),
	}

	// 添加到短期记忆
	m.shortTerm[sessionID] = append(m.shortTerm[sessionID], entry)
	if len(m.shortTerm[sessionID]) > m.maxShortTerm {
		// 移动到近端记忆
		oldest := m.shortTerm[sessionID][0]
		oldest.Level = MemoryLevelNearTerm
		m.nearTerm[sessionID] = append(m.nearTerm[sessionID], oldest)
		m.shortTerm[sessionID] = m.shortTerm[sessionID][1:]
	}

	// 同时添加到基础记忆
	if err := m.InMemoryMemory.Add(sessionID, message); err != nil {
		return err
	}

	// 更新摘要
	m.updateSummary(sessionID)

	// 保存版本快照
	m.saveSnapshot(sessionID)

	return nil
}

// GetShortTerm 获取短期记忆
func (m *VersionedMemory) GetShortTerm(sessionID string, limit int) ([]*MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := m.shortTerm[sessionID]
	if len(entries) == 0 {
		return []*MemoryEntry{}, nil
	}

	start := len(entries) - limit
	if start < 0 {
		start = 0
	}

	return entries[start:], nil
}

// GetNearTerm 获取近端记忆
func (m *VersionedMemory) GetNearTerm(sessionID string, limit int) ([]*MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := m.nearTerm[sessionID]
	if len(entries) == 0 {
		return []*MemoryEntry{}, nil
	}

	start := len(entries) - limit
	if start < 0 {
		start = 0
	}

	return entries[start:], nil
}

// GetLongTerm 获取长期记忆
func (m *VersionedMemory) GetLongTerm(sessionID string, limit int) ([]*MemoryEntry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entries := m.longTerm[sessionID]
	if len(entries) == 0 {
		return []*MemoryEntry{}, nil
	}

	start := len(entries) - limit
	if start < 0 {
		start = 0
	}

	return entries[start:], nil
}

// ConsolidateToLongTerm 将记忆整合到长期记忆（带压缩）
func (m *VersionedMemory) ConsolidateToLongTerm(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	nearEntries := m.nearTerm[sessionID]
	if len(nearEntries) == 0 {
		return nil
	}

	// 压缩和总结
	summary := m.compressAndSummarize(nearEntries)

	// 创建压缩后的长期记忆条目
	compressedEntry := &MemoryEntry{
		ID:        generateID(),
		Level:     MemoryLevelLongTerm,
		Message:   types.Message{Role: types.RoleSystem, Content: summary},
		Timestamp: time.Now(),
		SessionID: sessionID,
		Version:   1,
		Metadata:  map[string]interface{}{"compressed": true, "original_count": len(nearEntries)},
	}

	m.longTerm[sessionID] = append(m.longTerm[sessionID], compressedEntry)

	// 清理近端记忆
	m.nearTerm[sessionID] = nil

	return nil
}

// compressAndSummarize 压缩和总结记忆
func (m *VersionedMemory) compressAndSummarize(entries []*MemoryEntry) string {
	if len(entries) == 0 {
		return ""
	}

	var keyPoints []string
	messageContents := make([]string, 0, len(entries))

	for _, entry := range entries {
		if entry.Message.Content != "" {
			messageContents = append(messageContents, entry.Message.Content)
		}
	}

	// 简单实现：提取关键点（实际应该使用 AI 进行智能总结）
	summary := fmt.Sprintf("Session summary: %d messages processed", len(entries))
	if len(messageContents) > 0 {
		keyPoints = append(keyPoints, fmt.Sprintf("Topics discussed: %d items", len(messageContents)))
	}

	return fmt.Sprintf("%s\nKey points:\n- %s", summary, joinStrings(keyPoints, "\n- "))
}

// GetSummary 获取会话摘要
func (m *VersionedMemory) GetSummary(sessionID string) (*MemorySummary, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	summary, exists := m.summaries[sessionID]
	if !exists {
		return nil, fmt.Errorf("no summary found for session %s", sessionID)
	}

	return summary, nil
}

// updateSummary 更新会话摘要
func (m *VersionedMemory) updateSummary(sessionID string) {
	shortEntries := m.shortTerm[sessionID]
	nearEntries := m.nearTerm[sessionID]
	longEntries := m.longTerm[sessionID]

	totalCount := len(shortEntries) + len(nearEntries) + len(longEntries)

	summary, exists := m.summaries[sessionID]
	if !exists {
		summary = &MemorySummary{
			SessionID: sessionID,
			CreatedAt: time.Now(),
		}
	}

	summary.UpdatedAt = time.Now()
	summary.MessageCount = totalCount
	summary.Version++
	summary.Summary = fmt.Sprintf("Session with %d messages", totalCount)

	m.summaries[sessionID] = summary
}

// SaveSnapshot 保存当前记忆状态快照
func (m *VersionedMemory) saveSnapshot(sessionID string) {
	allEntries := append(append(m.shortTerm[sessionID], m.nearTerm[sessionID]...), m.longTerm[sessionID]...)

	snapshot := &MemorySnapshot{
		SessionID: sessionID,
		Version:   len(m.versionHistory[sessionID]) + 1,
		Timestamp: time.Now(),
		Entries:   allEntries,
		Summary:   m.summaries[sessionID],
	}

	m.versionHistory[sessionID] = append(m.versionHistory[sessionID], snapshot)

	// 限制快照数量
	if len(m.versionHistory[sessionID]) > 10 {
		m.versionHistory[sessionID] = m.versionHistory[sessionID][1:]
	}
}

// Rollback 回滚到指定版本
func (m *VersionedMemory) Rollback(sessionID string, version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	snapshots := m.versionHistory[sessionID]
	if len(snapshots) == 0 {
		return fmt.Errorf("no snapshots available for session %s", sessionID)
	}

	var targetSnapshot *MemorySnapshot
	for _, snapshot := range snapshots {
		if snapshot.Version == version {
			targetSnapshot = snapshot
			break
		}
	}

	if targetSnapshot == nil {
		return fmt.Errorf("snapshot version %d not found", version)
	}

	// 恢复记忆状态
	m.shortTerm[sessionID] = nil
	m.nearTerm[sessionID] = nil
	m.longTerm[sessionID] = nil

	for _, entry := range targetSnapshot.Entries {
		switch entry.Level {
		case MemoryLevelShortTerm:
			m.shortTerm[sessionID] = append(m.shortTerm[sessionID], entry)
		case MemoryLevelNearTerm:
			m.nearTerm[sessionID] = append(m.nearTerm[sessionID], entry)
		case MemoryLevelLongTerm:
			m.longTerm[sessionID] = append(m.longTerm[sessionID], entry)
		}
	}

	if targetSnapshot.Summary != nil {
		m.summaries[sessionID] = targetSnapshot.Summary
	}

	return nil
}

// CompareVersions 比较两个版本的差异
func (m *VersionedMemory) CompareVersions(sessionID string, version1, version2 int) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshots := m.versionHistory[sessionID]
	if len(snapshots) == 0 {
		return nil, fmt.Errorf("no snapshots available")
	}

	var snap1, snap2 *MemorySnapshot
	for _, snapshot := range snapshots {
		if snapshot.Version == version1 {
			snap1 = snapshot
		}
		if snapshot.Version == version2 {
			snap2 = snapshot
		}
	}

	if snap1 == nil || snap2 == nil {
		return nil, fmt.Errorf("one or both versions not found")
	}

	diff := map[string]interface{}{
		"version1":      version1,
		"version2":      version2,
		"entries_diff":  len(snap2.Entries) - len(snap1.Entries),
		"timestamp1":    snap1.Timestamp,
		"timestamp2":    snap2.Timestamp,
	}

	return diff, nil
}

// KnowledgeGraph 相关方法

// AddNode 添加知识节点
func (kg *KnowledgeGraph) AddNode(node *KnowledgeNode) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if node.ID == "" {
		node.ID = generateID()
	}

	kg.nodes[node.ID] = node
	kg.index[node.Type] = append(kg.index[node.Type], node.ID)

	return nil
}

// AddEdge 添加知识边
func (kg *KnowledgeGraph) AddEdge(edge *KnowledgeEdge) error {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	if edge.ID == "" {
		edge.ID = generateID()
	}

	// 验证节点存在
	if _, exists := kg.nodes[edge.Source]; !exists {
		return fmt.Errorf("source node %s not found", edge.Source)
	}
	if _, exists := kg.nodes[edge.Target]; !exists {
		return fmt.Errorf("target node %s not found", edge.Target)
	}

	kg.edges[edge.ID] = edge
	return nil
}

// GetNode 获取节点
func (kg *KnowledgeGraph) GetNode(id string) (*KnowledgeNode, bool) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()
	node, exists := kg.nodes[id]
	return node, exists
}

// GetRelatedNodes 获取相关节点
func (kg *KnowledgeGraph) GetRelatedNodes(nodeID string) ([]*KnowledgeNode, []*KnowledgeEdge, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	var relatedNodes []*KnowledgeNode
	var relatedEdges []*KnowledgeEdge

	for _, edge := range kg.edges {
		if edge.Source == nodeID {
			if target, exists := kg.nodes[edge.Target]; exists {
				relatedNodes = append(relatedNodes, target)
				relatedEdges = append(relatedEdges, edge)
			}
		}
		if edge.Target == nodeID {
			if source, exists := kg.nodes[edge.Source]; exists {
				relatedNodes = append(relatedNodes, source)
				relatedEdges = append(relatedEdges, edge)
			}
		}
	}

	return relatedNodes, relatedEdges, nil
}

// QueryByType 按类型查询节点
func (kg *KnowledgeGraph) QueryByType(nodeType string) ([]*KnowledgeNode, error) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	ids := kg.index[nodeType]
	var nodes []*KnowledgeNode

	for _, id := range ids {
		if node, exists := kg.nodes[id]; exists {
			nodes = append(nodes, node)
		}
	}

	return nodes, nil
}

// 辅助函数

func generateID() string {
	data := fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:8])
}

func calculateChecksum(message types.Message) string {
	data, _ := json.Marshal(message)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
