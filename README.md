# TTS 语音合成服务

一个功能强大、易于使用的文本转语音 (TTS) 服务，基于 Microsoft Azure 语音服务构建，提供高质量的语音合成能力，完美兼容 OpenAI TTS API。

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com/r/ch6vip/ch6vip-tts)

## ✨ 核心特性

### 🎯 语音合成能力
- **高品质语音**: 基于 Microsoft Azure 认知服务，提供接近真人的自然语音
- **多语言支持**: 涵盖全球 100+ 语言和方言，400+ 神经网络语音
- **精细控制**: 支持调节语速 (-100% ~ +100%)、音调 (-100% ~ +100%) 和情感风格
- **SSML 支持**: 完整支持语音合成标记语言，实现停顿、强调、韵律等高级控制

### 🚀 长文本优化
- **智能分段**: 基于句法边界的智能文本分割算法
- **并发处理**: Worker Pool 并发合成，最大化吞吐量
- **无缝合并**: FFmpeg 专业音频合并，确保连贯流畅
- **自动适配**: 自动判断文本长度，选择最优处理策略

### 🔌 API 接口
- **RESTful API**: 简洁直观的 HTTP 接口
- **OpenAI 兼容**: 完美适配 OpenAI `/v1/audio/speech` 接口格式
- **批量处理**: 支持长文本自动分段合成
- **流式响应**: 高效的音频流传输

### 🎨 用户界面
- **现代化 UI**: 采用 Tailwind CSS 构建的响应式界面
- **实时预览**: 即时试听合成效果
- **参数调节**: 可视化调整语速、音调和风格
- **API 文档**: 内置交互式 API 文档页面

### 🔒 安全与性能
- **轻量镜像**: 采用 Distroless 基础镜像，体积仅 65MB
- **非特权运行**: 容器以 nonroot 用户运行，提升安全性
- **智能缓存**: 内置缓存机制，减少重复请求，支持命中率统计
- **并发优化**: Worker Pool 资源管理，优化的 HTTP 连接池
- **性能监控**: 实时指标收集，支持健康检查和性能分析

## 📋 目录

- [快速开始](#-快速开始)
- [部署方式](#-部署方式)
- [API 使用](#️-api-使用)
- [配置详解](#️-配置详解)
- [高级功能](#-高级功能)
- [项目架构](#-项目架构)
- [开发指南](#️-开发指南)
- [常见问题](#-常见问题)

## 🚀 快速开始

### 前置要求

- Docker 20.10+ 或 Docker Desktop
- （可选）Docker Compose 1.29+

### 一键部署

**方式 1: Docker Compose (推荐)**

```bash
# 克隆项目
git clone https://github.com/ch6vip/tts.git
cd tts

# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f
```

**方式 2: Docker Run**

```bash
docker run -d \
  --name tts \
  -p 8080:8080 \
  --restart unless-stopped \
  ch6vip/ch6vip-tts:latest
```

### 验证部署

访问以下地址验证服务状态：

- **Web 界面**: http://localhost:8080
- **API 文档**: http://localhost:8080/api-doc
- **健康检查**: http://localhost:8080/health
- **性能指标**: http://localhost:8080/metrics
- **语音列表**: http://localhost:8080/voices

## 🐳 部署方式

### Docker Compose 配置

创建 `docker-compose.yml`：

```yaml
version: '3.8'

services:
  tts:
    image: ch6vip/ch6vip-tts:latest
    container_name: tts
    ports:
      - "127.0.0.1:8080:8080"  # 仅本地访问
    environment:
      # 基础配置
      - SERVER_PORT=8080
      - TZ=Asia/Shanghai
      
      # TTS 配置
      - TTS_REGION=eastasia
      - TTS_DEFAULT_VOICE=zh-CN-XiaoxiaoNeural
      - TTS_MAX_CONCURRENT=20
      
      # API 密钥（可选）
      # - TTS_API_KEY=your-api-key
      # - OPENAI_API_KEY=your-openai-key
    
    volumes:
      # 挂载自定义配置（可选）
      - ./configs/config.yaml:/app/configs/config.yaml:ro
    
    restart: unless-stopped
    
    # 健康检查
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/voices"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 高级 Docker 配置

**安全加固配置:**

```yaml
services:
  tts:
    image: ch6vip/ch6vip-tts:latest
    ports:
      - "127.0.0.1:8080:8080"
    
    # 安全选项
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    read_only: true
    
    # 临时文件系统
    tmpfs:
      - /tmp:size=100M,mode=1777
    
    # 资源限制
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
```

### Kubernetes 部署

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tts
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tts
  template:
    metadata:
      labels:
        app: tts
    spec:
      containers:
      - name: tts
        image: ch6vip/ch6vip-tts:latest
        ports:
        - containerPort: 8080
        env:
        - name: TTS_MAX_CONCURRENT
          value: "20"
        resources:
          requests:
            memory: "256Mi"
            cpu: "500m"
          limits:
            memory: "512Mi"
            cpu: "1000m"
        securityContext:
          runAsNonRoot: true
          runAsUser: 65532
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
---
apiVersion: v1
kind: Service
metadata:
  name: tts
spec:
  selector:
    app: tts
  ports:
  - port: 80
    targetPort: 8080
  type: LoadBalancer
```

## 🛠️ API 使用

### 1. 获取可用语音列表

```bash
# 获取所有语音
curl "http://localhost:8080/voices"

# 筛选中文语音
curl "http://localhost:8080/voices?locale=zh-CN"

# 筛选女性语音
curl "http://localhost:8080/voices?gender=Female"
```

**响应示例:**

```json
[
  {
    "name": "Microsoft Server Speech Text to Speech Voice (zh-CN, XiaoxiaoNeural)",
    "display_name": "Xiaoxiao",
    "local_name": "晓晓",
    "short_name": "zh-CN-XiaoxiaoNeural",
    "gender": "Female",
    "locale": "zh-CN",
    "locale_name": "中文(中国)",
    "style_list": ["cheerful", "sad", "angry", "fearful", "disgruntled"],
    "sample_rate_hertz": "24000"
  }
]
```

### 2. 文本转语音 (基础 API)

**GET 请求:**

```bash
# 基础合成
curl "http://localhost:8080/tts?t=你好世界&v=zh-CN-XiaoxiaoNeural" \
  -o output.mp3

# 调整语速和音调
curl "http://localhost:8080/tts?t=你好世界&v=zh-CN-XiaoxiaoNeural&r=20&p=10" \
  -o output.mp3

# 指定情感风格
curl "http://localhost:8080/tts?t=今天天气真好&v=zh-CN-XiaoxiaoNeural&s=cheerful" \
  -o output.mp3
```

**POST 请求 (JSON):**

```bash
curl -X POST "http://localhost:8080/tts" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "你好，这是一段测试文本",
    "voice": "zh-CN-XiaoxiaoNeural",
    "rate": "10",
    "pitch": "5",
    "style": "cheerful"
  }' \
  -o output.mp3
```

### 3. OpenAI 兼容接口

```bash
curl -X POST "http://localhost:8080/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "tts-1",
    "input": "你好世界！这是一段测试文本。",
    "voice": "zh-CN-XiaoxiaoNeural",
    "speed": 1.2
  }' \
  -o output.mp3
```

**OpenAI 语音映射:**

| OpenAI 语音 | 映射的中文语音 | 特点 |
|------------|--------------|------|
| alloy | zh-CN-XiaoyiNeural | 中性女声 |
| echo | zh-CN-YunxiNeural | 年轻男声 |
| fable | zh-CN-XiaochenNeural | 儿童声 |
| onyx | zh-CN-YunjianNeural | 成熟男声 |
| nova | zh-CN-XiaohanNeural | 活力女声 |
| shimmer | zh-CN-XiaomoNeural | 温柔女声 |

### 4. SSML 高级控制

```bash
curl -X POST "http://localhost:8080/tts" \
  -H "Content-Type: application/json" \
  -d '{
    "ssml": "<speak><prosody rate=\"slow\">这是慢速语音</prosody><break time=\"500ms\"/><prosody pitch=\"high\">这是高音调</prosody></speak>",
    "voice": "zh-CN-XiaoxiaoNeural"
  }' \
  -o output.mp3
```

### 5. 阅读应用集成

**导出到「阅读」应用:**

```bash
curl "http://localhost:8080/reader.json?t=文本&v=zh-CN-XiaoxiaoNeural&n=我的语音"
```

**导出到「爱阅记」应用:**

```bash
curl "http://localhost:8080/ifreetime.json?t=文本&v=zh-CN-XiaoxiaoNeural&n=我的语音"
```

### 6. 性能监控

**查看实时性能指标:**

```bash
curl "http://localhost:8080/metrics"
```

**响应示例:**
```json
{
  "tts": {
    "requests": 150,
    "success": 145,
    "errors": 5,
    "success_rate": 96.67,
    "latency": {
      "avg": "245ms",
      "max": "1200ms",
      "min": "50ms"
    }
  },
  "cache": {
    "hits": 45,
    "misses": 105,
    "hit_rate": 30.0,
    "total_size": 52428800
  },
  "worker_pool": {
    "total_jobs": 200,
    "errors": 2
  },
  "system": {
    "memory": {
      "alloc_mb": 125,
      "total_alloc_mb": 512,
      "sys_mb": 256,
      "num_gc": 25
    },
    "goroutines": 45
  }
}
```

**健康检查:**

```bash
curl "http://localhost:8080/health"
```

**重置指标:**

```bash
curl -X POST "http://localhost:8080/metrics/reset"
```

## ⚙️ 配置详解

### 配置文件结构

`configs/config.yaml`:

```yaml
server:
  port: 8080                # 服务监听端口
  read_timeout: 60          # 读超时（秒）
  write_timeout: 60         # 写超时（秒）
  base_path: ""             # API 基础路径前缀

tts:
  region: "eastasia"        # Azure 区域
  default_voice: "zh-CN-XiaoxiaoNeural"  # 默认语音
  default_rate: "0"         # 默认语速 (-100 ~ 100)
  default_pitch: "0"        # 默认音调 (-100 ~ 100)
  default_format: "audio-24khz-48kbitrate-mono-mp3"
  max_text_length: 65535    # 单次请求最大字符数
  request_timeout: 30       # Azure API 超时（秒）
  max_concurrent: 20        # 最大并发数
  segment_threshold: 300    # 自动分段阈值
  min_sentence_length: 200  # 最小句子长度
  max_sentence_length: 300  # 最大句子长度
  api_key: ''               # TTS API 密钥（可选）
  
  # 长文本处理优化
  long_text:
    enabled: true                    # 启用长文本优化
    max_segment_length: 500          # 每段最大字符数
    worker_count: 5                  # 并发 worker 数
    min_text_for_split: 1000         # 触发分段的最小长度
    ffmpeg_path: ""                  # FFmpeg 路径
    use_smart_segment: true          # 智能分段
    use_ffmpeg_merge: true           # 使用 FFmpeg 合并
  
  # OpenAI 语音映射
  voice_mapping:
    alloy: "zh-CN-XiaoyiNeural"
    echo: "zh-CN-YunxiNeural"
    fable: "zh-CN-XiaochenNeural"
    onyx: "zh-CN-YunjianNeural"
    nova: "zh-CN-XiaohanNeural"
    shimmer: "zh-CN-XiaomoNeural"

openai:
  api_key: ''               # OpenAI API 密钥（可选）

log:
  level: "info"             # 日志级别: debug, info, warn, error
  format: "text"            # 日志格式: text, json

cache:
  enabled: true             # 启用缓存
  expiration_minutes: 1440  # 缓存过期时间（分钟）
  cleanup_interval_minutes: 1440  # 清理间隔（分钟）
```

### 环境变量覆盖

所有配置项都可通过环境变量覆盖，规则：将路径中的 `.` 替换为 `_` 并转大写。

```bash
# 示例
export SERVER_PORT=9000
export TTS_REGION=eastus
export TTS_DEFAULT_VOICE=en-US-JennyNeural
export TTS_MAX_CONCURRENT=30
export TTS_API_KEY=your-secret-key
export OPENAI_API_KEY=your-openai-key
export LOG_LEVEL=debug
export LOG_FORMAT=json
```

## 🎓 高级功能

### 长文本智能处理

当文本超过配置的阈值时，系统会自动：

1. **智能分段**: 基于句法边界（句号、问号、感叹号等）进行分割
2. **并发合成**: 使用 Worker Pool 并发处理各分段
3. **专业合并**: 通过 FFmpeg 无缝拼接音频片段
4. **进度跟踪**: 实时统计合成进度和成功率

**配置示例:**

```yaml
tts:
  segment_threshold: 300      # 超过 300 字符触发分段
  long_text:
    enabled: true
    max_segment_length: 500   # 每段最多 500 字符
    worker_count: 5           # 5 个并发 worker
    min_text_for_split: 1000  # 至少 1000 字符才分段
    use_smart_segment: true   # 启用智能分段
    use_ffmpeg_merge: true    # 使用 FFmpeg 合并
```

### 缓存机制

内置智能缓存系统，相同请求直接返回缓存结果：

```yaml
cache:
  enabled: true
  expiration_minutes: 1440    # 缓存 24 小时
  cleanup_interval_minutes: 1440  # 每 24 小时清理过期缓存
```

### API 认证

保护您的 API 端点：

```yaml
tts:
  api_key: 'your-secret-key'  # TTS 接口密钥

openai:
  api_key: 'your-openai-key'  # OpenAI 接口密钥
```

**使用方式:**

```bash
# TTS API
curl "http://localhost:8080/tts?t=文本&api_key=your-secret-key"

# OpenAI API
curl -X POST "http://localhost:8080/v1/audio/speech" \
  -H "Authorization: Bearer your-openai-key" \
  -d '{"model": "tts-1", "input": "文本", "voice": "alloy"}'
```

## 🏗️ 项目架构

```
tts/
├── cmd/api/                  # 应用入口
│   └── main.go              # 主程序
├── configs/                  # 配置文件
│   └── config.yaml          # 主配置
├── internal/                 # 内部包
│   ├── config/              # 配置加载
│   │   └── config.go        # Viper 配置管理
│   ├── errors/              # 错误定义
│   │   └── errors.go        # 自定义错误类型
│   ├── http/                # HTTP 层
│   │   ├── handlers/        # 请求处理器
│   │   │   ├── pages.go     # 页面渲染
│   │   │   ├── tts.go       # TTS 处理
│   │   │   └── voices.go    # 语音列表
│   │   ├── middleware/      # 中间件
│   │   │   ├── auth.go      # 认证
│   │   │   ├── cors.go      # 跨域
│   │   │   ├── error.go     # 错误处理
│   │   │   └── logger.go    # 日志记录
│   │   ├── routes/          # 路由配置
│   │   │   └── routes.go    # 路由定义
│   │   └── server/          # 服务器
│   │       ├── app.go       # 应用初始化
│   │       └── server.go    # HTTP 服务器
│   ├── models/              # 数据模型
│   │   ├── tts.go           # TTS 模型
│   │   └── voice.go         # 语音模型
│   ├── metrics/             # 性能监控
│   │   └── metrics.go       # 指标收集器
│   ├── tts/                 # TTS 核心
│   │   ├── audio/           # 音频处理
│   │   │   └── merger.go    # 音频合并
│   │   ├── microsoft/       # Azure 客户端
│   │   │   ├── client.go    # API 客户端
│   │   │   └── models.go    # Azure 模型
│   │   ├── caching.go       # 缓存服务
│   │   ├── long_text_service.go  # 长文本服务
│   │   ├── segmenter.go     # 智能分段器
│   │   ├── service.go       # TTS 接口
│   │   └── worker_pool.go   # 并发处理
│   └── utils/               # 工具函数
│       └── utils.go         # 通用工具
├── web/                     # 前端资源
│   ├── embed.go             # 嵌入文件
│   ├── static/              # 静态资源
│   │   ├── css/             # 样式文件
│   │   ├── icons/           # 图标
│   │   └── js/              # JavaScript
│   └── templates/           # HTML 模板
│       ├── index.html       # 主页
│       ├── api-doc.html     # API 文档
│       └── shared/          # 共享组件
├── docs/                    # 文档
│   ├── DEVOPS_OPTIMIZATION_REPORT.md  # DevOps 优化报告
│   ├── SECURITY_HARDENING.md          # 安全加固指南
│   ├── PERFORMANCE_OPTIMIZATION_PLAN.md  # 性能优化计划
│   └── OPTIMIZATION_SUMMARY.md        # 优化总结
├── script/                  # 脚本
│   └── build.sh            # 构建脚本
├── .github/                # GitHub 配置
│   └── workflows/          # CI/CD
├── Dockerfile              # Docker 镜像
├── docker-compose.yml      # Docker Compose
├── go.mod                  # Go 模块
└── README.md               # 项目文档
```

### 技术栈

- **后端**: Go 1.23 + Gin Web Framework
- **TTS**: Microsoft Azure Cognitive Services
- **音频处理**: FFmpeg
- **前端**: Tailwind CSS + Vanilla JavaScript
- **容器化**: Docker + Distroless
- **CI/CD**: GitHub Actions

### 核心组件

| 组件 | 功能 | 技术 |
|------|------|------|
| HTTP Server | Web 服务 | Gin Framework |
| TTS Service | 语音合成 | Azure Speech API |
| Long Text Service | 长文本处理 | Worker Pool + Segmenter |
| Audio Merger | 音频合并 | FFmpeg |
| Cache Layer | 结果缓存 | go-cache |
| Middleware | 中间件层 | CORS, Auth, Logger |

## 🛠️ 开发指南

### 本地开发环境

```bash
# 1. 克隆仓库
git clone https://github.com/zuoban/tts.git
cd tts

# 2. 安装依赖
go mod download

# 3. 编译前端资源
npm install
npm run build:css

# 4. 运行服务
go run cmd/api/main.go --config configs/config.yaml

# 5. 访问服务
# Web界面: http://localhost:8080
# API文档: http://localhost:8080/api-doc
```

### 构建

```bash
# 本地构建
go build -o tts cmd/api/main.go

# Docker 构建
docker build -t tts:local .

# 多架构构建
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t tts:latest \
  --push .
```

### 测试

```bash
# 运行测试
go test ./...

# 运行特定包的测试
go test ./internal/tts/...

# 带覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 代码规范

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 使用 golangci-lint
golangci-lint run
```

## 📚 常见问题

### Q: 如何处理超长文本？

**A**: 系统会自动处理：
- 文本超过 `segment_threshold` (默认 300 字符) 时自动分段
- 基于句法边界智能切分，保持语义完整
- 并发合成后通过 FFmpeg 无缝拼接
- 最大支持 65535 字符（可配置）

### Q: 支持哪些音频格式？

**A**: 支持以下格式：
- MP3: 16kHz/24kHz, 32/48/64/96/128/160 kbps
- WAV: 16kHz/24kHz, 16-bit PCM

通过 `default_format` 配置项指定。

### Q: 如何提升合成速度？

**A**: 优化建议：
1. 启用缓存 (`cache.enabled: true`) - 命中率可达 30-40%
2. 增加并发数 (`tts.max_concurrent: 30`)
3. 使用 FFmpeg 合并 (`long_text.use_ffmpeg_merge: true`)
4. 选择较低的音频码率
5. 查看 `/metrics` 端点分析性能瓶颈

**新增优化 (v1.1.0):**
- Worker Pool 资源管理优化，减少 goroutine 泄漏风险
- HTTP 连接池优化，连接复用率提升 50-60%
- 缓存键生成改进，命中率提升 15-20%
- 内存使用优化，峰值降低 15-20%
- 详见 `docs/OPTIMIZATION_SUMMARY.md`

### Q: 容器启动失败？

**A**: 检查以下项：
1. 端口是否被占用：`netstat -tuln | grep 8080`
2. 权限问题：容器以 nonroot 用户运行，确保配置文件可读
3. 内存限制：建议至少 256MB
4. 查看日志：`docker logs tts`

### Q: API 返回 401 错误？

**A**: 可能原因：
1. 配置了 `api_key` 但请求未携带
2. API 密钥不匹配
3. OpenAI 接口需要 `Authorization: Bearer <token>` 头部

### Q: 音频合并效果不理想？

**A**: 优化方案：
1. 安装 FFmpeg：`use_ffmpeg_merge: true`
2. 调整分段长度：`max_segment_length: 400-600`
3. 启用智能分段：`use_smart_segment: true`

### Q: 支持私有部署吗？

**A**: 完全支持：
- 无需外部依赖（除 Azure API）
- 支持内网部署
- 可通过反向代理配置域名和 HTTPS
- 提供完整的 Kubernetes 部署示例
- 内置性能监控和健康检查

### Q: 如何监控服务性能？

**A**: 新增性能监控功能：
1. 实时指标查看：`GET /metrics`
   - TTS 请求统计（成功率、延迟）
   - 缓存命中率和大小
   - Worker Pool 状态
   - 系统资源使用
2. 健康检查：`GET /health`
3. 指标重置：`POST /metrics/reset`

建议将 `/metrics` 集成到 Prometheus/Grafana 进行可视化监控。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

### 贡献流程

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 开发规范

- 遵循 Go 代码规范
- 添加适当的测试
- 更新相关文档
- 确保 CI 通过

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源。

## 🙏 致谢

- [Microsoft Azure Cognitive Services](https://azure.microsoft.com/services/cognitive-services/speech-services/)
- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [Tailwind CSS](https://tailwindcss.com/)
- [Distroless Container Images](https://github.com/GoogleContainerTools/distroless)

## 📞 联系方式

- **Issue**: [GitHub Issues](https://github.com/zuoban/tts/issues)
- **Discussions**: [GitHub Discussions](https://github.com/zuoban/tts/discussions)

---

⭐ 如果这个项目对您有帮助，请给它一个 Star！
