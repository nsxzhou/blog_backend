.PHONY: dev build clean test

# 默认任务
default: dev

# 开发环境（使用air热加载）
dev:
	@if command -v air > /dev/null; then \
		air; \
	else \
		go install github.com/cosmtrek/air@latest && air; \
	fi

# 构建项目
build:
	@echo "构建项目..."
	@go build -o bin/blog-api main.go

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