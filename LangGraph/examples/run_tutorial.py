"""
LangGraph企业级Agent开发实战示例集合
运行所有章节的示例代码

使用方法：
python run_all_examples.py [章节号]

例如：
python run_all_examples.py 1    # 只运行第一章
python run_all_examples.py      # 运行所有章节
"""

import sys
import os
import asyncio
from pathlib import Path

# 添加项目根目录到路径
sys.path.append(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from utils import ColorfulLogger
from config import config_manager

def run_chapter_examples():
    """运行所有章节示例的主函数"""
    
    # 显示欢迎信息
    ColorfulLogger.header("🚀 LangGraph企业级Agent开发实战示例")
    ColorfulLogger.info("基于文章《企业级Agent开发实战(一)LangGraph快速入门》")
    ColorfulLogger.info("=" * 60)
    
    # 检查配置
    ColorfulLogger.step("检查配置...")
    config_manager.print_config_summary()
    
    if not config_manager.validate_config():
        ColorfulLogger.error("❌ 配置验证失败！请检查环境变量设置。")
        ColorfulLogger.info("💡 请确保设置了正确的API密钥：")
        ColorfulLogger.info("   - OPENAI_API_KEY 或 ANTHROPIC_API_KEY")
        ColorfulLogger.info("   - 可以复制 config.env.example 为 .env 并填入配置")
        return
    
    # 获取要运行的章节
    target_chapter = None
    if len(sys.argv) > 1:
        try:
            target_chapter = int(sys.argv[1])
            ColorfulLogger.info(f"🎯 只运行第 {target_chapter} 章示例")
        except ValueError:
            ColorfulLogger.warning("⚠️  参数格式错误，将运行所有章节")
    
    # 定义章节映射
    chapters = {
        1: ("基础概念", "01_basic_concepts"),
        2: ("图形API", "02_graph_api"),
        3: ("流式传输", "03_streaming"),
        4: ("记忆系统", "04_memory_system"),
        5: ("模型集成", "05_model_integration"),
        6: ("工具集成", "06_tool_integration"),
        7: ("人机交互", "07_human_in_the_loop"),
        8: ("MCP集成", "08_mcp_integration"),
        9: ("ReAct智能体", "09_react_agent")
    }
    
    # 运行指定章节或所有章节
    if target_chapter and target_chapter in chapters:
        chapters_to_run = {target_chapter: chapters[target_chapter]}
    else:
        chapters_to_run = chapters
    
    success_count = 0
    total_count = len(chapters_to_run)
    
    # 逐章节运行示例
    for chapter_num, (chapter_name, module_name) in chapters_to_run.items():
        ColorfulLogger.header(f"第{chapter_num}章：{chapter_name}")
        
        try:
            # 动态导入并运行模块
            module = __import__(module_name)
            
            if hasattr(module, 'main'):
                if module_name in ["08_mcp_integration", "09_react_agent"]:
                    # MCP集成和ReAct Agent需要异步运行
                    asyncio.run(module.main())
                else:
                    module.main()
                success_count += 1
                ColorfulLogger.success(f"✅ 第{chapter_num}章示例运行完成")
            else:
                ColorfulLogger.warning(f"⚠️  第{chapter_num}章模块没有main函数")
                
        except ImportError as e:
            ColorfulLogger.error(f"❌ 导入第{chapter_num}章模块失败: {e}")
        except Exception as e:
            ColorfulLogger.error(f"❌ 第{chapter_num}章示例运行失败: {e}")
        
        print("\n" + "="*80 + "\n")
    
    # 显示总结
    ColorfulLogger.header("📊 运行总结")
    ColorfulLogger.info(f"总章节数: {total_count}")
    ColorfulLogger.info(f"成功运行: {success_count}")
    ColorfulLogger.info(f"失败数量: {total_count - success_count}")
    
    if success_count == total_count:
        ColorfulLogger.success("🎉 所有示例运行成功！")
    else:
        ColorfulLogger.warning(f"⚠️  有 {total_count - success_count} 个章节运行失败")
    
    # 显示学习建议
    ColorfulLogger.info("\n💡 学习建议:")
    ColorfulLogger.info("1. 按顺序运行各章节示例，理解LangGraph的核心概念")
    ColorfulLogger.info("2. 尝试修改示例代码，观察不同参数的效果")
    ColorfulLogger.info("3. 结合文章内容，深入理解每个功能的应用场景")
    ColorfulLogger.info("4. 在自己的项目中应用这些模式和最佳实践")

def show_help():
    """显示帮助信息"""
    help_text = """
🔧 LangGraph示例运行器

使用方法：
  python run_all_examples.py [选项]

选项：
  无参数          运行所有章节示例
  [章节号]        运行指定章节（1-9）
  --help, -h     显示帮助信息

章节列表：
  1. 基础概念 - LangGraph核心概念和基础用法
  2. 图形API - 节点、边、状态的设计和使用
  3. 流式传输 - 实时观察智能体的执行过程
  4. 记忆系统 - 短期和长期记忆的实现
  5. 模型集成 - 多模型策略和性能监控
  6. 工具集成 - 工具调用和编排策略
  7. 人机交互 - 中断机制和协作界面
  8. MCP集成 - 外部系统连接和安全管理
  9. ReAct智能体 - 推理与行动结合的智能体

示例：
  python run_all_examples.py     # 运行所有示例
  python run_all_examples.py 1   # 只运行第1章
  python run_all_examples.py 5   # 只运行第5章

注意事项：
- 确保已正确配置API密钥（OPENAI_API_KEY 或 ANTHROPIC_API_KEY）
- 某些示例需要网络连接
- 建议按顺序学习各章节内容
"""
    print(help_text)

def check_environment():
    """检查运行环境"""
    ColorfulLogger.info("🔍 检查运行环境...")
    
    # 检查Python版本
    python_version = sys.version_info
    if python_version.major < 3 or (python_version.major == 3 and python_version.minor < 8):
        ColorfulLogger.error("❌ Python版本过低，需要Python 3.8+")
        return False
    
    ColorfulLogger.success(f"✅ Python版本: {python_version.major}.{python_version.minor}.{python_version.micro}")
    
    # 检查必要的包
    required_packages = [
        "langgraph",
        "langchain_core", 
        "langchain_openai",
        "typing_extensions"
    ]
    
    missing_packages = []
    for package in required_packages:
        try:
            __import__(package)
            ColorfulLogger.success(f"✅ {package}")
        except ImportError:
            missing_packages.append(package)
            ColorfulLogger.error(f"❌ {package} 未安装")
    
    if missing_packages:
        ColorfulLogger.error("❌ 缺少必要的包，请安装：")
        ColorfulLogger.info(f"pip install {' '.join(missing_packages)}")
        return False
    
    return True

def main():
    """主函数"""
    
    # 检查是否需要显示帮助
    if len(sys.argv) > 1 and sys.argv[1] in ['--help', '-h', 'help']:
        show_help()
        return
    
    # 检查环境
    if not check_environment():
        ColorfulLogger.error("❌ 环境检查失败")
        return
    
    # 运行示例
    try:
        run_chapter_examples()
    except KeyboardInterrupt:
        ColorfulLogger.warning("\n⚠️  用户中断执行")
    except Exception as e:
        ColorfulLogger.error(f"❌ 运行失败: {e}")

if __name__ == "__main__":
    main() 