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

func main() {
	// 解析命令行参数
	configFile := flag.String("config", "", "Path to config file")
	apiKey := flag.String("api-key", "", "API key for LLM provider")
	model := flag.String("model", "", "Model name to use")
	providerName := flag.String("provider", "openai", "LLM provider name")
	sessionID := flag.String("session", "default", "Session ID")
	interactive := flag.Bool("i", false, "Interactive mode")
	enableFeishu := flag.Bool("feishu", false, "Enable Feishu adapter")
	flag.Parse()

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

	// 检查 API Key
	if cfg.Provider.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: API key is required. Use --api-key or set in config file.")
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
