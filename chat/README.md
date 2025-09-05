# MCP-OpenAI Integration

集成OpenAI API和MCP工具的智能聊天助手，支持通过自然语言调用SSH命令和文件操作。

## 🌟 特性

- **🤖 智能对话**：基于OpenAI GPT模型的自然语言处理
- **🔧 MCP工具集成**：支持多个MCP服务器和工具调用
- **🖥️ SSH远程执行**：通过MCP工具执行远程SSH命令
- **📁 文件操作**：支持SSH文件传输功能
- **💬 多轮对话**：保持上下文记忆的智能对话
- **⚙️ 灵活配置**：支持多种MCP服务器配置
- **🎯 多种模式**：交互模式和单次消息模式

## 🏗️ 架构设计

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   用户输入      │───▶│  OpenAI API      │───▶│   AI响应        │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  聊天引擎       │◄──▶│  工具调用管理    │◄──▶│  MCP客户端      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │                       │
                                ▼                       ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │  工具执行结果    │◄──▶│  MCP服务器      │
                       └──────────────────┘    └─────────────────┘
                                                        │
                                                        ▼
                                               ┌─────────────────┐
                                               │  SSH执行        │
                                               └─────────────────┘
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- OpenAI API密钥
- SSH访问权限（用于远程执行）

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd AI-Learning/chat
```

2. **设置环境变量**
```bash
export OPENAI_API_KEY="your_openai_api_key_here"
```

3. **构建项目**
```bash
make all
```

4. **构建MCP服务器**（可选）
```bash
make build-mcp-servers
```

### 运行应用

#### 交互模式
```bash
make run
# 或者
./build/mcp-openai-chat -config config.yaml -interactive
```

#### 单次消息模式
```bash
make run-message
# 或者
./build/mcp-openai-chat -message "帮我检查服务器状态"
```

#### 完整环境运行
```bash
make run-full  # 包含MCP服务器构建
```

## 📖 使用指南

### 交互模式命令

在交互模式下，可以使用以下命令：

- `/help, /h` - 显示帮助信息
- `/tools, /t` - 显示可用工具
- `/clear, /c` - 清空对话历史
- `/history, /hist` - 显示消息历史
- `/save, /s` - 保存当前对话
- `/quit, /q` - 退出程序

### 示例对话

```
你: 帮我检查服务器192.168.1.100的磁盘使用情况

AI: 我来帮你检查服务器192.168.1.100的磁盘使用情况。

[自动调用ssh-jsonrpc.ssh_execute工具]
参数: {"host": "192.168.1.100", "command": "df -h"}

执行结果:
主机: 192.168.1.100
命令: df -h
退出码: 0
执行时长: 1.2s

标准输出:
Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        20G  8.5G   11G  45% /
/dev/sda2       100G   45G   50G  48% /home
tmpfs           2.0G     0  2.0G   0% /dev/shm

根据检查结果，服务器192.168.1.100的磁盘使用情况如下：
- 根分区（/）：使用了45%，还有11G可用空间
- 家目录分区（/home）：使用了48%，还有50G可用空间
- 磁盘使用率正常，暂无存储空间不足的风险
```

## ⚙️ 配置说明

### 配置文件结构

```yaml
openai:
  api_key: ""  # OpenAI API密钥（推荐使用环境变量）
  base_url: "https://api.openai.com/v1"  # API基础URL
  model: "gpt-4"  # 使用的模型
  temperature: 0.7  # 温度参数
  max_tokens: 2000  # 最大令牌数
  timeout: 30s  # 请求超时时间

mcp:
  timeout: 30s  # MCP请求超时时间
  servers:  # MCP服务器列表
    - name: "ssh-jsonrpc"
      command: "../go_jsonrpc/build/ssh-mcp-server"
      args: ["-config", "../go_jsonrpc/config.yaml"]
      description: "基于JSON-RPC实现的SSH MCP服务器"
      enabled: true
    - name: "ssh-sdk"
      command: "../go-sdk/build/ssh-mcp-server-sdk"
      args: ["-config", "../go-sdk/config.yaml"]
      description: "基于官方SDK实现的SSH MCP服务器"
      enabled: false

chat:
  max_history: 20  # 最大历史记录数
  system_prompt: "你是一个智能运维助手..."  # 系统提示词
  auto_save: true  # 是否自动保存对话
  save_path: "./conversations"  # 对话保存路径
  enable_mcp: true  # 是否启用MCP工具
  mcp_auto_call: true  # 是否自动调用MCP工具

log:
  level: "info"  # 日志级别
  file: "/tmp/mcp-openai-chat.log"  # 日志文件路径
  max_size: 100  # 日志文件最大大小（MB）
  max_backups: 3  # 保留的日志文件数量
  max_age: 28  # 日志文件保留天数
  compress: true  # 是否压缩旧日志文件
```

### MCP服务器配置

项目支持两种MCP服务器实现：

1. **JSON-RPC实现**（`ssh-jsonrpc`）
   - 基于原生Go和JSON-RPC 2.0协议
   - 位置：`../go_jsonrpc/`
   - 默认启用

2. **官方SDK实现**（`ssh-sdk`）
   - 基于官方MCP Go SDK
   - 位置：`../go-sdk/`
   - 默认禁用

可以通过修改配置文件中的`enabled`字段来切换使用的服务器。

## 🔧 开发指南

### 项目结构

```
chat/
├── main.go                 # 主程序入口
├── config.yaml            # 配置文件
├── Makefile               # 构建脚本
├── go.mod                 # Go模块定义
├── pkg/
│   ├── config/           # 配置管理
│   │   └── config.go
│   ├── mcp/              # MCP客户端
│   │   └── client.go
│   └── chat/             # 聊天引擎
│       └── engine.go
└── README.md             # 项目文档
```

### 构建命令

```bash
# 完整构建
make all

# 仅安装依赖
make deps

# 仅构建应用
make build

# 交叉编译
make build-cross

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查
make vet

# 清理构建文件
make clean

# 检查环境
make check-env

# 创建发布包
make release
```

### 添加新的MCP工具

1. 在MCP服务器中定义新工具
2. 更新配置文件中的服务器设置
3. 重新构建并启动应用

### 自定义系统提示词

修改配置文件中的`chat.system_prompt`字段来自定义AI助手的行为：

```yaml
chat:
  system_prompt: "你是一个专业的运维工程师，擅长Linux系统管理和故障排查..."
```

## 🛠️ 可用工具

当前支持的MCP工具：

### SSH执行工具
- **工具名称**：`ssh_execute`
- **功能**：在远程服务器上执行Shell命令
- **参数**：
  - `host`（必需）：目标服务器地址
  - `command`（必需）：要执行的命令
  - `user`（可选）：SSH用户名，默认为root
  - `port`（可选）：SSH端口，默认为22
  - `timeout`（可选）：超时时间（秒），默认为30

### SSH文件传输工具
- **工具名称**：`ssh_file_transfer`
- **功能**：SSH文件传输（上传/下载）
- **参数**：
  - `host`（必需）：目标服务器地址
  - `localPath`（必需）：本地文件路径
  - `remotePath`（必需）：远程文件路径
  - `direction`（必需）：传输方向（upload/download）
  - `user`（可选）：SSH用户名
  - `port`（可选）：SSH端口

## 🔒 安全注意事项

1. **API密钥安全**：
   - 使用环境变量存储OpenAI API密钥
   - 不要在配置文件中明文存储密钥

2. **SSH安全**：
   - 使用SSH密钥认证而非密码
   - 限制SSH访问的主机范围
   - 定期轮换SSH密钥

3. **命令执行安全**：
   - 谨慎执行具有破坏性的命令
   - 在生产环境中限制可执行的命令范围
   - 启用命令审计和日志记录

## 🐛 故障排查

### 常见问题

1. **OpenAI API调用失败**
   ```
   错误: OpenAI API调用失败: 401 Unauthorized
   解决: 检查OPENAI_API_KEY环境变量是否正确设置
   ```

2. **MCP服务器连接失败**
   ```
   错误: 连接服务器 ssh-jsonrpc 失败
   解决: 确保MCP服务器已构建并且路径正确
   ```

3. **SSH连接失败**
   ```
   错误: SSH连接失败: permission denied
   解决: 检查SSH密钥配置和用户权限
   ```

### 调试模式

启用详细日志：

```bash
# 修改配置文件
log:
  level: "debug"

# 或使用环境变量
export LOG_LEVEL=debug
```

### 日志查看

```bash
# 查看应用日志
tail -f /tmp/mcp-openai-chat.log

# 查看MCP服务器日志
tail -f /tmp/ssh-mcp-server.log
```

## 📝 更新日志

### v1.0.0
- ✅ 初始版本发布
- ✅ 集成OpenAI API和MCP工具
- ✅ 支持SSH命令执行和文件传输
- ✅ 交互模式和单次消息模式
- ✅ 完整的配置管理和错误处理

## 🤝 贡献指南

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [OpenAI](https://openai.com/) - 提供强大的GPT模型
- [Anthropic](https://www.anthropic.com/) - MCP协议的创建者
- [MCP官方文档](https://modelcontextprotocol.io/) - 协议规范和指导

## 📞 联系方式

如有问题或建议，请通过以下方式联系：

- 提交 Issue
- 发送 Pull Request
- 邮件联系：[your-email@example.com]

---

**享受与AI助手的智能对话体验！** 🚀 