# 企业级 Agent 开发实战(二) MCP 原理深度解析及项目实战

> Model Context Protocol (MCP) 作为连接 AI 模型与外部世界的桥梁，正在重新定义 AI 应用的可能性边界。从协议设计的精妙之处到企业级实战应用，让我们深入探索这个改变 AI 生态的重要协议。



## 📋 目录

- [前言](#前言)
- [MCP 介绍](#mcp-介绍)
  - [什么是 MCP](#什么是-mcp)
  - [为什么需要 MCP](#为什么需要-mcp)
  - [MCP 的核心价值](#mcp-的核心价值)
- [MCP 协议原理](#mcp-协议原理)
  - [JSON-RPC 2.0 基础与深度解析](#json-rpc-20-基础与深度解析)
  - [MCP 特有的消息扩展](#mcp-特有的消息扩展)
  - [协议生命周期](#协议生命周期)
- [MCP 架构](#mcp-架构)
- [抓包分析](#抓包分析)
- [实战开发](#实战开发)
  - [第一部分：手动实现](#第一部分手动实现---深入理解-mcp-协议)
  - [第二部分：官方SDK实现](#第二部分官方sdk实现---生产级开发)
  - [Go + OpenAI SDK 接入 MCP Client](#go--openai-sdk-接入-mcp-client)
- [总结](#总结)
- [参考](#参考)

## 前言

在 AI 快速发展的今天，我们看到了一个有趣的现象：AI 模型变得越来越强大，但它们与现实世界的连接却依然脆弱。传统的工具调用方式虽然能解决部分问题，但在面对复杂的企业级应用时，往往显得力不从心。

MCP（Model Context Protocol）的出现，为这个问题提供了一个优雅的解决方案。它不仅仅是另一个协议规范，更是一种全新的思维方式 - 如何让 AI 智能体以标准化、安全化、可扩展的方式与外部系统进行深度集成。

## MCP 介绍

### 什么是 MCP

**Model Context Protocol（模型上下文协议，简称 MCP）** 是由 AI 公司 Anthropic 在 [2024 年 11 月 25 日](https://www.anthropic.com/news/model-context-protocol) 推出的一个**开源标准化协议**。它的核心目的是解决 AI 模型（如 Claude、ChatGPT、DeepSeek 等）与外部数据源、工具交互时的**碎片化和复杂性问题**。

简单来说，MCP 希望为 AI 世界提供一个**统一、通用的“接线”标准**。正如 USB-C 接口可以用同一套线缆连接电脑、手机、显示器等各种设备，MCP 致力于让不同的 AI 模型都能通过一套标准协议，安全、高效地连接和调用各种各样的外部资源（如数据库、API、文件系统等）



### 为什么需要 MCP

在 MCP 出现之前，每个 AI 应用都需要为不同的数据源和工具编写专门的集成代码。这导致了几个问题：

1. **重复造轮子**：每个开发者都要为相同的服务编写集成代码
2. **维护成本高**：每个集成都需要单独维护和更新
3. **扩展性差**：添加新的数据源需要修改核心代码
4. **标准化缺失**：不同的集成方式导致开发体验不一致

MCP 通过提供标准化的协议解决了这些问题，让开发者可以：

- 一次编写，到处使用
- 插件化架构，易于扩展
- 统一的开发体验
- 社区共享的生态系统

### MCP 的核心价值

MCP 的价值不仅仅在于技术层面的标准化，更在于它为 AI 应用开发带来的范式转变：

**从孤立到连接**：AI 模型不再是孤立的智能体，而是可以与整个数字生态系统深度集成的智能助手。

**从静态到动态**：通过实时数据访问和工具调用，AI 应用可以处理动态变化的业务场景。

**从单一到协作**：多个 MCP 服务器可以协同工作，为 AI 提供丰富的上下文信息。

## MCP 协议原理

### JSON-RPC 2.0 基础与深度解析

MCP 建立在 [JSON-RPC 2.0 协议](https://wiki.geekdream.com/Specification/json-rpc_2.0.html) 之上，这是一个无状态且轻量级的远程过程调用（RPC）协议。JSON-RPC 2.0 于 2010 年发布，专为简单性而设计，使用 JSON 作为数据格式。

#### 为什么选择 JSON-RPC 而不是 REST

选择 JSON-RPC 而不是 REST API 背后有深层的技术考量：

**1. 状态管理的需要**：虽然 JSON-RPC 本身是无状态的，但它支持持久连接，这使得 AI 对话可以在连接层面维护会话状态。

**2. 双向通信支持**：JSON-RPC 天然支持双向通信，服务器可以主动向客户端推送通知，这对于 AI Agent 的实时更新至关重要。

**3. 结构化消息格式**：JSON-RPC 的标准化消息结构提供了更好的类型安全性和错误处理机制。

**4. 批量操作支持**：JSON-RPC 2.0 原生支持批量调用，这对于 AI 工具的批量执行非常有用。

#### JSON-RPC 2.0 核心概念

根据 JSON-RPC 2.0 规范，协议定义了以下核心概念：

**数据类型系统**：

- 基本类型：String（字符串）、Number（数值）、Boolean（布尔）、Null（空值）
- 结构化类型：Object（对象）、Array（数组）

**角色定义**：

- **客户端**：请求对象的来源及响应对象的处理程序
- **服务端**：响应对象的起源和请求对象的处理程序

让我们看一个符合 JSON-RPC 2.0 规范的 MCP 消息示例：

```json
// 请求对象（Request Object）
{
  "jsonrpc": "2.0",          // 必须准确写为"2.0"
  "id": "req-123",           // 客户端唯一标识ID
  "method": "tools/call",    // 要调用的方法名称
  "params": {                // 结构化参数值（可省略）
    "name": "get_weather",
    "arguments": {
      "location": "北京",
      "units": "celsius"
    }
  }
}

// 成功响应对象（Response Object）
{
  "jsonrpc": "2.0",
  "id": "req-123",           // 必须与请求对象的ID一致
  "result": {                // 成功时包含此成员
    "content": [
      {
        "type": "text",
        "text": "北京当前温度：15°C，晴朗"
      }
    ]
  }
}

// 错误响应对象
{
  "jsonrpc": "2.0",
  "id": "req-123",
  "error": {                 // 失败时包含此成员
    "code": -32602,
    "message": "Invalid params",
    "data": {
      "details": "location参数不能为空"
    }
  }
}
```

#### JSON-RPC 2.0 错误代码规范

JSON-RPC 2.0定义了标准化的错误代码系统，MCP完全遵循这一规范：

| 错误代码         | 错误信息         | 含义                         |
| ---------------- | ---------------- | ---------------------------- |
| -32700           | Parse error      | 服务端接收到无效的JSON       |
| -32600           | Invalid Request  | 发送的JSON不是有效的请求对象 |
| -32601           | Method not found | 该方法不存在或无效           |
| -32602           | Invalid params   | 无效的方法参数               |
| -32603           | Internal error   | JSON-RPC内部错误             |
| -32000 to -32099 | Server error     | 预留用于自定义的服务器错误   |

```json
// MCP 中的错误处理示例
{
  "jsonrpc": "2.0",
  "id": "tools-1",
  "error": {
    "code": -32601,
    "message": "Method not found",
    "data": {
      "method": "ssh_invalid_command",
      "available_methods": ["ssh_execute", "ssh_file_transfer"]
    }
  }
}
```

#### 通知机制（Notifications）

JSON-RPC 2.0的通知机制是MCP实时通信的基础。通知是不包含"id"成员的请求对象，表明客户端对响应不感兴趣：

```json
// MCP工具列表变化通知
{
  "jsonrpc": "2.0",
  "method": "notifications/tools/list_changed"
}

// MCP日志消息通知
{
  "jsonrpc": "2.0", 
  "method": "notifications/message",
  "params": {
    "level": "info",
    "logger": "ssh-server",
    "data": "SSH连接已建立"
  }
}
```

#### 批量调用支持

JSON-RPC 2.0原生支持批量调用，这让MCP可以高效处理多个工具调用：

```json
// 批量工具调用请求
[
  {
    "jsonrpc": "2.0",
    "id": "1",
    "method": "tools/call",
    "params": {"name": "check_disk_space", "arguments": {"path": "/"}}
  },
  {
    "jsonrpc": "2.0", 
    "id": "2",
    "method": "tools/call",
    "params": {"name": "check_memory", "arguments": {}}
  }
]

// 批量响应
[
  {
    "jsonrpc": "2.0",
    "id": "1", 
    "result": {"content": [{"type": "text", "text": "磁盘使用率: 45%"}]}
  },
  {
    "jsonrpc": "2.0",
    "id": "2",
    "result": {"content": [{"type": "text", "text": "内存使用率: 60%"}]}
  }
]
```

#### 参数结构规范

JSON-RPC 2.0 支持两种参数传递方式，MCP主要使用命名参数：

**1. 位置参数（索引数组）**：

```json
{
  "jsonrpc": "2.0",
  "method": "subtract", 
  "params": [42, 23],
  "id": 1
}
```

**2. 命名参数（关联对象）** - MCP推荐方式：

```json
{
  "jsonrpc": "2.0",
  "method": "tools/call",
  "params": {
    "name": "ssh_execute",
    "arguments": {
      "host": "192.168.1.100",
      "command": "uptime"
    }
  },
  "id": 1
}
```

这种基于 JSON-RPC 2.0 的设计让MCP既保持了协议的简洁性，又获得了强大的功能支持，为AI Agent与外部系统的通信提供了坚实的基础。

### MCP 特有的消息扩展

在JSON-RPC 2.0 基础上，MCP 定义了特定的方法和消息模式来支持AI Agent的需求：

#### 系统方法（以rpc开头的保留方法）

根据JSON-RPC 2.0规范，以"rpc"开头的方法名是预留的。MCP虽然不直接使用这些方法，但遵循了这一命名约定，定义了自己的方法命名空间：

**MCP核心方法命名空间**：

- `initialize` - 初始化连接
- `tools/*` - 工具相关操作
- `resources/*` - 资源相关操作  
- `prompts/*` - 提示相关操作
- `notifications/*` - 通知消息

#### MCP专用消息模式

**1. 初始化握手序列**：

```json
// 第一步：客户端初始化请求
{
  "jsonrpc": "2.0",
  "id": "init-1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "sampling": {}
    },
    "clientInfo": {
      "name": "xxx-Agent",
      "version": "1.0.0"
    }
  }
}

// 第二步：服务器能力响应
{
  "jsonrpc": "2.0", 
  "id": "init-1",
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {"listChanged": true},
      "resources": {"subscribe": true}
    },
    "serverInfo": {
      "name": "SSH-Server",
      "version": "1.0.0"
    }
  }
}

// 第三步：客户端确认通知（无响应）
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

**2. 工具发现和调用模式**：

```json
// 工具列表查询
{
  "jsonrpc": "2.0",
  "id": "tools-list-1", 
  "method": "tools/list"
}

// 工具调用（支持复杂参数结构）
{
  "jsonrpc": "2.0",
  "id": "tool-exec-1",
  "method": "tools/call",
  "params": {
    "name": "ssh_execute",
    "arguments": {
      "host": "192.168.1.100",
      "command": "ps aux | grep nginx",
      "timeout": 30
    }
  }
}
```

**3. 实时通知模式**：

```json
// 能力变化通知
{
  "jsonrpc": "2.0",
  "method": "notifications/tools/list_changed"
}

// 进度更新通知
{
  "jsonrpc": "2.0", 
  "method": "notifications/progress",
  "params": {
    "progressToken": "task-123",
    "value": 0.6,
    "message": "正在上传文件..."
  }
}
```

### 协议生命周期

MCP连接遵循严格的生命周期管理：

**1. 初始化阶段**

```json
// 客户端发送初始化请求
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "sampling": {}
    },
    "clientInfo": {
      "name": "xxx-Agent",
      "version": "1.0.0"
    }
  }
}
```

**2. 能力协商**
服务器响应其支持的功能：

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": { "listChanged": true },
      "resources": { "subscribe": true }
    },
    "serverInfo": {
      "name": "SSH-Server",
      "version": "1.0.0"
    }
  }
}
```

**3. 就绪确认**

```json
{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}
```

## MCP 架构

![1756608438](http://devyk.top/2022/202508311049060.png)

### 整体架构设计

MCP采用经典的客户端-服务器架构，但其设计哲学体现了现代分布式系统的最佳实践：

![image-20250831170534572](http://devyk.top/2022/202508311705261.png)

### 核心组件详解

**MCP Host（主机）**

- 负责整体的AI应用逻辑
- 管理多个MCP客户端连接
- 处理用户交互和AI模型调用
- 聚合来自不同服务器的上下文信息

**MCP Client（客户端）**

- 维护与单个服务器的专用连接
- 处理协议级别的消息路由
- 管理会话状态和生命周期
- 提供统一的API接口给Host

**MCP Server（服务器）**

- 暴露特定的工具、资源或提示
- 处理来自客户端的请求
- 管理与外部系统的连接
- 提供实时通知能力

### 传输层设计

MCP支持两种主要的传输方式：

**1. Stdio传输**
适用于本地进程通信：

```go
// Go语言示例：启动并连接MCP服务器进程
cmd := exec.Command("python", "mcp_server.py")

// 创建输入输出管道
stdinPipe, _ := cmd.StdinPipe()
stdoutPipe, _ := cmd.StdoutPipe()

// 启动进程
cmd.Start()

// 通过stdin发送JSON-RPC消息
message := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`
io.WriteString(stdinPipe, "Content-Length: " + strconv.Itoa(len(message)) + "\r\n\r\n")
io.WriteString(stdinPipe, message)

// 从stdout读取响应
scanner := bufio.NewScanner(stdoutPipe)
// 实际实现需要处理Content-Length头部和消息分帧
```

**2. HTTP流传输**
适用于网络通信：

```go
// HTTP SSE客户端示例
client := &http.Client{}
req, _ := http.NewRequest("POST", "http://localhost:8080/mcp/sse", nil)
req.Header.Set("Accept", "text/event-stream")
req.Header.Set("Authorization", "Bearer your-token-here")

resp, _ := client.Do(req)

// 处理服务器发送事件流
reader := bufio.NewReader(resp.Body)
for {
    line, _ := reader.ReadString('\n')
    if strings.HasPrefix(line, "data: ") {
        jsonData := line[6:] // 提取JSON数据
        // 处理JSON-RPC消息
    }
}
```

### 数据层协议

MCP的数据层定义了三种核心原语：

**工具（Tools）**
可执行的函数，让AI能够执行操作：

```json
{
  "name": "execute_command",
  "description": "在远程服务器上执行Shell命令",
  "inputSchema": {
    "type": "object",
    "properties": {
      "command": {"type": "string"},
      "host": {"type": "string"}
    },
    "required": ["command", "host"]
  }
}
```

**资源（Resources）**
只读的上下文数据：

```json
{
  "uri": "file:///var/log/app.log",
  "name": "应用日志",
  "description": "最新的应用运行日志",
  "mimeType": "text/plain"
}
```

**提示（Prompts）**
可重用的提示模板：

```json
{
  "name": "code_review",
  "description": "代码审查提示模板",
  "arguments": [
    {
      "name": "code",
      "description": "要审查的代码",
      "required": true
    }
  ]
}
```

## 抓包分析

### MCP协议完整流程解析

为了深入理解MCP的工作原理，我们通过在 **cursor 中配置一个远程执行命令的 mcp server** 来实际抓包分析一个完整的MCP会话。按照JSON-RPC 2.0和MCP协议规范，一个完整的MCP会话包含以下标准流程：

![image-20250904165929162](http://devyk.top/2022/202509041659453.png)



#### 环境准备

```bash
# macOS安装
brew install wireshark

# Ubuntu/Debian安装
sudo apt-get install wireshark tshark

# 验证安装
tshark --version
```

#### 通用分析方法

在实际分析中，我们不会预先知道具体的帧号，而是需要按以下方法逐步发现：

**第一步：确定MCP流量范围**

```bash
# 设置变量
PCAP_FILE="remote-exec-mcp-server.pcapng"
MCP_SERVER_IP="10.1.16.4"

# 查看与MCP服务器的所有通信
tshark -r "$PCAP_FILE" -Y "ip.addr == $MCP_SERVER_IP" \
  -T fields -e frame.number -e frame.time_relative -e frame.len | head -10


# 查看HTTP请求概览
tshark -r "$PCAP_FILE" -Y "ip.addr == $MCP_SERVER_IP and http.request.method" \
  -T fields -e frame.number -e http.request.method -e http.request.uri
```

![image-20250830210545856](http://devyk.top/2022/202508302105940.png)

**第二步：动态查找关键帧**

```bash
# 查找所有包含JSON-RPC方法的帧
echo "=== 发现JSON-RPC消息 ==="
tshark -r "$PCAP_FILE" -Y "tcp.payload matches \"method\"" \
  -T fields -e frame.number -e frame.time_relative | \
  while read frame time; do
    method=$(tshark -r "$PCAP_FILE" -Y "frame.number == $frame" \
      -T fields -e tcp.payload | xxd -r -p | grep -o '"method":"[^"]*"' | head -1)
    echo "帧 $frame (时间 ${time}s): $method"
  done
```

![image-20250830210640576](http://devyk.top/2022/202508302106002.png)

**重要说明**：在真实分析中，我们不会预先知道具体的帧号（如12489、12531等）。这些帧号是通过上述动态查找命令自动发现的。下面演示的具体步骤使用的帧号都是从实际抓包中动态提取出来的结果。

### 步骤1：尝试POST连接（失败）

**协议说明**：客户端首先尝试通过POST请求直接建立MCP连接，但被服务器拒绝

**发现POST尝试**：

```bash
# 查找第一个MCP相关请求
FIRST_POST=$(tshark -r remote-exec-mcp-server.pcapng \
  -Y "http.request.method == \"POST\" and http.request.uri == \"/mcp/sse\"" \
  -T fields -e frame.number | head -1)
echo "第一个POST尝试帧号: $FIRST_POST"

# 查看POST请求内容
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == $FIRST_POST" \
  -T fields -e tcp.payload | xxd -r -p | head -12
```

**POST请求内容**：

```http
第一个POST尝试帧号: 12489
POST /mcp/sse HTTP/1.1
host: 10.1.16.4:8000
connection: keep-alive
User-Agent: Cursor/1.2.2 (darwin arm64)
content-type: application/json
accept: application/json, text/event-stream
accept-language: *
sec-fetch-mode: cors
accept-encoding: gzip, deflate
content-length: 254

{
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "tools": true,
      "prompts": false,
      "resources": false,
      "logging": false,
      "roots": {
        "listChanged": false
      }
    },
    "clientInfo": {
      "name": "cursor-vscode",
      "version": "1.0.0"
    }
  },
  "jsonrpc": "2.0",
  "id": 0
}
```

**服务器拒绝响应**：

```bash
# 查找POST请求的响应
POST_RESPONSE=$(tshark -r remote-exec-mcp-server.pcapng \
  -Y "frame.number > $FIRST_POST and frame.number < $((FIRST_POST + 10)) and http.response.code == 405" \
  -T fields -e frame.number | head -1)
echo "POST_RESPONSE:$POST_RESPONSE"

# 查看405错误响应
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == $POST_RESPONSE" \
  -T fields -e tcp.payload | xxd -r -p | head -8
```

**405错误分析**：

```http
HTTP/1.1 405 Method Not Allowed
Content-Type: text/plain; charset=utf-8
X-Content-Type-Options: nosniff
Date: Tue, 26 Aug 2025 12:05:27 GMT
Content-Length: 19

Method not allowed
```

**重要发现**：

- 🚫 当前 MCP 服务器的`/mcp/sse`端点**不支持POST方法**
- 📄 严格遵循SSE标准（SSE只能通过GET建立）
- 🔄 客户端需要降级到标准的GET方法



### 步骤2：GET建立SSE连接（成功）

**协议说明**：POST失败后，客户端立即使用标准的GET方法建立SSE长连接

**查找GET请求**：

```bash
# 查找GET /mcp/sse请求（在POST失败后）
GET_SSE_FRAME=$(tshark -r remote-exec-mcp-server.pcapng \
  -Y "frame.number > $FIRST_POST and http.request.method == \"GET\" and http.request.uri == \"/mcp/sse\"" \
  -T fields -e frame.number | head -1)
echo "GET SSE请求帧号: $GET_SSE_FRAME"

# 查看GET请求内容
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == $GET_SSE_FRAME" \
  -T fields -e tcp.payload | xxd -r -p
```

![image-20250830211033998](http://devyk.top/2022/202508302110316.png)

**GET请求分析**：

```http
GET /mcp/sse HTTP/1.1
Host: 10.1.16.4:8000
Connection: keep-alive                    # 🔥 保持长连接
User-Agent: Cursor/1.2.2 (darwin arm64)
Accept: text/event-stream                 # 🔥 只请求SSE流（无JSON）
Cache-Control: no-cache                   # 禁用缓存，确保实时性
Pragma: no-cache                          # 强制禁用缓存
```

**关键差异对比**：

| 特征         | POST请求（失败）                    | GET请求（成功）   |
| ------------ | ----------------------------------- | ----------------- |
| Content-Type | application/json                    | 无                |
| Accept       | application/json, text/event-stream | text/event-stream |
| 请求体       | JSON初始化数据                      | 无                |
| 目的         | 直接发送初始化                      | 建立接收通道      |

### 步骤3：分配会话端点

**协议说明**：服务器通过SSE成功建立连接后，推送专用的消息端点

**查找SSE成功响应**：

```bash
# 查找包含"endpoint"事件的SSE响应帧
SSE_FRAME=$(tshark -r remote-exec-mcp-server.pcapng \
  -Y "frame.number > $GET_SSE_FRAME and tcp.payload matches \"endpoint\"" \
  -T fields -e frame.number | head -1)
echo "SSE端点响应帧号: $SSE_FRAME"

# 提取SSE响应内容
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == $SSE_FRAME" \
  -T fields -e tcp.payload | xxd -r -p | head -10
```

**SSE成功响应**：

```http
SSE端点响应帧号: 12521
HTTP/1.1 200 OK
Access-Control-Allow-Origin: *
Cache-Control: no-cache
Connection: keep-alive
Content-Type: text/event-stream
Date: Tue, 26 Aug 2025 12:05:27 GMT
Transfer-Encoding: chunked

6a
event: endpoint
data: http://10.1.16.4:8000/mcp/message?sessionId=648e138a-0bf9-4b2d-bed9-7a55d7a0d8e7
```

**MCP的容错设计发现**：

这个重要发现揭示了MCP协议的**容错和降级机制**：

1. **❌ 尝试直接初始化失败**：
   - POST /mcp/sse + JSON数据（期望一步完成）
   - 服务器返回405 Method Not Allowed

2. **✅ 降级到标准SSE成功**：
   - GET /mcp/sse（严格遵循SSE标准）
   - 成功建立长连接并获得会话端点

3. **🔄 分离通信模式**：
   - **下行通道**：SSE长连接接收服务器推送
   - **上行通道**：POST到分配的消息端点发送请求

**设计优势**：

- 🛡️ **严格标准遵循**：SSE端点完全按HTTP标准实现
- 🔄 **客户端容错**：POST失败后自动降级处理
- 🌐 **网络兼容性**：标准HTTP协议，防火墙友好
- 📡 **可靠双向通信**：SSE推送 + HTTP POST的稳定组合

### 步骤4：协议初始化

**协议说明**：获得会话端点后，客户端向专用端点发送正式的初始化请求

**HTTP请求**：

```shell
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 12531" \
  -T fields -e tcp.payload | xxd -r -p
# 查看完整的HTTP请求（包括头部）
POST /mcp/message?sessionId=648e138a-0bf9-4b2d-bed9-7a55d7a0d8e7 HTTP/1.1
host: 10.1.16.4:8000
connection: keep-alive
User-Agent: Cursor/1.2.2 (darwin arm64)
content-type: application/json
accept: */*
accept-language: *
sec-fetch-mode: cors
accept-encoding: gzip, deflate
content-length: 254

{
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-06-18",
    "capabilities": {
      "tools": true,
      "prompts": false,
      "resources": false,
      "logging": false,
      "roots": {
        "listChanged": false
      }
    },
    "clientInfo": {
      "name": "cursor-vscode",
      "version": "1.0.0"
    }
  },
  "jsonrpc": "2.0",
  "id": 1
}
```



### 步骤5：能力协商响应

**协议说明**：服务器返回自己支持的能力和最终确定的协议版本

**提取命令**：

```bash
# 查找初始化响应（通过SSE推送）
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number >= 12532 and frame.number <= 12550 and tcp.payload matches \"result\"" \
  -T fields -e frame.number -e tcp.payload | head -1 | while read frame payload; do
    echo "响应帧: $frame"
    echo "$payload" | xxd -r -p | sed -n 's/.*data: \(.*\)/\1/p' | jq .
done
```

**JSON-RPC响应**：

```json
响应帧: 12538
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "tools": {}
    },
    "serverInfo": {
      "name": "Remote Execution MCP Tool",
      "version": "1.0.0"
    }
  }
}
```

**版本协商结果**：

- 客户端请求：`2025-06-18`
- 服务器确认：`2025-03-26`
- 最终使用：`2025-03-26`（以服务器版本为准）

### 步骤6：初始化完成通知

**协议说明**：客户端发送通知确认初始化流程完成

**提取命令**：

```bash
# 查看初始化完成通知
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 12551" \
  -T fields -e tcp.payload | xxd -r -p
```

**JSON-RPC通知**：

```json
POST /mcp/message?sessionId=648e138a-0bf9-4b2d-bed9-7a55d7a0d8e7 HTTP/1.1
host: 10.1.16.4:8000
connection: keep-alive
User-Agent: Cursor/1.2.2 (darwin arm64)
mcp-protocol-version: 2025-03-26
content-type: application/json
accept: */*
accept-language: *
sec-fetch-mode: cors
accept-encoding: gzip, deflate
content-length: 54

{"method":"notifications/initialized","jsonrpc":"2.0"}
```

**特点**：

- 无`id`字段（JSON-RPC通知特征）
- 服务器无需响应此消息

### 步骤7：工具发现

**协议说明**：客户端请求服务器提供的工具列表

**HTTP请求分析**：

```bash
# 查看工具列表请求的HTTP头
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 12565" \
  -T fields -e tcp.payload | xxd -r -p 
```

**请求头和JSON-RPC内容**：

```http
POST /mcp/message?sessionId=648e138a-0bf9-4b2d-bed9-7a55d7a0d8e7 HTTP/1.1
host: 10.1.16.4:8000
connection: keep-alive
User-Agent: Cursor/1.2.2 (darwin arm64)
mcp-protocol-version: 2025-03-26
content-type: application/json
accept: */*
accept-language: *
sec-fetch-mode: cors
accept-encoding: gzip, deflate
content-length: 46

{"method":"tools/list","jsonrpc":"2.0","id":2}
```



### 步骤8：工具列表响应

**协议说明**：服务器通过SSE推送可用工具的完整信息

**提取命令**：

```bash
# 查看工具列表响应
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 12570" \
  -T fields -e tcp.payload | xxd -r -p
```

**JSON-RPC响应**：

```json
event: message
data: {
  "jsonrpc": "2.0",
  "id": 2,
  "result": {
    "tools": [{
      "annotations": {
        "readOnlyHint": false,
        "destructiveHint": true,
        "idempotentHint": false,
        "openWorldHint": true
      },
      "description": "Run a shell command or script to remotely execute commands on a machine",
      "inputSchema": {
        "properties": {
          "machineId": {
            "description": "The machine id to run the script on (e.g., '75590566982b48729186ce5be91f2352')",
            "type": "string"
          },
          "script": {
            "description": "The script or command to run (must be from whitelist)",
            "type": "string"
          }
        },
        "required": ["machineId", "script"],
        "type": "object"
      },
      "name": "remote_exec"
    }]
  }
}
```

**工具安全注解分析**：

- `destructiveHint: true`：警告这是破坏性操作
- `openWorldHint: true`：表明会与外部世界交互
- `readOnlyHint: false`：非只读操作
- `idempotentHint: false`：非幂等操作

### 步骤9：工具调用

**协议说明**：客户端调用具体工具执行任务

**HTTP请求头分析**：

```bash
# 查看工具调用请求的完整HTTP内容
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 14907" \
  -T fields -e tcp.payload | xxd -r -p 
```

**工具调用请求头和请求体**：

```http
POST /mcp/message?sessionId=648e138a-0bf9-4b2d-bed9-7a55d7a0d8e7 HTTP/1.1
host: 10.1.16.4:8000
connection: keep-alive
User-Agent: Cursor/1.2.2 (darwin arm64)
mcp-protocol-version: 2025-03-26
content-type: application/json
accept: */*
accept-language: *
sec-fetch-mode: cors
accept-encoding: gzip, deflate
content-length: 184

{
  "method": "tools/call",
  "params": {
    "name": "remote_exec",
    "arguments": {
      "machineId": "1fe6633dae938e35d27efc84f06ccc1c",
      "script": "lsblk"
    },
    "_meta": {
      "progressToken": 3
    }
  },
  "jsonrpc": "2.0",
  "id": 3
}
```



**参数解析**：

- `name`：工具名称（必须匹配步骤7中的工具名）
- `arguments`：工具参数（符合inputSchema定义）
- `_meta.progressToken`：进度跟踪令牌

### 步骤10：工具执行结果

**协议说明**：服务器执行工具并通过SSE返回结果

**提取命令**：

```bash
# 查找工具执行结果
tshark -r remote-exec-mcp-server.pcapng -Y "frame.number == 14936" \
  -T fields -e tcp.payload | xxd -r -p
```

**JSON-RPC响应**：

```json
event: message
data: {
  "jsonrpc": "2.0",
  "id": 3,
  "result": {
    "content": [{
      "type": "text",
      "text": "Machine: 1fe6633dae938e35d27efc84f06ccc1c\nCommand: lsblk\nExit Code: 0\nStdout:\nNAME        MAJ:MIN RM   SIZE RO TYPE MOUNTPOINTS\nnvme2n1     259:0    0 238.5G  0 disk \n├─nvme2n1p1 259:1    0     1M  0 part \n├─nvme2n1p2 259:2    0     4G  0 part /boot\n├─nvme2n1p3 259:3    0    15G  0 part /live\n└─nvme2n1p4 259:4    0 219.5G  0 part /\nnvme0n1     259:5    0   1.9T  0 disk \nnvme1n1     259:6    0   1.8T  0 disk \nnvme3n1     259:7    0   1.8T  0 disk \n"
    }]
  }
}
```

**结果格式分析**：

- `content`：标准化的内容数组
- `type: "text"`：文本类型内容
- `text`：实际的执行结果

### 协议流程验证

**验证完整流程的命令**：

```bash
# 统计所有JSON-RPC方法
tshark -r remote-exec-mcp-server.pcapng -Y "tcp.payload matches \"method\"" \
  -T fields -e frame.number -e tcp.payload | \
  while read frame payload; do
    method=$(echo "$payload" | xxd -r -p | grep -o '"method":"[^"]*"' | head -1)
    if [ -n "$method" ]; then
      echo "帧 $frame: $method"
    fi
  done

```

![image-20250830213140064](http://devyk.top/2022/202508302131390.png)

**验证会话一致性**：

```bash
# 检查所有消息是否使用同一会话ID
tshark -r remote-exec-mcp-server.pcapng -Y "http.request.uri contains sessionId" \
  -T fields -e http.request.uri | grep -o 'sessionId=[^&]*' | sort | uniq
```

![image-20250830213228404](http://devyk.top/2022/202508302132634.png)

### 传输层特性分析

**验证传输协议**：

```bash
# 确认使用HTTP而非WebSocket
tshark -r remote-exec-mcp-server.pcapng -Y "websocket" -c 10
# 无输出说明没有使用WebSocket

# 查看HTTP特征
tshark -r remote-exec-mcp-server.pcapng -Y "tcp.port == 8000 and tcp.payload" \
  -T fields -e tcp.payload | head -3 | xxd -r -p | grep -E "(GET|POST|HTTP)"
```

![image-20250830213317826](http://devyk.top/2022/202508302133441.png)

**MCP传输设计总结**：

- **HTTP + SSE**：双向通信但实现简单
- **会话隔离**：每个会话有独立的消息端点
- **协议协商**：支持版本兼容性检查
- **实时推送**：SSE用于服务器主动推送响应

### 重要发现：MCP协议的容错机制

通过深入分析真实的网络流量，我们发现了MCP协议一个重要而巧妙的设计特性：

#### 协议适应性和标准遵循

**观察到的行为**：

1. 客户端首先尝试 `POST /mcp/sse` 直接发送初始化数据
2. 服务器严格拒绝（405 Method Not Allowed）
3. 客户端立即降级到标准的 `GET /mcp/sse` 建立SSE连接
4. 成功建立连接后通过专用端点进行后续通信

**设计哲学**：

- 🎯 **尝试优化，接受标准**：客户端尝试一步完成初始化，失败后遵循标准流程
- 🛡️ **服务器严格性**：严格按照HTTP和SSE标准实现，确保兼容性
- 🔄 **客户端灵活性**：具备降级和容错能力，适应不同服务器实现

**实际价值**：

- ✅ **网络兼容性**：标准HTTP协议，所有网络设备都支持
- ✅ **调试便利性**：可以用curl、Postman等标准工具测试
- ✅ **企业友好**：防火墙、代理服务器完全兼容
- ✅ **标准遵循**：SSE规范的正确实现，确保互操作性

这种设计在企业级环境中具有极佳的网络兼容性和调试便利性，体现了MCP协议设计的成熟度和实用性。



## 实战开发

通过前面对MCP协议的深入理解，我们现在来动手构建一个完整的MCP生态系统。在这个实战部分，我将基于两种不同的实现方式来展示MCP的强大之处：

1. **手动实现**：从零开始构建 JSON-RPC 2.0 协议和 MCP 规范，深入理解协议细节
2. **官方SDK**：使用官方Go SDK快速构建生产级的MCP应用

让我们从实际需求出发：构建一个SSH远程执行服务，让AI助手能够安全地管理远程服务器。

### 第一部分：手动实现 - 深入理解 MCP 协议

#### 为什么要手动实现？

在直接使用官方 SDK 之前，手动实现 MCP 协议有几个重要价值：

1. **深度理解**：真正掌握 JSON-RPC 2.0 和 MCP 的工作原理
2. **灵活定制**：可以根据特殊需求进行深度定制
3. **调试能力**：遇到问题时能够快速定位和解决
4. **学习价值**：为后续使用官方SDK打下坚实基础

#### 项目架构设计

我们的手动实现支持两种传输方式，这正好对应了 MCP 协议的灵活性：

```
手动实现架构
├── Stdio传输 (进程间通信)
│   ├── ssh-mcp-server      # 服务器
│   └── ssh-mcp-client      # 客户端
└── HTTP SSE传输 (网络通信)
    ├── ssh-mcp-sse-server  # HTTP服务器
    └── ssh-mcp-sse-client  # HTTP客户端
```

这种设计让我们能够：

- **本地使用**：通过stdio进行快速的进程间通信
- **分布式部署**：通过HTTP SSE支持网络分布式架构

#### Stdio传输实现

##### 服务端核心实现

Stdio传输是MCP最基础的传输方式，服务器通过标准输入输出与客户端通信：

```go
// MCPServer 核心服务器结构
type MCPServer struct {
  config    *config.Config     // 服务器配置
  sshClient *ssh.Client        // SSH客户端
  ctx       context.Context    // 上下文
  cancel    context.CancelFunc // 取消函数
  mutex     sync.RWMutex       // 读写锁
  logger    *log.Logger        // 日志记录器

  // 输入输出流
  reader *bufio.Scanner // 输入流读取器
  writer io.Writer      // 输出流写入器

  // 状态管理
  initialized  bool                     // 是否已初始化
  capabilities types.ServerCapabilities // 服务器能力
}
```

**主消息循环**：

```go
// Run 启动MCP服务器主循环
func (s *MCPServer) Run() error {
  s.logger.Println("MCP服务器启动")
  defer s.logger.Println("MCP服务器停止")
  defer s.sshClient.Close()

  // 主消息循环：从stdin读取JSON-RPC消息
  for s.reader.Scan() {
    line := s.reader.Text()
    if line == "" {
      continue
    }

    var request types.MCPRequest
    if err := json.Unmarshal([]byte(line), &request); err != nil {
      s.sendError(nil, types.ParseError, "解析JSON失败", err.Error())
      continue
    }

    s.logger.Printf("收到消息: %s", line)
    if err := s.handleRequest(&request); err != nil {
      s.logger.Printf("处理请求失败: %v", err)
    }
  }

  return nil
}
```

**工具定义和处理**：

这是手动实现的核心部分，我们需要严格按照MCP规范定义工具：

```go
// handleToolsList 处理工具列表请求
func (s *MCPServer) handleToolsList(request *types.MCPRequest) error {
  // 定义可用工具 - 这里体现了手动实现的灵活性
  tools := []types.Tool{
    {
      Name:        "ssh_execute",
      Description: "在远程服务器上执行Shell命令",
      InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
          "host": map[string]interface{}{
            "type":        "string",
            "description": "目标服务器地址",
          },
          "command": map[string]interface{}{
            "type":        "string", 
            "description": "要执行的命令",
          },
          "user": map[string]interface{}{
            "type":        "string",
            "description": "SSH用户名",
            "default":     s.config.SSH.DefaultUser,
          },
          "password": map[string]interface{}{
            "type":        "string",
            "description": "SSH密码（可选，优先使用密钥认证）",
          },
        },
        "required": []string{"host", "command"},
      },
    },
    // 可以继续添加更多工具...
  }

  result := types.ToolsListResult{Tools: tools}
  return s.sendResult(request.ID, result)
}
```

**工具执行处理**：

```go
func (s *SSHMCPServer) handleToolsCall(params interface{}) interface{} {
    paramsMap := params.(map[string]interface{})
    toolName := paramsMap["name"].(string)
    arguments := paramsMap["arguments"].(map[string]interface{})
    
    switch toolName {
    case "ssh_execute":
        return s.executeSSHCommand(arguments)
    default:
        return &MCPError{
            Code:    -32601,
            Message: fmt.Sprintf("未知工具: %s", toolName),
        }
    }
}

func (s *SSHMCPServer) executeSSHCommand(args map[string]interface{}) interface{} {
    host := args["host"].(string)
    command := args["command"].(string)
    user, ok := args["user"].(string)
    if !ok {
        user = "root"
    }
    
    // 建立SSH连接
    client, err := s.getSSHClient(host, user)
    if err != nil {
        return &MCPError{
            Code:    -32000,
            Message: "SSH连接失败",
            Data:    map[string]interface{}{"error": err.Error()},
        }
    }
    
    // 执行命令
    session, err := client.NewSession()
    if err != nil {
        return &MCPError{
            Code:    -32000,
            Message: "创建SSH会话失败",
            Data:    map[string]interface{}{"error": err.Error()},
        }
    }
    defer session.Close()
    
    output, err := session.CombinedOutput(command)
    if err != nil {
        return &MCPError{
            Code:    -32000,
            Message: "命令执行失败",
            Data:    map[string]interface{}{"error": err.Error()},
        }
    }
    
    return map[string]interface{}{
        "content": []map[string]interface{}{
            {
                "type": "text",
                "text": string(output),
            },
        },
    }
}
```

##### 主消息循环

```go
func (s *SSHMCPServer) Run() {
    scanner := bufio.NewScanner(os.Stdin)
    
    for scanner.Scan() {
        line := scanner.Text()
        if line == "" {
            continue
        }
        
        var request MCPRequest
        if err := json.Unmarshal([]byte(line), &request); err != nil {
            s.sendError(nil, -32700, "解析错误", err.Error())
            continue
        }
        
        s.handleRequest(&request)
    }
}

func (s *SSHMCPServer) handleRequest(request *MCPRequest) {
    var result interface{}
    var err *MCPError
    
    switch request.Method {
    case "initialize":
        result = s.handleInitialize(request.Params)
    case "tools/list":
        result = s.handleToolsList()
    case "tools/call":
        result = s.handleToolsCall(request.Params)
    case "notifications/initialized":
        // 初始化完成通知，无需响应
        return
    default:
        err = &MCPError{
            Code:    -32601,
            Message: "方法未找到",
        }
    }
    
    response := MCPResponse{
        JSONRPC: "2.0",
        ID:      request.ID,
        Result:  result,
        Error:   err,
    }
    
    s.sendResponse(&response)
}
```

##### 配置和部署

创建配置文件 `config.yaml`：

```yaml
ssh:
  default_user: "admin"
  timeout: 30
  key_file: "~/.ssh/id_rsa"
  
logging:
  level: "info"
  file: "/var/log/ssh-mcp-server.log"
  
```



##### 客户端核心实现

Stdio客户端通过子进程启动服务器，并通过管道进行通信：

```go
// MCPClient MCP客户端
type MCPClient struct {
  cmd          *exec.Cmd
  stdin        io.WriteCloser
  stdout       io.ReadCloser
  stderr       io.ReadCloser
  messageID    int
  pendingReqs  map[int]chan *JSONRPCResponse
  mu           sync.RWMutex
  ctx          context.Context
  cancel       context.CancelFunc
  initialized  bool
}

// NewMCPClient 创建新的MCP客户端
func NewMCPClient(serverCommand []string) (*MCPClient, error) {
  ctx, cancel := context.WithCancel(context.Background())
  
  cmd := exec.CommandContext(ctx, serverCommand[0], serverCommand[1:]...)
  
  stdin, err := cmd.StdinPipe()
  if err != nil {
    cancel()
    return nil, fmt.Errorf("创建stdin管道失败: %w", err)
  }
  
  stdout, err := cmd.StdoutPipe()
  if err != nil {
    cancel()
    return nil, fmt.Errorf("创建stdout管道失败: %w", err)
  }
  
  stderr, err := cmd.StderrPipe()
  if err != nil {
    cancel()
    return nil, fmt.Errorf("创建stderr管道失败: %w", err)
  }
  
  return &MCPClient{
    cmd:         cmd,
    stdin:       stdin,
    stdout:      stdout,
    stderr:      stderr,
    pendingReqs: make(map[int]chan *JSONRPCResponse),
    ctx:         ctx,
    cancel:      cancel,
  }, nil
}
```

###### 连接管理

```go
// Connect 连接到MCP服务器
func (c *MCPClient) Connect() error {
  // 启动服务器进程
  if err := c.cmd.Start(); err != nil {
    return fmt.Errorf("启动服务器失败: %w", err)
  }
  
  // 启动消息读取协程
  go c.readMessages()
  
  // 发送初始化请求
  initReq := &JSONRPCRequest{
    JSONRPC: "2.0",
    ID:      c.nextMessageID(),
    Method:  "initialize",
    Params: map[string]interface{}{
      "protocolVersion": "2024-11-05",
      "capabilities":    map[string]interface{}{},
      "clientInfo": map[string]interface{}{
        "name":    "ssh-mcp-client",
        "version": "1.0.0",
      },
    },
  }
  
  resp, err := c.sendRequest(initReq)
  if err != nil {
    return fmt.Errorf("初始化失败: %w", err)
  }
  
  if resp.Error != nil {
    return fmt.Errorf("初始化错误: %v", resp.Error)
  }
  
  // 发送initialized通知
  initNotification := &JSONRPCRequest{
    JSONRPC: "2.0",
    Method:  "notifications/initialized",
  }
  
  if err := c.sendNotification(initNotification); err != nil {
    return fmt.Errorf("发送initialized通知失败: %w", err)
  }
  
  c.initialized = true
  return nil
}
```

#### HTTP SSE传输实现

##### 服务端核心实现

HTTP SSE传输支持网络分布式部署，服务器提供HTTP API接口：

```go
// SSEServer HTTP SSE MCP服务器
type SSEServer struct {
  mcpServer *MCPServer
  sessions  map[string]*SSESession
  mu        sync.RWMutex
}

// SSESession SSE会话
type SSESession struct {
  ID       string
  Writer   http.ResponseWriter
  Flusher  http.Flusher
  Messages chan []byte
  Done     chan struct{}
}

// NewSSEServer 创建新的SSE服务器
func NewSSEServer(mcpServer *MCPServer) *SSEServer {
  return &SSEServer{
    mcpServer: mcpServer,
    sessions:  make(map[string]*SSESession),
  }
}

// HandleSSE 处理SSE连接
func (s *SSEServer) HandleSSE(w http.ResponseWriter, r *http.Request) {
  // 设置SSE头
  w.Header().Set("Content-Type", "text/event-stream")
  w.Header().Set("Cache-Control", "no-cache")
  w.Header().Set("Connection", "keep-alive")
  w.Header().Set("Access-Control-Allow-Origin", "*")
  
  flusher, ok := w.(http.Flusher)
  if !ok {
    http.Error(w, "不支持流式响应", http.StatusInternalServerError)
    return
  }
  
  // 创建会话
  sessionID := generateSessionID()
  session := &SSESession{
    ID:       sessionID,
    Writer:   w,
    Flusher:  flusher,
    Messages: make(chan []byte, 100),
    Done:     make(chan struct{}),
  }
  
  s.mu.Lock()
  s.sessions[sessionID] = session
  s.mu.Unlock()
  
  // 发送端点事件
  endpoint := fmt.Sprintf("/messages?sessionId=%s", sessionID)
  s.sendSSEEvent(session, "endpoint", endpoint)
  
  // 处理会话
  s.handleSession(session, r.Context())
}
```

##### 客户端核心实现

HTTP SSE客户端通过HTTP请求与服务器通信：

```go
// SSEClient HTTP SSE MCP客户端
type SSEClient struct {
  baseURL     string
  httpClient  *http.Client
  messageID   int
  pendingReqs map[int]chan *JSONRPCResponse
  mu          sync.RWMutex
  ctx         context.Context
  cancel      context.CancelFunc
  endpoint    string
}

// Connect 连接到MCP服务器
func (c *SSEClient) Connect() error {
  // 建立SSE连接
  resp, err := c.httpClient.Get(c.baseURL + "/sse")
  if err != nil {
    return fmt.Errorf("连接SSE失败: %w", err)
  }
  
  // 读取SSE流，等待端点事件
  scanner := bufio.NewScanner(resp.Body)
  for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "event: endpoint") {
      // 读取下一行获取数据
      if scanner.Scan() {
        dataLine := scanner.Text()
        if strings.HasPrefix(dataLine, "data: ") {
          c.endpoint = strings.TrimPrefix(dataLine, "data: ")
          break
        }
      }
    }
  }
  
  // 发送初始化请求
  return c.initialize()
}
```

#### 运行流程对比

##### Stdio模式运行流程

```bash
# 构建项目
cd go_jsonrpc
make build

# 演示模式
make run-client

# 交互模式
./build/ssh-mcp-client \
  -tool ssh_execute \
  -tool-args '{"host":"192.168.71.111","command":"whoami && hostname && uptime","port":22,"user":"root","password":"xxx"}' \
  -server ./build/ssh-mcp-server \
  -args "-config config.yaml"
```

运行结果:

```
MCP服务器连接成功
开始MCP初始化
客户端发送请求: 
{"jsonrpc":"2.0","id":"req-1757037798493353000","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"sampling":{},"tools":true},"clientInfo":{"name":"SSH-MCP-Client","version":"1.0.0"}}}
服务端收到消息: 
{"jsonrpc":"2.0","id":"req-1757037798493353000","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{"sampling":{},"tools":true},"clientInfo":{"name":"SSH-MCP-Client","version":"1.0.0"}}}
服务端响应:
{"jsonrpc":"2.0","id":"req-1757037798493353000","result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"SSH-MCP-Server","version":"1.0.0"}}}
客户端收到响应:
{"jsonrpc":"2.0","id":"req-1757037798493353000","result":{"protocolVersion":"2025-03-26","capabilities":{"tools":{"listChanged":true}},"serverInfo":{"name":"SSH-MCP-Server","version":"1.0.0"}}}
客户端发送初始化完成的通知:
{"jsonrpc":"2.0","method":"notifications/initialized"}
服务端收到初始化完成的通知:
 {"jsonrpc":"2.0","method":"notifications/initialized"}
客户端发起执行命令的请求:
{"jsonrpc":"2.0","id":"req-1757037798773285000","method":"tools/call","params":{"name":"ssh_execute","arguments":{"command":"whoami \u0026\u0026 hostname \u0026\u0026 uptime","host":"192.168.71.111","password":"ppio123","port":22,"user":"root"}}}
[MCP-Client] 2025/09/05 10:03:18 client.go:347: 服务器日志: [MCP-Server] 2025/09/05 10:03:18 server.go:105: 收到消息: {"jsonrpc":"2.0","id":"req-1757037798773285000","method":"tools/call","params":{"name":"ssh_execute","arguments":{"command":"whoami \u0026\u0026 hostname \u0026\u0026 uptime","host":"192.168.71.111","password":"ppio123","port":22,"user":"root"}}}
服务端收到并处理:
...
客户端收到处理结果:
{"jsonrpc":"2.0","id":"req-1757037798773285000","result":{"content":[{"type":"text","text":"主机: 192.168.71.111\n命令: whoami \u0026\u0026 hostname \u0026\u0026 uptime\n退出码: 0\n执行时长: 331.195792ms\n标准输出:\nroot\nalam\n 10:03:19 up 15 days, 22:32,  1 user,  load average: 0.05, 0.28, 0.16\n\n"}],"isError":false}}

关闭MCP客户端
MCP客户端已关闭


```



##### HTTP SSE模式运行流程

```bash
# 终端1：启动SSE服务器
cd go_jsonrpc  
make run-sse-server

# 终端2：启动SSE客户端
make run-sse-client

# 或者手动启动
./build/ssh-mcp-sse-client \
  -server http://localhost:8000 \
  -mode call \
  -tool ssh_execute \
  -args '{"host":"192.168.71.111","command":"whoami","port":22,"user":"root","password":"xxx"}'
```

测试结果如下所示:

![image-20250905164814733](http://devyk.top/2022/202509051648764.png)



用 Cursor 测试也是正常的，说明我们手动实现的 MCP + JSON-RPC 是没有问题的

![image-20250905165940085](http://devyk.top/2022/202509051659088.png)

### 第二部分：官方SDK实现 - 生产级开发

#### 官方SDK的优势

在理解了MCP协议的底层实现后，官方SDK的价值就更加明显了：

1. **类型安全**：Go泛型确保编译时类型检查
2. **自动Schema生成**：通过jsonschema标签自动生成JSON Schema
3. **标准化API**：完全符合官方MCP协议规范
4. **开发效率**：减少样板代码，专注业务逻辑
5. **维护性**：官方维护，及时跟进协议更新

#### 项目结构

```
官方SDK实现
├── ssh-mcp-server-sdk  # 基于SDK的服务器
└── ssh-mcp-client-sdk  # 基于SDK的客户端
```

#### 核心数据结构定义

官方SDK的一大特色是使用结构体和jsonschema标签自动生成JSON Schema：

```go
// SSHExecuteParams SSH命令执行参数
// jsonschema标签会自动生成JSON Schema
type SSHExecuteParams struct {
  Host    string `json:"host" jsonschema:"description=目标主机地址"`
  Command string `json:"command" jsonschema:"description=要执行的命令"`
  User    string `json:"user,omitempty" jsonschema:"description=SSH用户名"`
  Port    int    `json:"port,omitempty" jsonschema:"description=SSH端口"`
  Timeout int    `json:"timeout,omitempty" jsonschema:"description=超时时间（秒）"`
}

// SSHExecuteResult SSH命令执行结果
// 结构化的返回结果，便于AI理解和处理
type SSHExecuteResult struct {
  Host     string `json:"host" jsonschema:"description=目标主机"`
  Command  string `json:"command" jsonschema:"description=执行的命令"`
  ExitCode int    `json:"exitCode" jsonschema:"description=退出码"`
  Stdout   string `json:"stdout" jsonschema:"description=标准输出"`
  Stderr   string `json:"stderr" jsonschema:"description=标准错误"`
  Duration string `json:"duration" jsonschema:"description=执行时长"`
}
```

#### 服务器实现

##### 服务器结构

```go
// MCPSSHServer 基于官方SDK的SSH MCP服务器
type MCPSSHServer struct {
  config    *config.Config  // 配置
  sshClient *ssh.Client     // SSH客户端
  server    *mcp.Server     // MCP服务器实例
}

// NewMCPSSHServer 创建新的SSH MCP服务器
func NewMCPSSHServer(cfg *config.Config) (*MCPSSHServer, error) {
  // 创建SSH客户端
  sshConfig := &ssh.Config{
    DefaultUser:    cfg.SSH.DefaultUser,
    DefaultPort:    cfg.SSH.DefaultPort,
    Timeout:        cfg.SSH.Timeout,
    KeyFile:        cfg.SSH.KeyFile,
    KnownHostsFile: cfg.SSH.KnownHostsFile,
    MaxConnections: cfg.SSH.MaxConnections,
  }
  sshClient := ssh.NewClient(sshConfig)

  // 创建MCP服务器实例
  serverImpl := &mcp.Implementation{
    Name:    cfg.Server.Name,
    Version: cfg.Server.Version,
  }

  // 定义服务器选项
  options := &mcp.ServerOptions{
    HasTools: true, // 声明支持工具
  }

  server := mcp.NewServer(serverImpl, options)

  mcpServer := &MCPSSHServer{
    config:    cfg,
    sshClient: sshClient,
    server:    server,
  }

  // 注册工具
  mcpServer.registerTools()

  return mcpServer, nil
}
```

##### 工具注册 - SDK的核心优势

这里展现了官方SDK最大的优势：类型安全的工具注册：

```go
// registerTools 注册MCP工具
func (s *MCPSSHServer) registerTools() {
  // 注册SSH命令执行工具
  sshExecuteTool := &mcp.Tool{
    Name:        "ssh_execute",
    Description: "在远程服务器上执行Shell命令",
    // InputSchema 将由AddTool自动生成！
  }

  // 使用官方SDK的AddTool方法注册工具（带类型安全）
  // 这里的泛型确保了参数类型的编译时检查
  mcp.AddTool(s.server, sshExecuteTool, s.handleSSHExecute)

  // 注册 SSH 文件传输工具
  sshFileTransferTool := &mcp.Tool{
    Name:        "ssh_file_transfer", 
    Description: "SSH 文件传输（上传/下载）",
    // InputSchema 也会自动生成
  }

  mcp.AddTool(s.server, sshFileTransferTool, s.handleSSHFileTransfer)
}
```

##### 工具处理函数 - 类型安全的威力

```go
// handleSSHExecute 处理SSH命令执行工具调用
// 注意：这里的参数类型是编译时检查的！
func (s *MCPSSHServer) handleSSHExecute(ctx context.Context, req *mcp.CallToolRequest, args SSHExecuteParams) (*mcp.CallToolResult, SSHExecuteResult, error) {
  log.Printf("执行SSH命令: %s@%s:%d - %s", args.User, args.Host, args.Port, args.Command)

  // 填充默认值
  if args.User == "" {
    args.User = s.config.SSH.DefaultUser
  }
  if args.Port == 0 {
    args.Port = s.config.SSH.DefaultPort
  }
  if args.Timeout == 0 {
    args.Timeout = int(s.config.SSH.Timeout.Seconds())
  }

  // 创建SSH连接信息
  connInfo := &ssh.ConnectionInfo{
    Host: args.Host,
    Port: args.Port,
    User: args.User,
  }

  // 执行SSH命令
  result, err := s.sshClient.Execute(connInfo, args.Command)
  if err != nil {
    return nil, SSHExecuteResult{}, fmt.Errorf("SSH命令执行失败: %w", err)
  }

  // 构建响应内容
  infoText := fmt.Sprintf("主机: %s\n命令: %s\n退出码: %d\n执行时长: %v\n",
    args.Host, args.Command, result.ExitCode, result.Duration)

  if result.Stdout != "" {
    infoText += fmt.Sprintf("标准输出:\n%s\n", result.Stdout)
  }

  if result.Stderr != "" {
    infoText += fmt.Sprintf("标准错误:\n%s\n", result.Stderr)
  }

  // 使用官方SDK的内容类型
  content := []mcp.Content{
    &mcp.TextContent{
      Text: infoText,
    },
  }

  // 构建结构化结果 - 这是SDK版本的优势
  structuredResult := SSHExecuteResult{
    Host:     args.Host,
    Command:  args.Command,
    ExitCode: result.ExitCode,
    Stdout:   result.Stdout,
    Stderr:   result.Stderr,
    Duration: result.Duration.String(),
  }

  return &mcp.CallToolResult{
    Content: content,
    IsError: result.ExitCode != 0,
  }, structuredResult, nil
}

```

##### 服务器启动

```go
// Run 启动MCP服务器
func (s *MCPSSHServer) Run(ctx context.Context) error {
  log.Println("启动SSH MCP服务器（基于官方SDK）")
  defer log.Println("SSH MCP服务器已停止")
  defer s.sshClient.Close()

  // 使用官方SDK的StdioTransport运行服务器
  transport := &mcp.StdioTransport{}
  return s.server.Run(ctx, transport)
}
```

#### 客户端实现

##### 客户端连接

官方SDK的客户端实现更加简洁：

```go
func main() {
  // 创建上下文
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // 创建MCP客户端实例
  clientImpl := &mcp.Implementation{
    Name:    "SSH-MCP-Client-SDK",
    Version: "1.0.0",
  }

  // 定义客户端选项
  options := &mcp.ClientOptions{}
  client := mcp.NewClient(clientImpl, options)

  // 创建命令传输
  serverCommand := []string{"./build/ssh-mcp-server-sdk", "-config", "config.yaml"}
  cmd := exec.Command(serverCommand[0], serverCommand[1:]...)
  transport := mcp.NewCommandTransport(cmd)

  // 连接到服务器
  fmt.Printf("连接到MCP服务器: %v\n", serverCommand)
  session, err := client.Connect(ctx, transport, nil)
  if err != nil {
    log.Fatalf("连接MCP服务器失败: %v", err)
  }
  defer session.Close()

  fmt.Printf("成功连接到MCP服务器\n")

  // 演示工具调用
  runDemo(ctx, session)
}
```

##### 工具调用示例

```go
// runDemo 运行演示
func runDemo(ctx context.Context, session *mcp.ClientSession) {
  fmt.Println("\n=== MCP SSH客户端演示（官方SDK版本）===")

  // 列出可用工具
  listTools(ctx, session)

  // 演示SSH命令执行
  fmt.Println("\n=== 演示SSH命令执行 ===")
  args := map[string]interface{}{
    "host":    "192.168.71.111",
    "command": "echo 'Hello from MCP SSH Server SDK!' && date && uptime",
    "user":    "root",
    "password": "ppio123",
  }

  params := &mcp.CallToolParams{
    Name:      "ssh_execute",
    Arguments: args,
  }

  result, err := session.CallTool(ctx, params)
  if err != nil {
    log.Printf("SSH命令执行失败: %v", err)
    return
  }

  fmt.Println("执行结果:")
  for _, content := range result.Content {
    if textContent, ok := content.(*mcp.TextContent); ok {
      fmt.Println(textContent.Text)
    }
  }
}
```

#### 传输方式支持

官方SDK同样支持多种传输方式，让我们看看具体实现：

##### Stdio传输（本地进程通信）

```go
// Run 启动MCP服务器
func (s *MCPSSHServer) Run(ctx context.Context) error {
  log.Println("启动SSH MCP服务器（基于官方SDK）")
  defer log.Println("SSH MCP服务器已停止")
  defer s.sshClient.Close()

  // 使用官方SDK的StdioTransport运行服务器
  transport := &mcp.StdioTransport{}
  return s.server.Run(ctx, transport)
}
```

##### HTTP SSE传输（网络通信）

```go
// RunSSE 启动基于HTTP SSE的MCP服务器
func (s *MCPSSHServer) RunSSE(ctx context.Context, port int) error {
  log.Printf("启动SSH MCP SSE服务器（基于官方SDK）在端口 %d", port)
  defer log.Println("SSH MCP SSE服务器已停止")
  defer s.sshClient.Close()

  // 创建HTTP服务器
  mux := http.NewServeMux()
  
  // 创建SSE处理器 - SDK自动处理协议细节
  handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
    return s.server
  })
  
  // 注册路由
  mux.Handle("/mcp/sse", handler)
  
  server := &http.Server{
    Addr:    fmt.Sprintf(":%d", port),
    Handler: mux,
  }
  
  log.Printf("MCP SSE服务器正在监听端口 %d", port)
  return server.ListenAndServe()
}
```

##### 客户端传输选择

```go
// 创建命令传输
var transport mcp.Transport
if *sseMode {
  // HTTP SSE传输 - 适用于远程服务器
  transport = mcp.NewSSEClientTransport(*serverURL, nil)
} else {
  // Stdio传输 - 适用于本地进程
  cmd := exec.Command(serverCommand[0], serverCommand[1:]...)
  transport = mcp.NewCommandTransport(cmd)
}
```

#### 运行流程展示

##### 构建和运行

1. **构建项目**：

```bash
cd go-sdk
make build
```

2. **Stdio模式**（推荐用于开发）：

```bash
./build/ssh-mcp-client-sdk -server ./build/ssh-mcp-server-sdk -args "-config config.yaml" -tool ssh_execute -tool-args '{"host":"192.168.71.111","command":"whoami && hostname && uptime","port":22,"user":"root","password":"xxx"}'
```

![image-20250905172453930](http://devyk.top/2022/202509051724717.png)



3. **HTTP SSE模式**（推荐用于生产）

```bash
./build/ssh-mcp-server-sdk -config config.yaml -sse -port 8080
```



![image-20250905172828311](http://devyk.top/2022/202509051728929.png)

cursor 测试也是正常的:

![image-20250905180132620](http://devyk.top/2022/202509051801212.png)





#### 两种实现方式对比

| 特性         | 手动JSON-RPC实现       | 官方SDK实现            |
| ------------ | ---------------------- | ---------------------- |
| **学习价值** | ⭐⭐⭐⭐⭐ 深度理解协议     | ⭐⭐⭐ 快速上手           |
| **开发效率** | ⭐⭐⭐ 需要更多代码       | ⭐⭐⭐⭐⭐ 高效简洁         |
| **类型安全** | ⭐⭐ 运行时检查          | ⭐⭐⭐⭐⭐ 编译时检查       |
| **传输方式** | ⭐⭐⭐⭐⭐ Stdio + HTTP SSE | ⭐⭐⭐⭐⭐ Stdio + HTTP SSE |
| **定制能力** | ⭐⭐⭐⭐⭐ 完全可控         | ⭐⭐⭐ SDK限制            |
| **维护成本** | ⭐⭐ 需要手动维护        | ⭐⭐⭐⭐⭐ 官方维护         |
| **生产就绪** | ⭐⭐⭐⭐ 需要更多测试      | ⭐⭐⭐⭐⭐ 生产级           |
| **调试能力** | ⭐⭐⭐⭐⭐ 完全可控         | ⭐⭐⭐ SDK抽象            |

##### 传输方式详细对比

| 传输方式       | 手动实现               | 官方SDK                  |
| -------------- | ---------------------- | ------------------------ |
| **Stdio**      | ✅ 完全自定义实现       | ✅ `mcp.StdioTransport{}` |
| **HTTP SSE**   | ✅ 自建HTTP服务器+SSE   | ✅ `mcp.NewSSEHandler()`  |
| **配置复杂度** | 🔧 需要手动处理所有细节 | 🎯 一行代码切换传输方式   |
| **错误处理**   | 🛠️ 自行实现重连逻辑     | 🔒 SDK内置错误恢复        |
| **性能优化**   | ⚡ 可深度优化           | 📊 SDK已优化              |

#### 选择建议

**学习阶段**：建议从手动实现开始

- 深入理解MCP协议原理
- 掌握JSON-RPC 2.0的实现细节
- 了解不同传输方式的特点
- 体验从零构建完整MCP生态的过程

**生产环境**：推荐使用官方SDK

- 类型安全，减少运行时错误
- 开发效率高，代码简洁
- 官方维护，及时更新
- 同样支持多种传输方式
- 内置错误处理和性能优化

**特殊需求**：考虑手动实现

- 需要极致的性能优化
- 需要深度定制协议行为
- 需要完全的控制权
- 需要特殊的传输协议（如WebSocket）

**传输方式选择**：

- **Stdio**：适用于本地开发和测试
- **HTTP SSE**：适用于生产环境和远程部署
- **选择原则**：开发用Stdio，生产用SSE

通过这两种实现方式的对比，我们不仅深入理解了MCP协议的工作原理，也掌握了在不同场景下选择合适实现方式的能力。更重要的是，我们看到了官方SDK如何在保持简洁性的同时，提供了与手动实现相同的功能覆盖。为了深入加深整个 AI LLM 使用 MCP 的流程，下一小节我们使用 Go 开发一个 AI Chat 来调用咱们开发的 MCP ,而不依赖 cursor IDE. 



### Go + OpenAI SDK 接入 MCP Client

在前面我们实现了两种MCP服务器（手动JSON-RPC和官方SDK），现在我们需要创建一个智能聊天客户端，它能够：

1. 集成OpenAI API进行自然语言处理
2. 连接到MCP服务器获取工具能力
3. 自动调用合适的MCP工具来完成用户任务
4. 提供流畅的对话体验

#### 项目架构设计

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

#### 核心组件实现

##### 1. 配置管理

首先定义配置结构，支持OpenAI和MCP的灵活配置：

```go
// Config 应用程序配置结构
type Config struct {
  OpenAI OpenAIConfig `yaml:"openai"` // OpenAI配置
  MCP    MCPConfig    `yaml:"mcp"`    // MCP配置
  Chat   ChatConfig   `yaml:"chat"`   // 聊天配置
  Log    LogConfig    `yaml:"log"`    // 日志配置
}

// OpenAIConfig OpenAI配置
type OpenAIConfig struct {
  APIKey      string        `yaml:"api_key"`      // OpenAI API密钥
  BaseURL     string        `yaml:"base_url"`     // API基础URL，支持自定义端点
  Model       string        `yaml:"model"`        // 使用的模型名称
  Temperature float32       `yaml:"temperature"`  // 温度参数
  MaxTokens   int           `yaml:"max_tokens"`   // 最大令牌数
  Timeout     time.Duration `yaml:"timeout"`      // 请求超时时间
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
```

##### 2. MCP客户端管理器

创建一个MCP客户端管理器，负责管理多个MCP服务器连接：

```go
// MCPClient MCP客户端管理器
type MCPClient struct {
  servers   map[string]*ServerConnection // 服务器连接映射
  tools     map[string]*ToolInfo         // 工具信息映射
  config    *config.Config               // 配置
  ctx       context.Context              // 上下文
  cancel    context.CancelFunc           // 取消函数
  mutex     sync.RWMutex                 // 读写锁
  logger    *log.Logger                  // 日志记录器
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
    if len(connectErrors) == len(enabledServers) {
      return fmt.Errorf("所有MCP服务器连接失败")
    }
  }
  
  c.logger.Printf("MCP客户端启动完成，成功连接 %d 个服务器", len(c.servers))
  return nil
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
    "name":      toolInfo.Tool.Name,
    "arguments": arguments,
  }
  
  response, err := c.sendServerRequest(conn, "tools/call", params)
  if err != nil {
    return nil, fmt.Errorf("调用工具失败: %w", err)
  }
  
  if response.Error != nil {
    return nil, fmt.Errorf("工具调用错误: %s", response.Error.Message)
  }
  
  // 解析并返回结果
  return c.parseToolResult(response.Result)
}
```

##### 3. 聊天引擎

集成OpenAI API和MCP工具调用的智能聊天引擎：

```go
// ChatEngine 聊天引擎
type ChatEngine struct {
  config    *config.Config     // 配置
  openai    *openai.Client     // OpenAI客户端
  mcpClient *mcp.MCPClient     // MCP客户端
  logger    *log.Logger        // 日志记录器
  
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
```

##### 4. 主程序实现

```go
func main() {
  // 解析命令行参数
  var configPath = flag.String("config", "config.yaml", "配置文件路径")
  var interactive = flag.Bool("interactive", true, "交互模式")
  var message = flag.String("message", "", "单次消息模式")
  flag.Parse()

  // 设置日志
  logger := log.New(os.Stdout, "[MCP-OpenAI] ", log.LstdFlags|log.Lshortfile)

  // 加载配置
  cfg, err := config.LoadConfig(*configPath)
  if err != nil {
    logger.Fatalf("加载配置失败: %v", err)
  }

  // 验证配置
  if err := cfg.Validate(); err != nil {
    logger.Fatalf("配置验证失败: %v", err)
  }

  // 创建聊天引擎
  engine, err := chat.NewChatEngine(cfg, logger)
  if err != nil {
    logger.Fatalf("创建聊天引擎失败: %v", err)
  }

  // 启动聊天引擎
  if err := engine.Start(); err != nil {
    logger.Fatalf("启动聊天引擎失败: %v", err)
  }

  // 设置信号处理
  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  // 根据模式运行
  if *message != "" {
    // 单次消息模式
    runSingleMessage(ctx, engine, *message, logger)
  } else if *interactive {
    // 交互模式
    runInteractiveMode(ctx, engine, logger)
  }
}
```

#### 配置文件示例

```yaml
openai:
  api_key: ""  # 从环境变量OPENAI_API_KEY获取
  base_url: "https://api.ppinfra.com/v3/openai"
  model: "deepseek/deepseek-v3-0324"
  temperature: 0.7
  max_tokens: 2000
  timeout: 30s

mcp:
  timeout: 30s
  servers:
    - name: "ssh-jsonrpc"
      command: "../mcp/go_jsonrpc/build/ssh-mcp-server"
      args: ["-config", "../mcp/go_jsonrpc/config.yaml"]
      description: "基于JSON-RPC实现的SSH MCP服务器"
      enabled: true
    - name: "ssh-sdk"
      command: "../mcp/go-sdk/build/ssh-mcp-server-sdk"
      args: ["-config", "../mcp/go-sdk/config.yaml"]
      description: "基于官方SDK实现的SSH MCP服务器"
      enabled: false

chat:
  max_history: 20
  system_prompt: "你是一个智能运维助手，可以通过MCP工具执行SSH命令和文件操作。请根据用户需求选择合适的工具来完成任务。在执行命令前，请先解释你将要做什么，然后执行相应的工具调用。"
  auto_save: true
  save_path: "./conversations"
  enable_mcp: true
  mcp_auto_call: true

log:
  level: "info"
  file: "/tmp/mcp-openai-chat.log"
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true 
```

#### 使用示例

##### 交互模式 && go-sdk

```bash
# 启动交互模式
export OPENAI_API_KEY="your_openai_api_key"
./build/mcp-openai-chat -config config.yaml -interactive
```

![image-20250905214133426](http://devyk.top/2022/202509052141082.png)

##### 单次消息模式 && 使用手撸 MCP

```bash
./build/mcp-openai-chat -message " 帮我看下这台设备的内存信息,IP:192.168.71.111;密码为:xxx"
```

![image-20250905214509973](http://devyk.top/2022/202509052145365.png)



#### 核心特性

1. **智能工具选择**：OpenAI模型自动选择合适的MCP工具
2. **多服务器支持**：可同时连接多个MCP服务器
3. **配置灵活**：支持切换不同的MCP实现（JSON-RPC vs SDK）
4. **对话记忆**：保持多轮对话的上下文
5. **错误处理**：完善的错误处理和重试机制
6. **日志记录**：详细的操作日志便于调试

通过这个 MCP 集成实现，我们成功地将 LLM 的强大语言理解能力与MCP的工具调用能力结合起来，创建了一个真正智能的运维助手。为后续我们开发真正的企业级 AI Agent 奠定了基础。



## 总结

通过本文的深入探讨，我们全面了解了Model Context Protocol从基础概念到实战应用的完整体系。MCP不仅仅是一个技术协议，更是 AI应用开发范式的重要转变。

### 核心价值回顾

**标准化的力量**：MCP通过统一的协议规范，解决了AI应用与外部系统集成的碎片化问题，让开发者可以专注于业务逻辑而非底层通信细节。

**生态系统效应**：随着越来越多的服务提供商和AI应用采用MCP标准，我们正在见证一个真正互联互通的AI生态系统的形成。

**企业级就绪**：从协议设计到安全机制，MCP充分考虑了企业级应用的需求，为大规模部署提供了可靠保障。

### 技术架构优势

MCP基于JSON-RPC 2.0的设计选择体现了深思熟虑：

- 有状态连接满足了AI对话的连续性需求
- 双向通信支持实时更新和通知
- 模块化架构确保了良好的扩展性

### 发展前景

随着AI Agent技术的快速发展，MCP有望成为连接AI智能体与数字世界的标准桥梁。我们可以预见：

1. **更丰富的生态**：更多的服务提供商将提供MCP服务器
2. **更强的互操作性**：不同AI应用之间的协作将变得更加容易
3. **更高的开发效率**：标准化将显著降低AI应用的开发成本

### 实践建议

对于准备采用MCP的开发者和企业，建议：

1. **从小规模开始**：选择一个具体的业务场景进行试点
2. **关注安全性**：在生产环境中务必实施适当的安全措施
3. **积极参与社区**：MCP生态系统的发展需要社区的共同参与

MCP的出现标志着AI应用开发进入了一个新的阶段。在这个阶段，AI不再是孤立的智能体，而是能够与整个数字生态系统深度集成的智能助手。这不仅为开发者带来了新的机遇，也为企业数字化转型提供了强有力的工具。

## 参考

- [MCP](https://github.com/modelcontextprotocol)
- [MCP DOCS](https://modelcontextprotocol.io/docs/getting-started/intro)
- [MCP 规范文档](https://modelcontextprotocol.io/specification/2025-03-26/basic)
- [JSON-RPC 2.0 规范（英文原版）](https://www.jsonrpc.org/specification)
- [JSON-RPC 2.0 规范（中文译版）](https://wiki.geekdream.com/Specification/json-rpc_2.0.html)
- [PPIO 大语言模型 API 文档](https://ppio.com/model-api/console)
- [Go SSH 包文档](https://pkg.go.dev/golang.org/x/crypto/ssh)