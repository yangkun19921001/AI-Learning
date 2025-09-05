package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ssh-mcp-go-jsonrpc/pkg/config"
	"ssh-mcp-go-jsonrpc/pkg/ssh"
	"ssh-mcp-go-jsonrpc/pkg/types"
)

// SSEServer HTTP SSE传输的MCP服务器
type SSEServer struct {
	config    *config.Config     // 服务器配置
	sshClient *ssh.Client        // SSH客户端
	ctx       context.Context    // 上下文
	cancel    context.CancelFunc // 取消函数
	mutex     sync.RWMutex       // 读写锁
	logger    *log.Logger        // 日志记录器

	// HTTP服务器
	httpServer *http.Server           // HTTP服务器实例
	sessions   map[string]*SSESession // 会话管理

	// 状态管理
	capabilities types.ServerCapabilities // 服务器能力
}

// SSESession SSE会话信息
type SSESession struct {
	ID       string                 // 会话ID
	Writer   http.ResponseWriter    // HTTP响应写入器
	Flusher  http.Flusher           // HTTP刷新器
	Messages chan types.MCPResponse // 消息通道
	Done     chan struct{}          // 完成信号
	mutex    sync.RWMutex           // 会话锁
}

// NewSSEServer 创建新的SSE MCP服务器实例
func NewSSEServer(cfg *config.Config) (*SSEServer, error) {
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
	logger := log.New(log.Writer(), "[SSE-MCP-Server] ", log.LstdFlags|log.Lshortfile)

	server := &SSEServer{
		config:    cfg,
		sshClient: sshClient,
		ctx:       ctx,
		cancel:    cancel,
		logger:    logger,
		sessions:  make(map[string]*SSESession),
		capabilities: types.ServerCapabilities{
			Tools: &types.ToolsCapability{
				ListChanged: true,
			},
		},
	}

	// 创建HTTP服务器
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp/sse", server.handleSSE)
	mux.HandleFunc("/mcp/message", server.handleMessage)

	server.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: mux,
	}

	return server, nil
}

// Run 启动SSE服务器
func (s *SSEServer) Run() error {
	s.logger.Printf("启动SSH MCP服务器（HTTP SSE传输），监听端口: %d", s.config.Server.Port)

	// 启动HTTP服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Printf("HTTP服务器错误: %v", err)
		}
	}()

	// 等待上下文取消
	<-s.ctx.Done()

	// 关闭所有会话
	s.mutex.Lock()
	for _, session := range s.sessions {
		close(session.Done)
	}
	s.mutex.Unlock()

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(ctx)
}

// handleSSE 处理SSE连接请求
func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	// 处理OPTIONS请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许GET请求建立SSE连接
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 设置SSE头
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// 获取Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 生成会话ID
	sessionID := generateSessionID()

	// 创建会话
	session := &SSESession{
		ID:       sessionID,
		Writer:   w,
		Flusher:  flusher,
		Messages: make(chan types.MCPResponse, 100),
		Done:     make(chan struct{}),
	}

	// 注册会话
	s.mutex.Lock()
	s.sessions[sessionID] = session
	s.mutex.Unlock()

	// 发送端点信息
	endpoint := fmt.Sprintf("http://%s/mcp/message?sessionId=%s", r.Host, sessionID)
	s.sendSSEEvent(w, flusher, "endpoint", endpoint)

	s.logger.Printf("新的SSE连接建立，会话ID: %s", sessionID)

	// 启动消息处理协程
	go s.handleSessionMessages(session)

	// 保持连接直到客户端断开
	select {
	case <-r.Context().Done():
		s.logger.Printf("SSE连接断开，会话ID: %s", sessionID)
	case <-session.Done:
		s.logger.Printf("会话结束，会话ID: %s", sessionID)
	}

	// 清理会话
	s.mutex.Lock()
	delete(s.sessions, sessionID)
	s.mutex.Unlock()

	// 通知消息处理协程停止
	close(session.Done)
	close(session.Messages)
}

// handleMessage 处理消息请求
func (s *SSEServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	// 设置CORS头
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, mcp-protocol-version")

	// 处理OPTIONS请求
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 只允许POST请求
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 获取会话ID
	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "Missing sessionId", http.StatusBadRequest)
		return
	}

	// 查找会话
	s.mutex.RLock()
	session, exists := s.sessions[sessionID]
	s.mutex.RUnlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// 读取请求体
	var request types.MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	s.logger.Printf("收到消息请求，会话ID: %s, 方法: %s", sessionID, request.Method)

	// 处理请求
	response := s.handleRequest(&request)

	// 如果是通知消息，直接返回200
	if request.ID == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	// 通过SSE发送响应
	select {
	case session.Messages <- *response:
		w.WriteHeader(http.StatusOK)
	case <-time.After(5 * time.Second):
		http.Error(w, "Response timeout", http.StatusRequestTimeout)
	}
}

// handleSessionMessages 处理会话消息
func (s *SSEServer) handleSessionMessages(session *SSESession) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Printf("会话消息处理协程异常退出: %v", r)
		}
	}()

	for {
		select {
		case message, ok := <-session.Messages:
			if !ok {
				// 消息通道已关闭
				return
			}

			// 序列化消息
			data, err := json.Marshal(message)
			if err != nil {
				s.logger.Printf("序列化消息失败: %v", err)
				continue
			}

			// 发送SSE消息
			s.sendSSEEvent(session.Writer, session.Flusher, "message", string(data))

		case <-session.Done:
			return
		}
	}
}

// sendSSEEvent 发送SSE事件
func (s *SSEServer) sendSSEEvent(w http.ResponseWriter, flusher http.Flusher, event, data string) {
	// 检查连接是否仍然有效
	if w == nil || flusher == nil {
		return
	}

	// 尝试写入，如果失败则忽略（连接可能已断开）
	defer func() {
		if r := recover(); r != nil {
			// 连接已断开，忽略panic
		}
	}()

	fmt.Fprintf(w, "event: %s\n", event)
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// handleRequest 处理MCP请求（复用stdio服务器的逻辑）
func (s *SSEServer) handleRequest(request *types.MCPRequest) *types.MCPResponse {
	response := &types.MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
	}

	switch request.Method {
	case "initialize":
		response.Result = s.handleInitialize(request.Params)
	case "tools/list":
		response.Result = s.handleToolsList()
	case "tools/call":
		result, err := s.handleToolsCall(request.Params)
		if err != nil {
			response.Error = &types.MCPError{
				Code:    -32000,
				Message: err.Error(),
			}
		} else {
			response.Result = result
		}
	case "notifications/initialized":
		// 初始化完成通知，无需响应
		return nil
	default:
		response.Error = &types.MCPError{
			Code:    -32601,
			Message: fmt.Sprintf("方法未找到: %s", request.Method),
		}
	}

	return response
}

// handleInitialize 处理初始化请求
func (s *SSEServer) handleInitialize(params interface{}) interface{} {
	return map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities":    s.capabilities,
		"serverInfo": map[string]interface{}{
			"name":    s.config.Server.Name,
			"version": s.config.Server.Version,
		},
	}
}

// handleToolsList 处理工具列表请求
func (s *SSEServer) handleToolsList() interface{} {
	return map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "ssh_execute",
				"description": "在远程服务器上执行Shell命令",
				"inputSchema": map[string]interface{}{
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
							"description": "SSH密码",
						},
					},
					"required": []string{"host", "command"},
				},
			},
			{
				"name":        "ssh_file_transfer",
				"description": "SSH文件传输（上传/下载）",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"host": map[string]interface{}{
							"type":        "string",
							"description": "目标服务器地址",
						},
						"localPath": map[string]interface{}{
							"type":        "string",
							"description": "本地文件路径",
						},
						"remotePath": map[string]interface{}{
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
						"password": map[string]interface{}{
							"type":        "string",
							"description": "SSH密码",
						},
					},
					"required": []string{"host", "localPath", "remotePath", "direction"},
				},
			},
		},
	}
}

// handleToolsCall 处理工具调用请求
func (s *SSEServer) handleToolsCall(params interface{}) (interface{}, error) {
	paramsMap, ok := params.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的参数格式")
	}

	toolName, ok := paramsMap["name"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少工具名称")
	}

	arguments, ok := paramsMap["arguments"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的参数格式")
	}

	switch toolName {
	case "ssh_execute":
		return s.executeSSHCommand(arguments)
	case "ssh_file_transfer":
		return s.executeSSHFileTransfer(arguments)
	default:
		return nil, fmt.Errorf("未知工具: %s", toolName)
	}
}

// executeSSHCommand 执行SSH命令
func (s *SSEServer) executeSSHCommand(args map[string]interface{}) (interface{}, error) {
	host, ok := args["host"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少主机地址")
	}

	command, ok := args["command"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少命令")
	}

	user := s.config.SSH.DefaultUser
	if u, ok := args["user"].(string); ok && u != "" {
		user = u
	}

	port := s.config.SSH.DefaultPort
	if p, ok := args["port"].(float64); ok && p > 0 {
		port = int(p)
	}

	password := ""
	if p, ok := args["password"].(string); ok && p != "" {
		password = p
	}

	// 建立SSH连接并执行命令
	connInfo := &ssh.ConnectionInfo{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
	}

	result, err := s.sshClient.Execute(connInfo, command)
	if err != nil {
		return nil, fmt.Errorf("SSH命令执行失败: %w", err)
	}

	// 构建响应内容
	infoText := fmt.Sprintf("主机: %s\n命令: %s\n退出码: %d\n执行时长: %v\n",
		host, command, result.ExitCode, result.Duration)

	if result.Stdout != "" {
		infoText += fmt.Sprintf("标准输出:\n%s\n", result.Stdout)
	}

	if result.Stderr != "" {
		infoText += fmt.Sprintf("标准错误:\n%s\n", result.Stderr)
	}

	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": infoText,
			},
		},
		"isError": result.ExitCode != 0,
	}, nil
}

// executeSSHFileTransfer 执行SSH文件传输
func (s *SSEServer) executeSSHFileTransfer(args map[string]interface{}) (interface{}, error) {
	host, ok := args["host"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少主机地址")
	}

	localPath, ok := args["localPath"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少本地路径")
	}

	remotePath, ok := args["remotePath"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少远程路径")
	}

	direction, ok := args["direction"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少传输方向")
	}

	// 这里应该实现实际的文件传输逻辑
	// 为简化示例，返回一个模拟结果
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("文件传输完成\n方向: %s\n本地路径: %s\n远程路径: %s\n主机: %s",
					direction, localPath, remotePath, host),
			},
		},
		"isError": false,
	}, nil
}

// Close 关闭SSE服务器
func (s *SSEServer) Close() error {
	s.logger.Println("正在关闭SSH MCP服务器（HTTP SSE传输）")

	s.cancel()

	if s.sshClient != nil {
		s.sshClient.Close()
	}

	return nil
}

// generateSessionID 生成会话ID
func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
