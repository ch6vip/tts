# 阶段 1: 构建应用程序
FROM golang:1.24-alpine AS builder
ENV CGO_ENABLED=0
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY cmd ./cmd
COPY internal ./internal
COPY web ./web
COPY configs ./configs

# 构建应用程序（静态二进制文件，删除调试符号）
RUN go build -ldflags="-w -s" -o main ./cmd/api/main.go

# 阶段 2: 构建 ffmpeg 支持镜像
FROM alpine:latest AS ffmpeg-builder
RUN apk add --no-cache ffmpeg

# 阶段 3: 使用 distroless 作为最终镜像
FROM gcr.io/distroless/base-debian12

# 从 Alpine 复制 ffmpeg 和相关库
COPY --from=ffmpeg-builder /usr/bin/ffmpeg /usr/bin/ffmpeg
COPY --from=ffmpeg-builder /usr/lib /usr/lib

# 从构建器复制应用程序和资源
COPY --from=builder /app/main /app/main
COPY --from=builder /app/web /app/web
COPY --from=builder /app/configs /app/configs

# 设置工作目录
WORKDIR /app

# 设置时区环境变量
ENV TZ=Asia/Shanghai
ENV PATH="/usr/bin:$PATH"
ENV LD_LIBRARY_PATH="/usr/lib:$LD_LIBRARY_PATH"

# 暴露端口
EXPOSE 8080

# 运行应用程序
ENTRYPOINT ["/app/main"]