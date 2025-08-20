"""
第九章：ReAct智能体示例
对应文章：九、ReAct智能体：推理与行动的结合
"""

import sys
import os
import asyncio
import time
from datetime import datetime
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.prebuilt import create_react_agent
from langgraph.graph import StateGraph, END, MessagesState
from langgraph.checkpoint.memory import InMemorySaver
from langchain_core.tools import tool
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

# 导入真实的MCP客户端
try:
    from langchain_mcp_adapters.client import MultiServerMCPClient
    MCP_AVAILABLE = True
    ColorfulLogger.success("✅ MCP客户端导入成功")
except ImportError as e:
    MCP_AVAILABLE = False
    ColorfulLogger.info("📚 MCP客户端不可用，跳过MCP集成示例")
    ColorfulLogger.info("💡 真实环境中请安装: uv add langchain-mcp-adapters")

class CustomAgentState(TypedDict):
    """自定义智能体状态"""
    messages: list
    remaining_steps: int  # ReAct agent 需要的字段
    intermediate_steps: list  # ReAct agent 需要的字段
    user_profile: dict
    conversation_context: dict
    task_progress: dict
    execution_stats: dict

# ===== 9.2 基础工具定义 =====

@tool
def calculator(expression: str) -> str:
    """计算数学表达式
    
    Args:
        expression: 要计算的数学表达式，如 "2 + 3 * 4"
    
    Returns:
        计算结果
    """
    try:
        # 安全的数学表达式计算
        allowed_chars = set('0123456789+-*/.() ')
        if not all(c in allowed_chars for c in expression):
            return "错误：表达式包含不允许的字符"
        
        result = eval(expression)
        return f"计算结果: {result}"
    except Exception as e:
        return f"计算错误: {str(e)}"

@tool
def search_web(query: str) -> str:
    """搜索网络信息
    
    Args:
        query: 搜索关键词
        
    Returns:
        搜索结果摘要
    """
    # 模拟网络搜索
    mock_results = {
        "天气": "今天天气晴朗，温度适宜，适合户外活动。",
        "新闻": "最新科技新闻：人工智能技术在各个领域取得突破性进展。",
        "股票": "股市今日表现良好，科技股领涨。",
        "python": "Python是一种高级编程语言，广泛用于数据科学和AI开发。"
    }
    
    # 简单关键词匹配
    for keyword, result in mock_results.items():
        if keyword in query.lower():
            return f"搜索 '{query}' 的结果: {result}"
    
    return f"搜索 '{query}' 的结果: 找到相关信息，请查看详细内容。"

@tool
def get_current_time() -> str:
    """获取当前时间
    
    Returns:
        当前的日期和时间
    """
    return datetime.now().strftime("%Y-%m-%d %H:%M:%S")

@tool
def save_note(note: str) -> str:
    """保存笔记
    
    Args:
        note: 要保存的笔记内容
        
    Returns:
        保存状态
    """
    # 模拟保存笔记
    timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
    filename = f"note_{timestamp}.txt"
    
    ColorfulLogger.info(f"📝 保存笔记到: {filename}")
    return f"笔记已保存到 {filename}"

@tool
def send_reminder(message: str, delay_minutes: int = 5) -> str:
    """设置提醒
    
    Args:
        message: 提醒内容
        delay_minutes: 延迟分钟数
        
    Returns:
        提醒设置状态
    """
    future_time = datetime.now()
    return f"已设置提醒：'{message}'，将在 {delay_minutes} 分钟后提醒"

# ===== 9.2 使用create_react_agent快速创建智能体 =====

def create_business_assistant():
    """创建商业助手ReAct智能体"""
    ColorfulLogger.info("=== 9.2 创建商业助手ReAct智能体 ===")
    
    # 定义可用工具
    tools = [calculator, search_web, get_current_time, save_note, send_reminder]
    
    # 创建模型
    try:
        model = get_llm()
        model.temperature = 0.1  # 设置较低的温度以确保稳定性
    except Exception as e:
        ColorfulLogger.error(f"创建模型失败: {e}")
        return None
    
    # 创建ReAct智能体
    try:
        agent = create_react_agent(
            model=model,
            tools=tools,
            state_modifier="你是一个专业的商业助手。请帮助用户解决各种商业问题，包括计算、信息查询、时间管理等。在使用工具时，请清楚地解释你的推理过程。"
        )
        
        ColorfulLogger.success("✅ 成功创建ReAct智能体")
        return agent
        
    except Exception as e:
        ColorfulLogger.error(f"创建ReAct智能体失败: {e}")
        return None

def test_business_assistant():
    """测试商业助手"""
    ColorfulLogger.info("=== 测试商业助手功能 ===")
    
    agent = create_business_assistant()
    if not agent:
        return
    
    # 测试用例
    test_cases = [
        "请帮我计算一下如果我每月投资1000元，年化收益率为8%，10年后总共能积累多少资金？",
        "现在几点了？请帮我搜索一下今天的天气情况。",
        "请帮我保存一个笔记：明天下午3点开会讨论项目进度。",
        "计算 25 * 30 + 100 的结果"
    ]
    
    for i, test_case in enumerate(test_cases, 1):
        ColorfulLogger.step(f"测试案例 {i}: {test_case}")
        
        config = {"configurable": {"thread_id": f"business_test_{i}"}}
        
        try:
            result = agent.invoke({
                "messages": [HumanMessage(content=test_case)]
            }, config)
            
            final_message = result["messages"][-1]
            ColorfulLogger.success(f"回复: {final_message.content[:200]}...")
            
        except Exception as e:
            ColorfulLogger.error(f"测试失败: {e}")
        
        print("-" * 50)

# ===== 9.3 自定义ReAct智能体的状态 =====

def create_enhanced_react_agent():
    """创建增强的ReAct智能体"""
    ColorfulLogger.info("\n=== 9.3 增强ReAct智能体（自定义状态）===")
    
    # 定义包含状态访问能力的工具
    @tool
    def update_user_profile(user_info: str) -> str:
        """更新用户档案信息
        
        Args:
            user_info: 用户信息描述
            
        Returns:
            更新状态
        """
        ColorfulLogger.info(f"🔄 更新用户档案: {user_info}")
        return f"用户档案已更新: {user_info}"
    
    @tool
    def track_task_progress(task: str, status: str) -> str:
        """跟踪任务进度
        
        Args:
            task: 任务名称
            status: 任务状态
            
        Returns:
            进度更新确认
        """
        ColorfulLogger.info(f"📊 任务进度更新: {task} -> {status}")
        return f"任务 '{task}' 状态更新为: {status}"
    
    @tool
    def analyze_conversation_context(topic: str) -> str:
        """分析对话上下文
        
        Args:
            topic: 要分析的话题
            
        Returns:
            分析结果
        """
        return f"已分析话题 '{topic}' 的上下文，发现相关模式和趋势"
    
    tools = [
        calculator, search_web, get_current_time, 
        update_user_profile, track_task_progress, analyze_conversation_context
    ]
    
    try:
        model = get_llm()
        
        # 创建增强的智能体（使用默认状态结构以避免兼容性问题）
        agent = create_react_agent(
            model=model,
            tools=tools,
            state_modifier="你是一个智能的个人助手。请记住用户的偏好和任务进度，提供个性化的帮助。在处理任务时，请更新相关的状态信息。"
        )
        
        ColorfulLogger.success("✅ 成功创建增强ReAct智能体")
        return agent
        
    except Exception as e:
        ColorfulLogger.error(f"创建增强智能体失败: {e}")
        return None

def test_enhanced_agent():
    """测试增强智能体"""
    ColorfulLogger.info("=== 测试增强智能体功能 ===")
    
    agent = create_enhanced_react_agent()
    if not agent:
        return
    
    test_cases = [
        "我是一名软件工程师，请更新我的用户档案",
        "请帮我跟踪'学习LangGraph'这个任务，状态设为'进行中'",
        "分析一下我们关于人工智能的对话上下文",
        "现在是什么时间？请计算距离今天晚上8点还有多少小时"
    ]
    
    for i, test_case in enumerate(test_cases, 1):
        ColorfulLogger.step(f"增强测试 {i}: {test_case}")
        
        config = {"configurable": {"thread_id": f"enhanced_test_{i}"}}
        
        try:
            result = agent.invoke({
                "messages": [HumanMessage(content=test_case)],
                "user_profile": {},
                "conversation_context": {},
                "task_progress": {},
                "execution_stats": {}
            }, config)
            
            final_message = result["messages"][-1]
            ColorfulLogger.success(f"回复: {final_message.content[:200]}...")
            
        except Exception as e:
            ColorfulLogger.error(f"增强测试失败: {e}")

# ===== 9.4 ReAct智能体的提示词优化 =====

def create_specialized_react_agent(domain: str):
    """创建专业领域的ReAct智能体"""
    ColorfulLogger.info(f"\n=== 9.4 专业领域ReAct智能体: {domain} ===")
    
    # 根据领域定制提示词
    domain_prompts = {
        "finance": """
        你是一位专业的金融分析师助手。在处理金融相关问题时：
        1. 始终考虑风险因素和市场波动性
        2. 提供基于数据的分析，避免直接的投资建议
        3. 使用计算器工具进行精确的数值计算
        4. 在需要最新市场信息时使用搜索工具
        5. 强调免责声明：所有信息仅供参考，不构成投资建议
        """,
        
        "technical": """
        你是一位技术专家助手。在解决技术问题时：
        1. 先理解问题的技术背景和要求
        2. 分步骤分析问题并制定解决方案
        3. 使用搜索工具获取最新的技术信息
        4. 提供详细的技术解释和实施步骤
        5. 建议最佳实践和潜在的注意事项
        """,
        
        "education": """
        你是一位教育助手。在教学过程中：
        1. 用简单易懂的语言解释复杂概念
        2. 提供实际例子和类比来帮助理解
        3. 鼓励互动和提问
        4. 使用计算器帮助数学相关的学习
        5. 保存重要的学习笔记
        """,
        
        "general": """
        你是一位通用智能助手。请：
        1. 仔细分析用户需求
        2. 选择合适的工具来获取信息或执行任务
        3. 提供清晰、有用的回答
        4. 在不确定时主动寻求澄清
        """
    }
    
    prompt = domain_prompts.get(domain, domain_prompts["general"])
    
    tools = [calculator, search_web, get_current_time, save_note]
    
    try:
        model = get_llm()
        
        agent = create_react_agent(
            model=model,
            tools=tools,
            state_modifier=prompt
        )
        
        ColorfulLogger.success(f"✅ 成功创建 {domain} 领域智能体")
        return agent
        
    except Exception as e:
        ColorfulLogger.error(f"创建 {domain} 智能体失败: {e}")
        return None

def test_specialized_agents():
    """测试专业领域智能体"""
    ColorfulLogger.info("=== 测试专业领域智能体 ===")
    
    # 测试金融领域智能体
    finance_agent = create_specialized_react_agent("finance")
    if finance_agent:
        ColorfulLogger.step("测试金融智能体")
        config = {"configurable": {"thread_id": "finance_test"}}
        
        try:
            result = finance_agent.invoke({
                "messages": [HumanMessage(content="请帮我分析一下投资10万元在股市和债券市场的风险收益比较")]
            }, config)
            
            ColorfulLogger.success(f"金融智能体回复: {result['messages'][-1].content[:150]}...")
        except Exception as e:
            ColorfulLogger.error(f"金融智能体测试失败: {e}")
    
    # 测试技术领域智能体
    tech_agent = create_specialized_react_agent("technical")
    if tech_agent:
        ColorfulLogger.step("测试技术智能体")
        config = {"configurable": {"thread_id": "tech_test"}}
        
        try:
            result = tech_agent.invoke({
                "messages": [HumanMessage(content="如何优化Python代码的性能？请给出具体的建议和示例")]
            }, config)
            
            ColorfulLogger.success(f"技术智能体回复: {result['messages'][-1].content[:150]}...")
        except Exception as e:
            ColorfulLogger.error(f"技术智能体测试失败: {e}")

# ===== 9.5 ReAct智能体的执行控制 =====

def create_controlled_react_agent():
    """创建可控制的ReAct智能体"""
    ColorfulLogger.info("\n=== 9.5 可控制ReAct智能体 ===")
    
    tools = [calculator, search_web, get_current_time]
    
    try:
        model = get_llm()
        
        # 创建带有执行限制的智能体
        agent = create_react_agent(
            model=model,
            tools=tools,
            state_modifier="你是一个高效的助手，请在最少的步骤内完成任务。如果任务复杂，请优先使用最相关的工具。"
        )
        
        ColorfulLogger.success("✅ 成功创建可控制智能体")
        return agent
        
    except Exception as e:
        ColorfulLogger.error(f"创建可控制智能体失败: {e}")
        return None

async def run_with_timeout(agent, inputs, config, timeout=30):
    """带超时控制的智能体执行"""
    ColorfulLogger.info(f"⏱️  执行智能体（超时: {timeout}秒）")
    
    try:
        # 设置超时
        result = await asyncio.wait_for(
            asyncio.coroutine(lambda: agent.invoke(inputs, config))(),
            timeout=timeout
        )
        return result
    except asyncio.TimeoutError:
        ColorfulLogger.error("⏰ 智能体执行超时")
        return {"error": "智能体执行超时"}
    except Exception as e:
        ColorfulLogger.error(f"❌ 执行错误: {str(e)}")
        return {"error": f"执行错误: {str(e)}"}

def test_controlled_execution():
    """测试控制执行"""
    ColorfulLogger.info("=== 测试控制执行 ===")
    
    agent = create_controlled_react_agent()
    if not agent:
        return
    
    # 测试正常执行
    ColorfulLogger.step("测试正常执行")
    config = {"configurable": {"thread_id": "controlled_test"}}
    
    try:
        result = agent.invoke({
            "messages": [HumanMessage(content="现在几点？请计算100乘以50等于多少？")]
        }, config)
        
        ColorfulLogger.success(f"正常执行结果: {result['messages'][-1].content[:150]}...")
        
    except Exception as e:
        ColorfulLogger.error(f"控制执行测试失败: {e}")

# ===== 9.6 ReAct智能体的流式执行 =====

def stream_react_agent_execution():
    """流式执行ReAct智能体"""
    ColorfulLogger.info("\n=== 9.6 流式执行ReAct智能体 ===")
    
    agent = create_business_assistant()
    if not agent:
        return
    
    inputs = {
        "messages": [HumanMessage(content="请帮我分析一下当前时间，然后计算如果投资5万元年收益率6%，5年后的本息总和")]
    }
    
    config = {"configurable": {"thread_id": "stream_session"}}
    
    ColorfulLogger.info("🤖 智能体开始流式工作...")
    
    try:
        # 流式执行，观察推理过程
        step_count = 0
        for chunk in agent.stream(inputs, config, stream_mode="values"):
            step_count += 1
            messages = chunk.get("messages", [])
            
            if messages:
                last_message = messages[-1]
                
                # 显示智能体的思考过程
                if hasattr(last_message, 'content'):
                    content = last_message.content
                    
                    if hasattr(last_message, 'type'):
                        if last_message.type == "ai":
                            if "思考" in content or "Thought" in content:
                                ColorfulLogger.info(f"🤔 步骤{step_count} - 思考: {content[:100]}...")
                            elif "行动" in content or "Action" in content:
                                ColorfulLogger.info(f"🔧 步骤{step_count} - 行动: {content[:100]}...")
                            else:
                                ColorfulLogger.info(f"💭 步骤{step_count} - 回应: {content[:100]}...")
                        
                        elif last_message.type == "tool":
                            ColorfulLogger.info(f"🛠️  步骤{step_count} - 工具结果: {content[:100]}...")
            
            # 避免输出过多
            if step_count > 10:
                ColorfulLogger.warning("流式输出步骤过多，停止显示")
                break
        
        ColorfulLogger.success("🎉 流式执行完成")
        
    except Exception as e:
        ColorfulLogger.error(f"流式执行失败: {e}")

# ===== 9.7 ReAct Agent + 真实MCP集成 =====

class MCPConnectionManager:
    """MCP连接管理器"""
    
    def __init__(self, sse_url: str):
        self.sse_url = sse_url
        self.client = None
        self.tools = []
    
    async def connect(self):
        """连接到MCP服务器"""
        if not MCP_AVAILABLE:
            raise Exception("MCP客户端不可用，请安装 langchain-mcp-adapters")
        
        ColorfulLogger.info(f"🔗 正在连接到MCP服务器: {self.sse_url}")
        
        try:
            # 创建MCP客户端配置（适配新版本API）
            server_config = {
                "mcp_server": {
                    "transport": "sse",
                    "url": self.sse_url
                }
            }
            
            # 连接到MCP服务器（使用新的API方式）
            self.client = MultiServerMCPClient(server_config)
            
            # 直接获取工具，不使用上下文管理器
            self.tools = await self.client.get_tools()
            
            ColorfulLogger.success("✅ 成功连接到MCP服务器")
            ColorfulLogger.info(f"🛠️ 发现 {len(self.tools)} 个可用工具")
            
            for tool in self.tools:
                ColorfulLogger.info(f"  • {tool.name}: {tool.description}")
            
            return True
            
        except Exception as e:
            ColorfulLogger.error(f"❌ 连接MCP服务器失败: {e}")
            raise Exception(f"MCP服务器连接失败: {e}") from e
    
    async def disconnect(self):
        """断开MCP连接"""
        if self.client:
            try:
                # 新版本API可能不需要显式断开连接
                ColorfulLogger.info("🔌 MCP连接已清理")
            except Exception as e:
                ColorfulLogger.warning(f"⚠️ 断开连接时出现警告: {e}")
    
    def get_tools(self):
        """获取MCP工具列表"""
        return self.tools

async def create_react_agent_with_mcp(sse_url: str):
    """创建集成真实MCP的ReAct Agent"""
    ColorfulLogger.info("\n=== 9.7 ReAct Agent + 真实MCP集成 ===")
    
    if not MCP_AVAILABLE:
        ColorfulLogger.warning("⚠️ MCP客户端不可用，跳过MCP集成示例")
        return None, None
    
    # 连接MCP服务器
    mcp_manager = MCPConnectionManager(sse_url)
    await mcp_manager.connect()
    
    # 获取MCP工具
    mcp_tools = mcp_manager.get_tools()
    
    if not mcp_tools:
        ColorfulLogger.warning("⚠️ 没有可用的MCP工具")
        await mcp_manager.disconnect()
        return None, None
    
    # 创建LLM
    llm = get_llm()
    
    # 创建系统提示
    system_prompt = """你是一个智能运维助手，具备以下能力：

🤖 **核心能力**:
- 通过MCP工具远程执行系统命令
- 分析系统状态和性能指标
- 提供运维建议和解决方案

🛠️ **可用工具**:
- remote_exec: 在远程设备上执行shell命令
  参数: machineId (设备ID), script (shell命令)

💡 **工作方式**:
1. 仔细分析用户需求
2. 制定执行计划
3. 使用合适的工具获取信息
4. 分析结果并提供专业建议

请按照ReAct模式工作：观察 → 思考 → 行动 → 观察，直到完成任务。
"""
    
    # 使用create_react_agent创建ReAct代理
    ColorfulLogger.info("🚀 创建ReAct Agent...")
    react_agent = create_react_agent(
        llm,
        mcp_tools,
        state_modifier=system_prompt
    )
    
    ColorfulLogger.success("✅ ReAct Agent创建成功")
    
    return react_agent, mcp_manager

async def test_react_agent_with_mcp(sse_url: str = None):
    """测试ReAct Agent + MCP集成"""
    ColorfulLogger.info("=== 测试ReAct Agent + MCP集成 ===")
    
    if not MCP_AVAILABLE:
        ColorfulLogger.info("⏭️ 跳过MCP集成测试（MCP客户端不可用）")
        return
    
    # 使用默认URL或用户提供的URL
    if not sse_url:
        sse_url = "http://10.1.16.4:8000/mcp/sse"  # 默认URL
    
    ColorfulLogger.info(f"🌐 使用MCP服务器: {sse_url}")
    
    try:
        # 创建ReAct Agent
        agent, mcp_manager = await create_react_agent_with_mcp(sse_url)
        
        if not agent or not mcp_manager:
            ColorfulLogger.warning("⚠️ 无法创建MCP集成的ReAct Agent")
            return
        
        # 定义测试场景
        test_scenarios = [
            {
                "name": "系统健康检查",
                "query": "请帮我全面检查设备 6fa31edaac8bee6cc75cd8ae1bc03930 的系统健康状况，包括CPU、内存、磁盘使用情况"
            },
            {
                "name": "性能分析",
                "query": "设备 6fa31edaac8bee6cc75cd8ae1bc03930 运行缓慢，请帮我分析可能的原因并提供优化建议"
            },
            {
                "name": "故障排查",
                "query": "设备 6fa31edaac8bee6cc75cd8ae1bc03930 上的某个服务可能有问题，请帮我检查系统日志和运行状态"
            }
        ]
        
        for i, scenario in enumerate(test_scenarios, 1):
            ColorfulLogger.step(f"🧪 测试场景 {i}: {scenario['name']}")
            ColorfulLogger.info(f"📝 用户请求: {scenario['query']}")
            
            try:
                # 调用ReAct Agent
                result = await agent.ainvoke({
                    "messages": [HumanMessage(content=scenario['query'])]
                })
                
                # 获取最终回复
                final_message = result["messages"][-1]
                
                if hasattr(final_message, 'content') and final_message.content:
                    # 显示ReAct推理过程
                    ColorfulLogger.info("\n🧠 ReAct推理过程:")
                    thought_count = 0
                    action_count = 0
                    
                    for j, msg in enumerate(result["messages"]):
                        if hasattr(msg, 'content') and msg.content:
                            content = msg.content
                            if "Thought:" in content or "思考" in content:
                                thought_count += 1
                                ColorfulLogger.step(f"  💭 思考{thought_count}: {content[:100]}...")
                            elif "Action:" in content or "行动" in content:
                                action_count += 1
                                ColorfulLogger.step(f"  ⚡ 行动{action_count}: {content[:100]}...")
                            elif "Observation:" in content or "观察" in content:
                                ColorfulLogger.step(f"  👁️ 观察: {content[:100]}...")
                    
                    # 显示最终结果
                    ColorfulLogger.success(f"✅ 最终回复: {final_message.content[:300]}...")
                else:
                    ColorfulLogger.info(f"📋 Agent结果: {str(final_message)[:300]}...")
                    
            except Exception as e:
                ColorfulLogger.error(f"❌ 测试场景失败: {str(e)}")
            
            print("=" * 80)
            
            # 添加延迟避免服务器过载
            await asyncio.sleep(2)
        
        # 清理资源
        await mcp_manager.disconnect()
        
        ColorfulLogger.success("✅ ReAct Agent + MCP集成测试完成")
        
    except Exception as e:
        ColorfulLogger.error(f"❌ MCP集成测试失败: {e}")

async def main():
    """主函数"""
    ColorfulLogger.header("第九章：ReAct智能体示例")
    
    try:
        # 1. 基础ReAct智能体测试
        test_business_assistant()
        
        # 2. 增强ReAct智能体测试
        test_enhanced_agent()
        
        # 3. 专业领域智能体测试
        test_specialized_agents()
        
        # 4. 控制执行测试
        test_controlled_execution()
        
        # 5. 流式执行测试
        stream_react_agent_execution()
        
        # 6. ReAct Agent + MCP集成测试
        sse_url = None
        if len(sys.argv) > 1:
            sse_url = sys.argv[1]
            ColorfulLogger.info(f"🌐 使用用户提供的SSE URL: {sse_url}")
        
        await test_react_agent_with_mcp(sse_url)
        
        ColorfulLogger.success("所有ReAct智能体示例执行完成！")
        
        # 总结
        ColorfulLogger.info("\n=== ReAct智能体开发要点总结 ===")
        ColorfulLogger.info("✓ 工具设计：确保工具描述清晰，参数定义明确")
        ColorfulLogger.info("✓ 提示词优化：根据应用领域定制合适的系统提示")
        ColorfulLogger.info("✓ 执行控制：设置合理的迭代次数和执行时间限制")
        ColorfulLogger.info("✓ 错误处理：为工具调用实现适当的错误处理机制")
        ColorfulLogger.info("✓ 状态管理：合理设计状态结构，避免信息冗余")
        ColorfulLogger.info("✓ 性能监控：跟踪智能体的执行效率和资源使用情况")
        ColorfulLogger.info("✓ MCP集成：通过标准化协议访问外部工具和服务")
        ColorfulLogger.info("✓ ReAct模式：结合推理与行动，适合复杂的多步骤任务")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    import sys
    asyncio.run(main()) 