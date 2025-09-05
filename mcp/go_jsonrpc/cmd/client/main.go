package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"ssh-mcp-go-jsonrpc/pkg/client"
	"ssh-mcp-go-jsonrpc/pkg/types"
)

func main() {
	// 解析命令行参数
	var serverCmd = flag.String("server", "./ssh-mcp-server", "MCP服务器命令")
	var serverArgs = flag.String("args", "", "服务器参数（用空格分隔）")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	var interactive = flag.Bool("interactive", false, "交互模式")
	var toolName = flag.String("tool", "", "要调用的工具名称")
	var toolArgs = flag.String("tool-args", "", "工具参数（JSON格式）")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Println("SSH MCP Client v1.0.0")
		fmt.Println("基于Go语言实现的MCP SSH客户端")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("SSH MCP Client - 基于MCP协议的SSH远程执行客户端")
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
		fmt.Printf("  %s -tool ssh_execute -tool-args '{\"host\":\"192.168.1.100\",\"command\":\"uptime\"}'\n", os.Args[0])
		fmt.Println()
		fmt.Printf("  # 指定服务器命令\n")
		fmt.Printf("  %s -server ./ssh-mcp-server -args \"-config config.yaml\"\n", os.Args[0])
		os.Exit(0)
	}

	// 构建服务器命令
	var serverCommand []string
	if *serverCmd != "" {
		serverCommand = append(serverCommand, *serverCmd)
		if *serverArgs != "" {
			args := strings.Fields(*serverArgs)
			serverCommand = append(serverCommand, args...)
		}
	} else {
		log.Fatal("必须指定服务器命令")
	}

	// 创建客户端配置
	config := &client.Config{
		ServerCommand: serverCommand,
		ClientInfo: types.ClientInfo{
			Name:    "SSH-MCP-Client",
			Version: "1.0.0",
		},
	}

	// 创建MCP客户端
	mcpClient := client.NewMCPClient(config)
	defer mcpClient.Close()

	// 连接到服务器
	if err := mcpClient.Connect(serverCommand); err != nil {
		log.Fatalf("连接MCP服务器失败: %v", err)
	}

	// 初始化连接
	if err := mcpClient.Initialize(); err != nil {
		log.Fatalf("初始化MCP连接失败: %v", err)
	}

	fmt.Printf("成功连接到MCP服务器: %s v%s\n",
		mcpClient.GetServerInfo().Name,
		mcpClient.GetServerInfo().Version)

	// 根据模式执行不同的逻辑
	if *interactive {
		runInteractiveMode(mcpClient)
	} else if *toolName != "" {
		runToolMode(mcpClient, *toolName, *toolArgs)
	} else {
		runDemoMode(mcpClient)
	}
}

// runInteractiveMode 运行交互模式
func runInteractiveMode(client *client.MCPClient) {
	fmt.Println("\n=== MCP SSH客户端交互模式 ===")
	fmt.Println("输入 'help' 查看帮助，输入 'quit' 退出")

	for {
		fmt.Print("\nmcp> ")
		var input string
		fmt.Scanln(&input)

		switch strings.ToLower(input) {
		case "quit", "exit", "q":
			fmt.Println("再见！")
			return
		case "help", "h":
			showInteractiveHelp()
		case "tools", "t":
			listTools(client)
		case "info", "i":
			showServerInfo(client)
		case "ssh":
			runSSHCommand(client)
		default:
			fmt.Printf("未知命令: %s，输入 'help' 查看帮助\n", input)
		}
	}
}

// runToolMode 运行工具调用模式
func runToolMode(client *client.MCPClient, toolName, toolArgs string) {
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
			"command": "echo 'Hello from MCP!'",
		}
	}

	// 调用工具
	result, err := client.CallTool(toolName, args)
	if err != nil {
		log.Fatalf("工具调用失败: %v", err)
	}

	// 显示结果
	fmt.Println("\n=== 工具调用结果 ===")
	for _, content := range result.Content {
		if textContent, ok := content.(*types.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
}

// runDemoMode 运行演示模式
func runDemoMode(client *client.MCPClient) {
	fmt.Println("\n=== MCP SSH客户端演示模式 ===")

	// 显示服务器信息
	showServerInfo(client)

	// 列出工具
	listTools(client)

	// 演示SSH命令执行
	fmt.Println("\n=== 演示SSH命令执行 ===")
	args := map[string]interface{}{
		"host":    "localhost",
		"command": "echo 'Hello from MCP SSH Server!' && date && uptime",
	}

	result, err := client.CallTool("ssh_execute", args)
	if err != nil {
		log.Printf("SSH命令执行失败: %v", err)
		return
	}

	fmt.Println("执行结果:")
	for _, content := range result.Content {
		if textContent, ok := content.(*types.TextContent); ok {
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
func showServerInfo(client *client.MCPClient) {
	fmt.Println("\n=== 服务器信息 ===")
	serverInfo := client.GetServerInfo()
	fmt.Printf("名称: %s\n", serverInfo.Name)
	fmt.Printf("版本: %s\n", serverInfo.Version)
	fmt.Printf("协议版本: %s\n", client.GetProtocolVersion())

	capabilities := client.GetServerCapabilities()
	fmt.Println("服务器能力:")
	if capabilities.Tools != nil {
		fmt.Printf("  - 工具支持: 是 (列表变化通知: %v)\n", capabilities.Tools.ListChanged)
	}
	if capabilities.Resources != nil {
		fmt.Printf("  - 资源支持: 是\n")
	}
	if capabilities.Prompts != nil {
		fmt.Printf("  - 提示支持: 是\n")
	}
}

// listTools 列出可用工具
func listTools(client *client.MCPClient) {
	fmt.Println("\n=== 可用工具列表 ===")

	tools, err := client.ListTools()
	if err != nil {
		log.Printf("获取工具列表失败: %v", err)
		return
	}

	for i, tool := range tools {
		fmt.Printf("%d. %s\n", i+1, tool.Name)
		fmt.Printf("   描述: %s\n", tool.Description)

		// 显示必需参数
		if required, ok := tool.InputSchema["required"].([]interface{}); ok {
			fmt.Print("   必需参数: ")
			for j, param := range required {
				if j > 0 {
					fmt.Print(", ")
				}
				fmt.Print(param)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

// runSSHCommand 运行SSH命令
func runSSHCommand(client *client.MCPClient) {
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

	result, err := client.CallTool("ssh_execute", args)
	if err != nil {
		fmt.Printf("SSH命令执行失败: %v\n", err)
		return
	}

	fmt.Println("\n=== 执行结果 ===")
	for _, content := range result.Content {
		if textContent, ok := content.(*types.TextContent); ok {
			fmt.Println(textContent.Text)
		}
	}
}
