.PHONY: dev build build-with-version install clean test run serve create-admin help

# 变量定义
APP_NAME = blog-api
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME = $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
LDFLAGS = -X 'github.com/nsxzhou1114/blog-api/cmd.Version=$(VERSION)' \
          -X 'github.com/nsxzhou1114/blog-api/cmd.GitCommit=$(GIT_COMMIT)' \
          -X 'github.com/nsxzhou1114/blog-api/cmd.BuildTime=$(BUILD_TIME)'

# 默认任务
default: help

# 开发环境（使用air热加载）
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		go install github.com/cosmtrek/air@latest && air; \
	fi

# 构建项目（不带版本信息）
build:
	@echo "构建项目..."
	@go build -o bin/$(APP_NAME) main.go
	@echo "构建完成: bin/$(APP_NAME)"

# 构建项目（带版本信息）
build-with-version:
	@echo "构建项目（带版本信息）..."
	@go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) main.go
	@echo "构建完成: bin/$(APP_NAME)"
	@echo "版本: $(VERSION)"
	@echo "提交: $(GIT_COMMIT)"
	@echo "构建时间: $(BUILD_TIME)"

# 为 Linux 构建项目
build-linux:
	@echo "为 Linux 构建项目..."
	@GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME) main.go
	@echo "构建完成: bin/$(APP_NAME)"

# 为 Linux 构建项目（带版本信息）
build-linux-with-version:
	@echo "为 Linux 构建项目（带版本信息）..."
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(APP_NAME) main.go
	@echo "构建完成: bin/$(APP_NAME)"
	@echo "版本: $(VERSION)"
	@echo "提交: $(GIT_COMMIT)"
	@echo "构建时间: $(BUILD_TIME)"

# 安装到系统路径
install: build-with-version
	@echo "安装到系统路径..."
	@sudo mv bin/$(APP_NAME) /usr/local/bin/
	@echo "安装完成，现在可以在任何地方使用 '$(APP_NAME)' 命令"

# 运行HTTP服务
serve: build
	@echo "启动HTTP服务..."
	@./bin/$(APP_NAME) serve

# 运行服务（直接运行）
run:
	@echo "直接运行服务..."
	@go run main.go serve

# 创建管理员用户
create-admin: build
	@echo "创建管理员用户..."
	@./bin/$(APP_NAME) user create-admin

# 运行测试
test:
	@echo "运行测试..."
	@go test -v ./...

# 清理临时文件
clean:
	@echo "清理临时文件..."
	@rm -rf tmp/
	@rm -rf bin/
	@echo "完成清理"

# 显示帮助信息
help:
	@echo "博客API项目 Makefile"
	@echo ""
	@echo "可用的命令:"
	@echo "  make dev                - 启动开发环境（热重载）"
	@echo "  make build              - 构建项目"
	@echo "  make build-with-version - 构建项目（带版本信息）"
	@echo "  make install            - 构建并安装到系统路径"
	@echo "  make serve              - 构建并启动HTTP服务"
	@echo "  make run                - 直接运行HTTP服务"
	@echo "  make create-admin       - 构建并创建管理员用户"
	@echo "  make test               - 运行测试"
	@echo "  make clean              - 清理临时文件"
	@echo ""
	@echo "命令行工具使用示例:"
	@echo "  ./bin/$(APP_NAME) --help              - 显示帮助"
	@echo "  ./bin/$(APP_NAME) version             - 显示版本信息"
	@echo "  ./bin/$(APP_NAME) serve               - 启动HTTP服务"
	@echo ""
	@echo "用户管理:"
	@echo "  ./bin/$(APP_NAME) user create-admin   - 创建管理员用户"
	@echo "  ./bin/$(APP_NAME) user list           - 列出用户"
	@echo "  ./bin/$(APP_NAME) user reset-password <username> - 重置用户密码"
	@echo "  ./bin/$(APP_NAME) user update-status <username> <status> - 更新用户状态"
	@echo ""
	@echo "数据库管理:"
	@echo "  ./bin/$(APP_NAME) db init-tables      - 初始化数据库表"
	@echo "  ./bin/$(APP_NAME) db export-mysql users users.json - 导出用户数据"
	@echo "  ./bin/$(APP_NAME) db import-mysql users users.json - 导入用户数据"
	@echo "  ./bin/$(APP_NAME) db sync-es articles - 同步文章到ES"
	@echo "  ./bin/$(APP_NAME) db cleanup          - 清理过期数据"
	@echo ""
	@echo "系统统计:"
	@echo "  ./bin/$(APP_NAME) stats system        - 系统统计信息"
	@echo "  ./bin/$(APP_NAME) stats users         - 用户统计信息"
	@echo "  ./bin/$(APP_NAME) stats articles      - 文章统计信息"
	@echo "  ./bin/$(APP_NAME) stats db-status     - 数据库状态"
	@echo ""
	@echo "配置:"
	@echo "  ./bin/$(APP_NAME) -c /path/to/config serve - 指定配置文件路径" 

