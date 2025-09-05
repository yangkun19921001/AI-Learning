package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"mcp-openai-integration/pkg/config"
)

// MCPRequest JSON-RPC 2.0请求结构
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// MCPResponse JSON-RPC 2.0响应结构
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError JSON-RPC 2.0错误结构
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool MCP工具定义
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []interface{} `json:"content"`
	IsError bool          `json:"isError"`
}

// Content 内容接口
type Content interface {
	GetType() string
	GetText() string
}

// TextContent 文本内容
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (t *TextContent) GetType() string {
	return "text"
}

func (t *TextContent) GetText() string {
	return t.Text
}

// MCPClient MCP客户端管理器
// 负责管理多个MCP服务器连接和工具调用
type MCPClient struct {
	servers map[string]*ServerConnection // 服务器连接映射
	tools   map[string]*ToolInfo         // 工具信息映射
	config  *config.Config               // 配置
	ctx     context.Context              // 上下文
	cancel  context.CancelFunc           // 取消函数
	mutex   sync.RWMutex                 // 读写锁
	logger  *log.Logger                  // 日志记录器
}

// ServerConnection 服务器连接信息
type ServerConnection struct {
	Name      string                            // 服务器名称
	Config    config.MCPServerConfig            // 服务器配置
	Cmd       *exec.Cmd                         // 进程命令
	Stdin     io.WriteCloser                    // 标准输入
	Stdout    io.ReadCloser                     // 标准输出
	Stderr    io.ReadCloser                     // 标准错误
	Responses map[interface{}]chan *MCPResponse // 响应通道映射
	Mutex     sync.RWMutex                      // 响应映射锁
	Connected bool                              // 连接状态
}

// ToolInfo 工具信息
type ToolInfo struct {
	Tool       Tool   // 工具定义
	ServerName string // 所属服务器名称
}

// NewMCPClient 创建新的MCP客户端
func NewMCPClient(cfg *config.Config, logger *log.Logger) *MCPClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &MCPClient{
		servers: make(map[string]*ServerConnection),
		tools:   make(map[string]*ToolInfo),
		config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
		logger:  logger,
	}
}

// Start 启动MCP客户端，连接所有启用的服务器
func (c *MCPClient) Start() error {
	c.logger.Println("启动MCP客户端管理器")

	enabledServers := c.config.GetEnabledMCPServers()
	if len(enabledServers) == 0 {
		return fmt.Errorf("没有启用的MCP服务器")
	}

	// 并行连接所有服务器
	var wg sync.WaitGroup
	errors := make(chan error, len(enabledServers))

	for _, serverConfig := range enabledServers {
		wg.Add(1)
		go func(cfg config.MCPServerConfig) {
			defer wg.Done()
			if err := c.connectServer(cfg); err != nil {
				errors <- fmt.Errorf("连接服务器 %s 失败: %w", cfg.Name, err)
			}
		}(serverConfig)
	}

	wg.Wait()
	close(errors)

	// 检查连接错误
	var connectErrors []error
	for err := range errors {
		connectErrors = append(connectErrors, err)
	}

	if len(connectErrors) > 0 {
		c.logger.Printf("部分服务器连接失败: %v", connectErrors)
		// 如果所有服务器都连接失败，返回错误
		if len(connectErrors) == len(enabledServers) {
			return fmt.Errorf("所有MCP服务器连接失败")
		}
	}

	c.logger.Printf("MCP客户端启动完成，成功连接 %d 个服务器", len(c.servers))
	return nil
}

// connectServer 连接单个MCP服务器
func (c *MCPClient) connectServer(serverConfig config.MCPServerConfig) error {
	c.logger.Printf("连接MCP服务器: %s", serverConfig.Name)

	// 创建服务器进程
	args := append([]string{}, serverConfig.Args...)
	cmd := exec.CommandContext(c.ctx, serverConfig.Command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin管道失败: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout管道失败: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动进程
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动服务器进程失败: %w", err)
	}

	// 创建服务器连接
	conn := &ServerConnection{
		Name:      serverConfig.Name,
		Config:    serverConfig,
		Cmd:       cmd,
		Stdin:     stdin,
		Stdout:    stdout,
		Stderr:    stderr,
		Responses: make(map[interface{}]chan *MCPResponse),
		Connected: false,
	}

	// 启动消息读取协程
	go c.readServerMessages(conn)
	go c.readServerErrors(conn)

	// 执行MCP初始化握手
	if err := c.initializeServer(conn); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("初始化服务器失败: %w", err)
	}

	// 获取服务器工具列表
	if err := c.loadServerTools(conn); err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("加载服务器工具失败: %w", err)
	}

	// 保存连接
	c.mutex.Lock()
	c.servers[serverConfig.Name] = conn
	conn.Connected = true
	c.mutex.Unlock()

	c.logger.Printf("成功连接到MCP服务器: %s", serverConfig.Name)
	return nil
}

// initializeServer 初始化MCP服务器连接
func (c *MCPClient) initializeServer(conn *ServerConnection) error {
	// 发送初始化请求
	params := map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]interface{}{
			"sampling": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "MCP-OpenAI-Integration",
			"version": "1.0.0",
		},
	}

	response, err := c.sendServerRequest(conn, "initialize", params)
	if err != nil {
		return fmt.Errorf("发送初始化请求失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("服务器初始化错误: %s", response.Error.Message)
	}

	// 发送初始化完成通知
	if err := c.sendServerNotification(conn, "notifications/initialized", nil); err != nil {
		return fmt.Errorf("发送初始化完成通知失败: %w", err)
	}

	return nil
}

// loadServerTools 加载服务器工具列表
func (c *MCPClient) loadServerTools(conn *ServerConnection) error {
	response, err := c.sendServerRequest(conn, "tools/list", nil)
	if err != nil {
		return fmt.Errorf("获取工具列表失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("服务器工具列表错误: %s", response.Error.Message)
	}

	// 解析工具列表
	result, ok := response.Result.(map[string]interface{})
	if !ok {
		return fmt.Errorf("无效的工具列表响应格式")
	}

	toolsData, ok := result["tools"].([]interface{})
	if !ok {
		return fmt.Errorf("无效的工具数据格式")
	}

	// 注册工具
	c.mutex.Lock()
	for _, toolData := range toolsData {
		toolBytes, err := json.Marshal(toolData)
		if err != nil {
			continue
		}

		var tool Tool
		if err := json.Unmarshal(toolBytes, &tool); err != nil {
			continue
		}

		// 使用服务器名称前缀避免工具名冲突
		toolKey := fmt.Sprintf("%s.%s", conn.Name, tool.Name)
		c.tools[toolKey] = &ToolInfo{
			Tool:       tool,
			ServerName: conn.Name,
		}

		c.logger.Printf("注册工具: %s (来自服务器: %s)", toolKey, conn.Name)
	}
	c.mutex.Unlock()

	return nil
}

// sendServerRequest 向服务器发送请求并等待响应
func (c *MCPClient) sendServerRequest(conn *ServerConnection, method string, params interface{}) (*MCPResponse, error) {
	id := fmt.Sprintf("req-%d", time.Now().UnixNano())

	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// 创建响应通道
	respChan := make(chan *MCPResponse, 1)
	conn.Mutex.Lock()
	conn.Responses[id] = respChan
	conn.Mutex.Unlock()

	// 序列化并发送请求
	data, err := json.Marshal(request)
	if err != nil {
		conn.Mutex.Lock()
		delete(conn.Responses, id)
		conn.Mutex.Unlock()
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	if _, err := fmt.Fprintf(conn.Stdin, "%s\n", string(data)); err != nil {
		conn.Mutex.Lock()
		delete(conn.Responses, id)
		conn.Mutex.Unlock()
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待响应
	select {
	case response := <-respChan:
		return response, nil
	case <-time.After(c.config.MCP.Timeout):
		conn.Mutex.Lock()
		delete(conn.Responses, id)
		conn.Mutex.Unlock()
		return nil, fmt.Errorf("请求超时")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("客户端已关闭")
	}
}

// sendServerNotification 向服务器发送通知（无需响应）
func (c *MCPClient) sendServerNotification(conn *ServerConnection, method string, params interface{}) error {
	notification := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("序列化通知失败: %w", err)
	}

	if _, err := fmt.Fprintf(conn.Stdin, "%s\n", string(data)); err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}

	return nil
}

// readServerMessages 读取服务器消息
func (c *MCPClient) readServerMessages(conn *ServerConnection) {
	scanner := bufio.NewScanner(conn.Stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var response MCPResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			c.logger.Printf("解析服务器 %s 响应失败: %v", conn.Name, err)
			continue
		}

		// 路由响应到对应的通道
		conn.Mutex.RLock()
		if respChan, exists := conn.Responses[response.ID]; exists {
			select {
			case respChan <- &response:
			default:
				c.logger.Printf("服务器 %s 响应通道已满，丢弃响应: %v", conn.Name, response.ID)
			}
		}
		conn.Mutex.RUnlock()
	}

	if err := scanner.Err(); err != nil {
		c.logger.Printf("读取服务器 %s 消息失败: %v", conn.Name, err)
	}
}

// readServerErrors 读取服务器错误输出
func (c *MCPClient) readServerErrors(conn *ServerConnection) {
	scanner := bufio.NewScanner(conn.Stderr)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// 区分真正的错误和正常的日志信息
			if c.isErrorMessage(line) {
				c.logger.Printf("服务器 %s 错误: %s", conn.Name, line)
			} else {
				c.logger.Printf("服务器 %s 日志: %s", conn.Name, line)
			}
		}
	}
}

// isErrorMessage 判断是否为错误消息
func (c *MCPClient) isErrorMessage(message string) bool {
	errorKeywords := []string{
		"ERROR", "FATAL", "PANIC", "error:", "fatal:", "panic:",
		"错误", "致命", "失败", "异常", "Error:", "Fatal:", "Panic:",
	}

	for _, keyword := range errorKeywords {
		if len(message) > 0 && (message[0:1] == keyword[0:1] ||
			strings.Contains(strings.ToLower(message), strings.ToLower(keyword))) {
			return true
		}
	}
	return false
}

// GetAvailableTools 获取所有可用工具
func (c *MCPClient) GetAvailableTools() map[string]*ToolInfo {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	tools := make(map[string]*ToolInfo)
	for name, tool := range c.tools {
		tools[name] = tool
	}
	return tools
}

// CallTool 调用指定工具
func (c *MCPClient) CallTool(toolName string, arguments map[string]interface{}) (*ToolCallResult, error) {
	c.mutex.RLock()
	toolInfo, exists := c.tools[toolName]
	if !exists {
		c.mutex.RUnlock()
		return nil, fmt.Errorf("工具 %s 不存在", toolName)
	}

	conn, exists := c.servers[toolInfo.ServerName]
	if !exists || !conn.Connected {
		c.mutex.RUnlock()
		return nil, fmt.Errorf("服务器 %s 未连接", toolInfo.ServerName)
	}
	c.mutex.RUnlock()

	// 调用工具
	params := map[string]interface{}{
		"name":      toolInfo.Tool.Name, // 使用原始工具名称
		"arguments": arguments,
	}

	response, err := c.sendServerRequest(conn, "tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("调用工具失败: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("工具调用错误: %s", response.Error.Message)
	}

	// 解析工具调用结果
	var result ToolCallResult
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("序列化工具结果失败: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("解析工具结果失败: %w", err)
	}

	// 转换内容类型
	var contents []interface{}
	for _, contentData := range result.Content {
		if contentMap, ok := contentData.(map[string]interface{}); ok {
			if contentType, ok := contentMap["type"].(string); ok && contentType == "text" {
				if text, ok := contentMap["text"].(string); ok {
					contents = append(contents, &TextContent{
						Type: "text",
						Text: text,
					})
				}
			}
		}
	}

	return &ToolCallResult{
		Content: contents,
		IsError: result.IsError,
	}, nil
}

// Close 关闭MCP客户端，断开所有服务器连接
func (c *MCPClient) Close() error {
	c.logger.Println("关闭MCP客户端管理器")

	c.cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	var errors []error
	for name, conn := range c.servers {
		if err := c.closeServerConnection(conn); err != nil {
			errors = append(errors, fmt.Errorf("关闭服务器 %s 失败: %w", name, err))
		}
	}

	// 清空连接和工具
	c.servers = make(map[string]*ServerConnection)
	c.tools = make(map[string]*ToolInfo)

	if len(errors) > 0 {
		return fmt.Errorf("关闭连接时发生错误: %v", errors)
	}

	c.logger.Println("MCP客户端管理器已关闭")
	return nil
}

// closeServerConnection 关闭单个服务器连接
func (c *MCPClient) closeServerConnection(conn *ServerConnection) error {
	conn.Connected = false

	// 关闭输入输出流
	if conn.Stdin != nil {
		conn.Stdin.Close()
	}
	if conn.Stdout != nil {
		conn.Stdout.Close()
	}
	if conn.Stderr != nil {
		conn.Stderr.Close()
	}

	// 终止进程
	if conn.Cmd != nil && conn.Cmd.Process != nil {
		if err := conn.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("终止进程失败: %w", err)
		}
		conn.Cmd.Wait()
	}

	// 清理响应通道
	conn.Mutex.Lock()
	for id, ch := range conn.Responses {
		close(ch)
		delete(conn.Responses, id)
	}
	conn.Mutex.Unlock()

	return nil
}
