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
	Adapters AdaptersConfig `json:"adapters,omitempty"`
}

// AdaptersConfig 适配器配置
type AdaptersConfig struct {
	Feishu *FeishuConfig `json:"feishu,omitempty"`
}

// FeishuConfig 飞书配置
type FeishuConfig struct {
	Enabled           bool   `json:"enabled"`
	AppID             string `json:"app_id"`
	AppSecret         string `json:"app_secret"`
	EncryptKey        string `json:"encrypt_key,omitempty"`
	VerificationToken string `json:"verification_token,omitempty"`
	ServerAddr        string `json:"server_addr"` // HTTP 服务器监听地址，如 ":8080"
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

// SaveToFile 保存配置到文件
func (c *Config) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
