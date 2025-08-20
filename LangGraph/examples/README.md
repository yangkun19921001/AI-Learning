# LangGraph企业级Agent开发实战示例

本目录包含与文章《企业级Agent开发实战(一)LangGraph快速入门》对应的完整示例代码。

## 📁 目录结构

```
examples/
├── README.md                    # 本文件
├── run_all_examples.py          # 运行所有示例的主入口
├── 01_basic_concepts.py         # 第一章：基础概念
├── 02_graph_api.py              # 第二章：图形API
├── 03_streaming.py              # 第三章：流式传输
├── 04_memory_system.py          # 第四章：记忆系统
├── 05_model_integration.py      # 第五章：模型集成
├── 06_tool_integration.py       # 第六章：工具集成
├── 07_human_in_the_loop.py      # 第七章：人机交互
├── 08_mcp_integration.py        # 第八章：MCP集成
└── 09_react_agent.py            # 第九章：ReAct智能体
```

## 🚀 快速开始

### 1. 环境准备

确保已安装Python 3.8+和必要的依赖包：

```bash
# 安装依赖
pip install langgraph langchain-core langchain-openai langchain-anthropic typing-extensions

# 或使用项目的依赖文件
pip install -r requirements.txt  # 如果存在
```

### 2. 配置API密钥

复制环境变量示例文件并填入你的API密钥：

```bash
cp ../config.env.example ../.env
```

编辑 `.env` 文件，设置以下变量之一：

```bash
# OpenAI配置
OPENAI_API_KEY=your_openai_api_key
OPENAI_BASE_URL=https://api.openai.com/v1  # 可选
OPENAI_MODEL=gpt-4o-mini

# 或 Anthropic配置
ANTHROPIC_API_KEY=your_anthropic_api_key
ANTHROPIC_MODEL=claude-3-haiku-20240307

# 其他配置
DEFAULT_LLM_PROVIDER=openai  # 或 anthropic
LOG_LEVEL=INFO
```

### 3. 运行示例

#### 运行所有示例
```bash
python run_all_examples.py
```

#### 运行特定章节
```bash
python run_all_examples.py 1    # 运行第1章
python run_all_examples.py 5    # 运行第5章
```

#### 查看帮助
```bash
python run_all_examples.py --help
```

## 📚 章节说明

### 第1章：基础概念 (`01_basic_concepts.py`)
- **内容**：LangGraph核心概念、状态管理、简单工作流
- **关键点**：StateGraph、节点、边、状态传递
- **示例**：基础聊天机器人、状态更新

### 第2章：图形API (`02_graph_api.py`)
- **内容**：节点设计、边的控制、条件逻辑
- **关键点**：条件边、工作流编译、项目管理示例
- **示例**：项目管理工作流、审批流程

### 第3章：流式传输 (`03_streaming.py`)
- **内容**：多种流式模式、实时反馈
- **关键点**：values、updates、messages、custom、debug模式
- **示例**：实时状态更新、进度显示、多模式组合

### 第4章：记忆系统 (`04_memory_system.py`)
- **内容**：短期记忆、长期记忆、存储策略
- **关键点**：Checkpointer、Store、记忆类型、环境配置
- **示例**：对话历史管理、用户偏好存储、记忆检索

### 第5章：模型集成 (`05_model_integration.py`)
- **内容**：多模型管理、性能监控、回退策略
- **关键点**：ModelManager、自定义配置、监控统计
- **示例**：模型选择、性能追踪、故障转移

### 第6章：工具集成 (`06_tool_integration.py`)
- **内容**：工具定义、调用机制、编排策略
- **关键点**：@tool装饰器、ToolNode、权限控制、错误处理
- **示例**：数据库搜索、邮件发送、并行执行、工具链

### 第7章：人机交互 (`07_human_in_the_loop.py`)
- **内容**：中断机制、审批流程、协作界面
- **关键点**：interrupt()、权限检查、审计日志
- **示例**：敏感操作审批、动态中断、合规报告

### 第8章：MCP集成 (`08_mcp_integration.py`)
- **内容**：MCP协议、外部系统连接、安全管理
- **关键点**：MCPClient、工具发现、资源访问、权限控制
- **示例**：CRM集成、HR系统、动态工具注册

### 第9章：ReAct智能体 (`09_react_agent.py`)
- **内容**：ReAct模式、智能体创建、执行控制
- **关键点**：create_react_agent、工具调用、流式执行
- **示例**：商业助手、专业领域智能体、执行监控

## 🛠️ 开发指南

### 自定义示例

你可以基于现有示例创建自己的智能体：

1. **复制示例文件**：
   ```bash
   cp 01_basic_concepts.py my_custom_agent.py
   ```

2. **修改配置**：
   - 更新状态定义
   - 自定义工具函数
   - 调整工作流逻辑

3. **测试运行**：
   ```bash
   python my_custom_agent.py
   ```

### 调试技巧

1. **启用调试日志**：
   ```python
   import logging
   logging.basicConfig(level=logging.DEBUG)
   ```

2. **使用流式输出观察执行过程**：
   ```python
   for chunk in app.stream(input_data, stream_mode="debug"):
       print(chunk)
   ```

3. **检查状态变化**：
   ```python
   result = app.invoke(input_data)
   print(f"最终状态: {result}")
   ```

### 性能优化

1. **模型选择**：根据任务复杂度选择合适的模型
2. **并行执行**：使用asyncio并行调用工具
3. **缓存机制**：缓存频繁访问的数据
4. **内存管理**：及时清理不需要的状态数据

## 🔧 故障排除

### 常见问题

1. **API密钥错误**：
   ```
   错误：缺少 OPENAI API Key
   解决：检查 .env 文件中的API密钥设置
   ```

2. **模块导入失败**：
   ```
   错误：No module named 'langgraph'
   解决：pip install langgraph
   ```

3. **中文编码问题**：
   ```python
   # 确保使用UTF-8编码
   import os
   os.environ['PYTHONIOENCODING'] = 'utf-8'
   ```

### 日志分析

查看详细的执行日志：

```bash
# 设置日志级别
export LOG_LEVEL=DEBUG
python run_all_examples.py 1
```

日志文件位置：`../logs/langgraph.log`

## 📊 性能基准

以下是各章节示例的典型性能表现（仅供参考）：

| 章节 | 平均执行时间 | 内存使用 | API调用次数 |
|------|-------------|----------|-------------|
| 第1章 | 2-5秒 | < 100MB | 1-2次 |
| 第3章 | 3-8秒 | < 150MB | 2-4次 |
| 第6章 | 5-15秒 | < 200MB | 3-8次 |
| 第9章 | 10-30秒 | < 300MB | 5-15次 |

*注：实际性能取决于网络环境、模型选择和任务复杂度*

## 🤝 贡献指南

欢迎提交改进建议：

1. Fork 项目
2. 创建功能分支：`git checkout -b feature/new-example`
3. 提交更改：`git commit -am 'Add new example'`
4. 推送分支：`git push origin feature/new-example`
5. 提交Pull Request

## 📄 许可证

本项目示例代码采用MIT许可证，详见LICENSE文件。

## 📞 支持

如果遇到问题，可以：

1. 查看[LangGraph官方文档](https://langchain-ai.github.io/langgraph/)
2. 搜索相关GitHub Issues
3. 在项目中提交新的Issue

---

**祝你学习愉快！🎉** 