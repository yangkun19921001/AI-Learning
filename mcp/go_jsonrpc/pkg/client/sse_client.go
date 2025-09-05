package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ssh-mcp-go-jsonrpc/pkg/types"
)

// SSEClient HTTP SSE传输的MCP客户端
type SSEClient struct {
	serverURL string             // 服务器URL
	sessionID string             // 会话ID
	endpoint  string             // 消息端点
	ctx       context.Context    // 上下文
	cancel    context.CancelFunc // 取消函数
	mutex     sync.RWMutex       // 读写锁
	logger    *log.Logger        // 日志记录器

	// HTTP客户端
	httpClient *http.Client // HTTP客户端

	// 响应管理
	responses map[interface{}]chan types.MCPResponse // 响应通道映射

	// 连接状态
	connected     bool          // 是否已连接
	endpointReady chan struct{} // 端点就绪通知通道
}

// NewSSEClient 创建新的SSE MCP客户端
func NewSSEClient(serverURL string) *SSEClient {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志记录器
	logger := log.New(log.Writer(), "[SSE-MCP-Client] ", log.LstdFlags|log.Lshortfile)

	return &SSEClient{
		serverURL:     serverURL,
		ctx:           ctx,
		cancel:        cancel,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		responses:     make(map[interface{}]chan types.MCPResponse),
		connected:     false,
		endpointReady: make(chan struct{}),
	}
}

// Connect 连接到MCP服务器
func (c *SSEClient) Connect() error {
	c.logger.Printf("连接到MCP服务器: %s", c.serverURL)

	// 建立SSE连接
	sseURL := fmt.Sprintf("%s/mcp/sse", c.serverURL)
	req, err := http.NewRequestWithContext(c.ctx, "GET", sseURL, nil)
	if err != nil {
		return fmt.Errorf("创建SSE请求失败: %w", err)
	}

	// 设置SSE头
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE连接失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return fmt.Errorf("SSE连接失败，状态码: %d", resp.StatusCode)
	}

	// 启动SSE消息读取协程
	go c.readSSEMessages(resp.Body)

	// 等待端点信息
	select {
	case <-c.endpointReady:
		// 端点信息已就绪
	case <-time.After(10 * time.Second):
		return fmt.Errorf("等待端点信息超时")
	case <-c.ctx.Done():
		return fmt.Errorf("连接已取消")
	}

	if c.endpoint == "" {
		return fmt.Errorf("未收到端点信息")
	}

	c.connected = true
	c.logger.Printf("SSE连接建立成功，端点: %s", c.endpoint)
	return nil
}

// readSSEMessages 读取SSE消息
func (c *SSEClient) readSSEMessages(body io.ReadCloser) {
	defer body.Close()

	scanner := bufio.NewScanner(body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// 空行表示事件结束
			currentEvent = ""
			continue
		}

		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			c.handleSSEEvent(currentEvent, data)
		}
	}

	if err := scanner.Err(); err != nil {
		c.logger.Printf("读取SSE消息失败: %v", err)
	}
}

// handleSSEEvent 处理SSE事件
func (c *SSEClient) handleSSEEvent(event, data string) {
	switch event {
	case "endpoint":
		// 收到端点信息
		c.endpoint = data
		c.logger.Printf("收到端点信息: %s", c.endpoint)
		// 通知端点就绪
		select {
		case c.endpointReady <- struct{}{}:
		default:
		}

	case "message":
		// 收到JSON-RPC响应消息
		var response types.MCPResponse
		if err := json.Unmarshal([]byte(data), &response); err != nil {
			c.logger.Printf("解析响应消息失败: %v", err)
			return
		}

		// 路由响应到对应的通道
		c.mutex.RLock()
		if respChan, exists := c.responses[response.ID]; exists {
			select {
			case respChan <- response:
			default:
				c.logger.Printf("响应通道已满，丢弃响应: %v", response.ID)
			}
		} else {
			c.logger.Printf("未找到响应通道: %v", response.ID)
		}
		c.mutex.RUnlock()

	default:
		c.logger.Printf("未知SSE事件: %s, 数据: %s", event, data)
	}
}

// sendRequest 发送JSON-RPC请求
func (c *SSEClient) sendRequest(method string, params interface{}) (*types.MCPResponse, error) {
	if !c.connected {
		return nil, fmt.Errorf("客户端未连接")
	}

	// 生成请求ID
	id := fmt.Sprintf("req-%d", time.Now().UnixNano())

	request := types.MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// 创建响应通道
	respChan := make(chan types.MCPResponse, 1)
	c.mutex.Lock()
	c.responses[id] = respChan
	c.mutex.Unlock()

	// 序列化请求
	data, err := json.Marshal(request)
	if err != nil {
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 发送HTTP POST请求
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.endpoint, bytes.NewReader(data))
	if err != nil {
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mcp-protocol-version", "2025-03-26")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	// 等待响应
	select {
	case response := <-respChan:
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return &response, nil
	case <-time.After(30 * time.Second):
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("请求超时")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("客户端已关闭")
	}
}

// sendNotification 发送JSON-RPC通知（无需响应）
func (c *SSEClient) sendNotification(method string, params interface{}) error {
	if !c.connected {
		return fmt.Errorf("客户端未连接")
	}

	notification := types.MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	// 序列化通知
	data, err := json.Marshal(notification)
	if err != nil {
		return fmt.Errorf("序列化通知失败: %w", err)
	}

	// 发送HTTP POST请求
	req, err := http.NewRequestWithContext(c.ctx, "POST", c.endpoint, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("mcp-protocol-version", "2025-03-26")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	return nil
}

// Initialize 初始化MCP连接
func (c *SSEClient) Initialize() error {
	params := map[string]interface{}{
		"protocolVersion": "2025-03-26",
		"capabilities": map[string]interface{}{
			"sampling": map[string]interface{}{},
		},
		"clientInfo": map[string]interface{}{
			"name":    "SSH-MCP-Client-SSE",
			"version": "1.0.0",
		},
	}

	response, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("初始化失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("服务器初始化错误: %s", response.Error.Message)
	}

	// 发送初始化完成通知
	return c.sendNotification("notifications/initialized", nil)
}

// ListTools 获取工具列表
func (c *SSEClient) ListTools() ([]types.Tool, error) {
	response, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("服务器错误: %s", response.Error.Message)
	}

	result := response.Result.(map[string]interface{})
	toolsData := result["tools"].([]interface{})

	var tools []types.Tool
	for _, toolData := range toolsData {
		toolBytes, _ := json.Marshal(toolData)
		var tool types.Tool
		json.Unmarshal(toolBytes, &tool)
		tools = append(tools, tool)
	}

	return tools, nil
}

// CallTool 调用工具
func (c *SSEClient) CallTool(name string, arguments map[string]interface{}) (*types.MCPResponse, error) {
	params := map[string]interface{}{
		"name":      name,
		"arguments": arguments,
	}

	return c.sendRequest("tools/call", params)
}

// Close 关闭SSE客户端
func (c *SSEClient) Close() error {
	c.logger.Println("关闭SSH MCP客户端（HTTP SSE传输）")

	c.connected = false
	c.cancel()

	// 清理响应通道
	c.mutex.Lock()
	for id, ch := range c.responses {
		close(ch)
		delete(c.responses, id)
	}
	c.mutex.Unlock()

	return nil
}
 