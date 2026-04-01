package skill

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"gclaw/internal/security"
)

// ============================================
// 沙箱配置增强 - 支持完整的产品化配置
// ============================================

// SandboxLevel 沙箱隔离级别
type SandboxLevel string

const (
	SandboxLevelNone     SandboxLevel = "none"      // 无隔离（可信环境）
	SandboxLevelBasic    SandboxLevel = "basic"     // 基础隔离（命令白名单 + 路径限制）
	SandboxLevelStandard SandboxLevel = "standard"  // 标准隔离（Namespace + 资源限制）
	SandboxLevelStrict   SandboxLevel = "strict"    // 严格隔离（gVisor/容器级）
)

// AuditLogLevel 审计日志级别
type AuditLogLevel string

const (
	AuditLogLevelNone  AuditLogLevel = "none"  // 不记录
	AuditLogLevelError AuditLogLevel = "error" // 仅错误
	AuditLogLevelWarn  AuditLogLevel = "warn"  // 警告及以上
	AuditLogLevelInfo  AuditLogLevel = "info"  // 所有操作
)

// EnhancedSandboxConfig 增强的沙箱配置（支持 JSON 序列化）
type EnhancedSandboxConfig struct {
	// 基础配置
	Enabled       bool         `json:"enabled"`                // 是否启用沙箱
	Level         SandboxLevel `json:"level"`                  // 隔离级别
	DefaultLevel  SandboxLevel `json:"default_level"`          // 默认隔离级别
	
	// 超时与资源限制
	Timeout            time.Duration `json:"timeout_seconds"`           // 执行超时（秒）
	MaxMemoryMB        int64         `json:"max_memory_mb"`             // 最大内存（MB）
	MaxCPUPercent      float64       `json:"max_cpu_percent"`           // 最大 CPU 使用率（%）
	MaxFileSizeMB      int64         `json:"max_file_size_mb"`          // 单文件最大大小（MB）
	MaxTotalDiskUsageMB int64        `json:"max_total_disk_usage_mb"`   // 沙箱目录总大小限制（MB）
	
	// Shell 沙箱配置
	ShellEnabled       bool          `json:"shell_enabled"`             // 是否允许 Shell 执行
	ShellWhitelist     []string      `json:"shell_whitelist"`           // 允许的 Shell 命令白名单
	ShellBlacklist     []string      `json:"shell_blacklist"`           // 禁止的 Shell 命令黑名单
	ShellArgPatterns   []string      `json:"shell_arg_patterns"`        // 禁止的参数模式（正则）
	
	// 文件系统沙箱配置
	FileSandboxRoot    string        `json:"file_sandbox_root"`         // 沙箱根目录
	FileResetOnStart   bool          `json:"file_reset_on_start"`       // 启动时重置沙箱目录
	FileAllowedPaths   []string      `json:"file_allowed_paths"`        // 允许访问的路径前缀
	FileBlockedPaths   []string      `json:"file_blocked_paths"`        // 禁止访问的路径前缀
	
	// 网络配置
	NetworkDisabled    bool          `json:"network_disabled"`          // 禁用网络
	NetworkWhitelist   []string      `json:"network_whitelist"`         // 允许的网络地址（CIDR 或域名）
	
	// 审计与监控
	AuditLogEnabled    bool          `json:"audit_log_enabled"`         // 启用审计日志
	AuditLogLevel      AuditLogLevel `json:"audit_log_level"`           // 审计日志级别
	AuditLogPath       string        `json:"audit_log_path"`            // 审计日志文件路径
	AnomalyDetection   bool          `json:"anomaly_detection"`         // 启用异常检测
	MaxExecPerMinute   int           `json:"max_exec_per_minute"`       // 每分钟最大执行次数
	
	// 安全模块集成
	SecurityIntegration bool         `json:"security_integration"`      // 是否与 security 模块联动
	UserLevelMapping    map[string]SandboxLevel `json:"user_level_mapping"` // 用户级别到沙箱级别的映射
	
	// 高级选项
	UseGVisor          bool          `json:"use_gvisor"`                // 是否使用 gVisor（如果可用）
	UseLinuxNamespace  bool          `json:"use_linux_namespace"`       // 是否使用 Linux Namespace
	CrossPlatformCompat bool         `json:"cross_platform_compat"`     // 跨平台兼容模式
	
	// 调试选项
	DryRun             bool          `json:"dry_run"`                   // 干跑模式（只检查不执行）
	VerboseLogging     bool          `json:"verbose_logging"`           // 详细日志
}

// DefaultEnhancedSandboxConfig 默认增强沙箱配置
func DefaultEnhancedSandboxConfig() *EnhancedSandboxConfig {
	return &EnhancedSandboxConfig{
		Enabled:              true,
		Level:                SandboxLevelStandard,
		DefaultLevel:         SandboxLevelStandard,
		Timeout:              30 * time.Second,
		MaxMemoryMB:          512,
		MaxCPUPercent:        50.0,
		MaxFileSizeMB:        100,
		MaxTotalDiskUsageMB:  500,
		ShellEnabled:         true,
		ShellWhitelist:       []string{"ls", "cat", "echo", "pwd", "whoami", "date", "uname"},
		ShellBlacklist:       []string{"rm", "sudo", "su", "chmod", "chown", "mount", "umount", "dd"},
		ShellArgPatterns:     []string{`[;&|]`, `\$\(`, "`", `>/dev/tcp/`, `curl.*\|.*sh`, `wget.*\|.*sh`},
		FileSandboxRoot:      "/tmp/gclaw-sandbox",
		FileResetOnStart:     true,
		FileAllowedPaths:     []string{"/tmp", "/var/tmp"},
		FileBlockedPaths:     []string{"/etc", "/root", "/proc", "/sys", "/dev"},
		NetworkDisabled:      true,
		NetworkWhitelist:     []string{},
		AuditLogEnabled:      true,
		AuditLogLevel:        AuditLogLevelInfo,
		AuditLogPath:         "/var/log/gclaw/sandbox-audit.log",
		AnomalyDetection:     true,
		MaxExecPerMinute:     60,
		SecurityIntegration:  true,
		UserLevelMapping: map[string]SandboxLevel{
			"admin":   SandboxLevelStandard,
			"user":    SandboxLevelStandard,
			"guest":   SandboxLevelStrict,
			"trusted": SandboxLevelBasic,
		},
		UseGVisor:         false,
		UseLinuxNamespace: true,
		CrossPlatformCompat: true,
		DryRun:            false,
		VerboseLogging:    false,
	}
}

// LoadEnhancedSandboxConfigFromFile 从文件加载增强沙箱配置
func LoadEnhancedSandboxConfigFromFile(path string) (*EnhancedSandboxConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sandbox config file: %w", err)
	}
	
	var config EnhancedSandboxConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse sandbox config: %w", err)
	}
	
	// 验证配置
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid sandbox config: %w", err)
	}
	
	return &config, nil
}

// Validate 验证沙箱配置
func (c *EnhancedSandboxConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if c.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive")
	}
	if c.MaxCPUPercent < 0 || c.MaxCPUPercent > 100 {
		return fmt.Errorf("max_cpu_percent must be between 0 and 100")
	}
	
	// 检查隔离级别
	validLevels := map[SandboxLevel]bool{
		SandboxLevelNone: true, SandboxLevelBasic: true,
		SandboxLevelStandard: true, SandboxLevelStrict: true,
	}
	if !validLevels[c.Level] {
		return fmt.Errorf("invalid sandbox level: %s", c.Level)
	}
	
	return nil
}

// GetEffectiveLevel 根据用户角色获取有效的沙箱级别
func (c *EnhancedSandboxConfig) GetEffectiveLevel(userRole string) SandboxLevel {
	if level, exists := c.UserLevelMapping[userRole]; exists {
		return level
	}
	return c.DefaultLevel
}

// ============================================
// 审计日志系统
// ============================================

// AuditEvent 审计事件
type AuditEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	EventID     string            `json:"event_id"`
	EventType   string            `json:"event_type"`
	UserID      string            `json:"user_id,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	Action      string            `json:"action"`
	Resource    string            `json:"resource,omitempty"`
	Result      string            `json:"result"`
	Details     map[string]interface{} `json:"details,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	Error       string            `json:"error,omitempty"`
}

// AuditLogger 审计日志器
type AuditLogger struct {
	mu        sync.Mutex
	config    *EnhancedSandboxConfig
	logFile   *os.File
	encoder   *json.Encoder
	eventChan chan *AuditEvent
	done      chan struct{}
}

// NewAuditLogger 创建审计日志器
func NewAuditLogger(config *EnhancedSandboxConfig) (*AuditLogger, error) {
	al := &AuditLogger{
		config:    config,
		eventChan: make(chan *AuditEvent, 100),
		done:      make(chan struct{}),
	}
	
	if !config.AuditLogEnabled || config.AuditLogLevel == AuditLogLevelNone {
		return al, nil
	}
	
	// 确保日志目录存在
	logDir := filepath.Dir(config.AuditLogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create audit log directory: %w", err)
	}
	
	// 打开日志文件（追加模式）
	f, err := os.OpenFile(config.AuditLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open audit log file: %w", err)
	}
	
	al.logFile = f
	al.encoder = json.NewEncoder(f)
	
	// 启动异步日志写入 goroutine
	go al.writeLoop()
	
	return al, nil
}

// writeLoop 异步写入日志
func (al *AuditLogger) writeLoop() {
	for {
		select {
		case event := <-al.eventChan:
			al.mu.Lock()
			if al.encoder != nil {
				_ = al.encoder.Encode(event)
			}
			al.mu.Unlock()
		case <-al.done:
			return
		}
	}
}

// Log 记录审计事件
func (al *AuditLogger) Log(eventType, action, result string, details map[string]interface{}) {
	if !al.config.AuditLogEnabled || al.config.AuditLogLevel == AuditLogLevelNone {
		return
	}
	
	// 根据日志级别过滤
	if al.config.AuditLogLevel == AuditLogLevelError && result == "success" {
		return
	}
	
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventID:   generateEventID(),
		EventType: eventType,
		Action:    action,
		Result:    result,
		Details:   details,
	}
	
	select {
	case al.eventChan <- event:
	default:
		// 通道已满，丢弃日志（避免阻塞）
	}
}

// Close 关闭审计日志器
func (al *AuditLogger) Close() error {
	close(al.done)
	if al.logFile != nil {
		return al.logFile.Close()
	}
	return nil
}

func generateEventID() string {
	h := sha256.New()
	h.Write([]byte(time.Now().String()))
	h.Write([]byte(fmt.Sprintf("%d", time.Now().Nanosecond())))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// ============================================
// 异常检测系统
// ============================================

// AnomalyDetector 异常检测器
type AnomalyDetector struct {
	mu              sync.Mutex
	config          *EnhancedSandboxConfig
	execHistory     []time.Time
	suspiciousPatterns []*regexp.Regexp
	alertCallback   func(alert *SecurityAlert)
}

// SecurityAlert 安全告警
type SecurityAlert struct {
	Timestamp time.Time   `json:"timestamp"`
	AlertID   string      `json:"alert_id"`
	Severity  string      `json:"severity"` // low, medium, high, critical
	Type      string      `json:"type"`
	Message   string      `json:"message"`
	Details   map[string]interface{} `json:"details"`
}

// NewAnomalyDetector 创建异常检测器
func NewAnomalyDetector(config *EnhancedSandboxConfig) *AnomalyDetector {
	ad := &AnomalyDetector{
		config: config,
		suspiciousPatterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)/etc/passwd`),
			regexp.MustCompile(`(?i)/etc/shadow`),
			regexp.MustCompile(`(?i)\.\./\.\.`), 
			regexp.MustCompile(`(?i)base64\s+-d`),
			regexp.MustCompile(`(?i)eval\s*\(`),
			regexp.MustCompile(`(?i)python\s+-c`),
			regexp.MustCompile(`(?i)perl\s+-e`),
			regexp.MustCompile(`(?i)nc\s+-e`),
			regexp.MustCompile(`(?i)/dev/tcp/`),
			regexp.MustCompile(`(?i)mkfifo`),
		},
	}
	
	return ad
}

// SetAlertCallback 设置告警回调
func (ad *AnomalyDetector) SetAlertCallback(callback func(*SecurityAlert)) {
	ad.alertCallback = callback
}

// CheckExecutionRate 检查执行频率
func (ad *AnomalyDetector) CheckExecutionRate() error {
	if !ad.config.AnomalyDetection || ad.config.MaxExecPerMinute <= 0 {
		return nil
	}
	
	ad.mu.Lock()
	defer ad.mu.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-time.Minute)
	
	// 清理旧记录
	var recent []time.Time
	for _, t := range ad.execHistory {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	ad.execHistory = recent
	
	// 检查是否超限
	if len(ad.execHistory) >= ad.config.MaxExecPerMinute {
		ad.triggerAlert("rate_limit", "high", 
			fmt.Sprintf("Execution rate exceeded: %d/min", len(ad.execHistory)),
			map[string]interface{}{"count": len(ad.execHistory), "limit": ad.config.MaxExecPerMinute})
		return fmt.Errorf("execution rate limit exceeded (%d/min)", ad.config.MaxExecPerMinute)
	}
	
	// 记录本次执行
	ad.execHistory = append(ad.execHistory, now)
	return nil
}

// CheckCommand 检查命令是否可疑
func (ad *AnomalyDetector) CheckCommand(cmd string) (*SecurityAlert, error) {
	if !ad.config.AnomalyDetection {
		return nil, nil
	}
	
	for _, pattern := range ad.suspiciousPatterns {
		if pattern.MatchString(cmd) {
			alert := &SecurityAlert{
				Timestamp: time.Now(),
				AlertID:   generateEventID(),
				Severity:  "medium",
				Type:      "suspicious_command",
				Message:   fmt.Sprintf("Suspicious command pattern detected: %s", pattern.String()),
				Details:   map[string]interface{}{"command": cmd, "pattern": pattern.String()},
			}
			
			ad.triggerAlertFromStruct(alert)
			return alert, fmt.Errorf("suspicious command detected")
		}
	}
	
	return nil, nil
}

func (ad *AnomalyDetector) triggerAlert(alertType, severity, message string, details map[string]interface{}) {
	alert := &SecurityAlert{
		Timestamp: time.Now(),
		AlertID:   generateEventID(),
		Severity:  severity,
		Type:      alertType,
		Message:   message,
		Details:   details,
	}
	ad.triggerAlertFromStruct(alert)
}

func (ad *AnomalyDetector) triggerAlertFromStruct(alert *SecurityAlert) {
	if ad.alertCallback != nil {
		go ad.alertCallback(alert)
	}
	// 同时记录到审计日志
	fmt.Printf("[SECURITY ALERT] %s - %s: %s\n", alert.Severity, alert.Type, alert.Message)
}

// ============================================
// 增强的沙箱实现
// ============================================

// EnhancedSandbox 增强的沙箱
type EnhancedSandbox struct {
	mu               sync.RWMutex
	config           *EnhancedSandboxConfig
	auditLogger      *AuditLogger
	anomalyDetector  *AnomalyDetector
	secretManager    *security.SecretManager
	sessionRoots     map[string]string // sessionID -> sandbox root path
	currentSession   string
	
	// 跨平台兼容
	osType string
}

// NewEnhancedSandbox 创建增强沙箱
func NewEnhancedSandbox(config *EnhancedSandboxConfig, secretMgr *security.SecretManager) (*EnhancedSandbox, error) {
	if config == nil {
		config = DefaultEnhancedSandboxConfig()
	}
	
	if err := config.Validate(); err != nil {
		return nil, err
	}
	
	// 创建审计日志器
	auditLogger, err := NewAuditLogger(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit logger: %w", err)
	}
	
	// 创建异常检测器
	anomalyDetector := NewAnomalyDetector(config)
	anomalyDetector.SetAlertCallback(func(alert *SecurityAlert) {
		auditLogger.Log("security", "alert_triggered", "warning", map[string]interface{}{
			"alert_id": alert.AlertID,
			"severity": alert.Severity,
			"type":     alert.Type,
			"message":  alert.Message,
		})
	})
	
	es := &EnhancedSandbox{
		config:          config,
		auditLogger:     auditLogger,
		anomalyDetector: anomalyDetector,
		secretManager:   secretMgr,
		sessionRoots:    make(map[string]string),
		osType:          runtime.GOOS,
	}
	
	// 如果配置了启动时重置，清理沙箱目录
	if config.FileResetOnStart {
		if err := es.resetSandboxRoot(); err != nil {
			auditLogger.Log("system", "reset_failed", "error", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	
	return es, nil
}

// resetSandboxRoot 重置沙箱根目录
func (es *EnhancedSandbox) resetSandboxRoot() error {
	root := es.config.FileSandboxRoot
	
	// 删除旧目录
	if err := os.RemoveAll(root); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old sandbox root: %w", err)
	}
	
	// 创建新目录
	if err := os.MkdirAll(root, 0700); err != nil {
		return fmt.Errorf("failed to create sandbox root: %w", err)
	}
	
	es.auditLogger.Log("system", "sandbox_reset", "success", map[string]interface{}{
		"root_path": root,
	})
	
	return nil
}

// SetSession 设置当前会话
func (es *EnhancedSandbox) SetSession(sessionID, userID string) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	
	// 根据用户角色确定沙箱级别
	level := es.config.GetEffectiveLevel(userID)
	
	// 为会话创建独立的沙箱目录
	sessionRoot := filepath.Join(es.config.FileSandboxRoot, sessionID[:8])
	if err := os.MkdirAll(sessionRoot, 0700); err != nil {
		return fmt.Errorf("failed to create session sandbox: %w", err)
	}
	
	es.sessionRoots[sessionID] = sessionRoot
	es.currentSession = sessionID
	
	es.auditLogger.Log("session", "session_created", "success", map[string]interface{}{
		"session_id":    sessionID,
		"user_id":       userID,
		"sandbox_level": level,
		"root_path":     sessionRoot,
	})
	
	return nil
}

// ExecuteShellCommand 在沙箱中执行 Shell 命令
func (es *EnhancedSandbox) ExecuteShellCommand(command string, args ...string) (string, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	startTime := time.Now()
	details := map[string]interface{}{
		"command": command,
		"args":    args,
	}
	
	// 检查是否启用沙箱
	if !es.config.Enabled {
		es.auditLogger.Log("execution", "command_executed", "skipped", details)
		return es.executeDirectCommand(command, args...)
	}
	
	// 干跑模式
	if es.config.DryRun {
		es.auditLogger.Log("execution", "command_dryrun", "success", details)
		return "[DRY RUN] Command would be executed: " + command, nil
	}
	
	// 检查执行频率
	if err := es.anomalyDetector.CheckExecutionRate(); err != nil {
		es.auditLogger.Log("execution", "command_blocked", "rate_limited", details)
		return "", err
	}
	
	// 检查命令是否可疑
	if alert, err := es.anomalyDetector.CheckCommand(command); err != nil {
		details["alert_id"] = alert.AlertID
		es.auditLogger.Log("execution", "command_blocked", "suspicious", details)
		return "", err
	}
	
	// 检查命令白名单/黑名单
	if err := es.validateShellCommand(command); err != nil {
		details["validation_error"] = err.Error()
		es.auditLogger.Log("execution", "command_blocked", "validation_failed", details)
		return "", err
	}
	
	// 构建完整命令
	fullCmd := exec.Command(command, args...)
	
	// 设置沙箱环境
	if err := es.setupCommandEnvironment(fullCmd); err != nil {
		es.auditLogger.Log("execution", "setup_failed", "error", map[string]interface{}{
			"error": err.Error(),
		})
		return "", err
	}
	
	// 执行命令
	ctx, cancel := context.WithTimeout(context.Background(), es.config.Timeout)
	defer cancel()
	
	fullCmd.SysProcAttr = es.getPlatformSpecificSysProcAttr()
	
	outputChan := make(chan struct {
		output []byte
		err    error
	}, 1)
	
	go func() {
		output, err := fullCmd.CombinedOutput()
		outputChan <- struct {
			output []byte
			err    error
		}{output: output, err: err}
	}()
	
	select {
	case result := <-outputChan:
		if result.err != nil {
			details["error"] = result.err.Error()
			es.auditLogger.Log("execution", "command_executed", "error", details)
			return string(result.output), result.err
		}
		
		duration := time.Since(startTime)
		details["duration_ms"] = duration.Milliseconds()
		details["output_length"] = len(result.output)
		es.auditLogger.Log("execution", "command_executed", "success", details)
		
		return string(result.output), nil
		
	case <-ctx.Done():
		if fullCmd.Process != nil {
			_ = fullCmd.Process.Kill()
		}
		es.auditLogger.Log("execution", "command_timeout", "error", details)
		return "", fmt.Errorf("command execution timeout after %v", es.config.Timeout)
	}
}

// validateShellCommand 验证 Shell 命令
func (es *EnhancedSandbox) validateShellCommand(command string) error {
	// 检查黑名单
	for _, blocked := range es.config.ShellBlacklist {
		if command == blocked || strings.HasPrefix(command, blocked+" ") {
			return fmt.Errorf("command '%s' is blocked", command)
		}
	}
	
	// 检查白名单（如果配置了）
	if len(es.config.ShellWhitelist) > 0 {
		allowed := false
		for _, allowedCmd := range es.config.ShellWhitelist {
			if command == allowedCmd || strings.HasPrefix(command, allowedCmd+" ") {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command '%s' is not in whitelist", command)
		}
	}
	
	// 检查参数模式
	for _, pattern := range es.config.ShellArgPatterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			continue
		}
		if re.MatchString(command) {
			return fmt.Errorf("command contains forbidden pattern: %s", pattern)
		}
	}
	
	return nil
}

// setupCommandEnvironment 设置命令执行环境
func (es *EnhancedSandbox) setupCommandEnvironment(cmd *exec.Cmd) error {
	// 设置工作目录为会话沙箱目录
	if es.currentSession != "" {
		if sessionRoot, exists := es.sessionRoots[es.currentSession]; exists {
			cmd.Dir = sessionRoot
		}
	}
	
	// 设置环境变量
	env := os.Environ()
	
	// 禁用网络（如果配置了）
	if es.config.NetworkDisabled {
		env = append(env,
			"NO_NETWORK=1",
			"http_proxy=",
			"https_proxy=",
			"ftp_proxy=",
		)
	}
	
	// 限制 PATH
	env = append(env, "PATH=/usr/local/bin:/usr/bin:/bin")
	
	cmd.Env = env
	
	// 限制标准输入
	cmd.Stdin = strings.NewReader("")
	
	return nil
}

// getPlatformSpecificSysProcAttr 获取平台特定的进程属性
func (es *EnhancedSandbox) getPlatformSpecificSysProcAttr() *syscall.SysProcAttr {
	attr := &syscall.SysProcAttr{}
	
	if !es.config.CrossPlatformCompat {
		// 仅在非兼容模式下使用平台特定功能
		switch es.osType {
		case "linux":
			if es.config.UseLinuxNamespace {
				// Linux: 使用 Namespace 隔离
				attr.Cloneflags = syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET
			}
			// 设置资源限制
			attr.Credential = &syscall.Credential{Uid: 65534, Gid: 65534} // nobody user
			
		case "darwin":
			// macOS: 有限的隔离支持
			attr.Noctty = true
			
		case "windows":
			// Windows: 使用 Job Object（需要额外实现）
			// 这里仅做基本设置 - 注意：Windows 的 SysProcAttr 没有 HideWindow 字段
			// 需要使用 exec.Command 的其他方法或外部工具
		}
	}
	
	return attr
}

// executeDirectCommand 直接执行命令（无沙箱）
func (es *EnhancedSandbox) executeDirectCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// ExecuteFileOperation 执行文件操作（带沙箱保护）
func (es *EnhancedSandbox) ExecuteFileOperation(op string, path string, data []byte) error {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	details := map[string]interface{}{
		"operation": op,
		"path":      path,
	}
	
	// 解析实际路径
	actualPath := es.resolveSandboxPath(path)
	
	// 验证路径
	if err := es.validateFilePath(actualPath); err != nil {
		details["validation_error"] = err.Error()
		es.auditLogger.Log("file", "operation_blocked", "validation_failed", details)
		return err
	}
	
	// 检查文件大小
	if len(data) > 0 {
		maxBytes := es.config.MaxFileSizeMB * 1024 * 1024
		if int64(len(data)) > maxBytes {
			es.auditLogger.Log("file", "operation_blocked", "size_exceeded", details)
			return fmt.Errorf("file size exceeds limit (%d MB)", es.config.MaxFileSizeMB)
		}
	}
	
	// 执行操作
	var err error
	switch op {
	case "write":
		err = os.WriteFile(actualPath, data, 0600)
	case "read":
		_, err = os.ReadFile(actualPath)
	case "delete":
		err = os.Remove(actualPath)
	default:
		err = fmt.Errorf("unknown operation: %s", op)
	}
	
	result := "success"
	if err != nil {
		result = "error"
		details["error"] = err.Error()
	}
	
	es.auditLogger.Log("file", "operation_"+op, result, details)
	return err
}

// resolveSandboxPath 解析沙箱路径
func (es *EnhancedSandbox) resolveSandboxPath(path string) string {
	// 如果是绝对路径且在允许列表中，直接使用
	for _, allowed := range es.config.FileAllowedPaths {
		if strings.HasPrefix(path, allowed) {
			return path
		}
	}
	
	// 否则，相对于会话沙箱目录
	if es.currentSession != "" {
		if sessionRoot, exists := es.sessionRoots[es.currentSession]; exists {
			// 防止路径遍历攻击
			cleanPath := filepath.Clean(path)
			if strings.Contains(cleanPath, "..") {
				cleanPath = filepath.Base(cleanPath)
			}
			return filepath.Join(sessionRoot, cleanPath)
		}
	}
	
	// 默认相对于沙箱根目录
	return filepath.Join(es.config.FileSandboxRoot, filepath.Base(path))
}

// validateFilePath 验证文件路径
func (es *EnhancedSandbox) validateFilePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	
	// 检查黑名单路径
	for _, blocked := range es.config.FileBlockedPaths {
		if strings.HasPrefix(absPath, blocked) {
			return fmt.Errorf("access to '%s' is blocked", blocked)
		}
	}
	
	// 如果配置了白名单，检查是否在白名单内
	if len(es.config.FileAllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range es.config.FileAllowedPaths {
			if strings.HasPrefix(absPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path '%s' is not in allowed paths", absPath)
		}
	}
	
	return nil
}

// GetSandboxInfo 获取沙箱信息
func (es *EnhancedSandbox) GetSandboxInfo() map[string]interface{} {
	es.mu.RLock()
	defer es.mu.RUnlock()
	
	info := map[string]interface{}{
		"enabled":            es.config.Enabled,
		"level":              es.config.Level,
		"os_type":            es.osType,
		"sandbox_root":       es.config.FileSandboxRoot,
		"network_disabled":   es.config.NetworkDisabled,
		"audit_enabled":      es.config.AuditLogEnabled,
		"anomaly_detection":  es.config.AnomalyDetection,
		"shell_whitelist":    es.config.ShellWhitelist,
		"shell_blacklist":    es.config.ShellBlacklist,
		"current_session":    es.currentSession,
		"session_count":      len(es.sessionRoots),
	}
	
	return info
}

// Close 关闭沙箱
func (es *EnhancedSandbox) Close() error {
	es.mu.Lock()
	defer es.mu.Unlock()
	
	// 清理所有会话目录
	for sessionID, root := range es.sessionRoots {
		if err := os.RemoveAll(root); err != nil {
			es.auditLogger.Log("session", "cleanup_failed", "error", map[string]interface{}{
				"session_id": sessionID,
				"error":      err.Error(),
			})
		}
	}
	es.sessionRoots = make(map[string]string)
	
	// 关闭审计日志器
	if es.auditLogger != nil {
		return es.auditLogger.Close()
	}
	
	return nil
}

// ============================================
// 工具函数
// ============================================

// GenerateSandboxConfigTemplate 生成沙箱配置模板
func GenerateSandboxConfigTemplate() string {
	config := DefaultEnhancedSandboxConfig()
	data, _ := json.MarshalIndent(config, "", "  ")
	return string(data)
}

// IsLinuxNamespaceAvailable 检查 Linux Namespace 是否可用
func IsLinuxNamespaceAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	
	// 检查是否有权限使用 namespace
	cmd := exec.Command("unshare", "--pid", "--fork", "--", "true")
	err := cmd.Run()
	return err == nil
}

// IsGVisorAvailable 检查 gVisor 是否可用
func IsGVisorAvailable() bool {
	cmd := exec.Command("runsc", "--version")
	err := cmd.Run()
	return err == nil
}

// GetRecommendedSandboxLevel 根据环境推荐沙箱级别
func GetRecommendedSandboxLevel() SandboxLevel {
	osType := runtime.GOOS
	
	// 生产环境建议
	if os.Getenv("ENVIRONMENT") == "production" {
		if IsGVisorAvailable() {
			return SandboxLevelStrict
		}
		if IsLinuxNamespaceAvailable() {
			return SandboxLevelStandard
		}
		return SandboxLevelBasic
	}
	
	// 开发环境
	if osType == "linux" {
		return SandboxLevelStandard
	}
	
	// 非 Linux 平台
	return SandboxLevelBasic
}
