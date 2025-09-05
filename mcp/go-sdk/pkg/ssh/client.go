package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client SSH客户端管理器
// 负责管理SSH连接池，提供连接复用和生命周期管理
type Client struct {
	config      *Config                // SSH配置
	connections map[string]*ssh.Client // 连接池：host -> ssh.Client
	mutex       sync.RWMutex           // 读写锁保护连接池
	ctx         context.Context        // 上下文
	cancel      context.CancelFunc     // 取消函数
}

// Config SSH客户端配置
type Config struct {
	DefaultUser    string        // 默认用户名
	DefaultPort    int           // 默认端口
	Timeout        time.Duration // 连接超时时间
	KeyFile        string        // 私钥文件路径
	KnownHostsFile string        // known_hosts文件路径
	MaxConnections int           // 最大连接数
}

// ConnectionInfo SSH连接信息
type ConnectionInfo struct {
	Host     string // 主机地址
	Port     int    // 端口
	User     string // 用户名
	Password string // 密码（可选）
	KeyFile  string // 私钥文件（可选）
}

// ExecuteResult 命令执行结果
type ExecuteResult struct {
	Command  string        // 执行的命令
	ExitCode int           // 退出码
	Stdout   string        // 标准输出
	Stderr   string        // 标准错误
	Duration time.Duration // 执行时长
}

// NewClient 创建新的SSH客户端管理器
func NewClient(config *Config) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	return &Client{
		config:      config,
		connections: make(map[string]*ssh.Client),
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Connect 建立SSH连接
// 如果连接已存在且有效，则复用现有连接
func (c *Client) Connect(info *ConnectionInfo) (*ssh.Client, error) {
	// 填充默认值
	if info.Port == 0 {
		info.Port = c.config.DefaultPort
	}
	if info.User == "" {
		info.User = c.config.DefaultUser
	}

	// 生成连接键
	connKey := fmt.Sprintf("%s@%s:%d", info.User, info.Host, info.Port)

	// 检查现有连接
	c.mutex.RLock()
	if client, exists := c.connections[connKey]; exists {
		// 测试连接是否仍然有效
		if c.isConnectionAlive(client) {
			c.mutex.RUnlock()
			return client, nil
		}
		// 连接已失效，需要清理
		delete(c.connections, connKey)
	}
	c.mutex.RUnlock()

	// 创建新连接
	client, err := c.createConnection(info)
	if err != nil {
		return nil, fmt.Errorf("创建SSH连接失败: %w", err)
	}

	// 存储连接
	c.mutex.Lock()
	// 检查连接数限制
	if len(c.connections) >= c.config.MaxConnections {
		c.mutex.Unlock()
		client.Close()
		return nil, fmt.Errorf("已达到最大连接数限制: %d", c.config.MaxConnections)
	}
	c.connections[connKey] = client
	c.mutex.Unlock()

	return client, nil
}

// Execute 执行SSH命令
func (c *Client) Execute(info *ConnectionInfo, command string) (*ExecuteResult, error) {
	startTime := time.Now()

	// 获取SSH连接
	client, err := c.Connect(info)
	if err != nil {
		return nil, fmt.Errorf("连接SSH服务器失败: %w", err)
	}

	// 创建会话
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()

	// 设置超时
	ctx, cancel := context.WithTimeout(c.ctx, c.config.Timeout)
	defer cancel()

	// 创建管道获取输出
	stdout, err := session.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stdout管道失败: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("创建stderr管道失败: %w", err)
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		return nil, fmt.Errorf("启动命令失败: %w", err)
	}

	// 读取输出
	var stdoutBuf, stderrBuf []byte
	done := make(chan error, 1)

	go func() {
		var wg sync.WaitGroup
		wg.Add(2)

		// 读取stdout
		go func() {
			defer wg.Done()
			stdoutBuf, _ = io.ReadAll(stdout)
		}()

		// 读取stderr
		go func() {
			defer wg.Done()
			stderrBuf, _ = io.ReadAll(stderr)
		}()

		wg.Wait()
		done <- session.Wait()
	}()

	// 等待命令完成或超时
	var exitCode int
	select {
	case err := <-done:
		if err != nil {
			if exitError, ok := err.(*ssh.ExitError); ok {
				exitCode = exitError.ExitStatus()
			} else {
				return nil, fmt.Errorf("命令执行失败: %w", err)
			}
		}
	case <-ctx.Done():
		session.Signal(ssh.SIGTERM)
		return nil, fmt.Errorf("命令执行超时")
	}

	duration := time.Since(startTime)

	return &ExecuteResult{
		Command:  command,
		ExitCode: exitCode,
		Stdout:   string(stdoutBuf),
		Stderr:   string(stderrBuf),
		Duration: duration,
	}, nil
}

// Close 关闭所有SSH连接
func (c *Client) Close() error {
	c.cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	var errors []error
	for connKey, client := range c.connections {
		if err := client.Close(); err != nil {
			errors = append(errors, fmt.Errorf("关闭连接 %s 失败: %w", connKey, err))
		}
	}

	// 清空连接池
	c.connections = make(map[string]*ssh.Client)

	if len(errors) > 0 {
		return fmt.Errorf("关闭连接时发生错误: %v", errors)
	}

	return nil
}

// GetConnectionCount 获取当前连接数
func (c *Client) GetConnectionCount() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return len(c.connections)
}

// createConnection 创建新的SSH连接
func (c *Client) createConnection(info *ConnectionInfo) (*ssh.Client, error) {
	// 准备SSH配置
	sshConfig := &ssh.ClientConfig{
		User:    info.User,
		Timeout: c.config.Timeout,
	}

	// 设置主机密钥验证
	if c.config.KnownHostsFile != "" {
		hostKeyCallback, err := knownhosts.New(c.config.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("加载known_hosts文件失败: %w", err)
		}
		sshConfig.HostKeyCallback = hostKeyCallback
	} else {
		// 警告：跳过主机密钥验证（仅用于开发环境）
		sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	// 设置认证方法
	auth, err := c.getAuthMethods(info)
	if err != nil {
		return nil, fmt.Errorf("获取认证方法失败: %w", err)
	}
	sshConfig.Auth = auth

	// 建立连接
	address := fmt.Sprintf("%s:%d", info.Host, info.Port)
	client, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("连接SSH服务器失败: %w", err)
	}

	return client, nil
}

// getAuthMethods 获取SSH认证方法
func (c *Client) getAuthMethods(info *ConnectionInfo) ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	// 如果提供了密码，优先使用密码认证
	if info.Password != "" {
		authMethods = append(authMethods, ssh.Password(info.Password))
	}

	// 然后尝试指定的私钥文件
	keyFile := info.KeyFile
	if keyFile == "" {
		keyFile = c.config.KeyFile
	}

	if keyFile != "" {
		// 扩展路径
		if keyFile[0] == '~' {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("获取用户主目录失败: %w", err)
			}
			keyFile = filepath.Join(homeDir, keyFile[2:])
		}

		// 读取私钥文件
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("读取私钥文件失败: %w", err)
		}

		// 解析私钥
		signer, err := ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// 最后添加SSH Agent认证（如果可用）
	if agentConn, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		authMethods = append(authMethods, ssh.PublicKeysCallback(
			func() ([]ssh.Signer, error) {
				agent := NewSSHAgent(agentConn)
				return agent.Signers()
			},
		))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("没有可用的认证方法")
	}

	return authMethods, nil
}

// isConnectionAlive 检查SSH连接是否仍然有效
func (c *Client) isConnectionAlive(client *ssh.Client) bool {
	// 尝试创建一个会话来测试连接
	session, err := client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

// SSHAgent SSH Agent接口实现
type SSHAgent struct {
	conn net.Conn
}

// NewSSHAgent 创建SSH Agent
func NewSSHAgent(conn net.Conn) *SSHAgent {
	return &SSHAgent{conn: conn}
}

// Signers 获取SSH Agent中的签名器
func (a *SSHAgent) Signers() ([]ssh.Signer, error) {
	// 这里应该实现SSH Agent协议
	// 为简化示例，返回空列表
	return []ssh.Signer{}, nil
}
