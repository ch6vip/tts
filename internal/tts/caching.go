package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sync/atomic"
	"time"
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
	next       Service
	cache      *cache.Cache
	hits       int64 // 缓存命中次数
	misses     int64 // 缓存未命中次数
	totalSize  int64 // 缓存总大小(字节)
}

// GetUnderlyingService returns the underlying service wrapped by the cache.
func (s *cachingService) GetUnderlyingService() Service {
	return s.next
}

// NewCachingService creates a new caching service.
func NewCachingService(next Service, defaultExpiration, cleanupInterval time.Duration) Service {
	return &cachingService{
		next:  next,
		cache: cache.New(defaultExpiration, cleanupInterval),
	}
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
	s.cache.Set(key, resp, cache.DefaultExpiration)
	
	// 更新缓存大小统计
	atomic.AddInt64(&s.totalSize, int64(len(resp.AudioContent)))

	return resp, nil
}

// generateCacheKey creates a unique SHA256 hash for a TTSRequest.
// 包含所有影响输出的参数,确保缓存的正确性
func (s *cachingService) generateCacheKey(req models.TTSRequest) string {
	hash := sha256.New()
	
	// 根据是否有 SSML 使用不同的键
	if req.SSML != "" {
		// SSML 优先,忽略其他参数
		hash.Write([]byte("ssml:"))
		hash.Write([]byte(req.SSML))
	} else {
		// 包含所有相关字段
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