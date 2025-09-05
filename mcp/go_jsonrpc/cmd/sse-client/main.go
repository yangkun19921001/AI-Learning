package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"ssh-mcp-go-jsonrpc/pkg/client"
)

func main() {
	// 解析命令行参数
	var serverURL = flag.String("server", "http://localhost:8000", "MCP服务器URL")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	var mode = flag.String("mode", "demo", "运行模式: demo, interactive, call")
	var toolName = flag.String("tool", "", "要调用的工具名称（call模式）")
	var toolArgs = flag.String("args", "{}", "工具参数JSON（call模式）")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Println("SSH MCP Client (HTTP SSE) v1.0.0")
		fmt.Println("基于HTTP SSE传输的MCP SSH客户端")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("SSH MCP Client (HTTP SSE) - 基于HTTP SSE传输的MCP SSH客户端")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Printf("  %s [选项]\n", os.Args[0])
		fmt.Println()
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("运行模式:")
		fmt.Println("  demo        - 演示模式，展示基本功能")
		fmt.Println("  interactive - 交互模式，手动输入命令")
		fmt.Println("  call        - 直接调用指定工具")
		fmt.Println()
		fmt.Println("示例:")
		fmt.Printf("  %s -server http://localhost:8000 -mode demo\n", os.Args[0])
		fmt.Printf("  %s -server http://localhost:8000 -mode call -tool ssh_execute -args '{\"host\":\"localhost\",\"command\":\"uptime\"}'\n", os.Args[0])
		os.Exit(0)
	}

	// 创建上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 创建SSE客户端
	sseClient := client.NewSSEClient(*serverURL)

	// 连接到服务器
	fmt.Printf("连接到MCP服务器: %s\n", *serverURL)
	if err := sseClient.Connect(); err != nil {
		log.Fatalf("连接MCP服务器失败: %v", err)
	}
	defer sseClient.Close()

	// 初始化MCP连接
	if err := sseClient.Initialize(); err != nil {
		log.Fatalf("初始化MCP连接失败: %v", err)
	}

	fmt.Printf("成功连接到MCP服务器\n")

	// 根据模式运行
	switch *mode {
	case "demo":
		runDemo(ctx, sseClient)
	case "interactive":
		runInteractive(ctx, sseClient)
	case "call":
		runDirectCall(ctx, sseClient, *toolName, *toolArgs)
	default:
		log.Fatalf("未知运行模式: %s", *mode)
	}
}

// runDemo 运行演示模式
func runDemo(ctx context.Context, client *client.SSEClient) {
	fmt.Println("\n=== MCP SSH客户端演示（HTTP SSE传输）===")

	// 列出可用工具
	listTools(ctx, client)

	// 演示SSH命令执行
	fmt.Println("\n=== 演示SSH命令执行 ===")
	args := map[string]interface{}{
		"host":    "localhost",
		"command": "echo 'Hello from MCP SSH Server (SSE)!' && date && uptime",
	}

	result, err := client.CallTool("ssh_execute", args)
	if err != nil {
		log.Printf("SSH命令执行失败: %v", err)
		return
	}

	if result.Error != nil {
		log.Printf("工具调用错误: %s", result.Error.Message)
		return
	}

	fmt.Println("执行结果:")
	if resultMap, ok := result.Result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if contentItem, ok := item.(map[string]interface{}); ok {
					if text, ok := contentItem["text"].(string); ok {
						fmt.Println(text)
					}
				}
			}
		}
	}
}

// runInteractive 运行交互模式
func runInteractive(ctx context.Context, client *client.SSEClient) {
	fmt.Println("\n=== 交互模式 ===")
	fmt.Println("输入 'list' 查看工具列表")
	fmt.Println("输入 'call <tool_name> <json_args>' 调用工具")
	fmt.Println("输入 'quit' 退出")

	// 这里可以实现交互式命令行界面
	// 为简化示例，直接显示提示信息
	fmt.Println("交互模式功能待实现...")
}

// runDirectCall 直接调用工具
func runDirectCall(ctx context.Context, client *client.SSEClient, toolName, toolArgs string) {
	if toolName == "" {
		log.Fatal("请指定工具名称")
	}

	fmt.Printf("调用工具: %s\n", toolName)
	fmt.Printf("参数: %s\n", toolArgs)

	// 解析参数
	args := make(map[string]interface{})
	if toolArgs != "{}" {
		if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
			log.Fatalf("解析工具参数失败: %v", err)
		}
	} else {
		// 使用默认参数
		args = map[string]interface{}{
			"host":    "localhost",
			"command": "uptime",
		}
	}

	result, err := client.CallTool(toolName, args)
	if err != nil {
		log.Fatalf("工具调用失败: %v", err)
	}

	if result.Error != nil {
		log.Fatalf("工具调用错误: %s", result.Error.Message)
	}

	fmt.Println("调用结果:")
	if resultMap, ok := result.Result.(map[string]interface{}); ok {
		if content, ok := resultMap["content"].([]interface{}); ok {
			for _, item := range content {
				if contentItem, ok := item.(map[string]interface{}); ok {
					if text, ok := contentItem["text"].(string); ok {
						fmt.Println(text)
					}
				}
			}
		}
	}
}

// listTools 列出可用工具
func listTools(ctx context.Context, client *client.SSEClient) {
	fmt.Println("\n=== 可用工具列表 ===")

	tools, err := client.ListTools()
	if err != nil {
		log.Printf("获取工具列表失败: %v", err)
		return
	}

	for i, tool := range tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name)
		fmt.Printf("   描述: %s\n", tool.Description)
		fmt.Printf("   输入模式: %v\n", tool.InputSchema["type"])
		fmt.Println()
	}
}
 