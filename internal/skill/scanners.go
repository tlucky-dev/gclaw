package skill

import (
	"strings"
)

// StaticCodeScanner 静态代码扫描器（示例实现）
type StaticCodeScanner struct {
	name string
}

// NewStaticCodeScanner 创建静态代码扫描器
func NewStaticCodeScanner() *StaticCodeScanner {
	return &StaticCodeScanner{
		name: "static_code_scanner",
	}
}

// Name 返回扫描器名称
func (s *StaticCodeScanner) Name() string {
	return s.name
}

// Scan 执行静态代码扫描
func (s *StaticCodeScanner) Scan(skill SkillDefinition) ([]string, []string, RiskLevel) {
	var issues []string
	var warnings []string
	riskLevel := RiskLevelLow

	metadata := skill.GetMetadata()

	// 检查技能名称是否包含可疑内容
	if strings.ContainsAny(metadata.Name, "<>&\"'") {
		issues = append(issues, "skill name contains suspicious characters")
		riskLevel = RiskLevelHigh
	}

	// 检查描述中是否包含危险关键词
	dangerousKeywords := []string{
		"eval", "exec", "system", "shell",
		"rm -rf", "chmod", "chown",
		"DROP TABLE", "DELETE FROM",
	}

	for _, keyword := range dangerousKeywords {
		if strings.Contains(strings.ToLower(metadata.Description), strings.ToLower(keyword)) {
			warnings = append(warnings, "description contains potentially dangerous keyword: "+keyword)
			if riskLevel < RiskLevelMedium {
				riskLevel = RiskLevelMedium
			}
		}
	}

	// 检查权限声明是否过度
	for _, perm := range metadata.Permissions {
		if perm.Required && len(perm.Resources) == 0 {
			warnings = append(warnings, "required permission without specific resources: "+perm.Name)
		}

		// 检查是否有通配符权限
		for _, action := range perm.Actions {
			if action == "*" || action == "all" {
				issues = append(issues, "wildcard action not allowed: "+perm.Name)
				riskLevel = RiskLevelHigh
			}
		}

		// 检查危险资源访问
		for _, res := range perm.Resources {
			if isDangerousResource(res) {
				issues = append(issues, "dangerous resource access: "+res)
				riskLevel = RiskLevelHigh
			}
		}
	}

	// 检查依赖是否为空 ID
	for _, dep := range metadata.Dependencies {
		if dep.SkillID == "" {
			issues = append(issues, "dependency with empty skill_id")
			if riskLevel < RiskLevelMedium {
				riskLevel = RiskLevelMedium
			}
		}
	}

	// 检查作者信息
	if metadata.Author == "" || metadata.Author == "anonymous" {
		warnings = append(warnings, "skill author is not specified or anonymous")
	}

	// 检查版本格式
	if !isValidVersion(metadata.Version) {
		warnings = append(warnings, "version format may not follow semantic versioning")
	}

	// 如果有清单，检查资源限制
	if metadata.Manifest != nil {
		if metadata.Manifest.MaxMemoryMB <= 0 {
			warnings = append(warnings, "no memory limit specified in manifest")
		}
		if metadata.Manifest.MaxCPUPercent <= 0 {
			warnings = append(warnings, "no CPU limit specified in manifest")
		}

		// 检查路径限制
		for _, path := range metadata.Manifest.BlockedPaths {
			if path == "/" || path == "/etc" || path == "/root" {
				warnings = append(warnings, "broad path blocking may cause issues: "+path)
			}
		}
	}

	return issues, warnings, riskLevel
}

// PermissionScanner 权限扫描器
type PermissionScanner struct {
	name string
}

// NewPermissionScanner 创建权限扫描器
func NewPermissionScanner() *PermissionScanner {
	return &PermissionScanner{
		name: "permission_scanner",
	}
}

// Name 返回扫描器名称
func (s *PermissionScanner) Name() string {
	return s.name
}

// Scan 执行权限扫描
func (s *PermissionScanner) Scan(skill SkillDefinition) ([]string, []string, RiskLevel) {
	var issues []string
	var warnings []string
	riskLevel := RiskLevelLow

	metadata := skill.GetMetadata()

	// 定义敏感权限
	sensitivePermissions := map[string]RiskLevel{
		"filesystem_write":    RiskLevelHigh,
		"filesystem_delete":   RiskLevelHigh,
		"network_access":      RiskLevelMedium,
		"database_write":      RiskLevelHigh,
		"database_delete":     RiskLevelHigh,
		"execute_command":     RiskLevelCritical,
		"admin_access":        RiskLevelCritical,
		"root_access":         RiskLevelCritical,
		"sudo":                RiskLevelCritical,
	}

	for _, perm := range metadata.Permissions {
		permName := strings.ToLower(perm.Name)
		
		// 检查是否是敏感权限
		if risk, exists := sensitivePermissions[permName]; exists {
			if perm.Required {
				issues = append(issues, "requires sensitive permission: "+perm.Name)
				if risk > riskLevel {
					riskLevel = risk
				}
			} else {
				warnings = append(warnings, "optional sensitive permission: "+perm.Name)
			}
		}

		// 检查动作
		for _, action := range perm.Actions {
			actionLower := strings.ToLower(action)
			if actionLower == "delete" || actionLower == "execute" {
				if perm.Required {
					warnings = append(warnings, "required destructive action: "+action+" on "+perm.Name)
					if riskLevel < RiskLevelHigh {
						riskLevel = RiskLevelHigh
					}
				}
			}
		}

		// 检查资源路径
		for _, res := range perm.Resources {
			// 检查系统关键路径
			systemPaths := []string{"/etc/", "/root/", "/proc/", "/sys/", "/dev/"}
			for _, sysPath := range systemPaths {
				if strings.HasPrefix(res, sysPath) {
					issues = append(issues, "access to system critical path: "+res)
					riskLevel = RiskLevelCritical
				}
			}

			// 检查是否尝试访问环境变量
			if strings.Contains(res, "$") || strings.Contains(res, "${") {
				warnings = append(warnings, "potential environment variable access: "+res)
				if riskLevel < RiskLevelMedium {
					riskLevel = RiskLevelMedium
				}
			}
		}
	}

	// 检查权限数量
	if len(metadata.Permissions) > 10 {
		warnings = append(warnings, "skill requests excessive number of permissions: "+string(rune(len(metadata.Permissions))))
	}

	// 检查是否有只读权限声明
	hasReadOnly := false
	for _, perm := range metadata.Permissions {
		if len(perm.Actions) == 1 && perm.Actions[0] == "read" {
			hasReadOnly = true
			break
		}
	}
	
	if !hasReadOnly && len(metadata.Permissions) > 0 {
		warnings = append(warnings, "no explicit read-only permissions declared")
	}

	return issues, warnings, riskLevel
}

// DependencyScanner 依赖扫描器
type DependencyScanner struct {
	name string
}

// NewDependencyScanner 创建依赖扫描器
func NewDependencyScanner() *DependencyScanner {
	return &DependencyScanner{
		name: "dependency_scanner",
	}
}

// Name 返回扫描器名称
func (s *DependencyScanner) Name() string {
	return s.name
}

// Scan 执行依赖扫描
func (s *DependencyScanner) Scan(skill SkillDefinition) ([]string, []string, RiskLevel) {
	var issues []string
	var warnings []string
	riskLevel := RiskLevelLow

	metadata := skill.GetMetadata()

	// 检查循环依赖（简单检查）
	depIDs := make(map[string]bool)
	for _, dep := range metadata.Dependencies {
		if depIDs[dep.SkillID] {
			issues = append(issues, "duplicate dependency: "+dep.SkillID)
			if riskLevel < RiskLevelMedium {
				riskLevel = RiskLevelMedium
			}
		}
		depIDs[dep.SkillID] = true
	}

	// 检查依赖版本格式
	for _, dep := range metadata.Dependencies {
		if dep.Version != "" {
			if !isValidVersion(dep.Version) && 
			   !strings.HasPrefix(dep.Version, "^") && 
			   !strings.HasPrefix(dep.Version, "~") {
				warnings = append(warnings, "dependency version may not follow semantic versioning: "+dep.SkillID)
			}
		}

		// 检查可选依赖过多
		if dep.Optional {
			warnings = append(warnings, "optional dependency: "+dep.SkillID)
		}
	}

	// 检查依赖自身
	if metadata.ID != "" {
		if depIDs[metadata.ID] {
			issues = append(issues, "skill depends on itself")
			riskLevel = RiskLevelHigh
		}
	}

	// 警告：没有依赖可能是孤立的技能
	if len(metadata.Dependencies) == 0 && len(metadata.Permissions) > 0 {
		warnings = append(warnings, "skill has permissions but no dependencies, may be isolated")
	}

	return issues, warnings, riskLevel
}

// isValidVersion 检查版本格式是否有效（简单语义化版本检查）
func isValidVersion(version string) bool {
	if version == "" {
		return false
	}
	
	// 去除前缀
	v := version
	if len(v) > 0 && (v[0] == '^' || v[0] == '~' || v[0] == 'v') {
		v = v[1:]
	}
	
	if v == "" {
		return false
	}

	// 简单检查：至少有一个数字和一个点
	hasDigit := false
	hasDot := false
	for _, c := range v {
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
		if c == '.' {
			hasDot = true
		}
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	
	return hasDigit && hasDot
}

// RegisterDefaultScanners 注册默认扫描器到技能注册表
func RegisterDefaultScanners(registry *SkillRegistry) {
	registry.RegisterScanner(NewStaticCodeScanner())
	registry.RegisterScanner(NewPermissionScanner())
	registry.RegisterScanner(NewDependencyScanner())
}
