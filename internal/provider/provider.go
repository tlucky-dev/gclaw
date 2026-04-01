package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gclaw/pkg/types"
)

// Provider LLM 提供商接口
type Provider interface {
	Chat(request *types.ChatRequest) (*types.ChatResponse, error)
	Name() string
}

// OpenAIProvider OpenAI 提供商实现
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	timeout int
	client  *http.Client
}

// NewOpenAIProvider 创建新的 OpenAI 提供商
func NewOpenAIProvider(apiKey, baseURL, model string, timeout int) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo"
	}
	if timeout <= 0 {
		timeout = 30
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Chat 发送聊天请求
func (p *OpenAIProvider) Chat(request *types.ChatRequest) (*types.ChatResponse, error) {
	if request.Model == "" {
		request.Model = p.model
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", string(body))
	}

	var response types.ChatResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &response, nil
}

// AnthropicProvider Anthropic 提供商实现
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	model   string
	timeout int
	client  *http.Client
}

// NewAnthropicProvider 创建新的 Anthropic 提供商
func NewAnthropicProvider(apiKey, baseURL, model string, timeout int) *AnthropicProvider {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	if model == "" {
		model = "claude-3-sonnet-20240229"
	}
	if timeout <= 0 {
		timeout = 30
	}

	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		timeout: timeout,
		client: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
	}
}

// Name 返回提供商名称
func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

// Chat 发送聊天请求（Anthropic 格式）
func (p *AnthropicProvider) Chat(request *types.ChatRequest) (*types.ChatResponse, error) {
	// Anthropic API 格式转换和调用逻辑
	// 这里仅作为框架示例
	return nil, fmt.Errorf("Anthropic provider not fully implemented yet")
}

// ProviderFactory 提供商工厂函数
type ProviderFactory func(apiKey, baseURL, model string, timeout int) Provider

// GetProviderFactory 获取提供商工厂
func GetProviderFactory(name string) (ProviderFactory, error) {
	switch name {
	case "openai":
		return func(apiKey, baseURL, model string, timeout int) Provider {
			return NewOpenAIProvider(apiKey, baseURL, model, timeout)
		}, nil
	case "anthropic":
		return func(apiKey, baseURL, model string, timeout int) Provider {
			return NewAnthropicProvider(apiKey, baseURL, model, timeout)
		}, nil
	case "modelscope":
		return func(apiKey, baseURL, model string, timeout int) Provider {
			return NewModelScopeProvider(apiKey, baseURL, model, timeout)
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// CreateProvider 创建提供商实例
func CreateProvider(name, apiKey, baseURL, model string, timeout int) (Provider, error) {
	factory, err := GetProviderFactory(name)
	if err != nil {
		return nil, err
	}
	return factory(apiKey, baseURL, model, timeout), nil
}
