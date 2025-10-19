package middleware

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"tts/internal/config"
)

// 全局 zerolog 实例，使用惰性初始化
var (
	logger zerolog.Logger
	initialized bool
)

// InitZerologWithConfig 使用配置初始化 zerolog 实例
func InitZerologWithConfig(logConfig *config.LogConfig) {
	// 设置日志级别
	var zerologLevel zerolog.Level
	switch logConfig.Level {
	case "debug":
		zerologLevel = zerolog.DebugLevel
	case "info":
		zerologLevel = zerolog.InfoLevel
	case "warn":
		zerologLevel = zerolog.WarnLevel
	case "error":
		zerologLevel = zerolog.ErrorLevel
	default:
		zerologLevel = zerolog.InfoLevel
	}

	// 配置 zerolog 输出
	if logConfig.Format == "json" {
		// JSON 格式输出
		logger = zerolog.New(os.Stdout).Level(zerologLevel).With().Timestamp().Logger()
	} else {
		// 控制台友好格式输出
		output := zerolog.ConsoleWriter{Out: os.Stdout}
		logger = zerolog.New(output).Level(zerologLevel).With().Timestamp().Logger()
	}
	
	initialized = true
}

// initZerolog 初始化 zerolog 实例（惰性初始化，使用默认配置）
func initZerolog() {
	// 默认配置：info 级别，控制台输出
	output := zerolog.ConsoleWriter{Out: os.Stdout}
	logger = zerolog.New(output).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	initialized = true
}

// getLogger 获取 zerolog 实例（惰性初始化）
func getLogger() zerolog.Logger {
	if !initialized {
		initZerolog()
	}
	return logger
}

// Logger 是一个HTTP中间件，记录请求的详细信息
// 使用 zerolog 实现高性能日志记录，减少内存分配
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 为每个请求创建一个trace_id
		traceID := uuid.New().String()
		c.Set("trace_id", traceID)

		// 处理请求
		c.Next()

		// 记录请求信息
		duration := time.Since(start)

		// 使用 zerolog 记录日志，减少内存分配
		log := getLogger()
		event := log.Info().
			Str("trace_id", traceID).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Str("ip", c.ClientIP()).
			Int("status", c.Writer.Status()).
			Dur("duration", duration).
			Str("user_agent", c.Request.UserAgent())

		if len(c.Errors) > 0 {
			// 如果有错误，记录错误日志
			event.Err(c.Errors.Last()).Msg("request completed with errors")
		} else {
			// 正常请求完成
			event.Msg("request completed")
		}
	}
}
