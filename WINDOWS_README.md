# Windows 版本使用说明

## 已构建的 Windows 可执行文件

成功构建了三个 Windows 架构版本：

| 文件名 | 架构 | 大小 | 适用系统 |
|--------|------|------|----------|
| `gclaw_windows_amd64.exe` | 64位 Intel/AMD | 8.3MB | Windows 10/11 64位 |
| `gclaw_windows_386.exe` | 32位 | 8.3MB | Windows 7/8/10/11 32位 |
| `gclaw_windows_arm64.exe` | ARM64 | 7.8MB | Surface Pro X, Windows on ARM |

## 快速开始

### 1. 下载并复制文件

将对应的 `.exe` 文件和配置文件复制到 Windows 系统：

```powershell
# 推荐大多数用户使用 amd64 版本
gclaw_windows_amd64.exe
config.minimal.json
config.example.json
config.modelscope.json
```

### 2. 重命名可执行文件（可选）

```powershell
# 重命名为简洁的名称
Copy-Item gclaw_windows_amd64.exe gclaw.exe
```

### 3. 准备配置文件

选择并配置以下任一配置文件：

**使用 OpenAI：**
```powershell
# 复制最小配置
Copy-Item config.minimal.json config.json
# 编辑 config.json，填入你的 API Key
notepad config.json
```

**使用 ModelScope（通义千问）：**
```powershell
# 复制 ModelScope 配置
Copy-Item config.modelscope.json config.json
# 编辑 config.json，填入你的 DashScope API Key
notepad config.json
```

### 4. 运行程序

```powershell
# 使用配置文件运行
.\gclaw.exe -config config.json

# 或使用命令行参数
.\gclaw.exe -provider openai -api-key YOUR_API_KEY -model gpt-3.5-turbo

# 使用 ModelScope
.\gclaw.exe -provider modelscope -api-key YOUR_DASHSCOPE_KEY -model qwen-turbo
```

### 5. 查看帮助

```powershell
.\gclaw.exe --help
```

## 配置文件示例

### 最小配置 (config.json)

```json
{
  "provider": "openai",
  "api_key": "sk-your-api-key-here",
  "model": "gpt-3.5-turbo",
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "root_dir": "./sandbox",
    "shell_whitelist": ["ls", "cat", "pwd", "echo"],
    "max_file_size_mb": 10,
    "session_isolation": true,
    "auto_reset": true
  }
}
```

### ModelScope 配置 (config.json)

```json
{
  "provider": "modelscope",
  "api_key": "your-dashscope-api-key",
  "model": "qwen-turbo",
  "api_url": "https://dashscope.aliyuncs.com/api/v1",
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "root_dir": "./sandbox",
    "shell_whitelist": ["ls", "cat", "pwd", "echo"],
    "max_file_size_mb": 10,
    "session_isolation": true,
    "auto_reset": true
  }
}
```

## Windows 特定说明

### 沙箱功能限制

在 Windows 平台上，部分 Linux 特有的沙箱功能会自动降级为兼容模式：

- ✅ **命令白名单/黑名单**: 完全支持
- ✅ **文件路径隔离**: 完全支持
- ✅ **会话隔离**: 完全支持
- ✅ **文件大小限制**: 完全支持
- ⚠️ **Linux Namespace**: 不可用（自动切换到兼容模式）
- ⚠️ **syscall 级别隔离**: 不可用（使用 Windows 原生机制）

### 推荐的沙箱配置

Windows 用户建议使用以下配置：

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "cross_platform_compat": true,
    "root_dir": "./sandbox",
    "shell_whitelist": ["dir", "type", "cd", "echo", "copy", "del"],
    "shell_blacklist": ["format", "diskpart", "reg", "netsh"],
    "max_execution_time_sec": 30,
    "max_file_size_mb": 10,
    "session_isolation": true,
    "auto_reset": true,
    "audit_log_enabled": true,
    "audit_log_level": "info"
  }
}
```

### 防火墙设置

如果沙箱启用网络访问（不推荐），可能需要配置 Windows 防火墙：

```powershell
# 允许程序访问网络（仅当需要时）
New-NetFirewallRule -DisplayName "GCLaw" -Direction Outbound -Program ".\gclaw.exe" -Action Allow
```

## 常见问题

### Q: 运行时提示"找不到指定的模块"
A: 本程序是静态编译的 Go 程序，不依赖外部 DLL。确保下载的 .exe 文件完整。

### Q: 杀毒软件报毒
A: 这是误报。Go 编译的程序有时会被误判。可以将程序添加到杀毒软件白名单，或从源代码自行编译。

### Q: 沙箱目录在哪里？
A: 默认在当前目录下的 `./sandbox` 文件夹。可以在配置文件中修改 `sandbox.root_dir`。

### Q: 如何查看审计日志？
A: 日志文件位于 `./logs/sandbox_audit.log`（可在配置中修改路径）。

### Q: 如何在后台运行？
A: 使用 PowerShell：
```powershell
Start-Process -FilePath ".\gclaw.exe" -ArgumentList "-config config.json" -WindowStyle Hidden
```

## 支持的模型

### OpenAI
- gpt-4, gpt-4-turbo, gpt-4o
- gpt-3.5-turbo
- gpt-4-vision-preview

### ModelScope (通义千问)
- qwen-turbo (最快)
- qwen-plus (平衡)
- qwen-max (最强)
- qwen-long (长文本)
- qwen-vl-max (视觉)

## 卸载

直接删除 `.exe` 文件和配置文件即可：

```powershell
Remove-Item gclaw.exe
Remove-Item config.json
Remove-Item -Recurse sandbox
Remove-Item -Recurse logs
```

## 技术支持

如遇问题，请检查：
1. 配置文件格式是否正确（JSON）
2. API Key 是否有效
3. 网络连接是否正常
4. 沙箱目录是否有读写权限

查看详细日志：
```powershell
.\gclaw.exe -config config.json -verbose
```
