package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"
)

// ResourceType 资源类型
type ResourceType string

const (
	ResourceCPU    ResourceType = "cpu"
	ResourceMemory ResourceType = "memory"
	ResourceDisk   ResourceType = "disk"
	ResourceNet    ResourceType = "network"
)

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	Type      ResourceType `json:"type"`
	Value     float64      `json:"value"`
	Unit      string       `json:"unit"`
	Timestamp time.Time    `json:"timestamp"`
}

// SystemMetrics 系统指标
type SystemMetrics struct {
	CPUUsagePercent    float64   `json:"cpu_usage_percent"`
	MemoryUsedBytes    uint64    `json:"memory_used_bytes"`
	MemoryTotalBytes   uint64    `json:"memory_total_bytes"`
	MemoryUsagePercent float64   `json:"memory_usage_percent"`
	GoroutineCount     int       `json:"goroutine_count"`
	DiskUsedBytes      uint64    `json:"disk_used_bytes,omitempty"`
	DiskTotalBytes     uint64    `json:"disk_total_bytes,omitempty"`
	Timestamp          time.Time `json:"timestamp"`
}

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheckResult 健康检查结果
type HealthCheckResult struct {
	Name      string        `json:"name"`
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	LatencyMs int64         `json:"latency_ms,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// Alert 告警
type Alert struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Level     AlertLevel        `json:"level"`
	Message   string            `json:"message"`
	Metric    string            `json:"metric"`
	Threshold float64           `json:"threshold"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// AlertHandler 告警处理函数
type AlertHandler func(alert *Alert) error

// Monitor 监控器
type Monitor struct {
	mu              sync.RWMutex
	metrics         *SystemMetrics
	alertHandlers   []AlertHandler
	alertThresholds map[string]float64
	checkInterval   time.Duration
	lastCheck       time.Time
	healthChecks    map[string]func() error
	history         []*SystemMetrics
	maxHistorySize  int
}

// NewMonitor 创建监控器
func NewMonitor(checkInterval time.Duration, maxHistorySize int) *Monitor {
	return &Monitor{
		metrics: &SystemMetrics{
			Timestamp: time.Now(),
		},
		alertHandlers: make([]AlertHandler, 0),
		alertThresholds: map[string]float64{
			"cpu":    80.0,
			"memory": 85.0,
		},
		checkInterval:  checkInterval,
		healthChecks:   make(map[string]func() error),
		history:        make([]*SystemMetrics, 0),
		maxHistorySize: maxHistorySize,
	}
}

// Start 启动监控
func (m *Monitor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(m.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.collectMetrics()
				m.checkAlerts()
			}
		}
	}()
}

// collectMetrics 收集系统指标
func (m *Monitor) collectMetrics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// CPU 使用率（简化实现）
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	m.metrics.Timestamp = time.Now()
	m.metrics.MemoryUsedBytes = memStats.Alloc
	m.metrics.MemoryTotalBytes = memStats.Sys
	m.metrics.MemoryUsagePercent = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	m.metrics.GoroutineCount = runtime.NumGoroutine()

	// 保存到历史
	m.history = append(m.history, m.metrics)
	if len(m.history) > m.maxHistorySize {
		m.history = m.history[1:]
	}

	m.lastCheck = time.Now()
}

// GetMetrics 获取当前指标
func (m *Monitor) GetMetrics() *SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics
}

// GetHistory 获取历史指标
func (m *Monitor) GetHistory(limit int) []*SystemMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if limit > len(m.history) {
		limit = len(m.history)
	}

	result := make([]*SystemMetrics, limit)
	copy(result, m.history[len(m.history)-limit:])
	return result
}

// RegisterAlertHandler 注册告警处理函数
func (m *Monitor) RegisterAlertHandler(handler AlertHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertHandlers = append(m.alertHandlers, handler)
}

// SetAlertThreshold 设置告警阈值
func (m *Monitor) SetAlertThreshold(metric string, threshold float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.alertThresholds[metric] = threshold
}

// checkAlerts 检查告警
func (m *Monitor) checkAlerts() {
	m.mu.RLock()
	metrics := m.metrics
	thresholds := make(map[string]float64)
	for k, v := range m.alertThresholds {
		thresholds[k] = v
	}
	handlers := make([]AlertHandler, len(m.alertHandlers))
	copy(handlers, m.alertHandlers)
	m.mu.RUnlock()

	// 检查 CPU
	if thresholds["cpu"] > 0 && metrics.CPUUsagePercent > thresholds["cpu"] {
		alert := &Alert{
			ID:        generateAlertID("cpu"),
			Name:      "High CPU Usage",
			Level:     AlertLevelWarning,
			Message:   fmt.Sprintf("CPU usage is %.2f%%", metrics.CPUUsagePercent),
			Metric:    "cpu",
			Threshold: thresholds["cpu"],
			Value:     metrics.CPUUsagePercent,
			Timestamp: time.Now(),
		}
		for _, handler := range handlers {
			handler(alert)
		}
	}

	// 检查内存
	if thresholds["memory"] > 0 && metrics.MemoryUsagePercent > thresholds["memory"] {
		alert := &Alert{
			ID:        generateAlertID("memory"),
			Name:      "High Memory Usage",
			Level:     AlertLevelCritical,
			Message:   fmt.Sprintf("Memory usage is %.2f%%", metrics.MemoryUsagePercent),
			Metric:    "memory",
			Threshold: thresholds["memory"],
			Value:     metrics.MemoryUsagePercent,
			Timestamp: time.Now(),
		}
		for _, handler := range handlers {
			handler(alert)
		}
	}
}

// RegisterHealthCheck 注册健康检查
func (m *Monitor) RegisterHealthCheck(name string, check func() error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.healthChecks[name] = check
}

// CheckHealth 执行健康检查
func (m *Monitor) CheckHealth() []HealthCheckResult {
	m.mu.RLock()
	checks := make(map[string]func() error)
	for k, v := range m.healthChecks {
		checks[k] = v
	}
	m.mu.RUnlock()

	results := make([]HealthCheckResult, 0, len(checks))

	for name, check := range checks {
		start := time.Now()
		err := check()
		latency := time.Since(start).Milliseconds()

		result := HealthCheckResult{
			Name:      name,
			Timestamp: time.Now(),
			LatencyMs: latency,
		}

		if err == nil {
			result.Status = HealthStatusHealthy
			result.Message = "OK"
		} else {
			result.Status = HealthStatusUnhealthy
			result.Message = err.Error()
		}

		results = append(results, result)
	}

	return results
}

// GetOverallHealth 获取整体健康状态
func (m *Monitor) GetOverallHealth() HealthStatus {
	results := m.CheckHealth()
	
	hasUnhealthy := false
	hasDegraded := false

	for _, r := range results {
		if r.Status == HealthStatusUnhealthy {
			hasUnhealthy = true
		} else if r.Status == HealthStatusDegraded {
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return HealthStatusUnhealthy
	}
	if hasDegraded {
		return HealthStatusDegraded
	}
	return HealthStatusHealthy
}

// ExportMetrics 导出指标为 Prometheus 格式
func (m *Monitor) ExportMetrics() string {
	metrics := m.GetMetrics()
	
	output := fmt.Sprintf(`# HELP gclaw_memory_usage_bytes Memory usage in bytes
# TYPE gclaw_memory_usage_bytes gauge
gclaw_memory_usage_bytes %d

# HELP gclaw_memory_total_bytes Total memory in bytes
# TYPE gclaw_memory_total_bytes gauge
gclaw_memory_total_bytes %d

# HELP gclaw_memory_usage_percent Memory usage percentage
# TYPE gclaw_memory_usage_percent gauge
gclaw_memory_usage_percent %.2f

# HELP gclaw_goroutines Number of goroutines
# TYPE gclaw_goroutines gauge
gclaw_goroutines %d

# HELP gclaw_last_check_timestamp Last check timestamp
# TYPE gclaw_last_check_timestamp gauge
gclaw_last_check_timestamp %d
`,
		metrics.MemoryUsedBytes,
		metrics.MemoryTotalBytes,
		metrics.MemoryUsagePercent,
		metrics.GoroutineCount,
		metrics.Timestamp.Unix(),
	)

	return output
}

// ServeHTTP 提供 HTTP 接口用于监控
func (m *Monitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/metrics":
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(m.ExportMetrics()))
	case "/health":
		w.Header().Set("Content-Type", "application/json")
		health := map[string]interface{}{
			"status":      m.GetOverallHealth(),
			"checks":      m.CheckHealth(),
			"timestamp":   time.Now(),
		}
		json.NewEncoder(w).Encode(health)
	case "/stats":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(m.GetMetrics())
	default:
		http.NotFound(w, r)
	}
}

// 辅助函数

func generateAlertID(metric string) string {
	return fmt.Sprintf("%s-%d", metric, time.Now().UnixNano())
}

// DefaultAlertHandler 默认告警处理（输出到 stderr）
func DefaultAlertHandler(alert *Alert) error {
	msg := fmt.Sprintf("[%s] ALERT: %s - %s (value: %.2f, threshold: %.2f)",
		alert.Level, alert.Name, alert.Message, alert.Value, alert.Threshold)
	fmt.Fprintln(os.Stderr, msg)
	return nil
}

// LogAlertHandler 日志告警处理
func LogAlertHandler(alert *Alert) error {
	logEntry := map[string]interface{}{
		"level":     alert.Level,
		"name":      alert.Name,
		"message":   alert.Message,
		"metric":    alert.Metric,
		"value":     alert.Value,
		"threshold": alert.Threshold,
		"timestamp": alert.Timestamp,
	}
	data, _ := json.Marshal(logEntry)
	fmt.Println(string(data))
	return nil
}
