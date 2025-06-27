# Quantix Makefile

.PHONY: help build test clean docker-build docker-run docker-stop format

# 默认目标
help:
	@echo "Quantix 构建和部署工具"
	@echo ""
	@echo "可用命令:"
	@echo "  build        - 构建应用"
	@echo "  test         - 运行测试"
	@echo "  clean        - 清理构建文件"
	@echo "  format       - 代码格式化"
	@echo "  docker-build - 构建Docker镜像"
	@echo "  docker-run   - 启动Docker服务"
	@echo "  docker-stop  - 停止Docker服务"
	@echo "  install      - 安装依赖"
	@echo "  run          - 运行应用"

# 构建应用
build:
	@echo "构建 Quantix 应用..."
	go build -o bin/quantix main.go
	@echo "构建完成: bin/quantix"

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./...
	@echo "测试完成"

# 运行特定测试
test-factors:
	@echo "运行技术指标测试..."
	go test -v ./test -run TestFactors

test-ml:
	@echo "运行机器学习测试..."
	go test -v ./test -run TestML

test-risk:
	@echo "运行风险管理测试..."
	go test -v ./test -run TestRisk

# 清理构建文件
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	rm -rf charts/
	rm -rf models/
	rm -rf uploads/
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

# 重启Docker服务
docker-restart:
	@echo "重启 Docker 服务..."
	docker-compose restart

# 数据库迁移
migrate:
	@echo "运行数据库迁移..."
	# 这里可以添加数据库迁移命令

# 生成文档
docs:
	@echo "生成文档..."
	godoc -http=:6060

# 性能测试
benchmark:
	@echo "运行性能测试..."
	go test -bench=. ./...

# 覆盖率测试
coverage:
	@echo "运行覆盖率测试..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

# 安全检查
security:
	@echo "运行安全检查..."
	gosec ./...

# 发布版本
release:
	@echo "发布新版本..."
	@read -p "请输入版本号 (例如: v1.0.0): " version; \
	git tag $$version; \
	git push origin $$version; \
	echo "版本 $$version 已发布"

# 开发环境设置
dev-setup:
	@echo "设置开发环境..."
	@echo "1. 安装 Go 1.22+"
	@echo "2. 安装 Docker 和 Docker Compose"
	@echo "3. 安装 gosec"
	@echo "4. 运行 'make install' 安装依赖"
	@echo "5. 运行 'make docker-run' 启动服务"

# 生产环境部署
deploy:
	@echo "部署到生产环境..."
	@echo "1. 构建生产镜像"
	make docker-build
	@echo "2. 推送镜像到仓库"
	@echo "3. 部署到生产服务器"
	@echo "部署完成"

# 监控
monitor:
	@echo "启动监控服务..."
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Quantix API: http://localhost:8080"

# 备份数据
backup:
	@echo "备份数据..."
	docker-compose exec postgres pg_dump -U quantix quantix > backup_$(shell date +%Y%m%d_%H%M%S).sql

# 恢复数据
restore:
	@echo "恢复数据..."
	@read -p "请输入备份文件名: " backup_file; \
	docker-compose exec -T postgres psql -U quantix quantix < $$backup_file 