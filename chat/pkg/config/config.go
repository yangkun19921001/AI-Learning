package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用程序配置结构
type Config struct {
	OpenAI OpenAIConfig `yaml:"openai"` // OpenAI配置
	MCP    MCPConfig    `yaml:"mcp"`    // MCP配置
	Chat   ChatConfig   `yaml:"chat"`   // 聊天配置
	Log    LogConfig    `yaml:"log"`    // 日志配置
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
	APIKey      string        `yaml:"api_key"`     // OpenAI API密钥
	BaseURL     string        `yaml:"base_url"`    // API基础URL，支持自定义端点
	Model       string        `yaml:"model"`       // 使用的模型名称
	Temperature float32       `yaml:"temperature"` // 温度参数
	MaxTokens   int           `yaml:"max_tokens"`  // 最大令牌数
	Timeout     time.Duration `yaml:"timeout"`     // 请求超时时间
}

// MCPConfig MCP客户端配置
type MCPConfig struct {
	Servers []MCPServerConfig `yaml:"servers"` // MCP服务器列表
	Timeout time.Duration     `yaml:"timeout"` // MCP请求超时时间
}

// MCPServerConfig MCP服务器配置
type MCPServerConfig struct {
	Name        string   `yaml:"name"`        // 服务器名称
	Command     string   `yaml:"command"`     // 服务器启动命令
	Args        []string `yaml:"args"`        // 命令参数
	Description string   `yaml:"description"` // 服务器描述
	Enabled     bool     `yaml:"enabled"`     // 是否启用
}

// ChatConfig 聊天配置
type ChatConfig struct {
	MaxHistory   int    `yaml:"max_history"`   // 最大历史记录数
	SystemPrompt string `yaml:"system_prompt"` // 系统提示词
	AutoSave     bool   `yaml:"auto_save"`     // 是否自动保存对话
	SavePath     string `yaml:"save_path"`     // 对话保存路径
	EnableMCP    bool   `yaml:"enable_mcp"`    // 是否启用MCP工具
	MCPAutoCall  bool   `yaml:"mcp_auto_call"` // 是否自动调用MCP工具
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`       // 日志级别：debug, info, warn, error
	File       string `yaml:"file"`        // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 日志文件最大大小（MB）
	MaxBackups int    `yaml:"max_backups"` // 保留的日志文件数量
	MaxAge     int    `yaml:"max_age"`     // 日志文件保留天数
	Compress   bool   `yaml:"compress"`    // 是否压缩旧日志文件
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		OpenAI: OpenAIConfig{
			APIKey:      "", // 需要从环境变量或配置文件设置
			BaseURL:     "https://api.openai.com/v1",
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   2000,
			Timeout:     30 * time.Second,
		},
		MCP: MCPConfig{
			Servers: []MCPServerConfig{
				{
					Name:        "ssh-jsonrpc",
					Command:     "../go_jsonrpc/build/ssh-mcp-server",
					Args:        []string{"-config", "../go_jsonrpc/config.yaml"},
					Description: "基于JSON-RPC实现的SSH MCP服务器",
					Enabled:     true,
				},
				{
					Name:        "ssh-sdk",
					Command:     "../go-sdk/build/ssh-mcp-server-sdk",
					Args:        []string{"-config", "../go-sdk/config.yaml"},
					Description: "基于官方SDK实现的SSH MCP服务器",
					Enabled:     false,
				},
			},
			Timeout: 30 * time.Second,
		},
		Chat: ChatConfig{
			MaxHistory:   20,
			SystemPrompt: "你是一个智能助手，可以通过MCP工具执行SSH命令和文件操作。请根据用户需求选择合适的工具来完成任务。",
			AutoSave:     true,
			SavePath:     "./conversations",
			EnableMCP:    true,
			MCPAutoCall:  true,
		},
		Log: LogConfig{
			Level:      "info",
			File:       "/tmp/mcp-openai-chat.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*Config, error) {
	// 如果配置文件不存在，使用默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := DefaultConfig()
		// 尝试从环境变量获取OpenAI API密钥
		if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
			config.OpenAI.APIKey = apiKey
		}
		return config, nil
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析YAML配置
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖API密钥（如果存在）
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.OpenAI.APIKey = apiKey
	}

	// 扩展路径中的波浪号
	if err := config.expandPaths(); err != nil {
		return nil, fmt.Errorf("扩展配置路径失败: %w", err)
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func (c *Config) SaveConfig(configPath string) error {
	// 确保配置目录存在
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	// 序列化配置为YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 写入配置文件
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// expandPaths 扩展配置中的路径（处理~符号）
func (c *Config) expandPaths() error {
	var err error

	// 扩展对话保存路径
	if c.Chat.SavePath != "" {
		c.Chat.SavePath, err = expandPath(c.Chat.SavePath)
		if err != nil {
			return fmt.Errorf("扩展对话保存路径失败: %w", err)
		}
	}

	// 扩展日志文件路径
	if c.Log.File != "" {
		c.Log.File, err = expandPath(c.Log.File)
		if err != nil {
			return fmt.Errorf("扩展日志文件路径失败: %w", err)
		}
	}

	// 扩展MCP服务器命令路径
	for i := range c.MCP.Servers {
		if c.MCP.Servers[i].Command != "" {
			c.MCP.Servers[i].Command, err = expandPath(c.MCP.Servers[i].Command)
			if err != nil {
				return fmt.Errorf("扩展MCP服务器命令路径失败: %w", err)
			}
		}
	}

	return nil
}

// expandPath 扩展路径中的波浪号为用户主目录
func expandPath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	if len(path) == 1 {
		return homeDir, nil
	}

	return filepath.Join(homeDir, path[2:]), nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	// 验证OpenAI配置
	if c.OpenAI.APIKey == "" {
		return fmt.Errorf("OpenAI API密钥不能为空，请设置OPENAI_API_KEY环境变量或在配置文件中指定")
	}
	if c.OpenAI.Model == "" {
		return fmt.Errorf("OpenAI模型不能为空")
	}
	if c.OpenAI.Temperature < 0 || c.OpenAI.Temperature > 2 {
		return fmt.Errorf("OpenAI温度参数必须在0-2范围内")
	}
	if c.OpenAI.MaxTokens <= 0 {
		return fmt.Errorf("OpenAI最大令牌数必须大于0")
	}

	// 验证MCP配置
	if len(c.MCP.Servers) == 0 {
		return fmt.Errorf("至少需要配置一个MCP服务器")
	}

	enabledCount := 0
	for _, server := range c.MCP.Servers {
		if server.Enabled {
			enabledCount++
			if server.Name == "" {
				return fmt.Errorf("MCP服务器名称不能为空")
			}
			if server.Command == "" {
				return fmt.Errorf("MCP服务器命令不能为空")
			}
		}
	}

	if enabledCount == 0 {
		return fmt.Errorf("至少需要启用一个MCP服务器")
	}

	// 验证聊天配置
	if c.Chat.MaxHistory <= 0 {
		return fmt.Errorf("最大历史记录数必须大于0")
	}

	// 验证日志配置
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Log.Level] {
		return fmt.Errorf("无效的日志级别: %s", c.Log.Level)
	}

	return nil
}

// GetEnabledMCPServers 获取启用的MCP服务器配置
func (c *Config) GetEnabledMCPServers() []MCPServerConfig {
	var enabled []MCPServerConfig
	for _, server := range c.MCP.Servers {
		if server.Enabled {
			enabled = append(enabled, server)
		}
	}
	return enabled
}
