# 飞书集成指南

## 概述

gclaw 提供了飞书（Feishu/Lark）集成模块，允许您通过飞书机器人与 gclaw AI 助手进行交互。

## 功能特性

- ✅ 接收飞书消息并自动回复
- ✅ 支持群聊和私聊
- ✅ 自动管理访问令牌（带缓存和刷新）
- ✅ URL 验证挑战自动处理
- ✅ 事件订阅 token 验证
- ✅ 优雅关闭和信号处理

## 配置步骤

### 1. 在飞书开放平台创建应用

1. 访问 [飞书开放平台](https://open.feishu.cn/)
2. 创建企业自建应用
3. 获取 `App ID` 和 `App Secret`

### 2. 配置应用权限

在应用管理后台添加以下权限：
- `发送消息` (im:message)
- `读取用户信息` (contact:user:readonly)

### 3. 配置事件订阅

1. 进入「事件订阅」页面
2. 设置请求网址为：`https://your-domain.com/webhook/feishu`
3. 复制 `Verification Token`
4. 订阅以下事件：
   - `接收消息 v1.0` (im.message.receive_v1)

### 4. 配置 gclaw

编辑配置文件（如 `config.json`）：

```json
{
  "provider": {
    "name": "openai",
    "api_key": "your-openai-api-key",
    "model": "gpt-3.5-turbo"
  },
  "adapters": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxxxxxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "encrypt_key": "",
      "verification_token": "your-verification-token",
      "server_addr": ":8080"
    }
  }
}
```

### 5. 启动 gclaw

#### 方式一：使用配置文件

```bash
./gclaw -config config.json
```

#### 方式二：使用命令行参数

```bash
./gclaw --api-key your-api-key --feishu
```

注意：使用命令行方式时，仍需要在配置文件中设置飞书相关参数。

## 部署说明

### 本地开发

```bash
# 启动服务
./gclaw -config config.json

# 服务将监听在配置的 server_addr（默认 :8080）
# Webhook 端点：http://localhost:8080/webhook/feishu
```

### 生产环境

1. **使用 HTTPS**：飞书要求 webhook 必须使用 HTTPS
2. **配置反向代理**：使用 Nginx 或 Caddy 作为反向代理
3. **域名备案**：确保域名可被飞书访问

#### Nginx 配置示例

```nginx
server {
    listen 443 ssl;
    server_name your-domain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location /webhook/feishu {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
}
```

### Docker 部署

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gclaw ./cmd

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gclaw .
COPY config.json .
EXPOSE 8080
CMD ["./gclaw", "-config", "config.json"]
```

## 使用示例

### 私聊机器人

直接在飞书中向机器人发送消息，机器人会自动回复。

### 群聊机器人

1. 将机器人添加到群组
2. 在群组中 @机器人 或直接发送消息
3. 机器人会自动回复

## API 参考

### 配置结构

```go
type FeishuConfig struct {
    Enabled           bool   `json:"enabled"`            // 是否启用
    AppID             string `json:"app_id"`             // 飞书 App ID
    AppSecret         string `json:"app_secret"`         // 飞书 App Secret
    EncryptKey        string `json:"encrypt_key"`        // 可选，消息加密密钥
    VerificationToken string `json:"verification_token"` // 可选，事件订阅验证 token
    ServerAddr        string `json:"server_addr"`        // HTTP 服务器监听地址
}
```

### 环境变量（可选）

也可以通过环境变量配置：

```bash
export FEISHU_APP_ID="cli_xxx"
export FEISHU_APP_SECRET="xxx"
export FEISHU_VERIFICATION_TOKEN="xxx"
export FEISHU_SERVER_ADDR=":8080"
```

## 故障排查

### 常见问题

1. **收不到消息**
   - 检查飞书开放平台的事件订阅配置
   - 确认 webhook URL 可公网访问
   - 查看日志确认是否有错误

2. **发送消息失败**
   - 检查 App 权限是否正确配置
   - 确认 access token 是否有效
   - 查看飞书 API 返回的错误码

3. **URL 验证失败**
   - 确保正确响应 challenge
   - 检查 verification token 配置

### 日志查看

gclaw 会在控制台输出详细日志，包括：
- HTTP 服务器启动信息
- 消息接收和发送状态
- 错误信息

## 安全建议

1. **保护 App Secret**：不要将 App Secret 提交到版本控制系统
2. **使用 HTTPS**：生产环境必须使用 HTTPS
3. **限制访问**：配置防火墙只允许飞书 IP 访问 webhook 端点
4. **定期更新凭证**：定期更换 App Secret 和 verification token

## 开发计划

- [ ] 支持富文本消息
- [ ] 支持卡片消息
- [ ] 支持文件上传下载
- [ ] 支持更多飞书事件类型
- [ ] 消息队列持久化
- [ ] 多租户支持

## 贡献

欢迎提交 Issue 和 Pull Request！
