# HTTP 中间件

这个目录包含了 TTS 服务的 HTTP 中间件。

## 日志中间件

### 概述

日志中间件 (`logger.go`) 负责记录 HTTP 请求的详细信息，包括请求方法、路径、IP 地址、状态码、持续时间和用户代理等。

### 性能优化

我们使用 [zerolog](https://github.com/rs/zerolog) 替代了原来的 [logrus](https://github.com/sirupsen/logrus) 来实现高性能的请求日志中间件，以减少内存分配。

### 性能对比

基准测试结果显示，zerolog 相比 logrus 有显著的性能提升：

| 中间件 | 每次操作耗时 | 内存分配 | 分配次数 |
|--------|-------------|----------|----------|
| zerolog | 2660 ns/op | 2183 B/op | 16 allocs/op |
| logrus  | 5731 ns/op | 3954 B/op | 49 allocs/op |

- **性能提升**: zerolog 比 logrus 快约 2.15 倍
- **内存使用**: zerolog 减少了约 45% 的内存分配
- **GC 压力**: zerolog 减少了约 67% 的分配次数，降低了 GC 压力

### 使用方法

日志中间件会自动记录所有请求的详细信息，包括：

- trace_id: 每个请求的唯一标识符
- method: HTTP 方法
- path: 请求路径
- ip: 客户端 IP 地址
- status: HTTP 状态码
- duration: 请求处理时间
- user_agent: 用户代理字符串

如果请求处理过程中发生错误，中间件还会记录错误信息。

### 配置

日志中间件使用应用程序的全局日志配置，支持以下配置选项：

- **格式**: JSON 或文本格式
- **级别**: debug, info, warn, error
- **输出**: 控制台输出

### 示例输出

#### JSON 格式
```json
{
  "level": "info",
  "trace_id": "550e8400-e29b-41d4-a716-446655440000",
  "method": "GET",
  "path": "/tts",
  "ip": "192.168.1.1",
  "status": 200,
  "duration": 150000000,
  "user_agent": "Mozilla/5.0...",
  "time": "2023-10-19T20:46:00Z",
  "message": "request completed"
}
```

#### 控制台格式
```
2:46PM INF request completed trace_id=550e8400-e29b-41d4-a716-446655440000 method=GET path=/tts ip=192.168.1.1 status=200 duration=150ms user_agent=Mozilla/5.0...
```

## 其他中间件

- **CORS**: 处理跨域资源共享
- **ErrorHandler**: 统一错误处理
- **Auth**: API 认证