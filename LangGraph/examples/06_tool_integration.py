"""
第六章：工具集成示例
对应文章：六、工具集成：扩展智能体的能力边界
"""

import sys
import os
import json
import time
import asyncio
from typing import Annotated, List
from functools import wraps
from concurrent.futures import ThreadPoolExecutor
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END, MessagesState
from langgraph.prebuilt import ToolNode
from langchain_core.tools import tool
from langchain_core.messages import HumanMessage, AIMessage, ToolMessage
from config import get_llm
from utils import ColorfulLogger

class ToolState(TypedDict):
    """工具状态"""
    messages: list
    tool_results: dict
    execution_log: list

# ===== 6.1 工具定义 =====

@tool
def search_database(query: str) -> str:
    """搜索公司数据库中的信息
    
    Args:
        query: 搜索关键词或SQL查询语句
    
    Returns:
        JSON格式的搜索结果
    """
    # 模拟数据库查询
    mock_results = {
        "customer": [
            {"id": "C001", "name": "张三", "status": "active"},
            {"id": "C002", "name": "李四", "status": "inactive"}
        ],
        "product": [
            {"id": "P001", "name": "产品A", "price": 100},
            {"id": "P002", "name": "产品B", "price": 200}
        ]
    }
    
    # 简单关键词匹配 - 支持中文关键词
    keyword_mapping = {
        "客户": "customer",
        "customer": "customer", 
        "用户": "customer",
        "产品": "product",
        "product": "product",
        "商品": "product"
    }
    
    query_lower = query.lower()
    for cn_keyword, en_keyword in keyword_mapping.items():
        if cn_keyword in query_lower or en_keyword in query_lower:
            if en_keyword in mock_results:
                return json.dumps({
                    "status": "success",
                    "query": query,
                    "results": mock_results[en_keyword][:2]  # 限制返回结果
                }, ensure_ascii=False)
    
    return json.dumps({
        "status": "success",
        "query": query,
        "results": []
    }, ensure_ascii=False)

@tool  
def send_email(recipient: str, subject: str, body: str) -> str:
    """发送邮件通知
    
    Args:
        recipient: 收件人邮箱地址
        subject: 邮件主题
        body: 邮件正文内容
    
    Returns:
        发送状态信息
    """
    try:
        # 模拟邮件发送
        time.sleep(0.5)  # 模拟网络延迟
        
        # 简单验证邮箱格式
        if "@" not in recipient:
            return f"❌ 邮件发送失败: 无效的邮箱地址 {recipient}"
        
        ColorfulLogger.info(f"📧 模拟发送邮件: {recipient} - {subject}")
        return f"✅ 邮件已成功发送至 {recipient}"
        
    except Exception as e:
        return f"❌ 邮件发送失败: {str(e)}"

@tool
def get_weather(location: str) -> str:
    """获取指定地点的天气信息
    
    Args:
        location: 地点名称（如：北京、上海）
    
    Returns:
        天气信息的JSON字符串
    """
    # 模拟天气数据
    mock_weather = {
        "北京": {"temperature": "15°C", "condition": "晴", "humidity": "45%"},
        "上海": {"temperature": "18°C", "condition": "多云", "humidity": "60%"},
        "广州": {"temperature": "25°C", "condition": "雨", "humidity": "80%"}
    }
    
    weather_data = mock_weather.get(location, {
        "temperature": "20°C", 
        "condition": "未知", 
        "humidity": "50%"
    })
    
    return json.dumps({
        "location": location,
        "weather": weather_data,
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S")
    }, ensure_ascii=False)

# ===== 6.3 错误处理和重试机制 =====

@tool
def robust_api_call(endpoint: str, params: dict) -> str:
    """带重试机制的API调用工具
    
    Args:
        endpoint: API端点URL
        params: 请求参数
    
    Returns:
        API响应结果
    """
    max_retries = 3
    base_delay = 1
    
    for attempt in range(max_retries):
        try:
            # 模拟API调用
            ColorfulLogger.step(f"尝试第 {attempt + 1} 次API调用: {endpoint}")
            
            # 模拟随机失败（70%成功率）
            import random
            if random.random() < 0.7:  # 70%成功率
                response_data = {
                    "status": "success",
                    "data": f"API调用成功，参数: {params}",
                    "endpoint": endpoint
                }
                
                return json.dumps({
                    "success": True,
                    "data": response_data,
                    "attempt": attempt + 1
                }, ensure_ascii=False)
            else:
                raise Exception("模拟网络错误")
                
        except Exception as e:
            if attempt == max_retries - 1:
                return json.dumps({
                    "success": False,
                    "error": f"API调用失败，已重试{max_retries}次: {str(e)}",
                    "attempt": attempt + 1
                }, ensure_ascii=False)
            
            # 指数退避
            delay = base_delay * (2 ** attempt)
            ColorfulLogger.warning(f"第 {attempt + 1} 次尝试失败，{delay}秒后重试...")
            time.sleep(delay)
    
    return json.dumps({"success": False, "error": "未知错误"}, ensure_ascii=False)

# ===== 6.4 工具权限和安全控制 =====

def require_permissions(required_permissions: List[str]):
    """装饰器：要求特定权限才能使用工具"""
    def decorator(func):
        @wraps(func)
        def wrapper(*args, **kwargs):
            # 模拟用户权限检查
            user_permissions = get_current_user_permissions()
            
            for permission in required_permissions:
                if permission not in user_permissions:
                    return f"❌ 权限不足：需要 {permission} 权限"
            
            return func(*args, **kwargs)
        return wrapper
    return decorator

def get_current_user_permissions():
    """获取当前用户权限（模拟）"""
    # 在实际应用中，这应该从用户会话或数据库中获取
    return ["database.read", "email.send", "weather.read"]

@tool
@require_permissions(["database.read"])
def secure_database_search(query: str) -> str:
    """受权限保护的数据库搜索"""
    return search_database(query)

@tool  
@require_permissions(["email.send"])
def secure_send_email(recipient: str, subject: str, body: str) -> str:
    """受权限保护的邮件发送"""
    return send_email(recipient, subject, body)

# ===== 6.2 工具调用节点的实现 =====

def tool_calling_agent_example():
    """6.2 工具调用节点的实现示例"""
    ColorfulLogger.info("=== 6.2 工具调用智能体 ===")
    
    # 定义所有可用工具
    tools = [search_database, send_email, get_weather]
    
    # 创建绑定工具的模型
    try:
        model_with_tools = get_llm().bind_tools(tools)
    except Exception as e:
        ColorfulLogger.error(f"绑定工具失败: {e}")
        return None
    
    def call_model(state: MessagesState):
        """模型调用节点"""
        messages = state["messages"]
        try:
            response = model_with_tools.invoke(messages)
            return {"messages": [response]}
        except Exception as e:
            ColorfulLogger.error(f"模型调用失败: {e}")
            return {"messages": [AIMessage(content="抱歉，处理请求时出现错误")]}
    
    def should_continue(state: MessagesState) -> str:
        """判断是否需要调用工具"""
        messages = state["messages"]
        last_message = messages[-1]
        
        # 如果最后一条消息包含工具调用，则执行工具
        if hasattr(last_message, 'tool_calls') and last_message.tool_calls:
            return "tools"
        # 否则结束
        return "end"
    
    # 构建图
    workflow = StateGraph(MessagesState)
    
    # 添加节点
    workflow.add_node("agent", call_model)
    workflow.add_node("tools", ToolNode(tools))
    
    # 设置边
    workflow.set_entry_point("agent")
    workflow.add_conditional_edges(
        "agent",
        should_continue,
        {"tools": "tools", "end": END}
    )
    workflow.add_edge("tools", "agent")
    
    # 编译应用
    app = workflow.compile()
    
    # 测试工具调用
    test_messages = [
        "请搜索公司数据库中的客户信息",
        "查询北京的天气情况",
        "给admin@example.com发送一封测试邮件，主题是'系统测试'"
    ]
    
    for message in test_messages:
        ColorfulLogger.step(f"测试消息: {message}")
        try:
            result = app.invoke({
                "messages": [HumanMessage(content=message)]
            })
            
            final_message = result["messages"][-1]
            ColorfulLogger.success(f"回复: {final_message.content[:100]}...")
            
        except Exception as e:
            ColorfulLogger.error(f"测试失败: {e}")
    
    return app

# ===== 6.5 工具编排策略 =====

def sequential_tool_workflow_example():
    """6.5.1 顺序执行策略示例"""
    ColorfulLogger.info("\n=== 6.5.1 顺序执行策略 ===")
    
    class SequentialState(TypedDict):
        query: str
        location: str
        recipient: str
        results: dict
        steps_completed: list
    
    def sequential_tool_node(state: SequentialState) -> SequentialState:
        """顺序执行多个工具的工作流"""
        steps_completed = []
        results = {}
        
        try:
            # 步骤1：搜索相关信息
            ColorfulLogger.step("步骤1: 搜索数据库信息")
            search_result = search_database.invoke({"query": state["query"]})
            results["search"] = search_result
            steps_completed.append("搜索完成")
            
            # 步骤2：获取天气信息
            ColorfulLogger.step("步骤2: 获取天气信息")
            weather_info = get_weather.invoke({"location": state["location"]})
            results["weather"] = weather_info
            steps_completed.append("天气查询完成")
            
            # 步骤3：发送汇总邮件
            ColorfulLogger.step("步骤3: 发送汇总邮件")
            summary = f"搜索结果：{search_result[:50]}...\n天气信息：{weather_info[:50]}..."
            email_result = send_email.invoke({
                "recipient": state["recipient"], 
                "subject": "信息汇总", 
                "body": summary
            })
            results["email"] = email_result
            steps_completed.append("邮件发送完成")
            
        except Exception as e:
            ColorfulLogger.error(f"顺序执行失败: {e}")
            results["error"] = str(e)
        
        return {
            **state,
            "results": results,
            "steps_completed": steps_completed
        }
    
    # 构建工作流
    workflow = StateGraph(SequentialState)
    workflow.add_node("sequential", sequential_tool_node)
    workflow.set_entry_point("sequential")
    workflow.add_edge("sequential", END)
    
    app = workflow.compile()
    
    # 测试顺序执行
    initial_state = {
        "query": "customer",
        "location": "北京",
        "recipient": "manager@example.com",
        "results": {},
        "steps_completed": []
    }
    
    result = app.invoke(initial_state)
    
    ColorfulLogger.success("顺序执行完成:")
    for step in result["steps_completed"]:
        ColorfulLogger.info(f"  ✓ {step}")
    
    return result

async def parallel_tool_workflow_example():
    """6.5.2 并行执行策略示例"""
    ColorfulLogger.info("\n=== 6.5.2 并行执行策略 ===")
    
    class ParallelState(TypedDict):
        query: str
        location: str
        user_id: str
        results: dict
        execution_time: str
    
    async def parallel_tool_node(state: ParallelState) -> ParallelState:
        """并行执行多个工具的工作流"""
        
        async def async_search():
            """异步搜索包装器"""
            return search_database.invoke({"query": state["query"]})
        
        async def async_weather():
            """异步天气查询包装器"""
            return get_weather.invoke({"location": state["location"]})
        
        async def async_user_profile():
            """异步用户资料查询（模拟）"""
            await asyncio.sleep(0.5)  # 模拟异步操作
            return json.dumps({
                "user_id": state["user_id"],
                "profile": "模拟用户资料"
            }, ensure_ascii=False)
        
        try:
            start_time = time.time()
            
            # 并行执行多个工具
            ColorfulLogger.step("开始并行执行工具...")
            results = await asyncio.gather(
                async_search(),
                async_weather(),
                async_user_profile(),
                return_exceptions=True
            )
            
            end_time = time.time()
            execution_time = f"并行执行完成，耗时: {end_time - start_time:.2f}秒"
            
            # 处理结果
            search_result, weather_result, profile_result = results
            
            return {
                **state,
                "results": {
                    "search_result": search_result,
                    "weather_result": weather_result,
                    "profile_result": profile_result
                },
                "execution_time": execution_time
            }
            
        except Exception as e:
            ColorfulLogger.error(f"并行执行失败: {e}")
            return {
                **state,
                "results": {"error": str(e)},
                "execution_time": "执行失败"
            }
    
    # 构建工作流
    workflow = StateGraph(ParallelState)
    workflow.add_node("parallel", parallel_tool_node)
    workflow.set_entry_point("parallel")
    workflow.add_edge("parallel", END)
    
    app = workflow.compile()
    
    # 测试并行执行
    initial_state = {
        "query": "product",
        "location": "上海",
        "user_id": "USER001",
        "results": {},
        "execution_time": ""
    }
    
    result = app.invoke(initial_state)
    
    ColorfulLogger.success(f"并行执行结果: {result['execution_time']}")
    return result

def conditional_tool_workflow_example():
    """6.5.3 条件分支策略示例"""
    ColorfulLogger.info("\n=== 6.5.3 条件分支策略 ===")
    
    class ConditionalState(TypedDict):
        user_input: str
        request_type: str
        result: str
    
    def classify_request(user_input: str) -> str:
        """分类用户请求"""
        if any(keyword in user_input.lower() for keyword in ["搜索", "查询", "数据库"]):
            return "information_query"
        elif any(keyword in user_input.lower() for keyword in ["天气", "气温", "下雨"]):
            return "weather_request"
        elif any(keyword in user_input.lower() for keyword in ["邮件", "发送", "通知"]):
            return "email_task"
        else:
            return "unknown"
    
    def conditional_tool_node(state: ConditionalState) -> ConditionalState:
        """基于条件选择不同的工具执行路径"""
        
        request_type = classify_request(state["user_input"])
        
        if request_type == "information_query":
            # 信息查询路径
            ColorfulLogger.step("执行信息查询路径")
            result = search_database.invoke({"query": "customer"})
            
        elif request_type == "weather_request":
            # 天气查询路径
            ColorfulLogger.step("执行天气查询路径")
            result = get_weather.invoke({"location": "北京"})
            
        elif request_type == "email_task":
            # 邮件任务路径
            ColorfulLogger.step("执行邮件发送路径")
            result = send_email.invoke({
                "recipient": "user@example.com", 
                "subject": "自动回复", 
                "body": "这是一封自动生成的邮件"
            })
            
        else:
            # 默认路径
            ColorfulLogger.step("执行默认处理路径")
            result = "抱歉，我不确定如何处理这个请求"
        
        return {
            **state,
            "request_type": request_type,
            "result": result
        }
    
    # 构建工作流
    workflow = StateGraph(ConditionalState)
    workflow.add_node("conditional", conditional_tool_node)
    workflow.set_entry_point("conditional")
    workflow.add_edge("conditional", END)
    
    app = workflow.compile()
    
    # 测试不同类型的请求
    test_inputs = [
        "请搜索客户信息",
        "今天北京天气怎么样？",
        "发送邮件通知",
        "我想了解产品价格"
    ]
    
    for user_input in test_inputs:
        ColorfulLogger.step(f"测试输入: {user_input}")
        
        result = app.invoke({
            "user_input": user_input,
            "request_type": "",
            "result": ""
        })
        
        ColorfulLogger.info(f"  请求类型: {result['request_type']}")
        ColorfulLogger.info(f"  处理结果: {result['result'][:50]}...")

class ToolChain:
    """6.5.4 工具链：将多个工具组合成一个复杂的处理管道"""
    
    def __init__(self, tools: List):
        self.tools = tools
        self.results = []
    
    def execute(self, initial_input: str) -> str:
        """执行工具链"""
        current_input = initial_input
        
        for i, tool in enumerate(self.tools):
            try:
                ColorfulLogger.step(f"执行工具链步骤 {i+1}: {tool.name}")
                
                # 调用工具
                if tool.name == "search_database":
                    result = tool.invoke({"query": current_input})
                elif tool.name == "get_weather":
                    result = tool.invoke({"location": "北京"})  # 默认查询北京天气
                elif tool.name == "send_email":
                    result = tool.invoke({
                        "recipient": "chain@example.com",
                        "subject": "工具链结果",
                        "body": current_input
                    })
                else:
                    result = str(current_input)
                
                self.results.append({
                    "step": i + 1,
                    "tool": tool.name,
                    "input": current_input,
                    "output": result
                })
                
                # 将当前结果作为下一个工具的输入
                current_input = result
                
            except Exception as e:
                error_msg = f"工具链在第{i+1}步失败: {str(e)}"
                self.results.append({
                    "step": i + 1,
                    "tool": tool.name,
                    "error": error_msg
                })
                return error_msg
        
        return current_input

def tool_chain_example():
    """工具链示例"""
    ColorfulLogger.info("\n=== 6.5.4 工具链策略 ===")
    
    # 创建研究工具链
    tools = [search_database, get_weather, send_email]
    chain = ToolChain(tools)
    
    # 执行工具链
    result = chain.execute("customer")
    
    ColorfulLogger.success("工具链执行完成:")
    for step_result in chain.results:
        if "error" in step_result:
            ColorfulLogger.error(f"  步骤 {step_result['step']}: {step_result['error']}")
        else:
            ColorfulLogger.info(f"  步骤 {step_result['step']} ({step_result['tool']}): 成功")
    
    return chain.results

def main():
    """主函数"""
    ColorfulLogger.header("第六章：工具集成示例")
    
    try:
        # 1. 工具调用智能体示例
        tool_calling_agent_example()
        
        # 2. 顺序执行策略示例
        sequential_tool_workflow_example()
        
        # 3. 并行执行策略示例（需要异步环境）
        try:
            # 在Jupyter或支持顶级await的环境中运行
            import asyncio
            if hasattr(asyncio, '_get_running_loop') and asyncio._get_running_loop() is not None:
                # 已经在事件循环中
                ColorfulLogger.warning("跳过并行执行示例（已在事件循环中）")
            else:
                asyncio.run(parallel_tool_workflow_example())
        except Exception as e:
            ColorfulLogger.warning(f"并行执行示例跳过: {e}")
        
        # 4. 条件分支策略示例
        conditional_tool_workflow_example()
        
        # 5. 工具链策略示例
        tool_chain_example()
        
        ColorfulLogger.success("所有工具集成示例执行完成！")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 