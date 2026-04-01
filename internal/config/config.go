package config

import (
	"encoding/json"
	"os"
)

// Config 配置结构
type Config struct {
	Provider ProviderConfig `json:"provider"`
	Memory   MemoryConfig   `json:"memory"`
	Engine   EngineConfig   `json:"engine"`
	Sandbox  SandboxConfig  `json:"sandbox,omitempty"`
}

// SandboxConfig 沙箱配置（嵌入增强的沙箱配置）
type SandboxConfig struct {
	Enabled              bool                    `json:"enabled"`
	Level                string                  `json:"level"`
	TimeoutSeconds       int                     `json:"timeout_seconds"`
	MaxMemoryMB          int64                   `json:"max_memory_mb"`
	MaxCPUPercent        float64                 `json:"max_cpu_percent"`
	MaxFileSizeMB        int64                   `json:"max_file_size_mb"`
	ShellWhitelist       []string                `json:"shell_whitelist"`
	ShellBlacklist       []string                `json:"shell_blacklist"`
	ShellArgPatterns     []string                `json:"shell_arg_patterns"`
	FileSandboxRoot      string                  `json:"file_sandbox_root"`
	FileResetOnStart     bool                    `json:"file_reset_on_start"`
	FileAllowedPaths     []string                `json:"file_allowed_paths"`
	FileBlockedPaths     []string                `json:"file_blocked_paths"`
	NetworkDisabled      bool                    `json:"network_disabled"`
	NetworkWhitelist     []string                `json:"network_whitelist"`
	AuditLogEnabled      bool                    `json:"audit_log_enabled"`
	AuditLogLevel        string                  `json:"audit_log_level"`
	AuditLogPath         string                  `json:"audit_log_path"`
	AnomalyDetection     bool                    `json:"anomaly_detection"`
	MaxExecPerMinute     int                     `json:"max_exec_per_minute"`
	SecurityIntegration  bool                    `json:"security_integration"`
	UserLevelMapping     map[string]string       `json:"user_level_mapping"`
	UseGVisor            bool                    `json:"use_gvisor"`
	UseLinuxNamespace    bool                    `json:"use_linux_namespace"`
	CrossPlatformCompat  bool                    `json:"cross_platform_compat"`
	DryRun               bool                    `json:"dry_run"`
	VerboseLogging       bool                    `json:"verbose_logging"`
}

// ProviderConfig 提供商配置
type ProviderConfig struct {
	Name     string `json:"name"`
	APIKey   string `json:"api_key"`
	BaseURL  string `json:"base_url,omitempty"`
	Model    string `json:"model"`
	Timeout  int    `json:"timeout,omitempty"`
}

// MemoryConfig 内存配置
type MemoryConfig struct {
	Type       string `json:"type"` // memory, redis, etc.
	MaxSize    int    `json:"max_size"`
	Expiration int    `json:"expiration,omitempty"` // seconds
}

// EngineConfig 引擎配置
type EngineConfig struct {
	MaxIterations int     `json:"max_iterations"`
	Temperature   float64 `json:"temperature"`
	MaxTokens     int     `json:"max_tokens"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			Name:    "openai",
			Model:   "gpt-3.5-turbo",
			Timeout: 30,
		},
		Memory: MemoryConfig{
			Type:    "memory",
			MaxSize: 100,
		},
		Engine: EngineConfig{
			MaxIterations: 10,
			Temperature:   0.7,
			MaxTokens:     2048,
		},
		Sandbox: SandboxConfig{
			Enabled:           true,
			Level:             "standard",
			TimeoutSeconds:    30,
			MaxMemoryMB:       512,
			MaxCPUPercent:     50.0,
			MaxFileSizeMB:     100,
			ShellWhitelist:    []string{"ls", "cat", "echo", "pwd", "whoami", "date", "uname"},
			ShellBlacklist:    []string{"rm", "sudo", "su", "chmod", "chown", "mount", "umount", "dd"},
			ShellArgPatterns:  []string{"[;&|]", `\$\\(`, "`", ">/dev/tcp/", `curl.*\\|.*sh`, `wget.*\\|.*sh`},
			FileSandboxRoot:   "/tmp/gclaw-sandbox",
			FileResetOnStart:  true,
			FileAllowedPaths:  []string{"/tmp", "/var/tmp"},
			FileBlockedPaths:  []string{"/etc", "/root", "/proc", "/sys", "/dev"},
			NetworkDisabled:   true,
			NetworkWhitelist:  []string{},
			AuditLogEnabled:   true,
			AuditLogLevel:     "info",
			AuditLogPath:      "/var/log/gclaw/sandbox-audit.log",
			AnomalyDetection:  true,
			MaxExecPerMinute:  60,
			SecurityIntegration: true,
			UserLevelMapping: map[string]string{
				"admin":   "standard",
				"user":    "standard",
				"guest":   "strict",
				"trusted": "basic",
			},
			UseGVisor:         false,
			UseLinuxNamespace: true,
			CrossPlatformCompat: true,
			DryRun:            false,
			VerboseLogging:    false,
		},
	}
}

// LoadFromFile 从文件加载配置
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveToFile 保存配置到文件（包级别函数）
func SaveToFile(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// SaveToFile 保存配置到文件（方法）
func (c *Config) SaveToFile(path string) error {
	return SaveToFile(c, path)
}
