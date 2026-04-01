# 沙箱增强实现文档

## 概述

本文档描述了 GCLaw 项目沙箱能力的全面增强，解决了原有沙箱实现的以下问题：

1. **未产品化** - 无配置项、无文档说明、无开关/分级
2. **隔离强度不足** - Shell 沙箱局限性、无 Namespace 隔离、无网络隔离、文件沙箱缺陷
3. **协同缺失** - 无审计日志、无异常检测、与 security 模块协同弱
4. **跨平台兼容性差** - 高度依赖 Linux 系统调用

## 新增功能

### 1. 完整的产品化配置

#### 1.1 沙箱隔离级别

```go
type SandboxLevel string

const (
    SandboxLevelNone     SandboxLevel = "none"      // 无隔离（可信环境）
    SandboxLevelBasic    SandboxLevel = "basic"     // 基础隔离（命令白名单 + 路径限制）
    SandboxLevelStandard SandboxLevel = "standard"  // 标准隔离（Namespace + 资源限制）
    SandboxLevelStrict   SandboxLevel = "strict"    // 严格隔离（gVisor/容器级）
)
```

#### 1.2 可配置项

所有沙箱参数现在都可以通过 `config.json` 或命令行参数配置：

- **基础配置**: 启用/禁用、隔离级别、超时时间
- **资源限制**: 最大内存、CPU 使用率、文件大小、磁盘总用量
- **Shell 沙箱**: 命令白名单/黑名单、参数模式过滤
- **文件沙箱**: 根目录、自动重置、允许/阻止路径
- **网络控制**: 禁用网络、网络白名单
- **审计监控**: 审计日志、异常检测、执行频率限制
- **安全集成**: 用户级别映射、与 security 模块联动
- **高级选项**: gVisor、Linux Namespace、跨平台兼容模式

#### 1.3 配置示例

```json
{
  "sandbox": {
    "enabled": true,
    "level": "standard",
    "timeout_seconds": 30,
    "max_memory_mb": 512,
    "max_cpu_percent": 50.0,
    "shell_whitelist": ["ls", "cat", "echo", "pwd"],
    "shell_blacklist": ["rm", "sudo", "chmod", "mount"],
    "file_sandbox_root": "/tmp/gclaw-sandbox",
    "file_reset_on_start": true,
    "network_disabled": true,
    "audit_log_enabled": true,
    "anomaly_detection": true,
    "user_level_mapping": {
      "admin": "standard",
      "guest": "strict"
    }
  }
}
```

### 2. 增强的隔离强度

#### 2.1 Shell 沙箱增强

- **多层验证**: 白名单 + 黑名单 + 正则模式匹配
- **可疑命令检测**: 自动识别潜在的危险命令模式
  - `/etc/passwd`, `/etc/shadow` 访问
  - `base64 -d`, `eval()`, `python -c` 代码执行
  - `/dev/tcp/`, `nc -e` 反向连接
  - 管道注入、命令拼接等

#### 2.2 Linux Namespace 隔离

```go
// Linux 平台特定隔离
attr.Cloneflags = syscall.CLONE_NEWPID | syscall.CLONE_NEWNS | syscall.CLONE_NEWNET
attr.Credential = &syscall.Credential{Uid: 65534, Gid: 65534} // nobody user
```

- **PID Namespace**: 进程隔离，无法看到宿主进程
- **Mount Namespace**: 文件系统隔离
- **Network Namespace**: 网络隔离（可选）
- **User Namespace**: 降权运行（nobody 用户）

#### 2.3 网络隔离

- 默认禁用所有网络访问
- 清除网络相关环境变量（`http_proxy`, `https_proxy` 等）
- 可选的网络白名单（CIDR 或域名）

#### 2.4 文件沙箱增强

- **会话隔离**: 每个会话独立的沙箱目录
- **自动重置**: 启动时或会话结束后自动清理
- **路径验证**: 防止目录遍历攻击（`../`）
- **大小限制**: 单文件大小和总磁盘用量限制

### 3. 审计与监控

#### 3.1 审计日志系统

记录所有沙箱内操作：

```json
{
  "timestamp": "2024-01-01T12:00:00Z",
  "event_id": "abc123def456",
  "event_type": "execution",
  "action": "command_executed",
  "result": "success",
  "details": {
    "command": "ls",
    "duration_ms": 15,
    "output_length": 1024
  }
}
```

**日志级别**:
- `none`: 不记录
- `error`: 仅错误
- `warn`: 警告及以上
- `info`: 所有操作

#### 3.2 异常检测系统

- **执行频率限制**: 每分钟最大执行次数
- **可疑模式检测**: 自动识别恶意命令
- **安全告警**: 实时告警通知（支持回调）

**告警级别**: low, medium, high, critical

#### 3.3 与 Security 模块集成

- 不同用户角色对应不同沙箱隔离级别
- 敏感信息访问控制
- 输入验证与沙箱联动

### 4. 跨平台兼容

#### 4.1 平台适配

| 功能 | Linux | macOS | Windows |
|------|-------|-------|---------|
| Namespace 隔离 | ✅ 完整支持 | ⚠️ 有限支持 | ❌ 不支持 |
| 命令白名单 | ✅ | ✅ | ✅ |
| 路径限制 | ✅ | ✅ | ✅ |
| 网络隔离 | ✅ | ✅ | ✅ |
| 审计日志 | ✅ | ✅ | ✅ |
| 异常检测 | ✅ | ✅ | ✅ |

#### 4.2 兼容模式

```go
CrossPlatformCompat: true  // 启用跨平台兼容模式
```

在非 Linux 平台上自动降级为通用隔离机制。

### 5. 高级功能

#### 5.1 gVisor 集成

```go
UseGVisor: true  // 如果系统安装了 runsc
```

提供容器级别的强隔离（生产环境推荐）。

#### 5.2 干跑模式

```go
DryRun: true  // 只检查不执行
```

用于测试和调试沙箱规则。

#### 5.3 动态配置更新

支持运行时更新沙箱配置，无需重启服务。

## 使用指南

### 快速开始

1. **加载配置**:
```go
config, err := skill.LoadEnhancedSandboxConfigFromFile("sandbox-config.json")
if err != nil {
    log.Fatal(err)
}
```

2. **创建沙箱**:
```go
sandbox, err := skill.NewEnhancedSandbox(config, secretManager)
if err != nil {
    log.Fatal(err)
}
defer sandbox.Close()
```

3. **设置会话**:
```go
err := sandbox.SetSession(sessionID, userID)
if err != nil {
    log.Fatal(err)
}
```

4. **执行命令**:
```go
output, err := sandbox.ExecuteShellCommand("ls", "-la")
if err != nil {
    log.Printf("Command failed: %v", err)
}
fmt.Println(output)
```

### 配置最佳实践

#### 开发环境
```json
{
  "level": "basic",
  "dry_run": false,
  "verbose_logging": true,
  "audit_log_level": "info"
}
```

#### 生产环境
```json
{
  "level": "strict",
  "use_gvisor": true,
  "network_disabled": true,
  "anomaly_detection": true,
  "max_exec_per_minute": 30,
  "audit_log_level": "info"
}
```

#### 可信环境
```json
{
  "level": "none",
  "enabled": false
}
```

## API 参考

### EnhancedSandbox 方法

| 方法 | 描述 |
|------|------|
| `NewEnhancedSandbox(config, secretMgr)` | 创建增强沙箱 |
| `SetSession(sessionID, userID)` | 设置当前会话 |
| `ExecuteShellCommand(cmd, args...)` | 执行 Shell 命令 |
| `ExecuteFileOperation(op, path, data)` | 执行文件操作 |
| `GetSandboxInfo()` | 获取沙箱信息 |
| `Close()` | 关闭沙箱 |

### 工具函数

| 函数 | 描述 |
|------|------|
| `GenerateSandboxConfigTemplate()` | 生成配置模板 |
| `IsLinuxNamespaceAvailable()` | 检查 Namespace 可用性 |
| `IsGVisorAvailable()` | 检查 gVisor 可用性 |
| `GetRecommendedSandboxLevel()` | 获取推荐隔离级别 |

## 安全考虑

1. **最小权限原则**: 默认使用最严格的隔离级别
2. **纵深防御**: 多层验证和检测机制
3. **审计追溯**: 所有操作可追溯
4. **及时更新**: 定期更新黑名单和检测模式

## 故障排查

### 常见问题

1. **命令执行失败但无明显错误**
   - 检查审计日志查看详细原因
   - 确认命令在白名单中
   - 检查是否触发异常检测

2. **沙箱目录权限问题**
   - 确保 `file_sandbox_root` 目录可写
   - 检查 SELinux/AppArmor 策略

3. **Namespace 不可用**
   - 确认内核支持（Linux 3.8+）
   - 检查是否有足够权限

## 未来计划

- [ ] Windows Job Object 完整实现
- [ ] seccomp-bpf 系统调用过滤
- [ ] 实时资源监控和限制
- [ ] 分布式沙箱协调
- [ ] 机器学习驱动的异常检测

## 附录

### A. 默认命令白名单
```
ls, cat, echo, pwd, whoami, date, uname
```

### B. 默认命令黑名单
```
rm, sudo, su, chmod, chown, mount, umount, dd
```

### C. 可疑模式正则
```regex
[;&|]           # 命令分隔符
\$\(`          # 命令替换
>/dev/tcp/      # TCP 重定向
curl.*\|.*sh    # 下载并执行
wget.*\|.*sh    # 下载并执行
/etc/passwd     # 敏感文件
base64\s+-d     # Base64 解码执行
```
