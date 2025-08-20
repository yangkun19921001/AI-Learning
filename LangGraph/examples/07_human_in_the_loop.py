"""
第七章：人机交互示例
对应文章：七、人机交互：在关键时刻引入人类智慧
"""

import sys
import os
import time
from datetime import datetime
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END, MessagesState
from langgraph.types import interrupt
from langgraph.checkpoint.memory import InMemorySaver
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

class InteractionState(TypedDict):
    """人机交互状态"""
    messages: list
    operation_type: str
    risk_level: str
    human_feedback: dict
    approved: bool
    reviewer: str

class InteractionAuditor:
    """人机交互审计器"""
    
    def __init__(self):
        self.audit_logs = []
    
    def log_interrupt(self, session_id: str, node_name: str, reason: str):
        """记录中断事件"""
        audit_record = {
            "event_type": "interrupt",
            "session_id": session_id,
            "node_name": node_name,
            "reason": reason,
            "timestamp": datetime.now().isoformat(),
            "user_context": f"session_{session_id}"
        }
        
        self.audit_logs.append(audit_record)
        ColorfulLogger.info(f"📝 审计记录: 中断事件 - {reason}")
    
    def log_human_decision(self, session_id: str, decision: dict):
        """记录人工决策"""
        audit_record = {
            "event_type": "human_decision",
            "session_id": session_id,
            "decision": decision,
            "timestamp": datetime.now().isoformat(),
            "reviewer": decision.get("reviewer_id", "unknown"),
            "justification": decision.get("justification", "")
        }
        
        self.audit_logs.append(audit_record)
        ColorfulLogger.info(f"📝 审计记录: 人工决策 - {decision.get('action', 'unknown')}")
    
    def generate_compliance_report(self):
        """生成合规报告"""
        interrupts = [log for log in self.audit_logs if log["event_type"] == "interrupt"]
        decisions = [log for log in self.audit_logs if log["event_type"] == "human_decision"]
        
        approved_decisions = [d for d in decisions if d["decision"].get("approved", False)]
        
        report = {
            "total_interrupts": len(interrupts),
            "total_decisions": len(decisions),
            "approval_rate": len(approved_decisions) / len(decisions) if decisions else 0,
            "compliance_violations": []  # 在实际应用中检测违规
        }
        
        return report

class HumanCollaborationInterface:
    """人机协作界面"""
    
    def __init__(self):
        self.pending_approvals = {}
        self.auditor = InteractionAuditor()
    
    def request_approval(self, session_id: str, context: dict):
        """请求人工审批"""
        approval_request = {
            "session_id": session_id,
            "timestamp": datetime.now().isoformat(),
            "context": context,
            "status": "pending"
        }
        
        self.pending_approvals[session_id] = approval_request
        
        ColorfulLogger.warning(f"🚨 请求人工审批 - 会话ID: {session_id}")
        ColorfulLogger.info(f"   操作: {context.get('operation', '未知')}")
        ColorfulLogger.info(f"   风险级别: {context.get('risk_level', '未知')}")
        
        # 记录审计日志
        self.auditor.log_interrupt(session_id, "approval_request", "需要人工审批")
        
        return approval_request
    
    def provide_approval(self, session_id: str, decision: dict):
        """提供审批决策（模拟人工审批）"""
        if session_id in self.pending_approvals:
            self.pending_approvals[session_id].update({
                "decision": decision,
                "status": "completed",
                "completed_at": datetime.now().isoformat()
            })
            
            # 记录审计日志
            self.auditor.log_human_decision(session_id, decision)
            
            ColorfulLogger.success(f"✅ 审批完成 - {'通过' if decision.get('approved') else '拒绝'}")
            return True
        
        return False
    
    def simulate_human_approval(self, session_id: str, auto_approve: bool = True):
        """模拟人工审批过程"""
        if session_id in self.pending_approvals:
            # 模拟审批延迟
            time.sleep(1)
            
            # 模拟人工决策
            decision = {
                "approved": auto_approve,
                "reviewer_id": "admin_001",
                "justification": "自动化测试审批" if auto_approve else "测试拒绝",
                "timestamp": datetime.now().isoformat()
            }
            
            return self.provide_approval(session_id, decision)
        
        return False

# ===== 7.1 中断机制示例 =====

def basic_interrupt_example():
    """7.1 基础中断机制示例"""
    ColorfulLogger.info("=== 7.1 基础中断机制 ===")
    
    def sensitive_operation_node(state: InteractionState) -> InteractionState:
        """敏感操作节点"""
        operation_type = state["operation_type"]
        
        ColorfulLogger.step(f"准备执行操作: {operation_type}")
        
        # 评估风险级别
        risk_level = assess_risk(operation_type)
        
        if risk_level == "high":
            ColorfulLogger.warning("检测到高风险操作，触发中断")
            interrupt("需要人工确认高风险操作")
        
        # 继续执行操作
        ColorfulLogger.success(f"执行操作: {operation_type}")
        
        return {
            **state,
            "risk_level": risk_level,
            "approved": True
        }
    
    def assess_risk(operation_type: str) -> str:
        """评估操作风险级别"""
        high_risk_operations = ["删除数据", "修改权限", "系统重启"]
        medium_risk_operations = ["更新配置", "发送通知"]
        
        if operation_type in high_risk_operations:
            return "high"
        elif operation_type in medium_risk_operations:
            return "medium"
        else:
            return "low"
    
    # 构建工作流
    workflow = StateGraph(InteractionState)
    workflow.add_node("operation", sensitive_operation_node)
    workflow.set_entry_point("operation")
    workflow.add_edge("operation", END)
    
    # 使用中断功能编译
    checkpointer = InMemorySaver()
    app = workflow.compile(
        checkpointer=checkpointer,
        interrupt_before=["operation"]  # 在敏感操作前中断
    )
    
    # 测试不同风险级别的操作
    test_operations = ["查看报表", "更新配置", "删除数据"]
    
    for operation in test_operations:
        ColorfulLogger.step(f"测试操作: {operation}")
        
        config = {"configurable": {"thread_id": f"test_{operation.replace(' ', '_')}"}}
        
        try:
            result = app.invoke({
                "messages": [],
                "operation_type": operation,
                "risk_level": "",
                "human_feedback": {},
                "approved": False,
                "reviewer": ""
            }, config)
            
            ColorfulLogger.success(f"操作 '{operation}' 正常完成")
            
        except Exception as e:
            if "interrupt" in str(e).lower():
                ColorfulLogger.warning(f"操作 '{operation}' 被中断，需要人工审批")
            else:
                ColorfulLogger.error(f"操作失败: {e}")

def content_based_interrupt_example():
    """7.2.1 基于内容的中断示例"""
    ColorfulLogger.info("\n=== 7.2.1 基于内容的中断 ===")
    
    def content_based_interrupt_node(state: MessagesState) -> MessagesState:
        """基于消息内容决定是否中断"""
        if not state["messages"]:
            return state
            
        last_message = state["messages"][-1].content.lower()
        
        # 检测敏感关键词
        sensitive_keywords = ["删除", "删库", "重置密码", "转账", "删除用户"]
        
        for keyword in sensitive_keywords:
            if keyword in last_message:
                ColorfulLogger.warning(f"检测到敏感关键词: {keyword}")
                interrupt(f"检测到敏感操作关键词: {keyword}，需要人工确认")
        
        # 正常处理
        try:
            llm = get_llm()
            response = llm.invoke(state["messages"])
            return {"messages": state["messages"] + [response]}
        except Exception as e:
            ColorfulLogger.error(f"处理失败: {e}")
            return {"messages": state["messages"] + [AIMessage(content="处理请求时出现错误")]}
    
    # 构建工作流
    workflow = StateGraph(MessagesState)
    workflow.add_node("process", content_based_interrupt_node)
    workflow.set_entry_point("process")
    workflow.add_edge("process", END)
    
    checkpointer = InMemorySaver()
    app = workflow.compile(checkpointer=checkpointer)
    
    # 测试不同类型的消息
    test_messages = [
        "请帮我查看系统状态",
        "我需要重置密码",
        "帮我删除所有用户数据",
        "请发送报告给管理员"
    ]
    
    for message in test_messages:
        ColorfulLogger.step(f"测试消息: {message}")
        
        config = {"configurable": {"thread_id": f"content_test_{hash(message) % 1000}"}}
        
        try:
            result = app.invoke({
                "messages": [HumanMessage(content=message)]
            }, config)
            
            ColorfulLogger.success("消息正常处理")
            
        except Exception as e:
            if "interrupt" in str(e).lower():
                ColorfulLogger.warning("消息触发中断，需要人工确认")
            else:
                ColorfulLogger.error(f"处理失败: {e}")

def permission_based_interrupt_example():
    """7.2.2 基于用户权限的中断示例"""
    ColorfulLogger.info("\n=== 7.2.2 基于用户权限的中断 ===")
    
    def get_user_role(config):
        """获取用户角色（模拟）"""
        return config.get("configurable", {}).get("user_role", "user")
    
    def extract_operation(message_content: str) -> str:
        """提取操作类型"""
        if any(word in message_content.lower() for word in ["删除", "删库"]):
            return "admin_operations"
        elif any(word in message_content.lower() for word in ["查看", "搜索"]):
            return "read_operations"
        else:
            return "general_operations"
    
    def permission_based_interrupt_node(state: MessagesState, config) -> MessagesState:
        """基于用户权限决定是否需要额外确认"""
        if not state["messages"]:
            return state
            
        user_role = get_user_role(config)
        operation = extract_operation(state["messages"][-1].content)
        
        ColorfulLogger.info(f"用户角色: {user_role}, 操作类型: {operation}")
        
        # 普通用户执行管理员操作需要审批
        if user_role == "user" and operation == "admin_operations":
            ColorfulLogger.warning("普通用户尝试执行管理员操作")
            interrupt("普通用户执行管理员操作，需要管理员审批")
        
        # 正常处理
        try:
            llm = get_llm()
            response = llm.invoke(state["messages"])
            return {"messages": state["messages"] + [response]}
        except Exception as e:
            return {"messages": state["messages"] + [AIMessage(content="处理请求时出现错误")]}
    
    # 构建工作流
    workflow = StateGraph(MessagesState)
    workflow.add_node("process", permission_based_interrupt_node)
    workflow.set_entry_point("process")
    workflow.add_edge("process", END)
    
    checkpointer = InMemorySaver()
    app = workflow.compile(checkpointer=checkpointer)
    
    # 测试不同权限用户的操作
    test_cases = [
        ("user", "请删除所有用户数据"),
        ("admin", "请删除所有用户数据"),
        ("user", "请查看系统状态"),
        ("admin", "请查看系统状态")
    ]
    
    for user_role, message in test_cases:
        ColorfulLogger.step(f"测试: {user_role} - {message}")
        
        config = {
            "configurable": {
                "thread_id": f"perm_test_{user_role}_{hash(message) % 1000}",
                "user_role": user_role
            }
        }
        
        try:
            result = app.invoke({
                "messages": [HumanMessage(content=message)]
            }, config)
            
            ColorfulLogger.success("操作正常完成")
            
        except Exception as e:
            if "interrupt" in str(e).lower():
                ColorfulLogger.warning("操作被中断，需要审批")
            else:
                ColorfulLogger.error(f"操作失败: {e}")

def human_collaboration_workflow_example():
    """7.4 实时协作界面示例"""
    ColorfulLogger.info("\n=== 7.4 实时协作界面 ===")
    
    # 创建协作界面
    collaboration = HumanCollaborationInterface()
    
    def smart_interrupt_node(state: InteractionState, config) -> InteractionState:
        """智能中断节点"""
        session_id = config["configurable"]["thread_id"]
        operation_type = state["operation_type"]
        
        # 评估是否需要人工介入
        if requires_human_review(operation_type):
            # 请求人工审批
            approval_request = collaboration.request_approval(
                session_id, 
                {
                    "operation": operation_type,
                    "risk_level": assess_risk(operation_type),
                    "context": state["messages"][-1:] if state["messages"] else []
                }
            )
            
            # 触发中断，等待审批
            interrupt(f"等待审批，请求ID: {approval_request['session_id']}")
        
        return state
    
    def requires_human_review(operation_type: str) -> bool:
        """判断是否需要人工审核"""
        high_risk_operations = ["删除数据", "修改权限", "系统配置"]
        return operation_type in high_risk_operations
    
    def assess_risk(operation_type: str) -> str:
        """评估风险级别"""
        if operation_type in ["删除数据", "系统重启"]:
            return "high"
        elif operation_type in ["修改配置", "发送通知"]:
            return "medium"
        else:
            return "low"
    
    def execute_operation_node(state: InteractionState, config) -> InteractionState:
        """执行操作节点"""
        session_id = config["configurable"]["thread_id"]
        
        # 检查是否已获得审批
        if session_id in collaboration.pending_approvals:
            approval = collaboration.pending_approvals[session_id]
            
            if approval["status"] == "completed":
                decision = approval["decision"]
                
                if decision["approved"]:
                    ColorfulLogger.success(f"操作已获得审批，执行: {state['operation_type']}")
                    return {
                        **state,
                        "approved": True,
                        "reviewer": decision["reviewer_id"]
                    }
                else:
                    ColorfulLogger.warning("操作被拒绝")
                    return {
                        **state,
                        "approved": False,
                        "reviewer": decision["reviewer_id"]
                    }
        
        # 正常执行（无需审批的操作）
        ColorfulLogger.success(f"直接执行操作: {state['operation_type']}")
        return {
            **state,
            "approved": True,
            "reviewer": "system"
        }
    
    # 构建工作流
    workflow = StateGraph(InteractionState)
    workflow.add_node("check_approval", smart_interrupt_node)
    workflow.add_node("execute", execute_operation_node)
    workflow.set_entry_point("check_approval")
    workflow.add_edge("check_approval", "execute")
    workflow.add_edge("execute", END)
    
    checkpointer = InMemorySaver()
    app = workflow.compile(checkpointer=checkpointer)
    
    # 测试协作流程
    test_operations = ["查看报表", "删除数据", "修改权限"]
    
    for operation in test_operations:
        ColorfulLogger.step(f"测试操作: {operation}")
        
        config = {"configurable": {"thread_id": f"collab_{operation.replace(' ', '_')}"}}
        
        try:
            # 第一次执行（可能触发中断）
            result = app.invoke({
                "messages": [HumanMessage(content=f"执行操作: {operation}")],
                "operation_type": operation,
                "risk_level": "",
                "human_feedback": {},
                "approved": False,
                "reviewer": ""
            }, config)
            
            ColorfulLogger.success(f"操作 '{operation}' 完成")
            
        except Exception as e:
            if "interrupt" in str(e).lower():
                # 模拟人工审批
                session_id = config["configurable"]["thread_id"]
                ColorfulLogger.info("模拟人工审批过程...")
                
                collaboration.simulate_human_approval(session_id, auto_approve=True)
                
                # 继续执行
                try:
                    result = app.invoke(None, config)
                    ColorfulLogger.success(f"审批后操作 '{operation}' 完成")
                except Exception as e2:
                    ColorfulLogger.error(f"审批后执行失败: {e2}")
            else:
                ColorfulLogger.error(f"操作失败: {e}")
    
    # 生成合规报告
    ColorfulLogger.info("\n=== 合规报告 ===")
    report = collaboration.auditor.generate_compliance_report()
    for key, value in report.items():
        ColorfulLogger.info(f"{key}: {value}")

def main():
    """主函数"""
    ColorfulLogger.header("第七章：人机交互示例")
    
    try:
        # 1. 基础中断机制示例
        basic_interrupt_example()
        
        # 2. 基于内容的中断示例
        content_based_interrupt_example()
        
        # 3. 基于权限的中断示例
        permission_based_interrupt_example()
        
        # 4. 实时协作界面示例
        human_collaboration_workflow_example()
        
        ColorfulLogger.success("所有人机交互示例执行完成！")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 