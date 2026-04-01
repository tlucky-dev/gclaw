package skill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gclaw/pkg/errors"
)

// SkillStatus 技能状态
type SkillStatus string

const (
	SkillStatusPending   SkillStatus = "pending"
	SkillStatusActive    SkillStatus = "active"
	SkillStatusDisabled  SkillStatus = "disabled"
	SkillStatusRejected  SkillStatus = "rejected"
	SkillStatusSuspended SkillStatus = "suspended"
)

// RiskLevel 风险等级
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// SkillPermission 技能权限
type SkillPermission struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Resources   []string `json:"resources,omitempty"`
	Actions     []string `json:"actions,omitempty"` // read, write, execute, delete
}

// SkillDependency 技能依赖
type SkillDependency struct {
	SkillID   string `json:"skill_id"`
	Version   string `json:"version"`
	Optional  bool   `json:"optional"`
}

// SkillManifest 技能清单（用于声明式依赖管理）
type SkillManifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Description  string            `json:"description"`
	Author       string            `json:"author"`
	License      string            `json:"license,omitempty"`
	Homepage     string            `json:"homepage,omitempty"`
	Permissions  []SkillPermission `json:"permissions,omitempty"`
	Dependencies []SkillDependency `json:"dependencies,omitempty"`
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
	Tags         []string          `json:"tags,omitempty"`
	MinRuntimeVersion string       `json:"min_runtime_version,omitempty"`
	MaxMemoryMB   int              `json:"max_memory_mb,omitempty"`
	MaxCPUPercent float64          `json:"max_cpu_percent,omitempty"`
	AllowedPaths  []string         `json:"allowed_paths,omitempty"`
	BlockedPaths  []string         `json:"blocked_paths,omitempty"`
}

// SkillMetadata 技能元数据
type SkillMetadata struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Status      SkillStatus       `json:"status"`
	Tags        []string          `json:"tags,omitempty"`
	Permissions []SkillPermission `json:"permissions,omitempty"`
	Dependencies []SkillDependency `json:"dependencies,omitempty"`
	ConfigSchema map[string]interface{} `json:"config_schema,omitempty"`
	Manifest    *SkillManifest    `json:"manifest,omitempty"`
	Checksum    string            `json:"checksum"`
	SourcePath  string            `json:"source_path,omitempty"`
	PluginPath  string            `json:"plugin_path,omitempty"`
}

// SkillDefinition 技能定义接口
type SkillDefinition interface {
	GetMetadata() *SkillMetadata
	Initialize(config map[string]interface{}) error
	Execute(args map[string]interface{}) (string, error)
	Validate() error
	GetRequiredPermissions() []SkillPermission
	GetDependencies() []SkillDependency
	Shutdown() error
}

// SkillAuditRecord 技能审核记录
type SkillAuditRecord struct {
	SkillID     string    `json:"skill_id"`
	Auditor     string    `json:"auditor"`
	AuditTime   time.Time `json:"audit_time"`
	Passed      bool      `json:"passed"`
	Comments    string    `json:"comments,omitempty"`
	RiskLevel   RiskLevel `json:"risk_level"`
	Issues      []string  `json:"issues,omitempty"`
	Warnings    []string  `json:"warnings,omitempty"`
	ScannerUsed []string  `json:"scanners_used,omitempty"`
}

// SkillScanner 技能扫描器接口
type SkillScanner interface {
	Scan(skill SkillDefinition) ([]string, []string, RiskLevel)
	Name() string
}

// HotReloadConfig 热加载配置
type HotReloadConfig struct {
	Enabled     bool          `json:"enabled"`
	WatchDir    string        `json:"watch_dir"`
	PollInterval time.Duration `json:"poll_interval"`
	AutoApprove bool          `json:"auto_approve"` // 仅用于开发环境
}

// SkillRegistry 技能注册表
type SkillRegistry struct {
	mu           sync.RWMutex
	skills       map[string]SkillDefinition
	metadata     map[string]*SkillMetadata
	auditRecords map[string][]*SkillAuditRecord
	statuses     map[string]SkillStatus
	hotReload    bool
	watchDir     string
	scanners     []SkillScanner
	fileHashes   map[string]string // 文件路径 -> hash，用于检测变化
	stopChan     chan struct{}
	pluginLoader *PluginLoader // 插件加载器
}

// NewSkillRegistry 创建新的技能注册表
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		skills:       make(map[string]SkillDefinition),
		metadata:     make(map[string]*SkillMetadata),
		auditRecords: make(map[string][]*SkillAuditRecord),
		statuses:     make(map[string]SkillStatus),
		hotReload:    true,
		scanners:     make([]SkillScanner, 0),
		fileHashes:   make(map[string]string),
		stopChan:     make(chan struct{}),
		pluginLoader: NewPluginLoader(),
	}
}

// Register 注册技能（需要经过审核）
func (r *SkillRegistry) Register(skill SkillDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	metadata := skill.GetMetadata()

	// 验证技能
	if err := skill.Validate(); err != nil {
		return errors.WrapError(errors.ErrValidation, "skill validation failed", err)
	}

	// 检查是否已存在
	if _, exists := r.skills[metadata.ID]; exists {
		return fmt.Errorf("skill '%s' already registered", metadata.ID)
	}

	// 默认状态为 pending，需要审核后才能激活
	r.skills[metadata.ID] = skill
	r.metadata[metadata.ID] = metadata
	r.statuses[metadata.ID] = SkillStatusPending

	return nil
}

// RegisterWithManifest 通过清单文件注册技能（声明式依赖管理）
func (r *SkillRegistry) RegisterWithManifest(manifestPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 读取并解析清单文件
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest SkillManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// 验证清单
	if err := validateManifest(&manifest); err != nil {
		return fmt.Errorf("invalid manifest: %w", err)
	}

	// 检查依赖
	if err := r.checkDependencies(&manifest); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	// 验证权限声明
	if err := validatePermissions(&manifest); err != nil {
		return fmt.Errorf("permission validation failed: %w", err)
	}

	// 计算文件 checksum
	checksum := calculateFileChecksumSimple(data)

	// 创建元数据
	metadata := &SkillMetadata{
		ID:           manifest.ID,
		Name:         manifest.Name,
		Version:      manifest.Version,
		Description:  manifest.Description,
		Author:       manifest.Author,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Status:       SkillStatusPending,
		Tags:         manifest.Tags,
		Permissions:  manifest.Permissions,
		Dependencies: manifest.Dependencies,
		ConfigSchema: manifest.ConfigSchema,
		Manifest:     &manifest,
		Checksum:     checksum,
		SourcePath:   filepath.Dir(manifestPath),
	}

	// 存储占位符，等待实际技能加载
	r.metadata[manifest.ID] = metadata
	r.statuses[manifest.ID] = SkillStatusPending

	return nil
}

// LoadPlugin 动态加载插件技能（支持热加载）
// 注意：Go 的 plugin 包仅支持 Linux/macOS，且需要特殊编译选项 (-buildmode=plugin)
// 编译插件示例：go build -buildmode=plugin -o myskill.so ./myskill/
func (r *SkillRegistry) LoadPlugin(pluginPath string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 验证插件文件
	if err := ValidatePluginFile(pluginPath); err != nil {
		return fmt.Errorf("invalid plugin file: %w", err)
	}

	// 打开插件（带超时）
	p, err := r.pluginLoader.loadPluginWithTimeout(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// 获取可用符号列表
	symbolNames := GetPluginSymbols(p)
	if len(symbolNames) == 0 {
		return fmt.Errorf("no valid skill symbols found in plugin")
	}

	// 查找技能符号
	sym, err := p.Lookup("Skill")
	if err != nil {
		// 尝试其他可能的符号名
		sym, err = p.Lookup("NewSkill")
		if err != nil {
			return fmt.Errorf("failed to find Skill or NewSkill symbol: %w", err)
		}
	}

	skill, ok := sym.(SkillDefinition)
	if !ok {
		return fmt.Errorf("invalid skill type in plugin: expected SkillDefinition")
	}

	// 获取元数据并更新插件路径
	metadata := skill.GetMetadata()
	metadata.PluginPath = pluginPath
	
	// 计算文件 checksum
	data, _ := os.ReadFile(pluginPath)
	metadata.Checksum = calculateFileChecksumSimple(data)

	// 验证技能
	if err := skill.Validate(); err != nil {
		return errors.WrapError(errors.ErrValidation, "plugin skill validation failed", err)
	}

	// 执行安全扫描
	issues, warnings, riskLevel := r.performSecurityScan(skill)
	
	// 创建审核记录
	record := &SkillAuditRecord{
		SkillID:     metadata.ID,
		Auditor:     "auto_scanner",
		AuditTime:   time.Now(),
		Passed:      riskLevel != RiskLevelCritical,
		RiskLevel:   riskLevel,
		Issues:      issues,
		Warnings:    warnings,
		ScannerUsed: []string{"static_analyzer", "permission_checker", "plugin_validator"},
	}

	r.auditRecords[metadata.ID] = append(r.auditRecords[metadata.ID], record)

	if riskLevel == RiskLevelCritical {
		return fmt.Errorf("skill blocked due to critical security issues: %v", issues)
	}

	// 存储插件信息
	absPath, _ := filepath.Abs(pluginPath)
	fileInfo, _ := os.Stat(pluginPath)
	r.pluginLoader.loadedPlugins[metadata.ID] = p
	r.pluginLoader.pluginInfo[metadata.ID] = &PluginInfo{
		Path:        absPath,
		LoadedAt:    time.Now(),
		Checksum:    metadata.Checksum,
		Size:        fileInfo.Size(),
		Version:     metadata.Version,
		SymbolNames: symbolNames,
	}

	r.skills[metadata.ID] = skill
	r.metadata[metadata.ID] = metadata
	r.statuses[metadata.ID] = SkillStatusPending // 需要手动审批

	return nil
}

// Approve 审核通过技能
func (r *SkillRegistry) Approve(skillID, auditor, comments string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	skill, exists := r.skills[skillID]
	if !exists {
		return fmt.Errorf("skill '%s' not found", skillID)
	}

	// 执行安全检查
	issues, warnings, riskLevel := r.performSecurityScan(skill)
	
	record := &SkillAuditRecord{
		SkillID:     skillID,
		Auditor:     auditor,
		AuditTime:   time.Now(),
		Passed:      len(issues) == 0 || riskLevel != RiskLevelCritical,
		Comments:    comments,
		RiskLevel:   riskLevel,
		Issues:      issues,
		Warnings:    warnings,
		ScannerUsed: []string{"manual_review"},
	}

	r.auditRecords[skillID] = append(r.auditRecords[skillID], record)

	if record.Passed {
		r.statuses[skillID] = SkillStatusActive
		return nil
	}

	return fmt.Errorf("skill approval failed: %v", issues)
}

// Reject 拒绝技能
func (r *SkillRegistry) Reject(skillID, auditor, comments string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skillID]; !exists {
		return fmt.Errorf("skill '%s' not found", skillID)
	}

	record := &SkillAuditRecord{
		SkillID:     skillID,
		Auditor:     auditor,
		AuditTime:   time.Now(),
		Passed:      false,
		Comments:    comments,
		RiskLevel:   RiskLevelCritical,
	}

	r.auditRecords[skillID] = append(r.auditRecords[skillID], record)
	r.statuses[skillID] = SkillStatusRejected

	return nil
}

// performSecurityScan 执行安全扫描（多扫描器）
func (r *SkillRegistry) performSecurityScan(skill SkillDefinition) ([]string, []string, RiskLevel) {
	var allIssues []string
	var allWarnings []string
	maxRisk := RiskLevelLow

	metadata := skill.GetMetadata()

	// 运行所有注册的扫描器
	for _, scanner := range r.scanners {
		issues, warnings, risk := scanner.Scan(skill)
		allIssues = append(allIssues, issues...)
		allWarnings = append(allWarnings, warnings...)
		if risk > maxRisk {
			maxRisk = risk
		}
	}

	// 内置基础检查
	builtinIssues, builtinWarnings, builtinRisk := r.builtinSecurityCheck(metadata)
	allIssues = append(allIssues, builtinIssues...)
	allWarnings = append(allWarnings, builtinWarnings...)
	if builtinRisk > maxRisk {
		maxRisk = builtinRisk
	}

	return allIssues, allWarnings, maxRisk
}

// builtinSecurityCheck 内置安全检查
func (r *SkillRegistry) builtinSecurityCheck(metadata *SkillMetadata) ([]string, []string, RiskLevel) {
	var issues []string
	var warnings []string
	riskLevel := RiskLevelLow

	// 检查权限声明
	for _, perm := range metadata.Permissions {
		if perm.Required && len(perm.Resources) > 0 {
			for _, res := range perm.Resources {
				if isDangerousResource(res) {
					issues = append(issues, fmt.Sprintf("dangerous resource access: %s", res))
					riskLevel = RiskLevelHigh
				}
			}
		}
		// 检查危险动作
		for _, action := range perm.Actions {
			if action == "delete" || action == "execute" {
				warnings = append(warnings, fmt.Sprintf("potentially dangerous action: %s", action))
				if riskLevel < RiskLevelMedium {
					riskLevel = RiskLevelMedium
				}
			}
		}
	}

	// 检查依赖
	for _, dep := range metadata.Dependencies {
		if dep.SkillID == "" {
			issues = append(issues, "invalid dependency: empty skill_id")
			if riskLevel < RiskLevelMedium {
				riskLevel = RiskLevelMedium
			}
		}
	}

	// 检查名称和描述
	if metadata.Name == "" {
		issues = append(issues, "empty skill name")
		if riskLevel < RiskLevelMedium {
			riskLevel = RiskLevelMedium
		}
	}

	if containsMaliciousContent(metadata.Description) {
		issues = append(issues, "potentially malicious content in description")
		riskLevel = RiskLevelCritical
	}

	// 检查资源限制声明
	if metadata.Manifest != nil {
		if metadata.Manifest.MaxMemoryMB <= 0 {
			warnings = append(warnings, "no memory limit specified")
		}
		if metadata.Manifest.MaxCPUPercent <= 0 {
			warnings = append(warnings, "no CPU limit specified")
		}
	}

	return issues, warnings, riskLevel
}

// isDangerousResource 检查是否是危险资源
func isDangerousResource(resource string) bool {
	dangerousPatterns := []string{
		"/etc/", "/root/", "/proc/", "/sys/",
		"sudo", "admin", "root",
		"DELETE", "DROP", "TRUNCATE",
	}

	for _, pattern := range dangerousPatterns {
		if containsIgnoreCase(resource, pattern) {
			return true
		}
	}
	return false
}

// containsMaliciousContent 检查是否包含恶意内容
func containsMaliciousContent(content string) bool {
	maliciousPatterns := []string{
		"<script>", "javascript:", "eval(",
		"DROP TABLE", "DELETE FROM",
		"rm -rf /", "chmod 777",
	}

	for _, pattern := range maliciousPatterns {
		if containsIgnoreCase(content, pattern) {
			return true
		}
	}
	return false
}

// containsIgnoreCase 忽略大小写检查子串
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || len(s) > len(substr) && 
		(findSubstringIgnoreCase(s, substr)))
}

func findSubstringIgnoreCase(s, substr string) bool {
	sLower := toLower(s)
	substrLower := toLower(substr)
	return containsSimple(sLower, substrLower)
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			result[i] = c + 32
		} else {
			result[i] = c
		}
	}
	return string(result[:len(s)])
}

func containsSimple(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Get 获取技能
func (r *SkillRegistry) Get(name string) (SkillDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	skill, exists := r.skills[name]
	return skill, exists
}

// GetStatus 获取技能状态
func (r *SkillRegistry) GetStatus(skillID string) (SkillStatus, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	status, exists := r.statuses[skillID]
	return status, exists
}

// List 列出所有技能
func (r *SkillRegistry) List() []SkillDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SkillDefinition, 0, len(r.skills))
	for _, skill := range r.skills {
		result = append(result, skill)
	}
	return result
}

// ListByStatus 按状态列出技能
func (r *SkillRegistry) ListByStatus(status SkillStatus) []SkillDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]SkillDefinition, 0)
	for id, skill := range r.skills {
		if r.statuses[id] == status {
			result = append(result, skill)
		}
	}
	return result
}

// Execute 执行技能
func (r *SkillRegistry) Execute(name string, args map[string]interface{}) (string, error) {
	r.mu.RLock()
	skill, exists := r.skills[name]
	status := r.statuses[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("skill '%s' not found", name)
	}

	if status != SkillStatusActive {
		return "", fmt.Errorf("skill '%s' is not active (current status: %s)", name, status)
	}

	return skill.Execute(args)
}

// Unregister 注销技能（支持热卸载）
func (r *SkillRegistry) Unregister(skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skillID]; !exists {
		return fmt.Errorf("skill '%s' not found", skillID)
	}

	delete(r.skills, skillID)
	delete(r.metadata, skillID)
	delete(r.statuses, skillID)

	return nil
}

// Reload 重新加载技能（支持热更新）
func (r *SkillRegistry) Reload(skillID string, newSkill SkillDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skillID]; !exists {
		return fmt.Errorf("skill '%s' not found", skillID)
	}

	// 验证新技能
	if err := newSkill.Validate(); err != nil {
		return errors.WrapError(errors.ErrValidation, "new skill validation failed", err)
	}

	// 保留原有状态和审核记录
	metadata := newSkill.GetMetadata()
	oldStatus := r.statuses[skillID]
	
	r.skills[skillID] = newSkill
	r.metadata[skillID] = metadata
	r.statuses[skillID] = oldStatus

	return nil
}

// GetAuditHistory 获取审核历史
func (r *SkillRegistry) GetAuditHistory(skillID string) []*SkillAuditRecord {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.auditRecords[skillID]
}

// ValidateDependencies 验证技能依赖
func (r *SkillRegistry) ValidateDependencies(skillID string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metadata, exists := r.metadata[skillID]
	if !exists {
		return fmt.Errorf("skill '%s' not found", skillID)
	}

	for _, dep := range metadata.Dependencies {
		if depSkill, exists := r.skills[dep.SkillID]; !exists {
			if !dep.Optional {
				return fmt.Errorf("missing required dependency: %s", dep.SkillID)
			}
		} else {
			// 检查版本兼容性
			if !isVersionCompatible(depSkill.GetMetadata().Version, dep.Version) {
				return fmt.Errorf("dependency version mismatch: %s requires %s, got %s",
					dep.SkillID, dep.Version, depSkill.GetMetadata().Version)
			}
		}
	}

	return nil
}

// isVersionCompatible 检查版本兼容性（简单语义化版本检查）
func isVersionCompatible(actual, required string) bool {
	if required == "" || actual == required {
		return true
	}
	// 简单实现：前缀匹配
	if len(required) > 0 && required[0] == '^' {
		// ^1.2.3 表示兼容 1.x.x
		reqMajor := getVersionPart(required[1:], 0)
		actMajor := getVersionPart(actual, 0)
		return reqMajor == actMajor
	}
	if len(required) > 0 && required[0] == '~' {
		// ~1.2.3 表示兼容 1.2.x
		reqMajor := getVersionPart(required[1:], 0)
		reqMinor := getVersionPart(required[1:], 1)
		actMajor := getVersionPart(actual, 0)
		actMinor := getVersionPart(actual, 1)
		return reqMajor == actMajor && reqMinor == actMinor
	}
	return actual == required
}

func getVersionPart(version string, index int) string {
	parts := splitVersion(version)
	if index < len(parts) {
		return parts[index]
	}
	return "0"
}

func splitVersion(version string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(version); i++ {
		if version[i] == '.' {
			parts = append(parts, version[start:i])
			start = i + 1
		}
	}
	parts = append(parts, version[start:])
	return parts
}

// validateManifest 验证技能清单
func validateManifest(manifest *SkillManifest) error {
	if manifest.ID == "" {
		return fmt.Errorf("manifest ID is required")
	}
	if manifest.Name == "" {
		return fmt.Errorf("manifest name is required")
	}
	if manifest.Version == "" {
		return fmt.Errorf("manifest version is required")
	}
	if manifest.Author == "" {
		return fmt.Errorf("manifest author is required")
	}
	return nil
}

// checkDependencies 检查依赖是否满足
func (r *SkillRegistry) checkDependencies(manifest *SkillManifest) error {
	for _, dep := range manifest.Dependencies {
		if dep.SkillID == "" {
			return fmt.Errorf("dependency skill_id is required")
		}
		// 检查依赖是否存在（如果非可选）
		if !dep.Optional {
			if _, exists := r.skills[dep.SkillID]; !exists {
				// 依赖不存在，但允许后续安装
				// 这里只是警告，不阻止注册
			}
		}
	}
	return nil
}

// validatePermissions 验证权限声明
func validatePermissions(manifest *SkillManifest) error {
	for _, perm := range manifest.Permissions {
		if perm.Name == "" {
			return fmt.Errorf("permission name is required")
		}
		// 检查是否有过度权限
		for _, action := range perm.Actions {
			if action == "*" || action == "all" {
				return fmt.Errorf("wildcard permissions are not allowed: %s", perm.Name)
			}
		}
	}
	return nil
}

// calculateFileChecksumSimple 计算文件内容的简单 checksum
func calculateFileChecksumSimple(data []byte) string {
	// 使用简单的 hash 实现，避免额外依赖
	sum := uint64(0)
	for i, b := range data {
		sum += uint64(b) << (uint(i % 8) * 8)
	}
	return fmt.Sprintf("%016x", sum)
}

// RegisterScanner 注册技能扫描器
func (r *SkillRegistry) RegisterScanner(scanner SkillScanner) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scanners = append(r.scanners, scanner)
}

// StartHotReload 启动热加载监控
func (r *SkillRegistry) StartHotReload(config HotReloadConfig) error {
	if !config.Enabled {
		return nil
	}

	r.watchDir = config.WatchDir
	r.fileHashes = make(map[string]string)

	// 初始扫描目录
	if err := r.scanWatchDirectory(); err != nil {
		return err
	}

	// 启动定时轮询
	go func() {
		ticker := time.NewTicker(config.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-r.stopChan:
				return
			case <-ticker.C:
				if err := r.scanWatchDirectory(); err != nil {
					fmt.Printf("Warning: failed to scan watch directory: %v\n", err)
				}
			}
		}
	}()

	return nil
}

// StopHotReload 停止热加载监控
func (r *SkillRegistry) StopHotReload() {
	close(r.stopChan)
}

// scanWatchDirectory 扫描监控目录检测变化
func (r *SkillRegistry) scanWatchDirectory() error {
	if r.watchDir == "" {
		return nil
	}

	entries, err := os.ReadDir(r.watchDir)
	if err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	currentFiles := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".json" && filepath.Ext(name) != ".so" {
			continue
		}

		currentFiles[name] = true
		filePath := filepath.Join(r.watchDir, name)

		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		checksum := calculateFileChecksumSimple(data)

		// 检测文件变化
		oldChecksum, exists := r.fileHashes[name]
		if !exists {
			// 新文件
			fmt.Printf("New skill file detected: %s\n", name)
			// 可以尝试自动加载
		} else if oldChecksum != checksum {
			// 文件已修改
			fmt.Printf("Skill file modified: %s\n", name)
			// 可以尝试重新加载
		}

		r.fileHashes[name] = checksum
	}

	// 检测删除的文件
	for name := range r.fileHashes {
		if !currentFiles[name] {
			fmt.Printf("Skill file removed: %s\n", name)
			// 可以尝试卸载
		}
	}

	return nil
}
