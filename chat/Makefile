# MCP-OpenAI Integration Makefile
# 集成OpenAI API和MCP工具的智能聊天助手构建脚本

# 项目信息
PROJECT_NAME = mcp-openai-integration
VERSION = 1.0.0
BUILD_TIME = $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go相关配置
GO = go
GOCMD = $(GO)
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOMOD = $(GOCMD) mod

# 构建标志
LDFLAGS = -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitCommit=$(GIT_COMMIT)"

# 目标文件
BINARY = mcp-openai-chat
SOURCE = .

# 输出目录
BUILD_DIR = build
DIST_DIR = dist

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	@echo "正在安装依赖（OpenAI+MCP集成）..."
	$(GOMOD) tidy
	$(GOMOD) download

# 构建二进制文件
.PHONY: build
build:
	@echo "正在构建MCP-OpenAI集成应用..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY) $(SOURCE)

# 交叉编译
.PHONY: build-cross
build-cross: clean deps
	@echo "正在进行交叉编译（OpenAI+MCP集成）..."
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 $(SOURCE)
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-arm64 $(SOURCE)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-amd64 $(SOURCE)
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-arm64 $(SOURCE)
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe $(SOURCE)

# 运行测试
.PHONY: test
test:
	@echo "正在运行测试..."
	$(GOTEST) -v ./...

# 代码格式化
.PHONY: fmt
fmt:
	@echo "正在格式化代码..."
	$(GOCMD) fmt ./...

# 代码检查
.PHONY: vet
vet:
	@echo "正在进行代码检查..."
	$(GOCMD) vet ./...

# 清理构建文件
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

# 构建MCP服务器依赖
.PHONY: build-mcp-servers
build-mcp-servers:
	@echo "正在构建MCP服务器依赖..."
	@if [ -d "../go_jsonrpc" ]; then \
		echo "构建JSON-RPC MCP服务器..."; \
		cd ../go_jsonrpc && make build-server; \
	else \
		echo "警告: go_jsonrpc目录不存在"; \
	fi
	@if [ -d "../go-sdk" ]; then \
		echo "构建SDK MCP服务器..."; \
		cd ../go-sdk && make build-server; \
	else \
		echo "警告: go-sdk目录不存在"; \
	fi

# 运行应用（交互模式）
.PHONY: run
run: build
	@echo "正在启动MCP-OpenAI聊天助手（交互模式）..."
	@echo "请确保已设置OPENAI_API_KEY环境变量"
	./$(BUILD_DIR)/$(BINARY) -config config.yaml -interactive

# 运行应用（单次消息模式）
.PHONY: run-message
run-message: build
	@echo "正在启动MCP-OpenAI聊天助手（单次消息模式）..."
	@echo "请确保已设置OPENAI_API_KEY环境变量"
	./$(BUILD_DIR)/$(BINARY) -config config.yaml -message "帮我检查本地系统信息"

# 运行应用（包含MCP服务器构建）
.PHONY: run-full
run-full: build-mcp-servers build
	@echo "正在启动完整的MCP-OpenAI集成环境..."
	@echo "请确保已设置OPENAI_API_KEY环境变量"
	./$(BUILD_DIR)/$(BINARY) -config config.yaml -interactive

# 检查环境
.PHONY: check-env
check-env:
	@echo "检查环境配置..."
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "❌ 错误: 未设置OPENAI_API_KEY环境变量"; \
		echo "请运行: export OPENAI_API_KEY=your_api_key"; \
		exit 1; \
	else \
		echo "✅ OPENAI_API_KEY已设置"; \
	fi
	@if [ -f "config.yaml" ]; then \
		echo "✅ 配置文件存在"; \
	else \
		echo "❌ 错误: config.yaml文件不存在"; \
		exit 1; \
	fi

# 安装应用到系统
.PHONY: install
install: build
	@echo "正在安装MCP-OpenAI聊天助手..."
	sudo cp $(BUILD_DIR)/$(BINARY) /usr/local/bin/
	@echo "✅ 安装完成，可以使用 '$(BINARY)' 命令启动"

# 卸载应用
.PHONY: uninstall
uninstall:
	@echo "正在卸载MCP-OpenAI聊天助手..."
	sudo rm -f /usr/local/bin/$(BINARY)
	@echo "✅ 卸载完成"

# 创建发布包
.PHONY: release
release: build-cross
	@echo "正在创建发布包..."
	@mkdir -p $(DIST_DIR)/release
	
	# 创建各平台发布包
	@for platform in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64; do \
		echo "创建 $$platform 发布包..."; \
		mkdir -p $(DIST_DIR)/release/$(PROJECT_NAME)-$(VERSION)-$$platform; \
		if [ "$$platform" = "windows-amd64" ]; then \
			cp $(DIST_DIR)/$(BINARY)-$$platform.exe $(DIST_DIR)/release/$(PROJECT_NAME)-$(VERSION)-$$platform/$(BINARY).exe; \
		else \
			cp $(DIST_DIR)/$(BINARY)-$$platform $(DIST_DIR)/release/$(PROJECT_NAME)-$(VERSION)-$$platform/$(BINARY); \
		fi; \
		cp config.yaml $(DIST_DIR)/release/$(PROJECT_NAME)-$(VERSION)-$$platform/; \
		cp README.md $(DIST_DIR)/release/$(PROJECT_NAME)-$(VERSION)-$$platform/ 2>/dev/null || echo "README.md not found"; \
		cd $(DIST_DIR)/release && tar -czf $(PROJECT_NAME)-$(VERSION)-$$platform.tar.gz $(PROJECT_NAME)-$(VERSION)-$$platform; \
		cd ../..; \
	done
	
	@echo "✅ 发布包创建完成，位于 $(DIST_DIR)/release/"

# 显示帮助信息
.PHONY: help
help:
	@echo "MCP-OpenAI Integration 构建脚本"
	@echo ""
	@echo "可用目标:"
	@echo "  all                - 清理、安装依赖并构建应用"
	@echo "  deps               - 安装Go依赖"
	@echo "  build              - 构建应用"
	@echo "  build-cross        - 交叉编译所有平台"
	@echo "  build-mcp-servers  - 构建MCP服务器依赖"
	@echo "  test               - 运行测试"
	@echo "  fmt                - 格式化代码"
	@echo "  vet                - 代码检查"
	@echo "  clean              - 清理构建文件"
	@echo "  run                - 运行应用（交互模式）"
	@echo "  run-message        - 运行应用（单次消息模式）"
	@echo "  run-full           - 运行完整环境（包含MCP服务器）"
	@echo "  check-env          - 检查环境配置"
	@echo "  install            - 安装到系统"
	@echo "  uninstall          - 从系统卸载"
	@echo "  release            - 创建发布包"
	@echo "  help               - 显示此帮助信息"
	@echo ""
	@echo "环境要求:"
	@echo "  - Go 1.21+"
	@echo "  - OPENAI_API_KEY环境变量"
	@echo "  - MCP服务器（可选，通过build-mcp-servers构建）"
	@echo ""
	@echo "示例:"
	@echo "  make check-env     - 检查环境"
	@echo "  make run-full      - 完整运行"
	@echo "  make release       - 创建发布包" 