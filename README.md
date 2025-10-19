# TTS 语音合成服务

<div align="center">

一个高性能、企业级的文本转语音 (TTS) 服务，基于 Microsoft Azure 认知服务构建，提供专业级语音合成能力，完美兼容 OpenAI TTS API 接口规范。

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=flat&logo=docker)](https://hub.docker.com/r/ch6vip/ch6vip-tts)

[功能特性](#-核心特性) • [快速开始](#-快速开始) • [API文档](#-api-使用) • [配置说明](#️-配置说明) • [架构设计](#️-项目架构)

</div>

---

## ✨ 核心特性

### 🎯 核心功能
- **🎤 高品质语音合成**: 基于 Microsoft Azure 认知服务，提供自然流畅、接近真人的语音效果
- **🌍 多语言多音色**: 支持 100+ 种语言和 400+ 种不同风格的语音音色
- **📝 长文本智能处理**: 
  - 智能分段算法，基于句子边界进行优化切分
  - 并发处理引擎，支持多达 50 个并发任务
  - FFmpeg 音频无缝合并，确保连贯流畅的播放体验
  - 适配超长文本（支持 65535+ 字符）

### 🔌 API 兼容性
- **OpenAI API 完全兼容**: 无缝对接 OpenAI `/v1/audio/speech` 接口，零成本迁移
- **多种接口格式**: 支持 RESTful API、URL 参数、JSON 格式等多种调用方式
- **灵活的认证机制**: 支持 API Key 认证，确保服务安全

### ⚡ 性能优化
- **智能缓存系统**: 
  - 基于内容哈希的缓存策略
  - 支持缓存大小限制和自动清理
  - 可配置的过期时间和清理周期
- **并发工作池**: 高效的任务调度和资源管理
- **性能监控**: 内置 Prometheus 兼容的 metrics 端点

### 🎨 用户界面
- **现代化 Web UI**: 基于 Tailwind CSS 构建的简洁美观界面
- **在线调试工具**: 实时测试不同语音参数和效果
- **API 文档页面**: 交互式 API 文档，方便开发者集成

### 🚀 部署友好
- **轻量级容器**: 优化的 Docker 镜像，体积小，启动快
- **零依赖部署**: 单一二进制文件，开箱即用
- **灵活配置**: 支持配置文件、环境变量多种配置方式

---

## 🚀 快速开始

### 前置要求

- **Docker** 20.10+ 或 **Docker Compose** 2.0+
- （可选）**FFmpeg** - 用于长文本音频合并优化

### 方式一：Docker Compose（推荐）

1. **克隆项目**
```bash
git clone https://github.com/ch6vip/tts.git
cd tts
```

2. **启动服务**
```bash
docker-compose up -d
```

3. **访问服务**
- Web UI: http://localhost:8081
- API 文档: http://localhost:8081/api-doc
- Health Check: http://localhost:8081/health

### 方式二：Docker Run

```bash
docker run -d \
  --name tts-service \
  -p 8081:8080 \
  -e TTS_API_KEY=your_api_key \
  -e TTS_REGION=eastasia \
  --restart unless-stopped \
  ch6vip/ch6vip-tts:main
```

### 方式三：本地编译运行

```bash
# 1. 克隆项目
git clone https://github.com/ch6vip/tts.git
cd tts

# 2. 安装依赖
go mod download
npm install

# 3. 编译前端资源
npm run build:css

# 4. 编译并运行
go run cmd/api/main.go

# 或构建二进制文件
go build -o tts cmd/api/main.go
./tts
```

---

## 🛠️ API 使用

### 1. 获取可用语音列表

```bash
curl "http://localhost:8081/voices"
```

**响应示例：**
```json
[
  {
    "ShortName": "zh-CN-XiaoxiaoNeural",
    "DisplayName": "晓晓",
    "LocalName": "晓晓",
    "Gender": "Female",
    "Locale": "zh-CN",
    "StyleList": ["general", "assistant", "chat"]
  }
]
```

### 2. 简单文本转语音

**GET 请求：**
```bash
curl "http://localhost:8081/tts?t=你好世界&v=zh-CN-XiaoxiaoNeural" -o output.mp3
```

**POST 请求：**
```bash
curl -X POST "http://localhost:8081/tts" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "你好世界",
    "voice": "zh-CN-XiaoxiaoNeural",
    "rate": "0",
    "pitch": "0"
  }' \
  -o output.mp3
```

### 3. OpenAI 兼容接口

```bash
curl -X POST "http://localhost:8081/v1/audio/speech" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your_api_key" \
  -d '{
    "model": "tts-1",
    "input": "欢迎使用 TTS 服务！这是一个功能强大的文本转语音解决方案。",
    "voice": "alloy",
    "speed": 1.0
  }' \
  -o output.mp3
```

**OpenAI 语音映射：**
| OpenAI Voice | Microsoft Voice | 描述 |
|-------------|----------------|------|
| `alloy` | zh-CN-XiaoyiNeural | 中性女声 |
| `echo` | zh-CN-YunxiNeural | 年轻男声 |
| `fable` | zh-CN-XiaochenNeural | 儿童声音 |
| `onyx` | zh-CN-YunjianNeural | 成熟男声 |
| `nova` | zh-CN-XiaohanNeural | 活力女声 |
| `shimmer` | zh-CN-XiaomoNeural | 温柔女声 |

### 4. 长文本处理

服务会自动检测文本长度并智能处理：
- 文本 ≤ 1000 字符：单次合成
- 文本 > 1000 字符：自动分段并发处理

```bash
curl -X POST "http://localhost:8081/tts" \
  -H "Content-Type: application/json" \
  -d '{
    "text": "这是一段很长的文本...(可以是数千字)",
    "voice": "zh-CN-XiaoxiaoNeural"
  }' \
  -o long_output.mp3
```

### 5. 高级 SSML 控制

```bash
curl -X POST "http://localhost:8081/tts" \
  -H "Content-Type: application/json" \
  -d '{
    "ssml": "<speak><prosody rate=\"+20%\" pitch=\"+10%\">快速且高音调的语音</prosody></speak>",
    "voice": "zh-CN-XiaoxiaoNeural"
  }' \
  -o ssml_output.mp3
```

---

## ⚙️ 配置说明

### 配置文件位置

服务按以下优先级查找配置文件：
1. 命令行参数：`./tts -config /path/to/config.yaml`
2. 项目目录：`./configs/config.yaml`
3. 系统目录：`/etc/tts/config.yaml`
4. 嵌入默认配置（无需外部文件）

### 核心配置项

#### 服务器配置
```yaml
server:
  port: 8080              # 服务监听端口
  read_timeout: 60        # 读取超时（秒）
  write_timeout: 60       # 写入超时（秒）
  base_path: ""          # API 基础路径前缀
```

#### TTS 服务配置
```yaml
tts:
  api_key: ""                          # Azure TTS API 密钥 *必填*
  region: "eastasia"                   # Azure 服务区域
  default_voice: "zh-CN-XiaoxiaoNeural" # 默认语音
  default_format: "audio-24khz-48kbitrate-mono-mp3"
  max_text_length: 65535               # 单次请求最大字符数
  request_timeout: 30                  # 请求超时（秒）
  max_concurrent: 20                   # 最大并发请求数
```

#### 长文本处理配置
```yaml
tts:
  long_text:
    enabled: true                # 启用长文本优化
    max_segment_length: 500      # 每段最大字符数（建议 400-600）
    worker_count: 5              # 并发工作线程数（建议 3-10）
    min_text_for_split: 1000     # 触发分段的最小文本长度
    ffmpeg_path: ""             # FFmpeg 路径（留空使用系统 PATH）
    use_smart_segment: true      # 启用智能分段（基于句子边界）
    use_ffmpeg_merge: true       # 使用 FFmpeg 合并（推荐）
```

#### 缓存配置
```yaml
cache:
  enabled: true                      # 启用缓存
  expiration_minutes: 1440           # 缓存过期时间（分钟），默认 1 天
  cleanup_interval_minutes: 1440     # 清理间隔（分钟）
  max_total_size: 1073741824        # 最大缓存大小（字节），0 表示不限制
```

#### OpenAI 兼容配置
```yaml
openai:
  api_key: ""  # OpenAI API 密钥验证（可选）
```

#### 日志配置
```yaml
log:
  level: "info"    # 日志级别: trace, debug, info, warn, error, fatal
  format: "text"   # 日志格式: text, json
```

### 环境变量

所有配置项都可以通过环境变量覆盖，使用下划线连接，例如：

```bash
export SERVER_PORT=8080
export TTS_API_KEY=your_azure_api_key
export TTS_REGION=eastasia
export CACHE_ENABLED=true
export LOG_LEVEL=debug
```

---

## 🏗️ 项目架构

### 目录结构

```
tts/
├── cmd/
│   └── api/
│       └── main.go              # 应用程序入口
├── internal/                    # 内部包（私有代码）
│   ├── config/                  # 配置管理
│   │   └── config.go           # 配置加载和验证
│   ├── http/                    # HTTP 服务层
│   │   ├── handlers/           # 请求处理器
│   │   │   ├── tts.go         # TTS API 处理
│   │   │   ├── voices.go      # 语音列表处理
│   │   │   ├── pages.go       # 页面渲染
│   │   │   └── metrics.go     # 性能指标
│   │   ├── middleware/         # 中间件
│   │   │   ├── auth.go        # 认证中间件
│   │   │   ├── cors.go        # CORS 处理
│   │   │   ├── logger.go      # 日志中间件
│   │   │   └── error.go       # 错误处理
│   │   ├── routes/             # 路由配置
│   │   │   └── routes.go      # 路由注册
│   │   └── server/             # 服务器管理
│   │       ├── app.go         # 应用初始化
│   │       └── server.go      # HTTP 服务器
│   ├── tts/                     # TTS 核心服务
│   │   ├── service.go          # 服务接口定义
│   │   ├── long_text_service.go # 长文本处理服务
│   │   ├── segmenter.go        # 文本分段器
│   │   ├── worker_pool.go      # 并发工作池
│   │   ├── caching.go          # 缓存服务
│   │   ├── microsoft/          # Microsoft Azure TTS 客户端
│   │   │   ├── client.go      # HTTP 客户端
│   │   │   └── models.go      # 数据模型
│   │   └── audio/              # 音频处理
│   │       └── merger.go      # 音频合并器
│   ├── models/                  # 数据模型
│   │   ├── tts.go             # TTS 请求/响应模型
│   │   └── voice.go           # 语音模型
│   ├── metrics/                 # 性能指标
│   │   └── metrics.go         # Prometheus 指标
│   ├── errors/                  # 错误处理
│   │   └── errors.go          # 自定义错误类型
│   └── utils/                   # 工具函数
│       └── utils.go           # 通用工具
├── configs/                     # 配置文件
│   ├── config.yaml            # 默认配置
│   └── embed.go               # 配置文件嵌入
├── web/                         # Web 前端资源
│   ├── templates/              # HTML 模板
│   │   ├── index.html         # 主页
│   │   ├── api-doc.html       # API 文档页
│   │   └── shared/            # 共享模板
│   ├── static/                 # 静态资源
│   │   ├── css/               # 样式文件
│   │   ├── js/                # JavaScript 文件
│   │   └── icons/             # 图标资源
│   └── embed.go               # 静态资源嵌入

├── script/                      # 构建脚本
│   └── build.sh               # 编译脚本
├── docker-compose.yml           # Docker Compose 配置
├── Dockerfile.optimized         # 优化的 Dockerfile
├── go.mod                       # Go 模块定义
├── go.sum                       # Go 依赖校验
├── package.json                 # Node.js 依赖
└── tailwind.config.js           # Tailwind CSS 配置
```

### 技术栈

- **后端**: Go 1.24+, Gin Web Framework
- **TTS 引擎**: Microsoft Azure 认知服务
- **日志**: Zerolog (高性能结构化日志)
- **配置**: Viper (支持多种配置源)
- **缓存**: go-cache (内存缓存)
- **音频处理**: FFmpeg (可选)
- **前端**: Tailwind CSS, Vanilla JavaScript
- **容器化**: Docker, Docker Compose

### 核心组件

#### 1. 长文本处理服务 ([`LongTextTTSService`](internal/tts/long_text_service.go:16))
- **智能分段**: 基于句子边界的智能文本切分
- **并发处理**: 工作池模式，支持多任务并行处理
- **音频合并**: FFmpeg 无缝合并或简单二进制拼接

#### 2. 缓存系统 ([`CachingService`](internal/tts/caching.go))
- **LRU 策略**: 最近最少使用的缓存淘汰
- **容量管理**: 支持最大缓存大小限制
- **自动清理**: 定期清理过期缓存

#### 3. 工作池 ([`WorkerPool`](internal/tts/worker_pool.go))
- **动态调度**: 智能任务分配
- **统计监控**: 实时性能指标
- **错误处理**: 完善的错误恢复机制

---

## 🧪 开发指南

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/tts/...

# 运行测试并显示覆盖率
go test -cover ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 性能测试

```bash
# 运行基准测试
go test -bench=. ./internal/tts/...

# 运行中间件性能测试
go test -bench=. ./internal/http/middleware/...
```

### 代码质量

```bash
# 代码格式化
go fmt ./...

# 代码检查
go vet ./...

# 使用 golangci-lint（推荐）
golangci-lint run
```

### 构建二进制文件

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o tts-linux-amd64 cmd/api/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o tts-windows-amd64.exe cmd/api/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o tts-darwin-amd64 cmd/api/main.go

# 使用构建脚本
chmod +x script/build.sh
./script/build.sh
```

### Docker 构建

```bash
# 构建镜像
docker build -t tts:latest -f Dockerfile.optimized .

# 多架构构建
docker buildx build --platform linux/amd64,linux/arm64 -t tts:latest .
```

---

## 📊 性能监控

### Metrics 端点

访问 `http://localhost:8081/metrics` 获取性能指标：

```json
{
  "requests_total": 1000,
  "requests_success": 995,
  "requests_failed": 5,
  "cache_hits": 650,
  "cache_misses": 350,
  "cache_hit_rate": 0.65,
  "avg_response_time_ms": 245.5
}
```

### 健康检查

```bash
curl http://localhost:8081/health
```

**响应：**
```json
{
  "status": "healthy",
  "timestamp": "2025-10-19T23:00:00Z"
}
```

---

## 🔧 故障排查

### 常见问题

#### 1. 服务启动失败

**问题**: `failed to load config`
**解决**: 检查配置文件路径和格式，确保 YAML 语法正确

```bash
# 验证配置文件
docker run --rm -v $(pwd)/configs:/configs ch6vip/ch6vip-tts:main -config /configs/config.yaml
```

#### 2. TTS 请求失败

**问题**: `unauthorized` 或 `invalid api key`
**解决**: 确认 Azure TTS API 密钥和区域配置正确

```bash
# 设置环境变量
export TTS_API_KEY=your_valid_api_key
export TTS_REGION=eastasia
```

#### 3. 长文本合成问题

**问题**: 音频不连贯或合并失败
**解决**: 
- 确保安装 FFmpeg：`apt-get install ffmpeg`
- 启用 FFmpeg 合并：`use_ffmpeg_merge: true`
- 调整分段参数：`max_segment_length: 500`

#### 4. 缓存占用过多内存

**问题**: 内存使用持续增长
**解决**: 设置缓存大小限制

```yaml
cache:
  max_total_size: 1073741824  # 1GB
```

### 日志调试

启用 debug 日志查看详细信息：

```bash
# 通过环境变量
export LOG_LEVEL=debug

# 或在配置文件中
log:
  level: "debug"
  format: "json"  # JSON 格式便于解析
```

---

## 🤝 贡献指南

欢迎贡献代码、报告问题或提出建议！

### 提交问题

1. 在 [Issues](https://github.com/ch6vip/tts/issues) 页面搜索是否已存在相关问题
2. 创建新 Issue，提供详细的问题描述和复现步骤
3. 附上日志、配置文件等相关信息

### 提交代码

1. Fork 本仓库
2. 创建特性分支：`git checkout -b feature/your-feature`
3. 提交更改：`git commit -am 'Add some feature'`
4. 推送分支：`git push origin feature/your-feature`
5. 创建 Pull Request

### 代码规范

- 遵循 Go 语言官方代码风格
- 添加适当的注释和文档
- 编写单元测试
- 确保所有测试通过

---

## 📄 许可证

本项目基于 [MIT License](LICENSE) 开源，您可以自由使用、修改和分发本项目。

---

## 🙏 致谢

- [Microsoft Azure 认知服务](https://azure.microsoft.com/services/cognitive-services/) - 提供高质量 TTS API
- [Gin Web Framework](https://github.com/gin-gonic/gin) - 高性能 Go Web 框架
- [Zerolog](https://github.com/rs/zerolog) - 零分配 JSON 日志库
- [FFmpeg](https://ffmpeg.org/) - 强大的音频处理工具

---

## 📞 联系方式

- **Issues**: [GitHub Issues](https://github.com/ch6vip/tts/issues)
- **Discussions**: [GitHub Discussions](https://github.com/ch6vip/tts/discussions)

---

<div align="center">

**[⬆ 回到顶部](#tts-语音合成服务)**

Made with ❤️ by the TTS Team

</div>
