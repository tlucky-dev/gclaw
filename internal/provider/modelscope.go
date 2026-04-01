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

// ModelScopeProvider 魔搭社区（ModelScope）提供商实现
type ModelScopeProvider struct {
	apiKey  string
	baseURL string
	model   string
	timeout int
	client  *http.Client
}

// NewModelScopeProvider 创建新的 ModelScope 提供商
func NewModelScopeProvider(apiKey, baseURL, model string, timeout int) *ModelScopeProvider {
	if baseURL == "" {
		// ModelScope 默认 API 地址
		baseURL = "https://dashscope.aliyuncs.com/api/v1"
	}
	if model == "" {
		// 默认使用通义千问模型
		model = "qwen-turbo"
	}
	if timeout <= 0 {
		timeout = 30
	}

	return &ModelScopeProvider{
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
func (p *ModelScopeProvider) Name() string {
	return "modelscope"
}

// ChatRequest ModelScope 聊天请求格式
type ChatRequest struct {
	Model     string    `json:"model"`
	Input     InputData `json:"input"`
	Parameters *Parameters `json:"parameters,omitempty"`
}

// InputData 输入数据结构
type InputData struct {
	Messages []types.Message `json:"messages"`
}

// Parameters 可选参数
type Parameters struct {
	Temperature   float64 `json:"temperature,omitempty"`
	MaxTokens     int     `json:"max_tokens,omitempty"`
	TopP          float64 `json:"top_p,omitempty"`
	Stop          []string `json:"stop,omitempty"`
	EnableSearch  bool    `json:"enable_search,omitempty"`
	IncrementalOutput bool `json:"incremental_output,omitempty"`
}

// ChatResponse ModelScope 聊天响应格式
type ChatResponse struct {
	Output  OutputData `json:"output"`
	Usage   UsageInfo  `json:"usage"`
	RequestID string   `json:"request_id"`
}

// OutputData 输出数据
type OutputData struct {
	Text         string           `json:"text,omitempty"`
	Choices      []OutputChoice   `json:"choices,omitempty"`
	FinishReason string           `json:"finish_reason,omitempty"`
}

// OutputChoice 输出选择（ModelScope 特定）
type OutputChoice struct {
	Message      types.Message `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// UsageInfo 使用信息
type UsageInfo struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Chat 发送聊天请求到 ModelScope
func (p *ModelScopeProvider) Chat(request *types.ChatRequest) (*types.ChatResponse, error) {
	if request.Model == "" {
		request.Model = p.model
	}

	url := fmt.Sprintf("%s/services/aigc/text-generation/generation", p.baseURL)

	// 构建 ModelScope 特定格式的请求
	msRequest := ChatRequest{
		Model: request.Model,
		Input: InputData{
			Messages: request.Messages,
		},
		Parameters: &Parameters{
			Temperature:   request.Temperature,
			MaxTokens:     request.MaxTokens,
			TopP:          0.8,
			EnableSearch:  false,
			IncrementalOutput: false,
		},
	}

	jsonData, err := json.Marshal(msRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置 ModelScope 特定的请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	req.Header.Set("X-DashScope-SSE", "disable") // 禁用 SSE，使用普通响应

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
		return nil, fmt.Errorf("ModelScope API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析 ModelScope 响应格式
	var msResponse ChatResponse
	err = json.Unmarshal(body, &msResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// 转换为标准 ChatResponse 格式
	response := &types.ChatResponse{
		ID:      msResponse.RequestID,
		Model:   request.Model,
		Message: types.Message{
			Role:    "assistant",
			Content: msResponse.Output.Text,
		},
		Usage: types.Usage{
			PromptTokens:     msResponse.Usage.InputTokens,
			CompletionTokens: msResponse.Usage.OutputTokens,
			TotalTokens:      msResponse.Usage.TotalTokens,
		},
		FinishReason: msResponse.Output.FinishReason,
	}

	return response, nil
}

// StreamChat 流式聊天（ModelScope 支持 SSE）
func (p *ModelScopeProvider) StreamChat(request *types.ChatRequest, callback func(*types.ChatResponse) error) error {
	if request.Model == "" {
		request.Model = p.model
	}

	url := fmt.Sprintf("%s/services/aigc/text-generation/generation", p.baseURL)

	msRequest := ChatRequest{
		Model: request.Model,
		Input: InputData{
			Messages: request.Messages,
		},
		Parameters: &Parameters{
			Temperature:   request.Temperature,
			MaxTokens:     request.MaxTokens,
			IncrementalOutput: true, // 启用增量输出
		},
	}

	jsonData, err := json.Marshal(msRequest)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))
	req.Header.Set("X-DashScope-SSE", "enable") // 启用 SSE
	req.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ModelScope API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 处理 SSE 流 - 简化版本，逐行读取
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			// 简单解析 SSE 数据
			lines := bytes.Split(buf[:n], []byte("\n"))
			for _, line := range lines {
				if bytes.HasPrefix(line, []byte("data:")) {
					data := bytes.TrimPrefix(line, []byte("data:"))
					data = bytes.TrimSpace(data)
					
					var msResponse ChatResponse
					if err := json.Unmarshal(data, &msResponse); err != nil {
						continue
					}

					response := &types.ChatResponse{
						ID:    msResponse.RequestID,
						Model: request.Model,
						Message: types.Message{
							Role:    "assistant",
							Content: msResponse.Output.Text,
						},
					}

					if err := callback(response); err != nil {
						return err
					}
				}
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("error reading stream: %w", err)
		}
	}

	return nil
}
