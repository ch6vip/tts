package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"time"
	"tts/internal/config"
	"tts/internal/models"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// CacheStats 缓存统计信息
type CacheStats struct {
	Hits       int64   `json:"hits"`
	Misses     int64   `json:"misses"`
	HitRate    float64 `json:"hit_rate"`
	ItemCount  int     `json:"item_count"`
	TotalSize  int64   `json:"total_size_bytes"`
}

// cachingService is a struct that wraps a tts.Service to add a caching layer.
type cachingService struct {
	next         Service
	cache        *cache.Cache
	hits         int64 // 缓存命中次数
	misses       int64 // 缓存未命中次数
	totalSize    int64 // 缓存总大小(字节)
	maxTotalSize int64 // 缓存最大总大小限制(字节)，0表示不限制
}

// GetUnderlyingService returns the underlying service wrapped by the cache.
func (s *cachingService) GetUnderlyingService() Service {
	return s.next
}

// NewCachingService creates a new caching service.
// maxTotalSize 参数是可选的，默认为0表示不限制缓存大小
func NewCachingService(next Service, defaultExpiration, cleanupInterval time.Duration, maxTotalSize ...int64) Service {
	var maxSize int64 = 0 // 默认不限制
	if len(maxTotalSize) > 0 {
		maxSize = maxTotalSize[0]
	}
	
	c := &cachingService{
		next:         next,
		cache:        cache.New(defaultExpiration, cleanupInterval),
		maxTotalSize: maxSize,
	}
	
	// 设置缓存项被删除时的回调函数，用于更新总大小统计
	c.cache.OnEvicted(func(key string, value interface{}) {
		if resp, ok := value.(*models.TTSResponse); ok {
			atomic.AddInt64(&c.totalSize, -int64(len(resp.AudioContent)))
		}
	})
	
	return c
}

// ListVoices forwards the call to the next service without caching.
func (s *cachingService) ListVoices(ctx context.Context, locale string) ([]models.Voice, error) {
	return s.next.ListVoices(ctx, locale)
}

// SynthesizeSpeech synthesizes speech, using a cache to store and retrieve results.
func (s *cachingService) SynthesizeSpeech(ctx context.Context, req models.TTSRequest) (*models.TTSResponse, error) {
	// Generate a unique cache key for the request.
	key := s.generateCacheKey(req)

	// Try to retrieve the response from the cache.
	if resp, found := s.cache.Get(key); found {
		atomic.AddInt64(&s.hits, 1)
		logrus.WithField("key", key).Debug("Cache hit")
		result := resp.(*models.TTSResponse)
		result.CacheHit = true
		return result, nil
	}

	atomic.AddInt64(&s.misses, 1)
	logrus.WithField("key", key).Debug("Cache miss")

	// If not in cache, call the actual TTS service.
	resp, err := s.next.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store the successful response in the cache.
	// 首先检查是否超过最大缓存大小限制
	currentSize := atomic.LoadInt64(&s.totalSize)
	responseSize := int64(len(resp.AudioContent))
	
	// 如果设置了最大限制且添加此项会超过限制，则不缓存
	if s.maxTotalSize > 0 && (currentSize+responseSize) > s.maxTotalSize {
		logrus.WithFields(logrus.Fields{
			"key":          key,
			"current_size": currentSize,
			"response_size": responseSize,
			"max_size":     s.maxTotalSize,
		}).Debug("Skipping cache due to size limit")
		return resp, nil
	}
	
	s.cache.Set(key, resp, cache.DefaultExpiration)
	
	// 更新缓存大小统计
	atomic.AddInt64(&s.totalSize, responseSize)

	return resp, nil
}

// generateCacheKey creates a unique SHA256 hash for a TTSRequest.
// 包含所有影响输出的参数,确保缓存的正确性
func (s *cachingService) generateCacheKey(req models.TTSRequest) string {
	hash := sha256.New()
	
	// 获取音频格式，优先使用请求中指定的格式，否则使用默认格式
	cfg := config.Get()
	format := req.Format
	if format == "" {
		format = cfg.TTS.DefaultFormat
	}
	
	// 根据是否有 SSML 使用不同的键
	if req.SSML != "" {
		// SSML 模式：包含 SSML 内容和所有可能影响输出的参数
		hash.Write([]byte("ssml:"))
		hash.Write([]byte(req.SSML))
		
		// 即使使用 SSML，也要包含 Voice 参数，因为它可能影响默认的语言设置
		if req.Voice != "" {
			hash.Write([]byte("|voice:"))
			hash.Write([]byte(req.Voice))
		}
		
		// 包含音频格式，因为它直接影响输出
		hash.Write([]byte("|format:"))
		hash.Write([]byte(format))
	} else {
		// Text 模式：包含所有相关字段
		hash.Write([]byte("text:"))
		hash.Write([]byte(req.Text))
		hash.Write([]byte("|voice:"))
		hash.Write([]byte(req.Voice))
		hash.Write([]byte("|rate:"))
		hash.Write([]byte(req.Rate))
		hash.Write([]byte("|pitch:"))
		hash.Write([]byte(req.Pitch))
		hash.Write([]byte("|style:"))
		hash.Write([]byte(req.Style))
		
		// 包含音频格式
		hash.Write([]byte("|format:"))
		hash.Write([]byte(format))
	}
	
	return hex.EncodeToString(hash.Sum(nil))
}

// GetStats 获取缓存统计信息
func (s *cachingService) GetStats() CacheStats {
	hits := atomic.LoadInt64(&s.hits)
	misses := atomic.LoadInt64(&s.misses)
	total := hits + misses
	
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(hits) / float64(total) * 100
	}
	
	return CacheStats{
		Hits:      hits,
		Misses:    misses,
		HitRate:   hitRate,
		ItemCount: s.cache.ItemCount(),
		TotalSize: atomic.LoadInt64(&s.totalSize),
	}
}

// ClearCache 清空缓存
func (s *cachingService) ClearCache() {
	s.cache.Flush()
	atomic.StoreInt64(&s.totalSize, 0)
	logrus.Info("Cache cleared")
}

// GetCacheKey 公开方法用于测试或调试
func (s *cachingService) GetCacheKey(req models.TTSRequest) string {
	return s.generateCacheKey(req)
}