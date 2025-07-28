# 变量定义
BINARY_NAME=mcp-server
BINARY_PATH=./bin/$(BINARY_NAME)
SOURCE_PATH=./cmd/$(BINARY_NAME)
GO_VERSION=$(shell go version | awk '{print $$3}')

# 默认目标
.PHONY: all
all: clean build

# 构建可执行文件
.PHONY: build
build:
	@echo "构建 $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -ldflags="-s -w" -o $(BINARY_PATH) $(SOURCE_PATH)
	@echo "✓ 构建完成: $(BINARY_PATH)"

# 开发模式构建（包含调试信息）
.PHONY: build-dev
build-dev:
	@echo "构建开发版本 $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -o $(BINARY_PATH) $(SOURCE_PATH)
	@echo "✓ 开发版本构建完成: $(BINARY_PATH)"

# 运行程序
.PHONY: run
run: build
	@echo "启动 $(BINARY_NAME)..."
	@$(BINARY_PATH)

# 直接运行（不构建）
.PHONY: run-direct
run-direct:
	@echo "直接运行 $(BINARY_NAME)..."
	@go run $(SOURCE_PATH)

# 清理构建文件
.PHONY: clean
clean:
	@echo "清理构建文件..."
	@rm -rf bin/
	@echo "✓ 清理完成"

# 格式化代码
.PHONY: fmt
fmt:
	@echo "格式化代码..."
	@go fmt ./...
	@echo "✓ 代码格式化完成"

# 代码检查
.PHONY: vet
vet:
	@echo "检查代码..."
	@go vet ./...
	@echo "✓ 代码检查完成"

# 下载依赖
.PHONY: tidy
tidy:
	@echo "整理依赖..."
	@go mod tidy
	@echo "✓ 依赖整理完成"

# 跨平台构建
.PHONY: build-cross
build-cross:
	@echo "跨平台构建..."
	@mkdir -p bin
	@echo "构建 Linux amd64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux-amd64 $(SOURCE_PATH)
	@echo "构建 Windows amd64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-windows-amd64.exe $(SOURCE_PATH)
	@echo "构建 macOS amd64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-darwin-amd64 $(SOURCE_PATH)
	@echo "构建 macOS arm64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o bin/$(BINARY_NAME)-darwin-arm64 $(SOURCE_PATH)
	@echo "✓ 跨平台构建完成"
	@ls -la bin/

# 显示帮助信息
.PHONY: help
help:
	@echo "可用的命令:"
	@echo "  build        - 构建可执行文件"
	@echo "  build-dev    - 构建开发版本（包含调试信息）"
	@echo "  build-cross  - 跨平台构建"
	@echo "  run          - 构建并运行程序"
	@echo "  run-direct   - 直接运行程序（不构建）"
	@echo "  clean        - 清理构建文件"
	@echo "  fmt          - 格式化代码"
	@echo "  vet          - 代码检查"
	@echo "  tidy         - 整理依赖"
	@echo "  help         - 显示此帮助信息"
	@echo ""
	@echo "Go版本: $(GO_VERSION)"

# 完整的开发流程
.PHONY: dev
dev: fmt vet tidy build-dev
	@echo "✓ 开发环境准备完成" 