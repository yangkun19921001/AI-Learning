"""
第一章：LangGraph基础概念示例
对应文章：一、什么是LangGraph？核心概念解析
"""

import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

class AgentState(TypedDict):
    """基础智能体状态"""
    task: str
    steps_completed: list
    current_step: str
    result: str

class ProjectState(TypedDict):
    """项目状态示例"""
    requirements: str      # 项目需求
    current_phase: str    # 当前阶段
    completed_tasks: list # 已完成任务
    issues: list         # 遇到的问题
    team_feedback: str   # 团队反馈

def basic_state_example():
    """1.2 状态管理示例"""
    ColorfulLogger.info("=== 1.2 状态管理：智能体的'记忆系统' ===")
    
    # 创建初始状态
    initial_state = AgentState(
        task="开发一个客户关系管理系统",
        steps_completed=[],
        current_step="开始",
        result=""
    )
    
    ColorfulLogger.success(f"初始状态创建成功：{initial_state}")
    return initial_state

def simple_workflow_example():
    """创建简单的工作流示例"""
    ColorfulLogger.info("=== 创建第一个LangGraph应用 ===")
    
    class SimpleState(TypedDict):
        messages: list
        step_count: int

    def process_message(state: SimpleState) -> SimpleState:
        """处理消息节点"""
        ColorfulLogger.step("处理用户消息...")
        
        try:
            llm = get_llm()  # 使用配置的LLM
            
            user_message = state["messages"][-1]
            ColorfulLogger.info(f"用户输入: {user_message.content}")
            
            ai_response = llm.invoke(state["messages"])
            ColorfulLogger.success(f"AI回复: {ai_response.content}")
            
            return {
                "messages": state["messages"] + [ai_response],
                "step_count": state["step_count"] + 1
            }
        except Exception as e:
            ColorfulLogger.error(f"处理消息失败: {e}")
            return {
                "messages": state["messages"] + [AIMessage(content="抱歉，处理消息时出现错误")],
                "step_count": state["step_count"] + 1
            }

    # 构建图
    workflow = StateGraph(SimpleState)
    workflow.add_node("process", process_message)
    workflow.set_entry_point("process")
    workflow.add_edge("process", END)

    # 编译应用
    app = workflow.compile()
    
    # 运行示例
    result = app.invoke({
        "messages": [HumanMessage(content="你好！请介绍一下LangGraph的核心概念。")],
        "step_count": 0
    })
    
    ColorfulLogger.success("工作流执行完成！")
    return result

def main():
    """主函数"""
    ColorfulLogger.header("第一章：LangGraph基础概念示例")
    
    try:
        # 1. 状态管理示例
        basic_state_example()
        
        # 2. 简单工作流示例
        result = simple_workflow_example()
        
        ColorfulLogger.info(f"最终结果: 处理了 {result['step_count']} 步")
        ColorfulLogger.info(f"对话历史: {len(result['messages'])} 条消息")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 