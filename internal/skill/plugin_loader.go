package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"
)

// PluginLoader 插件加载器
type PluginLoader struct {
	mu          sync.Mutex
	loadedPlugins map[string]*plugin.Plugin
	pluginInfo    map[string]*PluginInfo
	loadTimeout   time.Duration
}

// PluginInfo 插件信息
type PluginInfo struct {
	Path        string    `json:"path"`
	LoadedAt    time.Time `json:"loaded_at"`
	Checksum    string    `json:"checksum"`
	Size        int64     `json:"size"`
	Version     string    `json:"version"`
	SymbolNames []string  `json:"symbol_names"`
}

// NewPluginLoader 创建新的插件加载器
func NewPluginLoader() *PluginLoader {
	return &PluginLoader{
		loadedPlugins: make(map[string]*plugin.Plugin),
		pluginInfo:    make(map[string]*PluginInfo),
		loadTimeout:   30 * time.Second,
	}
}

// LoadPluginSymbol 从已加载的插件中查找符号
func (pl *PluginLoader) LoadPluginSymbol(pluginID, symbolName string) (plugin.Symbol, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	p, exists := pl.loadedPlugins[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' not loaded", pluginID)
	}

	sym, err := p.Lookup(symbolName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup symbol '%s': %w", symbolName, err)
	}

	return sym, nil
}

// GetPluginInfo 获取插件信息
func (pl *PluginLoader) GetPluginInfo(pluginID string) (*PluginInfo, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	info, exists := pl.pluginInfo[pluginID]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' not found", pluginID)
	}

	return info, nil
}

// ListLoadedPlugins 列出所有已加载的插件
func (pl *PluginLoader) ListLoadedPlugins() []*PluginInfo {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	result := make([]*PluginInfo, 0, len(pl.pluginInfo))
	for _, info := range pl.pluginInfo {
		result = append(result, info)
	}
	return result
}

// UnloadPlugin 卸载插件（注意：Go 的 plugin 包不支持卸载，此方法仅清理元数据）
func (pl *PluginLoader) UnloadPlugin(pluginID string) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if _, exists := pl.loadedPlugins[pluginID]; !exists {
		return fmt.Errorf("plugin '%s' not loaded", pluginID)
	}

	// 注意：Go 的 plugin 包不支持真正的卸载
	// 这里只清理元数据，实际的 .so 文件仍然在内存中
	delete(pl.loadedPlugins, pluginID)
	delete(pl.pluginInfo, pluginID)

	return nil
}

// SetLoadTimeout 设置加载超时时间
func (pl *PluginLoader) SetLoadTimeout(timeout time.Duration) {
	pl.loadTimeout = timeout
}

// loadPluginWithTimeout 带超时的插件加载
func (pl *PluginLoader) loadPluginWithTimeout(path string) (*plugin.Plugin, error) {
	done := make(chan struct{})
	var p *plugin.Plugin
	var loadErr error

	go func() {
		defer close(done)
		p, loadErr = plugin.Open(path)
	}()

	select {
	case <-done:
		return p, loadErr
	case <-time.After(pl.loadTimeout):
		return nil, fmt.Errorf("plugin loading timeout after %v", pl.loadTimeout)
	}
}

// ValidatePluginFile 验证插件文件
func ValidatePluginFile(pluginPath string) error {
	// 检查文件是否存在
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		return fmt.Errorf("plugin file does not exist: %s", pluginPath)
	}

	// 检查文件扩展名
	ext := filepath.Ext(pluginPath)
	if ext != ".so" {
		return fmt.Errorf("invalid plugin file extension: %s (expected .so)", ext)
	}

	// 检查文件大小（防止空文件或异常大的文件）
	info, err := os.Stat(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to stat plugin file: %w", err)
	}

	size := info.Size()
	if size == 0 {
		return fmt.Errorf("plugin file is empty")
	}
	if size > 100*1024*1024 { // 限制 100MB
		return fmt.Errorf("plugin file is too large: %d bytes", size)
	}

	// 检查文件权限
	mode := info.Mode()
	if mode&0111 == 0 { // 没有执行权限
		return fmt.Errorf("plugin file lacks execute permission")
	}

	return nil
}

// GetPluginSymbols 获取插件中可用的符号列表（通过预定义符号探测）
func GetPluginSymbols(p *plugin.Plugin) []string {
	// Go 的 plugin 包不提供直接列出所有符号的方法
	// 这里通过尝试查找预定义符号来探测
	commonSymbols := []string{
		"Skill", "NewSkill", "Init", "Execute", "Validate",
		"Metadata", "Manifest", "Version", "Name",
	}

	var found []string
	for _, sym := range commonSymbols {
		if _, err := p.Lookup(sym); err == nil {
			found = append(found, sym)
		}
	}

	return found
}
