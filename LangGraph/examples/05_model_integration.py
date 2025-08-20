"""
第五章：模型集成示例
对应文章：五、模型集成：灵活的AI引擎选择
"""

import sys
import os
import time
from typing import Dict, Any
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END
from langchain_openai import ChatOpenAI
from langchain_anthropic import ChatAnthropic
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm, config_manager
from utils import ColorfulLogger

class ModelState(TypedDict):
    """模型状态"""
    task_type: str
    complexity: str
    result: str
    model_used: str
    performance_stats: dict

class ModelManager:
    """模型管理器，支持多种自定义配置"""
    
    def __init__(self):
        self.models = {}
    
    def register_openai_model(self, name: str, config: dict):
        """注册自定义OpenAI模型"""
        try:
            self.models[name] = ChatOpenAI(
                model=config.get("model", "gpt-4o-mini"),
                api_key=config.get("api_key"),
                base_url=config.get("base_url", "https://api.openai.com/v1"),
                temperature=config.get("temperature", 0.1),
                max_tokens=config.get("max_tokens", 4000)
            )
            ColorfulLogger.success(f"成功注册OpenAI模型: {name}")
        except Exception as e:
            ColorfulLogger.error(f"注册OpenAI模型失败 {name}: {e}")
    
    def register_anthropic_model(self, name: str, config: dict):
        """注册自定义Anthropic模型"""
        try:
            self.models[name] = ChatAnthropic(
                model=config.get("model", "claude-3-haiku-20240307"),
                api_key=config.get("api_key"),
                base_url=config.get("base_url"),
                temperature=config.get("temperature", 0.1),
                max_tokens=config.get("max_tokens", 4000)
            )
            ColorfulLogger.success(f"成功注册Anthropic模型: {name}")
        except Exception as e:
            ColorfulLogger.error(f"注册Anthropic模型失败 {name}: {e}")
    
    def get_model(self, name: str):
        """获取指定模型"""
        return self.models.get(name)

class ModelMonitor:
    """模型性能监控器"""
    
    def __init__(self):
        self.stats = {}
    
    def track_request(self, model_name: str, start_time: float, 
                     end_time: float, token_count: int, success: bool):
        """跟踪模型请求性能"""
        if model_name not in self.stats:
            self.stats[model_name] = {
                "total_requests": 0,
                "successful_requests": 0,
                "total_latency": 0,
                "total_tokens": 0,
                "avg_latency": 0,
                "success_rate": 0
            }
        
        stats = self.stats[model_name]
        stats["total_requests"] += 1
        
        if success:
            stats["successful_requests"] += 1
            latency = end_time - start_time
            stats["total_latency"] += latency
            stats["total_tokens"] += token_count
            
            # 更新平均值
            stats["avg_latency"] = stats["total_latency"] / stats["successful_requests"]
        
        stats["success_rate"] = stats["successful_requests"] / stats["total_requests"]
    
    def get_stats(self, model_name: str = None):
        """获取统计信息"""
        if model_name:
            return self.stats.get(model_name, {})
        return self.stats

def model_manager_example():
    """5.1 自定义模型配置示例"""
    ColorfulLogger.info("=== 5.1 自定义模型配置 ===")
    
    # 初始化模型管理器
    model_manager = ModelManager()
    
    # 注册不同的模型配置
    try:
        # 注册标准OpenAI模型
        model_manager.register_openai_model("standard_gpt", {
            "model": "gpt-4o-mini",
            "api_key": os.getenv("OPENAI_API_KEY"),
            "base_url": "https://api.openai.com/v1",
            "temperature": 0.1,
            "max_tokens": 4000
        })
        
        # 注册创意写作模型（更高温度）
        model_manager.register_openai_model("creative_gpt", {
            "model": "gpt-4o-mini",
            "api_key": os.getenv("OPENAI_API_KEY"),
            "base_url": "https://api.openai.com/v1",
            "temperature": 0.8,
            "max_tokens": 4000
        })
        
        # 如果有Anthropic API Key，也注册Claude
        if os.getenv("ANTHROPIC_API_KEY"):
            model_manager.register_anthropic_model("claude_balanced", {
                "model": "claude-3-haiku-20240307",
                "api_key": os.getenv("ANTHROPIC_API_KEY"),
                "temperature": 0.3
            })
    
    except Exception as e:
        ColorfulLogger.warning(f"模型注册过程中出现问题: {e}")
    
    return model_manager

def multi_model_strategy_example(model_manager: ModelManager):
    """5.2 多模型策略选择示例"""
    ColorfulLogger.info("\n=== 5.2 多模型策略选择 ===")
    
    def get_appropriate_model(task_type: str, complexity: str = "medium"):
        """根据任务类型和复杂度选择合适的模型"""
        
        if task_type == "code_analysis":
            if complexity == "high":
                return model_manager.get_model("standard_gpt") or get_llm()
            else:
                return model_manager.get_model("standard_gpt") or get_llm()
        
        elif task_type == "creative_writing":
            # 创意写作使用更高的温度
            model = model_manager.get_model("creative_gpt")
            if model:
                return model
            else:
                # 备选方案：调整现有模型温度
                fallback_model = get_llm()
                fallback_model.temperature = 0.8
                return fallback_model
        
        elif task_type == "data_analysis":
            # 数据分析需要精确性
            model = model_manager.get_model("standard_gpt")
            if model:
                model.temperature = 0.0
                return model
            else:
                fallback_model = get_llm()
                fallback_model.temperature = 0.0
                return fallback_model
        
        else:
            # 通用任务使用默认配置
            return model_manager.get_model("standard_gpt") or get_llm()
    
    # 测试不同任务类型的模型选择
    test_cases = [
        ("code_analysis", "high"),
        ("creative_writing", "medium"),
        ("data_analysis", "low"),
        ("general", "medium")
    ]
    
    for task_type, complexity in test_cases:
        ColorfulLogger.step(f"测试任务类型: {task_type}, 复杂度: {complexity}")
        model = get_appropriate_model(task_type, complexity)
        ColorfulLogger.info(f"选择的模型: {type(model).__name__}, 温度: {getattr(model, 'temperature', '未知')}")
    
    return get_appropriate_model

def model_monitoring_example():
    """5.3 模型性能监控示例"""
    ColorfulLogger.info("\n=== 5.3 模型性能监控 ===")
    
    # 创建监控器实例
    monitor = ModelMonitor()
    
    def monitored_model_call(model, prompt, model_name: str):
        """监控模型调用的包装函数"""
        start_time = time.time()
        
        try:
            response = model.invoke([HumanMessage(content=prompt)])
            end_time = time.time()
            
            # 估算token数量（简化版本）
            token_count = len(prompt) // 4 + len(response.content) // 4
            
            monitor.track_request(model_name, start_time, end_time, token_count, True)
            return response
            
        except Exception as e:
            end_time = time.time()
            monitor.track_request(model_name, start_time, end_time, 0, False)
            raise e
    
    # 测试监控功能
    try:
        model = get_llm()
        
        ColorfulLogger.step("执行监控测试...")
        for i in range(3):
            prompt = f"请简单回答：什么是人工智能？（测试 {i+1}）"
            response = monitored_model_call(model, prompt, "test_model")
            ColorfulLogger.info(f"调用 {i+1} 完成")
            time.sleep(0.1)  # 避免过快请求
        
        # 显示统计信息
        stats = monitor.get_stats("test_model")
        ColorfulLogger.success("=== 模型性能统计 ===")
        for key, value in stats.items():
            ColorfulLogger.info(f"{key}: {value}")
            
    except Exception as e:
        ColorfulLogger.error(f"监控测试失败: {e}")
    
    return monitor

def resilient_model_call_example():
    """5.4 模型回退策略示例"""
    ColorfulLogger.info("\n=== 5.4 模型回退策略 ===")
    
    def resilient_model_call(prompt: str, task_type: str = "general") -> str:
        """具有回退机制的模型调用"""
        
        # 定义模型优先级（实际环境中应配置真实的不同模型）
        model_priority = [
            ("primary_model", "主要模型"),
            ("backup_model", "备份模型"),
            ("fallback_model", "最后备选")
        ]
        
        for model_name, description in model_priority:
            try:
                # 在实际环境中，这里应该是不同的模型实例
                model = get_llm()
                response = model.invoke([HumanMessage(content=prompt)])
                ColorfulLogger.success(f"✅ 使用 {description} 成功")
                return response.content
                
            except Exception as e:
                ColorfulLogger.warning(f"❌ {description} 失败: {e}")
                continue
        
        raise Exception("所有模型都不可用")
    
    # 测试回退机制
    try:
        ColorfulLogger.step("测试回退机制...")
        prompt = "请解释什么是LangGraph"
        result = resilient_model_call(prompt)
        ColorfulLogger.info(f"回退测试成功，结果: {result[:100]}...")
        
    except Exception as e:
        ColorfulLogger.error(f"回退测试失败: {e}")

def smart_processing_workflow_example():
    """智能处理工作流示例"""
    ColorfulLogger.info("\n=== 智能处理工作流 ===")
    
    model_manager = ModelManager()
    monitor = ModelMonitor()
    
    # 注册一个测试模型
    try:
        model_manager.register_openai_model("workflow_model", {
            "model": "gpt-4o-mini",
            "api_key": os.getenv("OPENAI_API_KEY"),
            "temperature": 0.1
        })
    except:
        ColorfulLogger.warning("使用默认模型作为备选")
    
    def smart_processing_node(state: ModelState) -> ModelState:
        """智能处理节点，根据任务选择模型"""
        task_type = state.get("task_type", "general")
        complexity = state.get("complexity", "medium")
        
        # 选择合适的模型
        model = model_manager.get_model("workflow_model") or get_llm()
        
        # 构建提示词
        prompt = f"任务类型：{task_type}，复杂度：{complexity}。请处理以下内容：演示LangGraph的模型集成功能"
        
        try:
            # 监控模型调用
            start_time = time.time()
            response = model.invoke([HumanMessage(content=prompt)])
            end_time = time.time()
            
            # 记录性能
            token_count = len(prompt) // 4 + len(response.content) // 4
            monitor.track_request("workflow_model", start_time, end_time, token_count, True)
            
            return {
                **state,
                "result": response.content,
                "model_used": "workflow_model",
                "performance_stats": monitor.get_stats("workflow_model")
            }
            
        except Exception as e:
            ColorfulLogger.error(f"模型调用失败: {e}")
            return {
                **state,
                "result": "处理失败",
                "model_used": "none",
                "performance_stats": {}
            }
    
    # 构建工作流
    workflow = StateGraph(ModelState)
    workflow.add_node("process", smart_processing_node)
    workflow.set_entry_point("process")
    workflow.add_edge("process", END)
    
    app = workflow.compile()
    
    # 测试不同类型的任务
    test_tasks = [
        {"task_type": "creative_writing", "complexity": "medium"},
        {"task_type": "data_analysis", "complexity": "high"},
        {"task_type": "code_analysis", "complexity": "low"}
    ]
    
    for task in test_tasks:
        ColorfulLogger.step(f"处理任务: {task}")
        
        initial_state = {
            "task_type": task["task_type"],
            "complexity": task["complexity"],
            "result": "",
            "model_used": "",
            "performance_stats": {}
        }
        
        result = app.invoke(initial_state)
        
        ColorfulLogger.info(f"任务完成:")
        ColorfulLogger.info(f"  使用模型: {result['model_used']}")
        ColorfulLogger.info(f"  结果长度: {len(result['result'])} 字符")
        if result['performance_stats']:
            stats = result['performance_stats']
            ColorfulLogger.info(f"  平均延迟: {stats.get('avg_latency', 0):.2f}秒")
            ColorfulLogger.info(f"  成功率: {stats.get('success_rate', 0):.2%}")

def main():
    """主函数"""
    ColorfulLogger.header("第五章：模型集成示例")
    
    try:
        # 1. 模型管理器示例
        model_manager = model_manager_example()
        
        # 2. 多模型策略示例
        multi_model_strategy_example(model_manager)
        
        # 3. 模型性能监控示例
        monitor = model_monitoring_example()
        
        # 4. 模型回退策略示例
        resilient_model_call_example()
        
        # 5. 智能处理工作流示例
        smart_processing_workflow_example()
        
        ColorfulLogger.success("所有模型集成示例执行完成！")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 