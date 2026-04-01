package skill

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ThirdPartyScannerConfig 第三方扫描器配置
type ThirdPartyScannerConfig struct {
	Name        string            `json:"name"`
	Enabled     bool              `json:"enabled"`
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Timeout     time.Duration     `json:"timeout"`
	InputFormat string            `json:"input_format"` // json, text, file
	OutputFormat string           `json:"output_format"` // json, text
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

// ThirdPartyScannerResult 第三方扫描结果
type ThirdPartyScannerResult struct {
	ScannerName string    `json:"scanner_name"`
	ScanTime    time.Time `json:"scan_time"`
	Passed      bool      `json:"passed"`
	Issues      []string  `json:"issues,omitempty"`
	Warnings    []string  `json:"warnings,omitempty"`
	RiskLevel   RiskLevel `json:"risk_level"`
	RawOutput   string    `json:"raw_output,omitempty"`
}

// ThirdPartyScannerManager 第三方扫描器管理器
type ThirdPartyScannerManager struct {
	mu        sync.RWMutex
	scanners  []*ThirdPartyScannerConfig
	results   map[string][]*ThirdPartyScannerResult
	cacheDir  string
}

// NewThirdPartyScannerManager 创建第三方扫描器管理器
func NewThirdPartyScannerManager(cacheDir string) *ThirdPartyScannerManager {
	return &ThirdPartyScannerManager{
		scanners: make([]*ThirdPartyScannerConfig, 0),
		results:  make(map[string][]*ThirdPartyScannerResult),
		cacheDir: cacheDir,
	}
}

// RegisterScanner 注册第三方扫描器
func (m *ThirdPartyScannerManager) RegisterScanner(config *ThirdPartyScannerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if config.Name == "" {
		return fmt.Errorf("scanner name is required")
	}
	if config.Command == "" {
		return fmt.Errorf("scanner command is required")
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}

	// 检查命令是否存在
	if _, err := exec.LookPath(config.Command); err != nil {
		// 命令不存在，但允许注册（可能在 PATH 中或后续安装）
		fmt.Printf("Warning: scanner command '%s' not found in PATH: %v\n", config.Command, err)
	}

	m.scanners = append(m.scanners, config)
	return nil
}

// ScanWithThirdParty 使用第三方扫描器执行扫描
func (m *ThirdPartyScannerManager) ScanWithThirdParty(skill SkillDefinition) ([]*ThirdPartyScannerResult, error) {
	m.mu.RLock()
	scanners := make([]*ThirdPartyScannerConfig, len(m.scanners))
	copy(scanners, m.scanners)
	m.mu.RUnlock()

	var results []*ThirdPartyScannerResult
	var lastErr error

	for _, scanner := range scanners {
		if !scanner.Enabled {
			continue
		}

		result, err := m.executeScanner(scanner, skill)
		if err != nil {
			lastErr = err
			result = &ThirdPartyScannerResult{
				ScannerName: scanner.Name,
				ScanTime:    time.Now(),
				Passed:      false,
				Issues:      []string{fmt.Sprintf("scanner execution failed: %v", err)},
				RiskLevel:   RiskLevelHigh,
			}
		}

		results = append(results, result)

		// 保存结果到缓存
		m.saveResultToCache(scanner.Name, skill.GetMetadata().ID, result)
	}

	m.mu.Lock()
	skillID := skill.GetMetadata().ID
	m.results[skillID] = append(m.results[skillID], results...)
	m.mu.Unlock()

	return results, lastErr
}

// executeScanner 执行单个第三方扫描器
func (m *ThirdPartyScannerManager) executeScanner(config *ThirdPartyScannerConfig, skill SkillDefinition) (*ThirdPartyScannerResult, error) {
	metadata := skill.GetMetadata()

	// 准备输入数据
	var inputData []byte
	var err error

	switch config.InputFormat {
	case "json":
		inputData, err = json.Marshal(metadata)
	case "text":
		inputData = []byte(fmt.Sprintf("%s\n%s\n%s", metadata.Name, metadata.Version, metadata.Description))
	case "file":
		// 创建临时文件
		tmpFile, err := os.CreateTemp(m.cacheDir, "skill_scan_*.json")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmpFile.Name())

		if err := json.NewEncoder(tmpFile).Encode(metadata); err != nil {
			tmpFile.Close()
			return nil, fmt.Errorf("failed to write temp file: %w", err)
		}
		tmpFile.Close()
		inputData = []byte(tmpFile.Name())
	default:
		inputData, err = json.Marshal(metadata)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to prepare input data: %w", err)
	}

	// 构建命令
	args := make([]string, len(config.Args))
	copy(args, config.Args)

	// 替换占位符
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "{{skill_id}}", metadata.ID)
		args[i] = strings.ReplaceAll(arg, "{{skill_name}}", metadata.Name)
		args[i] = strings.ReplaceAll(arg, "{{input_file}}", string(inputData))
	}

	cmd := exec.Command(config.Command, args...)

	// 设置环境变量
	env := os.Environ()
	for k, v := range config.EnvVars {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env

	// 设置标准输入
	if config.InputFormat != "file" {
		cmd.Stdin = bytes.NewReader(inputData)
	}

	// 执行命令（带超时）
	done := make(chan struct {
		output []byte
		err    error
	}, 1)

	go func() {
		output, err := cmd.CombinedOutput()
		done <- struct {
			output []byte
			err    error
		}{output: output, err: err}
	}()

	select {
	case res := <-done:
		return m.parseScannerOutput(config, res.output, res.err)
	case <-time.After(config.Timeout):
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil, fmt.Errorf("scanner timeout after %v", config.Timeout)
	}
}

// parseScannerOutput 解析扫描器输出
func (m *ThirdPartyScannerManager) parseScannerOutput(config *ThirdPartyScannerConfig, output []byte, execErr error) (*ThirdPartyScannerResult, error) {
	result := &ThirdPartyScannerResult{
		ScannerName: config.Name,
		ScanTime:    time.Now(),
		RawOutput:   string(output),
	}

	if execErr != nil {
		result.Passed = false
		result.Issues = append(result.Issues, fmt.Sprintf("execution error: %v", execErr))
		result.RiskLevel = RiskLevelHigh
		return result, nil
	}

	// 根据输出格式解析结果
	switch config.OutputFormat {
	case "json":
		// 尝试解析 JSON 输出
		var jsonResult struct {
			Passed   bool     `json:"passed"`
			Issues   []string `json:"issues"`
			Warnings []string `json:"warnings"`
			Risk     string   `json:"risk_level"`
		}
		if err := json.Unmarshal(output, &jsonResult); err == nil {
			result.Passed = jsonResult.Passed
			result.Issues = jsonResult.Issues
			result.Warnings = jsonResult.Warnings
			if jsonResult.Risk != "" {
				result.RiskLevel = RiskLevel(jsonResult.Risk)
			} else {
				result.RiskLevel = RiskLevelLow
				if len(jsonResult.Issues) > 0 {
					result.RiskLevel = RiskLevelMedium
				}
			}
			return result, nil
		}
		// JSON 解析失败，降级为文本解析
		fallthrough
	case "text":
		// 简单的文本解析：查找关键词
		outputStr := string(output)
		if strings.Contains(strings.ToLower(outputStr), "error") ||
			strings.Contains(strings.ToLower(outputStr), "critical") ||
			strings.Contains(strings.ToLower(outputStr), "failed") {
			result.Passed = false
			result.RiskLevel = RiskLevelHigh
			result.Issues = append(result.Issues, outputStr)
		} else if strings.Contains(strings.ToLower(outputStr), "warning") {
			result.Passed = true
			result.RiskLevel = RiskLevelLow
			result.Warnings = append(result.Warnings, outputStr)
		} else {
			result.Passed = true
			result.RiskLevel = RiskLevelLow
		}
	}

	return result, nil
}

// saveResultToCache 保存结果到缓存
func (m *ThirdPartyScannerManager) saveResultToCache(scannerName, skillID string, result *ThirdPartyScannerResult) {
	if m.cacheDir == "" {
		return
	}

	cacheFile := filepath.Join(m.cacheDir, fmt.Sprintf("%s_%s_result.json", scannerName, skillID))
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return
	}

	_ = os.WriteFile(cacheFile, data, 0644)
}

// GetCachedResults 获取缓存的扫描结果
func (m *ThirdPartyScannerManager) GetCachedResults(skillID string) []*ThirdPartyScannerResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.results[skillID]
}

// ListScanners 列出所有注册的扫描器
func (m *ThirdPartyScannerManager) ListScanners() []*ThirdPartyScannerConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*ThirdPartyScannerConfig, len(m.scanners))
	copy(result, m.scanners)
	return result
}

// RemoveScanner 移除扫描器
func (m *ThirdPartyScannerManager) RemoveScanner(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, scanner := range m.scanners {
		if scanner.Name == name {
			m.scanners = append(m.scanners[:i], m.scanners[i+1:]...)
			return true
		}
	}
	return false
}

// CommonThirdPartyScanners 常见第三方扫描器预设配置
var CommonThirdPartyScannerConfigs = map[string]*ThirdPartyScannerConfig{
	// TruffleHog - 检测代码中的密钥和凭据
	"trufflehog": {
		Name:        "trufflehog",
		Enabled:     false, // 默认禁用，需要手动启用
		Command:     "trufflehog",
		Args:        []string{"git", "file://{{input_file}}", "--json"},
		Timeout:     60 * time.Second,
		InputFormat: "file",
		OutputFormat: "json",
	},
	// Semgrep - 静态代码分析
	"semgrep": {
		Name:        "semgrep",
		Enabled:     false,
		Command:     "semgrep",
		Args:        []string{"--json", "--config", "auto", "{{input_file}}"},
		Timeout:     120 * time.Second,
		InputFormat: "file",
		OutputFormat: "json",
	},
	// Bandit - Python 安全扫描
	"bandit": {
		Name:        "bandit",
		Enabled:     false,
		Command:     "bandit",
		Args:        []string{"-f", "json", "-o", "/dev/stdout", "{{input_file}}"},
		Timeout:     60 * time.Second,
		InputFormat: "file",
		OutputFormat: "json",
	},
	// ShellCheck - Shell 脚本检查
	"shellcheck": {
		Name:        "shellcheck",
		Enabled:     false,
		Command:     "shellcheck",
		Args:        []string{"-f", "json", "{{input_file}}"},
		Timeout:     30 * time.Second,
		InputFormat: "file",
		OutputFormat: "json",
	},
	// ClamAV - 病毒扫描
	"clamav": {
		Name:        "clamav",
		Enabled:     false,
		Command:     "clamscan",
		Args:        []string{"--stdout", "{{input_file}}"},
		Timeout:     120 * time.Second,
		InputFormat: "file",
		OutputFormat: "text",
	},
}

// EnableCommonScanner 启用常见扫描器
func (m *ThirdPartyScannerManager) EnableCommonScanner(name string) error {
	config, exists := CommonThirdPartyScannerConfigs[name]
	if !exists {
		return fmt.Errorf("unknown scanner: %s", name)
	}

	// 克隆配置并启用
	clone := *config
	clone.Enabled = true

	return m.RegisterScanner(&clone)
}

// EnableAllCommonScanners 启用所有常见扫描器（如果可用）
func (m *ThirdPartyScannerManager) EnableAllCommonScanners() []string {
	var enabled []string
	var unavailable []string

	for name, config := range CommonThirdPartyScannerConfigs {
		// 检查命令是否可用
		if _, err := exec.LookPath(config.Command); err == nil {
			clone := *config
			clone.Enabled = true
			if err := m.RegisterScanner(&clone); err == nil {
				enabled = append(enabled, name)
			}
		} else {
			unavailable = append(unavailable, name)
		}
	}

	if len(unavailable) > 0 {
		fmt.Printf("Note: The following scanners are not available: %v\n", unavailable)
	}

	return enabled
}
