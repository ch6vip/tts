package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics 性能指标收集器
type Metrics struct {
	// TTS 相关指标
	TTSRequests      int64         // 总请求数
	TTSSuccess       int64         // 成功请求数
	TTSErrors        int64         // 失败请求数
	TTSTotalLatency  int64         // 总延迟(纳秒)
	TTSMaxLatency    int64         // 最大延迟(纳秒)
	TTSMinLatency    int64         // 最小延迟(纳秒)
	
	// 缓存相关指标
	CacheHits        int64         // 缓存命中次数
	CacheMisses      int64         // 缓存未命中次数
	CacheTotalSize   int64         // 缓存总大小(字节)
	
	// Worker Pool 指标
	WorkerPoolJobs   int64         // 工作池处理的总任务数
	WorkerPoolErrors int64         // 工作池错误数
	
	mu               sync.RWMutex  // 用于 min/max 更新
}

// GlobalMetrics 全局指标实例
var GlobalMetrics = &Metrics{
	TTSMinLatency: 1<<63 - 1, // 最大 int64
}

// RecordTTSRequest 记录一次 TTS 请求
func (m *Metrics) RecordTTSRequest(latency time.Duration, err error) {
	atomic.AddInt64(&m.TTSRequests, 1)
	
	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.TTSTotalLatency, latencyNs)
	
	if err != nil {
		atomic.AddInt64(&m.TTSErrors, 1)
	} else {
		atomic.AddInt64(&m.TTSSuccess, 1)
	}
	
	// 更新 max/min 延迟
	m.mu.Lock()
	if latencyNs > m.TTSMaxLatency {
		m.TTSMaxLatency = latencyNs
	}
	if latencyNs < m.TTSMinLatency {
		m.TTSMinLatency = latencyNs
	}
	m.mu.Unlock()
}

// RecordCacheHit 记录缓存命中
func (m *Metrics) RecordCacheHit(size int64) {
	atomic.AddInt64(&m.CacheHits, 1)
	atomic.AddInt64(&m.CacheTotalSize, size)
}

// RecordCacheMiss 记录缓存未命中
func (m *Metrics) RecordCacheMiss() {
	atomic.AddInt64(&m.CacheMisses, 1)
}

// RecordWorkerPoolJob 记录工作池任务
func (m *Metrics) RecordWorkerPoolJob(err error) {
	atomic.AddInt64(&m.WorkerPoolJobs, 1)
	if err != nil {
		atomic.AddInt64(&m.WorkerPoolErrors, 1)
	}
}

// GetSnapshot 获取指标快照
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	requests := atomic.LoadInt64(&m.TTSRequests)
	success := atomic.LoadInt64(&m.TTSSuccess)
	errors := atomic.LoadInt64(&m.TTSErrors)
	totalLatency := atomic.LoadInt64(&m.TTSTotalLatency)
	cacheHits := atomic.LoadInt64(&m.CacheHits)
	cacheMisses := atomic.LoadInt64(&m.CacheMisses)
	
	m.mu.RLock()
	maxLatency := m.TTSMaxLatency
	minLatency := m.TTSMinLatency
	m.mu.RUnlock()
	
	// 如果没有请求,设置 min 为 0
	if requests == 0 {
		minLatency = 0
	}
	
	avgLatency := int64(0)
	if requests > 0 {
		avgLatency = totalLatency / requests
	}
	
	cacheHitRate := 0.0
	totalCacheOps := cacheHits + cacheMisses
	if totalCacheOps > 0 {
		cacheHitRate = float64(cacheHits) / float64(totalCacheOps) * 100
	}
	
	successRate := 0.0
	if requests > 0 {
		successRate = float64(success) / float64(requests) * 100
	}
	
	return MetricsSnapshot{
		TTSRequests:      requests,
		TTSSuccess:       success,
		TTSErrors:        errors,
		SuccessRate:      successRate,
		AvgLatency:       time.Duration(avgLatency),
		MaxLatency:       time.Duration(maxLatency),
		MinLatency:       time.Duration(minLatency),
		CacheHits:        cacheHits,
		CacheMisses:      cacheMisses,
		CacheHitRate:     cacheHitRate,
		CacheTotalSize:   atomic.LoadInt64(&m.CacheTotalSize),
		WorkerPoolJobs:   atomic.LoadInt64(&m.WorkerPoolJobs),
		WorkerPoolErrors: atomic.LoadInt64(&m.WorkerPoolErrors),
		Timestamp:        time.Now(),
	}
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	TTSRequests      int64         `json:"tts_requests"`
	TTSSuccess       int64         `json:"tts_success"`
	TTSErrors        int64         `json:"tts_errors"`
	SuccessRate      float64       `json:"success_rate"`
	AvgLatency       time.Duration `json:"avg_latency"`
	MaxLatency       time.Duration `json:"max_latency"`
	MinLatency       time.Duration `json:"min_latency"`
	CacheHits        int64         `json:"cache_hits"`
	CacheMisses      int64         `json:"cache_misses"`
	CacheHitRate     float64       `json:"cache_hit_rate"`
	CacheTotalSize   int64         `json:"cache_total_size"`
	WorkerPoolJobs   int64         `json:"worker_pool_jobs"`
	WorkerPoolErrors int64         `json:"worker_pool_errors"`
	Timestamp        time.Time     `json:"timestamp"`
}

// Reset 重置所有指标
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.TTSRequests, 0)
	atomic.StoreInt64(&m.TTSSuccess, 0)
	atomic.StoreInt64(&m.TTSErrors, 0)
	atomic.StoreInt64(&m.TTSTotalLatency, 0)
	atomic.StoreInt64(&m.CacheHits, 0)
	atomic.StoreInt64(&m.CacheMisses, 0)
	atomic.StoreInt64(&m.CacheTotalSize, 0)
	atomic.StoreInt64(&m.WorkerPoolJobs, 0)
	atomic.StoreInt64(&m.WorkerPoolErrors, 0)

	m.mu.Lock()
	m.TTSMaxLatency = 0
	m.TTSMinLatency = 1<<63 - 1 // 重置为 int64 最大值
	m.mu.Unlock()
}