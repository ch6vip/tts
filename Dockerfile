# 定义构建器阶段
FROM golang:1.22-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go.mod 和 go.sum 文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 为目标平台构建应用程序
# CGO_ENABLED=0 禁用 CGO，以确保二进制文件是静态链接的
# -ldflags="-s -w" 剥离调试信息，减小二进制文件大小
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /app/main ./cmd/api

# 使用 distroless 作为最终镜像
FROM gcr.io/distroless/base-debian12

# 从构建器阶段复制应用程序和资源
COPY --from=builder /app/main /app/main
COPY configs /app/configs

# 设置工作目录
WORKDIR /app

# 设置时区环境变量
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

# 运行应用程序
ENTRYPOINT ["/app/main"]