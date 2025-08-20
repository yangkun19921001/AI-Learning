"""
LangGraph Tutorial Configuration Manager
支持多种LLM提供商和自定义配置的配置管理器
"""

import os
import logging
from typing import Optional, Dict, Any, Union
from dataclasses import dataclass
from pathlib import Path
from dotenv import load_dotenv

from langchain_openai import ChatOpenAI
from langchain_anthropic import ChatAnthropic
from langchain_core.language_models import BaseChatModel


@dataclass
class LLMConfig:
    """LLM配置数据类"""
    provider: str
    api_key: str
    base_url: Optional[str] = None
    model: str = "gpt-4o-mini"
    temperature: float = 0.1
    max_tokens: int = 4000
    api_version: Optional[str] = None
    deployment_name: Optional[str] = None


class ConfigManager:
    """配置管理器 - 支持多种LLM提供商"""
    
    def __init__(self, env_file: Optional[str] = None):
        """
        初始化配置管理器
        
        Args:
            env_file: 环境变量文件路径，默认为 .env
        """
        self.env_file = env_file or ".env"
        self._load_environment()
        self._setup_logging()
        
    def _load_environment(self):
        """加载环境变量"""
        if os.path.exists(self.env_file):
            load_dotenv(self.env_file)
            print(f"✅ 已加载配置文件: {self.env_file}")
        else:
            print(f"⚠️  配置文件 {self.env_file} 不存在，使用默认配置")
    
    def _setup_logging(self):
        """设置日志配置"""
        log_level = os.getenv("LOG_LEVEL", "INFO")
        log_format = os.getenv(
            "LOG_FORMAT",
            "%(asctime)s - %(name)s - %(levelname)s - %(message)s"
        )

        # 创建日志目录（在创建文件处理器之前）
        Path("logs").mkdir(exist_ok=True)

        logging.basicConfig(
            level=getattr(logging, log_level.upper()),
            format=log_format,
            handlers=[
                logging.StreamHandler(),
                logging.FileHandler("logs/langgraph.log", encoding="utf-8")
            ]
        )
    
    def get_llm_config(self, provider: Optional[str] = None) -> LLMConfig:
        """
        获取LLM配置
        
        Args:
            provider: LLM提供商 (openai|anthropic|azure|custom)
            
        Returns:
            LLMConfig: LLM配置对象
        """
        provider = provider or os.getenv("DEFAULT_LLM_PROVIDER", "openai")
        
        if provider == "openai":
            return self._get_openai_config()
        elif provider == "anthropic":
            return self._get_anthropic_config()
        elif provider == "azure":
            return self._get_azure_config()
        elif provider == "custom":
            return self._get_custom_config()
        else:
            raise ValueError(f"不支持的LLM提供商: {provider}")
    
    def _get_openai_config(self) -> LLMConfig:
        """获取OpenAI配置"""
        return LLMConfig(
            provider="openai",
            api_key=os.getenv("OPENAI_API_KEY", ""),
            base_url=os.getenv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
            model=os.getenv("OPENAI_MODEL", "gpt-4o-mini"),
            temperature=float(os.getenv("OPENAI_TEMPERATURE", "0.1")),
            max_tokens=int(os.getenv("OPENAI_MAX_TOKENS", "4000"))
        )
    
    def _get_anthropic_config(self) -> LLMConfig:
        """获取Anthropic配置"""
        return LLMConfig(
            provider="anthropic",
            api_key=os.getenv("ANTHROPIC_API_KEY", ""),
            base_url=os.getenv("ANTHROPIC_BASE_URL", "https://api.anthropic.com"),
            model=os.getenv("ANTHROPIC_MODEL", "claude-3-haiku-20240307"),
            temperature=float(os.getenv("ANTHROPIC_TEMPERATURE", "0.1")),
            max_tokens=int(os.getenv("ANTHROPIC_MAX_TOKENS", "4000"))
        )
    
    def _get_azure_config(self) -> LLMConfig:
        """获取Azure OpenAI配置"""
        return LLMConfig(
            provider="azure",
            api_key=os.getenv("AZURE_OPENAI_API_KEY", ""),
            base_url=os.getenv("AZURE_OPENAI_ENDPOINT", ""),
            model=os.getenv("AZURE_MODEL", "gpt-4o-mini"),
            temperature=float(os.getenv("AZURE_OPENAI_TEMPERATURE", "0.1")),
            max_tokens=int(os.getenv("AZURE_OPENAI_MAX_TOKENS", "4000")),
            api_version=os.getenv("AZURE_OPENAI_API_VERSION", "2024-02-01"),
            deployment_name=os.getenv("AZURE_DEPLOYMENT_NAME", "")
        )
    
    def _get_custom_config(self) -> LLMConfig:
        """获取自定义LLM配置"""
        return LLMConfig(
            provider="custom",
            api_key=os.getenv("CUSTOM_LLM_API_KEY", ""),
            base_url=os.getenv("CUSTOM_LLM_BASE_URL", ""),
            model=os.getenv("CUSTOM_LLM_MODEL", ""),
            temperature=float(os.getenv("CUSTOM_LLM_TEMPERATURE", "0.1")),
            max_tokens=int(os.getenv("CUSTOM_LLM_MAX_TOKENS", "4000"))
        )
    
    def create_llm(self, provider: Optional[str] = None, **kwargs) -> BaseChatModel:
        """
        创建LLM实例
        
        Args:
            provider: LLM提供商
            **kwargs: 额外的配置参数
            
        Returns:
            BaseChatModel: LLM实例
        """
        config = self.get_llm_config(provider)
        
        # 合并kwargs到配置中
        for key, value in kwargs.items():
            if hasattr(config, key):
                setattr(config, key, value)
        
        if not config.api_key:
            raise ValueError(f"缺少 {config.provider.upper()} API Key")
        
        if config.provider == "openai":
            return ChatOpenAI(
                api_key=config.api_key,
                base_url=config.base_url,
                model=config.model,
                temperature=config.temperature,
                max_tokens=config.max_tokens
            )
        elif config.provider == "anthropic":
            return ChatAnthropic(
                api_key=config.api_key,
                base_url=config.base_url,
                model=config.model,
                temperature=config.temperature,
                max_tokens=config.max_tokens
            )
        elif config.provider == "azure":
            return ChatOpenAI(
                api_key=config.api_key,
                azure_endpoint=config.base_url,
                azure_deployment=config.deployment_name,
                api_version=config.api_version,
                temperature=config.temperature,
                max_tokens=config.max_tokens
            )
        elif config.provider == "custom":
            # 自定义LLM，使用OpenAI兼容接口
            return ChatOpenAI(
                api_key=config.api_key,
                base_url=config.base_url,
                model=config.model,
                temperature=config.temperature,
                max_tokens=config.max_tokens
            )
        else:
            raise ValueError(f"不支持的LLM提供商: {config.provider}")
    
    def get_database_config(self) -> Dict[str, str]:
        """获取数据库配置"""
        return {
            "database_url": os.getenv("DATABASE_URL", "sqlite:///data/langgraph_tutorial.db"),
            "checkpoint_db_url": os.getenv("CHECKPOINT_DB_URL", "sqlite:///data/checkpoints.db")
        }
    
    def get_workflow_config(self) -> Dict[str, Any]:
        """获取工作流配置"""
        return {
            "max_iterations": int(os.getenv("MAX_ITERATIONS", "10")),
            "default_timeout": int(os.getenv("DEFAULT_TIMEOUT", "300")),
            "enable_human_feedback": os.getenv("ENABLE_HUMAN_FEEDBACK", "true").lower() == "true"
        }
    
    def get_debug_config(self) -> Dict[str, Any]:
        """获取调试配置"""
        return {
            "debug_mode": os.getenv("DEBUG_MODE", "false").lower() == "true",
            "show_graph_visualization": os.getenv("SHOW_GRAPH_VISUALIZATION", "true").lower() == "true",
            "save_intermediate_states": os.getenv("SAVE_INTERMEDIATE_STATES", "true").lower() == "true"
        }
    
    def validate_config(self, provider: Optional[str] = None) -> bool:
        """
        验证配置是否完整
        
        Args:
            provider: LLM提供商
            
        Returns:
            bool: 配置是否有效
        """
        try:
            config = self.get_llm_config(provider)
            
            if not config.api_key:
                print(f"❌ 缺少 {config.provider.upper()} API Key")
                return False
            
            if config.provider == "azure" and not config.deployment_name:
                print("❌ Azure OpenAI 缺少 deployment_name")
                return False
            
            if config.provider == "custom" and not config.base_url:
                print("❌ 自定义LLM 缺少 base_url")
                return False
            
            print(f"✅ {config.provider.upper()} 配置验证通过")
            return True
            
        except Exception as e:
            print(f"❌ 配置验证失败: {e}")
            return False
    
    def print_config_summary(self):
        """打印配置摘要"""
        print("\n" + "="*50)
        print("📋 LangGraph 配置摘要")
        print("="*50)
        
        # LLM配置
        provider = os.getenv("DEFAULT_LLM_PROVIDER", "openai")
        config = self.get_llm_config(provider)
        
        print(f"🤖 LLM提供商: {config.provider.upper()}")
        print(f"📦 模型: {config.model}")
        print(f"🌡️  温度: {config.temperature}")
        print(f"📝 最大Token: {config.max_tokens}")
        
        if config.base_url:
            print(f"🔗 API地址: {config.base_url}")
        
        # 数据库配置
        db_config = self.get_database_config()
        print(f"💾 数据库: {db_config['database_url']}")
        
        # 工作流配置
        workflow_config = self.get_workflow_config()
        print(f"🔄 最大迭代: {workflow_config['max_iterations']}")
        print(f"👤 人工反馈: {'启用' if workflow_config['enable_human_feedback'] else '禁用'}")
        
        # 调试配置
        debug_config = self.get_debug_config()
        print(f"🐛 调试模式: {'启用' if debug_config['debug_mode'] else '禁用'}")
        
        print("="*50)


# 全局配置实例
config_manager = ConfigManager()


def get_llm(provider: Optional[str] = None, **kwargs) -> BaseChatModel:
    """
    获取LLM实例的便捷函数
    
    Args:
        provider: LLM提供商
        **kwargs: 额外配置参数
        
    Returns:
        BaseChatModel: LLM实例
    """
    return config_manager.create_llm(provider, **kwargs)


if __name__ == "__main__":
    # 配置测试
    config_manager.print_config_summary()
    
    # 验证配置
    if config_manager.validate_config():
        print("\n✅ 配置验证成功！")
        
        # 创建LLM实例测试
        try:
            llm = config_manager.create_llm()
            print(f"🎉 成功创建LLM实例: {type(llm).__name__}")
        except Exception as e:
            print(f"❌ 创建LLM实例失败: {e}")
    else:
        print("\n❌ 配置验证失败，请检查环境变量！") 