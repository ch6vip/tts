package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"tts/internal/metrics"
)

// MetricsHandler 处理性能指标查询
type MetricsHandler struct{}

// NewMetricsHandler 创建新的 metrics handler
func NewMetricsHandler() *MetricsHandler {
	return &MetricsHandler{}
}

// GetMetrics 获取性能指标
func (h *MetricsHandler) GetMetrics(c *gin.Context) {
	snapshot := metrics.GlobalMetrics.GetSnapshot()
	
	// 获取系统内存统计
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	response := gin.H{
		"tts": gin.H{
			"requests":     snapshot.TTSRequests,
			"success":      snapshot.TTSSuccess,
			"errors":       snapshot.TTSErrors,
			"success_rate": snapshot.SuccessRate,
			"latency": gin.H{
				"avg": snapshot.AvgLatency.String(),
				"max": snapshot.MaxLatency.String(),
				"min": snapshot.MinLatency.String(),
			},
		},
		"cache": gin.H{
			"hits":       snapshot.CacheHits,
			"misses":     snapshot.CacheMisses,
			"hit_rate":   snapshot.CacheHitRate,
			"total_size": snapshot.CacheTotalSize,
		},
		"worker_pool": gin.H{
			"total_jobs": snapshot.WorkerPoolJobs,
			"errors":     snapshot.WorkerPoolErrors,
		},
		"system": gin.H{
			"memory": gin.H{
				"alloc_mb":       memStats.Alloc / 1024 / 1024,
				"total_alloc_mb": memStats.TotalAlloc / 1024 / 1024,
				"sys_mb":         memStats.Sys / 1024 / 1024,
				"num_gc":         memStats.NumGC,
			},
			"goroutines": runtime.NumGoroutine(),
		},
		"timestamp": snapshot.Timestamp.Format(time.RFC3339),
	}
	
	c.JSON(http.StatusOK, response)
}

// ResetMetrics 重置性能指标
func (h *MetricsHandler) ResetMetrics(c *gin.Context) {
	metrics.GlobalMetrics.Reset()
	c.JSON(http.StatusOK, gin.H{
		"message": "Metrics reset successfully",
	})
}

// HealthCheck 健康检查端点
func (h *MetricsHandler) HealthCheck(c *gin.Context) {
	snapshot := metrics.GlobalMetrics.GetSnapshot()
	
	// 简单的健康状态判断
	healthy := true
	reason := "ok"
	
	// 如果错误率超过 50%,认为不健康
	if snapshot.TTSRequests > 10 && snapshot.SuccessRate < 50 {
		healthy = false
		reason = "high error rate"
	}
	
	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}
	
	c.JSON(status, gin.H{
		"healthy": healthy,
		"reason":  reason,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}