# TTS 服务

一个简单易用的文本转语音 (TTS) 服务，基于 Microsoft Azure 语音服务，提供高质量的语音合成能力，并兼容 OpenAI TTS API。

## ✨ 功能特点

- **高质量语音合成**：基于 Microsoft Azure 语音服务，提供自然流畅的语音。
- **多种语言和声音**：支持全球多种语言和丰富的语音选择。
- **高度可调**：支持调节语速、语调和说话风格。
- **OpenAI API 兼容**：兼容 OpenAI 的 TTS API，可无缝集成到现有应用中。
- **长文本支持**：自动对长文本进行分段处理，确保合成的稳定性和流畅性。
- **灵活部署**：支持 Docker 部署方式。
- **Web 用户界面**：提供一个简单直观的 Web 界面，方便快速测试和使用。
- **详细的 API 文档**：提供在线 API 文档，方便开发者集成。
- **SSML 支持**：支持语音合成标记语言 (SSML)，实现更精细的语音控制。

## 🚀 快速开始

### Docker 部署

我们提供多种方式来部署 TTS 服务。

#### 方式一：使用 Docker Compose (推荐)

这是最简单且推荐的部署方式，便于管理和自定义配置。

1.  **确保您已安装 Docker Compose。**
2.  在项目根目录下，我们已经为您提供了一个 `docker-compose.yml` 文件。
3.  执行以下命令启动服务：
    ```shell
    docker-compose up -d
    ```
    服务将在后台运行。如需自定义配置（例如修改端口、挂载配置文件或设置环境变量），请直接编辑 `docker-compose.yml` 文件，文件内包含了详细的注释和示例。

#### 方式二：使用 `docker run` 命令

如果您不希望使用 Docker Compose，也可以直接使用 `docker run` 命令。

##### 1. 快速部署

这是最简单的启动方式，直接从 Docker Hub 拉取并运行镜像。

- **默认端口 (8080):**
  ```shell
  docker run -d -p 8080:8080 --name=tts ch6vip/ch6vip-tts:main
  ```

- **映射到不同主机端口 (例如 9000):**
  ```shell
  docker run -d -p 9000:8080 --name=tts ch6vip/ch6vip-tts:main
  ```

##### 2. 高级用法 - 自定义配置

您可以通过挂载配置文件或使用环境变量来覆盖默认配置。

- **通过挂载配置文件:**
  将您本地的 `config.yaml` 文件挂载到容器的 `/app/configs/config.yaml` 路径。
  ```shell
  # 假设您的配置文件位于 /path/to/your/config.yaml
  docker run -d -p 8080:8080 \
    -v /path/to/your/config.yaml:/app/configs/config.yaml \
    --name=tts ch6vip/ch6vip-tts:main
  ```

- **通过环境变量:**
  根据配置规则（将 `.` 替换为 `_` 并大写），使用 `-e` 参数来设置环境变量。
  ```shell
  # 示例：修改服务端口并设置 OpenAI API 密钥
  docker run -d -p 9000:9000 \
    -e SERVER_PORT=9000 \
    -e OPENAI_API_KEY="your_openai_api_key" \
    --name=tts ch6vip/ch6vip-tts:main
  ```

#### 方式三：本地构建 Docker 镜像

如果您需要基于源码进行自定义修改，可以从 `Dockerfile` 在本地构建镜像。

```shell
# 从项目根目录构建
docker build -t your-custom-tts-image .
```
构建完成后，您可以使用与上面相同的 `docker run` 或 Docker Compose 命令来运行您本地构建的镜像（只需将 `image` 字段指向 `your-custom-tts-image`）。

部署完成后，您可以：
- 访问 `http://localhost:<映射的主机端口>` 使用 Web 界面。
- 访问 `http://localhost:<映射的主机端口>/api-doc` 查看 API 文档。

## 🛠️ API 使用

服务提供了两种 API：基础 API 和 OpenAI 兼容 API。

### 基础 API

#### 获取语音列表

- **Endpoint**: `GET /voices`
- **说明**: 获取所有支持的语音列表。
- **示例**:
  ```shell
  curl "http://localhost:8080/voices"
  ```

#### 文本转语音

- **Endpoint**: `GET /tts` 或 `POST /tts`
- **说明**: 将文本转换为语音。
- **参数**:
    - `t` (或 `text`): 要转换的文本 (必填)。
    - `v` (或 `voice`): 语音名称，例如 `zh-CN-XiaoxiaoNeural`。
    - `r` (或 `rate`): 语速，范围 `-100` 到 `100`。
    - `p` (或 `pitch`): 语调，范围 `-100` 到 `100`。
    - `s` (或 `style`): 说话风格，例如 `cheerful`。
- **示例**:
  ```shell
  # 基础请求
  curl "http://localhost:8080/tts?t=你好，世界&v=zh-CN-XiaoxiaoNeural"

  # 调整语速和语调
  curl "http://localhost:8080/tts?t=你好，世界&v=zh-CN-XiaoxiaoNeural&r=20&p=10"
  ```

### OpenAI 兼容 API

- **Endpoint**: `POST /v1/audio/speech`
- **说明**: 兼容 OpenAI 的 TTS API 格式。
- **请求体**:
  ```json
  {
    "model": "tts-1",
    "input": "你好，世界！",
    "voice": "zh-CN-XiaoxiaoNeural"
  }
  ```
- **参数映射**:
    - `model`: 对应基础 API 的 `style` 参数。
    - `voice`: 对应基础 API 的 `voice` 参数。如果使用 OpenAI 的标准语音（如 `alloy`），会自动映射到预设的中文语音。
- **示例**:
  ```shell
  curl -X POST "http://localhost:8080/v1/audio/speech" \
    -H "Content-Type: application/json" \
    -d '{
      "model": "tts-1",
      "input": "你好，世界！",
      "voice": "zh-CN-XiaoxiaoNeural"
    }'
  ```

## ⚙️ 配置选项

您可以通过环境变量或配置文件 (`configs/config.yaml`) 来自定义服务。环境变量的优先级高于配置文件。

### 配置文件详解

默认配置文件路径为 `configs/config.yaml`。

```yaml
server:
  port: 8080                # 服务监听端口
  read_timeout: 60          # HTTP 读取超时时间（秒）
  write_timeout: 60         # HTTP 写入超时时间（秒）
  base_path: ""             # API 基础路径前缀，如 "/api"

tts:
  region: "eastasia"        # Azure 语音服务区域
  default_voice: "zh-CN-XiaoxiaoNeural"  # 默认语音
  default_rate: "0"         # 默认语速，范围 -100 到 100
  default_pitch: "0"        # 默认语调，范围 -100 到 100
  default_format: "audio-24khz-48kbitrate-mono-mp3"  # 默认音频格式
  max_text_length: 65535    # 最大文本长度
  request_timeout: 30       # 请求 Azure 服务的超时时间（秒）
  max_concurrent: 20        # 长文本分段合成时的最大并发数
  segment_threshold: 300    # 文本自动分段的长度阈值
  min_sentence_length: 200  # 合并句子的最小长度
  max_sentence_length: 300  # 单个句子的最大长度
  api_key: ''               # /tts 接口的认证密钥 (可选)

  # OpenAI 到微软 TTS 中文语音的映射
  voice_mapping:
    alloy: "zh-CN-XiaoyiNeural"
    echo: "zh-CN-YunxiNeural"
    fable: "zh-CN-XiaochenNeural"
    onyx: "zh-CN-YunjianNeural"
    nova: "zh-CN-XiaohanNeural"
    shimmer: "zh-CN-XiaomoNeural"

openai:
  api_key: ''               # OpenAI 兼容接口的认证密钥 (可选)

ssml:
  # 需要在转义时保留的 SSML 标签
  preserve_tags:
    - name: break
      pattern: <break\s+[^>]*/>
    # ... 其他标签
```

### 环境变量

所有配置项都可以通过环境变量覆盖。将配置路径中的 `.` 替换为 `_` 并转换为大写即可。

例如，要覆盖服务端口和 OpenAI API 密钥：

```shell
export SERVER_PORT=9000
export OPENAI_API_KEY="your_openai_api_key"
```

## 🏗️ 本地构建与运行

如果您想从源码构建和运行：

```shell
# 1. 克隆仓库
git clone https://github.com/zuoban/tts.git
cd tts

# 2. 构建
go build -o tts ./cmd/api

# 3. 运行
./tts --config ./configs/config.yaml
```

## 📁 项目结构

```
.
├── cmd/api/            # Go 应用入口
├── configs/            # 配置文件
├── internal/           # 项目内部代码
│   ├── config/         # 配置加载
│   ├── http/           # HTTP 服务、路由、处理器和中间件
│   ├── models/         # 数据模型
│   └── tts/            # TTS 服务核心逻辑和 Microsoft 客户端
├── script/             # 构建脚本
├── web/                # Web 前端资源
│   ├── static/         # CSS, JS, 图标等
│   └── templates/      # HTML 模板
```

## 📄 许可证

本项目基于 [MIT](LICENSE) 许可证。
