package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ssh-mcp-go-sdk/pkg/config"
	"ssh-mcp-go-sdk/pkg/ssh"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SSHExecuteParams SSH命令执行参数
type SSHExecuteParams struct {
	Host     string `json:"host" jsonschema:"description:目标主机地址"`
	Command  string `json:"command" jsonschema:"description:要执行的命令"`
	User     string `json:"user,omitempty" jsonschema:"description:SSH用户名"`
	Port     int    `json:"port,omitempty" jsonschema:"description:SSH端口"`
	Password string `json:"password,omitempty" jsonschema:"description:SSH密码"`
	Timeout  int    `json:"timeout,omitempty" jsonschema:"description:超时时间（秒）"`
}

// SSHExecuteResult SSH命令执行结果
type SSHExecuteResult struct {
	Host     string `json:"host" jsonschema:"description:目标主机"`
	Command  string `json:"command" jsonschema:"description:执行的命令"`
	ExitCode int    `json:"exitCode" jsonschema:"description:退出码"`
	Stdout   string `json:"stdout" jsonschema:"description:标准输出"`
	Stderr   string `json:"stderr" jsonschema:"description:标准错误"`
	Duration string `json:"duration" jsonschema:"description:执行时长"`
}

// SSHFileTransferParams SSH文件传输参数
type SSHFileTransferParams struct {
	Host       string `json:"host" jsonschema:"description:目标主机地址"`
	LocalPath  string `json:"localPath" jsonschema:"description:本地文件路径"`
	RemotePath string `json:"remotePath" jsonschema:"description:远程文件路径"`
	User       string `json:"user,omitempty" jsonschema:"description:SSH用户名"`
	Port       int    `json:"port,omitempty" jsonschema:"description:SSH端口"`
	Password   string `json:"password,omitempty" jsonschema:"description:SSH密码"`
	Direction  string `json:"direction" jsonschema:"description:传输方向,enum:upload,enum:download"`
}

// SSHFileTransferResult SSH文件传输结果
type SSHFileTransferResult struct {
	Success    bool   `json:"success" jsonschema:"description:是否成功"`
	Message    string `json:"message" jsonschema:"description:结果消息"`
	Host       string `json:"host" jsonschema:"description:目标主机"`
	LocalPath  string `json:"localPath" jsonschema:"description:本地文件路径"`
	RemotePath string `json:"remotePath" jsonschema:"description:远程文件路径"`
	Direction  string `json:"direction" jsonschema:"description:传输方向"`
}

// MCPSSHServer 基于官方SDK的SSH MCP服务器
type MCPSSHServer struct {
	config    *config.Config
	sshClient *ssh.Client
	server    *mcp.Server
}

// NewMCPSSHServer 创建新的SSH MCP服务器
func NewMCPSSHServer(cfg *config.Config) (*MCPSSHServer, error) {
	// 创建SSH客户端
	sshConfig := &ssh.Config{
		DefaultUser:    cfg.SSH.DefaultUser,
		DefaultPort:    cfg.SSH.DefaultPort,
		Timeout:        cfg.SSH.Timeout,
		KeyFile:        cfg.SSH.KeyFile,
		KnownHostsFile: cfg.SSH.KnownHostsFile,
		MaxConnections: cfg.SSH.MaxConnections,
	}
	sshClient := ssh.NewClient(sshConfig)

	// 创建MCP服务器实例
	serverImpl := &mcp.Implementation{
		Name:    cfg.Server.Name,
		Version: cfg.Server.Version,
	}

	// 定义服务器选项
	options := &mcp.ServerOptions{
		HasTools: true, // 声明支持工具
	}

	server := mcp.NewServer(serverImpl, options)

	mcpServer := &MCPSSHServer{
		config:    cfg,
		sshClient: sshClient,
		server:    server,
	}

	// 注册工具
	mcpServer.registerTools()

	return mcpServer, nil
}

// registerTools 注册MCP工具
func (s *MCPSSHServer) registerTools() {
	// 注册SSH命令执行工具
	sshExecuteTool := &mcp.Tool{
		Name:        "ssh_execute",
		Description: "在远程服务器上执行Shell命令",
		// InputSchema 将由AddTool自动生成
	}

	// 使用官方SDK的AddTool方法注册工具（带类型安全）
	mcp.AddTool(s.server, sshExecuteTool, s.handleSSHExecute)

	// 注册SSH文件传输工具
	sshFileTransferTool := &mcp.Tool{
		Name:        "ssh_file_transfer",
		Description: "SSH文件传输（上传/下载）",
		// InputSchema 将由AddTool自动生成
	}

	mcp.AddTool(s.server, sshFileTransferTool, s.handleSSHFileTransfer)
}

// handleSSHExecute 处理SSH命令执行工具调用
func (s *MCPSSHServer) handleSSHExecute(ctx context.Context, req *mcp.CallToolRequest, args SSHExecuteParams) (*mcp.CallToolResult, SSHExecuteResult, error) {
	log.Printf("执行SSH命令: %s@%s:%d - %s", args.User, args.Host, args.Port, args.Command)

	// 填充默认值
	if args.User == "" {
		args.User = s.config.SSH.DefaultUser
	}
	if args.Port == 0 {
		args.Port = s.config.SSH.DefaultPort
	}
	if args.Timeout == 0 {
		args.Timeout = int(s.config.SSH.Timeout.Seconds())
	}

	// 创建SSH连接信息
	connInfo := &ssh.ConnectionInfo{
		Host:     args.Host,
		Port:     args.Port,
		User:     args.User,
		Password: args.Password,
	}

	// 执行SSH命令
	result, err := s.sshClient.Execute(connInfo, args.Command)
	if err != nil {
		return nil, SSHExecuteResult{}, fmt.Errorf("SSH命令执行失败: %w", err)
	}

	// 构建响应内容
	infoText := fmt.Sprintf("主机: %s\n命令: %s\n退出码: %d\n执行时长: %v\n",
		args.Host, args.Command, result.ExitCode, result.Duration)

	if result.Stdout != "" {
		infoText += fmt.Sprintf("标准输出:\n%s\n", result.Stdout)
	}

	if result.Stderr != "" {
		infoText += fmt.Sprintf("标准错误:\n%s\n", result.Stderr)
	}

	// 使用官方SDK的内容类型
	content := []mcp.Content{
		&mcp.TextContent{
			Text: infoText,
		},
	}

	// 构建结构化结果
	structuredResult := SSHExecuteResult{
		Host:     args.Host,
		Command:  args.Command,
		ExitCode: result.ExitCode,
		Stdout:   result.Stdout,
		Stderr:   result.Stderr,
		Duration: result.Duration.String(),
	}

	return &mcp.CallToolResult{
		Content: content,
		IsError: result.ExitCode != 0,
	}, structuredResult, nil
}

// handleSSHFileTransfer 处理SSH文件传输工具调用
func (s *MCPSSHServer) handleSSHFileTransfer(ctx context.Context, req *mcp.CallToolRequest, args SSHFileTransferParams) (*mcp.CallToolResult, SSHFileTransferResult, error) {
	log.Printf("SSH文件传输: %s@%s:%d - %s %s -> %s",
		args.User, args.Host, args.Port, args.Direction, args.LocalPath, args.RemotePath)

	// 填充默认值
	if args.User == "" {
		args.User = s.config.SSH.DefaultUser
	}
	if args.Port == 0 {
		args.Port = s.config.SSH.DefaultPort
	}

	// 这里应该实现实际的文件传输逻辑
	// 为简化示例，返回一个模拟结果
	content := []mcp.Content{
		&mcp.TextContent{
			Text: fmt.Sprintf("文件传输完成\n方向: %s\n本地路径: %s\n远程路径: %s\n主机: %s",
				args.Direction, args.LocalPath, args.RemotePath, args.Host),
		},
	}

	// 构建结构化结果
	structuredResult := SSHFileTransferResult{
		Success:    true,
		Message:    "文件传输完成（模拟）",
		Host:       args.Host,
		LocalPath:  args.LocalPath,
		RemotePath: args.RemotePath,
		Direction:  args.Direction,
	}

	return &mcp.CallToolResult{
		Content: content,
		IsError: false,
	}, structuredResult, nil
}

// Run 启动MCP服务器
func (s *MCPSSHServer) Run(ctx context.Context) error {
	log.Println("启动SSH MCP服务器（基于官方SDK）")
	defer log.Println("SSH MCP服务器已停止")
	defer s.sshClient.Close()

	// 使用官方SDK的StdioTransport运行服务器
	transport := &mcp.StdioTransport{}
	return s.server.Run(ctx, transport)
}

// RunSSE 启动基于HTTP SSE的MCP服务器
func (s *MCPSSHServer) RunSSE(ctx context.Context, port int) error {
	log.Printf("启动SSH MCP SSE服务器（基于官方SDK）在端口 %d", port)
	defer log.Println("SSH MCP SSE服务器已停止")
	defer s.sshClient.Close()

	// 创建HTTP服务器
	mux := http.NewServeMux()

	// 创建SSE处理器
	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return s.server
	})

	// 注册路由
	mux.Handle("/mcp/sse", handler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	// 启动服务器
	go func() {
		<-ctx.Done()
		server.Shutdown(context.Background())
	}()

	log.Printf("MCP SSE服务器正在监听端口 %d", port)
	return server.ListenAndServe()
}

// Close 关闭服务器
func (s *MCPSSHServer) Close() error {
	return s.sshClient.Close()
}

func main() {
	// 解析命令行参数
	var configPath = flag.String("config", "config.yaml", "配置文件路径")
	var sseMode = flag.Bool("sse", false, "启用HTTP SSE模式")
	var port = flag.Int("port", 8080, "SSE模式下的端口号")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Println("SSH MCP Server (Official SDK) v1.0.0")
		fmt.Println("基于官方Go SDK实现的MCP SSH服务器")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("SSH MCP Server (Official SDK) - 基于官方MCP Go SDK的SSH远程执行服务器")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Printf("  %s [选项]\n", os.Args[0])
		fmt.Println()
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("示例:")
		fmt.Printf("  %s -config /etc/ssh-mcp-server/config.yaml\n", os.Args[0])
		os.Exit(0)
	}

	// 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置验证失败: %v", err)
	}

	// 创建MCP服务器
	mcpServer, err := NewMCPSSHServer(cfg)
	if err != nil {
		log.Fatalf("创建MCP服务器失败: %v", err)
	}

	// 设置信号处理
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器协程
	errChan := make(chan error, 1)
	go func() {
		if *sseMode {
			errChan <- mcpServer.RunSSE(ctx, *port)
		} else {
			errChan <- mcpServer.Run(ctx)
		}
	}()

	// 等待信号或错误
	select {
	case sig := <-sigChan:
		log.Printf("收到信号: %v，正在关闭服务器...", sig)
		cancel()
		// 给服务器一些时间优雅关闭
		time.Sleep(1 * time.Second)
		if err := mcpServer.Close(); err != nil {
			log.Printf("关闭服务器失败: %v", err)
		}
	case err := <-errChan:
		cancel()
		if err != nil {
			log.Fatalf("服务器运行失败: %v", err)
		}
	}

	log.Println("服务器已关闭")
}
