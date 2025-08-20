"""
第二章：图形API示例
对应文章：二、图形API：构建智能体的蓝图
"""

import sys
import os
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

class ProjectState(TypedDict):
    """项目状态"""
    requirements: str
    current_phase: str
    completed_tasks: list
    issues: list
    team_feedback: str

def analyze_requirements(state: ProjectState) -> ProjectState:
    """需求分析节点"""
    ColorfulLogger.step("正在进行需求分析...")
    
    requirements = state["requirements"]
    
    try:
        # 使用LLM分析需求
        llm = get_llm()
        analysis_prompt = f"请分析以下项目需求，提取关键要点：{requirements}"
        analysis = llm.invoke([HumanMessage(content=analysis_prompt)])
        
        ColorfulLogger.success(f"需求分析完成: {analysis.content}...")
        
        return {
            **state,
            "current_phase": "需求分析",
            "completed_tasks": state["completed_tasks"] + ["需求分析完成"]
        }
    except Exception as e:
        ColorfulLogger.error(f"需求分析失败: {e}")
        return {
            **state,
            "current_phase": "需求分析",
            "issues": state["issues"] + [f"需求分析失败: {str(e)}"]
        }

def create_plan(state: ProjectState) -> ProjectState:
    """制定计划节点"""
    ColorfulLogger.step("正在制定项目计划...")
    
    try:
        llm = get_llm()
        plan_prompt = f"基于需求分析结果，为以下项目制定详细计划：{state['requirements']}"
        plan = llm.invoke([HumanMessage(content=plan_prompt)])
        
        ColorfulLogger.success(f"项目计划制定完成: {plan.content}...")
        
        return {
            **state,
            "current_phase": "计划制定",
            "completed_tasks": state["completed_tasks"] + ["项目计划制定完成"]
        }
    except Exception as e:
        ColorfulLogger.error(f"计划制定失败: {e}")
        return {
            **state,
            "current_phase": "计划制定",
            "issues": state["issues"] + [f"计划制定失败: {str(e)}"]
        }

def graph_api_example():
    """2.1-2.3 图形API完整示例"""
    ColorfulLogger.info("=== 图形API：构建智能体的蓝图 ===")
    
    # 构建完整的工作流
    workflow = StateGraph(ProjectState)
    
    # 添加节点
    workflow.add_node("analyze", analyze_requirements)
    workflow.add_node("plan", create_plan)
    
    # 设置入口点
    workflow.set_entry_point("analyze")
    # 添加常规边
    workflow.add_edge("analyze", "plan")
    workflow.add_edge("plan", END)
    
    # 编译成可执行应用
    app = workflow.compile()
    
    # 执行工作流
    ColorfulLogger.info("开始执行 Go 语言的 web 框架开发工作流...")
    
    initial_state = {
        "requirements": "开发一个高可用的 Go 语言的 web 框架，需要支持大量并发用户",
        "current_phase": "",
        "completed_tasks": [],
        "issues": [],
        "team_feedback": ""
    }
    
    result = app.invoke(initial_state)
    
    ColorfulLogger.success("工作流执行完成！")
    return result

def visualization_example():
    """工作流可视化示例"""
    ColorfulLogger.info("=== 工作流可视化 ===")
    
    try:
        from utils import GraphVisualizer
        
        # 创建简单的工作流用于可视化
        workflow = StateGraph(ProjectState)
        workflow.add_node("analyze", analyze_requirements)
        workflow.add_node("plan", create_plan)
        workflow.set_entry_point("analyze")
        workflow.add_edge("analyze", "plan")
        workflow.add_edge("plan", END)
        
        # 生成Mermaid图表
        mermaid_chart = GraphVisualizer.generate_mermaid(
            workflow, 
            title="项目管理工作流"
        )
        
        ColorfulLogger.success("Mermaid图表生成成功")
        print(mermaid_chart)
        
    except Exception as e:
        ColorfulLogger.error(f"可视化生成失败: {e}")

def main():
    """主函数"""
    ColorfulLogger.header("第二章：图形API示例")
    
    try:
        # 1. 完整的图形API示例
        result = graph_api_example()
        
        # 打印结果
        ColorfulLogger.info("=== 执行结果 ===")
        ColorfulLogger.info(f"当前阶段: {result['current_phase']}")
        ColorfulLogger.info(f"已完成任务: {result['completed_tasks']}")
        ColorfulLogger.info(f"问题列表: {result['issues']}")
        ColorfulLogger.info(f"团队反馈: {result['team_feedback']}")
        
        # 2. 工作流可视化示例
        visualization_example()
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 