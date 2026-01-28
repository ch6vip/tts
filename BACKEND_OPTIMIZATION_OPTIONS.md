# 后端优化选项（Go/Gin）

这个项目是一个用 Go（Gin）实现的 HTTP API，对 Microsoft TTS 做了封装，包含：
- 请求日志 + `trace_id`（`internal/http/middleware/logger.go`）
- 错误处理中间件（`internal/http/middleware/error.go`）
- 可选的响应缓存（`internal/tts/caching.go`）
- 长文本分段 + worker pool + FFmpeg 合并（`internal/tts/long_text_service.go`、`internal/tts/worker_pool.go`、`internal/tts/audio/merger.go`）
- 指标接口（`internal/http/handlers/metrics.go`、`internal/metrics/metrics.go`）

下面是可供选择的具体后端改进项。每一项都包含：影响、风险、修改位置/方向（以及必要时的实现选项）。

## P0（正确性 / 生产安全）

### P0-1 修复长文本 worker-pool 结果串话（并发 bug）
- 问题本质：`WorkerPool` 只有一个全局 `results` channel；`LongTextTTSService` 直接从该 channel 消费结果，但没有按“请求/任务”维度做隔离与路由。
- 现象：当两个（或更多）长文本请求并发时，A 请求的收集协程可能读到 B 请求的分段结果，导致：
  - 音频片段顺序错乱或混入其他请求内容（最严重：串音）
  - 某个请求永远等不到“自己”的某些分段结果（卡住或超时）
- 位置：
  - `internal/tts/long_text_service.go`：`processSegmentsConcurrently` 从 `s.workerPool.Results()` 收集结果（共享消费）
  - `internal/tts/worker_pool.go`：`WorkerPool.results` 为单一 channel（全局广播/队列）
- 推荐方案（A）：在 `WorkerPool` 增加“按 jobID 路由”的结果分发器（dispatcher）
  - **目标**：每个请求只消费自己的结果；不同请求之间互不干扰
  - **接口形态**（示例）：
    - `Submit(job)`：要求 job 显式携带 `JobID`（不要靠字符串前缀解析）
    - `Results(jobID) <-chan *SegmentResult`：返回该 job 专属的结果 channel
    - `Cancel(jobID)` / 自动清理：请求结束、失败或 `ctx.Done()` 时释放映射，避免泄漏
  - **实现要点**：
    - `WorkerPool` 维护 `map[jobID]chan result` + mutex
    - worker 产出结果后写入 dispatcher；dispatcher 再投递到对应 job channel（投递失败/已取消则丢弃并记日志）
    - 给 job channel 设置 buffer，避免短暂的消费者慢导致 worker 阻塞
- 备选方案（B）：每个请求创建独立 worker pool（实现最直观，但会引入 goroutine/队列/连接复用的额外抖动，通常不如方案 A）。
- 影响：正确性 + 高并发下稳定性（最高优先级）。
- 风险：中（涉及并发/通道生命周期）；需要补充并发测试覆盖。

### P0-2 将配置里的 Server 超时真正应用到 `http.Server`
- 问题：`ServerConfig.ReadTimeout/WriteTimeout` 已存在，但没有应用到 `http.Server`。
- 位置：`internal/http/server/server.go`
- 改动：设置 `ReadTimeout`、`WriteTimeout`、`IdleTimeout`、`ReadHeaderTimeout`、`MaxHeaderBytes`。
- 影响：防 slowloris/慢连接拖死；提升稳定性。
- 风险：低-中（超时设置太严格会影响极慢客户端）。

### P0-3 trace_id 缺失时避免 error middleware panic
- 问题：`ErrorHandler` 里 `traceID.(string)` 未检查存在/类型，极端情况下可能 panic。
- 位置：`internal/http/middleware/error.go`
- 改动：如果不存在/类型不对，安全回退到 `"unknown"`。
- 影响：避免边界情况下的 500/panic（比如中间件顺序变化、单测）。
- 风险：低。

## P1（性能 / 吞吐）

### P1-1 落地 `tts.max_concurrent`（当前未使用）
- 问题：`TTSConfig.MaxConcurrent` 有定义/校验，但实际没有限制上游并发；压力上来时上游调用可能突刺。
- 位置：`internal/tts/microsoft/client.go`（以及从配置透传/注入 limiter），也可在 `Client` 内做共享限流。
- 改动：用 semaphore/weighted limiter 包住上游 HTTP 调用（voices + synth 都可控）。
- 影响：资源使用更可预测；减少触发上游限流/429 的概率。
- 风险：低-中（需要结合实际负载调参）。

### P1-2 避免“修改缓存对象”；缓存命中返回副本
- 问题：缓存返回 `*models.TTSResponse`，命中时会对缓存对象设置 `CacheHit = true`；该对象会被多个请求共享。
- 位置：`internal/tts/caching.go`
- 改动：缓存里存不可变数据（例如 `[]byte`）；每次返回新建 `TTSResponse`（复制 struct；若你希望严格不可变，可考虑复制音频 slice）。
- 影响：正确性 + data race 安全（配合 `-race`）+ 指标更干净。
- 风险：低。

### P1-3 缓存 FFmpeg 可用性检测结果
- 问题：`FFmpegMerger.Merge()` 每次都会通过 `checkFFmpeg()` 执行 `ffmpeg -version`。
- 位置：`internal/tts/audio/merger.go`
- 改动：只检测一次（`sync.Once`）或缓存布尔值并定期复检。
- 影响：减少每次请求的额外开销（尤其是分段很多时）。
- 风险：低。

### P1-4 让长文本合并策略可配置（`use_ffmpeg_merge` 当前未使用）
- 问题：配置里有 `tts.long_text.use_ffmpeg_merge`，但 `LongTextTTSService` 实际总是用 FFmpeg merger（失败才 fallback）。
- 位置：`internal/tts/long_text_service.go`、`configs/config.yaml`、`internal/config/config.go`
- 改动：关闭时使用 `SimpleMerger`；开启时使用 FFmpeg merger。
- 影响：运维灵活性更高（比如某些环境没有 ffmpeg）。
- 风险：低。

## P2（安全 / API 加固）

### P2-1 避免把 API key 放在 query string；支持 Authorization header
- 问题：`TTSAuth` 使用 `?api_key=...`，容易被日志/缓存/Referer 泄露，也更容易被扫。
- 位置：`internal/http/middleware/auth.go`
- 改动建议：
  - 支持 `Authorization: Bearer <key>`（推荐），并可选保留 query key 以向后兼容
  - 使用常量时间比较（`subtle.ConstantTimeCompare`）
  - 如有需要，可对 `/health`、公开 `/voices` 等路由做不同策略
- 影响：安全性 + 更标准的客户端接入方式。
- 风险：低-中（若不做兼容，客户端需要改；建议先兼容再逐步下线 query key）。

### P2-2 收紧 `/metrics/reset`
- 问题：一个会改变状态的管理接口目前未鉴权。
- 位置：`internal/http/routes/routes.go` + `internal/http/handlers/metrics.go`
- 改动：加管理 key、限制仅 localhost 访问，或在生产环境移除。
- 影响：避免被轻易滥用（隐藏指标、频繁 reset 造成干扰/DoS）。
- 风险：低。

### P2-3 CORS 可配置（生产避免 `*`）
- 问题：CORS 允许所有 origin。
- 位置：`internal/http/middleware/cors.go`、`internal/config/config.go`
- 改动：加入 `cors.allowed_origins`、`allowed_methods`、`allowed_headers` 等配置，提供安全默认值。
- 影响：减少意外的浏览器侧暴露面。
- 风险：低-中（前端部署需要配合配置）。

## P3（可观测性 / 可运维性）

### P3-1 增加按路由/状态码的请求指标 + 延迟直方图
- 问题：当前 metrics 更偏 TTS 内部，HTTP 层视角有限。
- 位置：`internal/metrics/metrics.go`、`internal/http/handlers/metrics.go`，新增一个 Gin middleware。
- 改动：按路由模板 + 状态码分类计数；增加延迟桶/直方图（或直接接 Prometheus）。
- 影响：定位问题更快，容量规划更清晰。
- 风险：低-中。

### P3-2 将 trace_id 透出到响应和上游日志
- 位置：`internal/http/middleware/logger.go`（已生成 trace_id）、handlers。
- 改动：增加响应头 `X-Trace-Id`；必要时把 trace_id 带到上游请求 header 中。
- 影响：客户端/服务端链路关联更容易。
- 风险：低。

## P4（可维护性 / 设计）

### P4-1 移除 routes 初始化里的不安全类型断言
- 问题：`routes.SetupRoutes` 假定 `ttsService` 是 `*microsoft.Client` 或能解出它的缓存包装，直接 `.(*microsoft.Client)`；这会限制未来扩展（多 provider）并可能 panic。
- 位置：`internal/http/routes/routes.go`
- 改动选项：
  - A：让 `LongTextTTSService` 依赖 `tts.Service`（接口）而不是 `*microsoft.Client`
  - B：引入一个接口，例如 `type MicrosoftClientProvider interface { Microsoft() *microsoft.Client }`
- 影响：更易扩展 + 更少运行时 panic。
- 风险：中（需要重构 + 测试）。

### P4-2 caching 里不要再依赖全局 config
- 问题：`cachingService.generateCacheKey()` 调用 `config.Get()`（全局单例）；依赖隐藏、测试不友好。
- 位置：`internal/tts/caching.go`
- 改动：在 `NewCachingService` 里把 `defaultFormat`（以及其他默认值）作为参数传入。
- 影响：依赖更清晰，可测试性更好。
- 风险：低。

### P4-3 增加 `go test -race` + 并发长文本测试
- 位置：在 `internal/tts/` 下新增测试用例
- 目标：捕获 data race（缓存对象修改、worker pool 路由）、验证 P0-1 修复不回归。
- 影响：降低回归概率。
- 风险：低。

## 快速建议（想用“少改动换大收益”的最小方案）
1) P0-1 worker-pool 结果隔离（只要长文本存在并发使用，就必须修）
2) P0-2 server 超时配置落地 + P0-3 error middleware 防 panic
3) P1-1 max_concurrent 限流 + P1-2 缓存返回副本/不可变
4) P2-1 使用 header 鉴权 + P2-2 收紧 metrics reset
