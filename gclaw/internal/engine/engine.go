package engine

import (
	"fmt"
	"log"

	"gclaw/internal/memory"
	"gclaw/internal/provider"
	"gclaw/internal/tools"
	"gclaw/pkg/errors"
	"gclaw/pkg/types"
)

// Engine 引擎接口
type Engine interface {
	Run(sessionID, userInput string) (*types.ChatResponse, error)
	Reset(sessionID string) error
}

// GCLawEngine gclaw 核心引擎
type GCLawEngine struct {
	provider    provider.Provider
	memory      memory.Memory
	toolRegistry *tools.ToolRegistry
	maxIterations int
	temperature   float64
	maxTokens     int
}

// NewGCLawEngine 创建新的引擎实例
func NewGCLawEngine(
	p provider.Provider,
	m memory.Memory,
	tr *tools.ToolRegistry,
	maxIterations int,
	temperature float64,
	maxTokens int,
) *GCLawEngine {
	return &GCLawEngine{
		provider:      p,
		memory:        m,
		toolRegistry:  tr,
		maxIterations: maxIterations,
		temperature:   temperature,
		maxTokens:     maxTokens,
	}
}

// Run 执行用户请求
func (e *GCLawEngine) Run(sessionID, userInput string) (*types.ChatResponse, error) {
	// 添加用户消息到记忆
	userMessage := types.Message{
		Role:    types.RoleUser,
		Content: userInput,
	}
	
	if err := e.memory.Add(sessionID, userMessage); err != nil {
		return nil, errors.WrapError(errors.ErrMemory, "failed to add message", err)
	}

	// 获取历史消息
	history, err := e.memory.Get(sessionID, e.maxIterations*2)
	if err != nil {
		return nil, errors.WrapError(errors.ErrMemory, "failed to get history", err)
	}

	// 构建工具列表
	toolDefs := make([]types.ToolDefinition, 0)
	for _, tool := range e.toolRegistry.List() {
		params := tool.Parameters()
		toolDef := types.ToolDefinition{
			Type: "function",
			Function: types.FunctionDetails{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  params,
			},
		}
		toolDefs = append(toolDefs, toolDef)
	}

	// 构建聊天请求
	chatRequest := &types.ChatRequest{
		Messages:    history,
		Tools:       toolDefs,
		Temperature: e.temperature,
		MaxTokens:   e.maxTokens,
	}

	// 迭代执行，处理工具调用
	var lastResponse *types.ChatResponse
	for iteration := 0; iteration < e.maxIterations; iteration++ {
		// 调用 LLM
		response, err := e.provider.Chat(chatRequest)
		if err != nil {
			return nil, errors.WrapError(errors.ErrProvider, "failed to call provider", err)
		}

		lastResponse = response

		// 检查是否需要调用工具
		if len(response.Message.ToolCalls) == 0 {
			// 没有工具调用，返回最终响应
			// 将助手响应添加到记忆
			if err := e.memory.Add(sessionID, response.Message); err != nil {
				log.Printf("Warning: failed to add assistant message to memory: %v", err)
			}
			return response, nil
		}

		// 处理工具调用
		toolMessages := make([]types.Message, 0)
		for _, toolCall := range response.Message.ToolCalls {
			// 解析参数
			args := toolCall.Function.Arguments
			
			// 执行工具
			result, err := e.toolRegistry.Execute(toolCall.Function.Name, args)
			if err != nil {
				// 工具执行失败，继续处理其他工具
				log.Printf("Tool execution error: %v", err)
				result = fmt.Sprintf("Error: %v", err)
			}

			// 添加工具结果消息
			toolMessage := types.Message{
				Role:       types.RoleTool,
				Content:    result,
				ToolCallID: toolCall.ID,
			}
			toolMessages = append(toolMessages, toolMessage)

			// 将工具结果添加到记忆
			if err := e.memory.Add(sessionID, toolMessage); err != nil {
				log.Printf("Warning: failed to add tool message to memory: %v", err)
			}
		}

		// 更新聊天请求，添加工具结果
		chatRequest.Messages = append(chatRequest.Messages, response.Message)
		chatRequest.Messages = append(chatRequest.Messages, toolMessages...)
	}

	// 达到最大迭代次数，返回最后的结果
	if lastResponse != nil {
		if err := e.memory.Add(sessionID, lastResponse.Message); err != nil {
			log.Printf("Warning: failed to add final message to memory: %v", err)
		}
	}

	return lastResponse, nil
}

// Reset 重置会话
func (e *GCLawEngine) Reset(sessionID string) error {
	return e.memory.Clear(sessionID)
}

// GetHistory 获取会话历史
func (e *GCLawEngine) GetHistory(sessionID string, limit int) ([]types.Message, error) {
	return e.memory.Get(sessionID, limit)
}

// ClearHistory 清空会话历史
func (e *GCLawEngine) ClearHistory(sessionID string) error {
	return e.memory.Clear(sessionID)
}
