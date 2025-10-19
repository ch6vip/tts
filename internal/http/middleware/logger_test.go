package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestLogger(t *testing.T) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 创建一个缓冲区来捕获日志输出
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

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 检查响应状态码
	assert.Equal(t, http.StatusOK, w.Code)

	// 检查日志输出
	logOutput := buf.String()
	assert.NotEmpty(t, logOutput)

	// 解析 JSON 日志（如果是 JSON 格式）
	if strings.Contains(logOutput, "{") {
		var logEntry map[string]interface{}
		err := json.Unmarshal([]byte(logOutput), &logEntry)
		assert.NoError(t, err)

		// 检查必要的字段
		assert.Contains(t, logEntry, "trace_id")
		assert.Contains(t, logEntry, "method")
		assert.Contains(t, logEntry, "path")
		assert.Contains(t, logEntry, "status")
		assert.Contains(t, logEntry, "duration")
		assert.Contains(t, logEntry, "user_agent")

		// 检查字段值
		assert.Equal(t, "GET", logEntry["method"])
		assert.Equal(t, "/test", logEntry["path"])
		assert.Equal(t, float64(200), logEntry["status"])
		assert.Equal(t, "test-agent", logEntry["user_agent"])
		assert.Equal(t, "request completed", logEntry["message"])
	} else {
		// 如果是控制台格式，检查关键字段
		assert.Contains(t, logOutput, "GET")
		assert.Contains(t, logOutput, "/test")
		assert.Contains(t, logOutput, "200")
		assert.Contains(t, logOutput, "test-agent")
		assert.Contains(t, logOutput, "request completed")
	}
}

func TestLoggerWithError(t *testing.T) {
	// 设置 Gin 为测试模式
	gin.SetMode(gin.TestMode)

	// 创建一个缓冲区来捕获日志输出
	var buf bytes.Buffer

	// 初始化 zerolog 使用我们的缓冲区
	logger = zerolog.New(&buf).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	initialized = true

	// 创建 Gin 路由
	router := gin.New()
	router.Use(Logger())

	// 添加一个测试路由，会返回错误
	router.GET("/error", func(c *gin.Context) {
		c.String(http.StatusInternalServerError, "error")
		_ = c.Error(errors.New("test error"))
	})

	// 创建一个测试请求
	req, _ := http.NewRequest("GET", "/error", nil)

	// 执行请求
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 检查响应状态码
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	// 检查日志输出
	logOutput := buf.String()
	assert.NotEmpty(t, logOutput)

	// 检查错误日志
	assert.Contains(t, logOutput, "request completed with errors")
}