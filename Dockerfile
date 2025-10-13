# 使用 distroless 作为最终镜像
FROM gcr.io/distroless/base-debian12

# 复制应用程序和资源
ARG TARGETARCH
COPY main-${TARGETARCH} /app/main
COPY web /app/web
COPY configs /app/configs

# 设置工作目录
WORKDIR /app

# 设置时区环境变量
ENV TZ=Asia/Shanghai

# 暴露端口
EXPOSE 8080

# 运行应用程序
ENTRYPOINT ["/app/main"]