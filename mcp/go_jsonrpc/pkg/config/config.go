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
	Server ServerConfig `yaml:"server"` // 服务器配置
	SSH    SSHConfig    `yaml:"ssh"`    // SSH配置
	Log    LogConfig    `yaml:"log"`    // 日志配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Name            string        `yaml:"name"`             // 服务器名称
	Version         string        `yaml:"version"`          // 服务器版本
	ProtocolVersion string        `yaml:"protocol_version"` // MCP协议版本
	Port            int           `yaml:"port"`             // HTTP服务器端口（用于SSE传输）
	Timeout         time.Duration `yaml:"timeout"`          // 请求超时时间
}

// SSHConfig SSH连接配置
type SSHConfig struct {
	DefaultUser    string        `yaml:"default_user"`     // 默认SSH用户名
	DefaultPort    int           `yaml:"default_port"`     // 默认SSH端口
	Timeout        time.Duration `yaml:"timeout"`          // SSH连接超时时间
	KeyFile        string        `yaml:"key_file"`         // SSH私钥文件路径
	KnownHostsFile string        `yaml:"known_hosts_file"` // known_hosts文件路径
	MaxConnections int           `yaml:"max_connections"`  // 最大并发连接数
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
		Server: ServerConfig{
			Name:            "SSH-MCP-Server",
			Version:         "1.0.0",
			ProtocolVersion: "2025-03-26",
			Port:            8000,
			Timeout:         30 * time.Second,
		},
		SSH: SSHConfig{
			DefaultUser:    "root",
			DefaultPort:    22,
			Timeout:        30 * time.Second,
			KeyFile:        "~/.ssh/id_rsa",
			KnownHostsFile: "~/.ssh/known_hosts",
			MaxConnections: 10,
		},
		Log: LogConfig{
			Level:      "info",
			File:       "/var/log/ssh-mcp-server.log",
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
		return DefaultConfig(), nil
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

	// 扩展SSH密钥文件路径
	if c.SSH.KeyFile != "" {
		c.SSH.KeyFile, err = expandPath(c.SSH.KeyFile)
		if err != nil {
			return fmt.Errorf("扩展SSH密钥文件路径失败: %w", err)
		}
	}

	// 扩展known_hosts文件路径
	if c.SSH.KnownHostsFile != "" {
		c.SSH.KnownHostsFile, err = expandPath(c.SSH.KnownHostsFile)
		if err != nil {
			return fmt.Errorf("扩展known_hosts文件路径失败: %w", err)
		}
	}

	// 扩展日志文件路径
	if c.Log.File != "" {
		c.Log.File, err = expandPath(c.Log.File)
		if err != nil {
			return fmt.Errorf("扩展日志文件路径失败: %w", err)
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
	// 验证服务器配置
	if c.Server.Name == "" {
		return fmt.Errorf("服务器名称不能为空")
	}
	if c.Server.Version == "" {
		return fmt.Errorf("服务器版本不能为空")
	}
	if c.Server.ProtocolVersion == "" {
		return fmt.Errorf("协议版本不能为空")
	}

	// 验证SSH配置
	if c.SSH.DefaultPort <= 0 || c.SSH.DefaultPort > 65535 {
		return fmt.Errorf("SSH端口必须在1-65535范围内")
	}
	if c.SSH.MaxConnections <= 0 {
		return fmt.Errorf("最大连接数必须大于0")
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
 