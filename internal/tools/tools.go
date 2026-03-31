package tools

import (
	"encoding/json"
	"fmt"
)

// ToolExecutor 工具执行器接口
type ToolExecutor interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(args map[string]interface{}) (string, error)
}

// ToolRegistry 工具注册表
type ToolRegistry struct {
	tools map[string]ToolExecutor
}

// NewToolRegistry 创建新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]ToolExecutor),
	}
}

// Register 注册工具
func (r *ToolRegistry) Register(tool ToolExecutor) {
	r.tools[tool.Name()] = tool
}

// Get 获取工具
func (r *ToolRegistry) Get(name string) (ToolExecutor, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List 列出所有工具
func (r *ToolRegistry) List() []ToolExecutor {
	result := make([]ToolExecutor, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// Execute 执行工具
func (r *ToolRegistry) Execute(name string, args map[string]interface{}) (string, error) {
	tool, exists := r.tools[name]
	if !exists {
		return "", fmt.Errorf("tool '%s' not found", name)
	}
	return tool.Execute(args)
}

// BaseTool 基础工具结构
type BaseTool struct {
	NameVal        string                 `json:"name"`
	DescriptionVal string                 `json:"description"`
	ParametersVal  map[string]interface{} `json:"parameters"`
}

// Name 返回工具名称
func (t *BaseTool) Name() string {
	return t.NameVal
}

// Description 返回工具描述
func (t *BaseTool) Description() string {
	return t.DescriptionVal
}

// Parameters 返回工具参数
func (t *BaseTool) Parameters() map[string]interface{} {
	return t.ParametersVal
}

// ShellTool Shell 命令执行工具
type ShellTool struct {
	BaseTool
}

// NewShellTool 创建 Shell 工具
func NewShellTool() *ShellTool {
	return &ShellTool{
		BaseTool: BaseTool{
			NameVal:        "shell",
			DescriptionVal: "Execute shell commands",
			ParametersVal: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The shell command to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}

// Execute 执行 Shell 命令
func (t *ShellTool) Execute(args map[string]interface{}) (string, error) {
	cmd, ok := args["command"].(string)
	if !ok {
		return "", fmt.Errorf("invalid command argument")
	}
	
	// 注意：实际实现中需要安全地执行 shell 命令
	// 这里仅作为示例框架
	return fmt.Sprintf("Would execute: %s", cmd), nil
}

// FileReadTool 文件读取工具
type FileReadTool struct {
	BaseTool
}

// NewFileReadTool 创建文件读取工具
func NewFileReadTool() *FileReadTool {
	return &FileReadTool{
		BaseTool: BaseTool{
			NameVal:        "file_read",
			DescriptionVal: "Read file contents",
			ParametersVal: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The path of the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
	}
}

// Execute 执行文件读取
func (t *FileReadTool) Execute(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid path argument")
	}
	
	// 注意：实际实现中需要安全地读取文件
	// 这里仅作为示例框架
	return fmt.Sprintf("Would read file: %s", path), nil
}

// FileWriteTool 文件写入工具
type FileWriteTool struct {
	BaseTool
}

// NewFileWriteTool 创建文件写入工具
func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{
		BaseTool: BaseTool{
			NameVal:        "file_write",
			DescriptionVal: "Write content to a file",
			ParametersVal: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "The path of the file to write",
					},
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The content to write",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}
}

// Execute 执行文件写入
func (t *FileWriteTool) Execute(args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("invalid path argument")
	}
	
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("invalid content argument")
	}
	
	// 注意：实际实现中需要安全地写入文件
	// 这里仅作为示例框架
	return fmt.Sprintf("Would write to %s: %s", path, content), nil
}

// SearchTool 搜索工具
type SearchTool struct {
	BaseTool
}

// NewSearchTool 创建搜索工具
func NewSearchTool() *SearchTool {
	return &SearchTool{
		BaseTool: BaseTool{
			NameVal:        "search",
			DescriptionVal: "Search the web or knowledge base",
			ParametersVal: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

// Execute 执行搜索
func (t *SearchTool) Execute(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("invalid query argument")
	}
	
	// 注意：实际实现中需要调用搜索 API
	// 这里仅作为示例框架
	return fmt.Sprintf("Would search for: %s", query), nil
}

// ToToolDefinition 将工具转换为 ToolDefinition
func ToToolDefinition(tool ToolExecutor) map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        tool.Name(),
			"description": tool.Description(),
			"parameters":  tool.Parameters(),
		},
	}
}

// ToJSONSchema 将参数转换为 JSON Schema 字符串
func ToJSONSchema(params map[string]interface{}) (string, error) {
	data, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
