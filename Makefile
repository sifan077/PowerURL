.PHONY: run build clean test help

# 默认目标
help:
	@echo "可用的命令:"
	@echo "  make run    - 运行服务器"
	@echo "  make build  - 构建二进制文件"
	@echo "  make clean  - 清理构建文件"
	@echo "  make test   - 运行测试"
	@echo "  make tidy   - 整理依赖"

# 运行服务器
run:
	go run cmd/server/main.go

# 构建二进制文件
build:
	go build -o bin/server.exe cmd/server/main.go

# 清理构建文件
clean:
	@if exist bin rmdir /s /q bin

# 运行测试
test:
	go test ./...

# 整理依赖
tidy:
	go mod tidy