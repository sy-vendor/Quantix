# Quantix Makefile

.PHONY: help build clean docker-build docker-run docker-stop format install run

# 默认目标
help:
	@echo "Quantix 构建和部署工具"
	@echo ""
	@echo "开发命令:"
	@echo "  build        - 构建应用"
	@echo "  clean        - 清理构建文件"
	@echo "  format       - 代码格式化"
	@echo "  install      - 安装依赖"
	@echo "  run          - 运行应用"
	@echo ""
	@echo "Docker命令:"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 启动Docker服务"
	@echo "  docker-stop  - 停止Docker服务"
	@echo "  docker-logs  - 查看Docker日志"

# 构建应用
build:
	@echo "构建 Quantix 应用..."
	@mkdir -p bin
	go build -ldflags="-s -w" -o bin/quantix main.go
	@echo "构建完成: bin/quantix"

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf charts/
	rm -rf models/
	rm -rf uploads/
	rm -rf logs/
	@echo "清理完成"

# 代码格式化
format:
	@echo "格式化代码..."
	go fmt ./...
	gofmt -s -w .

# 安装依赖
install:
	@echo "安装依赖..."
	go mod download
	go mod tidy
	@echo "依赖安装完成"

# 运行应用
run:
	@echo "运行 Quantix 应用..."
	go run main.go

# 构建Docker镜像
docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t quantix:latest .

# 启动Docker服务
docker-run:
	@echo "启动 Docker 服务..."
	docker-compose up -d

# 停止Docker服务
docker-stop:
	@echo "停止 Docker 服务..."
	docker-compose down

# 查看Docker日志
docker-logs:
	docker-compose logs -f