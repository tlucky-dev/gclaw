# GCLaw 命令行使用指南

本指南详细介绍 GCLaw 的所有命令行功能、配置流程和最佳实践。

## 目录

- [快速开始](#快速开始)
- [命令概览](#命令概览)
- [详细命令说明](#详细命令说明)
- [配置向导](#配置向导)
- [运行模式](#运行模式)
- [沙箱控制](#沙箱控制)
- [高级用法](#高级用法)
- [故障排查](#故障排查)

---

## 快速开始

### 1. 初始化配置（推荐新手）

```bash
# 交互式创建 OpenAI 配置
./gclaw -init openai

# 交互式创建 ModelScope 配置
./gclaw -init modelscope

# 交互式创建 Azure 配置
./gclaw -init azure
```

### 2. 运行 GCLaw

```bash
# 交互模式
./gclaw -i -config config.openai.json

# 单次执行模式
echo "查询当前日期" | ./gclaw -config config.openai.json

# 使用命令行参数覆盖配置
./gclaw -config config.json -model gpt-4 -sandbox-level strict
```

---

## 命令概览

| 命令/参数 | 说明 | 示例 |
|-----------|------|------|
| `-version` | 显示版本信息 | `./gclaw -version` |
| `-init` | 初始化配置文件 | `./gclaw -init openai` |
| `-config` | 指定配置文件路径 | `./gclaw -config config.json` |
| `-i` | 交互模式 | `./gclaw -i -config config.json` |
| `-api-key` | API Key（覆盖配置） | `./gclaw -api-key sk-xxx` |
| `-model` | 模型名称（覆盖配置） | `./gclaw -model gpt-4` |
| `-provider` | 提供商名称 | `./gclaw -provider modelscope` |
| `-sandbox-level` | 沙箱隔离级别 | `./gclaw -sandbox-level strict` |
| `-sandbox-dryrun` | 沙箱干跑模式 | `./gclaw -sandbox-dryrun` |
| `-max-iterations` | 最大迭代次数 | `./gclaw -max-iterations 5` |
| `-temperature` | 温度参数 | `./gclaw -temperature 0.8` |
| `-max-tokens` | 最大 Token 数 | `./gclaw -max-tokens 4096` |
| `-session` | 会话 ID | `./gclaw -session my-session` |

---

## 详细命令说明

### 1. 版本信息 (`-version`)

显示 GCLaw 的版本号和简要帮助信息。

```bash
$ ./gclaw -version

gclaw version 1.2.0
A secure AI agent framework with enhanced sandbox capabilities

Usage:
  gclaw [options]

Quick Start:
  gclaw -init openai              # Initialize OpenAI config
  gclaw -i -config config.json    # Run in interactive mode

Options:
  [所有可用参数列表]
```

**使用场景：**
- 确认安装的版本
- 快速查看可用命令

---

### 2. 配置初始化 (`-init`)

**最重要的命令！** 通过交互式向导创建配置文件。

#### 语法
```bash
./gclaw -init <provider_type>
```

#### 支持的提供商类型
- `openai` - OpenAI / 兼容 OpenAI API 的服务
- `modelscope` - ModelScope / DashScope（通义千问）
- `azure` - Azure OpenAI Service

#### 完整示例

```bash
$ ./gclaw -init openai

=== GCLaw Configuration Wizard (openai) ===

Default model: gpt-3.5-turbo

Enter your API Key: sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
Enter model name [gpt-3.5-turbo]: gpt-4
Enter base URL [https://api.openai.com/v1]: 

--- Sandbox Configuration ---
Isolation levels: none | basic | standard | strict
Enter sandbox level [standard]: standard
Enable dry-run mode? (y/N): n

--- Engine Configuration ---
Enter max iterations [10]: 
Enter temperature [0.70]: 

Save to file [config.openai.json]: 

✓ Configuration saved to: config.openai.json

To run gclaw with this config:
  gclaw -i -config config.openai.json
```

#### ModelScope 配置示例

```bash
$ ./gclaw -init modelscope

=== GCLaw Configuration Wizard (modelscope) ===

Default model: qwen-turbo

Enter your API Key: sk-xxxxxx
Enter model name [qwen-turbo]: qwen-max
Enter base URL [https://dashscope.aliyuncs.com/compatible-mode/v1]: 

--- Sandbox Configuration ---
Enter sandbox level [standard]: strict
Enable dry-run mode? (y/N): y

✓ Configuration saved to: config.modelscope.json
```

**生成的配置文件内容：**

```json
{
  "provider": {
    "name": "modelscope",
    "api_key": "sk-xxxxxx",
    "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model": "qwen-max",
    "timeout": 30
  },
  "memory": {
    "type": "memory",
    "max_size": 100
  },
  "engine": {
    "max_iterations": 10,
    "temperature": 0.7,
    "max_tokens": 2048
  },
  "sandbox": {
    "enabled": true,
    "level": "strict",
    "dry_run": true,
    "timeout_seconds": 30,
    "max_memory_mb": 512,
    "shell_whitelist": ["ls", "cat", "echo", "pwd", "whoami", "date", "uname"],
    "shell_blacklist": ["rm", "sudo", "su", "chmod", "chown", "mount", "umount", "dd"],
    "audit_log_enabled": true,
    "audit_log_level": "info",
    "anomaly_detection": true
  }
}
```

---

### 3. 交互模式 (`-i`)

启动交互式 REPL 环境，可以连续对话。

```bash
./gclaw -i -config config.openai.json
```

**运行效果：**

```
$ ./gclaw -i -config config.openai.json

gclaw interactive mode. Type 'exit' or 'quit' to end.
Type 'reset' to clear conversation history.

> 今天天气怎么样？
[AI 回答...]

> 帮我创建一个测试文件
[AI 执行工具调用并创建文件...]

> reset
Session reset.

> exit
```

**特殊命令：**
- `exit` / `quit` - 退出程序
- `reset` - 清空当前会话历史

---

### 4. 单次执行模式（默认）

从标准输入读取单行指令，执行后退出。

```bash
# 方式 1：管道输入
echo "列出当前目录文件" | ./gclaw -config config.json

# 方式 2：here-string
./gclaw -config config.json <<< "查询系统信息"

# 方式 3：重定向
echo "创建文件 test.txt" > input.txt
./gclaw -config config.json < input.txt
```

**适用场景：**
- Shell 脚本集成
- 自动化任务
- CI/CD 流程

---

## 配置向导

### 配置项详解

#### 1. 提供商配置 (Provider)

| 字段 | 说明 | 默认值 | 示例 |
|------|------|--------|------|
| `name` | 提供商名称 | `openai` | `openai`, `modelscope`, `azure` |
| `api_key` | API 密钥 | 必填 | `sk-xxx` |
| `base_url` | API 基础 URL | 根据 provider | `https://api.openai.com/v1` |
| `model` | 模型名称 | `gpt-3.5-turbo` | `gpt-4`, `qwen-turbo` |
| `timeout` | 超时时间 (秒) | `30` | `60` |

#### 2. 沙箱配置 (Sandbox)

| 字段 | 说明 | 默认值 | 可选值 |
|------|------|--------|--------|
| `enabled` | 是否启用沙箱 | `true` | `true/false` |
| `level` | 隔离级别 | `standard` | `none/basic/standard/strict` |
| `dry_run` | 干跑模式 | `false` | `true/false` |
| `timeout_seconds` | 超时时间 | `30` | 整数 |
| `max_memory_mb` | 最大内存 (MB) | `512` | 整数 |
| `shell_whitelist` | Shell 白名单 | 基础命令 | 字符串数组 |
| `shell_blacklist` | Shell 黑名单 | 危险命令 | 字符串数组 |
| `audit_log_enabled` | 审计日志 | `true` | `true/false` |
| `anomaly_detection` | 异常检测 | `true` | `true/false` |

#### 3. 引擎配置 (Engine)

| 字段 | 说明 | 默认值 | 建议值 |
|------|------|--------|--------|
| `max_iterations` | 最大迭代次数 | `10` | `5-15` |
| `temperature` | 温度参数 | `0.7` | `0.5-0.9` |
| `max_tokens` | 最大 Token 数 | `2048` | `1024-4096` |

---

## 运行模式

### 模式对比

| 模式 | 命令 | 适用场景 | 优点 |
|------|------|----------|------|
| **交互模式** | `-i` | 开发调试、探索性使用 | 连续对话、上下文保持 |
| **单次模式** | 默认 | 脚本集成、自动化 | 简单直接、易集成 |

### 模式切换示例

```bash
# 从交互模式开始
./gclaw -i -config config.json

# 切换到单次执行（新终端）
echo "新任务" | ./gclaw -config config.json -session session-123
```

---

## 沙箱控制

### 隔离级别说明

| 级别 | 说明 | 适用场景 |
|------|------|----------|
| `none` | 无沙箱 | 完全可信环境、性能敏感场景 |
| `basic` | 基础隔离 | 内部开发环境、受信任用户 |
| `standard` | 标准隔离 | 生产环境、普通用户（推荐） |
| `strict` | 严格隔离 | 公开服务、不可信用户 |

### 命令行覆盖沙箱配置

```bash
# 临时启用严格模式
./gclaw -config config.json -sandbox-level strict

# 启用干跑模式（只模拟不执行）
./gclaw -config config.json -sandbox-dryrun

# 组合使用
./gclaw -config config.json -sandbox-level strict -sandbox-dryrun
```

### 沙箱级别与用户角色映射

在配置文件中设置：

```json
{
  "sandbox": {
    "user_level_mapping": {
      "admin": "standard",
      "user": "standard",
      "guest": "strict",
      "trusted": "basic"
    }
  }
}
```

---

## 高级用法

### 1. 多环境配置

```bash
# 开发环境
./gclaw -init openai
mv config.openai.json config.dev.json
# 编辑 config.dev.json，设置 sandbox-level=basic

# 生产环境
cp config.dev.json config.prod.json
# 编辑 config.prod.json，设置 sandbox-level=strict

# 切换环境
./gclaw -i -config config.dev.json
./gclaw -i -config config.prod.json
```

### 2. 模型热切换

```bash
# 使用默认模型
./gclaw -config config.json <<< "你好"

# 临时切换到 GPT-4
./gclaw -config config.json -model gpt-4 <<< "复杂任务"

# 临时切换到通义千问
./gclaw -config config.json -provider modelscope -model qwen-max <<< "中文任务"
```

### 3. 脚本集成

```bash
#!/bin/bash
# auto-task.sh

TASK="$1"

# 检查参数
if [ -z "$TASK" ]; then
    echo "Usage: $0 <task>"
    exit 1
fi

# 执行任务（严格沙箱模式）
RESULT=$(echo "$TASK" | ./gclaw \
    -config config.prod.json \
    -sandbox-level strict \
    -max-iterations 5 \
    -temperature 0.5)

echo "$RESULT"

# 记录日志
echo "[$(date)] $TASK -> $RESULT" >> task.log
```

### 4. Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gclaw ./cmd/main.go

FROM alpine:latest
COPY --from=builder /app/gclaw /gclaw
COPY config.json /config.json
ENTRYPOINT ["/gclaw", "-config", "/config.json"]
```

```bash
docker build -t gclaw .
docker run -d gclaw
```

---

## 故障排查

### 常见问题

#### 1. API Key 错误

```
Error: API key is required. Use --api-key or set in config file.
Tip: Run 'gclaw -init openai' to create a config file interactively.
```

**解决方案：**
```bash
# 重新初始化配置
./gclaw -init openai

# 或命令行指定
./gclaw -api-key sk-xxx -config config.json
```

#### 2. 配置文件不存在

```
Error loading config file: open config.json: no such file or directory
```

**解决方案：**
```bash
# 检查文件是否存在
ls -la config*.json

# 使用完整路径
./gclaw -config /path/to/config.json
```

#### 3. 沙箱执行失败

```
Sandbox execution failed: command not in whitelist
```

**解决方案：**
```bash
# 降低沙箱级别
./gclaw -config config.json -sandbox-level basic

# 或启用干跑模式调试
./gclaw -config config.json -sandbox-dryrun

# 或编辑配置文件添加白名单命令
```

#### 4. ModelScope 连接失败

```
Error creating provider: invalid base URL
```

**解决方案：**
```bash
# 检查 BaseURL 配置
# ModelScope 默认地址：
# https://dashscope.aliyuncs.com/compatible-mode/v1

# 重新初始化 ModelScope 配置
./gclaw -init modelscope
```

### 调试技巧

```bash
# 1. 查看详细帮助
./gclaw -version

# 2. 验证配置文件
cat config.json | python -m json.tool

# 3. 测试单次执行
echo "test" | ./gclaw -config config.json

# 4. 检查沙箱状态
./gclaw -config config.json -sandbox-level none <<< "echo debug"
```

---

## 附录：完整配置示例

### OpenAI 完整配置

```json
{
  "provider": {
    "name": "openai",
    "api_key": "sk-your-api-key",
    "base_url": "https://api.openai.com/v1",
    "model": "gpt-4",
    "timeout": 60
  },
  "memory": {
    "type": "memory",
    "max_size": 200
  },
  "engine": {
    "max_iterations": 15,
    "temperature": 0.7,
    "max_tokens": 4096
  },
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "timeout_seconds": 60,
    "max_memory_mb": 1024,
    "max_cpu_percent": 80.0,
    "max_file_size_mb": 50,
    "shell_whitelist": ["ls", "cat", "echo", "pwd", "whoami", "date", "uname", "grep", "find"],
    "shell_blacklist": ["rm", "sudo", "su", "chmod", "chown", "mount", "umount", "dd", "mkfs"],
    "shell_arg_patterns": ["[;&|]", "\\$\\(", "`", ">/dev/tcp/"],
    "file_sandbox_root": "/tmp/gclaw-sandbox",
    "file_reset_on_start": true,
    "file_allowed_paths": ["/tmp", "/var/tmp"],
    "file_blocked_paths": ["/etc", "/root", "/proc", "/sys", "/dev"],
    "network_disabled": true,
    "audit_log_enabled": true,
    "audit_log_level": "debug",
    "audit_log_path": "/var/log/gclaw/sandbox-audit.log",
    "anomaly_detection": true,
    "max_exec_per_minute": 30,
    "security_integration": true,
    "dry_run": false,
    "verbose_logging": true
  }
}
```

### ModelScope 完整配置

```json
{
  "provider": {
    "name": "modelscope",
    "api_key": "sk-dashscope-key",
    "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
    "model": "qwen-max",
    "timeout": 60
  },
  "memory": {
    "type": "memory",
    "max_size": 100
  },
  "engine": {
    "max_iterations": 10,
    "temperature": 0.7,
    "max_tokens": 2048
  },
  "sandbox": {
    "enabled": true,
    "level": "strict",
    "timeout_seconds": 30,
    "max_memory_mb": 512,
    "shell_whitelist": ["ls", "cat", "echo", "pwd"],
    "shell_blacklist": ["rm", "sudo", "curl", "wget"],
    "audit_log_enabled": true,
    "audit_log_level": "info",
    "anomaly_detection": true,
    "dry_run": false
  }
}
```

---

## 更新日志

### v1.2.0
- ✅ 新增交互式配置向导 (`-init`)
- ✅ 支持 ModelScope 模型接入
- ✅ 增强沙箱控制参数
- ✅ 完善命令行文档

### v1.1.0
- 增强沙箱隔离能力
- 添加审计日志系统
- 支持飞书适配器

### v1.0.0
- 初始版本发布
