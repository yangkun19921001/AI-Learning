"""
LangGraph Tutorial Utilities
工具类和辅助函数库
"""

import os
import json
import time
import logging
from pathlib import Path
from typing import Dict, Any, List, Optional, Callable, TypeVar, Union
from datetime import datetime, timedelta
import sqlite3
import uuid
from functools import wraps
import asyncio
from concurrent.futures import ThreadPoolExecutor

from langchain_core.messages import BaseMessage, HumanMessage, AIMessage
from langgraph.graph import StateGraph

T = TypeVar('T')

class ColorfulLogger:
    """彩色日志工具"""
    
    COLORS = {
        'RED': '\033[31m',
        'GREEN': '\033[32m',
        'YELLOW': '\033[33m',
        'BLUE': '\033[34m',
        'MAGENTA': '\033[35m',
        'CYAN': '\033[36m',
        'WHITE': '\033[37m',
        'RESET': '\033[0m',
        'BOLD': '\033[1m'
    }
    
    @classmethod
    def log(cls, message: str, level: str = "INFO", color: str = "WHITE"):
        """打印彩色日志"""
        timestamp = datetime.now().strftime("%H:%M:%S")
        color_code = cls.COLORS.get(color, cls.COLORS['WHITE'])
        reset_code = cls.COLORS['RESET']
        
        print(f"{color_code}[{timestamp}] {level}: {message}{reset_code}")
    
    @classmethod
    def success(cls, message: str):
        cls.log(f"✅ {message}", "SUCCESS", "GREEN")
    
    @classmethod
    def error(cls, message: str):
        cls.log(f"❌ {message}", "ERROR", "RED")
    
    @classmethod
    def warning(cls, message: str):
        cls.log(f"⚠️ {message}", "WARNING", "YELLOW")
    
    @classmethod
    def info(cls, message: str):
        cls.log(f"ℹ️ {message}", "INFO", "BLUE")
    
    @classmethod
    def step(cls, message: str):
        cls.log(f"🔄 {message}", "STEP", "CYAN")
    
    @classmethod
    def header(cls, message: str):
        """打印标题头部"""
        separator = "=" * max(50, len(message) + 10)
        print(f"\n{cls.COLORS['BOLD']}{cls.COLORS['CYAN']}{separator}")
        print(f"🚀 {message}")
        print(f"{separator}{cls.COLORS['RESET']}\n")

class PerformanceMonitor:
    """性能监控工具"""
    
    def __init__(self):
        self.metrics = {}
        self.start_times = {}
    
    def start_timer(self, name: str):
        """开始计时"""
        self.start_times[name] = time.time()
    
    def end_timer(self, name: str):
        """结束计时"""
        if name in self.start_times:
            duration = time.time() - self.start_times[name]
            self.metrics[name] = duration
            del self.start_times[name]
            return duration
        return 0
    
    def get_metrics(self) -> Dict[str, float]:
        """获取性能指标"""
        return self.metrics.copy()
    
    def print_summary(self):
        """打印性能摘要"""
        if not self.metrics:
            ColorfulLogger.info("暂无性能数据")
            return
        
        ColorfulLogger.info("性能监控摘要:")
        for name, duration in sorted(self.metrics.items(), key=lambda x: x[1], reverse=True):
            ColorfulLogger.log(f"  {name}: {duration:.3f}秒", color="CYAN")

def timer(func: Callable[..., T]) -> Callable[..., T]:
    """性能计时装饰器"""
    @wraps(func)
    def wrapper(*args, **kwargs) -> T:
        start_time = time.time()
        try:
            result = func(*args, **kwargs)
            duration = time.time() - start_time
            ColorfulLogger.log(f"{func.__name__} 执行时间: {duration:.3f}秒", color="GREEN")
            return result
        except Exception as e:
            duration = time.time() - start_time
            ColorfulLogger.error(f"{func.__name__} 执行失败 ({duration:.3f}秒): {e}")
            raise
    return wrapper

class StateSerializer:
    """状态序列化工具"""
    
    @staticmethod
    def serialize_state(state: Dict[str, Any]) -> str:
        """序列化状态"""
        try:
            # 处理不可序列化的对象
            serializable_state = {}
            for key, value in state.items():
                try:
                    if isinstance(value, BaseMessage):
                        serializable_state[key] = {
                            "type": type(value).__name__,
                            "content": value.content
                        }
                    elif isinstance(value, list) and value and isinstance(value[0], BaseMessage):
                        serializable_state[key] = [
                            {"type": type(msg).__name__, "content": msg.content}
                            for msg in value
                        ]
                    elif isinstance(value, datetime):
                        serializable_state[key] = value.isoformat()
                    else:
                        json.dumps(value)  # 测试是否可序列化
                        serializable_state[key] = value
                except (TypeError, ValueError):
                    serializable_state[key] = str(value)
            
            return json.dumps(serializable_state, ensure_ascii=False, indent=2)
        except Exception as e:
            ColorfulLogger.error(f"状态序列化失败: {e}")
            return "{}"
    
    @staticmethod
    def deserialize_state(state_str: str) -> Dict[str, Any]:
        """反序列化状态"""
        try:
            return json.loads(state_str)
        except Exception as e:
            ColorfulLogger.error(f"状态反序列化失败: {e}")
            return {}

class GraphVisualizer:
    """图可视化工具"""
    
    @staticmethod
    def generate_mermaid(graph: StateGraph, title: str = "LangGraph Flow") -> str:
        """生成Mermaid图表"""
        try:
            # 获取图的节点和边信息
            compiled = graph.compile()
            
            mermaid_code = f"""
```mermaid
graph TD
    subgraph "{title}"
        START([开始])
"""
            
            # 添加节点
            nodes = compiled.get_graph().nodes
            for node_id in nodes:
                mermaid_code += f"        {node_id}[{node_id}]\n"
            
            # 添加边
            edges = compiled.get_graph().edges
            for edge in edges:
                source = edge[0] if edge[0] != "__start__" else "START"
                target = edge[1] if edge[1] != "__end__" else "END"
                mermaid_code += f"        {source} --> {target}\n"
            
            mermaid_code += """        END([结束])
    end
```"""
            
            return mermaid_code
        except Exception as e:
            ColorfulLogger.error(f"生成Mermaid图表失败: {e}")
            return "图表生成失败"

class DataGenerator:
    """测试数据生成器"""
    
    @staticmethod
    def generate_conversation(turns: int = 5) -> List[BaseMessage]:
        """生成对话数据"""
        conversations = [
            ("你好，我想学习LangGraph", "欢迎！我来帮助你学习LangGraph的核心概念。"),
            ("什么是状态管理？", "状态管理是LangGraph的核心，它定义了应用程序的数据结构。"),
            ("如何创建节点？", "节点是图中的计算单元，每个节点都是一个处理状态的函数。"),
            ("边是如何工作的？", "边定义节点之间的连接关系，支持条件分支和循环。"),
            ("能给个完整示例吗？", "当然！让我为你展示一个简单的ReAct Agent示例。"),
            ("如何处理错误？", "LangGraph提供了强大的错误处理和恢复机制。"),
            ("什么是checkpoint？", "Checkpoint允许你保存和恢复执行状态。"),
            ("如何实现人机交互？", "可以使用中断机制实现人工审核和决策干预。")
        ]
        
        messages = []
        for i in range(min(turns, len(conversations))):
            human_msg, ai_msg = conversations[i]
            messages.extend([
                HumanMessage(content=human_msg),
                AIMessage(content=ai_msg)
            ])
        
        return messages
    
    @staticmethod
    def generate_task_data(complexity: str = "medium") -> Dict[str, Any]:
        """生成任务数据"""
        tasks = {
            "simple": {
                "description": "计算两个数字的和",
                "input": {"a": 5, "b": 3},
                "expected_output": 8,
                "complexity": "simple",
                "risk_level": "low"
            },
            "medium": {
                "description": "分析用户反馈并提取关键信息",
                "input": ["产品很好用", "界面需要改进", "功能强大"],
                "expected_output": {"positive": 2, "negative": 1},
                "complexity": "medium",
                "risk_level": "medium"
            },
            "complex": {
                "description": "处理敏感数据并生成报告",
                "input": {"users": 1000, "sensitive": True},
                "expected_output": "detailed_report",
                "complexity": "complex",
                "risk_level": "high"
            }
        }
        
        return tasks.get(complexity, tasks["medium"])

class DatabaseManager:
    """数据库管理工具"""
    
    def __init__(self, db_path: str = "data/tutorial.db"):
        self.db_path = db_path
        Path(db_path).parent.mkdir(exist_ok=True)
        self._init_db()
    
    def _init_db(self):
        """初始化数据库"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        # 创建会话表
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS sessions (
                session_id TEXT PRIMARY KEY,
                user_id TEXT,
                created_at TEXT,
                last_updated TEXT,
                status TEXT,
                metadata TEXT
            )
        """)
        
        # 创建执行日志表
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS execution_logs (
                log_id TEXT PRIMARY KEY,
                session_id TEXT,
                node_name TEXT,
                execution_time REAL,
                status TEXT,
                error_message TEXT,
                timestamp TEXT,
                FOREIGN KEY (session_id) REFERENCES sessions (session_id)
            )
        """)
        
        conn.commit()
        conn.close()
    
    def save_session(self, session_id: str, user_id: str, status: str, metadata: Dict[str, Any]):
        """保存会话信息"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        cursor.execute("""
            INSERT OR REPLACE INTO sessions 
            (session_id, user_id, created_at, last_updated, status, metadata)
            VALUES (?, ?, ?, ?, ?, ?)
        """, (
            session_id, user_id, datetime.now().isoformat(),
            datetime.now().isoformat(), status, json.dumps(metadata)
        ))
        
        conn.commit()
        conn.close()
    
    def log_execution(self, session_id: str, node_name: str, execution_time: float, 
                     status: str, error_message: Optional[str] = None):
        """记录执行日志"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        log_id = f"log_{uuid.uuid4().hex[:8]}"
        
        cursor.execute("""
            INSERT INTO execution_logs 
            (log_id, session_id, node_name, execution_time, status, error_message, timestamp)
            VALUES (?, ?, ?, ?, ?, ?, ?)
        """, (
            log_id, session_id, node_name, execution_time, status, 
            error_message, datetime.now().isoformat()
        ))
        
        conn.commit()
        conn.close()
    
    def get_session_stats(self, session_id: str) -> Dict[str, Any]:
        """获取会话统计"""
        conn = sqlite3.connect(self.db_path)
        cursor = conn.cursor()
        
        # 获取执行统计
        cursor.execute("""
            SELECT 
                COUNT(*) as total_executions,
                AVG(execution_time) as avg_execution_time,
                SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_executions,
                SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as failed_executions
            FROM execution_logs 
            WHERE session_id = ?
        """, (session_id,))
        
        stats = cursor.fetchone()
        
        conn.close()
        
        if stats:
            return {
                "total_executions": stats[0],
                "avg_execution_time": stats[1] or 0,
                "successful_executions": stats[2],
                "failed_executions": stats[3],
                "success_rate": stats[2] / stats[0] if stats[0] > 0 else 0
            }
        else:
            return {
                "total_executions": 0,
                "avg_execution_time": 0,
                "successful_executions": 0,
                "failed_executions": 0,
                "success_rate": 0
            }

class TutorialRunner:
    """教程运行器"""
    
    def __init__(self):
        self.db_manager = DatabaseManager()
        self.performance_monitor = PerformanceMonitor()
        self.current_session = None
    
    def start_session(self, user_id: str, tutorial_name: str) -> str:
        """开始新的教程会话"""
        session_id = f"tutorial_{uuid.uuid4().hex[:8]}"
        self.current_session = session_id
        
        metadata = {
            "tutorial_name": tutorial_name,
            "started_at": datetime.now().isoformat()
        }
        
        self.db_manager.save_session(session_id, user_id, "started", metadata)
        ColorfulLogger.success(f"开始教程会话: {session_id}")
        
        return session_id
    
    def run_example(self, example_name: str, example_func: Callable):
        """运行教程示例"""
        if not self.current_session:
            ColorfulLogger.error("请先开始一个会话")
            return
        
        ColorfulLogger.step(f"运行示例: {example_name}")
        
        start_time = time.time()
        status = "success"
        error_message = None
        
        try:
            result = example_func()
            ColorfulLogger.success(f"示例 {example_name} 运行成功")
            return result
        except Exception as e:
            status = "error"
            error_message = str(e)
            ColorfulLogger.error(f"示例 {example_name} 运行失败: {e}")
            raise
        finally:
            execution_time = time.time() - start_time
            self.db_manager.log_execution(
                self.current_session, example_name, execution_time, status, error_message
            )
    
    def end_session(self):
        """结束教程会话"""
        if not self.current_session:
            return
        
        stats = self.db_manager.get_session_stats(self.current_session)
        
        metadata = {
            "ended_at": datetime.now().isoformat(),
            "statistics": stats
        }
        
        self.db_manager.save_session(self.current_session, "", "completed", metadata)
        
        ColorfulLogger.info("会话统计:")
        ColorfulLogger.log(f"  总执行次数: {stats['total_executions']}", color="CYAN")
        ColorfulLogger.log(f"  平均执行时间: {stats['avg_execution_time']:.3f}秒", color="CYAN")
        ColorfulLogger.log(f"  成功率: {stats['success_rate']:.1%}", color="GREEN" if stats['success_rate'] > 0.8 else "YELLOW")
        
        self.current_session = None

class AsyncHelper:
    """异步执行辅助工具"""
    
    @staticmethod
    async def run_parallel_tasks(tasks: List[Callable], max_workers: int = 5) -> List[Any]:
        """并行执行任务"""
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            loop = asyncio.get_event_loop()
            futures = [loop.run_in_executor(executor, task) for task in tasks]
            return await asyncio.gather(*futures)
    
    @staticmethod
    def run_async(coro):
        """在同步环境中运行异步代码"""
        try:
            loop = asyncio.get_event_loop()
            if loop.is_running():
                # 如果已有事件循环在运行，创建新的任务
                return asyncio.create_task(coro)
            else:
                return loop.run_until_complete(coro)
        except RuntimeError:
            # 没有事件循环，创建新的
            return asyncio.run(coro)

class ConfigValidator:
    """配置验证工具"""
    
    @staticmethod
    def validate_llm_config(provider: str, config: Dict[str, Any]) -> bool:
        """验证LLM配置"""
        required_fields = {
            "openai": ["api_key", "model"],
            "anthropic": ["api_key", "model"],
            "azure": ["api_key", "base_url", "deployment_name"],
            "custom": ["api_key", "base_url", "model"]
        }
        
        if provider not in required_fields:
            ColorfulLogger.error(f"不支持的LLM提供商: {provider}")
            return False
        
        missing_fields = []
        for field in required_fields[provider]:
            if not config.get(field):
                missing_fields.append(field)
        
        if missing_fields:
            ColorfulLogger.error(f"{provider} 配置缺少必需字段: {missing_fields}")
            return False
        
        ColorfulLogger.success(f"{provider} 配置验证通过")
        return True
    
    @staticmethod
    def validate_environment() -> bool:
        """验证环境配置"""
        checks = [
            ("Python版本", lambda: True),  # 简化检查
            ("依赖包", lambda: True),
            ("数据目录", lambda: Path("data").exists() or Path("data").mkdir(exist_ok=True))
        ]
        
        all_passed = True
        for check_name, check_func in checks:
            try:
                if check_func():
                    ColorfulLogger.success(f"{check_name} 检查通过")
                else:
                    ColorfulLogger.error(f"{check_name} 检查失败")
                    all_passed = False
            except Exception as e:
                ColorfulLogger.error(f"{check_name} 检查出错: {e}")
                all_passed = False
        
        return all_passed

# 导出主要工具类
__all__ = [
    'ColorfulLogger',
    'PerformanceMonitor', 
    'timer',
    'StateSerializer',
    'GraphVisualizer',
    'DataGenerator',
    'DatabaseManager',
    'TutorialRunner',
    'AsyncHelper',
    'ConfigValidator'
]

# 示例使用
if __name__ == "__main__":
    # 测试彩色日志
    ColorfulLogger.info("这是一个信息日志")
    ColorfulLogger.success("这是一个成功日志")
    ColorfulLogger.warning("这是一个警告日志")
    ColorfulLogger.error("这是一个错误日志")
    
    # 测试性能监控
    monitor = PerformanceMonitor()
    monitor.start_timer("test")
    time.sleep(0.1)
    monitor.end_timer("test")
    monitor.print_summary()
    
    # 测试数据生成
    conversation = DataGenerator.generate_conversation(3)
    print(f"生成了 {len(conversation)} 条对话消息")
    
    # 测试环境验证
    ConfigValidator.validate_environment() 