# MCP 集成指南

## 什么是 MCP？

MCP (Model Context Protocol) 是一个开放协议，允许 AI 模型与外部工具和服务进行标准化通信。通过 MCP，gclaw 可以无缝连接各种 MCP Server，获得丰富的工具能力。

## 架构优势

### 为什么不自己造轮子？

1. **复用现有生态**：已有数千个 MCP Server 可用（文件系统、数据库、API 等）
2. **标准化接口**：统一的 JSON-RPC 2.0 协议
3. **安全隔离**：每个 Server 独立进程运行
4. **动态发现**：自动获取可用工具列表

### gclaw 的 MCP 集成方式

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   gclaw     │────▶│  MCP Client  │────▶│  MCP Server     │
│  (AI Core)  │◀────│  (stdio)     │◀────│  (e.g. filesystem)│
└─────────────┘     └──────────────┘     └─────────────────┘
                            │
                            ▼
                    ┌─────────────────┐
                    │  MCP Server     │
                    │  (e.g. postgres) │
                    └─────────────────┘
```

## 使用示例

### 1. 连接文件系统 MCP Server

```go
import "gclaw/pkg/mcp"

// 创建客户端
client, err := mcp.NewClient("npx", "-y", "@modelcontextprotocol/server-filesystem", "/home/user")
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// 初始化
result, err := client.Initialize(mcp.ClientInfo{
    Name:    "gclaw",
    Version: "0.1.0",
})

// 获取工具列表
tools, err := client.ListTools()
for _, tool := range tools.Tools {
    fmt.Printf("可用工具：%s - %s\n", tool.Name, tool.Description)
}

// 调用工具
callResult, err := client.CallTool("read_file", map[string]interface{}{
    "path": "/home/user/test.txt",
})
```

### 2. 配置文件集成

在 `config.json` 中添加 MCP Server 配置：

```json
{
  "mcp": {
    "servers": [
      {
        "name": "filesystem",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/home/user"],
        "enabled": true
      },
      {
        "name": "postgresql",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-postgres", "postgresql://localhost/mydb"],
        "enabled": false
      }
    ]
  }
}
```

## 可用的 MCP Servers

### 官方维护
- `@modelcontextprotocol/server-filesystem` - 文件系统操作
- `@modelcontextprotocol/server-postgres` - PostgreSQL 数据库
- `@modelcontextprotocol/server-memory` - 向量记忆存储
- `@modelcontextprotocol/server-github` - GitHub API
- `@modelcontextprotocol/server-slack` - Slack 集成

### 社区贡献
- `mcp-server-playwright` - 浏览器自动化
- `mcp-server-fetch` - HTTP 请求
- `mcp-server-shell` - Shell 命令执行
- 更多见：https://github.com/modelcontextprotocol/servers

## 飞书集成方案

### 方案 A：使用现有 MCP Server
如果存在飞书 MCP Server，直接配置使用：
```json
{
  "mcp": {
    "servers": [
      {
        "name": "feishu",
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-feishu"],
        "config": {
          "app_id": "xxx",
          "app_secret": "xxx"
        }
      }
    ]
  }
}
```

### 方案 B：开发飞书 MCP Server
如果不存在，可创建一个独立的飞书 MCP Server：
```bash
feishu-mcp-server/
├── package.json      # 定义入口和依赖
├── index.js         # MCP Server 实现
└── README.md        # 使用说明
```

然后在 gclaw 中配置使用该 Server。

## 下一步

1. 实现 MCP Server 管理器（启动/停止/监控多个 Server）
2. 添加配置热重载支持
3. 实现工具缓存和懒加载
4. 开发飞书专用 MCP Server（如需要）

## 参考资源

- MCP 官方文档：https://modelcontextprotocol.io
- MCP Servers 仓库：https://github.com/modelcontextprotocol/servers
- MCP TypeScript SDK：https://github.com/modelcontextprotocol/typescript-sdk
