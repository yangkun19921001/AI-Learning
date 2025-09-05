package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"mcp-openai-integration/pkg/chat"
	"mcp-openai-integration/pkg/config"
)

func main() {
	// 解析命令行参数
	var configPath = flag.String("config", "config.yaml", "配置文件路径")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	var interactive = flag.Bool("interactive", true, "交互模式")
	var message = flag.String("message", "", "单次消息模式")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Println("MCP-OpenAI Integration v1.0.0")
		fmt.Println("集成OpenAI API和MCP工具的智能聊天助手")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("MCP-OpenAI Integration - 集成OpenAI API和MCP工具的智能聊天助手")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Printf("  %s [选项]\n", os.Args[0])
		fmt.Println()
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("示例:")
		fmt.Printf("  # 交互模式\n")
		fmt.Printf("  %s -interactive\n", os.Args[0])
		fmt.Println()
		fmt.Printf("  # 单次消息模式\n")
		fmt.Printf("  %s -message \"帮我检查服务器192.168.1.100的磁盘使用情况\"\n", os.Args[0])
		fmt.Println()
		fmt.Printf("  # 指定配置文件\n")
		fmt.Printf("  %s -config /path/to/config.yaml\n", os.Args[0])
		os.Exit(0)
	}

	// 设置日志
	logger := log.New(os.Stdout, "[MCP-OpenAI] ", log.LstdFlags|log.Lshortfile)

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		logger.Fatalf("配置验证失败: %v", err)
	}

	logger.Printf("配置加载完成，启用MCP: %v", cfg.Chat.EnableMCP)

	// 创建聊天引擎
	engine, err := chat.NewChatEngine(cfg, logger)
	if err != nil {
		logger.Fatalf("创建聊天引擎失败: %v", err)
	}

	// 启动聊天引擎
	if err := engine.Start(); err != nil {
		logger.Fatalf("启动聊天引擎失败: %v", err)
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动信号处理协程
	go func() {
		<-sigChan
		logger.Println("收到退出信号，正在关闭...")
		cancel()
		if err := engine.Close(); err != nil {
			logger.Printf("关闭聊天引擎失败: %v", err)
		}
		os.Exit(0)
	}()

	// 根据模式运行
	if *message != "" {
		// 单次消息模式
		runSingleMessage(ctx, engine, *message, logger)
	} else if *interactive {
		// 交互模式
		runInteractiveMode(ctx, engine, logger)
	} else {
		logger.Println("请指定运行模式：-interactive 或 -message")
		os.Exit(1)
	}
}

// runSingleMessage 运行单次消息模式
func runSingleMessage(ctx context.Context, engine *chat.ChatEngine, message string, logger *log.Logger) {
	logger.Printf("处理单次消息: %s", message)

	// 设置超时
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// 发送消息
	response, err := engine.Chat(ctx, message)
	if err != nil {
		logger.Fatalf("处理消息失败: %v", err)
	}

	// 输出响应
	fmt.Println("\n=== AI 响应 ===")
	fmt.Println(response)
}

// runInteractiveMode 运行交互模式
func runInteractiveMode(ctx context.Context, engine *chat.ChatEngine, logger *log.Logger) {
	fmt.Println("\n=== MCP-OpenAI 智能助手 ===")
	fmt.Println("集成了MCP工具的OpenAI聊天助手")
	fmt.Printf("可用工具: %v\n", engine.GetAvailableTools())
	fmt.Println("输入 '/help' 查看帮助，输入 '/quit' 退出")
	fmt.Println("=" + strings.Repeat("=", 50))

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n你: ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// 处理特殊命令
		if strings.HasPrefix(input, "/") {
			if handleCommand(input, engine, logger) {
				break
			}
			continue
		}

		// 处理普通消息
		fmt.Print("AI: ")

		// 设置消息超时
		msgCtx, cancel := context.WithTimeout(ctx, 120*time.Second)

		response, err := engine.Chat(msgCtx, input)
		cancel()

		if err != nil {
			fmt.Printf("❌ 处理消息失败: %v\n", err)
			continue
		}

		fmt.Println(response)
	}

	if err := scanner.Err(); err != nil {
		logger.Printf("读取输入失败: %v", err)
	}
}

// handleCommand 处理特殊命令
func handleCommand(command string, engine *chat.ChatEngine, logger *log.Logger) bool {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return false
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help", "/h":
		showInteractiveHelp()
	case "/quit", "/exit", "/q":
		fmt.Println("再见！")
		return true
	case "/tools", "/t":
		showAvailableTools(engine)
	case "/clear", "/c":
		engine.ClearHistory()
		fmt.Println("✅ 对话历史已清空")
	case "/history", "/hist":
		showMessageHistory(engine)
	case "/save", "/s":
		filename := fmt.Sprintf("conversation_%s.json", time.Now().Format("20060102_150405"))
		if err := engine.SaveConversation(filename); err != nil {
			fmt.Printf("❌ 保存对话失败: %v\n", err)
		} else {
			fmt.Printf("✅ 对话已保存到: %s\n", filename)
		}
	default:
		fmt.Printf("❌ 未知命令: %s，输入 '/help' 查看帮助\n", command)
	}

	return false
}

// showInteractiveHelp 显示交互模式帮助
func showInteractiveHelp() {
	fmt.Println("\n=== 交互模式帮助 ===")
	fmt.Println("可用命令:")
	fmt.Println("  /help, /h      - 显示此帮助信息")
	fmt.Println("  /tools, /t     - 显示可用工具")
	fmt.Println("  /clear, /c     - 清空对话历史")
	fmt.Println("  /history, /hist - 显示消息历史")
	fmt.Println("  /save, /s      - 保存当前对话")
	fmt.Println("  /quit, /q      - 退出程序")
	fmt.Println()
	fmt.Println("功能说明:")
	fmt.Println("  • 支持自然语言对话")
	fmt.Println("  • 自动调用MCP工具执行SSH命令")
	fmt.Println("  • 支持多轮对话和上下文记忆")
	fmt.Println("  • 可以执行复杂的系统管理任务")
	fmt.Println()
	fmt.Println("示例对话:")
	fmt.Println("  你: 帮我检查服务器192.168.1.100的磁盘使用情况")
	fmt.Println("  AI: 我来帮你检查服务器的磁盘使用情况...")
	fmt.Println("      [自动调用ssh_execute工具执行df -h命令]")
}

// showAvailableTools 显示可用工具
func showAvailableTools(engine *chat.ChatEngine) {
	tools := engine.GetAvailableTools()

	fmt.Println("\n=== 可用工具 ===")
	if len(tools) == 0 {
		fmt.Println("❌ 没有可用的工具")
		return
	}

	for i, tool := range tools {
		fmt.Printf("%d. %s\n", i+1, tool)
	}

	fmt.Printf("\n总计: %d 个工具\n", len(tools))
}

// showMessageHistory 显示消息历史
func showMessageHistory(engine *chat.ChatEngine) {
	messages := engine.GetMessageHistory()

	fmt.Println("\n=== 消息历史 ===")
	if len(messages) == 0 {
		fmt.Println("暂无消息历史")
		return
	}

	for i, msg := range messages {
		role := msg.Role
		switch role {
		case "system":
			role = "系统"
		case "user":
			role = "用户"
		case "assistant":
			role = "助手"
		case "tool":
			role = "工具"
		}

		content := msg.Content
		if len(content) > 100 {
			content = content[:100] + "..."
		}

		fmt.Printf("%d. [%s] %s\n", i+1, role, content)
	}

	fmt.Printf("\n总计: %d 条消息\n", len(messages))
}
