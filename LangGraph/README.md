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





## 📜 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 🙏 致谢

感谢 LangChain 团队创建了如此优秀的框架，感谢所有为开源社区做出贡献的开发者们！

---

## 📞 联系我们

如果你在学习过程中遇到问题或有任何建议，欢迎：

- 📧 提交 Issue
- 💬 参与讨论
- 🌟 Star 项目以示支持

**开始你的 LangGraph 学习之旅吧！** 🚀 