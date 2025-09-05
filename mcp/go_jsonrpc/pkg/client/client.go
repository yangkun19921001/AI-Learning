package client

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

	"ssh-mcp-go-jsonrpc/pkg/types"
)

// MCPClient MCP客户端实现
// 负责与MCP服务器进行通信，管理连接生命周期和消息路由
type MCPClient struct {
	// 进程管理
	cmd    *exec.Cmd      // 服务器进程
	stdin  io.WriteCloser // 向服务器写入数据的管道
	stdout io.ReadCloser  // 从服务器读取数据的管道
	stderr io.ReadCloser  // 服务器错误输出管道

	// 消息管理
	responses map[interface{}]chan *types.MCPResponse // 响应通道映射
	mutex     sync.RWMutex                            // 保护响应映射的读写锁

	// 状态管理
	ctx    context.Context    // 上下文
	cancel context.CancelFunc // 取消函数
	logger *log.Logger        // 日志记录器

	// 客户端信息
	clientInfo types.ClientInfo // 客户端信息

	// 服务器能力
	serverCapabilities *types.ServerCapabilities // 服务器能力
	serverInfo         *types.ServerInfo         // 服务器信息
	protocolVersion    string                    // 协议版本
}

// Config 客户端配置
type Config struct {
	ServerCommand []string         // 服务器启动命令
	ClientInfo    types.ClientInfo // 客户端信息
	Timeout       time.Duration    // 请求超时时间
}

// NewMCPClient 创建新的MCP客户端
func NewMCPClient(config *Config) *MCPClient {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建日志记录器
	logger := log.New(log.Writer(), "[MCP-Client] ", log.LstdFlags|log.Lshortfile)

	return &MCPClient{
		responses:  make(map[interface{}]chan *types.MCPResponse),
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
		clientInfo: config.ClientInfo,
	}
}

// Connect 连接到MCP服务器
func (c *MCPClient) Connect(serverCommand []string) error {
	c.logger.Printf("启动MCP服务器: %v", serverCommand)

	// 创建服务器进程
	c.cmd = exec.CommandContext(c.ctx, serverCommand[0], serverCommand[1:]...)

	var err error

	// 创建输入管道
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建stdin管道失败: %w", err)
	}

	// 创建输出管道
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("创建stdout管道失败: %w", err)
	}

	// 创建错误输出管道
	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动服务器进程
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("启动服务器进程失败: %w", err)
	}

	// 启动消息读取协程
	go c.readMessages()
	go c.readErrors()

	c.logger.Println("MCP服务器连接成功")
	return nil
}

// Initialize 初始化MCP连接
func (c *MCPClient) Initialize() error {
	c.logger.Println("开始MCP初始化")

	// 构建初始化参数
	params := types.InitializeParams{
		ProtocolVersion: "2025-03-26",
		Capabilities: types.ClientCapabilities{
			Sampling: &types.SamplingCapability{},
			Tools:    true,
		},
		ClientInfo: c.clientInfo,
	}

	// 发送初始化请求
	response, err := c.sendRequest("initialize", params)
	if err != nil {
		return fmt.Errorf("初始化请求失败: %w", err)
	}

	if response.Error != nil {
		return fmt.Errorf("服务器初始化错误: %s", response.Error.Message)
	}

	// 解析初始化结果
	var result types.InitializeResult
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return fmt.Errorf("序列化初始化结果失败: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return fmt.Errorf("解析初始化结果失败: %w", err)
	}

	// 保存服务器信息
	c.serverCapabilities = &result.Capabilities
	c.serverInfo = &result.ServerInfo
	c.protocolVersion = result.ProtocolVersion

	c.logger.Printf("服务器信息: %s v%s", result.ServerInfo.Name, result.ServerInfo.Version)
	c.logger.Printf("协议版本: %s", result.ProtocolVersion)

	// 发送初始化完成通知
	if err := c.sendNotification("notifications/initialized", nil); err != nil {
		return fmt.Errorf("发送初始化完成通知失败: %w", err)
	}

	c.logger.Println("MCP初始化完成")
	return nil
}

// ListTools 获取工具列表
func (c *MCPClient) ListTools() ([]types.Tool, error) {
	c.logger.Println("获取工具列表")

	response, err := c.sendRequest("tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("服务器错误: %s", response.Error.Message)
	}

	// 解析工具列表结果
	var result types.ToolsListResult
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("序列化工具列表结果失败: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("解析工具列表结果失败: %w", err)
	}

	c.logger.Printf("获取到 %d 个工具", len(result.Tools))
	return result.Tools, nil
}

// CallTool 调用工具
func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) (*types.ToolCallResult, error) {
	c.logger.Printf("调用工具: %s", name)

	params := types.ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	response, err := c.sendRequest("tools/call", params)
	if err != nil {
		return nil, fmt.Errorf("工具调用失败: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf("工具调用错误: %s", response.Error.Message)
	}

	// 解析工具调用结果
	var result types.ToolCallResult
	resultBytes, err := json.Marshal(response.Result)
	if err != nil {
		return nil, fmt.Errorf("序列化工具调用结果失败: %w", err)
	}

	if err := json.Unmarshal(resultBytes, &result); err != nil {
		return nil, fmt.Errorf("解析工具调用结果失败: %w", err)
	}

	c.logger.Printf("工具调用完成: %s", name)
	return &result, nil
}

// sendRequest 发送请求并等待响应
func (c *MCPClient) sendRequest(method string, params interface{}) (*types.MCPResponse, error) {
	// 生成请求ID
	id := fmt.Sprintf("req-%d", time.Now().UnixNano())

	// 构建请求
	request := types.MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// 创建响应通道
	respChan := make(chan *types.MCPResponse, 1)
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

	c.logger.Printf("发送请求: %s", string(data))

	// 发送请求
	if _, err := fmt.Fprintf(c.stdin, "%s\n", string(data)); err != nil {
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	// 等待响应
	select {
	case response := <-respChan:
		return response, nil
	case <-time.After(30 * time.Second):
		c.mutex.Lock()
		delete(c.responses, id)
		c.mutex.Unlock()
		return nil, fmt.Errorf("请求超时")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("客户端已关闭")
	}
}

// sendNotification 发送通知（无需响应）
func (c *MCPClient) sendNotification(method string, params interface{}) error {
	// 构建通知（无ID字段）
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

	c.logger.Printf("发送通知: %s", string(data))

	// 发送通知
	if _, err := fmt.Fprintf(c.stdin, "%s\n", string(data)); err != nil {
		return fmt.Errorf("发送通知失败: %w", err)
	}

	return nil
}

// readMessages 读取服务器消息
func (c *MCPClient) readMessages() {
	scanner := bufio.NewScanner(c.stdout)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		c.logger.Printf("收到响应: %s", line)

		// 解析响应
		var response types.MCPResponse
		if err := json.Unmarshal([]byte(line), &response); err != nil {
			c.logger.Printf("解析响应失败: %v", err)
			continue
		}

		// 路由响应到对应的通道
		c.mutex.RLock()
		if respChan, exists := c.responses[response.ID]; exists {
			select {
			case respChan <- &response:
			default:
				c.logger.Printf("响应通道已满，丢弃响应: %v", response.ID)
			}
		} else {
			c.logger.Printf("未找到对应的响应通道: %v", response.ID)
		}
		c.mutex.RUnlock()
	}

	if err := scanner.Err(); err != nil {
		c.logger.Printf("读取消息失败: %v", err)
	}
}

// readErrors 读取服务器错误输出
func (c *MCPClient) readErrors() {
	scanner := bufio.NewScanner(c.stderr)

	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			// 区分错误和正常日志
			if strings.Contains(line, "ERROR") || strings.Contains(line, "FATAL") || strings.Contains(line, "错误") {
				c.logger.Printf("服务器错误: %s", line)
			} else {
				c.logger.Printf("服务器日志: %s", line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		c.logger.Printf("读取错误输出失败: %v", err)
	}
}

// GetServerInfo 获取服务器信息
func (c *MCPClient) GetServerInfo() *types.ServerInfo {
	return c.serverInfo
}

// GetServerCapabilities 获取服务器能力
func (c *MCPClient) GetServerCapabilities() *types.ServerCapabilities {
	return c.serverCapabilities
}

// GetProtocolVersion 获取协议版本
func (c *MCPClient) GetProtocolVersion() string {
	return c.protocolVersion
}

// Close 关闭客户端连接
func (c *MCPClient) Close() error {
	c.logger.Println("关闭MCP客户端")

	// 取消上下文
	c.cancel()

	// 关闭输入输出流
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	// 等待进程结束
	if c.cmd != nil && c.cmd.Process != nil {
		if err := c.cmd.Wait(); err != nil {
			c.logger.Printf("等待服务器进程结束失败: %v", err)
		}
	}

	// 清理响应通道
	c.mutex.Lock()
	for id, ch := range c.responses {
		close(ch)
		delete(c.responses, id)
	}
	c.mutex.Unlock()

	c.logger.Println("MCP客户端已关闭")
	return nil
}

// IsConnected 检查是否已连接
func (c *MCPClient) IsConnected() bool {
	return c.cmd != nil && c.cmd.Process != nil
}
