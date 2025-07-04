# 多阶段构建
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o quantix .

# 运行阶段
FROM alpine:latest

# 安装运行时依赖
RUN apk --no-cache add ca-certificates tzdata wget

# 设置工作目录
WORKDIR /app

# 从builder阶段复制二进制文件
COPY --from=builder /app/quantix .
COPY --from=builder /app/charts ./charts
COPY --from=builder /app/models ./models
COPY --from=builder /app/uploads ./uploads

# 创建必要的目录
RUN mkdir -p logs && \
    addgroup -g 1001 -S quantix && adduser -u 1001 -S quantix -G quantix && chown -R quantix:quantix /app

# 切换到非root用户
USER quantix

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 启动应用
CMD ["./quantix"] 