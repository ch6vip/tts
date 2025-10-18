# TTS 语音合成服务

一个功能强大、易于使用的文本转语音 (TTS) 服务，基于 Microsoft Azure 语音服务构建，提供高质量的语音合成能力，并完美兼容 OpenAI TTS API。

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com/r/ch6vip/ch6vip-tts)

## ✨ 核心特性

- **高品质语音**: 基于 Microsoft Azure，提供自然流畅的语音合成。
- **多语言支持**: 支持超过100种语言和400多种声音。
- **长文本优化**: 智能分段和并发处理，轻松应对长文本。
- **OpenAI 兼容**: 完美适配 OpenAI `/v1/audio/speech` 接口。
- **现代化界面**: 简洁美观的 Web UI，方便在线使用。
- **轻量化部署**: 优化的 Docker 镜像，资源占用少。
- **智能缓存**: 内置缓存机制，提高响应速度，节省资源。

## 🚀 快速开始

### 前置要求

- Docker 20.10+

### 一键部署

```bash
# 克隆项目
git clone https://github.com/ch6vip/tts.git
cd tts

# 启动服务
docker-compose up -d
```

服务将在 `http://localhost:8081` 启动。

## 🐳 部署

### Docker Compose

这是最简单的部署方式。`docker-compose.yml` 文件已经包含在项目中。

```yaml
version: '3.8'
services:
  tts:
    image: ch6vip/ch6vip-tts:main
    ports:
      - "127.0.0.1:8081:8080"
    restart: always
```

### Docker Run

```bash
docker run -d \
  --name tts \
  -p 8081:8080 \
  --restart unless-stopped \
  ch6vip/ch6vip-tts:main
```

## 🛠️ API 使用

### 获取语音列表

```bash
curl "http://localhost:8081/voices"
```

### 文本转语音

```bash
curl "http://localhost:8081/tts?t=你好世界&v=zh-CN-XiaoxiaoNeural" -o output.mp3
```

### OpenAI 兼容接口

```bash
curl -X POST "http://localhost:8081/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tts-1",
    "input": "你好世界！",
    "voice": "alloy"
  }' \
  -o output.mp3
```

## ⚙️ 配置

可以通过 `configs/config.yaml` 文件或环境变量进行配置。

### 主要配置项

| 配置项 | 环境变量 | 描述 |
| --- | --- | --- |
| `server.port` | `SERVER_PORT` | 服务监听端口 |
| `tts.region` | `TTS_REGION` | Azure 区域 |
| `tts.api_key` | `TTS_API_KEY` | TTS API 密钥 |
| `openai.api_key` | `OPENAI_API_KEY` | OpenAI API 密钥 |
| `cache.enabled` | `CACHE_ENABLED` | 是否启用缓存 |

## 🏗️ 项目架构

```
tts/
├── cmd/api/          # 应用入口
├── internal/         # 内部核心逻辑
│   ├── http/         # HTTP 服务
│   ├── tts/          # TTS 核心服务
│   └── ...
├── web/              # 前端资源
├── configs/          # 配置文件
├── Dockerfile
└── docker-compose.yml
```

## 🛠️ 开发

### 本地开发

```bash
# 安装依赖
go mod download
npm install

# 编译前端
npm run build:css

# 运行服务
go run cmd/api/main.go
```

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。
