package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Logger 是一个HTTP中间件，记录请求的详细信息
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

		entry := logrus.WithFields(logrus.Fields{
			"trace_id":   traceID,
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"status":     c.Writer.Status(),
			"duration":   duration,
			"user_agent": c.Request.UserAgent(),
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.String())
		} else {
			entry.Info("request completed")
		}
	}
}
