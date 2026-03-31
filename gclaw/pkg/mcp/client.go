package mcp

import (
"bufio"
"bytes"
"encoding/json"
"fmt"
"io"
"os/exec"
"sync"
"time"
)

// Client MCP 客户端，通过 stdio 与 MCP Server 通信
type Client struct {
cmd       *exec.Cmd
stdin     io.WriteCloser
stdout    *bufio.Reader
mu        sync.Mutex
requestID int64
closed    bool
}

// NewClient 创建新的 MCP 客户端
func NewClient(command string, args ...string) (*Client, error) {
cmd := exec.Command(command, args...)

stdin, err := cmd.StdinPipe()
if err != nil {
return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
}

stdout, err := cmd.StdoutPipe()
if err != nil {
return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
}

client := &Client{
cmd:    cmd,
stdin:  stdin,
stdout: bufio.NewReader(stdout),
}

// 启动进程
if err := cmd.Start(); err != nil {
return nil, fmt.Errorf("failed to start command: %w", err)
}

return client, nil
}

// Initialize 初始化 MCP 连接
func (c *Client) Initialize(clientInfo ClientInfo) (*InitializeResult, error) {
req := InitializeRequest{
ProtocolVersion: "2024-11-05",
Capabilities:    ClientCapabilities{},
ClientInfo:      clientInfo,
}

var result InitializeResult
if err := c.sendRequest("initialize", req, &result); err != nil {
return nil, err
}

// 发送 initialized 通知
notification := Notification{
JSONRPC: "2.0",
Method:  "notifications/initialized",
}
if err := c.sendNotification(notification); err != nil {
return nil, fmt.Errorf("failed to send initialized notification: %w", err)
}

return &result, nil
}

// ListTools 获取可用工具列表
func (c *Client) ListTools() (*ListToolsResult, error) {
var result ListToolsResult
if err := c.sendRequest("tools/list", ListToolsRequest{}, &result); err != nil {
return nil, err
}
return &result, nil
}

// CallTool 调用指定工具
func (c *Client) CallTool(name string, arguments map[string]interface{}) (*CallToolResult, error) {
req := CallToolRequest{
Name:      name,
Arguments: arguments,
}

var result CallToolResult
if err := c.sendRequest("tools/call", req, &result); err != nil {
return nil, err
}
return &result, nil
}

// Close 关闭客户端连接
func (c *Client) Close() error {
c.mu.Lock()
defer c.mu.Unlock()

if c.closed {
return nil
}

c.closed = true

if err := c.stdin.Close(); err != nil {
return err
}

// 等待进程退出，最多等 5 秒
done := make(chan error, 1)
go func() {
done <- c.cmd.Wait()
}()

select {
case err := <-done:
return err
case <-time.After(5 * time.Second):
// 超时则强制终止
return c.cmd.Process.Kill()
}
}

// sendRequest 发送 JSON-RPC 请求并接收响应
func (c *Client) sendRequest(method string, params interface{}, result interface{}) error {
c.mu.Lock()
defer c.mu.Unlock()

c.requestID++
id := c.requestID

req := JSONRPCRequest{
JSONRPC: "2.0",
ID:      id,
Method:  method,
}

if params != nil {
data, err := json.Marshal(params)
if err != nil {
return fmt.Errorf("failed to marshal params: %w", err)
}
req.Params = data
}

// 发送请求
if err := c.writeRequest(req); err != nil {
return err
}

// 读取响应
response, err := c.readResponse()
if err != nil {
return err
}

// 检查错误
if response.Error != nil {
return fmt.Errorf("RPC error %d: %s", response.Error.Code, response.Error.Message)
}

// 解析结果
if err := json.Unmarshal(response.Result, result); err != nil {
return fmt.Errorf("failed to unmarshal result: %w", err)
}

return nil
}

// sendNotification 发送通知（不需要响应）
func (c *Client) sendNotification(notification Notification) error {
c.mu.Lock()
defer c.mu.Unlock()

data, err := json.Marshal(notification)
if err != nil {
return err
}

_, err = c.stdin.Write(append(data, '\n'))
return err
}

// writeRequest 写入请求到 stdin
func (c *Client) writeRequest(req JSONRPCRequest) error {
data, err := json.Marshal(req)
if err != nil {
return err
}

_, err = c.stdin.Write(append(data, '\n'))
return err
}

// readResponse 从 stdout 读取响应
func (c *Client) readResponse() (*JSONRPCResponse, error) {
line, err := c.stdout.ReadBytes('\n')
if err != nil {
return nil, fmt.Errorf("failed to read response: %w", err)
}

// 去掉末尾的换行符
line = bytes.TrimSpace(line)

var response JSONRPCResponse
if err := json.Unmarshal(line, &response); err != nil {
return nil, fmt.Errorf("failed to parse response: %w", err)
}

return &response, nil
}
