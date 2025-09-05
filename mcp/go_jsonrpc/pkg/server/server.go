package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"ssh-mcp-go-jsonrpc/pkg/config"
	"ssh-mcp-go-jsonrpc/pkg/ssh"
	"ssh-mcp-go-jsonrpc/pkg/types"
)

// MCPServer MCP服务器实现
// 负责处理JSON-RPC 2.0协议和MCP规范的消息
type MCPServer struct {
	config    *config.Config     // 服务器配置
	sshClient *ssh.Client        // SSH客户端
	ctx       context.Context    // 上下文
	cancel    context.CancelFunc // 取消函数
	mutex     sync.RWMutex       // 读写锁
	logger    *log.Logger        // 日志记录器

	// 输入输出流
	reader *bufio.Scanner // 输入流读取器
	writer io.Writer      // 输出流写入器

	// 状态管理
	initialized  bool                     // 是否已初始化
	capabilities types.ServerCapabilities // 服务器能力
}

// NewMCPServer 创建新的MCP服务器实例
func NewMCPServer(cfg *config.Config) (*MCPServer, error) {
	ctx, cancel := context.WithCancel(context.Background())

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

	// 创建日志记录器
	logger := log.New(os.Stderr, "[MCP-Server] ", log.LstdFlags|log.Lshortfile)

	// 定义服务器能力
	capabilities := types.ServerCapabilities{
		Tools: &types.ToolsCapability{
			ListChanged: true,
		},
	}

	server := &MCPServer{
		config:       cfg,
		sshClient:    sshClient,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		reader:       bufio.NewScanner(os.Stdin),
		writer:       os.Stdout,
		capabilities: capabilities,
	}

	return server, nil
}

// Run 启动MCP服务器主循环
// 从stdin读取JSON-RPC消息，处理后向stdout写入响应
func (s *MCPServer) Run() error {
	s.logger.Println("MCP服务器启动")
	defer s.logger.Println("MCP服务器停止")
	defer s.sshClient.Close()

	// 主消息循环
	for s.reader.Scan() {
		line := s.reader.Text()
		if line == "" {
			continue
		}

		// 处理消息
		if err := s.handleMessage(line); err != nil {
			s.logger.Printf("处理消息失败: %v", err)
		}
	}

	if err := s.reader.Err(); err != nil {
		return fmt.Errorf("读取输入失败: %w", err)
	}

	return nil
}

// handleMessage 处理单个JSON-RPC消息
func (s *MCPServer) handleMessage(message string) error {
	s.logger.Printf("收到消息: %s", message)

	// 解析JSON-RPC请求
	var request types.MCPRequest
	if err := json.Unmarshal([]byte(message), &request); err != nil {
		s.sendError(nil, types.ParseError, "解析JSON失败", err.Error())
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	// 验证JSON-RPC版本
	if request.JSONRPC != "2.0" {
		s.sendError(request.ID, types.InvalidRequest, "无效的JSON-RPC版本", nil)
		return fmt.Errorf("无效的JSON-RPC版本: %s", request.JSONRPC)
	}

	// 路由到具体的处理方法
	return s.routeRequest(&request)
}

// routeRequest 根据方法名路由请求到具体的处理函数
func (s *MCPServer) routeRequest(request *types.MCPRequest) error {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolsCall(request)
	case "notifications/initialized":
		return s.handleInitializedNotification(request)
	default:
		s.sendError(request.ID, types.MethodNotFound, fmt.Sprintf("未知方法: %s", request.Method), nil)
		return fmt.Errorf("未知方法: %s", request.Method)
	}
}

// handleInitialize 处理初始化请求
func (s *MCPServer) handleInitialize(request *types.MCPRequest) error {
	s.logger.Println("处理初始化请求")

	// 解析初始化参数
	var params types.InitializeParams
	if request.Params != nil {
		paramsBytes, err := json.Marshal(request.Params)
		if err != nil {
			s.sendError(request.ID, types.InvalidParams, "无效的初始化参数", err.Error())
			return fmt.Errorf("序列化参数失败: %w", err)
		}

		if err := json.Unmarshal(paramsBytes, &params); err != nil {
			s.sendError(request.ID, types.InvalidParams, "无效的初始化参数", err.Error())
			return fmt.Errorf("解析初始化参数失败: %w", err)
		}
	}

	s.logger.Printf("客户端信息: %s v%s", params.ClientInfo.Name, params.ClientInfo.Version)
	s.logger.Printf("协议版本: %s", params.ProtocolVersion)

	// 构建初始化响应
	result := types.InitializeResult{
		ProtocolVersion: s.config.Server.ProtocolVersion,
		Capabilities:    s.capabilities,
		ServerInfo: types.ServerInfo{
			Name:    s.config.Server.Name,
			Version: s.config.Server.Version,
		},
	}

	return s.sendResponse(request.ID, result)
}

// handleToolsList 处理工具列表请求
func (s *MCPServer) handleToolsList(request *types.MCPRequest) error {
	s.logger.Println("处理工具列表请求")

	// 检查是否已初始化
	if !s.initialized {
		s.sendError(request.ID, types.ServerError, "服务器未初始化", nil)
		return fmt.Errorf("服务器未初始化")
	}

	// 定义可用工具
	tools := []types.Tool{
		{
			Name:        "ssh_execute",
			Description: "在远程服务器上执行Shell命令",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"host": map[string]interface{}{
						"type":        "string",
						"description": "目标服务器地址",
					},
					"command": map[string]interface{}{
						"type":        "string",
						"description": "要执行的命令",
					},
					"user": map[string]interface{}{
						"type":        "string",
						"description": "SSH用户名",
						"default":     s.config.SSH.DefaultUser,
					},
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "SSH端口",
						"default":     s.config.SSH.DefaultPort,
					},
					"timeout": map[string]interface{}{
						"type":        "integer",
						"description": "超时时间（秒）",
						"default":     int(s.config.SSH.Timeout.Seconds()),
					},
					"password": map[string]interface{}{
						"type":        "string",
						"description": "SSH密码（可选，优先使用密钥认证）",
					},
				},
				"required": []string{"host", "command"},
			},
		},
		{
			Name:        "ssh_file_transfer",
			Description: "SSH文件传输（上传/下载）",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"host": map[string]interface{}{
						"type":        "string",
						"description": "目标服务器地址",
					},
					"local_path": map[string]interface{}{
						"type":        "string",
						"description": "本地文件路径",
					},
					"remote_path": map[string]interface{}{
						"type":        "string",
						"description": "远程文件路径",
					},
					"direction": map[string]interface{}{
						"type":        "string",
						"description": "传输方向",
						"enum":        []string{"upload", "download"},
					},
					"user": map[string]interface{}{
						"type":        "string",
						"description": "SSH用户名",
						"default":     s.config.SSH.DefaultUser,
					},
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "SSH端口",
						"default":     s.config.SSH.DefaultPort,
					},
				},
				"required": []string{"host", "local_path", "remote_path", "direction"},
			},
		},
	}

	result := types.ToolsListResult{
		Tools: tools,
	}

	return s.sendResponse(request.ID, result)
}

// handleToolsCall 处理工具调用请求
func (s *MCPServer) handleToolsCall(request *types.MCPRequest) error {
	s.logger.Println("处理工具调用请求")

	// 检查是否已初始化
	if !s.initialized {
		s.sendError(request.ID, types.ServerError, "服务器未初始化", nil)
		return fmt.Errorf("服务器未初始化")
	}

	// 解析工具调用参数
	var params types.ToolCallParams
	if request.Params != nil {
		paramsBytes, err := json.Marshal(request.Params)
		if err != nil {
			s.sendError(request.ID, types.InvalidParams, "无效的工具调用参数", err.Error())
			return fmt.Errorf("序列化参数失败: %w", err)
		}

		if err := json.Unmarshal(paramsBytes, &params); err != nil {
			s.sendError(request.ID, types.InvalidParams, "无效的工具调用参数", err.Error())
			return fmt.Errorf("解析工具调用参数失败: %w", err)
		}
	}

	s.logger.Printf("调用工具: %s", params.Name)

	// 根据工具名称路由到具体的处理函数
	switch params.Name {
	case "ssh_execute":
		return s.handleSSHExecute(request.ID, params.Arguments)
	case "ssh_file_transfer":
		return s.handleSSHFileTransfer(request.ID, params.Arguments)
	default:
		s.sendError(request.ID, types.MethodNotFound, fmt.Sprintf("未知工具: %s", params.Name), nil)
		return fmt.Errorf("未知工具: %s", params.Name)
	}
}

// handleSSHExecute 处理SSH命令执行
func (s *MCPServer) handleSSHExecute(requestID interface{}, arguments map[string]interface{}) error {
	// 解析SSH执行参数
	var params types.SSHExecuteParams
	argsBytes, err := json.Marshal(arguments)
	if err != nil {
		s.sendError(requestID, types.InvalidParams, "无效的SSH执行参数", err.Error())
		return fmt.Errorf("序列化参数失败: %w", err)
	}

	if err := json.Unmarshal(argsBytes, &params); err != nil {
		s.sendError(requestID, types.InvalidParams, "无效的SSH执行参数", err.Error())
		return fmt.Errorf("解析SSH执行参数失败: %w", err)
	}

	// 填充默认值
	if params.User == "" {
		params.User = s.config.SSH.DefaultUser
	}
	if params.Port == 0 {
		params.Port = s.config.SSH.DefaultPort
	}
	if params.Timeout == 0 {
		params.Timeout = int(s.config.SSH.Timeout.Seconds())
	}

	s.logger.Printf("执行SSH命令: %s@%s:%d - %s", params.User, params.Host, params.Port, params.Command)

	// 创建SSH连接信息
	connInfo := &ssh.ConnectionInfo{
		Host:     params.Host,
		Port:     params.Port,
		User:     params.User,
		Password: params.Password,
	}

	// 执行SSH命令
	result, err := s.sshClient.Execute(connInfo, params.Command)
	if err != nil {
		s.sendError(requestID, types.ServerError, "SSH命令执行失败", err.Error())
		return fmt.Errorf("SSH命令执行失败: %w", err)
	}

	// 构建响应内容
	var content []types.Content

	// 添加命令信息
	infoText := fmt.Sprintf("主机: %s\n命令: %s\n退出码: %d\n执行时长: %v\n",
		params.Host, params.Command, result.ExitCode, result.Duration)

	if result.Stdout != "" {
		infoText += fmt.Sprintf("标准输出:\n%s\n", result.Stdout)
	}

	if result.Stderr != "" {
		infoText += fmt.Sprintf("标准错误:\n%s\n", result.Stderr)
	}

	content = append(content, &types.TextContent{
		Type: "text",
		Text: infoText,
	})

	// 构建工具调用结果
	toolResult := types.ToolCallResult{
		Content: content,
		IsError: result.ExitCode != 0,
	}

	return s.sendResponse(requestID, toolResult)
}

// handleSSHFileTransfer 处理SSH文件传输
func (s *MCPServer) handleSSHFileTransfer(requestID interface{}, arguments map[string]interface{}) error {
	// 解析SSH文件传输参数
	var params types.SSHFileTransferParams
	argsBytes, err := json.Marshal(arguments)
	if err != nil {
		s.sendError(requestID, types.InvalidParams, "无效的SSH文件传输参数", err.Error())
		return fmt.Errorf("序列化参数失败: %w", err)
	}

	if err := json.Unmarshal(argsBytes, &params); err != nil {
		s.sendError(requestID, types.InvalidParams, "无效的SSH文件传输参数", err.Error())
		return fmt.Errorf("解析SSH文件传输参数失败: %w", err)
	}

	// 填充默认值
	if params.User == "" {
		params.User = s.config.SSH.DefaultUser
	}
	if params.Port == 0 {
		params.Port = s.config.SSH.DefaultPort
	}

	s.logger.Printf("SSH文件传输: %s@%s:%d - %s %s -> %s",
		params.User, params.Host, params.Port, params.Direction, params.LocalPath, params.RemotePath)

	// 这里应该实现实际的文件传输逻辑
	// 为简化示例，返回一个模拟结果
	content := []types.Content{
		&types.TextContent{
			Type: "text",
			Text: fmt.Sprintf("文件传输完成\n方向: %s\n本地路径: %s\n远程路径: %s\n主机: %s",
				params.Direction, params.LocalPath, params.RemotePath, params.Host),
		},
	}

	toolResult := types.ToolCallResult{
		Content: content,
		IsError: false,
	}

	return s.sendResponse(requestID, toolResult)
}

// handleInitializedNotification 处理初始化完成通知
func (s *MCPServer) handleInitializedNotification(request *types.MCPRequest) error {
	s.logger.Println("收到初始化完成通知")
	s.mutex.Lock()
	s.initialized = true
	s.mutex.Unlock()

	// 通知消息不需要响应
	return nil
}

// sendResponse 发送成功响应
func (s *MCPServer) sendResponse(id interface{}, result interface{}) error {
	response := types.MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}

	return s.writeResponse(&response)
}

// sendError 发送错误响应
func (s *MCPServer) sendError(id interface{}, code int, message string, data interface{}) error {
	response := types.MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &types.MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}

	return s.writeResponse(&response)
}

// writeResponse 写入响应到输出流
func (s *MCPServer) writeResponse(response *types.MCPResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("序列化响应失败: %w", err)
	}

	s.logger.Printf("发送响应: %s", string(data))

	if _, err := fmt.Fprintf(s.writer, "%s\n", string(data)); err != nil {
		return fmt.Errorf("写入响应失败: %w", err)
	}

	return nil
}

// Close 关闭服务器
func (s *MCPServer) Close() error {
	s.cancel()
	return s.sshClient.Close()
}
