package tts

import (
	"context"
	"testing"
	"tts/internal/models"
)

// TestGenerateCacheKey 测试缓存键生成是否包含所有影响输出的参数
func TestGenerateCacheKey(t *testing.T) {
	// 这里我们需要一个方法来设置测试配置，但由于config包的设计，
	// 我们将通过其他方式测试
	
	// 测试用例1: Text模式，所有参数都不同
	req1 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例2: Text模式，格式不同
	req2 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "0",
		Pitch: "0",
		Style: "general",
		Format: "audio-16khz-128kbitrate-mono-mp3",
	}
	
	// 测试用例3: Text模式，语速不同
	req3 := models.TTSRequest{
		Text:  "测试文本",
		Voice: "zh-CN-XiaoxiaoNeural",
		Rate:  "10",
		Pitch: "0",
		Style: "general",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例4: SSML模式，内容相同但语音不同
	req4 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-XiaoxiaoNeural",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例5: SSML模式，内容相同但语音不同
	req5 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-YunxiNeural",
		Format: "audio-24khz-48kbitrate-mono-mp3",
	}
	
	// 测试用例6: SSML模式，内容相同但格式不同
	req6 := models.TTSRequest{
		SSML:  "<speak version='1.0' xmlns='http://www.w3.org/2001/10/synthesis' xml:lang='zh-CN'><voice name='zh-CN-XiaoxiaoNeural'><prosody rate='0%'>测试文本</prosody></voice></speak>",
		Voice: "zh-CN-XiaoxiaoNeural",
		Format: "audio-16khz-128kbitrate-mono-mp3",
	}
	
	// 创建缓存服务（使用模拟的下一个服务）
	cacheService := &cachingService{
		next: &mockTTSService{},
	}
	
	// 生成缓存键
	key1 := cacheService.generateCacheKey(req1)
	key2 := cacheService.generateCacheKey(req2)
	key3 := cacheService.generateCacheKey(req3)
	key4 := cacheService.generateCacheKey(req4)
	key5 := cacheService.generateCacheKey(req5)
	key6 := cacheService.generateCacheKey(req6)
	
	// 验证不同参数生成不同的缓存键
	if key1 == key2 {
		t.Error("不同的音频格式应该生成不同的缓存键")
	}
	
	if key1 == key3 {
		t.Error("不同的语速应该生成不同的缓存键")
	}
	
	if key4 == key5 {
		t.Error("SSML模式下，不同的语音应该生成不同的缓存键")
	}
	
	if key4 == key6 {
		t.Error("SSML模式下，不同的音频格式应该生成不同的缓存键")
	}
	
	// 验证相同参数生成相同的缓存键
	key1Copy := cacheService.generateCacheKey(req1)
	if key1 != key1Copy {
		t.Error("相同参数应该生成相同的缓存键")
	}
	
	key4Copy := cacheService.generateCacheKey(req4)
	if key4 != key4Copy {
		t.Error("相同参数应该生成相同的缓存键")
	}
}

// mockTTSService 是一个模拟的TTS服务，用于测试
type mockTTSService struct{}

func (m *mockTTSService) ListVoices(ctx context.Context, locale string) ([]models.Voice, error) {
	return nil, nil
}

func (m *mockTTSService) SynthesizeSpeech(ctx context.Context, req models.TTSRequest) (*models.TTSResponse, error) {
	return nil, nil
}