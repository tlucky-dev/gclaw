package mcp

import "encoding/json"

// JSON-RPC 2.0 标准结构
type JSONRPCRequest struct {
JSONRPC string          `json:"jsonrpc"`
ID      interface{}     `json:"id"` // int or string
Method  string          `json:"method"`
Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
JSONRPC string          `json:"jsonrpc"`
ID      interface{}     `json:"id"`
Result  json.RawMessage `json:"result,omitempty"`
Error   *JSONRPCError   `json:"error,omitempty"`
}

type JSONRPCError struct {
Code    int         `json:"code"`
Message string      `json:"message"`
Data    interface{} `json:"data,omitempty"`
}

// MCP 特定类型
type InitializeRequest struct {
ProtocolVersion string                 `json:"protocolVersion"`
Capabilities    ClientCapabilities     `json:"capabilities"`
ClientInfo      ClientInfo             `json:"clientInfo"`
Meta            map[string]interface{} `json:"_meta,omitempty"`
}

type ClientInfo struct {
Name    string `json:"name"`
Version string `json:"version"`
}

type ClientCapabilities struct {
// 预留未来扩展
}

type InitializeResult struct {
ProtocolVersion string             `json:"protocolVersion"`
Capabilities    ServerCapabilities `json:"capabilities"`
ServerInfo      ServerInfo         `json:"serverInfo"`
Instructions    string             `json:"instructions,omitempty"`
}

type ServerInfo struct {
Name    string `json:"name"`
Version string `json:"version"`
}

type ServerCapabilities struct {
Tools *ToolCapabilities `json:"tools,omitempty"`
}

type ToolCapabilities struct {
ListChanged bool `json:"listChanged,omitempty"`
}

// 工具相关
type ListToolsRequest struct {
Cursor string `json:"cursor,omitempty"`
}

type ListToolsResult struct {
Tools      []Tool `json:"tools"`
NextCursor string `json:"nextCursor,omitempty"`
}

type Tool struct {
Name        string          `json:"name"`
Description string          `json:"description,omitempty"`
InputSchema json.RawMessage `json:"inputSchema"` // JSON Schema
}

type CallToolRequest struct {
Name      string                 `json:"name"`
Arguments map[string]interface{} `json:"arguments,omitempty"`
Meta      map[string]interface{} `json:"_meta,omitempty"`
}

type CallToolResult struct {
Content []ContentBlock `json:"content"`
IsError bool           `json:"isError,omitempty"`
}

type ContentBlock struct {
Type     string          `json:"type"` // "text", "image", "resource"
Text     string          `json:"text,omitempty"`
Data     string          `json:"data,omitempty"`     // base64 for image
MimeType string          `json:"mimeType,omitempty"` // for image/resource
Resource json.RawMessage `json:"resource,omitempty"`
}

// 通知类型
type Notification struct {
JSONRPC string          `json:"jsonrpc"`
Method  string          `json:"method"`
Params  json.RawMessage `json:"params,omitempty"`
}
