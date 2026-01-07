# 构建阶段
FROM golang:1.25-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /build

# 复制 go mod 文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o hle-agent ./cmd/agent/main.go

# 运行阶段
FROM alpine:latest

# 安装必要的运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && update-ca-certificates

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/hle-agent .
COPY --from=builder /build/config.yaml ./config.yaml

# 创建必要的目录
RUN mkdir -p /app/data /app/logs /app/results && \
    chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口（如果需要 HTTP API）
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release

# 运行应用
CMD ["./hle-agent"]

