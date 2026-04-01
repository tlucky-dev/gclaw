# ModelScope 模型接入指南

## 概述

GCLaw 现已支持阿里云魔搭社区（ModelScope）的模型接入，您可以使用通义千问等国产大模型。

## 快速开始

### 1. 获取 API Key

访问 [阿里云 DashScope](https://dashscope.console.aliyun.com/) 注册账号并获取 API Key。

### 2. 配置文件

创建 `config.json` 文件：

```json
{
  "provider": {
    "name": "modelscope",
    "api_key": "your-dashscope-api-key",
    "base_url": "https://dashscope.aliyuncs.com/api/v1",
    "model": "qwen-turbo",
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
    "level": "standard"
  }
}
```

### 3. 运行

```bash
./gclaw -config config.json
```

## 支持的模型

| 模型名称 | 描述 | 推荐场景 |
|---------|------|---------|
| qwen-turbo | 通义千问 Turbo | 通用对话、文本生成 |
| qwen-plus | 通义千问 Plus | 复杂任务、推理 |
| qwen-max | 通义千问 Max | 高难度任务 |
| qwen-long | 通义千问 Long | 长文本处理 |

## 配置说明

### Provider 配置

- **name**: 设置为 `modelscope`
- **api_key**: 您的 DashScope API Key
- **base_url**: ModelScope API 地址（默认：`https://dashscope.aliyuncs.com/api/v1`）
- **model**: 要使用的模型名称
- **timeout**: 请求超时时间（秒）

### 沙箱配置

完整的沙箱配置请参考 `config.example.json`，主要选项包括：

- **enabled**: 是否启用沙箱
- **level**: 隔离级别（none/basic/standard/strict）
- **shell_whitelist**: Shell 命令白名单
- **shell_blacklist**: Shell 命令黑名单
- **network_disabled**: 是否禁用网络访问
- **audit_log_enabled**: 是否启用审计日志

## 示例配置文件

项目提供了两个示例配置文件：

- `config.minimal.json`: 最小化配置（OpenAI）
- `config.modelscope.json`: ModelScope 专用配置

## 注意事项

1. **API Key 安全**: 请勿将 API Key 提交到版本控制系统
2. **费用**: ModelScope 部分模型可能产生费用，请查阅官方定价
3. **速率限制**: 注意 API 调用频率限制
4. **网络**: 确保服务器可以访问 `dashscope.aliyuncs.com`

## 故障排查

### 常见问题

1. **认证失败**: 检查 API Key 是否正确
2. **模型不存在**: 确认模型名称拼写正确
3. **超时**: 增加 timeout 值或检查网络连接
4. **配额不足**: 登录控制台查看剩余配额

### 日志位置

审计日志默认位于：`/var/log/gclaw/sandbox-audit.log`

## 更多资源

- [ModelScope 官方文档](https://help.aliyun.com/zh/dashscope/)
- [通义千问模型介绍](https://www.aliyun.com/product/tongyi)
- [GCLaw 沙箱增强文档](./SANDBOX_ENHANCEMENT.md)
