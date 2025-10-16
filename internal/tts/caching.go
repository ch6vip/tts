package tts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"
	"tts/internal/models"

	"github.com/patrickmn/go-cache"
	"github.com/sirupsen/logrus"
)

// cachingService is a struct that wraps a tts.Service to add a caching layer.
type cachingService struct {
	next  Service
	cache *cache.Cache
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
		logrus.WithField("key", key).Info("Cache hit")
		return resp.(*models.TTSResponse), nil
	}

	logrus.WithField("key", key).Info("Cache miss")

	// If not in cache, call the actual TTS service.
	resp, err := s.next.SynthesizeSpeech(ctx, req)
	if err != nil {
		return nil, err
	}

	// Store the successful response in the cache.
	s.cache.Set(key, resp, cache.DefaultExpiration)

	return resp, nil
}

// generateCacheKey creates a unique SHA256 hash for a TTSRequest.
func (s *cachingService) generateCacheKey(req models.TTSRequest) string {
	hash := sha256.New()
	// Include all relevant fields from the request in the hash.
	hash.Write([]byte(req.Text))
	hash.Write([]byte(req.Voice))
	hash.Write([]byte(req.Rate))
	hash.Write([]byte(req.Pitch))
	hash.Write([]byte(req.Style))
	return hex.EncodeToString(hash.Sum(nil))
}