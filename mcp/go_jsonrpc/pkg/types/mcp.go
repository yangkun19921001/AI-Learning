package types

import "encoding/json"

// MCPRequest 表示JSON-RPC 2.0请求对象
// 完全符合JSON-RPC 2.0规范的请求结构
type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`          // 必须为"2.0"
	ID      interface{} `json:"id,omitempty"`     // 请求标识符，通知消息可省略
	Method  string      `json:"method"`           // 要调用的方法名
	Params  interface{} `json:"params,omitempty"` // 方法参数，可省略
}

// MCPResponse 表示JSON-RPC 2.0响应对象
// 成功响应包含result字段，错误响应包含error字段
type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`          // 必须为"2.0"
	ID      interface{} `json:"id,omitempty"`     // 与请求ID一致
	Result  interface{} `json:"result,omitempty"` // 成功时的结果
	Error   *MCPError   `json:"error,omitempty"`  // 错误时的错误信息
}

// MCPError 表示JSON-RPC 2.0错误对象
// 包含标准化的错误代码和消息
type MCPError struct {
	Code    int         `json:"code"`           // 错误代码
	Message string      `json:"message"`        // 错误消息
	Data    interface{} `json:"data,omitempty"` // 额外的错误数据
}

// 标准JSON-RPC 2.0错误代码
const (
	ParseError     = -32700 // 解析错误
	InvalidRequest = -32600 // 无效请求
	MethodNotFound = -32601 // 方法未找到
	InvalidParams  = -32602 // 无效参数
	InternalError  = -32603 // 内部错误
	ServerError    = -32000 // 服务器错误（-32000到-32099为预留范围）
)

// MCP协议特定的消息结构

// InitializeParams MCP初始化参数
type InitializeParams struct {
	ProtocolVersion string             `json:"protocolVersion"` // MCP协议版本
	Capabilities    ClientCapabilities `json:"capabilities"`    // 客户端能力
	ClientInfo      ClientInfo         `json:"clientInfo"`      // 客户端信息
}

// ClientCapabilities 客户端能力声明
type ClientCapabilities struct {
	Sampling  *SamplingCapability `json:"sampling,omitempty"`  // 采样能力
	Tools     bool                `json:"tools,omitempty"`     // 工具支持
	Prompts   bool                `json:"prompts,omitempty"`   // 提示支持
	Resources bool                `json:"resources,omitempty"` // 资源支持
	Logging   bool                `json:"logging,omitempty"`   // 日志支持
	Roots     *RootsCapability    `json:"roots,omitempty"`     // 根目录能力
}

// SamplingCapability 采样能力配置
type SamplingCapability struct{}

// RootsCapability 根目录能力配置
type RootsCapability struct {
	ListChanged bool `json:"listChanged"` // 支持列表变化通知
}

// ClientInfo 客户端信息
type ClientInfo struct {
	Name    string `json:"name"`    // 客户端名称
	Version string `json:"version"` // 客户端版本
}

// InitializeResult MCP初始化响应结果
type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"` // 服务器支持的协议版本
	Capabilities    ServerCapabilities `json:"capabilities"`    // 服务器能力
	ServerInfo      ServerInfo         `json:"serverInfo"`      // 服务器信息
}

// ServerCapabilities 服务器能力声明
type ServerCapabilities struct {
	Tools     *ToolsCapability     `json:"tools,omitempty"`     // 工具能力
	Resources *ResourcesCapability `json:"resources,omitempty"` // 资源能力
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`   // 提示能力
	Logging   *LoggingCapability   `json:"logging,omitempty"`   // 日志能力
}

// ToolsCapability 工具能力配置
type ToolsCapability struct {
	ListChanged bool `json:"listChanged"` // 支持工具列表变化通知
}

// ResourcesCapability 资源能力配置
type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe"`   // 支持资源订阅
	ListChanged bool `json:"listChanged"` // 支持资源列表变化通知
}

// PromptsCapability 提示能力配置
type PromptsCapability struct {
	ListChanged bool `json:"listChanged"` // 支持提示列表变化通知
}

// LoggingCapability 日志能力配置
type LoggingCapability struct{}

// ServerInfo 服务器信息
type ServerInfo struct {
	Name    string `json:"name"`    // 服务器名称
	Version string `json:"version"` // 服务器版本
}

// Tool 工具定义
type Tool struct {
	Name        string                 `json:"name"`        // 工具名称
	Description string                 `json:"description"` // 工具描述
	InputSchema map[string]interface{} `json:"inputSchema"` // 输入参数JSON Schema
}

// ToolsListResult 工具列表响应结果
type ToolsListResult struct {
	Tools []Tool `json:"tools"` // 工具列表
}

// ToolCallParams 工具调用参数
type ToolCallParams struct {
	Name      string                 `json:"name"`            // 工具名称
	Arguments map[string]interface{} `json:"arguments"`       // 工具参数
	Meta      *ToolCallMeta          `json:"_meta,omitempty"` // 元数据
}

// ToolCallMeta 工具调用元数据
type ToolCallMeta struct {
	ProgressToken interface{} `json:"progressToken,omitempty"` // 进度跟踪令牌
}

// ToolCallResult 工具调用结果
type ToolCallResult struct {
	Content []Content `json:"content"` // 结果内容
	IsError bool      `json:"isError"` // 是否为错误结果
}

// UnmarshalJSON 自定义JSON反序列化，处理Content接口类型
func (t *ToolCallResult) UnmarshalJSON(data []byte) error {
	var temp struct {
		Content []json.RawMessage `json:"content"`
		IsError bool              `json:"isError"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	t.IsError = temp.IsError
	t.Content = make([]Content, len(temp.Content))

	for i, rawContent := range temp.Content {
		var wrapper ContentWrapper
		if err := json.Unmarshal(rawContent, &wrapper); err != nil {
			return err
		}
		t.Content[i] = wrapper.Content
	}

	return nil
}

// Content 内容接口，支持多种内容类型
type Content interface {
	GetType() string
}

// TextContent 文本内容
type TextContent struct {
	Type string `json:"type"` // 内容类型，固定为"text"
	Text string `json:"text"` // 文本内容
}

func (t *TextContent) GetType() string {
	return "text"
}

// ImageContent 图片内容
type ImageContent struct {
	Type     string `json:"type"`     // 内容类型，固定为"image"
	Data     string `json:"data"`     // 图片数据（base64编码）
	MimeType string `json:"mimeType"` // MIME类型
}

func (i *ImageContent) GetType() string {
	return "image"
}

// ContentWrapper 内容包装器，用于JSON序列化/反序列化
type ContentWrapper struct {
	Content
}

// UnmarshalJSON 自定义JSON反序列化，支持多种内容类型
func (c *ContentWrapper) UnmarshalJSON(data []byte) error {
	var temp struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	switch temp.Type {
	case "text":
		var textContent TextContent
		if err := json.Unmarshal(data, &textContent); err != nil {
			return err
		}
		c.Content = &textContent
	case "image":
		var imageContent ImageContent
		if err := json.Unmarshal(data, &imageContent); err != nil {
			return err
		}
		c.Content = &imageContent
	default:
		// 默认处理为文本内容
		var textContent TextContent
		if err := json.Unmarshal(data, &textContent); err != nil {
			return err
		}
		c.Content = &textContent
	}

	return nil
}

// SSH相关的参数结构

// SSHExecuteParams SSH命令执行参数
type SSHExecuteParams struct {
	Host     string `json:"host"`               // 目标主机地址
	Command  string `json:"command"`            // 要执行的命令
	User     string `json:"user,omitempty"`     // SSH用户名，默认为root
	Port     int    `json:"port,omitempty"`     // SSH端口，默认为22
	Timeout  int    `json:"timeout,omitempty"`  // 超时时间（秒），默认为30
	Password string `json:"password,omitempty"` // SSH密码（可选）
}

// SSHFileTransferParams SSH文件传输参数
type SSHFileTransferParams struct {
	Host       string `json:"host"`           // 目标主机地址
	LocalPath  string `json:"localPath"`      // 本地文件路径
	RemotePath string `json:"remotePath"`     // 远程文件路径
	User       string `json:"user,omitempty"` // SSH用户名
	Port       int    `json:"port,omitempty"` // SSH端口
	Direction  string `json:"direction"`      // 传输方向：upload/download
}
 