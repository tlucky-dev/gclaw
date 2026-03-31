# gclaw

gclaw 是一个用 Go 语言实现的 AI Agent 框架，灵感来源于 [openclaw](https://github.com/openclaw/openclaw) 项目。

## 功能特性

- 🤖 支持多个 LLM 提供商（OpenAI、Anthropic 等）
- 🔧 内置工具系统（Shell 命令、文件读写、搜索等）
- 💾 会话记忆管理
- 🔄 自动工具调用和迭代执行
- ⚙️ 灵活的配置系统
- 💻 交互式和单次运行模式
- 📱 **飞书集成**（消息机器人）

## 项目结构

```
gclaw/
├── cmd/                  # 命令行入口
│   └── main.go
├── internal/             # 内部实现
│   ├── adapters/         # 外部平台适配器
│   │   └── feishu/       # 飞书适配器
│   ├── config/           # 配置管理
│   ├── engine/           # 核心引擎
│   ├── memory/           # 记忆存储
│   ├── provider/         # LLM 提供商
│   └── tools/            # 工具系统
├── pkg/                  # 公共包
│   ├── errors/           # 错误定义
│   └── types/            # 类型定义
├── go.mod
├── go.sum
├── config.example.json   # 配置示例
└── FEISHU_ADAPTER.md     # 飞书集成文档
```

## 快速开始

### 安装

```bash
git clone <repository-url>
cd gclaw
go build -o gclaw ./cmd/...
```

### 使用方法

#### 1. 使用配置文件

创建配置文件 `config.json`：

```json
{
  "provider": {
    "name": "openai",
    "api_key": "your-api-key",
    "model": "gpt-3.5-turbo"
  },
  "memory": {
    "type": "memory",
    "max_size": 100
  },
  "engine": {
    "max_iterations": 10,
    "temperature": 0.7,
    "max_tokens": 2048
  }
}
```

运行：

```bash
./gclaw -config config.json -i
```

#### 2. 使用命令行参数

```bash
export OPENAI_API_KEY="your-api-key"
./gclaw -api-key $OPENAI_API_KEY -model gpt-4 -i
```

#### 3. 单次运行模式

```bash
echo "列出当前目录的文件" | ./gclaw -api-key $OPENAI_API_KEY
```

### 命令行选项

| 选项 | 说明 | 默认值 |
|------|------|--------|
| `-config` | 配置文件路径 | - |
| `-api-key` | LLM API Key | - |
| `-model` | 模型名称 | gpt-3.5-turbo |
| `-provider` | 提供商名称 | openai |
| `-session` | 会话 ID | default |
| `-i` | 交互式模式 | false |
| `-feishu` | 启用飞书适配器 | false |

## 飞书集成

gclaw 支持飞书（Feishu/Lark）机器人集成，可以通过飞书接收和发送消息。

详细配置和使用说明请查看 [飞书集成指南](FEISHU_ADAPTER.md)。

快速启动飞书机器人：

```bash
# 1. 编辑 config.json，配置飞书参数
# 2. 启动服务
./gclaw -config config.json

# 或者使用命令行参数
./gclaw -config config.json -feishu
```

服务启动后，将在配置的地址（默认 `:8080`）监听飞书 webhook 请求。

## 内置工具

### Shell 工具
执行 shell 命令

### 文件读取工具
读取文件内容

### 文件写入工具
写入内容到文件

### 搜索工具
搜索网络或知识库

## 扩展工具

可以通过实现 `ToolExecutor` 接口来添加自定义工具：

```go
type ToolExecutor interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(args map[string]interface{}) (string, error)
}
```

示例：

```go
type MyCustomTool struct {
    tools.BaseTool
}

func (t *MyCustomTool) Execute(args map[string]interface{}) (string, error) {
    // 实现你的逻辑
    return "result", nil
}
```

## 配置说明

### Provider 配置

- `name`: 提供商名称 (openai, anthropic)
- `api_key`: API 密钥
- `base_url`: API 基础 URL (可选)
- `model`: 模型名称
- `timeout`: 请求超时时间 (秒)

### Memory 配置

- `type`: 存储类型 (memory)
- `max_size`: 最大消息数
- `expiration`: 过期时间 (秒，可选)

### Engine 配置

- `max_iterations`: 最大迭代次数
- `temperature`: 温度参数
- `max_tokens`: 最大 token 数

## 开发

### 构建

```bash
go build -o gclaw ./cmd/...
```

### 测试

```bash
go test ./...
```

## License

MIT
