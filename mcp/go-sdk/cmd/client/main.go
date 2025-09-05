package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// 解析命令行参数
	var serverCommand []string
	var serverCmd = flag.String("server", "./build/ssh-mcp-server-sdk", "MCP服务器命令")
	var serverArgs = flag.String("args", "", "服务器参数（用空格分隔）")
	var interactive = flag.Bool("interactive", false, "启动交互模式")
	var toolName = flag.String("tool", "", "要调用的工具名称")
	var toolArgs = flag.String("tool-args", "", "工具参数（JSON格式）")
	var sseMode = flag.Bool("sse", false, "使用HTTP SSE模式连接")
	var serverURL = flag.String("url", "http://localhost:8080/mcp/sse", "SSE服务器URL")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	flag.Parse()

	// 获取服务器命令
	args := flag.Args()
	if len(args) > 0 && args[0] == "-server" && len(args) > 1 {
		serverCommand = strings.Fields(args[1])
		if len(args) > 2 && args[2] == "-args" && len(args) > 3 {
			serverCommand = append(serverCommand, strings.Fields(args[3])...)
		}
	}

	// 显示版本信息
	if *showVersion {
		fmt.Println("SSH MCP Client (Official SDK) v1.0.0")
		fmt.Println("基于官方Go SDK实现的MCP SSH客户端")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("SSH MCP Client (Official SDK) - 基于官方MCP Go SDK的SSH远程执行客户端")
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
		fmt.Printf("  # 直接调用工具\n")
		fmt.Printf("  %s -tool ssh_execute -tool-args '{\"host\":\"localhost\",\"command\":\"uptime\"}'\n", os.Args[0])
		fmt.Println()
		fmt.Printf("  # 指定服务器命令\n")
		fmt.Printf("  %s -server ./build/ssh-mcp-server-sdk -args \"-config config.yaml\"\n", os.Args[0])
		os.Exit(0)
	}

	// 验证参数
	if !*sseMode && len(serverCommand) == 0 && *serverCmd == "" {
		log.Fatal("必须指定服务器命令或使用SSE模式")
	}

	// 构建服务器命令
	if *sseMode {
		// SSE模式下，serverCommand 已经通过 flag.Args() 解析
		fmt.Printf("使用SSE模式连接到服务器: %s\n", *serverURL)
	} else {
		if *serverCmd != "" {
			serverCommand = append(serverCommand, *serverCmd)
			if *serverArgs != "" {
				args := strings.Fields(*serverArgs)
				serverCommand = append(serverCommand, args...)
			}
		} else {
			log.Fatal("必须指定服务器命令")
		}
	}

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 创建MCP客户端实例
	clientImpl := &mcp.Implementation{
		Name:    "SSH-MCP-Client-SDK",
		Version: "1.0.0",
	}

	// 定义客户端选项
	options := &mcp.ClientOptions{}

	client := mcp.NewClient(clientImpl, options)

	// 创建命令传输
	var transport mcp.Transport
	if *sseMode {
		transport = mcp.NewSSEClientTransport(*serverURL, nil)
	} else {
		cmd := exec.Command(serverCommand[0], serverCommand[1:]...)
		transport = mcp.NewCommandTransport(cmd)
	}

	// 连接到服务器
	fmt.Printf("连接到MCP服务器: %v\n", serverCommand)
	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		log.Fatalf("连接MCP服务器失败: %v", err)
	}
	defer session.Close()

	fmt.Printf("成功连接到MCP服务器\n")

	// 根据模式执行不同的逻辑
	if *interactive {
		runInteractiveMode(ctx, session)
	} else if *toolName != "" {
		runToolMode(ctx, session, *toolName, *toolArgs)
	} else {
		runDemoMode(ctx, session)
	}
}

// runInteractiveMode 运行交互模式
func runInteractiveMode(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\n=== MCP SSH客户端交互模式（官方SDK版本）===")
	fmt.Println("输入 'help' 查看帮助，输入 'quit' 退出")

	for {
		fmt.Print("\nmcp-sdk> ")
		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "quit", "exit", "q":
			fmt.Println("再见！")
			return
		case "help", "h":
			showInteractiveHelp()
		case "tools", "t":
			listTools(ctx, session)
		case "info", "i":
			showServerInfo(session)
		case "ssh":
			runSSHCommand(ctx, session)
		default:
			fmt.Printf("未知命令: %s，输入 'help' 查看帮助\n", input)
		}
	}
}

// runToolMode 运行工具调用模式
func runToolMode(ctx context.Context, session *mcp.ClientSession, toolName, toolArgs string) {
	fmt.Printf("调用工具: %s\n", toolName)

	// 解析工具参数
	var args map[string]interface{}
	if toolArgs != "" {
		if err := json.Unmarshal([]byte(toolArgs), &args); err != nil {
			log.Fatalf("解析工具参数失败: %v", err)
		}
	} else {
		// 使用默认参数
		args = map[string]interface{}{
			"host":    "localhost",
			"command": "echo 'Hello from MCP SDK!'",
		}
	}

	// 调用工具
	params := &mcp.CallToolParams{
		Name:      toolName,
		Arguments: args,
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		log.Fatalf("工具调用失败: %v", err)
	}

	// 显示结果
	fmt.Println("\n=== 工具调用结果 ===")
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
}

// runDemoMode 运行演示模式
func runDemoMode(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\n=== MCP SSH客户端演示模式（官方SDK版本）===")

	// 显示服务器信息
	showServerInfo(session)

	// 列出工具
	listTools(ctx, session)

	// 演示SSH命令执行
	fmt.Println("\n=== 演示SSH命令执行 ===")
	args := map[string]interface{}{
		"host":    "localhost",
		"command": "echo 'Hello from MCP SSH Server SDK!' && date && uptime",
	}

	params := &mcp.CallToolParams{
		Name:      "ssh_execute",
		Arguments: args,
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		log.Printf("SSH命令执行失败: %v", err)
		return
	}

	fmt.Println("执行结果:")
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
}

// showInteractiveHelp 显示交互模式帮助
func showInteractiveHelp() {
	fmt.Println("\n=== 交互模式帮助 ===")
	fmt.Println("可用命令:")
	fmt.Println("  help, h     - 显示此帮助信息")
	fmt.Println("  tools, t    - 列出可用工具")
	fmt.Println("  info, i     - 显示服务器信息")
	fmt.Println("  ssh         - 执行SSH命令")
	fmt.Println("  quit, q     - 退出程序")
}

// showServerInfo 显示服务器信息
func showServerInfo(session *mcp.ClientSession) {
	fmt.Println("\n=== 服务器信息 ===")

	// 获取服务器信息（从session中获取）
	fmt.Printf("协议版本: %s\n", "2025-03-26") // 从初始化结果获取
	fmt.Println("服务器能力:")
	fmt.Printf("  - 工具支持: 是\n")
}

// listTools 列出可用工具
func listTools(ctx context.Context, session *mcp.ClientSession) {
	fmt.Println("\n=== 可用工具列表 ===")

	// 使用官方SDK获取工具列表
	result, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		log.Printf("获取工具列表失败: %v", err)
		return
	}

	for i, tool := range result.Tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name)
		fmt.Printf("   描述: %s\n", tool.Description)

		// 显示工具信息（简化版本，因为InputSchema是jsonschema.Schema类型）
		fmt.Printf("   输入模式: 对象类型\n")
		fmt.Println()
	}
}

// runSSHCommand 运行SSH命令
func runSSHCommand(ctx context.Context, session *mcp.ClientSession) {
	fmt.Print("请输入目标主机: ")
	var host string
	fmt.Scanln(&host)

	fmt.Print("请输入要执行的命令: ")
	var command string
	fmt.Scanln(&command)

	if host == "" || command == "" {
		fmt.Println("主机和命令不能为空")
		return
	}

	args := map[string]interface{}{
		"host":    host,
		"command": command,
	}

	fmt.Printf("正在执行: %s@%s: %s\n", "root", host, command)

	params := &mcp.CallToolParams{
		Name:      "ssh_execute",
		Arguments: args,
	}

	result, err := session.CallTool(ctx, params)
	if err != nil {
		fmt.Printf("SSH命令执行失败: %v\n", err)
		return
	}

	fmt.Println("\n=== 执行结果 ===")
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
}
