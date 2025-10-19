package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync/atomic"
	"time"
	"tts/internal/config"
	"tts/internal/models"

	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog"
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
	logger       zerolog.Logger
}

// GetUnderlyingService returns the underlying service wrapped by the cache.
func (s *cachingService) GetUnderlyingService() Service {
	return s.next
}

// NewCachingService creates a new caching service.
// maxTotalSize 参数是可选的，默认为0表示不限制缓存大小
func NewCachingService(next Service, defaultExpiration, cleanupInterval time.Duration, logger zerolog.Logger, maxTotalSize ...int64) Service {
	var maxSize int64 = 0 // 默认不限制
	if len(maxTotalSize) > 0 {
		maxSize = maxTotalSize[0]
	}
	
	c := &cachingService{
		next:         next,
		cache:        cache.New(defaultExpiration, cleanupInterval),
		maxTotalSize: maxSize,
		logger:       logger,
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
		s.logger.Debug().Str("key", key).Msg("Cache hit")
		result := resp.(*models.TTSResponse)
		result.CacheHit = true
		return result, nil
	}

	atomic.AddInt64(&s.misses, 1)
	s.logger.Debug().Str("key", key).Msg("Cache miss")

	// If not in cache, call the actual TTS service.
	resp, err := s.next.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store the successful response in the cache.
	// 首先检查是否超过最大缓存大小限制
	currentSize := atomic.LoadInt64(&s.totalSize)
	responseSize := int64(len(resp.AudioContent))
	
	// 如果设置了最大限制且添加此项会超过限制，尝试驱逐旧缓存项
	if s.maxTotalSize > 0 && (currentSize+responseSize) > s.maxTotalSize {
		s.logger.Debug().
			Str("key", key).
			Int64("current_size", currentSize).
			Int64("response_size", responseSize).
			Int64("max_size", s.maxTotalSize).
			Msg("Cache size limit reached, attempting to evict old items")
		
		// 尝试驱逐缓存项以腾出空间
		spaceNeeded := (currentSize + responseSize) - s.maxTotalSize
		if s.evictCacheItems(spaceNeeded) {
			// 驱逐成功，继续缓存
			s.logger.Debug().
				Int64("space_freed", spaceNeeded).
				Msg("Successfully evicted cache items")
		} else {
			// 驱逐失败或无法腾出足够空间，跳过缓存
			s.logger.Debug().
				Str("key", key).
				Msg("Unable to free enough space, skipping cache")
			return resp, nil
		}
	}
	
	s.cache.Set(key, resp, cache.DefaultExpiration)
	
	// 更新缓存大小统计
	atomic.AddInt64(&s.totalSize, responseSize)

	return resp, nil
}

// evictCacheItems 尝试驱逐足够的缓存项以腾出指定的空间
// 返回 true 表示成功腾出足够空间，false 表示失败
func (s *cachingService) evictCacheItems(spaceNeeded int64) bool {
	items := s.cache.Items()
	if len(items) == 0 {
		return false
	}
	
	// 将缓存项转换为切片以便排序
	type cacheItem struct {
		key        string
		expiration int64
		size       int64
	}
	
	itemList := make([]cacheItem, 0, len(items))
	for k, v := range items {
		if resp, ok := v.Object.(*models.TTSResponse); ok {
			itemList = append(itemList, cacheItem{
				key:        k,
				expiration: v.Expiration,
				size:       int64(len(resp.AudioContent)),
			})
		}
	}
	
	if len(itemList) == 0 {
		return false
	}
	
	// 按过期时间排序（最早过期的在前面）
	// 如果过期时间相同，按大小排序（大的在前面，优先删除大项）
	for i := 0; i < len(itemList)-1; i++ {
		for j := i + 1; j < len(itemList); j++ {
			if itemList[i].expiration > itemList[j].expiration ||
				(itemList[i].expiration == itemList[j].expiration && itemList[i].size < itemList[j].size) {
				itemList[i], itemList[j] = itemList[j], itemList[i]
			}
		}
	}
	
	// 逐个删除缓存项，直到腾出足够空间
	var freedSpace int64
	evictedCount := 0
	
	for _, item := range itemList {
		if freedSpace >= spaceNeeded {
			break
		}
		
		s.cache.Delete(item.key)
		freedSpace += item.size
		evictedCount++
		
		s.logger.Debug().
			Str("key", item.key).
			Int64("size", item.size).
			Int64("freed_space", freedSpace).
			Msg("Evicted cache item")
	}
	
	s.logger.Info().
		Int("evicted_count", evictedCount).
		Int64("space_needed", spaceNeeded).
		Int64("space_freed", freedSpace).
		Msg("Cache eviction completed")
	
	return freedSpace >= spaceNeeded
}

// normalizeValue 标准化参数值，去除前后空格并转换为小写
func normalizeValue(value string) string {
	return strings.TrimSpace(strings.ToLower(value))
}

// generateCacheKey 创建一个标准化的缓存键，只包含影响TTS输出的核心参数
// 使用直接字段拼接和哈希的方式，消除JSON序列化的CPU开销
func (s *cachingService) generateCacheKey(req models.TTSRequest) string {
	hash := sha256.New()
	
	// 获取音频格式，优先使用请求中指定的格式，否则使用默认格式
	cfg := config.Get()
	format := req.Format
	if format == "" {
		format = cfg.TTS.DefaultFormat
	}
	
	// 根据是否有 SSML 使用不同的模式，包含所有关键字段
	if req.SSML != "" {
		// SSML 模式
		hash.Write([]byte("mode:ssml"))
		hash.Write([]byte("|content:"))
		hash.Write([]byte(normalizeValue(req.SSML)))
		hash.Write([]byte("|voice:"))
		hash.Write([]byte(normalizeValue(req.Voice)))
		hash.Write([]byte("|rate:"))
		hash.Write([]byte(normalizeValue(req.Rate)))
		hash.Write([]byte("|pitch:"))
		hash.Write([]byte(normalizeValue(req.Pitch)))
		hash.Write([]byte("|style:"))
		hash.Write([]byte(normalizeValue(req.Style)))
		hash.Write([]byte("|format:"))
		hash.Write([]byte(normalizeValue(format)))
	} else {
		// 文本模式
		hash.Write([]byte("mode:text"))
		hash.Write([]byte("|content:"))
		hash.Write([]byte(normalizeValue(req.Text)))
		hash.Write([]byte("|voice:"))
		hash.Write([]byte(normalizeValue(req.Voice)))
		hash.Write([]byte("|rate:"))
		hash.Write([]byte(normalizeValue(req.Rate)))
		hash.Write([]byte("|pitch:"))
		hash.Write([]byte(normalizeValue(req.Pitch)))
		hash.Write([]byte("|style:"))
		hash.Write([]byte(normalizeValue(req.Style)))
		hash.Write([]byte("|format:"))
		hash.Write([]byte(normalizeValue(format)))
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
	s.logger.Info().Msg("Cache cleared")
}

// GetCacheKey 公开方法用于测试或调试
func (s *cachingService) GetCacheKey(req models.TTSRequest) string {
	return s.generateCacheKey(req)
}