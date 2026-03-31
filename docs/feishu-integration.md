# 飞书集成模块

gclaw 提供了完整的飞书集成能力，无需重复造轮子创建插件市场。通过复用飞书开放平台现有的应用市场机制，gclaw 可以作为飞书机器人直接接入。

## 架构设计

采用"核心 + 适配器 + 工具集"架构：

```
┌─────────────────┐
│   gclaw Core    │
│   (AI Engine)   │
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼──────────┐
│ Feishu│ │ Feishu      │
│Adapter│ │ Tools       │
│(双向) │ │ (主动调用)  │
└───┬───┘ └──┬──────────┘
    │        │
    │        ▼
    │   ┌─────────────┐
    │   │ 飞书 API    │
    │   │ - 发消息    │
    │   │ - 获取用户  │
    │   │ - 创建群聊  │
    │   └─────────────┘
    │
    ▼
┌─────────────┐
│ 飞书机器人  │
│ (Webhook)   │
└─────────────┘
```

## 功能特性

### 1. 飞书适配器（Feishu Adapter）
- **位置**: `internal/adapters/feishu/`
- **功能**: 
  - 接收飞书 Webhook 消息
  - 自动处理 URL 验证
  - 解密飞书加密消息
  - 发送回复消息
- **使用方式**: 作为飞书机器人在飞书客户端中使用

### 2. 飞书工具集（Feishu Tools）
- **位置**: `internal/tools/feishu.go`
- **功能**:
  - `send_message`: 发送消息到指定用户/群组
  - `get_user_info`: 获取用户详细信息
  - `create_chat`: 创建群聊
- **使用方式**: AI Agent 主动调用飞书 API

## 配置方法

### 环境变量方式

```bash
export FEISHU_APP_ID="cli_xxxxxxxxxxxxx"
export FEISHU_APP_SECRET="xxxxxxxxxxxxxxxxxxxxx"
```

### 配置文件方式

创建 `config.json`:

```json
{
  "provider": {
    "name": "openai",
    "api_key": "sk-xxx",
    "model": "gpt-4"
  },
  "adapters": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxxx",
      "encrypt_key": "xxxxxxxxxxxxxxxxxxxxx",
      "verification_token": "xxxxxxxxxxxxxxxxxxxxx",
      "server_addr": ":8080"
    }
  }
}
```

## 使用方式

### 方式一：作为飞书机器人（推荐）

1. **在飞书开放平台创建应用**
   - 访问 https://open.feishu.cn/
   - 创建企业自建应用
   - 获取 App ID 和 App Secret
   - 配置事件订阅（URL 指向 gclaw 服务器）
   - 开通必要的 API 权限

2. **启动 gclaw 飞书模式**

```bash
./gclaw -config config.json -feishu
```

3. **在飞书中与机器人对话**
   - 将机器人添加到群聊或私聊
   - 发送消息，AI 会自动回复

### 方式二：AI 主动调用飞书 API

在交互式模式下，AI 可以主动使用飞书工具：

```bash
./gclaw -config config.json -i
```

示例对话：
```
> 给张三发个消息说下午三点开会
[AI 调用 feishu tool -> send_message]
Message sent successfully, message_id: om_xxxxxxxxxxxxx

> 查询李四的手机号
[AI 调用 feishu tool -> get_user_info]
{
  "user_id": "ou_xxxxx",
  "name": "李四",
  "mobile": "13800138000"
}
```

## API 说明

### FeishuTool 参数

```json
{
  "action": "send_message|get_user_info|create_chat",
  "params": {
    // 根据 action 不同而不同
  }
}
```

#### send_message
```json
{
  "action": "send_message",
  "params": {
    "receive_id": "ou_xxxxx",
    "msg_type": "text",
    "content": {"text": "Hello"}
  }
}
```

#### get_user_info
```json
{
  "action": "get_user_info",
  "params": {
    "user_id": "ou_xxxxx"
  }
}
```

#### create_chat
```json
{
  "action": "create_chat",
  "params": {
    "name": "项目讨论组",
    "chat_type": "group"
  }
}
```

## 安全建议

1. **权限最小化**: 只申请必要的 API 权限
2. **IP 白名单**: 在飞书开放平台配置服务器 IP 白名单
3. **密钥管理**: 不要将 App Secret 提交到代码仓库
4. **消息验证**: 始终验证飞书请求签名
5. **速率限制**: 注意飞书 API 调用频率限制

## 扩展开发

如需添加更多飞书功能，可在 `internal/tools/feishu.go` 中扩展：

```go
// 示例：添加发送日历事件功能
func (t *FeishuTool) createCalendarEvent(...) {
    // 调用飞书日历 API
}
```

## 参考资源

- [飞书开放平台](https://open.feishu.cn/)
- [飞书 API 文档](https://open.feishu.cn/document/ukTMzNjL4YDMzNSM2UDNwzjyLzjN)
- [OpenClaw 项目](https://github.com/openclaw/openclaw)

## 总结

gclaw 的飞书集成模块遵循"复用生态"原则：
- ✅ 利用飞书现有的应用市场和管理后台
- ✅ 通过标准 API 协议集成，无需自建插件系统
- ✅ 提供双向通信能力（适配器接收 + 工具主动调用）
- ✅ 开箱即用，配置简单

这种设计避免了重复造轮子，让 gclaw 能够快速融入飞书生态系统。
