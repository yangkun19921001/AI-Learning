# MCP SSH 远程执行服务器项目集合

> 基于Go语言实现的MCP（Model Context Protocol）SSH远程执行服务器和客户端项目集合，包含手动JSON-RPC实现和官方SDK实现两个版本，以及完整的HTTP SSE传输支持。

## 📁 项目结构

```
mcp/
├── go_jsonrpc/           # 手动JSON-RPC 2.0实现版本
│   ├── cmd/
│   │   ├── server/       # Stdio传输服务器
│   │   ├── client/       # Stdio传输客户端
│   │   ├── sse-server/   # HTTP SSE传输服务器
│   │   └── sse-client/   # HTTP SSE传输客户端
│   ├── pkg/              # 核心包
├── go-sdk/               # 官方SDK实现版本
│   ├── cmd/
│   │   ├── server/       # SDK服务器
│   │   └── client/       # SDK客户端
│   ├── pkg/              # 核心包
└── README.md             # 本文档
```

## 🎯 项目特色

### 🔧 两种实现方案

**1. 手动JSON-RPC实现 (`go_jsonrpc/`)**
- ✅ 完全自主实现JSON-RPC 2.0协议
- ✅ 深度理解MCP协议细节
- ✅ 支持Stdio和HTTP SSE两种传输方式
- ✅ 完整的错误处理和会话管理
- ✅ 适合学习和定制化开发

**2. 官方SDK实现 (`go-sdk/`)**
- ✅ 使用官方MCP Go SDK v0.3.1
- ✅ 类型安全的工具调用
- ✅ 自动JSON Schema生成
- ✅ 结构化输出支持
- ✅ 适合生产环境使用

### 🌐 传输方式支持

**Stdio传输**（两个项目都支持）
- 标准输入输出通信
- 进程间通信
- 适合本地工具调用

**HTTP SSE传输**（go_jsonrpc专有）
- Server-Sent Events实时通信
- 支持网络分布式部署
- 完整的会话管理
- 符合HTTP标准，防火墙友好

## 🚀 快速开始

### 环境要求

- Go 1.21 或更高版本
- SSH客户端工具
- 目标服务器的SSH访问权限

### 选择实现方案

**如果您是新手或想深入理解MCP协议：**
- 推荐使用 `go_jsonrpc/` 项目
- 包含完整的协议实现细节
- 支持HTTP SSE网络传输

**如果您要在生产环境使用：**
- 推荐使用 `go-sdk/` 项目  
- 基于官方SDK，更加稳定
- 类型安全，开发效率高

## 📖 详细使用指南

### 1. go_jsonrpc 项目（手动实现）

#### 特性概览
- **4个可执行程序**：server、client、sse-server、sse-client
- **2种传输方式**：Stdio、HTTP SSE
- **完整协议实现**：JSON-RPC 2.0 + MCP规范
- **网络支持**：支持分布式部署

#### 安装和构建

```bash
cd go_jsonrpc

# 安装依赖
make deps

# 构建所有程序
make build

# 查看构建结果
ls -la build/
# ssh-mcp-server          # Stdio传输服务器
# ssh-mcp-client          # Stdio传输客户端  
# ssh-mcp-sse-server      # HTTP SSE传输服务器
# ssh-mcp-sse-client      # HTTP SSE传输客户端
```

#### 配置文件

编辑 `config.yaml`：

```yaml
server:
  name: "SSH-MCP-Server"
  version: "1.0.0"
  protocol_version: "2025-03-26"
  port: 8000  # HTTP服务器端口（用于SSE传输）
  timeout: 30s

ssh:
  default_user: "root"
  default_port: 22
  timeout: 30s
  key_file: "~/.ssh/id_rsa"
  known_hosts_file: "~/.ssh/known_hosts"
  max_connections: 10

log:
  level: "info"
  file: "/tmp/ssh-mcp-server.log"
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true
```

#### 使用方式

**方式1：Stdio传输（进程间通信）**

```bash
# 演示模式（推荐新手）
make run-client

# 交互模式
./build/ssh-mcp-client -mode interactive \
  -server ./build/ssh-mcp-server \
  -args "-config config.yaml"

# 直接工具调用
./build/ssh-mcp-client \
  -tool ssh_execute \
  -tool-args '{"host":"localhost","command":"uptime"}' \
  -server ./build/ssh-mcp-server \
  -args "-config config.yaml"
```

**方式2：HTTP SSE传输（网络通信）**

```bash
# 启动SSE服务器（在一个终端）
make run-sse-server
# 或者
./build/ssh-mcp-sse-server -config config.yaml

# 连接SSE客户端（在另一个终端）
make run-sse-client
# 或者
./build/ssh-mcp-sse-client -server http://localhost:8000 -mode demo

# 直接工具调用
./build/ssh-mcp-sse-client \
  -server http://localhost:8000 \
  -mode call \
  -tool ssh_execute \
  -args '{"host":"localhost","command":"df -h"}'
```

#### HTTP SSE API端点

当运行SSE服务器时，提供以下HTTP端点：

- `GET /mcp/sse` - 建立SSE连接，获取会话端点
- `POST /mcp/message?sessionId=<id>` - 发送MCP消息

可以用curl测试：

```bash
# 建立SSE连接
curl -N -H "Accept: text/event-stream" \
  http://localhost:8000/mcp/sse

# 发送初始化请求
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"sampling":{}},"clientInfo":{"name":"test-client","version":"1.0.0"}}}' \
  "http://localhost:8000/mcp/message?sessionId=<从SSE获得的会话ID>"
```

### 2. go-sdk 项目（官方SDK）

#### 特性概览
- **2个可执行程序**：server、client
- **官方SDK**：基于github.com/modelcontextprotocol/go-sdk
- **类型安全**：Go泛型 + 自动Schema生成
- **结构化输出**：便于AI模型理解

#### 安装和构建

```bash
cd go-sdk

# 安装依赖
make deps

# 构建所有程序
make build

# 查看构建结果
ls -la build/
# ssh-mcp-server-sdk      # MCP服务器
# ssh-mcp-client-sdk      # MCP客户端
```

#### 配置文件

编辑 `config.yaml`：

```yaml
server:
  name: "SSH-MCP-Server-SDK"
  version: "1.0.0"

ssh:
  default_user: "root"
  default_port: 22
  timeout: 30s
  key_file: "~/.ssh/id_rsa"
  known_hosts_file: "~/.ssh/known_hosts"
  max_connections: 10

log:
  level: "info"
  file: "/tmp/ssh-mcp-server-sdk.log"
```

#### 使用方式

```bash
# 演示模式（推荐）
make run-client

# 交互模式
./build/ssh-mcp-client-sdk -mode interactive \
  -server ./build/ssh-mcp-server-sdk \
  -args "-config config.yaml"

# 直接工具调用
./build/ssh-mcp-client-sdk \
  -tool ssh_execute \
  -tool-args '{"host":"localhost","command":"ps aux"}' \
  -server ./build/ssh-mcp-server-sdk \
  -args "-config config.yaml"
```

## 🛠️ 可用工具

两个项目都提供相同的SSH工具：

### 1. ssh_execute - SSH命令执行

**功能**：在远程服务器上执行Shell命令

**参数**：
- `host` (必需): 目标服务器地址
- `command` (必需): 要执行的命令  
- `user` (可选): SSH用户名
- `port` (可选): SSH端口，默认22
- `timeout` (可选): 超时时间（秒）

**示例**：
```json
{
  "host": "192.168.1.100",
  "command": "ps aux | grep nginx",
  "user": "admin",
  "port": 22,
  "timeout": 30
}
```

### 2. ssh_file_transfer - SSH文件传输

**功能**：在本地和远程服务器之间传输文件

**参数**：
- `host` (必需): 目标服务器地址
- `localPath` (必需): 本地文件路径
- `remotePath` (必需): 远程文件路径
- `direction` (必需): 传输方向，"upload" 或 "download"
- `user` (可选): SSH用户名
- `port` (可选): SSH端口

**示例**：
```json
{
  "host": "192.168.1.100", 
  "localPath": "/tmp/local-file.txt",
  "remotePath": "/tmp/remote-file.txt",
  "direction": "upload"
}
```

## 🔄 运行模式

两个项目都支持多种运行模式：

### 1. 演示模式 (demo)
- 自动展示所有功能
- 执行预定义的示例命令
- 适合快速了解功能

### 2. 交互模式 (interactive)  
- 提供交互式命令行界面
- 可以手动输入和执行命令
- 支持工具列表查看

### 3. 直接调用模式 (call)
- 直接调用指定工具
- 适合脚本化和自动化
- 支持JSON参数传递

## 📊 性能对比

| 特性 | go_jsonrpc | go-sdk |
|------|------------|--------|
| **实现方式** | 手动JSON-RPC | 官方SDK |
| **传输方式** | Stdio + HTTP SSE | Stdio |
| **类型安全** | 运行时检查 | 编译时检查 |
| **Schema生成** | 手动定义 | 自动生成 |
| **网络支持** | ✅ 支持 | ❌ 不支持 |
| **学习价值** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| **生产就绪** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ |
| **定制能力** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

## 🧪 测试和验证

### 功能测试

**测试go_jsonrpc项目**：
```bash
cd go_jsonrpc

# 运行所有测试
make test

# 测试Stdio传输
make run-client

# 测试HTTP SSE传输
make run-sse-server &  # 后台运行服务器
make run-sse-client    # 运行客户端
```

**测试go-sdk项目**：
```bash
cd go-sdk

# 运行所有测试
make test

# 测试基本功能
make run-client
```

### 网络测试

**测试HTTP SSE端点**：
```bash
# 启动SSE服务器
cd go_jsonrpc
./build/ssh-mcp-sse-server -config config.yaml &

# 测试SSE连接
curl -N -H "Accept: text/event-stream" \
  http://localhost:8000/mcp/sse

# 测试工具调用（需要先建立SSE连接获取sessionId）
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}' \
  "http://localhost:8000/mcp/message?sessionId=<会话ID>"
```

## 🔧 开发和扩展

### 添加新工具

**在go_jsonrpc项目中**：
1. 编辑 `pkg/server/server.go` 的 `handleToolsList` 方法
2. 在 `handleToolsCall` 方法中添加工具路由
3. 实现具体的工具处理方法

**在go-sdk项目中**：
1. 定义工具参数和结果结构体（带jsonschema标签）
2. 在 `registerTools` 方法中注册工具
3. 实现工具处理函数

### 自定义传输方式

**go_jsonrpc项目支持**：
- 修改 `pkg/server/sse_server.go` 自定义HTTP端点
- 修改 `pkg/client/sse_client.go` 自定义客户端行为

## 🚨 故障排除

### 常见问题

1. **SSH连接失败**
```bash
# 检查SSH密钥
ssh-add -l

# 测试SSH连接
ssh -i ~/.ssh/id_rsa user@host

# 检查配置文件中的SSH设置
```

2. **HTTP SSE连接失败**
```bash
# 检查端口是否占用
lsof -i :8000

# 检查防火墙设置
curl -I http://localhost:8000/mcp/sse
```

3. **工具调用失败**
```bash
# 启用调试日志
# 在config.yaml中设置:
log:
  level: "debug"
```

### 调试技巧

**查看详细日志**：
```bash
# go_jsonrpc
./build/ssh-mcp-server -config config.yaml 2>&1 | tee server.log

# go-sdk  
./build/ssh-mcp-server-sdk -config config.yaml 2>&1 | tee server-sdk.log
```

**抓包分析**（仅HTTP SSE）：
```bash
# 使用tcpdump抓包
sudo tcpdump -i lo0 -w mcp-traffic.pcap port 8000

# 使用wireshark分析
wireshark mcp-traffic.pcap
```

## 📚 参考资料

- [MCP协议规范](https://modelcontextprotocol.io/specification/2025-03-26/basic)
- [JSON-RPC 2.0规范](https://www.jsonrpc.org/specification)
- [官方MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [Go SSH包文档](https://pkg.go.dev/golang.org/x/crypto/ssh)

## 🤝 贡献

1. Fork 项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 打开 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。

---

## 💡 使用建议

**学习MCP协议**：建议从 `go_jsonrpc` 项目开始，可以深入理解协议细节

**生产环境**：推荐使用 `go-sdk` 项目，更加稳定可靠

**网络部署**：使用 `go_jsonrpc` 的HTTP SSE传输功能

**工具开发**：两个项目都提供了良好的扩展示例 