package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// BenchmarkZerologLogger 测试 zerolog 日志中间件的性能
func BenchmarkZerologLogger(b *testing.B) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 创建一个空缓冲区来丢弃日志输出
	var buf bytes.Buffer

	// 初始化 zerolog 使用我们的缓冲区
	logger = zerolog.New(&buf).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	initialized = true

	// 创建 Gin 路由
	router := gin.New()
	router.Use(Logger())

	// 添加一个测试路由
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	// 创建一个测试请求
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")

	b.ResetTimer()
	b.ReportAllocs()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkLoggerWithDifferentFormats 测试不同格式的 zerolog 日志中间件性能
func BenchmarkLoggerWithDifferentFormats(b *testing.B) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 创建一个空缓冲区来丢弃日志输出
	var buf bytes.Buffer

	// 初始化 zerolog 使用 JSON 格式
	logger = zerolog.New(&buf).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	initialized = true

	// 创建 Gin 路由
	router := gin.New()
	router.Use(Logger())

	// 添加一个测试路由
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	// 创建一个测试请求
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")

	b.ResetTimer()
	b.ReportAllocs()

	// 运行基准测试
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}