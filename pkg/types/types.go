package types

// MessageRole 消息角色
type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

// Message 对话消息
type Message struct {
	Role           MessageRole `json:"role"`
	Content        string      `json:"content"`
	Name           string      `json:"name,omitempty"`
	ToolCalls      []ToolCall  `json:"tool_calls,omitempty"`
	ToolCallID     string      `json:"tool_call_id,omitempty"`
	
	// 适配器相关字段（用于外部消息源）
	ID             string                 `json:"id,omitempty"`
	ConversationID string                 `json:"conversation_id,omitempty"`
	SenderID       string                 `json:"sender_id,omitempty"`
	Timestamp      interface{}            `json:"timestamp,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ToolCall 工具调用
type ToolCall struct {
	ID       string        `json:"id"`
	Type     string        `json:"type"`
	Function FunctionCall  `json:"function"`
}

// FunctionCall 函数调用
type FunctionCall struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolDefinition 工具定义
type ToolDefinition struct {
	Type        string          `json:"type"`
	Function    FunctionDetails `json:"function"`
}

// FunctionDetails 函数详情
type FunctionDetails struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages    []Message        `json:"messages"`
	Model       string           `json:"model,omitempty"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
}

// ChatResponse 聊天响应
type ChatResponse struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	Message   Message   `json:"message"`
	Usage     Usage     `json:"usage,omitempty"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Usage token 使用情况
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
