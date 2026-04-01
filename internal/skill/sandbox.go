package skill

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// SandboxConfig 沙箱配置
type SandboxConfig struct {
	Enabled         bool          `json:"enabled"`
	MaxMemoryMB     int64         `json:"max_memory_mb"`
	MaxCPUPercent   float64       `json:"max_cpu_percent"`
	Timeout         time.Duration `json:"timeout"`
	AllowedPaths    []string      `json:"allowed_paths"`
	BlockedPaths    []string      `json:"blocked_paths"`
	NetworkDisabled bool          `json:"network_disabled"`
	UseGVisor       bool          `json:"use_gvisor"` // 是否使用 gVisor
}

// DefaultSandboxConfig 默认沙箱配置
func DefaultSandboxConfig() *SandboxConfig {
	return &SandboxConfig{
		Enabled:         true,
		MaxMemoryMB:     512,
		MaxCPUPercent:   50.0,
		Timeout:         30 * time.Second,
		AllowedPaths:    []string{"/tmp", "/var/tmp"},
		BlockedPaths:    []string{"/etc", "/root", "/proc", "/sys"},
		NetworkDisabled: true,
		UseGVisor:       false,
	}
}

// SkillSandbox 技能沙箱执行环境
type SkillSandbox struct {
	mu       sync.Mutex
	config   *SandboxConfig
	running  map[string]*SandboxInstance
}

// SandboxInstance 沙箱实例
type SandboxInstance struct {
	ID          string
	SkillID     string
	Pid         int
	StartTime   time.Time
	MemoryLimit int64
	CPULimit    float64
	CancelFunc  context.CancelFunc
	Context     context.Context
}

// NewSkillSandbox 创建技能沙箱
func NewSkillSandbox(config *SandboxConfig) *SkillSandbox {
	if config == nil {
		config = DefaultSandboxConfig()
	}
	
	return &SkillSandbox{
		config:  config,
		running: make(map[string]*SandboxInstance),
	}
}

// ExecuteInSandbox 在沙箱中执行技能
func (s *SkillSandbox) ExecuteInSandbox(skillID string, executeFunc func() (string, error)) (string, error) {
	s.mu.Lock()
	
	// 检查是否启用沙箱
	if !s.config.Enabled {
		s.mu.Unlock()
		return executeFunc()
	}
	
	// 创建沙箱实例
	instanceID := fmt.Sprintf("%s_%d", skillID, time.Now().UnixNano())
	ctx, cancel := context.WithTimeout(context.Background(), s.config.Timeout)
	
	instance := &SandboxInstance{
		ID:          instanceID,
		SkillID:     skillID,
		StartTime:   time.Now(),
		MemoryLimit: s.config.MaxMemoryMB,
		CPULimit:    s.config.MaxCPUPercent,
		CancelFunc:  cancel,
		Context:     ctx,
	}
	
	s.running[instanceID] = instance
	s.mu.Unlock()
	
	defer func() {
		s.mu.Lock()
		delete(s.running, instanceID)
		s.mu.Unlock()
		cancel()
	}()
	
	// 执行结果通道
	resultChan := make(chan struct {
		result string
		err    error
	}, 1)
	
	// 在 goroutine 中执行
	go func() {
		result, err := executeFunc()
		resultChan <- struct {
			result string
			err    error
		}{result: result, err: err}
	}()
	
	// 等待执行完成或超时
	select {
	case res := <-resultChan:
		return res.result, res.err
	case <-ctx.Done():
		return "", fmt.Errorf("skill execution timeout after %v", s.config.Timeout)
	}
}

// ExecResult 命令执行结果
type ExecResult struct {
	Output []byte
	Err    error
}

// ExecuteCommandInSandbox 在沙箱中执行外部命令（使用系统工具限制资源）
func (s *SkillSandbox) ExecuteCommandInSandbox(cmd *exec.Cmd) ([]byte, error) {
	if !s.config.Enabled {
		return cmd.CombinedOutput()
	}
	
	// 创建工作目录检查
	if len(s.config.AllowedPaths) > 0 {
		workDir := cmd.Dir
		if workDir == "" {
			var err error
			workDir, err = os.Getwd()
			if err != nil {
				return nil, fmt.Errorf("failed to get working directory: %w", err)
			}
		}
		
		absWorkDir, err := filepath.Abs(workDir)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve working directory: %w", err)
		}
		
		if !isPathAllowed(absWorkDir, s.config.AllowedPaths, s.config.BlockedPaths) {
			return nil, fmt.Errorf("working directory not allowed: %s", workDir)
		}
	}
	
	// 设置环境变量以禁用网络（如果配置了）
	if s.config.NetworkDisabled {
		cmd.Env = append(cmd.Env, "NO_NETWORK=1")
	}
	
	// 执行命令并处理超时
	done := make(chan ExecResult, 1)
	
	go func() {
		output, err := cmd.CombinedOutput()
		done <- ExecResult{Output: output, Err: err}
	}()
	
	select {
	case result := <-done:
		return result.Output, result.Err
	case <-time.After(s.config.Timeout):
		// 超时终止进程
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("command execution timeout after %v", s.config.Timeout)
	}
}

// GetRunningInstances 获取所有运行中的沙箱实例
func (s *SkillSandbox) GetRunningInstances() []*SandboxInstance {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	instances := make([]*SandboxInstance, 0, len(s.running))
	for _, inst := range s.running {
		instances = append(instances, inst)
	}
	
	return instances
}

// TerminateInstance 终止指定的沙箱实例
func (s *SkillSandbox) TerminateInstance(instanceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	instance, exists := s.running[instanceID]
	if !exists {
		return fmt.Errorf("instance not found: %s", instanceID)
	}
	
	if instance.CancelFunc != nil {
		instance.CancelFunc()
	}
	
	// 如果有 PID，尝试终止进程
	if instance.Pid > 0 {
		proc, err := os.FindProcess(instance.Pid)
		if err == nil {
			_ = proc.Kill()
		}
	}
	
	delete(s.running, instanceID)
	return nil
}

// TerminateAllInstances 终止所有沙箱实例
func (s *SkillSandbox) TerminateAllInstances() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	for id, instance := range s.running {
		if instance.CancelFunc != nil {
			instance.CancelFunc()
		}
		
		if instance.Pid > 0 {
			proc, err := os.FindProcess(instance.Pid)
			if err == nil {
				_ = proc.Kill()
			}
		}
		
		delete(s.running, id)
	}
	
	return nil
}

// UpdateConfig 更新沙箱配置
func (s *SkillSandbox) UpdateConfig(config *SandboxConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if config != nil {
		s.config = config
	}
}

// isPathAllowed 检查路径是否在允许列表中且不在阻止列表中
func isPathAllowed(path string, allowed, blocked []string) bool {
	// 首先检查是否在阻止列表中
	for _, blockedPath := range blocked {
		if path == blockedPath || filepath.HasPrefix(path, blockedPath+"/") {
			return false
		}
	}
	
	// 如果没有指定允许列表，则默认允许
	if len(allowed) == 0 {
		return true
	}
	
	// 检查是否在允许列表中
	for _, allowedPath := range allowed {
		if path == allowedPath || filepath.HasPrefix(path, allowedPath+"/") {
			return true
		}
	}
	
	return false
}

// getSysProcAttr 获取系统进程属性（用于资源限制）
func getSysProcAttr(config *SandboxConfig) interface{} {
	// 这里返回 nil，实际实现需要根据操作系统设置 cgroup 或其他限制
	// Linux 示例：设置 cgroup 限制
	// Windows 示例：使用 Job Object
	// macOS 示例：使用 resource limits
	
	// 注意：完整的实现需要导入 syscall 包并设置具体的平台相关属性
	// 这里提供一个框架，具体实现可以根据需求扩展
	return nil
}

// GVisorSandbox gVisor 沙箱实现（高级功能）
type GVisorSandbox struct {
	config *SandboxConfig
}

// NewGVisorSandbox 创建 gVisor 沙箱
func NewGVisorSandbox(config *SandboxConfig) *GVisorSandbox {
	return &GVisorSandbox{
		config: config,
	}
}

// IsAvailable 检查 gVisor 是否可用
func (g *GVisorSandbox) IsAvailable() bool {
	// 检查 runsc 命令是否存在
	cmd := exec.Command("runsc", "--version")
	err := cmd.Run()
	return err == nil
}

// ExecuteInGVisor 在 gVisor 中执行技能
func (g *GVisorSandbox) ExecuteInGVisor(skillID string, containerImage string, command []string) (string, error) {
	if !g.IsAvailable() {
		return "", fmt.Errorf("gVisor (runsc) is not available on this system")
	}
	
	// 构建 runsc 命令
	args := []string{
		"run",
		"--rootless",
	}
	
	// 添加内存限制
	if g.config.MaxMemoryMB > 0 {
		args = append(args, "--memory-limit", fmt.Sprintf("%d", g.config.MaxMemoryMB*1024*1024))
	}
	
	// 添加其他参数...
	args = append(args, containerImage)
	args = append(args, command...)
	
	cmd := exec.Command("runsc", args...)
	
	ctx, cancel := context.WithTimeout(context.Background(), g.config.Timeout)
	defer cancel()
	
	// Go 1.19 compatible: use cmd.Wait() with context instead of WithContext
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	
	select {
	case err := <-done:
		output, _ := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("gVisor execution failed: %w, output: %s", err, string(output))
		}
		return string(output), nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", fmt.Errorf("gVisor execution timeout after %v", g.config.Timeout)
	}
}
