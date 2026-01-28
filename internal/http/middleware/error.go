package middleware

import (
	"errors"
	"net/http"
	custom_errors "tts/internal/errors"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// ErrorHandler 是一个处理错误的 Gin 中间件
func ErrorHandler(logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next() // 先执行后续的处理函数

		// 检查是否有错误发生
		if len(c.Errors) > 0 {
			err := c.Errors.Last().Err

			// 记录错误日志
			traceID := "unknown"
			if v, ok := c.Get("trace_id"); ok {
				if s, ok := v.(string); ok && s != "" {
					traceID = s
				}
			}
			log := logger.With().Str("trace_id", traceID).Logger()
			log.Error().Err(err).Msg("请求处理时发生错误")

			// 根据错误类型映射到 HTTP 状态码和响应
			var httpStatus int
			var errorMsg string

			switch {
			case errors.Is(err, custom_errors.ErrInvalidInput):
				httpStatus = http.StatusBadRequest
				errorMsg = err.Error()
			case errors.Is(err, custom_errors.ErrUpstreamServiceFailed):
				httpStatus = http.StatusBadGateway
				errorMsg = "上游服务失败"
			case errors.Is(err, custom_errors.ErrNotFound):
				httpStatus = http.StatusNotFound
				errorMsg = "资源未找到"
			case errors.Is(err, custom_errors.ErrRateLimited):
				httpStatus = http.StatusTooManyRequests
				errorMsg = "请求过于频繁"
			default:
				httpStatus = http.StatusInternalServerError
				errorMsg = "内部服务器错误"
			}

			// 如果响应尚未提交，则发送JSON错误响应
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(httpStatus, gin.H{"error": errorMsg})
			}
		}
	}
}
