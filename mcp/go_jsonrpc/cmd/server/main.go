package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ssh-mcp-go-jsonrpc/pkg/config"
	"ssh-mcp-go-jsonrpc/pkg/server"
)

func main() {
	// 解析命令行参数
	var configPath = flag.String("config", "config.yaml", "配置文件路径")
	var showVersion = flag.Bool("version", false, "显示版本信息")
	var showHelp = flag.Bool("help", false, "显示帮助信息")
	flag.Parse()

	// 显示版本信息
	if *showVersion {
		fmt.Println("SSH MCP Server v1.0.0")
		fmt.Println("基于Go语言实现的MCP SSH服务器")
		os.Exit(0)
	}

	// 显示帮助信息
	if *showHelp {
		fmt.Println("SSH MCP Server - 基于MCP协议的SSH远程执行服务器")
		fmt.Println()
		fmt.Println("用法:")
		fmt.Printf("  %s [选项]\n", os.Args[0])
		fmt.Println()
		fmt.Println("选项:")
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println("示例:")
		fmt.Printf("  %s -config /etc/ssh-mcp-server/config.yaml\n", os.Args[0])
		fmt.Println()
		fmt.Println("配置文件示例:")
		fmt.Println("  server:")
		fmt.Println("    name: \"SSH-MCP-Server\"")
		fmt.Println("    version: \"1.0.0\"")
		fmt.Println("    protocol_version: \"2025-03-26\"")
		fmt.Println("    timeout: 30s")
		fmt.Println("  ssh:")
		fmt.Println("    default_user: \"root\"")
		fmt.Println("    default_port: 22")
		fmt.Println("    timeout: 30s")
		fmt.Println("    key_file: \"~/.ssh/id_rsa\"")
		fmt.Println("    known_hosts_file: \"~/.ssh/known_hosts\"")
		fmt.Println("    max_connections: 10")
		fmt.Println("  log:")
		fmt.Println("    level: \"info\"")
		fmt.Println("    file: \"/var/log/ssh-mcp-server.log\"")
		fmt.Println("    max_size: 100")
		fmt.Println("    max_backups: 3")
		fmt.Println("    max_age: 28")
		fmt.Println("    compress: true")
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
	mcpServer, err := server.NewMCPServer(cfg)
	if err != nil {
		log.Fatalf("创建MCP服务器失败: %v", err)
	}

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器协程
	errChan := make(chan error, 1)
	go func() {
		errChan <- mcpServer.Run()
	}()

	// 等待信号或错误
	select {
	case sig := <-sigChan:
		log.Printf("收到信号: %v，正在关闭服务器...", sig)
		if err := mcpServer.Close(); err != nil {
			log.Printf("关闭服务器失败: %v", err)
		}
	case err := <-errChan:
		if err != nil {
			log.Fatalf("服务器运行失败: %v", err)
		}
	}

	log.Println("服务器已关闭")
}
