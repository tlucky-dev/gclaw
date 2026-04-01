package cache

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheLevel 缓存级别
type CacheLevel string

const (
	CacheLevelMemory CacheLevel = "memory" // 内存缓存：最快，容量小
	CacheLevelDisk   CacheLevel = "disk"   // 磁盘缓存：较慢，容量大
	CacheLevelModel  CacheLevel = "model"  // 模型缓存：预计算结果
)

// CacheEntry 缓存条目
type CacheEntry struct {
	Key       string      `json:"key"`
	Value     interface{} `json:"value"`
	Level     CacheLevel  `json:"level"`
	CreatedAt time.Time   `json:"created_at"`
	ExpiresAt time.Time   `json:"expires_at,omitempty"`
	HitCount  int64       `json:"hit_count"`
	Size      int64       `json:"size_bytes"`
}

// CacheStats 缓存统计
type CacheStats struct {
	TotalEntries   int64   `json:"total_entries"`
	MemoryEntries  int64   `json:"memory_entries"`
	DiskEntries    int64   `json:"disk_entries"`
	ModelEntries   int64   `json:"model_entries"`
	HitRate        float64 `json:"hit_rate"`
	MissRate       float64 `json:"miss_rate"`
	Evictions      int64   `json:"evictions"`
	TotalHits      int64   `json:"total_hits"`
	TotalMisses    int64   `json:"total_misses"`
	MemoryUsage    int64   `json:"memory_usage_bytes"`
	DiskUsage      int64   `json:"disk_usage_bytes"`
}

// MultiLevelCache 多级缓存系统
type MultiLevelCache struct {
	mu            sync.RWMutex
	memoryCache   map[string]*list.Element
	memoryList    *list.List // LRU 列表
	diskCacheDir  string
	modelCache    map[string]interface{} // 预计算模型结果
	maxMemorySize int
	maxDiskSize   int64
	currentMemSize int64
	currentDiskSize int64
	stats         *CacheStats
	defaultTTL    time.Duration
}

// NewMultiLevelCache 创建多级缓存系统
func NewMultiLevelCache(maxMemorySize int, maxDiskSize int64, diskCacheDir string) (*MultiLevelCache, error) {
	// 确保磁盘缓存目录存在
	if diskCacheDir != "" {
		if err := os.MkdirAll(diskCacheDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create disk cache directory: %w", err)
		}
	}

	return &MultiLevelCache{
		memoryCache:   make(map[string]*list.Element),
		memoryList:    list.New(),
		diskCacheDir:  diskCacheDir,
		modelCache:    make(map[string]interface{}),
		maxMemorySize: maxMemorySize,
		maxDiskSize:   maxDiskSize,
		stats: &CacheStats{
			TotalEntries:  0,
			HitRate:       0,
			MissRate:      100,
		},
		defaultTTL: 1 * time.Hour,
	}, nil
}

// Get 从缓存获取数据（自动从多级缓存中查找）
func (c *MultiLevelCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. 先查内存缓存
	if elem, exists := c.memoryCache[key]; exists {
		entry := elem.Value.(*CacheEntry)
		// 检查是否过期
		if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
			c.removeEntry(elem)
			c.stats.TotalMisses++
			c.updateHitRate()
			return nil, false
		}
		// 更新访问计数和 LRU
		entry.HitCount++
		c.memoryList.MoveToFront(elem)
		c.stats.TotalHits++
		c.updateHitRate()
		return entry.Value, true
	}

	// 2. 查磁盘缓存
	if c.diskCacheDir != "" {
		if value, err := c.getFromDisk(key); err == nil {
			// 提升到内存缓存
			c.setMemory(key, value, c.defaultTTL)
			c.stats.TotalHits++
			c.updateHitRate()
			return value, true
		}
	}

	// 3. 查模型缓存
	if value, exists := c.modelCache[key]; exists {
		c.stats.TotalHits++
		c.updateHitRate()
		return value, true
	}

	c.stats.TotalMisses++
	c.updateHitRate()
	return nil, false
}

// Set 设置缓存（默认存入内存缓存）
func (c *MultiLevelCache) Set(key string, value interface{}) error {
	return c.SetWithTTL(key, value, c.defaultTTL)
}

// SetWithTTL 设置带过期时间的缓存
func (c *MultiLevelCache) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 估算大小

	// 存入内存缓存
	return c.setMemory(key, value, ttl)
}

// setMemory 存入内存缓存
func (c *MultiLevelCache) setMemory(key string, value interface{}, ttl time.Duration) error {
	size := estimateSize(value)

	// 如果已存在，先移除旧条目
	if elem, exists := c.memoryCache[key]; exists {
		c.removeEntry(elem)
	}

	// 检查是否需要驱逐
	for c.currentMemSize+size > int64(c.maxMemorySize) && c.memoryList.Len() > 0 {
		c.evictOldest()
	}

	entry := &CacheEntry{
		Key:       key,
		Value:     value,
		Level:     CacheLevelMemory,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(ttl),
		HitCount:  0,
		Size:      size,
	}

	elem := c.memoryList.PushFront(entry)
	c.memoryCache[key] = elem
	c.currentMemSize += size
	c.stats.TotalEntries++
	c.stats.MemoryEntries++

	return nil
}

// SetToDisk 存入磁盘缓存
func (c *MultiLevelCache) SetToDisk(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.diskCacheDir == "" {
		return fmt.Errorf("disk cache not configured")
	}

	size := estimateSize(value)

	// 检查磁盘空间
	for c.currentDiskSize+size > c.maxDiskSize && c.diskCacheDir != "" {
		// 简单实现：不清理，实际应该清理最旧的
		break
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	filePath := filepath.Join(c.diskCacheDir, c.keyToFilename(key))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write to disk: %w", err)
	}

	c.currentDiskSize += size
	c.stats.DiskEntries++

	return nil
}

// SetToModel 存入模型缓存（用于预计算结果）
func (c *MultiLevelCache) SetToModel(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.modelCache[key] = value
	c.stats.ModelEntries++
	c.stats.TotalEntries++

	return nil
}

// Delete 删除缓存条目
func (c *MultiLevelCache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, exists := c.memoryCache[key]; exists {
		entry := elem.Value.(*CacheEntry)
		c.currentMemSize -= entry.Size
		c.removeEntry(elem)
	}

	if c.diskCacheDir != "" {
		filePath := filepath.Join(c.diskCacheDir, c.keyToFilename(key))
		os.Remove(filePath)
	}

	delete(c.modelCache, key)

	return nil
}

// Clear 清空所有缓存
func (c *MultiLevelCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.memoryCache = make(map[string]*list.Element)
	c.memoryList = list.New()
	c.modelCache = make(map[string]interface{})
	c.currentMemSize = 0

	if c.diskCacheDir != "" {
		os.RemoveAll(c.diskCacheDir)
		os.MkdirAll(c.diskCacheDir, 0755)
	}
	c.currentDiskSize = 0

	c.stats = &CacheStats{
		TotalEntries:  0,
		HitRate:       0,
		MissRate:      100,
	}

	return nil
}

// GetStats 获取缓存统计信息
func (c *MultiLevelCache) GetStats() *CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.stats.MemoryUsage = c.currentMemSize
	c.stats.DiskUsage = c.currentDiskSize
	c.stats.TotalEntries = int64(len(c.memoryCache)) + int64(len(c.modelCache))

	return c.stats
}

// evictOldest 驱逐最旧的条目
func (c *MultiLevelCache) evictOldest() {
	elem := c.memoryList.Back()
	if elem != nil {
		c.removeEntry(elem)
		c.stats.Evictions++
	}
}

// removeEntry 移除条目
func (c *MultiLevelCache) removeEntry(elem *list.Element) {
	entry := elem.Value.(*CacheEntry)
	delete(c.memoryCache, entry.Key)
	c.memoryList.Remove(elem)
	c.currentMemSize -= entry.Size
	c.stats.TotalEntries--
	c.stats.MemoryEntries--
}

// getFromDisk 从磁盘获取
func (c *MultiLevelCache) getFromDisk(key string) (interface{}, error) {
	filePath := filepath.Join(c.diskCacheDir, c.keyToFilename(key))
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var value interface{}
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}

	return value, nil
}

// keyToFilename 将 key 转换为安全的文件名
func (c *MultiLevelCache) keyToFilename(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// updateHitRate 更新命中率
func (c *MultiLevelCache) updateHitRate() {
	total := c.stats.TotalHits + c.stats.TotalMisses
	if total > 0 {
		c.stats.HitRate = float64(c.stats.TotalHits) / float64(total) * 100
		c.stats.MissRate = float64(c.stats.TotalMisses) / float64(total) * 100
	}
}

// estimateSize 估算数据大小
func estimateSize(value interface{}) int64 {
	data, err := json.Marshal(value)
	if err != nil {
		return 100 // 默认值
	}
	return int64(len(data))
}

// WarmUp 预热缓存（批量加载数据）
func (c *MultiLevelCache) WarmUp(entries map[string]interface{}, ttl time.Duration) error {
	for key, value := range entries {
		if err := c.SetWithTTL(key, value, ttl); err != nil {
			return fmt.Errorf("failed to warm up cache for key %s: %w", key, err)
		}
	}
	return nil
}

// ExportStats 导出统计信息为 JSON
func (c *MultiLevelCache) ExportStats() (string, error) {
	stats := c.GetStats()
	data, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
