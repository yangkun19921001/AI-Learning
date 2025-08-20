"""
第四章：记忆系统示例
对应文章：四、记忆系统：构建有历史的智能体
"""

import sys
import os
import uuid
from datetime import datetime
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from typing_extensions import TypedDict
from langgraph.graph import StateGraph, END, MessagesState
from langgraph.checkpoint.memory import InMemorySaver, MemorySaver
from langgraph.store.memory import InMemoryStore
from langchain_core.messages import HumanMessage, AIMessage
from config import get_llm
from utils import ColorfulLogger

# 导入持久化存储相关模块
SQLITE_AVAILABLE = False
POSTGRES_AVAILABLE = False
SQLITE_STORE_AVAILABLE = False

# SQLite Checkpoint 支持
try:
    from langgraph.checkpoint.sqlite import SqliteSaver
    SQLITE_AVAILABLE = True
    print("✅ SQLite Checkpoint 支持可用")
except ImportError as e:
    print(f"⚠️ SQLite Checkpoint 不可用: {e}")

# PostgreSQL 支持 (可选)
try:
    from langgraph.checkpoint.postgres import PostgresSaver
    from langgraph.store.postgres import PostgresStore
    POSTGRES_AVAILABLE = True
    print("✅ PostgreSQL 支持可用")
except ImportError:
    print("ℹ️ PostgreSQL 支持不可用 (可选)")

# SQLite Store 支持
try:
    from langgraph.store.sqlite import SqliteStore
    SQLITE_STORE_AVAILABLE = True
    print("✅ SQLite Store 支持可用")
except ImportError as e:
    print(f"⚠️ SQLite Store 不可用: {e}")

class ConversationState(TypedDict):
    """对话状态"""
    messages: list
    user_preferences: dict
    conversation_history: list
    context_summary: str

class MemoryStorageManager:
    """记忆存储管理器 - 支持多种持久化存储"""
    
    def __init__(self, environment: str = "development"):
        self.environment = environment
        self.short_term_storage = None  # Checkpointer
        self.long_term_storage = None   # Store
        self.storage_type = os.getenv("MEMORY_STORAGE_TYPE", "sqlite").lower()  # sqlite, postgres, memory
        self.db_path = os.getenv("MEMORY_DB_PATH", "./data/memory.db")
        self.postgres_url = os.getenv("POSTGRES_URL", "postgresql://user:password@localhost:5432/langgraph_memory")
        
        # 确保数据目录存在
        os.makedirs(os.path.dirname(self.db_path), exist_ok=True)
        
        self._setup_storage()
    
    def _setup_storage(self):
        """根据环境和配置设置存储"""
        ColorfulLogger.info(f"🔧 配置 {self.environment} 环境存储...")
        ColorfulLogger.info(f"📦 存储类型: {self.storage_type}")
        
        if self.environment == "production":
            self._setup_production_storage()
        elif self.environment == "development":
            self._setup_development_storage()
        elif self.environment == "testing":
            self._setup_testing_storage()
        else:
            ColorfulLogger.warning(f"⚠️ 未知环境: {self.environment}，使用开发环境配置")
            self._setup_development_storage()
    
    def _setup_production_storage(self):
        """生产环境存储配置"""
        ColorfulLogger.info("🏭 配置生产环境存储...")
        
        if self.storage_type == "postgres" and POSTGRES_AVAILABLE:
            try:
                # PostgreSQL 存储
                ColorfulLogger.info(f"🐘 使用 PostgreSQL: {self.postgres_url}")
                self.short_term_storage = PostgresSaver.from_conn_string(self.postgres_url).__enter__()
                self.long_term_storage = PostgresStore.from_conn_string(self.postgres_url).__enter__()
                ColorfulLogger.success("✅ PostgreSQL 存储配置成功")
            except Exception as e:
                ColorfulLogger.error(f"❌ PostgreSQL 连接失败: {e}")
                self._fallback_to_sqlite()
        
        elif self.storage_type == "sqlite":
            self._setup_sqlite_storage()
        
        else:
            ColorfulLogger.warning("⚠️ 生产环境回退到内存存储（不推荐）")
            self._setup_memory_storage()
    
    def _setup_development_storage(self):
        """开发环境存储配置"""
        ColorfulLogger.info("🛠️ 配置开发环境存储...")
        
        if self.storage_type == "sqlite":
            self._setup_sqlite_storage()
        elif self.storage_type == "postgres" and POSTGRES_AVAILABLE:
            try:
                ColorfulLogger.info(f"🐘 开发环境使用 PostgreSQL: {self.postgres_url}")
                self.short_term_storage = PostgresSaver.from_conn_string(self.postgres_url).__enter__()
                self.long_term_storage = PostgresStore.from_conn_string(self.postgres_url).__enter__()
                ColorfulLogger.success("✅ 开发环境 PostgreSQL 配置成功")
            except Exception as e:
                ColorfulLogger.error(f"❌ PostgreSQL 连接失败，回退到 SQLite: {e}")
                self._setup_sqlite_storage()
        else:
            # 默认使用内存存储进行快速开发
            self._setup_memory_storage()
    
    def _setup_testing_storage(self):
        """测试环境存储配置"""
        ColorfulLogger.info("🧪 配置测试环境存储...")
        
        if self.storage_type == "sqlite":
            # 测试环境使用临时SQLite数据库
            test_db_path = "./data/test_memory.db"
            if SQLITE_AVAILABLE:
                try:
                    self.short_term_storage = SqliteSaver.from_conn_string(test_db_path).__enter__()
                    ColorfulLogger.success(f"✅ 测试环境 SQLite Checkpointer: {test_db_path}")
                except Exception as e:
                    ColorfulLogger.error(f"❌ SQLite Checkpointer 创建失败: {e}")
                    self.short_term_storage = MemorySaver()
            else:
                self.short_term_storage = MemorySaver()
            
            if SQLITE_STORE_AVAILABLE:
                try:
                    self.long_term_storage = SqliteStore.from_conn_string(test_db_path).__enter__()
                    ColorfulLogger.success(f"✅ 测试环境 SQLite Store: {test_db_path}")
                except Exception as e:
                    ColorfulLogger.error(f"❌ SQLite Store 创建失败: {e}")
                    self.long_term_storage = InMemoryStore()
            else:
                self.long_term_storage = InMemoryStore()
        else:
            # 测试环境默认使用内存存储
            self._setup_memory_storage()
    
    def _setup_sqlite_storage(self):
        """SQLite 存储配置"""
        ColorfulLogger.info(f"🗄️ 配置 SQLite 存储: {self.db_path}")
        
        # 短期记忆 (Checkpointer)
        if SQLITE_AVAILABLE:
            try:
                # 保存上下文管理器以便后续清理
                self._sqlite_saver_cm = SqliteSaver.from_conn_string(self.db_path)
                self.short_term_storage = self._sqlite_saver_cm.__enter__()
                ColorfulLogger.success("✅ SQLite Checkpointer 配置成功")
            except Exception as e:
                ColorfulLogger.error(f"❌ SQLite Checkpointer 创建失败: {e}")
                self.short_term_storage = MemorySaver()
                self._sqlite_saver_cm = None
        else:
            self.short_term_storage = MemorySaver()
            self._sqlite_saver_cm = None
        
        # 长期记忆 (Store)
        if SQLITE_STORE_AVAILABLE:
            try:
                # 保存上下文管理器以便后续清理
                self._sqlite_store_cm = SqliteStore.from_conn_string(self.db_path)
                self.long_term_storage = self._sqlite_store_cm.__enter__()
                ColorfulLogger.success("✅ SQLite Store 配置成功")
            except Exception as e:
                ColorfulLogger.error(f"❌ SQLite Store 创建失败: {e}")
                self.long_term_storage = InMemoryStore()
                self._sqlite_store_cm = None
        else:
            self.long_term_storage = InMemoryStore()
            self._sqlite_store_cm = None
    
    def _setup_memory_storage(self):
        """内存存储配置"""
        ColorfulLogger.info("💾 配置内存存储...")
        self.short_term_storage = InMemorySaver()
        self.long_term_storage = InMemoryStore()
        ColorfulLogger.success("✅ 内存存储配置完成")
    
    def _fallback_to_sqlite(self):
        """回退到 SQLite 存储"""
        ColorfulLogger.warning("⚠️ 回退到 SQLite 存储...")
        self._setup_sqlite_storage()
    
    def get_storage_info(self):
        """获取存储信息"""
        return {
            "environment": self.environment,
            "storage_type": self.storage_type,
            "short_term_storage": type(self.short_term_storage).__name__,
            "long_term_storage": type(self.long_term_storage).__name__,
            "db_path": self.db_path if self.storage_type == "sqlite" else None,
            "postgres_url": self.postgres_url if self.storage_type == "postgres" else None
        }
    
    def cleanup(self):
        """清理资源"""
        try:
            # 清理 SQLite 上下文管理器
            if hasattr(self, '_sqlite_saver_cm') and self._sqlite_saver_cm:
                self._sqlite_saver_cm.__exit__(None, None, None)
            if hasattr(self, '_sqlite_store_cm') and self._sqlite_store_cm:
                self._sqlite_store_cm.__exit__(None, None, None)
            
            # 清理其他资源
            if hasattr(self.short_term_storage, 'close'):
                self.short_term_storage.close()
            if hasattr(self.long_term_storage, 'close'):
                self.long_term_storage.close()
            
            ColorfulLogger.info("🧹 存储资源已清理")
        except Exception as e:
            ColorfulLogger.warning(f"⚠️ 清理资源时出现警告: {e}")

def short_term_memory_example():
    """4.1 短期记忆示例"""
    ColorfulLogger.info("=== 4.1 短期记忆：线程级的对话上下文 ===")
    
    # 创建存储管理器
    storage_manager = MemoryStorageManager("development")
    
    def chat_node(state: MessagesState) -> MessagesState:
        """对话节点"""
        try:
            llm = get_llm()
            response = llm.invoke(state["messages"])
            return {"messages": [response]}
        except Exception as e:
            ColorfulLogger.error(f"LLM调用失败: {e}")
            return {"messages": [AIMessage(content="抱歉，出现了错误")]}
    
    # 构建图
    workflow = StateGraph(MessagesState)
    workflow.add_node("chat", chat_node)
    workflow.set_entry_point("chat")
    workflow.add_edge("chat", END)
    
    # 编译时启用短期记忆
    app = workflow.compile(checkpointer=storage_manager.short_term_storage)
    
    # 模拟多轮对话
    config = {"configurable": {"thread_id": "user_123_session"}}
    
    ColorfulLogger.step("第一轮对话...")
    result1 = app.invoke({
        "messages": [HumanMessage(content="你好，我是张三")]
    }, config)
    ColorfulLogger.success(f"AI回复: {result1['messages'][-1].content}")
    
    ColorfulLogger.step("第二轮对话（智能体应该记住用户姓名）...")
    result2 = app.invoke({
        "messages": [HumanMessage(content="我刚才说我叫什么？")]
    }, config)
    ColorfulLogger.success(f"AI回复: {result2['messages'][-1].content}")
    
    return result2

def conversation_trimming_example():
    """4.1.3 对话历史管理示例"""
    ColorfulLogger.info("\n=== 4.1.3 对话历史管理 ===")
    
    def trim_conversation(state: MessagesState) -> MessagesState:
        """修剪对话历史，保留最近的重要消息"""
        messages = state["messages"]
        
        # 保留系统消息和最近10轮对话
        system_messages = [msg for msg in messages if msg.type == "system"]
        recent_messages = messages[-10:]  # 最近10条消息
        
        trimmed_messages = system_messages + recent_messages
        
        ColorfulLogger.info(f"对话历史修剪: {len(messages)} -> {len(trimmed_messages)} 条消息")
        
        return {"messages": trimmed_messages}
    
    # 模拟长对话历史
    long_conversation = [
        HumanMessage(content=f"这是第 {i} 条消息")
        for i in range(1, 21)  # 20条消息
    ]
    
    initial_state = {"messages": long_conversation}
    result = trim_conversation(initial_state)
    
    ColorfulLogger.success(f"修剪后的消息数量: {len(result['messages'])}")
    return result

def long_term_memory_example():
    """4.2 长期记忆示例"""
    ColorfulLogger.info("\n=== 4.2 长期记忆：跨对话的知识积累 ===")
    
    # 创建存储管理器，支持持久化存储
    storage_manager = MemoryStorageManager("development")
    
    # 显示存储配置信息
    storage_info = storage_manager.get_storage_info()
    ColorfulLogger.info("📊 存储配置信息:")
    for key, value in storage_info.items():
        if value:
            ColorfulLogger.info(f"  • {key}: {value}")
    
    ColorfulLogger.info("\n💡 提示: 可通过环境变量配置存储类型:")
    ColorfulLogger.info("  • MEMORY_STORAGE_TYPE=sqlite (默认)")
    ColorfulLogger.info("  • MEMORY_STORAGE_TYPE=postgres")
    ColorfulLogger.info("  • MEMORY_STORAGE_TYPE=memory")
    ColorfulLogger.info("  • MEMORY_DB_PATH=./data/memory.db (SQLite路径)")
    ColorfulLogger.info("  • POSTGRES_URL=postgresql://user:pass@host:port/db")
    
    def save_user_preference(state, config, *, store):
        """保存用户偏好到长期记忆"""
        user_id = config["configurable"]["user_id"]
        namespace = ("user_preferences", user_id)
        
        last_message = state["messages"][-1].content
        if "我喜欢" in last_message:
            # 提取偏好信息
            preference = last_message.replace("我喜欢", "").strip()
            store.put(namespace, "preference", {
                "preference": preference,
                "timestamp": datetime.now().isoformat()
            })
            ColorfulLogger.success(f"已保存用户偏好: {preference}")
        
        return state
    
    def retrieve_user_preference(state, config, *, store):
        """检索用户偏好"""
        user_id = config["configurable"]["user_id"]
        namespace = ("user_preferences", user_id)
        
        try:
            preference_data = store.get(namespace, "preference")
            if preference_data:
                preference = preference_data.value["preference"]
                ColorfulLogger.info(f"检索到用户偏好: {preference}")
                return {
                    **state,
                    "user_preference": preference
                }
        except Exception as e:
            ColorfulLogger.warning(f"检索偏好失败: {e}")
        
        return state
    
    def chat_with_memory(state, config, *, store):
        """带记忆的对话节点"""
        try:
            # 先检索用户偏好
            state = retrieve_user_preference(state, config, store=store)
            
            llm = get_llm()
            
            # 构建包含偏好信息的提示
            user_pref = state.get("user_preference", "无")
            system_message = f"用户偏好: {user_pref}。请在回复中考虑用户的偏好。"
            
            messages_with_context = [HumanMessage(content=system_message)] + state["messages"]
            response = llm.invoke(messages_with_context)
            
            new_state = {"messages": state["messages"] + [response]}
            
            # 保存新的偏好信息
            save_user_preference(new_state, config, store=store)
            
            return new_state
            
        except Exception as e:
            ColorfulLogger.error(f"对话失败: {e}")
            return {"messages": state["messages"] + [AIMessage(content="抱歉，出现了错误")]}
    
    # 构建图
    workflow = StateGraph(MessagesState)
    workflow.add_node("chat", chat_with_memory)
    workflow.set_entry_point("chat")
    workflow.add_edge("chat", END)
    
    # 编译时关联长期记忆存储
    app = workflow.compile(
        checkpointer=storage_manager.short_term_storage,
        store=storage_manager.long_term_storage
    )
    
    # 第一次对话：建立偏好
    config1 = {"configurable": {"thread_id": "session1", "user_id": "user_456"}}
    
    ColorfulLogger.step("建立用户偏好...")
    result1 = app.invoke({
        "messages": [HumanMessage(content="你好，我喜欢简洁的回答")]
    }, config1)
    
    # 第二次对话：在新会话中使用偏好
    config2 = {"configurable": {"thread_id": "session2", "user_id": "user_456"}}
    
    ColorfulLogger.step("在新会话中应用偏好...")
    result2 = app.invoke({
        "messages": [HumanMessage(content="请介绍一下Python")]
    }, config2)
    
    ColorfulLogger.success("长期记忆测试完成")
    
    # 清理存储资源
    storage_manager.cleanup()
    
    return result2

def memory_types_example():
    """4.2.3 记忆类型示例"""
    ColorfulLogger.info("\n=== 4.2.3 记忆的类型与组织 ===")
    
    store = InMemoryStore()
    user_id = "demo_user"
    
    # 语义记忆：事实性知识
    ColorfulLogger.step("保存语义记忆...")
    user_facts = {
        "name": "张三",
        "occupation": "软件工程师",
        "preferences": {
            "communication_style": "简洁直接",
            "language": "中文"
        }
    }
    store.put(("semantic", user_id), "user_profile", user_facts)
    
    # 情节记忆：具体经历
    ColorfulLogger.step("保存情节记忆...")
    episode_id = str(uuid.uuid4())
    episode = {
        "timestamp": "2024-01-15T10:30:00",
        "context": "用户咨询项目进度",
        "action": "提供了详细的进度报告",
        "outcome": "用户表示满意"
    }
    store.put(("episodic", user_id), episode_id, episode)
    
    # 程序记忆：操作方式
    ColorfulLogger.step("保存程序记忆...")
    procedure_id = str(uuid.uuid4())
    procedure = {
        "situation": "用户询问技术问题",
        "approach": "先确认具体场景，再提供分步解决方案",
        "effectiveness": "高"
    }
    store.put(("procedural", user_id), procedure_id, procedure)
    
    # 检索记忆
    ColorfulLogger.step("检索各类记忆...")
    
    semantic_memory = store.get(("semantic", user_id), "user_profile")
    ColorfulLogger.info(f"语义记忆: {semantic_memory.value if semantic_memory else '无'}")
    
    # 列出情节记忆
    episodic_memories = list(store.search(("episodic", user_id)))
    ColorfulLogger.info(f"情节记忆数量: {len(episodic_memories)}")
    
    # 列出程序记忆
    procedural_memories = list(store.search(("procedural", user_id)))
    ColorfulLogger.info(f"程序记忆数量: {len(procedural_memories)}")

def memory_retention_policy_example():
    """记忆保留策略示例"""
    ColorfulLogger.info("\n=== 记忆保留策略 ===")
    
    def setup_memory_retention_policy():
        """设置记忆保留策略"""
        return {
            "short_term": {
                "max_messages": 50,        # 最多保留50条消息
                "ttl_days": 7             # 7天后自动清理
            },
            "long_term": {
                "user_preferences": {"ttl_days": 365},  # 用户偏好保留1年
                "interaction_history": {"ttl_days": 30}, # 交互历史保留30天
                "sensitive_data": {"ttl_days": 1}        # 敏感数据1天后清理
            }
        }
    
    policy = setup_memory_retention_policy()
    ColorfulLogger.info(f"记忆保留策略: {policy}")
    
    # 模拟数据清理
    def cleanup_expired_memories(store, policy):
        """清理过期记忆"""
        ColorfulLogger.step("执行记忆清理...")
        
        # 这里应该实现实际的清理逻辑
        # 基于timestamp和TTL策略删除过期数据
        
        ColorfulLogger.success("记忆清理完成")
    
    store = InMemoryStore()
    cleanup_expired_memories(store, policy)

def persistent_storage_demo():
    """4.3 持久化存储演示"""
    ColorfulLogger.info("\n=== 4.3 持久化存储演示 ===")
    
    # 测试不同存储类型
    storage_types = ["memory", "sqlite"]
    
    for storage_type in storage_types:
        ColorfulLogger.step(f"测试 {storage_type.upper()} 存储...")
        
        # 临时设置环境变量
        original_storage_type = os.getenv("MEMORY_STORAGE_TYPE")
        os.environ["MEMORY_STORAGE_TYPE"] = storage_type
        
        try:
            # 创建存储管理器
            storage_manager = MemoryStorageManager("production")
            storage_info = storage_manager.get_storage_info()
            
            ColorfulLogger.info(f"📦 {storage_type.upper()} 存储配置:")
            ColorfulLogger.info(f"  • 短期存储: {storage_info['short_term_storage']}")
            ColorfulLogger.info(f"  • 长期存储: {storage_info['long_term_storage']}")
            
            if storage_type == "sqlite" and storage_info.get('db_path'):
                ColorfulLogger.info(f"  • 数据库路径: {storage_info['db_path']}")
                # 检查数据库文件是否存在
                if os.path.exists(storage_info['db_path']):
                    file_size = os.path.getsize(storage_info['db_path'])
                    ColorfulLogger.info(f"  • 数据库大小: {file_size} bytes")
                else:
                    ColorfulLogger.info("  • 数据库文件: 将在首次使用时创建")
            
            # 清理资源
            storage_manager.cleanup()
            
        except Exception as e:
            ColorfulLogger.error(f"❌ {storage_type.upper()} 存储测试失败: {e}")
        
        finally:
            # 恢复原始环境变量
            if original_storage_type:
                os.environ["MEMORY_STORAGE_TYPE"] = original_storage_type
            elif "MEMORY_STORAGE_TYPE" in os.environ:
                del os.environ["MEMORY_STORAGE_TYPE"]
        
        print("-" * 50)
    
    ColorfulLogger.success("✅ 持久化存储演示完成")

def main():
    """主函数"""
    ColorfulLogger.header("第四章：记忆系统示例")
    
    try:
        # 1. 短期记忆示例
        short_term_memory_example()
        
        # 2. 对话历史管理
        conversation_trimming_example()
        
        # 3. 长期记忆示例
        long_term_memory_example()
        
        # 4. 记忆类型示例
        memory_types_example()
        
        # 5. 记忆保留策略
        memory_retention_policy_example()
        
        # 6. 持久化存储演示
        persistent_storage_demo()
        
        ColorfulLogger.success("所有记忆系统示例执行完成！")
        
    except Exception as e:
        ColorfulLogger.error(f"示例运行失败: {e}")

if __name__ == "__main__":
    main() 