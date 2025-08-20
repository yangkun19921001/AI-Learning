# LangGraph 深度学习教程

> 从入门到精通的 LangGraph 实战教程，包含完整代码示例和渐进式学习路径

## 🎯 教程特色

- **🔧 支持自定义LLM配置**: OpenAI、Anthropic、Azure、自定义API端点
- **📚 渐进式学习路径**: 从简单到复杂，循序渐进
- **💻 完整代码示例**: 每个概念都有详细的实战代码
- **🎨 彩色日志输出**: 清晰的执行状态显示
- **💾 持久化存储**: Checkpoint、记忆管理、会话恢复

## 🚀 快速开始

### 1. 环境准备

```bash
# 克隆项目（或直接使用现有目录）
cd AI-Learning/LangGraph

# 安装依赖（当前使用的是 python 3.13）
uv sync

```

### 2. 配置设置

```bash
# 复制配置文件
cp config.env.example .env

# 编辑配置文件，填入你的API信息
vim .env
```

配置示例：
```env
# 选择你的LLM提供商
DEFAULT_LLM_PROVIDER=openai

# OpenAI配置
OPENAI_API_KEY=your_api_key_here
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_MODEL=gpt-4o-mini

# 或使用其他提供商...
```

### 3. 运行教程

```bash

uv run examples/run_tutorial.py 1

```


## 教程系列文章

 - [企业级Agent开发实战(一)LangGraph快速入门.md](https://github.com/yangkun19921001/AI-Learning/blob/main/LangGraph/docs/%E4%BC%81%E4%B8%9A%E7%BA%A7Agent%E5%BC%80%E5%8F%91%E5%AE%9E%E6%88%98(%E4%B8%80)LangGraph%E5%BF%AB%E9%80%9F%E5%85%A5%E9%97%A8.md)

 - [企业级Agent开发实战(二)MCP协议分析及实战MCPServer.md]()

 - [企业级Agent开发实战(三)运维分析Agent开发.md]()


## 📜 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

感谢 LangChain 团队创建了如此优秀的框架，感谢所有为开源社区做出贡献的开发者们！

---


**开始你的 LangGraph 学习之旅吧！** 🚀 