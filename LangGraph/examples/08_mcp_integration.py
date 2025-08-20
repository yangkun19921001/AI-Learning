"""
第八章：MCP集成示例
对应文章：八、MCP集成：标准化的外部工具访问

本章演示如何使用 langchain_mcp_adapters.client.MultiServerMCPClient 
连接到真实的MCP服务器，并在LangGraph中配合LLM使用。
"""

import sys
import os
import json
import asyncio
from datetime import datetime
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict, Annotated
from typing import List
from langgraph.graph import StateGraph, END, MessagesState
from langgraph.graph.message import add_messages
from langchain_core.messages import HumanMessage, AIMessage, SystemMessage, BaseMessage
from langchain_core.tools import tool
from config import get_llm
from utils import ColorfulLogger

# 导入真实的MCP客户端（可选）
try:
    from langchain_mcp_adapters.client import MultiServerMCPClient
    MCP_AVAILABLE = True
    ColorfulLogger.success("✅ MCP客户端导入成功")
except ImportError as e:
    MCP_AVAILABLE = False
    ColorfulLogger.info("📚 MCP客户端不可用，将使用教学模拟版本")
    ColorfulLogger.info("💡 真实环境中请安装: uv add langchain-mcp-adapters")

# ===== 8.1 MCP 基础概念和配置 =====

def create_mcp_server_config(server_url: str) -> dict:
    """创建MCP服务器配置
    
    Args:
        server_url: MCP服务器的SSE URL
        
    Returns:
        dict: MCP服务器配置
    """
    return {
        "mcp_server": {
            "transport": "sse",
            "url": server_url
        }
    }

class MCPClientManager:
    """MCP客户端管理器"""
    
    def __init__(self):
        self.client = None
        self.available_tools = []
        self.server_config = {}
    
    async def connect_to_server(self, server_url: str) -> bool:
        """连接到MCP服务器
        
        Args:
            server_url: MCP服务器URL
            
        Returns:
            bool: 连接是否成功
        """
        if not MCP_AVAILABLE:
            ColorfulLogger.error("❌ MCP客户端不可用，无法连接")
            return False
        
        try:
            # 创建服务器配置
            self.server_config = create_mcp_server_config(server_url)
            ColorfulLogger.info(f"🔗 正在连接到MCP服务器: {server_url}")
            
            # 创建MCP客户端
            self.client = MultiServerMCPClient(self.server_config)
            
            # 获取可用工具
            self.available_tools = await self.client.get_tools()
            
            ColorfulLogger.success(f"✅ 成功连接到MCP服务器")
            ColorfulLogger.info(f"🛠️ 发现 {len(self.available_tools)} 个可用工具")
            
            # 显示工具信息
            for tool in self.available_tools:
                ColorfulLogger.info(f"  • {tool.name}: {tool.description}")
            
            return True
            
        except Exception as e:
            ColorfulLogger.error(f"❌ 连接MCP服务器失败: {e}")
            return False
    
    def get_tools(self):
        """获取可用工具列表"""
        return self.available_tools if self.available_tools else []
    
    async def call_tool(self, tool_name: str, arguments: dict):
        """调用MCP工具
        
        Args:
            tool_name: 工具名称
            arguments: 工具参数
            
        Returns:
            工具执行结果
        """
        if not self.client:
            raise Exception("MCP客户端未连接")
        
        # 查找工具
        target_tool = None
        for tool in self.available_tools:
            if tool.name == tool_name:
                target_tool = tool
                break
        
        if not target_tool:
            raise Exception(f"工具 {tool_name} 不存在")
        
        try:
            # 调用工具
            result = await target_tool.ainvoke(arguments)
            return result
        except Exception as e:
            raise Exception(f"工具调用失败: {e}")

# ===== 8.2 在LangGraph中使用MCP工具 =====

async def mcp_setup_example(server_url: str = None):
    """8.2.1 MCP服务器连接示例"""
    ColorfulLogger.header("第八章：MCP集成示例")
    ColorfulLogger.info("=== 8.2.1 MCP服务器连接 ===")
    
    # 使用默认的测试URL（如果没有提供）
    if not server_url:
        server_url = "http://localhost:3000/sse"
        ColorfulLogger.info(f"🔗 使用默认测试URL: {server_url}")
    
    # 创建MCP客户端管理器
    mcp_manager = MCPClientManager()
    
    # 尝试连接到MCP服务器
    success = await mcp_manager.connect_to_server(server_url)
    
    if not success:
        ColorfulLogger.warning("⚠️ 无法连接到MCP服务器，将使用模拟模式演示")
        return create_mock_mcp_manager()
    
    return mcp_manager

def create_mock_mcp_manager():
    """创建模拟的MCP管理器（当真实服务器不可用时）"""
    
    class MockTool:
        def __init__(self, name: str, description: str):
            self.name = name
            self.description = description
        
        async def ainvoke(self, arguments: dict):
            # 模拟工具执行
            if self.name == "get_system_info":
                return {
                    "status": "success",
                    "data": {
                        "hostname": "demo-server",
                        "os": "Ubuntu 22.04",
                        "cpu_usage": "15%",
                        "memory_usage": "68%",
                        "disk_usage": "42%"
                    }
                }
            elif self.name == "execute_command":
                command = arguments.get("command", "")
                return {
                    "status": "success",
                    "command": command,
                    "output": f"Mock output for command: {command}",
                    "exit_code": 0
                }
            else:
                return {"status": "success", "message": f"Mock result for {self.name}"}
    
    class MockMCPManager:
        def __init__(self):
            self.available_tools = [
                MockTool("get_system_info", "获取系统信息"),
                MockTool("execute_command", "执行系统命令"),
                MockTool("check_service_status", "检查服务状态")
            ]
        
        def get_tools(self):
            return self.available_tools
        
        async def call_tool(self, tool_name: str, arguments: dict):
            for tool in self.available_tools:
                if tool.name == tool_name:
                    return await tool.ainvoke(arguments)
            raise Exception(f"工具 {tool_name} 不存在")
    
    ColorfulLogger.info("🎭 创建模拟MCP管理器")
    return MockMCPManager()

async def mcp_tool_calling_example(mcp_manager):
    """8.2.2 在LangGraph节点中使用MCP工具"""
    ColorfulLogger.info("\n=== 8.2.2 MCP工具调用 ===")
    
    async def mcp_tool_node(state: MessagesState) -> MessagesState:
        """使用MCP工具的LangGraph节点"""
        
        if not state["messages"]:
            return state
        
        last_message = state["messages"][-1].content
        ColorfulLogger.step(f"处理用户请求: {last_message}")
        
        response_content = ""
        
        try:
            # 获取可用工具列表
            available_tools = mcp_manager.get_tools()
            tool_names = [tool.name for tool in available_tools]
            
            # 根据用户请求选择合适的MCP工具
            if ("执行命令" in last_message or "运行" in last_message or "设备ID" in last_message) and available_tools:
                # 查找远程执行工具
                remote_exec_tool = None
                
                for tool in available_tools:
                    if "remote_exec" in tool.name:
                        remote_exec_tool = tool
                        break
                
                if remote_exec_tool:
                    command = extract_command_from_message(last_message)
                    device_id = extract_device_id_from_message(last_message)
                    
                    ColorfulLogger.step(f"调用MCP工具: {remote_exec_tool.name}")
                    ColorfulLogger.info(f"设备ID: {device_id}, 命令: {command}")
                    
                    try:
                        # 调用远程执行工具（使用正确的参数名）
                        result = await mcp_manager.call_tool(remote_exec_tool.name, {
                            "machineId": device_id,
                            "script": command  # 使用script参数而不是command
                        })
                        response_content = f"🚀 远程执行结果:\n```\n{result}\n```"
                    except Exception as tool_error:
                        response_content = f"❌ 工具调用失败: {str(tool_error)}"
                else:
                    response_content = "未找到可用的远程执行工具"
                    
            elif "工具列表" in last_message or "可用工具" in last_message or "你能做什么" in last_message:
                # 显示真实的可用工具
                if available_tools:
                    response_content = "🛠️ 当前可用的MCP工具:\n\n"
                    for i, tool in enumerate(available_tools, 1):
                        response_content += f"{i}. **{tool.name}**\n"
                        response_content += f"   📝 描述: {tool.description}\n\n"
                    
                    response_content += "💡 使用示例:\n"
                    response_content += "- 说 '执行命令 ls -la' 来远程执行命令\n"
                    response_content += "- 说 '查看工具列表' 来显示所有可用工具"
                else:
                    response_content = "当前没有可用的MCP工具"
                    
            else:
                # 默认显示帮助信息
                if available_tools:
                    response_content = f"""🤖 我是MCP集成助手！当前连接了 {len(available_tools)} 个MCP工具：

"""
                    for tool in available_tools:
                        response_content += f"• {tool.name}\n"
                    
                    response_content += """
💡 您可以说：
- "执行命令 ls -la" - 远程执行系统命令
- "查看工具列表" - 显示详细的工具信息
- "你能做什么" - 显示帮助信息"""
                else:
                    response_content = "抱歉，当前没有可用的MCP工具"
            
        except Exception as e:
            ColorfulLogger.error(f"❌ MCP工具调用失败: {e}")
            response_content = f"抱歉，工具调用失败: {str(e)}"
        
        return {
            "messages": state["messages"] + [AIMessage(content=response_content)]
        }
    
    # 构建工作流
    workflow = StateGraph(MessagesState)
    workflow.add_node("mcp_tools", mcp_tool_node)
    workflow.set_entry_point("mcp_tools")
    workflow.add_edge("mcp_tools", END)
    
    app = workflow.compile()
    
    # 测试MCP工具调用
    test_messages = [
        "查看这台设备ID：6fa31edaac8bee6cc75cd8ae1bc03930 的磁盘信息",
        "查看工具列表",
        "设备ID：6fa31edaac8bee6cc75cd8ae1bc03930 执行命令 ls -la",
        "设备ID：6fa31edaac8bee6cc75cd8ae1bc03930 运行 ps aux | head -5"
    ]
    
    for message in test_messages:
        ColorfulLogger.step(f"测试消息: {message}")
        
        result = await app.ainvoke({
            "messages": [HumanMessage(content=message)]
        })
        
        response = result["messages"][-1].content
        ColorfulLogger.success(f"回复: {response[:200]}...")
        print("-" * 50)

# ===== 8.3 与LLM智能体集成 =====

async def mcp_llm_integration_example(mcp_manager):
    """8.3 MCP工具与LLM智能体集成 - 参考张高兴博客实现"""
    ColorfulLogger.info("\n=== 8.3 MCP工具与LLM智能体集成 ===")
    
    # 获取MCP工具
    mcp_tools = mcp_manager.get_tools()
    
    if not mcp_tools:
        ColorfulLogger.warning("⚠️ 没有可用的MCP工具")
        return
    
    ColorfulLogger.info(f"🛠️ 可用的MCP工具: {[tool.name for tool in mcp_tools]}")
    
    # 创建LLM并绑定工具
    llm = get_llm()
    llm_with_tools = llm.bind_tools(mcp_tools)
    
    # 定义状态 - 使用简单的TypedDict
    class State(TypedDict):
        messages: Annotated[List[BaseMessage], add_messages]
    
    # 定义智能体节点
    def agent_node(state: State):
        """智能体节点 - 分析用户请求并决定是否调用工具"""
        messages = state["messages"]
        
        # 添加系统提示
        system_prompt = """你是一个智能运维助手，可以通过MCP工具远程执行系统命令。

当用户询问系统状态、执行命令或查看信息时，请使用available tools来完成任务。

可用工具:
- remote_exec: 在远程设备上执行shell命令
  参数: machineId (设备ID), script (要执行的shell命令)

请根据用户需求智能选择和调用工具。如果用户提供了设备ID，请使用该ID；否则使用默认设备ID。
"""
        
        # 构建完整的消息列表
        full_messages = [SystemMessage(content=system_prompt)] + messages
        
        # 调用LLM
        response = llm_with_tools.invoke(full_messages)
        return {"messages": [response]}
    
    # 导入ToolNode
    from langgraph.prebuilt import ToolNode
    
    # 定义工具节点 - 使用ToolNode自动处理工具调用
    tool_node = ToolNode(mcp_tools)
    
    # 定义路由函数
    def should_continue(state: State):
        """判断是否需要继续调用工具"""
        last_message = state["messages"][-1]
        
        # 如果LLM返回了工具调用，则路由到工具节点
        if hasattr(last_message, 'tool_calls') and last_message.tool_calls:
            return "call_tool"
        return "end"
    
    # 构建工作流图
    workflow = StateGraph(State)
    
    # 添加节点
    workflow.add_node("agent", agent_node)
    workflow.add_node("call_tool", tool_node)
    
    # 设置入口点
    workflow.set_entry_point("agent")
    
    # 添加条件边
    workflow.add_conditional_edges(
        "agent",
        should_continue,
        {
            "call_tool": "call_tool",
            "end": END
        }
    )
    
    # 工具调用后返回智能体
    workflow.add_edge("call_tool", "agent")
    
    # 编译图
    app = workflow.compile()
    
    # 测试智能体 - 使用真实的工具调用场景
    test_queries = [
        "查看设备 6fa31edaac8bee6cc75cd8ae1bc03930 的系统负载情况",
        "在设备 6fa31edaac8bee6cc75cd8ae1bc03930 上执行 df -h 查看磁盘使用",
        "检查设备 6fa31edaac8bee6cc75cd8ae1bc03930 的内存使用情况 free -h"
    ]
    
    for query in test_queries:
        ColorfulLogger.step(f"🤖 智能体测试: {query}")
        
        try:
            # 调用智能体
            result = await app.ainvoke({"messages": [HumanMessage(content=query)]})
            
            # 获取最终回复
            final_message = result["messages"][-1]
            
            if hasattr(final_message, 'content') and final_message.content:
                ColorfulLogger.success(f"✅ 智能体回复: {final_message.content[:300]}...")
            else:
                ColorfulLogger.info(f"📋 工具调用结果: {str(final_message)[:300]}...")
                
        except Exception as e:
            ColorfulLogger.error(f"❌ 智能体调用失败: {str(e)}")
        
        print("=" * 60)

# ===== 辅助函数 =====

def extract_command_from_message(message: str) -> str:
    """从消息中提取命令"""
    # 简单的命令提取逻辑
    if "执行命令" in message:
        parts = message.split("执行命令")
        if len(parts) > 1:
            return parts[1].strip()
    
    if "运行" in message:
        parts = message.split("运行")
        if len(parts) > 1:
            return parts[1].strip()
    
    # 常见命令模式
    common_commands = ["ls -la", "ps aux", "df -h", "top", "free", "uptime", "whoami"]
    for cmd in common_commands:
        if cmd in message.lower():
            return cmd
    
    return "ls -la"

def extract_device_id_from_message(message: str) -> str:
    """从消息中提取设备ID"""
    import re
    
    # 查找设备ID模式
    device_id_pattern = r'设备ID[：:]\s*([a-f0-9]{32})'
    match = re.search(device_id_pattern, message)
    
    if match:
        return match.group(1)
    
    # 查找32位十六进制字符串
    hex_pattern = r'([a-f0-9]{32})'
    match = re.search(hex_pattern, message)
    
    if match:
        return match.group(1)
    
    # 默认设备ID
    return "6fa31edaac8bee6cc75cd8ae1bc03930"

def format_system_info_response(result: dict) -> str:
    """格式化系统信息响应"""
    if isinstance(result, dict) and "data" in result:
        data = result["data"]
        response = "🖥️ 系统信息:\n"
        for key, value in data.items():
            response += f"  • {key}: {value}\n"
        return response
    return str(result)

def format_command_response(result: dict) -> str:
    """格式化命令执行响应"""
    if isinstance(result, dict):
        command = result.get("command", "unknown")
        output = result.get("output", "")
        exit_code = result.get("exit_code", 0)
        
        response = f"⚡ 命令: {command}\n"
        response += f"📤 退出码: {exit_code}\n"
        response += f"📋 输出:\n{output}"
        return response
    return str(result)

def format_service_response(result: dict) -> str:
    """格式化服务状态响应"""
    if isinstance(result, dict):
        return f"🔍 服务状态检查结果:\n{json.dumps(result, ensure_ascii=False, indent=2)}"
    return str(result)

# ===== 主函数 =====

async def main():
    """第八章主函数"""
    ColorfulLogger.header("第八章：MCP集成示例")
    
    try:
        # 8.2.1 MCP服务器连接
        mcp_manager = await mcp_setup_example("http://10.1.16.4:8000/mcp/sse")
        
        # 8.2.2 MCP工具调用
        await mcp_tool_calling_example(mcp_manager)
        
        # 8.3 与LLM智能体集成
        await mcp_llm_integration_example(mcp_manager)
        
        ColorfulLogger.success("✅ 第8章示例运行完成")
        
        # 总结
        ColorfulLogger.info("\n=== MCP集成开发要点总结 ===")
        ColorfulLogger.info("✓ MCP协议提供标准化的外部工具访问接口")
        ColorfulLogger.info("✓ 支持多种传输方式：stdio、sse等")
        ColorfulLogger.info("✓ 可以轻松集成到LangGraph工作流中")
        ColorfulLogger.info("✓ 与LLM结合实现智能工具选择和调用")
        ColorfulLogger.info("✓ 提供安全、可控的外部系统访问能力")
        
    except Exception as e:
        ColorfulLogger.error(f"❌ 示例运行失败: {e}")

if __name__ == "__main__":
    asyncio.run(main()) 