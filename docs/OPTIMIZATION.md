# gclaw 深度优化文档

本文档详细说明了 gclaw 项目的深度优化功能，包括技能系统、记忆系统、性能与资源管理、部署运维能力以及安全加固。

## 目录

1. [技能系统深度优化](#1-技能系统深度优化)
2. [记忆系统增强](#2-记忆系统增强)
3. [性能与资源管理](#3-性能与资源管理)
4. [部署与运维能力](#4-部署与运维能力)
5. [安全隐患修复](#5-安全隐患修复)

---

## 1. 技能系统深度优化

### 1.1 技能审核机制

为防止恶意技能注入，我们实现了完整的技能审核流程：

```go
import "gclaw/internal/skill"

// 创建技能注册表
registry := skill.NewSkillRegistry()

// 注册技能（初始状态为 pending）
err := registry.Register(mySkill)

// 审核技能
err = registry.Approve("skill_id", "auditor_name", "审核通过")

// 拒绝技能
err = registry.Reject("skill_id", "auditor_name", "拒绝原因")
```

**安全检查项：**
- 危险资源访问检测（/etc/, /root/, sudo 等）
- 恶意内容检测（SQL 注入、命令注入、XSS 等）
- 依赖关系验证
- 权限声明审查

### 1.2 声明式依赖管理

技能可以声明其依赖和所需权限：

```go
type SkillMetadata struct {
    ID           string
    Name         string
    Version      string
    Permissions  []SkillPermission
    Dependencies []SkillDependency
}

// 示例依赖声明
dependencies := []skill.SkillDependency{
    {SkillID: "file_manager", Version: "^1.0.0", Optional: false},
    {SkillID: "network_client", Version: "~2.1.0", Optional: true},
}
```

### 1.3 技能热加载

支持无需重启服务的技能更新：

```go
// 热更新技能
err := registry.Reload("skill_id", newSkillVersion)

// 热卸载技能
err := registry.Unregister("skill_id")

// 验证依赖
err := registry.ValidateDependencies("skill_id")
```

---

## 2. 记忆系统增强

### 2.1 三级记忆架构

实现了短期、近端、长期三级记忆存储：

```go
import "gclaw/internal/memory"

// 创建三级记忆系统
mem := memory.NewVersionedMemory(
    1000,  // 总容量
    10,    // 短期记忆最大条数
    50,    // 近端记忆最大条数
)

// 获取不同级别的记忆
shortTerm, _ := mem.GetShortTerm(sessionID, 5)
nearTerm, _ := mem.GetNearTerm(sessionID, 10)
longTerm, _ := mem.GetLongTerm(sessionID, 20)
```

### 2.2 记忆压缩与总结

自动将旧记忆压缩整合到长期记忆：

```go
// 整合到长期记忆（带压缩）
err := mem.ConsolidateToLongTerm(sessionID)

// 获取会话摘要
summary, _ := mem.GetSummary(sessionID)
```

### 2.3 记忆版本控制

支持记忆状态的回滚和比较：

```go
// 回滚到指定版本
err := mem.Rollback(sessionID, versionNumber)

// 比较版本差异
diff, _ := mem.CompareVersions(sessionID, v1, v2)
```

### 2.4 知识图谱支持

扩展记忆系统支持知识图谱结构：

```go
// 添加知识节点
node := &memory.KnowledgeNode{
    Type: "entity",
    Name: "概念名称",
    Properties: map[string]interface{}{"key": "value"},
}
kg.AddNode(node)

// 添加关系边
edge := &memory.KnowledgeEdge{
    Source:   "node1_id",
    Target:   "node2_id",
    Relation: "related_to",
}
kg.AddEdge(edge)

// 查询相关节点
nodes, edges, _ := kg.GetRelatedNodes(nodeID)
```

---

## 3. 性能与资源管理

### 3.1 多级缓存系统

实现内存、磁盘、模型三级缓存：

```go
import "gclaw/internal/cache"

// 创建多级缓存
cache, _ := cache.NewMultiLevelCache(
    100 * 1024 * 1024,  // 内存缓存 100MB
    1 * 1024 * 1024 * 1024, // 磁盘缓存 1GB
    "/tmp/gclaw_cache", // 磁盘缓存目录
)

// 设置缓存
cache.Set("key", value)
cache.SetWithTTL("key", value, 1*time.Hour)

// 获取缓存
value, exists := cache.Get("key")

// 获取统计信息
stats := cache.GetStats()
fmt.Printf("Hit Rate: %.2f%%\n", stats.HitRate)
```

### 3.2 资源隔离机制

通过 cgroup 进行资源隔离（在 Docker 中自动生效）：

```yaml
# docker-compose.yml
deploy:
  resources:
    limits:
      cpus: '2.0'
      memory: 2G
    reservations:
      cpus: '0.5'
      memory: 512M
```

### 3.3 并行任务调度

优化任务调度算法，支持优先级管理：

```go
// 监控器自动收集指标并告警
monitor.Start(ctx)

// 注册健康检查
monitor.RegisterHealthCheck("database", checkDatabase)
monitor.RegisterHealthCheck("api", checkAPI)
```

### 3.4 资源使用监控

实时监控和预警机制：

```go
import "gclaw/internal/monitor"

// 创建监控器
mon := monitor.NewMonitor(10*time.Second, 100)

// 启动监控
mon.Start(ctx)

// 注册告警处理
mon.RegisterAlertHandler(monitor.DefaultAlertHandler)

// 设置告警阈值
mon.SetAlertThreshold("cpu", 80.0)
mon.SetAlertThreshold("memory", 85.0)

// 获取指标
metrics := mon.GetMetrics()

// Prometheus 集成
http.Handle("/metrics", mon)
```

---

## 4. 部署与运维能力

### 4.1 容器化支持

提供完整的 Docker 支持：

```bash
# 构建镜像
docker build -f deploy/Dockerfile -t gclaw:latest .

# 运行容器
docker run -d \
  -p 8080:8080 \
  -v ./config.json:/app/config.json:ro \
  -v gclaw_data:/app/data \
  gclaw:latest

# 使用 docker-compose
cd deploy
docker-compose up -d
```

### 4.2 配置热更新

实现配置的动态更新（需配合外部配置中心或文件监听）：

```go
// 监控配置文件变化
// 检测到变化后重新加载配置
// 无需重启服务
```

### 4.3 健康检查机制

自动检测和恢复功能：

```bash
# HTTP 健康检查端点
curl http://localhost:8080/health

# 返回格式
{
  "status": "healthy",
  "checks": [
    {"name": "database", "status": "healthy", "message": "OK"},
    {"name": "api", "status": "healthy", "message": "OK"}
  ],
  "timestamp": "2024-01-01T00:00:00Z"
}
```

### 4.4 监控告警系统

集成 Prometheus 和 Grafana：

```bash
# 启动完整监控栈
cd deploy
docker-compose up -d

# 访问 Prometheus: http://localhost:9090
# 访问 Grafana: http://localhost:3000 (admin/admin)
```

---

## 5. 安全隐患修复

### 5.1 默认配置加固

修复默认配置过于宽松的问题：

```json
{
  "security": {
    "bind_address": "127.0.0.1",
    "require_auth": true,
    "rate_limit": 100,
    "cors_origins": ["https://trusted-domain.com"]
  }
}
```

**安全建议：**
- 不要绑定到 0.0.0.0（除非必要）
- 启用身份验证
- 配置速率限制
- 限制 CORS 来源

### 5.2 敏感信息加密存储

使用 AES-GCM 加密 API 密钥和凭据：

```go
import "gclaw/internal/security"

// 创建敏感信息管理器
sm, _ := security.NewSecretManager(
    "your-encryption-key", // 或使用空字符串自动生成
    "/app/data/vault.enc",
)

// 存储敏感信息（自动加密）
sm.Store("openai_api_key", "sk-xxx", security.SecretTypeAPIKey)

// 获取敏感信息（自动解密）
apiKey, _ := sm.Get("openai_api_key")

// 列出所有密钥（不暴露值）
keys := sm.List()
```

### 5.3 权限最小化

运行时不使用管理员权限：

```dockerfile
# Dockerfile 中使用非 root 用户
RUN addgroup -g 1000 gclaw && \
    adduser -D -u 1000 -G gclaw gclaw
USER gclaw
```

### 5.4 输入校验与注入防护

防止提示词注入和其他注入攻击：

```go
import "gclaw/internal/security"

// 创建输入验证器
validator := security.NewInputValidator()

// 验证用户输入
err := validator.Validate(userInput)
if err != nil {
    // 检测到潜在攻击
    log.Printf("Blocked input: %v", err)
}

// 清理输入
sanitized := validator.Sanitize(userInput)

// 检测注入攻击
isInjection, pattern := validator.DetectInjection(userInput)
if isInjection {
    log.Printf("Injection attempt detected: %s", pattern)
}
```

**检测的注入模式：**
- 提示词注入："ignore previous instructions", "forget everything"
- SQL 注入：`' OR '1'='1`, `; DROP TABLE`
- 命令注入：`;`, `|`, `&`, backticks
- XSS：`<script>`, `javascript:`

### 5.5 路径安全验证

防止目录遍历攻击：

```go
// 检查路径是否安全
if !security.IsSecurePath(userPath, "/app/data") {
    return errors.New("invalid path")
}
```

---

## 快速开始

### 安装

```bash
# 克隆项目
git clone <repository-url>
cd gclaw

# 构建
go build -o gclaw ./cmd/main.go

# 或者使用 Docker
docker build -f deploy/Dockerfile -t gclaw:latest .
```

### 配置

```bash
# 复制示例配置
cp config.example.json config.json

# 编辑配置，确保：
# 1. 设置正确的 API Key
# 2. 配置安全选项
# 3. 设置合理的资源限制
```

### 运行

```bash
# 直接运行
./gclaw -config config.json

# Docker 运行
docker run -d -p 8080:8080 -v ./config.json:/app/config.json gclaw:latest

# Docker Compose（推荐，包含监控）
cd deploy
docker-compose up -d
```

### 监控

```bash
# 查看健康状态
curl http://localhost:8080/health

# 查看指标
curl http://localhost:8080/metrics

# 查看统计
curl http://localhost:8080/stats
```

---

## 最佳实践

1. **安全第一**
   - 始终加密敏感信息
   - 使用最小权限原则
   - 定期更新依赖
   - 启用输入验证

2. **性能优化**
   - 合理配置缓存大小
   - 监控资源使用
   - 及时清理过期数据
   - 使用连接池

3. **可维护性**
   - 使用版本控制管理配置
   - 记录所有操作日志
   - 实施健康检查
   - 配置告警通知

4. **可扩展性**
   - 使用模块化设计
   - 实现技能热加载
   - 支持水平扩展
   - 预留扩展接口

---

## 故障排除

### 常见问题

1. **内存使用过高**
   - 检查缓存配置
   - 调整记忆系统大小限制
   - 启用记忆压缩

2. **技能加载失败**
   - 检查审核状态
   - 验证依赖关系
   - 查看审核历史

3. **监控数据缺失**
   - 检查 Prometheus 配置
   - 验证网络连通性
   - 查看服务日志

### 日志位置

```bash
# Docker 日志
docker logs gclaw

# 本地日志
tail -f /app/logs/gclaw.log
```

---

## 贡献指南

欢迎提交 Issue 和 Pull Request！请确保：

1. 遵循代码规范
2. 添加必要的测试
3. 更新相关文档
4. 通过安全检查

---

## License

MIT
