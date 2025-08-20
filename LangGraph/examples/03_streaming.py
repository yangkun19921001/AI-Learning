"""
第三章：流式传输示例
对应文章：三、流式传输：实时感知智能体思考
"""

import sys
import os
import time
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END
from langgraph.config import get_stream_writer
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

class StreamingState(TypedDict):
    """流式传输状态"""
    messages: list
    step_count: int
    progress: dict

def values_mode_example():
    """3.3 values模式示例"""
    ColorfulLogger.info("=== values模式：观察完整状态 ===")
    
    class SimpleState(TypedDict):
        counter: int
        message: str
    
    def increment_step(state: SimpleState) -> SimpleState:
        return {
            "counter": state["counter"] + 1,
            "message": f"步骤 {state['counter'] + 1} 完成"
        }
    
    def final_step(state: SimpleState) -> SimpleState:
        return {
            "counter": state["counter"] + 1,
            "message": "所有步骤已完成"
        }
    
    # 构建简单工作流
    workflow = StateGraph(SimpleState)
    workflow.add_node("step1", increment_step)
    workflow.add_node("step2", increment_step) 
    workflow.add_node("final", final_step)
    workflow.set_entry_point("step1")
    workflow.add_edge("step1", "step2")
    workflow.add_edge("step2", "final")
    workflow.add_edge("final", END)
    
    app = workflow.compile()
    
    initial_state = {"counter": 0, "message": "开始"}
    
    ColorfulLogger.info("开始流式获取完整状态...")
    for chunk in app.stream(initial_state, stream_mode="values"):
        ColorfulLogger.step(f"📊 当前完整状态: {chunk}")
        time.sleep(0.5)  # 模拟处理时间

def updates_mode_example():
    """3.4 updates模式示例"""
    ColorfulLogger.info("\n=== updates模式：追踪状态变化 ===")
    
    class TaskState(TypedDict):
        tasks: list
        completed: list
        current_task: str
    
    def start_task(state: TaskState) -> TaskState:
        new_task = "数据收集"
        return {
            "tasks": state["tasks"],
            "completed": state["completed"],
            "current_task": new_task
        }
    
    def process_task(state: TaskState) -> TaskState:
        return {
            "tasks": state["tasks"],
            "completed": state["completed"] + [state["current_task"]],
            "current_task": "任务处理中"
        }
    
    def finish_task(state: TaskState) -> TaskState:
        return {
            "tasks": state["tasks"],
            "completed": state["completed"],
            "current_task": "全部完成"
        }
    
    workflow = StateGraph(TaskState)
    workflow.add_node("start", start_task)
    workflow.add_node("process", process_task)
    workflow.add_node("finish", finish_task)
    workflow.set_entry_point("start")
    workflow.add_edge("start", "process")
    workflow.add_edge("process", "finish")
    workflow.add_edge("finish", END)
    
    app = workflow.compile()
    
    initial_state = {
        "tasks": ["数据收集", "数据处理", "报告生成"],
        "completed": [],
        "current_task": ""
    }
    
    ColorfulLogger.info("开始流式获取状态更新...")
    for chunk in app.stream(initial_state, stream_mode="updates"):
        if chunk:
            node_name = list(chunk.keys())[0]
            update_data = chunk[node_name]
            ColorfulLogger.step(f"🔄 节点 {node_name} 更新: {update_data}")
            time.sleep(0.5)

def messages_mode_example():
    """3.5 messages模式示例"""
    ColorfulLogger.info("\n=== messages模式：实时显示AI生成内容 ===")
    
    class ChatState(TypedDict):
        messages: list
    
    def chat_node(state: ChatState) -> ChatState:
        try:
            llm = get_llm()
            user_message = state["messages"][-1]
            
            ColorfulLogger.info(f"用户: {user_message.content}")
            
            # 调用LLM
            ai_response = llm.invoke(state["messages"])
            
            return {
                "messages": state["messages"] + [ai_response]
            }
        except Exception as e:
            ColorfulLogger.error(f"LLM调用失败: {e}")
            return {
                "messages": state["messages"] + [AIMessage(content="抱歉，出现了错误")]
            }
    
    workflow = StateGraph(ChatState)
    workflow.add_node("chat", chat_node)
    workflow.set_entry_point("chat")
    workflow.add_edge("chat", END)
    
    app = workflow.compile()
    
    initial_state = {
        "messages": [HumanMessage(content="请简单介绍一下流式传输的优势")]
    }
    
    ColorfulLogger.info("开始流式获取LLM生成内容...")
    try:
        for message_chunk, metadata in app.stream(initial_state, stream_mode="messages"):
            if hasattr(message_chunk, 'content') and message_chunk.content:
                print(message_chunk.content, end="", flush=True)
        print()  # 换行
        ColorfulLogger.success("消息流传输完成")
    except Exception as e:
        ColorfulLogger.error(f"messages模式示例失败: {e}")
        # 使用普通模式作为备选
        result = app.invoke(initial_state)
        ColorfulLogger.info(f"AI回复: {result['messages'][-1].content}")

def custom_mode_example():
    """3.6 custom模式示例"""
    ColorfulLogger.info("\n=== custom模式：传输自定义数据 ===")
    
    class ProcessState(TypedDict):
        input_data: str
        result: str
    
    def progress_node(state: ProcessState) -> ProcessState:
        """展示进度的节点"""
        writer = get_stream_writer()
        
        # 模拟长时间处理过程
        for i in range(1, 6):
            progress_msg = f"处理进度: {i*20}%"
            writer(progress_msg)
            ColorfulLogger.step(f"📈 {progress_msg}")
            time.sleep(0.8)
        
        return {
            "input_data": state["input_data"],
            "result": "处理完成"
        }
    
    workflow = StateGraph(ProcessState)
    workflow.add_node("process", progress_node)
    workflow.set_entry_point("process")
    workflow.add_edge("process", END)
    
    app = workflow.compile()
    
    initial_state = {
        "input_data": "用户数据",
        "result": ""
    }
    
    ColorfulLogger.info("开始流式获取自定义进度数据...")
    for chunk in app.stream(initial_state, stream_mode="custom"):
        ColorfulLogger.info(f"📈 进度更新: {chunk}")

def multi_mode_example():
    """3.7 多模式组合示例"""
    ColorfulLogger.info("\n=== 多模式组合：全方位监控 ===")
    
    class MultiState(TypedDict):
        step: int
        message: str
        status: str
    
    def multi_step_node(state: MultiState) -> MultiState:
        writer = get_stream_writer()
        
        # 发送自定义进度信息
        writer(f"开始执行步骤 {state['step'] + 1}")
        
        new_step = state["step"] + 1
        return {
            "step": new_step,
            "message": f"步骤 {new_step} 已完成",
            "status": "运行中" if new_step < 3 else "完成"
        }
    
    workflow = StateGraph(MultiState)
    workflow.add_node("multi_step", multi_step_node)
    workflow.set_entry_point("multi_step")
    workflow.add_edge("multi_step", END)
    
    app = workflow.compile()
    
    initial_state = {
        "step": 0,
        "message": "初始化",
        "status": "开始"
    }
    
    ColorfulLogger.info("开始多模式流式传输...")
    for mode, chunk in app.stream(
        initial_state, 
        stream_mode=["updates", "custom"]
    ):
        if mode == "updates":
            ColorfulLogger.step(f"🔄 状态更新: {chunk}")
        elif mode == "custom":
            ColorfulLogger.info(f"📈 自定义数据: {chunk}")
        time.sleep(0.5)

def main():
    """主函数"""
    ColorfulLogger.header("第三章：流式传输示例")
    
    try:
        # 1. values模式示例
        values_mode_example()
        
        # 2. updates模式示例  
        updates_mode_example()
        
        # 3. messages模式示例
        messages_mode_example()
        
        # 4. custom模式示例
        custom_mode_example()
        
        # 5. 多模式组合示例
        multi_mode_example()
        
        ColorfulLogger.success("所有流式传输示例执行完成！")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 