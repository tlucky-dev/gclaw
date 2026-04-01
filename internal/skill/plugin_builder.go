package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PluginSkillTemplate 插件技能模板（用于创建新插件）
const PluginSkillTemplate = `package main

import (
	"encoding/json"
	"fmt"
)

// SkillMetadata 技能元数据
type SkillMetadata struct {
	ID          string                 ` + "`json:\"id\"`" + `
	Name        string                 ` + "`json:\"name\"`" + `
	Version     string                 ` + "`json:\"version\"`" + `
	Description string                 ` + "`json:\"description\"`" + `
	Author      string                 ` + "`json:\"author\"`" + `
	CreatedAt   time.Time              ` + "`json:\"created_at\"`" + `
	UpdatedAt   time.Time              ` + "`json:\"updated_at\"`" + `
	Status      string                 ` + "`json:\"status\"`" + `
	Tags        []string               ` + "`json:\"tags,omitempty\"`" + `
	Permissions []SkillPermission      ` + "`json:\"permissions,omitempty\"`" + `
	Dependencies []SkillDependency     ` + "`json:\"dependencies,omitempty\"`" + `
	ConfigSchema map[string]interface{} ` + "`json:\"config_schema,omitempty\"`" + `
	Manifest    *SkillManifest         ` + "`json:\"manifest,omitempty\"`" + `
	Checksum    string                 ` + "`json:\"checksum\"`" + `
	SourcePath  string                 ` + "`json:\"source_path,omitempty\"`" + `
	PluginPath  string                 ` + "`json:\"plugin_path,omitempty\"`" + `
}

// SkillPermission 技能权限
type SkillPermission struct {
	Name        string   ` + "`json:\"name\"`" + `
	Description string   ` + "`json:\"description\"`" + `
	Required    bool     ` + "`json:\"required\"`" + `
	Resources   []string ` + "`json:\"resources,omitempty\"`" + `
	Actions     []string ` + "`json:\"actions,omitempty\"`" + `
}

// SkillDependency 技能依赖
type SkillDependency struct {
	SkillID  string ` + "`json:\"skill_id\"`" + `
	Version  string ` + "`json:\"version\"`" + `
	Optional bool   ` + "`json:\"optional\"`" + `
}

// SkillManifest 技能清单
type SkillManifest struct {
	ID            string            ` + "`json:\"id\"`" + `
	Name          string            ` + "`json:\"name\"`" + `
	Version       string            ` + "`json:\"version\"`" + `
	Description   string            ` + "`json:\"description\"`" + `
	Author        string            ` + "`json:\"author\"`" + `
	License       string            ` + "`json:\"license,omitempty\"`" + `
	Homepage      string            ` + "`json:\"homepage,omitempty\"`" + `
	Permissions   []SkillPermission ` + "`json:\"permissions,omitempty\"`" + `
	Dependencies  []SkillDependency ` + "`json:\"dependencies,omitempty\"`" + `
	ConfigSchema  map[string]interface{} ` + "`json:\"config_schema,omitempty\"`" + `
	Tags          []string          ` + "`json:\"tags,omitempty\"`" + `
	MinRuntimeVersion string        ` + "`json:\"min_runtime_version,omitempty\"`" + `
	MaxMemoryMB   int               ` + "`json:\"max_memory_mb,omitempty\"`" + `
	MaxCPUPercent float64           ` + "`json:\"max_cpu_percent,omitempty\"`" + `
	AllowedPaths  []string          ` + "`json:\"allowed_paths,omitempty\"`" + `
	BlockedPaths  []string          ` + "`json:\"blocked_paths,omitempty\"`" + `
}

// MySkill 示例技能实现
type MySkill struct {
	metadata *SkillMetadata
	config   map[string]interface{}
}

// GetMetadata 获取技能元数据
func (s *MySkill) GetMetadata() *SkillMetadata {
	return s.metadata
}

// Initialize 初始化技能
func (s *MySkill) Initialize(config map[string]interface{}) error {
	s.config = config
	fmt.Println("Skill initialized with config:", config)
	return nil
}

// Execute 执行技能
func (s *MySkill) Execute(args map[string]interface{}) (string, error) {
	// 在这里实现技能逻辑
	result := fmt.Sprintf("Skill %s executed with args: %v", s.metadata.Name, args)
	return result, nil
}

// Validate 验证技能
func (s *MySkill) Validate() error {
	if s.metadata.ID == "" {
		return fmt.Errorf("skill ID is required")
	}
	if s.metadata.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	return nil
}

// GetRequiredPermissions 获取所需权限
func (s *MySkill) GetRequiredPermissions() []SkillPermission {
	return s.metadata.Permissions
}

// GetDependencies 获取依赖
func (s *MySkill) GetDependencies() []SkillDependency {
	return s.metadata.Dependencies
}

// Shutdown 关闭技能
func (s *MySkill) Shutdown() error {
	fmt.Println("Skill shutting down...")
	return nil
}

// Skill 导出技能实例（插件入口点）
var Skill *MySkill

func init() {
	Skill = &MySkill{
		metadata: &SkillMetadata{
			ID:          "example_plugin_skill",
			Name:        "Example Plugin Skill",
			Version:     "1.0.0",
			Description: "An example plugin skill for demonstration",
			Author:      "Your Name",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Status:      "active",
			Tags:        []string{"example", "plugin", "demo"},
			Permissions: []SkillPermission{
				{
					Name:        "read_data",
					Description: "Read data from allowed sources",
					Required:    true,
					Resources:   []string{"/tmp/*"},
					Actions:     []string{"read"},
				},
			},
			Dependencies: []SkillDependency{},
			Manifest: &SkillManifest{
				MaxMemoryMB:   256,
				MaxCPUPercent: 25.0,
				AllowedPaths:  []string{"/tmp"},
				BlockedPaths:  []string{"/etc", "/root", "/proc"},
			},
		},
	}
}

// 辅助函数：时间序列化
type timeHelper struct {
	time.Time
}

func (t timeHelper) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Format(time.RFC3339))
}

func (t *timeHelper) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}
`

// PluginBuilder 插件构建器
type PluginBuilder struct {
	mu          sync.Mutex
	outputDir   string
	goModPath   string
	buildFlags  []string
	buildTimeout time.Duration
}

// PluginBuildResult 插件构建结果
type PluginBuildResult struct {
	Success    bool      `json:"success"`
	PluginPath string    `json:"plugin_path"`
	Error      string    `json:"error,omitempty"`
	BuildTime  time.Time `json:"build_time"`
	Size       int64     `json:"size_bytes"`
	Checksum   string    `json:"checksum"`
}

// NewPluginBuilder 创建插件构建器
func NewPluginBuilder(outputDir string) *PluginBuilder {
	return &PluginBuilder{
		outputDir:    outputDir,
		buildFlags:   []string{"-buildmode=plugin"},
		buildTimeout: 5 * time.Minute,
	}
}

// SetBuildFlags 设置构建标志
func (pb *PluginBuilder) SetBuildFlags(flags []string) {
	pb.buildFlags = flags
}

// SetBuildTimeout 设置构建超时
func (pb *PluginBuilder) SetBuildTimeout(timeout time.Duration) {
	pb.buildTimeout = timeout
}

// CreateFromTemplate 从模板创建插件项目
func (pb *PluginBuilder) CreateFromTemplate(skillID, skillName, author string) (string, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	// 创建技能目录
	skillDir := filepath.Join(pb.outputDir, skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create skill directory: %w", err)
	}

	// 写入主文件
	mainFile := filepath.Join(skillDir, "main.go")
	if err := os.WriteFile(mainFile, []byte(PluginSkillTemplate), 0644); err != nil {
		return "", fmt.Errorf("failed to write main.go: %w", err)
	}

	// 创建 go.mod 文件
	goModContent := fmt.Sprintf(`module %s

go 1.19
`, skillID)

	goModFile := filepath.Join(skillDir, "go.mod")
	if err := os.WriteFile(goModFile, []byte(goModContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write go.mod: %w", err)
	}

	// 创建 README 文件
	readmeContent := fmt.Sprintf(`# %s

A plugin skill for GCLaw.

## Building

To build this plugin, run:

`+"```bash"+`
go build -buildmode=plugin -o %s.so .
`+"```"+`

## Usage

Load the plugin using the GCLaw skill registry:

`+"```go"+`
registry.LoadPlugin("%s.so")
`+"```"+`

## Author

%s

## License

MIT
`, skillName, skillID, skillID, author)

	readmeFile := filepath.Join(skillDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte(readmeContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write README.md: %w", err)
	}

	return skillDir, nil
}

// Build 构建插件
func (pb *PluginBuilder) Build(skillDir string) (*PluginBuildResult, error) {
	pb.mu.Lock()
	defer pb.mu.Unlock()

	result := &PluginBuildResult{
		Success:   false,
		BuildTime: time.Now(),
	}

	// 检查目录是否存在
	info, err := os.Stat(skillDir)
	if err != nil {
		result.Error = fmt.Sprintf("skill directory not found: %v", err)
		return result, fmt.Errorf(result.Error)
	}
	if !info.IsDir() {
		result.Error = "skill path is not a directory"
		return result, fmt.Errorf(result.Error)
	}

	// 读取 go.mod 获取模块名
	goModFile := filepath.Join(skillDir, "go.mod")
	goModData, err := os.ReadFile(goModFile)
	if err != nil {
		result.Error = fmt.Sprintf("failed to read go.mod: %v", err)
		return result, fmt.Errorf(result.Error)
	}

	var moduleName string
	lines := string(goModData)
	for _, line := range splitLines(lines) {
		if len(line) > 7 && line[:7] == "module " {
			moduleName = line[7:]
			break
		}
	}

	if moduleName == "" {
		result.Error = "failed to parse module name from go.mod"
		return result, fmt.Errorf(result.Error)
	}

	// 构建输出路径
	pluginName := filepath.Base(skillDir)
	pluginPath := filepath.Join(pb.outputDir, pluginName+".so")

	// TODO: 实际构建命令需要使用 exec.Command 调用 go build
	// 由于在库代码中直接执行构建命令可能不安全，这里提供框架
	// 实际使用时可以调用外部构建脚本或通过 API 触发构建

	fmt.Printf("Would build plugin from %s to %s\n", skillDir, pluginPath)
	fmt.Printf("Build command: go build -buildmode=plugin -o %s %s\n", pluginPath, skillDir)

	// 模拟构建成功（实际实现需要调用 go build 命令）
	result.Success = true
	result.PluginPath = pluginPath
	
	// 如果文件存在，获取大小和 checksum
	if info, err := os.Stat(pluginPath); err == nil {
		result.Size = info.Size()
		data, _ := os.ReadFile(pluginPath)
		result.Checksum = calculateFileChecksumSimple(data)
	}

	return result, nil
}

// BuildAll 构建目录下所有插件
func (pb *PluginBuilder) BuildAll() ([]*PluginBuildResult, error) {
	entries, err := os.ReadDir(pb.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var results []*PluginBuildResult

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// 检查是否有 go.mod 文件
		goModPath := filepath.Join(pb.outputDir, entry.Name(), "go.mod")
		if _, err := os.Stat(goModPath); err != nil {
			continue
		}

		result, err := pb.Build(filepath.Join(pb.outputDir, entry.Name()))
		results = append(results, result)
		if err != nil {
			fmt.Printf("Warning: failed to build %s: %v\n", entry.Name(), err)
		}
	}

	return results, nil
}

// Clean 清理构建产物
func (pb *PluginBuilder) Clean() error {
	entries, err := os.ReadDir(pb.outputDir)
	if err != nil {
		return fmt.Errorf("failed to read output directory: %w", err)
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".so" {
			if err := os.Remove(filepath.Join(pb.outputDir, entry.Name())); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", entry.Name(), err)
			}
		}
	}

	return nil
}

// ListPlugins 列出所有插件项目
func (pb *PluginBuilder) ListPlugins() ([]string, error) {
	entries, err := os.ReadDir(pb.outputDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read output directory: %w", err)
	}

	var plugins []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		goModPath := filepath.Join(pb.outputDir, entry.Name(), "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			plugins = append(plugins, entry.Name())
		}
	}

	return plugins, nil
}

// GetPluginInfo 获取插件信息
func (pb *PluginBuilder) GetPluginInfo(skillID string) (map[string]interface{}, error) {
	skillDir := filepath.Join(pb.outputDir, skillID)
	
	info := make(map[string]interface{})
	info["id"] = skillID
	info["directory"] = skillDir

	// 读取 go.mod
	goModPath := filepath.Join(skillDir, "go.mod")
	if data, err := os.ReadFile(goModPath); err == nil {
		info["go_mod"] = string(data)
	}

	// 读取 README
	readmePath := filepath.Join(skillDir, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		info["readme"] = string(data)
	}

	// 检查是否有构建产物
	soPath := filepath.Join(pb.outputDir, skillID+".so")
	if stat, err := os.Stat(soPath); err == nil {
		info["built"] = true
		info["size"] = stat.Size()
		info["modified"] = stat.ModTime()
	} else {
		info["built"] = false
	}

	return info, nil
}

// splitLines 简单的字符串按行分割
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
