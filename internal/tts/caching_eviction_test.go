package tts

import (
	"context"
	"testing"
	"time"
	"tts/internal/models"

	"github.com/rs/zerolog"
)

// mockTTSServiceForEviction 是一个用于测试缓存驱逐的模拟服务
type mockTTSServiceForEviction struct {
	callCount int
}

func (m *mockTTSServiceForEviction) SynthesizeSpeech(ctx context.Context, req models.TTSRequest) (*models.TTSResponse, error) {
	m.callCount++
	// 返回固定大小的音频内容用于测试
	audioSize := 1000 // 1KB
	if req.Text == "large" {
		audioSize = 5000 // 5KB
	}
	return &models.TTSResponse{
		AudioContent: make([]byte, audioSize),
		ContentType:  "audio/mpeg",
	}, nil
}

func (m *mockTTSServiceForEviction) ListVoices(ctx context.Context, locale string) ([]models.Voice, error) {
	return []models.Voice{}, nil
}

// TestCacheEviction 测试缓存驱逐功能
func TestCacheEviction(t *testing.T) {
	logger := zerolog.Nop()
	mockService := &mockTTSServiceForEviction{}
	
	// 创建一个最大大小为 3KB 的缓存服务
	maxSize := int64(3000)
	cachingService := NewCachingService(
		mockService,
		5*time.Minute,
		10*time.Minute,
		logger,
		maxSize,
	).(*cachingService)
	
	ctx := context.Background()
	
	// 第一次请求 - 应该缓存（1KB）
	req1 := models.TTSRequest{Text: "test1", Voice: "voice1"}
	resp1, err := cachingService.SynthesizeSpeech(ctx, req1)
	if err != nil {
		t.Fatalf("第一次请求失败: %v", err)
	}
	if resp1 == nil {
		t.Fatal("第一次请求返回 nil")
	}
	
	// 第二次请求 - 应该缓存（1KB，总计 2KB）
	req2 := models.TTSRequest{Text: "test2", Voice: "voice1"}
	resp2, err := cachingService.SynthesizeSpeech(ctx, req2)
	if err != nil {
		t.Fatalf("第二次请求失败: %v", err)
	}
	if resp2 == nil {
		t.Fatal("第二次请求返回 nil")
	}
	
	// 检查缓存统计
	stats := cachingService.GetStats()
	if stats.ItemCount != 2 {
		t.Errorf("期望缓存项数量为 2，实际为 %d", stats.ItemCount)
	}
	if stats.TotalSize != 2000 {
		t.Errorf("期望缓存总大小为 2000，实际为 %d", stats.TotalSize)
	}
	
	// 第三次请求 - 应该缓存（1KB，总计 3KB，达到上限）
	req3 := models.TTSRequest{Text: "test3", Voice: "voice1"}
	resp3, err := cachingService.SynthesizeSpeech(ctx, req3)
	if err != nil {
		t.Fatalf("第三次请求失败: %v", err)
	}
	if resp3 == nil {
		t.Fatal("第三次请求返回 nil")
	}
	
	stats = cachingService.GetStats()
	if stats.ItemCount != 3 {
		t.Errorf("期望缓存项数量为 3，实际为 %d", stats.ItemCount)
	}
	if stats.TotalSize != 3000 {
		t.Errorf("期望缓存总大小为 3000，实际为 %d", stats.TotalSize)
	}
	
	// 第四次请求 - 大文件（5KB），应该触发驱逐
	// 为了容纳 5KB，需要删除所有现有项（3KB）并还不够，所以这个请求不会被缓存
	req4 := models.TTSRequest{Text: "large", Voice: "voice1"}
	resp4, err := cachingService.SynthesizeSpeech(ctx, req4)
	if err != nil {
		t.Fatalf("第四次请求失败: %v", err)
	}
	if resp4 == nil {
		t.Fatal("第四次请求返回 nil")
	}
	
	// 由于大文件（5KB）超过了最大缓存大小（3KB），即使删除所有项也无法缓存
	// 所以缓存应该被清空，但新项不会被添加
	stats = cachingService.GetStats()
	if stats.ItemCount != 0 {
		t.Errorf("期望缓存项数量为 0（因为大文件无法缓存），实际为 %d", stats.ItemCount)
	}
	
	// 第五次请求 - 小文件（1KB），应该能够缓存
	req5 := models.TTSRequest{Text: "test5", Voice: "voice1"}
	resp5, err := cachingService.SynthesizeSpeech(ctx, req5)
	if err != nil {
		t.Fatalf("第五次请求失败: %v", err)
	}
	if resp5 == nil {
		t.Fatal("第五次请求返回 nil")
	}
	
	stats = cachingService.GetStats()
	if stats.ItemCount != 1 {
		t.Errorf("期望缓存项数量为 1，实际为 %d", stats.ItemCount)
	}
	if stats.TotalSize != 1000 {
		t.Errorf("期望缓存总大小为 1000，实际为 %d", stats.TotalSize)
	}
}

// TestCacheEvictionWithPartialSpace 测试部分空间驱逐
func TestCacheEvictionWithPartialSpace(t *testing.T) {
	logger := zerolog.Nop()
	mockService := &mockTTSServiceForEviction{}
	
	// 创建一个最大大小为 5KB 的缓存服务
	maxSize := int64(5000)
	cachingService := NewCachingService(
		mockService,
		5*time.Minute,
		10*time.Minute,
		logger,
		maxSize,
	).(*cachingService)
	
	ctx := context.Background()
	
	// 添加 4 个 1KB 的项（总计 4KB）
	for i := 1; i <= 4; i++ {
		req := models.TTSRequest{Text: string(rune(i)), Voice: "voice1"}
		_, err := cachingService.SynthesizeSpeech(ctx, req)
		if err != nil {
			t.Fatalf("请求 %d 失败: %v", i, err)
		}
	}
	
	stats := cachingService.GetStats()
	if stats.ItemCount != 4 {
		t.Errorf("期望缓存项数量为 4，实际为 %d", stats.ItemCount)
	}
	if stats.TotalSize != 4000 {
		t.Errorf("期望缓存总大小为 4000，实际为 %d", stats.TotalSize)
	}
	
	// 添加一个 2KB 的项，应该触发驱逐（需要删除至少 1KB）
	// 为了容纳 2KB，当前 4KB + 2KB = 6KB > 5KB，需要删除至少 1KB
	req := models.TTSRequest{Text: "new", Voice: "voice1"}
	_, err := cachingService.SynthesizeSpeech(ctx, req)
	if err != nil {
		t.Fatalf("新请求失败: %v", err)
	}
	
	// 驱逐应该删除足够的旧项来容纳新项
	stats = cachingService.GetStats()
	// 新项应该被成功缓存
	if stats.TotalSize > maxSize {
		t.Errorf("缓存总大小 %d 超过了最大限制 %d", stats.TotalSize, maxSize)
	}
	
	// 至少应该有新添加的项
	if stats.ItemCount == 0 {
		t.Error("期望缓存中至少有新添加的项")
	}
}