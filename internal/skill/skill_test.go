package skill

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestSkillRegistry 测试技能注册表基本功能
func TestSkillRegistry(t *testing.T) {
	registry := NewSkillRegistry()
	
	if registry == nil {
		t.Fatal("Failed to create skill registry")
	}
	
	// 测试空列表
	skills := registry.List()
	if len(skills) != 0 {
		t.Errorf("Expected empty list, got %d skills", len(skills))
	}
}

// TestRiskLevel 测试风险等级
func TestRiskLevel(t *testing.T) {
	levels := []RiskLevel{
		RiskLevelLow,
		RiskLevelMedium,
		RiskLevelHigh,
		RiskLevelCritical,
	}
	
	expected := []string{"low", "medium", "high", "critical"}
	
	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], level)
		}
	}
}

// TestSkillStatus 测试技能状态
func TestSkillStatus(t *testing.T) {
	statuses := []SkillStatus{
		SkillStatusPending,
		SkillStatusActive,
		SkillStatusDisabled,
		SkillStatusRejected,
		SkillStatusSuspended,
	}
	
	expected := []string{"pending", "active", "disabled", "rejected", "suspended"}
	
	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], status)
		}
	}
}

// TestIsDangerousResource 测试危险资源检测
func TestIsDangerousResource(t *testing.T) {
	dangerous := []string{
		"/etc/passwd",
		"/root/.ssh",
		"/proc/self",
		"sudo command",
		"admin access",
		"DROP TABLE users",
	}
	
	safe := []string{
		"/tmp/data",
		"/var/log/app.log",
		"user data",
	}
	
	for _, res := range dangerous {
		if !isDangerousResource(res) {
			t.Errorf("Expected %s to be dangerous", res)
		}
	}
	
	for _, res := range safe {
		if isDangerousResource(res) {
			t.Errorf("Expected %s to be safe", res)
		}
	}
}

// TestContainsMaliciousContent 测试恶意内容检测
func TestContainsMaliciousContent(t *testing.T) {
	malicious := []string{
		"<script>alert('xss')</script>",
		"javascript:void(0)",
		"eval(malicious_code)",
		"DROP TABLE users",
		"rm -rf /",
		"chmod 777 /etc",
	}
	
	safe := []string{
		"normal text",
		"user description",
		"hello world",
	}
	
	for _, content := range malicious {
		if !containsMaliciousContent(content) {
			t.Errorf("Expected %s to be detected as malicious", content)
		}
	}
	
	for _, content := range safe {
		if containsMaliciousContent(content) {
			t.Errorf("Expected %s to be safe", content)
		}
	}
}

// TestVersionCompatibility 测试版本兼容性检查
func TestVersionCompatibility(t *testing.T) {
	tests := []struct {
		actual   string
		required string
		expected bool
	}{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"1.2.5", "~1.2.3", true},
		{"1.3.0", "~1.2.3", false},
		{"1.0.0", "", true},
	}
	
	for _, test := range tests {
		result := isVersionCompatible(test.actual, test.required)
		if result != test.expected {
			t.Errorf("isVersionCompatible(%s, %s) = %v, expected %v",
				test.actual, test.required, result, test.expected)
		}
	}
}

// TestValidateManifest 测试清单验证
func TestValidateManifest(t *testing.T) {
	valid := &SkillManifest{
		ID:      "test-skill",
		Name:    "Test Skill",
		Version: "1.0.0",
		Author:  "Test Author",
	}
	
	if err := validateManifest(valid); err != nil {
		t.Errorf("Expected valid manifest, got error: %v", err)
	}
	
	invalid := []*SkillManifest{
		{Name: "Test", Version: "1.0.0", Author: "Author"}, // missing ID
		{ID: "test", Version: "1.0.0", Author: "Author"},   // missing Name
		{ID: "test", Name: "Test", Author: "Author"},       // missing Version
		{ID: "test", Name: "Test", Version: "1.0.0"},       // missing Author
	}
	
	for _, m := range invalid {
		if err := validateManifest(m); err == nil {
			t.Errorf("Expected error for invalid manifest: %v", m)
		}
	}
}

// TestValidatePermissions 测试权限验证
func TestValidatePermissions(t *testing.T) {
	valid := &SkillManifest{
		ID:      "test",
		Name:    "Test",
		Version: "1.0.0",
		Author:  "Author",
		Permissions: []SkillPermission{
			{
				Name:     "read_data",
				Required: true,
				Actions:  []string{"read"},
				Resources: []string{"/tmp/*"},
			},
		},
	}
	
	if err := validatePermissions(valid); err != nil {
		t.Errorf("Expected valid permissions, got error: %v", err)
	}
	
	invalid := &SkillManifest{
		ID:      "test",
		Name:    "Test",
		Version: "1.0.0",
		Author:  "Author",
		Permissions: []SkillPermission{
			{
				Name:     "wildcard_perm",
				Required: true,
				Actions:  []string{"*"},
			},
		},
	}
	
	if err := validatePermissions(invalid); err == nil {
		t.Error("Expected error for wildcard permissions")
	}
}

// TestCalculateChecksum 测试 checksum 计算
func TestCalculateChecksum(t *testing.T) {
	data1 := []byte("hello world")
	data2 := []byte("hello world")
	data3 := []byte("different content")
	
	checksum1 := calculateFileChecksumSimple(data1)
	checksum2 := calculateFileChecksumSimple(data2)
	checksum3 := calculateFileChecksumSimple(data3)
	
	if checksum1 != checksum2 {
		t.Error("Same content should produce same checksum")
	}
	
	if checksum1 == checksum3 {
		t.Error("Different content should produce different checksum")
	}
}

// TestPluginLoader 测试插件加载器
func TestPluginLoader(t *testing.T) {
	loader := NewPluginLoader()
	
	if loader == nil {
		t.Fatal("Failed to create plugin loader")
	}
	
	// 测试空列表
	plugins := loader.ListLoadedPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected empty list, got %d plugins", len(plugins))
	}
	
	// 测试获取不存在的插件
	_, err := loader.GetPluginInfo("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent plugin")
	}
}

// TestSandboxConfig 测试沙箱配置
func TestSandboxConfig(t *testing.T) {
	config := DefaultSandboxConfig()
	
	if !config.Enabled {
		t.Error("Default config should have sandbox enabled")
	}
	
	if config.MaxMemoryMB != 512 {
		t.Errorf("Expected default memory limit 512MB, got %d", config.MaxMemoryMB)
	}
	
	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.Timeout)
	}
}

// TestIsPathAllowed 测试路径允许检查
func TestIsPathAllowed(t *testing.T) {
	allowed := []string{"/tmp", "/var/tmp"}
	blocked := []string{"/etc", "/root"}
	
	tests := []struct {
		path     string
		expected bool
	}{
		{"/tmp/data", true},
		{"/var/tmp/file", true},
		{"/etc/passwd", false},
		{"/root/.ssh", false},
		{"/home/user", false}, // 不在 allowed 列表中，所以不允许
	}
	
	for _, test := range tests {
		result := isPathAllowed(test.path, allowed, blocked)
		if result != test.expected {
			t.Errorf("isPathAllowed(%s) = %v, expected %v", test.path, result, test.expected)
		}
	}
}

// TestThirdPartyScannerManager 测试第三方扫描器管理器
func TestThirdPartyScannerManager(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewThirdPartyScannerManager(tmpDir)
	
	if manager == nil {
		t.Fatal("Failed to create scanner manager")
	}
	
	// 测试注册扫描器
	config := &ThirdPartyScannerConfig{
		Name:    "test_scanner",
		Command: "echo",
		Timeout: 10 * time.Second,
	}
	
	if err := manager.RegisterScanner(config); err != nil {
		t.Errorf("Failed to register scanner: %v", err)
	}
	
	// 测试列出扫描器
	scanners := manager.ListScanners()
	if len(scanners) != 1 {
		t.Errorf("Expected 1 scanner, got %d", len(scanners))
	}
	
	// 测试移除扫描器
	if !manager.RemoveScanner("test_scanner") {
		t.Error("Failed to remove scanner")
	}
	
	scanners = manager.ListScanners()
	if len(scanners) != 0 {
		t.Errorf("Expected 0 scanners after removal, got %d", len(scanners))
	}
}

// TestHotReloadConfig 测试热加载配置
func TestHotReloadConfig(t *testing.T) {
	config := HotReloadConfig{
		Enabled:      true,
		WatchDir:     "/tmp/skills",
		PollInterval: 5 * time.Second,
		AutoApprove:  false,
	}
	
	if !config.Enabled {
		t.Error("Config should be enabled")
	}
	
	if config.PollInterval != 5*time.Second {
		t.Errorf("Expected poll interval 5s, got %v", config.PollInterval)
	}
}

// TestSkillPermission 测试技能权限
func TestSkillPermission(t *testing.T) {
	perm := SkillPermission{
		Name:        "filesystem_read",
		Description: "Read files from allowed paths",
		Required:    true,
		Resources:   []string{"/tmp/*", "/var/data/*"},
		Actions:     []string{"read", "list"},
	}
	
	if perm.Name != "filesystem_read" {
		t.Errorf("Expected name filesystem_read, got %s", perm.Name)
	}
	
	if !perm.Required {
		t.Error("Permission should be required")
	}
	
	if len(perm.Actions) != 2 {
		t.Errorf("Expected 2 actions, got %d", len(perm.Actions))
	}
}

// TestSkillDependency 测试技能依赖
func TestSkillDependency(t *testing.T) {
	dep := SkillDependency{
		SkillID:  "base-skill",
		Version:  "^1.0.0",
		Optional: false,
	}
	
	if dep.SkillID != "base-skill" {
		t.Errorf("Expected skill_id base-skill, got %s", dep.SkillID)
	}
	
	if dep.Optional {
		t.Error("Dependency should not be optional")
	}
}

// TestPluginBuilder 测试插件构建器
func TestPluginBuilder(t *testing.T) {
	tmpDir := t.TempDir()
	builder := NewPluginBuilder(tmpDir)
	
	if builder == nil {
		t.Fatal("Failed to create plugin builder")
	}
	
	// 测试创建插件项目
	skillDir, err := builder.CreateFromTemplate("test-skill", "Test Skill", "Test Author")
	if err != nil {
		t.Fatalf("Failed to create plugin: %v", err)
	}
	
	// 验证目录结构
	files := []string{"main.go", "go.mod", "README.md"}
	for _, file := range files {
		path := filepath.Join(skillDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s to exist", file)
		}
	}
	
	// 测试列出插件
	plugins, err := builder.ListPlugins()
	if err != nil {
		t.Fatalf("Failed to list plugins: %v", err)
	}
	
	if len(plugins) != 1 {
		t.Errorf("Expected 1 plugin, got %d", len(plugins))
	}
	
	if plugins[0] != "test-skill" {
		t.Errorf("Expected plugin name test-skill, got %s", plugins[0])
	}
}

// TestCommonScannerConfigs 测试常见扫描器配置
func TestCommonScannerConfigs(t *testing.T) {
	if len(CommonThirdPartyScannerConfigs) == 0 {
		t.Error("Expected common scanner configs to be defined")
	}
	
	expectedScanners := []string{"trufflehog", "semgrep", "bandit", "shellcheck", "clamav"}
	
	for _, name := range expectedScanners {
		if _, exists := CommonThirdPartyScannerConfigs[name]; !exists {
			t.Errorf("Expected scanner %s to be defined", name)
		}
	}
}
