# SSH MCP Server/Client Makefile
# 基于Go语言实现的MCP SSH服务器和客户端构建脚本

# 项目信息
PROJECT_NAME = ssh-mcp-go-jsonrpc
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
SERVER_BINARY = ssh-mcp-server
CLIENT_BINARY = ssh-mcp-client
SSE_SERVER_BINARY = ssh-mcp-sse-server
SSE_CLIENT_BINARY = ssh-mcp-sse-client
SERVER_SOURCE = ./cmd/server
CLIENT_SOURCE = ./cmd/client
SSE_SERVER_SOURCE = ./cmd/sse-server
SSE_CLIENT_SOURCE = ./cmd/sse-client

# 输出目录
BUILD_DIR = build
DIST_DIR = dist

# 默认目标
.PHONY: all
all: clean deps build

# 安装依赖
.PHONY: deps
deps:
	@echo "正在安装依赖..."
	$(GOMOD) tidy
	$(GOMOD) download

# 构建所有二进制文件
.PHONY: build
build: build-server build-client build-sse-server build-sse-client

# 构建服务器
.PHONY: build-server
build-server:
	@echo "正在构建MCP服务器..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_BINARY) $(SERVER_SOURCE)

# 构建客户端
.PHONY: build-client
build-client:
	@echo "正在构建MCP客户端..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_BINARY) $(CLIENT_SOURCE)

# 构建SSE服务器
.PHONY: build-sse-server
build-sse-server:
	@echo "正在构建MCP SSE服务器..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SSE_SERVER_BINARY) $(SSE_SERVER_SOURCE)

# 构建SSE客户端
.PHONY: build-sse-client
build-sse-client:
	@echo "正在构建MCP SSE客户端..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(SSE_CLIENT_BINARY) $(SSE_CLIENT_SOURCE)

# 交叉编译
.PHONY: build-cross
build-cross: clean deps
	@echo "正在进行交叉编译..."
	@mkdir -p $(DIST_DIR)
	
	# Linux amd64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(SERVER_BINARY)-linux-amd64 $(SERVER_SOURCE)
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(CLIENT_BINARY)-linux-amd64 $(CLIENT_SOURCE)
	
	# Linux arm64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(SERVER_BINARY)-linux-arm64 $(SERVER_SOURCE)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(CLIENT_BINARY)-linux-arm64 $(CLIENT_SOURCE)
	
	# macOS amd64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(SERVER_BINARY)-darwin-amd64 $(SERVER_SOURCE)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(CLIENT_BINARY)-darwin-amd64 $(CLIENT_SOURCE)
	
	# macOS arm64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(SERVER_BINARY)-darwin-arm64 $(SERVER_SOURCE)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(CLIENT_BINARY)-darwin-arm64 $(CLIENT_SOURCE)
	
	# Windows amd64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(SERVER_BINARY)-windows-amd64.exe $(SERVER_SOURCE)
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(CLIENT_BINARY)-windows-amd64.exe $(CLIENT_SOURCE)

# 运行测试
.PHONY: test
test:
	@echo "正在运行测试..."
	$(GOTEST) -v ./...

# 运行测试（带覆盖率）
.PHONY: test-coverage
test-coverage:
	@echo "正在运行测试（带覆盖率）..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

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

# 安装工具
.PHONY: install-tools
install-tools:
	@echo "正在安装开发工具..."
	$(GOGET) -u golang.org/x/tools/cmd/goimports
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

# 代码质量检查
.PHONY: lint
lint:
	@echo "正在进行代码质量检查..."
	golangci-lint run

# 清理构建文件
.PHONY: clean
clean:
	@echo "正在清理构建文件..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html

# 安装到系统
.PHONY: install
install: build
	@echo "正在安装到系统..."
	sudo cp $(BUILD_DIR)/$(SERVER_BINARY) /usr/local/bin/
	sudo cp $(BUILD_DIR)/$(CLIENT_BINARY) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(SERVER_BINARY)
	sudo chmod +x /usr/local/bin/$(CLIENT_BINARY)

# 卸载
.PHONY: uninstall
uninstall:
	@echo "正在从系统卸载..."
	sudo rm -f /usr/local/bin/$(SERVER_BINARY)
	sudo rm -f /usr/local/bin/$(CLIENT_BINARY)

# 运行服务器（开发模式）
.PHONY: run-server
run-server: build-server
	@echo "正在启动MCP服务器..."
	./$(BUILD_DIR)/$(SERVER_BINARY) -config config.yaml

# 运行客户端（演示模式）
.PHONY: run-client
run-client: build-client
	@echo "正在启动MCP客户端..."
	./$(BUILD_DIR)/$(CLIENT_BINARY) -server ./$(BUILD_DIR)/$(SERVER_BINARY) -args "-config config.yaml"

# 运行客户端（交互模式）
.PHONY: run-client-interactive
run-client-interactive: build-client
	@echo "正在启动MCP客户端（交互模式）..."
	./$(BUILD_DIR)/$(CLIENT_BINARY) -interactive -server ./$(BUILD_DIR)/$(SERVER_BINARY) -args "-config config.yaml"

# 创建发布包
.PHONY: package
package: build-cross
	@echo "正在创建发布包..."
	@mkdir -p $(DIST_DIR)/packages
	
	# 创建Linux包
	tar -czf $(DIST_DIR)/packages/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(SERVER_BINARY)-linux-amd64 $(CLIENT_BINARY)-linux-amd64
	tar -czf $(DIST_DIR)/packages/$(PROJECT_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(DIST_DIR) $(SERVER_BINARY)-linux-arm64 $(CLIENT_BINARY)-linux-arm64
	
	# 创建macOS包
	tar -czf $(DIST_DIR)/packages/$(PROJECT_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR) $(SERVER_BINARY)-darwin-amd64 $(CLIENT_BINARY)-darwin-amd64
	tar -czf $(DIST_DIR)/packages/$(PROJECT_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(DIST_DIR) $(SERVER_BINARY)-darwin-arm64 $(CLIENT_BINARY)-darwin-arm64
	
	# 创建Windows包
	zip -j $(DIST_DIR)/packages/$(PROJECT_NAME)-$(VERSION)-windows-amd64.zip $(DIST_DIR)/$(SERVER_BINARY)-windows-amd64.exe $(DIST_DIR)/$(CLIENT_BINARY)-windows-amd64.exe

# 显示帮助信息
.PHONY: help
help:
	@echo "SSH MCP Server/Client 构建脚本"
	@echo ""
	@echo "可用目标:"
	@echo "  all                    - 清理、安装依赖并构建所有二进制文件"
	@echo "  deps                   - 安装Go依赖"
	@echo "  build                  - 构建所有二进制文件"
	@echo "  build-server          - 构建MCP服务器"
	@echo "  build-client          - 构建MCP客户端"
	@echo "  build-cross           - 交叉编译所有平台"
	@echo "  test                   - 运行测试"
	@echo "  test-coverage         - 运行测试并生成覆盖率报告"
	@echo "  fmt                    - 格式化代码"
	@echo "  vet                    - 代码检查"
	@echo "  lint                   - 代码质量检查"
	@echo "  clean                  - 清理构建文件"
	@echo "  install               - 安装到系统"
	@echo "  uninstall             - 从系统卸载"
	@echo "  run-server            - 运行服务器（开发模式）"
	@echo "  run-client            - 运行客户端（演示模式）"
	@echo "  run-client-interactive - 运行客户端（交互模式）"
	@echo "  package               - 创建发布包"
	@echo "  install-tools         - 安装开发工具"
	@echo "  help                  - 显示此帮助信息"
	@echo ""
	@echo "示例:"
	@echo "  make all              - 完整构建"
	@echo "  make run-server       - 启动服务器"
	@echo "  make run-client       - 启动客户端演示" 
# 运行SSE服务器
.PHONY: run-sse-server
run-sse-server: build-sse-server
	@echo "正在启动MCP SSE服务器..."
	./$(BUILD_DIR)/$(SSE_SERVER_BINARY) -config config.yaml

# 运行SSE客户端
.PHONY: run-sse-client
run-sse-client: build-sse-client
	@echo "正在启动MCP SSE客户端..."
	./$(BUILD_DIR)/$(SSE_CLIENT_BINARY) -server http://localhost:8000 -mode demo

# SSE演示：同时启动服务器和客户端
.PHONY: demo-sse
demo-sse: build-sse-server build-sse-client
	@echo "启动SSE演示..."
	@echo "在一个终端运行: make run-sse-server"
	@echo "在另一个终端运行: make run-sse-client"

