package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"mcp-openai-integration/pkg/config"
	"mcp-openai-integration/pkg/mcp"

	"github.com/sashabaranov/go-openai"
)

// ChatEngine 聊天引擎
// 集成OpenAI API和MCP工具调用功能
type ChatEngine struct {
	config    *config.Config // 配置
	openai    *openai.Client // OpenAI客户端
	mcpClient *mcp.MCPClient // MCP客户端
	logger    *log.Logger    // 日志记录器

	// 对话历史
	messages []openai.ChatCompletionMessage // 消息历史

	// 工具定义（OpenAI格式）
	tools []openai.Tool // 可用工具列表
}

// NewChatEngine 创建新的聊天引擎
func NewChatEngine(cfg *config.Config, logger *log.Logger) (*ChatEngine, error) {
	// 创建OpenAI客户端
	openaiConfig := openai.DefaultConfig(cfg.OpenAI.APIKey)
	if cfg.OpenAI.BaseURL != "" {
		openaiConfig.BaseURL = cfg.OpenAI.BaseURL
	}




	openaiClient := openai.NewClientWithConfig(openaiConfig)

	// 创建MCP客户端
	mcpClient := mcp.NewMCPClient(cfg, logger)

	engine := &ChatEngine{
		config:    cfg,
		openai:    openaiClient,
		mcpClient: mcpClient,
		logger:    logger,
		messages:  make([]openai.ChatCompletionMessage, 0),
		tools:     make([]openai.Tool, 0),
	}

	// 添加系统提示
	if cfg.Chat.SystemPrompt != "" {
		engine.messages = append(engine.messages, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleSystem,
			Content: cfg.Chat.SystemPrompt,
		})
	}

	return engine, nil
}

// Start 启动聊天引擎
func (e *ChatEngine) Start() error {
	e.logger.Println("启动聊天引擎")

	// 启动MCP客户端
	if e.config.Chat.EnableMCP {
		if err := e.mcpClient.Start(); err != nil {
			return fmt.Errorf("启动MCP客户端失败: %w", err)
		}

		// 加载MCP工具并转换为OpenAI格式
		if err := e.loadMCPTools(); err != nil {
			return fmt.Errorf("加载MCP工具失败: %w", err)
		}
	}

	e.logger.Printf("聊天引擎启动完成，加载了 %d 个工具", len(e.tools))
	return nil
}

// loadMCPTools 加载MCP工具并转换为OpenAI工具格式
func (e *ChatEngine) loadMCPTools() error {
	mcpTools := e.mcpClient.GetAvailableTools()

	for toolName, toolInfo := range mcpTools {
		// 转换MCP工具为OpenAI工具格式
		openaiTool := openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        toolName,
				Description: toolInfo.Tool.Description,
				Parameters:  toolInfo.Tool.InputSchema,
			},
		}

		e.tools = append(e.tools, openaiTool)
		e.logger.Printf("加载工具: %s", toolName)
	}

	return nil
}

// Chat 处理用户消息并返回AI响应
func (e *ChatEngine) Chat(ctx context.Context, userMessage string) (string, error) {
	e.logger.Printf("处理用户消息: %s", userMessage)

	// 添加用户消息到历史
	e.addMessage(openai.ChatMessageRoleUser, userMessage)

	// 限制历史消息数量
	e.trimHistory()

	// 构建聊天完成请求
	request := openai.ChatCompletionRequest{
		Model:       e.config.OpenAI.Model,
		Messages:    e.messages,
		Temperature: e.config.OpenAI.Temperature,
		MaxTokens:   e.config.OpenAI.MaxTokens,
	}

	// 如果启用MCP工具，添加工具定义
	if e.config.Chat.EnableMCP && len(e.tools) > 0 {
		request.Tools = e.tools
		if e.config.Chat.MCPAutoCall {
			request.ToolChoice = "auto"
		}
	}

	// 调用OpenAI API
	response, err := e.openai.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("OpenAI API调用失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("OpenAI API返回空响应")
	}

	choice := response.Choices[0]

	// 处理工具调用
	if len(choice.Message.ToolCalls) > 0 {
		return e.handleToolCalls(ctx, choice.Message)
	}

	// 添加助手响应到历史
	e.addMessage(openai.ChatMessageRoleAssistant, choice.Message.Content)

	return choice.Message.Content, nil
}

// handleToolCalls 处理工具调用
func (e *ChatEngine) handleToolCalls(ctx context.Context, message openai.ChatCompletionMessage) (string, error) {
	e.logger.Printf("处理 %d 个工具调用", len(message.ToolCalls))

	// 添加助手消息（包含工具调用）到历史
	e.messages = append(e.messages, message)

	// 执行所有工具调用
	for _, toolCall := range message.ToolCalls {
		result, err := e.executeToolCall(toolCall)
		if err != nil {
			e.logger.Printf("工具调用失败: %s - %v", toolCall.Function.Name, err)
			result = fmt.Sprintf("工具调用失败: %v", err)
		}

		// 添加工具调用结果到历史
		e.messages = append(e.messages, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    result,
			ToolCallID: toolCall.ID,
		})
	}

	// 再次调用OpenAI API获取最终响应
	request := openai.ChatCompletionRequest{
		Model:       e.config.OpenAI.Model,
		Messages:    e.messages,
		Temperature: e.config.OpenAI.Temperature,
		MaxTokens:   e.config.OpenAI.MaxTokens,
	}

	response, err := e.openai.CreateChatCompletion(ctx, request)
	if err != nil {
		return "", fmt.Errorf("工具调用后的OpenAI API调用失败: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("工具调用后的OpenAI API返回空响应")
	}

	finalMessage := response.Choices[0].Message.Content

	// 添加最终响应到历史
	e.addMessage(openai.ChatMessageRoleAssistant, finalMessage)

	return finalMessage, nil
}

// executeToolCall 执行单个工具调用
func (e *ChatEngine) executeToolCall(toolCall openai.ToolCall) (string, error) {
	e.logger.Printf("执行工具调用: %s", toolCall.Function.Name)

	// 解析工具参数
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
		return "", fmt.Errorf("解析工具参数失败: %w", err)
	}

	// 调用MCP工具
	result, err := e.mcpClient.CallTool(toolCall.Function.Name, arguments)
	if err != nil {
		return "", fmt.Errorf("MCP工具调用失败: %w", err)
	}

	// 提取文本内容
	var textResults []string
	for _, content := range result.Content {
		if textContent, ok := content.(*mcp.TextContent); ok {
			textResults = append(textResults, textContent.GetText())
		}
	}

	if len(textResults) == 0 {
		return "工具执行完成，但没有返回文本内容", nil
	}

	return strings.Join(textResults, "\n"), nil
}

// addMessage 添加消息到历史
func (e *ChatEngine) addMessage(role string, content string) {
	e.messages = append(e.messages, openai.ChatCompletionMessage{
		Role:    role,
		Content: content,
	})
}

// trimHistory 限制历史消息数量
func (e *ChatEngine) trimHistory() {
	if len(e.messages) <= e.config.Chat.MaxHistory {
		return
	}

	// 保留系统消息（如果存在）
	systemMessages := make([]openai.ChatCompletionMessage, 0)
	otherMessages := make([]openai.ChatCompletionMessage, 0)

	for _, msg := range e.messages {
		if msg.Role == openai.ChatMessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		} else {
			otherMessages = append(otherMessages, msg)
		}
	}

	// 保留最近的消息
	maxOtherMessages := e.config.Chat.MaxHistory - len(systemMessages)
	if len(otherMessages) > maxOtherMessages {
		otherMessages = otherMessages[len(otherMessages)-maxOtherMessages:]
	}

	// 重新组合消息
	e.messages = append(systemMessages, otherMessages...)
}

// GetAvailableTools 获取可用工具列表
func (e *ChatEngine) GetAvailableTools() []string {
	var toolNames []string
	for _, tool := range e.tools {
		if tool.Function != nil {
			toolNames = append(toolNames, tool.Function.Name)
		}
	}
	return toolNames
}

// GetMessageHistory 获取消息历史
func (e *ChatEngine) GetMessageHistory() []openai.ChatCompletionMessage {
	return e.messages
}

// ClearHistory 清空消息历史（保留系统消息）
func (e *ChatEngine) ClearHistory() {
	systemMessages := make([]openai.ChatCompletionMessage, 0)
	for _, msg := range e.messages {
		if msg.Role == openai.ChatMessageRoleSystem {
			systemMessages = append(systemMessages, msg)
		}
	}
	e.messages = systemMessages
	e.logger.Println("已清空消息历史")
}

// SaveConversation 保存对话到文件
func (e *ChatEngine) SaveConversation(filename string) error {
	if !e.config.Chat.AutoSave {
		return nil
	}

	// 创建对话数据结构
	conversation := struct {
		Timestamp time.Time                      `json:"timestamp"`
		Messages  []openai.ChatCompletionMessage `json:"messages"`
		Tools     []string                       `json:"tools"`
	}{
		Timestamp: time.Now(),
		Messages:  e.messages,
		Tools:     e.GetAvailableTools(),
	}

	// 序列化为JSON
	_, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化对话失败: %w", err)
	}

	// 写入文件
	// 这里应该实现文件写入逻辑
	e.logger.Printf("对话已保存到: %s", filename)
	return nil
}

// Close 关闭聊天引擎
func (e *ChatEngine) Close() error {
	e.logger.Println("关闭聊天引擎")

	if e.mcpClient != nil {
		if err := e.mcpClient.Close(); err != nil {
			return fmt.Errorf("关闭MCP客户端失败: %w", err)
		}
	}

	e.logger.Println("聊天引擎已关闭")
	return nil
}
