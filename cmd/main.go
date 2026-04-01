package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"gclaw/internal/adapters/feishu"
	"gclaw/internal/config"
	"gclaw/internal/engine"
	"gclaw/internal/memory"
	"gclaw/internal/provider"
	"gclaw/internal/tools"
	"gclaw/pkg/types"
)

const (
	Version = "1.2.0"
	AppName = "gclaw"
)

func main() {
	// 定义命令行标志
	version := flag.Bool("version", false, "Show version information")
	initConfig := flag.String("init", "", "Initialize config file for provider (openai|modelscope|azure)")
	configFile := flag.String("config", "", "Path to config file")
	apiKey := flag.String("api-key", "", "API key for LLM provider")
	model := flag.String("model", "", "Model name to use")
	providerName := flag.String("provider", "openai", "LLM provider name")
	sessionID := flag.String("session", "default", "Session ID")
	interactive := flag.Bool("i", false, "Interactive mode")
	enableFeishu := flag.Bool("feishu", false, "Enable Feishu adapter")
	
	// 沙箱相关参数
	sandboxLevel := flag.String("sandbox-level", "", "Sandbox isolation level (none|basic|standard|strict)")
	sandboxDryRun := flag.Bool("sandbox-dryrun", false, "Enable sandbox dry-run mode (simulate without execution)")
	
	// 引擎参数
	maxIterations := flag.Int("max-iterations", 0, "Maximum iterations for engine")
	temperature := flag.Float64("temperature", 0, "Temperature for LLM generation")
	maxTokens := flag.Int("max-tokens", 0, "Maximum tokens for response")
	
	flag.Parse()

	// 显示版本信息
	if *version {
		printVersion()
		return
	}

	// 初始化配置文件
	if *initConfig != "" {
		if err := initConfigFile(*initConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 加载配置
	var cfg *config.Config
	var err error

	if *configFile != "" {
		cfg, err = config.LoadFromFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		cfg = config.DefaultConfig()
	}

	// 覆盖命令行参数
	if *apiKey != "" {
		cfg.Provider.APIKey = *apiKey
	}
	if *model != "" {
		cfg.Provider.Model = *model
	}
	if *providerName != "" {
		cfg.Provider.Name = *providerName
	}
	if *sandboxLevel != "" {
		cfg.Sandbox.Level = *sandboxLevel
	}
	if *sandboxDryRun {
		cfg.Sandbox.DryRun = true
	}
	if *maxIterations > 0 {
		cfg.Engine.MaxIterations = *maxIterations
	}
	if *temperature > 0 {
		cfg.Engine.Temperature = *temperature
	}
	if *maxTokens > 0 {
		cfg.Engine.MaxTokens = *maxTokens
	}

	// 检查 API Key
	if cfg.Provider.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key is required. Use --api-key or set in config file.")
		fmt.Fprintln(os.Stderr, "Tip: Run 'gclaw -init openai' to create a config file interactively.")
		os.Exit(1)
	}

	// 创建提供商
	p, err := provider.CreateProvider(
		cfg.Provider.Name,
		cfg.Provider.APIKey,
		cfg.Provider.BaseURL,
		cfg.Provider.Model,
		cfg.Provider.Timeout,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating provider: %v\n", err)
		os.Exit(1)
	}

	// 创建内存存储
	m := memory.NewInMemoryMemory(cfg.Memory.MaxSize)

	// 创建工具注册表并注册默认工具
	toolRegistry := tools.NewToolRegistry()
	toolRegistry.Register(tools.NewShellTool())
	toolRegistry.Register(tools.NewFileReadTool())
	toolRegistry.Register(tools.NewFileWriteTool())
	toolRegistry.Register(tools.NewSearchTool())
	toolRegistry.Register(tools.NewFeishuTool()) // 注册飞书工具

	// 创建引擎
	eng := engine.NewGCLawEngine(
		p,
		m,
		toolRegistry,
		cfg.Engine.MaxIterations,
		cfg.Engine.Temperature,
		cfg.Engine.MaxTokens,
	)

	// 创建上下文，支持优雅退出
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n收到退出信号，正在关闭...")
		cancel()
	}()

	// 启动飞书适配器（如果启用）
	if *enableFeishu || (cfg.Adapters.Feishu != nil && cfg.Adapters.Feishu.Enabled) {
		if err := startFeishuAdapter(ctx, cfg, eng); err != nil {
			fmt.Fprintf(os.Stderr, "启动飞书适配器失败：%v\n", err)
			os.Exit(1)
		}
	}

	// 处理输入
	if *interactive {
		runInteractiveMode(eng, *sessionID)
	} else if !*enableFeishu && (cfg.Adapters.Feishu == nil || !cfg.Adapters.Feishu.Enabled) {
		runSingleMode(eng, *sessionID)
	} else {
		// 仅飞书模式，等待信号
		fmt.Println("gclaw 运行在飞书模式下。按 Ctrl+C 退出。")
		<-ctx.Done()
	}
}

// printVersion 打印版本信息
func printVersion() {
	fmt.Printf("%s version %s\n", AppName, Version)
	fmt.Println("A secure AI agent framework with enhanced sandbox capabilities")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gclaw [options]")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("  gclaw -init openai              # Initialize OpenAI config")
	fmt.Println("  gclaw -i -config config.json    # Run in interactive mode")
	fmt.Println()
	fmt.Println("Options:")
	flag.PrintDefaults()
}

// initConfigFile 交互式初始化配置文件
func initConfigFile(providerType string) error {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Printf("=== GCLaw Configuration Wizard (%s) ===\n\n", providerType)
	
	var cfg config.Config
	cfg = *config.DefaultConfig()
	cfg.Provider.Name = providerType
	
	// 根据提供商类型设置默认值
	switch providerType {
	case "openai":
		cfg.Provider.BaseURL = "https://api.openai.com/v1"
		cfg.Provider.Model = "gpt-3.5-turbo"
		fmt.Println("Default model: gpt-3.5-turbo")
	case "modelscope":
		cfg.Provider.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		cfg.Provider.Model = "qwen-turbo"
		fmt.Println("Default model: qwen-turbo")
	case "azure":
		cfg.Provider.BaseURL = "https://YOUR_RESOURCE.openai.azure.com"
		cfg.Provider.Model = "gpt-35-turbo"
		fmt.Println("Default model: gpt-35-turbo")
	default:
		return fmt.Errorf("unsupported provider: %s", providerType)
	}
	
	fmt.Println()
	
	// API Key
	fmt.Print("Enter your API Key: ")
	apiKey, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	cfg.Provider.APIKey = strings.TrimSpace(apiKey)
	if cfg.Provider.APIKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}
	
	// Model
	fmt.Printf("Enter model name [%s]: ", cfg.Provider.Model)
	model, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	model = strings.TrimSpace(model)
	if model != "" {
		cfg.Provider.Model = model
	}
	
	// Base URL
	fmt.Printf("Enter base URL [%s]: ", cfg.Provider.BaseURL)
	baseURL, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	baseURL = strings.TrimSpace(baseURL)
	if baseURL != "" {
		cfg.Provider.BaseURL = baseURL
	}
	
	// Sandbox Level
	fmt.Println("\n--- Sandbox Configuration ---")
	fmt.Println("Isolation levels: none | basic | standard | strict")
	fmt.Printf("Enter sandbox level [standard]: ")
	sandboxLevel, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	sandboxLevel = strings.TrimSpace(sandboxLevel)
	if sandboxLevel != "" {
		cfg.Sandbox.Level = sandboxLevel
	} else {
		cfg.Sandbox.Level = "standard"
	}
	
	// Dry Run Mode
	fmt.Print("Enable dry-run mode? (y/N): ")
	dryRun, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	cfg.Sandbox.DryRun = strings.ToLower(strings.TrimSpace(dryRun)) == "y"
	
	// Engine Settings
	fmt.Println("\n--- Engine Configuration ---")
	fmt.Printf("Enter max iterations [%d]: ", cfg.Engine.MaxIterations)
	maxIter, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	maxIter = strings.TrimSpace(maxIter)
	if maxIter != "" {
		fmt.Sscanf(maxIter, "%d", &cfg.Engine.MaxIterations)
	}
	
	fmt.Printf("Enter temperature [%.2f]: ", cfg.Engine.Temperature)
	temp, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	temp = strings.TrimSpace(temp)
	if temp != "" {
		fmt.Sscanf(temp, "%f", &cfg.Engine.Temperature)
	}
	
	// Generate filename
	filename := fmt.Sprintf("config.%s.json", providerType)
	fmt.Printf("\nSave to file [%s]: ", filename)
	saveFilename, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	saveFilename = strings.TrimSpace(saveFilename)
	if saveFilename != "" {
		filename = saveFilename
	}
	
	// Save config
	if err := config.SaveToFile(&cfg, filename); err != nil {
		return err
	}
	
	fmt.Printf("\n✓ Configuration saved to: %s\n", filename)
	fmt.Println("\nTo run gclaw with this config:")
	fmt.Printf("  gclaw -i -config %s\n", filename)
	
	return nil
}

// startFeishuAdapter 启动飞书适配器
func startFeishuAdapter(ctx context.Context, cfg *config.Config, eng engine.Engine) error {
	if cfg.Adapters.Feishu == nil {
		return fmt.Errorf("飞书配置未找到")
	}

	feishuCfg := cfg.Adapters.Feishu
	if !feishuCfg.Enabled {
		return nil
	}

	// 验证必要配置
	if feishuCfg.AppID == "" || feishuCfg.AppSecret == "" {
		return fmt.Errorf("飞书 app_id 和 app_secret 不能为空")
	}

	// 创建飞书适配器
	adapter := feishu.NewAdapter(feishu.Config{
		AppID:             feishuCfg.AppID,
		AppSecret:         feishuCfg.AppSecret,
		EncryptKey:        feishuCfg.EncryptKey,
		VerificationToken: feishuCfg.VerificationToken,
	})

	// 创建 HTTP 服务器
	serverAddr := feishuCfg.ServerAddr
	if serverAddr == "" {
		serverAddr = ":8080"
	}
	
	server := feishu.NewServer(adapter, serverAddr)

	// 启动消息处理循环
	go server.ProcessMessages(ctx, func(msg types.Message) error {
		// 使用引擎处理消息
		response, err := eng.Run(msg.ConversationID, msg.Content)
		if err != nil {
			fmt.Printf("处理消息错误：%v\n", err)
			return err
		}

		// 发送回复到飞书
		if err := adapter.SendMessage(msg.ConversationID, response.Message.Content); err != nil {
			fmt.Printf("发送回复失败：%v\n", err)
			return err
		}

		fmt.Printf("已回复飞书消息：%s -> %s\n", msg.ConversationID, response.Message.Content)
		return nil
	})

	// 启动 HTTP 服务器（阻塞）
	go func() {
		if err := server.Start(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "飞书 HTTP 服务器错误：%v\n", err)
		}
	}()

	fmt.Println("飞书适配器已启动")
	return nil
}

func runInteractiveMode(eng engine.Engine, sessionID string) {
	fmt.Println("gclaw interactive mode. Type 'exit' or 'quit' to end.")
	fmt.Println("Type 'reset' to clear conversation history.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
			break
		}

		if strings.ToLower(input) == "reset" {
			err := eng.Reset(sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error resetting session: %v\n", err)
			} else {
				fmt.Println("Session reset.")
			}
			continue
		}

		response, err := eng.Run(sessionID, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			continue
		}

		fmt.Println(response.Message.Content)
		fmt.Println()
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

func runSingleMode(eng engine.Engine, sessionID string) {
	// 从标准输入读取单行输入
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "No input provided")
		os.Exit(1)
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		fmt.Fprintln(os.Stderr, "Empty input")
		os.Exit(1)
	}

	response, err := eng.Run(sessionID, input)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(response.Message.Content)
}
